package hostedcontrolplane

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/api/util/ipnet"
	"github.com/openshift/hypershift/support/api"
	"github.com/openshift/hypershift/support/testutil"
	"github.com/openshift/hypershift/support/util"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ServiceStrategyConfig defines the publishing strategy for each service type.
type ServiceStrategyConfig struct {
	KASType          hyperv1.PublishingStrategyType
	KASHostname      string
	OAuthType        hyperv1.PublishingStrategyType
	OAuthHostname    string
	KonnectivityType hyperv1.PublishingStrategyType
	KonnectivityHostname string
	IgnitionType     hyperv1.PublishingStrategyType
	IgnitionHostname string
}

// AllRoutes returns a config where all services use Route with hostnames.
func AllRoutes() ServiceStrategyConfig {
	return ServiceStrategyConfig{
		KASType:          hyperv1.Route,
		KASHostname:      "api.example.com",
		OAuthType:        hyperv1.Route,
		OAuthHostname:    "oauth.example.com",
		KonnectivityType: hyperv1.Route,
		KonnectivityHostname: "konnectivity.example.com",
		IgnitionType:     hyperv1.Route,
		IgnitionHostname: "ignition.example.com",
	}
}

// KASLoadBalancerOthersRoute returns a config where KAS uses LoadBalancer and others use Route.
func KASLoadBalancerOthersRoute() ServiceStrategyConfig {
	return ServiceStrategyConfig{
		KASType:          hyperv1.LoadBalancer,
		OAuthType:        hyperv1.Route,
		OAuthHostname:    "oauth.example.com",
		KonnectivityType: hyperv1.Route,
		KonnectivityHostname: "konnectivity.example.com",
		IgnitionType:     hyperv1.Route,
		IgnitionHostname: "ignition.example.com",
	}
}

// toServiceMappings converts ServiceStrategyConfig to ServicePublishingStrategyMapping slice.
func (c ServiceStrategyConfig) toServiceMappings() []hyperv1.ServicePublishingStrategyMapping {
	mappings := []hyperv1.ServicePublishingStrategyMapping{
		{
			Service: hyperv1.APIServer,
			ServicePublishingStrategy: hyperv1.ServicePublishingStrategy{
				Type: c.KASType,
			},
		},
		{
			Service: hyperv1.OAuthServer,
			ServicePublishingStrategy: hyperv1.ServicePublishingStrategy{
				Type: c.OAuthType,
			},
		},
		{
			Service: hyperv1.Konnectivity,
			ServicePublishingStrategy: hyperv1.ServicePublishingStrategy{
				Type: c.KonnectivityType,
			},
		},
		{
			Service: hyperv1.Ignition,
			ServicePublishingStrategy: hyperv1.ServicePublishingStrategy{
				Type: c.IgnitionType,
			},
		},
	}

	// Add hostnames for Route types
	if c.KASType == hyperv1.Route && c.KASHostname != "" {
		mappings[0].ServicePublishingStrategy.Route = &hyperv1.RoutePublishingStrategy{Hostname: c.KASHostname}
	}
	if c.OAuthType == hyperv1.Route && c.OAuthHostname != "" {
		mappings[1].ServicePublishingStrategy.Route = &hyperv1.RoutePublishingStrategy{Hostname: c.OAuthHostname}
	}
	if c.KonnectivityType == hyperv1.Route && c.KonnectivityHostname != "" {
		mappings[2].ServicePublishingStrategy.Route = &hyperv1.RoutePublishingStrategy{Hostname: c.KonnectivityHostname}
	}
	if c.IgnitionType == hyperv1.Route && c.IgnitionHostname != "" {
		mappings[3].ServicePublishingStrategy.Route = &hyperv1.RoutePublishingStrategy{Hostname: c.IgnitionHostname}
	}

	return mappings
}

