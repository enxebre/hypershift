package gcp

import (
	"context"
	"fmt"
	"os"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/images"
	"github.com/openshift/hypershift/support/upsert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/blang/semver"
)

const (
	ImageStreamCAPIGCP = "gcp-cluster-api-controllers"
)

// GCPCluster is a placeholder type for the actual CAPI GCP cluster type
// This will be replaced with the real type from sigs.k8s.io/cluster-api-provider-gcp
// when that provider is added to the vendor dependencies
type GCPCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GCPClusterSpec   `json:"spec,omitempty"`
	Status            GCPClusterStatus `json:"status,omitempty"`
}

// GCPClusterSpec defines the desired state of GCPCluster
type GCPClusterSpec struct {
	Project  string                   `json:"project"`
	Region   string                   `json:"region"`
	Network  GCPNetworkSpec          `json:"network,omitempty"`
	ControlPlaneEndpoint capiv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
	AdditionalLabels     map[string]string   `json:"additionalLabels,omitempty"`
}

// GCPNetworkSpec defines the GCP network configuration
type GCPNetworkSpec struct {
	Name string `json:"name,omitempty"`
}

// GCPClusterStatus defines the observed state of GCPCluster
type GCPClusterStatus struct {
	Ready bool `json:"ready,omitempty"`
}

// DeepCopyObject implements runtime.Object interface
func (g *GCPCluster) DeepCopyObject() runtime.Object {
	if g == nil {
		return nil
	}
	out := new(GCPCluster)
	g.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (g *GCPCluster) DeepCopyInto(out *GCPCluster) {
	*out = *g
	out.TypeMeta = g.TypeMeta
	g.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	g.Spec.DeepCopyInto(&out.Spec)
	out.Status = g.Status
}

// DeepCopy returns a deep copy of the GCPCluster
func (g *GCPCluster) DeepCopy() *GCPCluster {
	if g == nil {
		return nil
	}
	out := new(GCPCluster)
	g.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (g *GCPClusterSpec) DeepCopyInto(out *GCPClusterSpec) {
	*out = *g
	if g.AdditionalLabels != nil {
		in, out := &g.AdditionalLabels, &out.AdditionalLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// GCP implements the Platform interface for Google Cloud Platform.
type GCP struct {
	capiProviderImage string
	payloadVersion    *semver.Version
}

// New creates a new GCP platform implementation.
func New(capiProviderImage string, payloadVersion *semver.Version) *GCP {
	if payloadVersion != nil {
		payloadVersion.Pre = nil
		payloadVersion.Build = nil
	}

	return &GCP{
		capiProviderImage: capiProviderImage,
		payloadVersion:    payloadVersion,
	}
}

// ReconcileCAPIInfraCR reconciles the CAPI infrastructure resources for GCP.
func (g GCP) ReconcileCAPIInfraCR(
	ctx context.Context,
	c client.Client,
	createOrUpdate upsert.CreateOrUpdateFN,
	hcluster *hyperv1.HostedCluster,
	controlPlaneNamespace string,
	apiEndpoint hyperv1.APIEndpoint,
) (client.Object, error) {
	gcpCluster := &GCPCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcluster.Name,
			Namespace: controlPlaneNamespace,
		},
	}

	if _, err := createOrUpdate(ctx, c, gcpCluster, func() error {
		return reconcileGCPCluster(gcpCluster, hcluster, apiEndpoint)
	}); err != nil {
		return nil, fmt.Errorf("failed to reconcile GCP CAPI cluster: %w", err)
	}

	return gcpCluster, nil
}

// reconcileGCPCluster sets up the GCPCluster resource with the necessary configuration.
func reconcileGCPCluster(gcpCluster *GCPCluster, hcluster *hyperv1.HostedCluster, apiEndpoint hyperv1.APIEndpoint) error {
	if hcluster.Spec.Platform.GCP == nil {
		return fmt.Errorf("GCP platform configuration is missing")
	}

	gcpSpec := hcluster.Spec.Platform.GCP

	// Set basic cluster configuration
	gcpCluster.Spec.Project = gcpSpec.Project
	gcpCluster.Spec.Region = gcpSpec.Region

	// Set network configuration
	if gcpSpec.Network != nil {
		if gcpSpec.Network.Network != "" {
			gcpCluster.Spec.Network.Name = gcpSpec.Network.Network
		}
		if gcpSpec.Network.Subnetwork != "" {
			// For CAPI GCP, we'll set the subnet on the machine template
			// Here we just ensure the network is properly configured
		}
	}

	// Set control plane endpoint
	gcpCluster.Spec.ControlPlaneEndpoint = capiv1.APIEndpoint{
		Host: apiEndpoint.Host,
		Port: apiEndpoint.Port,
	}

	// Add labels if specified
	if gcpSpec.Labels != nil {
		if gcpCluster.Spec.AdditionalLabels == nil {
			gcpCluster.Spec.AdditionalLabels = make(map[string]string)
		}
		for k, v := range gcpSpec.Labels {
			gcpCluster.Spec.AdditionalLabels[k] = v
		}
	}

	return nil
}

// CAPIProviderDeploymentSpec returns the deployment specification for the GCP CAPI provider.
func (g GCP) CAPIProviderDeploymentSpec(hcluster *hyperv1.HostedCluster, _ *hyperv1.HostedControlPlane) (*appsv1.DeploymentSpec, error) {
	image := g.capiProviderImage
	if envImage := os.Getenv(images.GCPCAPIProviderEnvVar); len(envImage) > 0 {
		image = envImage
	}
	if override, ok := hcluster.Annotations[hyperv1.ClusterAPIGCPProviderImage]; ok {
		image = override
	}

	defaultMode := int32(0640)
	deploymentSpec := &appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: ptr.To[int64](10),
				Containers: []corev1.Container{{
					Name:            "manager",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/manager",
					},
					Args: []string{
						"--leader-elect",
						"--v=2",
						"--feature-gates=MachinePool=false",
						"--webhook-port=9443",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "webhook-server",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name: "MY_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("100Mi"),
							corev1.ResourceCPU:    resource.MustParse("10m"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "credentials",
							MountPath: "/etc/gcp",
							ReadOnly:  true,
						},
						{
							Name:      "capi-webhooks-tls",
							MountPath: "/tmp/k8s-webhook-server/serving-certs",
							ReadOnly:  true,
						},
					},
				}},
				Volumes: []corev1.Volume{
					{
						Name: "credentials",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "gcp-cloud-credentials",
								DefaultMode: &defaultMode,
							},
						},
					},
					{
						Name: "capi-webhooks-tls",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "capi-webhooks-tls",
								DefaultMode: &defaultMode,
							},
						},
					},
				},
			},
		},
	}

	return deploymentSpec, nil
}