// TestRouterInfrastructureWithStrategies is a comprehensive test that validates
// the router infrastructure behavior with various ServicePublishingStrategy configurations.
//
// For each test case, it generates fixtures in testdata/router-infra/<case>/ containing:
// - services.yaml: LoadBalancer services created (router, private-router)
// - routes.yaml: Route resources created
// - summary.yaml: A summary of what was created and the UseHCPRouter decision
//
// This provides a single place to see all router-related resources for each
// ServicePublishingStrategy configuration.
//
// Run with UPDATE=true to regenerate fixtures:
//
//	UPDATE=true go test ./control-plane-operator/controllers/hostedcontrolplane/... -run TestRouterInfrastructureWithStrategies
func TestRouterInfrastructureWithStrategies(t *testing.T) {
	const (
		hcpName      = "hcp"
		hcpNamespace = "hcp-namespace"
	)

	tests := []struct {
		name           string
		platformType   hyperv1.PlatformType
		endpointAccess interface{} // hyperv1.AWSEndpointAccessType or hyperv1.GCPEndpointAccessType
		strategies     ServiceStrategyConfig
		hcpAnnotations map[string]string
		setup          func(t *testing.T)
		subDir         string
	}{
		// ============================================
		// AWS Test Cases
		// ============================================
		{
			name:           "When AWS is Public with KAS LoadBalancer",
			platformType:   hyperv1.AWSPlatform,
			endpointAccess: hyperv1.Public,
			strategies:     KASLoadBalancerOthersRoute(),
			subDir:         "AWS_Public_KAS_LoadBalancer",
		},
		{
			name:           "When AWS is Public with KAS Route",
			platformType:   hyperv1.AWSPlatform,
			endpointAccess: hyperv1.Public,
			strategies:     AllRoutes(),
			subDir:         "AWS_Public_KAS_Route",
		},
		{
			name:           "When AWS is Private with all Routes",
			platformType:   hyperv1.AWSPlatform,
			endpointAccess: hyperv1.Private,
			strategies:     AllRoutes(),
			subDir:         "AWS_Private_All_Routes",
		},
		{
			name:           "When AWS is PublicAndPrivate with KAS LoadBalancer",
			platformType:   hyperv1.AWSPlatform,
			endpointAccess: hyperv1.PublicAndPrivate,
			strategies:     KASLoadBalancerOthersRoute(),
			subDir:         "AWS_PublicAndPrivate_KAS_LoadBalancer",
		},
		{
			name:           "When AWS is PublicAndPrivate with all Routes",
			platformType:   hyperv1.AWSPlatform,
			endpointAccess: hyperv1.PublicAndPrivate,
			strategies:     AllRoutes(),
			subDir:         "AWS_PublicAndPrivate_All_Routes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			services := tt.strategies.toServiceMappings()
			hcp := buildHCPForRouterTest(hcpName, hcpNamespace, tt.platformType, tt.endpointAccess, services, tt.hcpAnnotations)

			ctx := ctrl.LoggerInto(t.Context(), zapr.NewLogger(zaptest.NewLogger(t)))
			c := fake.NewClientBuilder().WithScheme(api.Scheme).WithObjects(hcp).Build()

			r := HostedControlPlaneReconciler{
				Client:               c,
				Log:                  ctrl.LoggerFrom(ctx),
				DefaultIngressDomain: "apps.example.com",
			}

			// Reconcile all infrastructure - no conditionals, all strategies are defined
			if err := r.reconcileHCPRouterServices(ctx, hcp, controllerutil.CreateOrUpdate); err != nil {
				t.Fatalf("reconcileHCPRouterServices failed: %v", err)
			}

			if err := r.reconcileAPIServerService(ctx, hcp, controllerutil.CreateOrUpdate); err != nil {
				t.Fatalf("reconcileAPIServerService failed: %v", err)
			}

			if err := r.reconcileOAuthServerService(ctx, hcp, controllerutil.CreateOrUpdate); err != nil {
				t.Fatalf("reconcileOAuthServerService failed: %v", err)
			}

			if err := r.reconcileKonnectivityServerService(ctx, hcp, controllerutil.CreateOrUpdate); err != nil {
				t.Fatalf("reconcileKonnectivityServerService failed: %v", err)
			}

			// Reconcile Ignition route
			ignitionStrategy := util.ServicePublishingStrategyByTypeForHCP(hcp, hyperv1.Ignition)
			if ignitionStrategy != nil && ignitionStrategy.Type == hyperv1.Route {
				ignitionRoute := &routev1.Route{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ignition-server",
						Namespace: hcp.Namespace,
					},
				}
				if _, err := controllerutil.CreateOrUpdate(ctx, c, ignitionRoute, func() error {
					return reconcileIgnitionRoute(ignitionRoute, hcp, r.DefaultIngressDomain)
				}); err != nil {
					t.Fatalf("reconcileIgnitionRoute failed: %v", err)
				}
			}

			// Collect all created resources
			result := &RouterInfraResult{
				TestCase:       tt.name,
				Platform:       string(tt.platformType),
				LabelHCPRoutes: util.LabelHCPRoutes(hcp),
				IsPrivateHCP:   util.IsPrivateHCP(hcp),
				IsPublicHCP:    util.IsPublicHCP(hcp),
			}

			// List services (router LB services)
			var allServices corev1.ServiceList
			if err := c.List(ctx, &allServices, client.InNamespace(hcpNamespace)); err != nil {
				t.Fatalf("failed to list services: %v", err)
			}
			// Filter to only router-related services
			var routerServices []corev1.Service
			for _, svc := range allServices.Items {
				if svc.Name == "router" || svc.Name == "private-router" {
					routerServices = append(routerServices, svc)
				}
			}
			result.Services = routerServices

			// List routes
			var routes routev1.RouteList
			if err := c.List(ctx, &routes, client.InNamespace(hcpNamespace)); err != nil {
				t.Fatalf("failed to list routes: %v", err)
			}
			result.Routes = routes.Items

			// Generate fixtures in the subDir
			fixtureDir := filepath.Join("router-infra", tt.subDir)

			// 1. Generate summary fixture
			summaryYaml := generateSummaryYaml(result, services)
			testutil.CompareWithFixture(t, summaryYaml,
				testutil.WithSubDir(fixtureDir),
				testutil.WithSuffix("_summary"))

			// 2. Generate services fixture
			routerServiceList := &corev1.ServiceList{Items: routerServices}
			servicesYaml, err := util.SerializeResource(routerServiceList, api.Scheme)
			if err != nil {
				t.Fatalf("failed to serialize services: %v", err)
			}
			testutil.CompareWithFixture(t, servicesYaml,
				testutil.WithSubDir(fixtureDir),
				testutil.WithSuffix("_services"))

			// 3. Generate individual service fixtures
			for _, svc := range routerServices {
				svcCopy := svc.DeepCopy()
				svcYaml, err := util.SerializeResource(svcCopy, api.Scheme)
				if err != nil {
					t.Fatalf("failed to serialize service %s: %v", svc.Name, err)
				}
				svcSuffix := fmt.Sprintf("_service_%s", strings.ReplaceAll(svc.Name, "-", "_"))
				testutil.CompareWithFixture(t, svcYaml,
					testutil.WithSubDir(fixtureDir),
					testutil.WithSuffix(svcSuffix))
			}

			// 4. Generate routes fixture
			routesYaml, err := util.SerializeResource(&routes, api.Scheme)
			if err != nil {
				t.Fatalf("failed to serialize routes: %v", err)
			}
			testutil.CompareWithFixture(t, routesYaml,
				testutil.WithSubDir(fixtureDir),
				testutil.WithSuffix("_routes"))

			// 5. Generate individual route fixtures
			for _, route := range routes.Items {
				routeCopy := route.DeepCopy()
				routeYaml, err := util.SerializeResource(routeCopy, api.Scheme)
				if err != nil {
					t.Fatalf("failed to serialize route %s: %v", route.Name, err)
				}
				routeSuffix := fmt.Sprintf("_route_%s", strings.ReplaceAll(route.Name, "-", "_"))
				testutil.CompareWithFixture(t, routeYaml,
					testutil.WithSubDir(fixtureDir),
					testutil.WithSuffix(routeSuffix))
			}
		})
	}
}

// RouterInfraResult captures the router infrastructure state for a test case.
type RouterInfraResult struct {
	TestCase       string
	Platform       string
	LabelHCPRoutes bool
	IsPrivateHCP   bool
	IsPublicHCP    bool
	Services       []corev1.Service
	Routes         []routev1.Route
}

// generateSummaryYaml creates a YAML summary of the router infrastructure decisions.
func generateSummaryYaml(result *RouterInfraResult, serviceStrategies []hyperv1.ServicePublishingStrategyMapping) string {
	var sb strings.Builder
	sb.WriteString("# Router Infrastructure Summary\n")
	sb.WriteString("# This file shows the decisions made for this ServicePublishingStrategy configuration\n")
	sb.WriteString("#\n")
	sb.WriteString(fmt.Sprintf("# Test Case: %s\n", result.TestCase))
	sb.WriteString("#\n\n")

	sb.WriteString("input:\n")
	sb.WriteString(fmt.Sprintf("  platform: %s\n", result.Platform))
	sb.WriteString(fmt.Sprintf("  isPrivateHCP: %t\n", result.IsPrivateHCP))
	sb.WriteString(fmt.Sprintf("  isPublicHCP: %t\n", result.IsPublicHCP))
	sb.WriteString("  servicePublishingStrategies:\n")
	for _, sps := range serviceStrategies {
		sb.WriteString(fmt.Sprintf("    - service: %s\n", sps.Service))
		sb.WriteString(fmt.Sprintf("      type: %s\n", sps.ServicePublishingStrategy.Type))
		if sps.ServicePublishingStrategy.Route != nil && sps.ServicePublishingStrategy.Route.Hostname != "" {
			sb.WriteString(fmt.Sprintf("      hostname: %s\n", sps.ServicePublishingStrategy.Route.Hostname))
		}
	}

	sb.WriteString("\n")
	sb.WriteString("decisions:\n")
	sb.WriteString(fmt.Sprintf("  labelHCPRoutes: %t\n", result.LabelHCPRoutes))
	sb.WriteString("  # LabelHCPRoutes determines if routes should be labeled for the HCP router.\n")
	sb.WriteString("  # True when: Private-only OR (Public with dedicated DNS for KAS via Route)\n")

	sb.WriteString("\n")
	sb.WriteString("createdResources:\n")
	sb.WriteString("  services:\n")
	if len(result.Services) == 0 {
		sb.WriteString("    # No router services created\n")
	} else {
		for _, svc := range result.Services {
			sb.WriteString(fmt.Sprintf("    - name: %s\n", svc.Name))
			sb.WriteString(fmt.Sprintf("      type: %s\n", svc.Spec.Type))
			if internal, ok := svc.Annotations["service.beta.kubernetes.io/aws-load-balancer-internal"]; ok && internal == "true" {
				sb.WriteString("      internal: true\n")
			} else if internal, ok := svc.Annotations["networking.gke.io/load-balancer-type"]; ok && internal == "Internal" {
				sb.WriteString("      internal: true\n")
			}
		}
	}

	sb.WriteString("  routes:\n")
	if len(result.Routes) == 0 {
		sb.WriteString("    # No routes created\n")
	} else {
		for _, route := range result.Routes {
			sb.WriteString(fmt.Sprintf("    - name: %s\n", route.Name))
			sb.WriteString(fmt.Sprintf("      host: %s\n", route.Spec.Host))
			// Check if route has the HCP label (will be picked up by HCP router)
			if _, hasHCPLabel := route.Labels[util.HCPRouteLabel]; hasHCPLabel {
				sb.WriteString("      labeledForHCPRouter: true\n")
			} else {
				sb.WriteString("      labeledForHCPRouter: false\n")
			}
			// Check visibility
			if visibility, ok := route.Labels[hyperv1.RouteVisibilityLabel]; ok {
				sb.WriteString(fmt.Sprintf("      visibility: %s\n", visibility))
			}
			if _, isInternal := route.Labels[util.InternalRouteLabel]; isInternal {
				sb.WriteString("      internal: true\n")
			}
		}
	}

	return sb.String()
}