// ReconcileCredentials reconciles GCP credentials in the control plane namespace.
func (g GCP) ReconcileCredentials(ctx context.Context, c client.Client, createOrUpdate upsert.CreateOrUpdateFN, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string) error {
	// Look for GCP credentials secret in the HostedCluster namespace
	credentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hcluster.Namespace,
			Name:      "gcp-cloud-credentials",
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(credentialsSecret), credentialsSecret); err != nil {
		return fmt.Errorf("failed to get GCP credentials secret: %w", err)
	}

	// Copy credentials to control plane namespace
	controlPlaneSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: controlPlaneNamespace,
			Name:      "gcp-cloud-credentials",
		},
	}

	_, err := createOrUpdate(ctx, c, controlPlaneSecret, func() error {
		controlPlaneSecret.Type = credentialsSecret.Type
		controlPlaneSecret.Data = credentialsSecret.Data
		return nil
	})

	return err
}

// ReconcileSecretEncryption reconciles GCP KMS secret encryption configuration.
func (g GCP) ReconcileSecretEncryption(ctx context.Context, c client.Client, createOrUpdate upsert.CreateOrUpdateFN, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string) error {
	// GCP KMS encryption is handled through etcd encryption configuration
	// For now, we'll implement basic support that can be extended later
	if hcluster.Spec.Platform.GCP != nil && hcluster.Spec.Platform.GCP.KMSKeyName != "" {
		// TODO: Implement GCP KMS encryption support
		// This would involve creating the necessary secrets and configurations
		// for etcd encryption using GCP Cloud KMS
	}
	return nil
}

// CAPIProviderPolicyRules returns the RBAC policy rules required by the GCP CAPI provider.
func (g GCP) CAPIProviderPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"infrastructure.cluster.x-k8s.io"},
			Resources: []string{
				"gcpclusters",
				"gcpmachinetemplates",
				"gcpmachines",
			},
			Verbs: []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{
				"events",
				"secrets",
			},
			Verbs: []string{"*"},
		},
		{
			APIGroups: []string{"cluster.x-k8s.io"},
			Resources: []string{
				"clusters",
				"machines",
				"machinesets",
			},
			Verbs: []string{"get", "list", "watch", "update", "patch"},
		},
	}
}

// DeleteCredentials cleans up GCP credentials from the control plane namespace.
func (g GCP) DeleteCredentials(ctx context.Context, c client.Client, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string) error {
	credentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: controlPlaneNamespace,
			Name:      "gcp-cloud-credentials",
		},
	}

	err := c.Delete(ctx, credentialsSecret)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete GCP credentials secret: %w", err)
	}

	return nil
}