// buildHCPForRouterTest creates a HostedControlPlane for testing router infrastructure.
func buildHCPForRouterTest(name, namespace string, platformType hyperv1.PlatformType, endpointAccess interface{}, services []hyperv1.ServicePublishingStrategyMapping, annotations map[string]string) *hyperv1.HostedControlPlane {
	hcp := &hyperv1.HostedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": "cluster_name",
			},
			Annotations: map[string]string{},
		},
		Spec: hyperv1.HostedControlPlaneSpec{
			IssuerURL: "https://test-oidc-bucket.s3.us-east-1.amazonaws.com/test-cluster",
			Configuration: &hyperv1.ClusterConfiguration{
				FeatureGate: &configv1.FeatureGateSpec{
					FeatureGateSelection: configv1.FeatureGateSelection{
						FeatureSet: configv1.Default,
					},
				},
			},
			Services: services,
			Networking: hyperv1.ClusterNetworking{
				ClusterNetwork: []hyperv1.ClusterNetworkEntry{
					{CIDR: *ipnet.MustParseCIDR("10.132.0.0/14")},
				},
			},
			Etcd: hyperv1.EtcdSpec{
				ManagementType: hyperv1.Managed,
			},
			Platform: hyperv1.PlatformSpec{
				Type: platformType,
			},
			ReleaseImage: "quay.io/openshift-release-dev/ocp-release:4.16.10-x86_64",
		},
	}

	// Merge annotations
	for k, v := range annotations {
		hcp.Annotations[k] = v
	}

	// Configure platform-specific settings
	switch platformType {
	case hyperv1.AWSPlatform:
		hcp.Spec.Platform.AWS = &hyperv1.AWSPlatformSpec{
			RolesRef: hyperv1.AWSRolesRef{
				NodePoolManagementARN: "arn:aws:iam::123456789012:role/test-node-pool-management-role",
			},
		}
		if ea, ok := endpointAccess.(hyperv1.AWSEndpointAccessType); ok {
			hcp.Spec.Platform.AWS.EndpointAccess = ea
		}
	case hyperv1.GCPPlatform:
		hcp.Spec.Platform.GCP = &hyperv1.GCPPlatformSpec{}
		if ea, ok := endpointAccess.(hyperv1.GCPEndpointAccessType); ok {
			hcp.Spec.Platform.GCP.EndpointAccess = ea
		}
	case hyperv1.AzurePlatform:
		hcp.Spec.Platform.Azure = &hyperv1.AzurePlatformSpec{
			SubnetID:        "/subscriptions/mySubscriptionID/resourceGroups/myResourceGroupName/providers/Microsoft.Network/virtualNetworks/myVnetName/subnets/mySubnetName",
			SecurityGroupID: "/subscriptions/mySubscriptionID/resourceGroups/myResourceGroupName/providers/Microsoft.Network/networkSecurityGroups/myNSGName",
			VnetID:          "/subscriptions/mySubscriptionID/resourceGroups/myResourceGroupName/providers/Microsoft.Network/virtualNetworks/myVnetName",
		}
	case hyperv1.IBMCloudPlatform:
		hcp.Spec.Platform.IBMCloud = &hyperv1.IBMCloudPlatformSpec{
			ProviderType: configv1.IBMCloudProviderTypeVPC,
		}
		hcp.Spec.Networking.APIServer = &hyperv1.APIServerNetworking{
			Port:             ptr.To[int32](2040),
			AdvertiseAddress: ptr.To("1.2.3.4"),
		}
	}

	return hcp
}

// reconcileIgnitionRoute simulates what the v2 ignitionserver component does
// for the ignition-server route. This allows us to test the route labeling
// behavior without requiring the full component framework.
func reconcileIgnitionRoute(route *routev1.Route, hcp *hyperv1.HostedControlPlane, defaultIngressDomain string) error {
	serviceName := "ignition-server-proxy"
	// For IBM Cloud, we don't deploy the ignition server proxy.
	if hcp.Spec.Platform.Type == hyperv1.IBMCloudPlatform {
		serviceName = "ignition-server"
	}

	if util.IsPrivateHCP(hcp) {
		return util.ReconcileInternalRoute(route, hcp.Name, serviceName)
	}

	strategy := util.ServicePublishingStrategyByTypeForHCP(hcp, hyperv1.Ignition)
	if strategy == nil {
		return fmt.Errorf("ignition service strategy not specified")
	}

	hostname := ""
	if strategy.Route != nil {
		hostname = strategy.Route.Hostname
	}
	return util.ReconcileExternalRoute(route, hostname, defaultIngressDomain, serviceName, util.LabelHCPRoutes(hcp))
}
