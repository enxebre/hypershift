package gcp

import (
	"context"
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/upsert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/blang/semver"
)

func TestGCP_ReconcileCAPIInfraCR(t *testing.T) {
	testCases := []struct {
		name           string
		hcluster       *hyperv1.HostedCluster
		expectedError  bool
		validateResult func(t *testing.T, result client.Object)
	}{
		{
			name: "basic GCP cluster configuration",
			hcluster: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project",
							Region:  "us-central1",
							Zone:    "us-central1-a",
							Network: &hyperv1.GCPNetworkSpec{
								Network:    "test-network",
								Subnetwork: "test-subnet",
							},
							Labels: map[string]string{
								"environment": "test",
								"team":        "platform",
							},
						},
					},
				},
			},
			expectedError: false,
			validateResult: func(t *testing.T, result client.Object) {
				gcpCluster, ok := result.(*GCPCluster)
				if !ok {
					t.Fatalf("expected GCPCluster, got %T", result)
				}

				if gcpCluster.Spec.Project != "test-project" {
					t.Errorf("expected project 'test-project', got %s", gcpCluster.Spec.Project)
				}

				if gcpCluster.Spec.Region != "us-central1" {
					t.Errorf("expected region 'us-central1', got %s", gcpCluster.Spec.Region)
				}

				if gcpCluster.Spec.Network.Name != "test-network" {
					t.Errorf("expected network 'test-network', got %s", gcpCluster.Spec.Network.Name)
				}

				expectedLabels := map[string]string{
					"environment": "test",
					"team":        "platform",
				}
				for k, v := range expectedLabels {
					if gcpCluster.Spec.AdditionalLabels[k] != v {
						t.Errorf("expected label %s=%s, got %s", k, v, gcpCluster.Spec.AdditionalLabels[k])
					}
				}
			},
		},
		{
			name: "missing GCP platform configuration",
			hcluster: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP:  nil,
					},
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup fake client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = capiv1.AddToScheme(scheme)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			// Create GCP platform instance
			gcp := New("test-image", nil)

			// Create upsert function
			createOrUpdate := upsert.New(false)

			// Test the reconciliation
			result, err := gcp.ReconcileCAPIInfraCR(
				context.TODO(),
				client,
				createOrUpdate,
				tc.hcluster,
				"test-control-plane-namespace",
				hyperv1.APIEndpoint{
					Host: "api.test-cluster.example.com",
					Port: 6443,
				},
			)

			// Check expectations
			if tc.expectedError && err == nil {
				t.Fatal("expected an error, but got none")
			}
			if !tc.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.expectedError && tc.validateResult != nil {
				tc.validateResult(t, result)
			}
		})
	}
}

func TestGCP_CAPIProviderDeploymentSpec(t *testing.T) {
	testCases := []struct {
		name        string
		hcluster    *hyperv1.HostedCluster
		annotations map[string]string
		envVars     map[string]string
		expectImage string
	}{
		{
			name: "default image",
			hcluster: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
			},
			expectImage: "test-default-image",
		},
		{
			name: "annotation override",
			hcluster: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						hyperv1.ClusterAPIGCPProviderImage: "test-annotation-image",
					},
				},
			},
			expectImage: "test-annotation-image",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment variables if specified
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			// Create GCP platform instance
			gcp := New("test-default-image", nil)

			// Test the deployment spec generation
			deploymentSpec, err := gcp.CAPIProviderDeploymentSpec(tc.hcluster, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate the image
			if len(deploymentSpec.Template.Spec.Containers) == 0 {
				t.Fatal("expected at least one container")
			}

			actualImage := deploymentSpec.Template.Spec.Containers[0].Image
			if actualImage != tc.expectImage {
				t.Errorf("expected image %s, got %s", tc.expectImage, actualImage)
			}

			// Validate basic container configuration
			container := deploymentSpec.Template.Spec.Containers[0]
			if container.Name != "manager" {
				t.Errorf("expected container name 'manager', got %s", container.Name)
			}

			// Check for required volume mounts
			expectedVolumeMounts := []string{"credentials", "capi-webhooks-tls"}
			for _, expected := range expectedVolumeMounts {
				found := false
				for _, mount := range container.VolumeMounts {
					if mount.Name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected volume mount %s not found", expected)
				}
			}

			// Check for required volumes
			expectedVolumes := []string{"credentials", "capi-webhooks-tls"}
			for _, expected := range expectedVolumes {
				found := false
				for _, volume := range deploymentSpec.Template.Spec.Volumes {
					if volume.Name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected volume %s not found", expected)
				}
			}
		})
	}
}

func TestGCP_ReconcileCredentials(t *testing.T) {
	testCases := []struct {
		name                   string
		existingSecret         *corev1.Secret
		expectedError          bool
		validateControlPlane   func(t *testing.T, secret *corev1.Secret)
	}{
		{
			name: "copy credentials secret successfully",
			existingSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gcp-cloud-credentials",
					Namespace: "test-namespace",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"service-account.json": []byte(`{"type": "service_account"}`),
				},
			},
			expectedError: false,
			validateControlPlane: func(t *testing.T, secret *corev1.Secret) {
				if secret.Type != corev1.SecretTypeOpaque {
					t.Errorf("expected secret type %s, got %s", corev1.SecretTypeOpaque, secret.Type)
				}
				expectedData := `{"type": "service_account"}`
				if string(secret.Data["service-account.json"]) != expectedData {
					t.Errorf("expected data %s, got %s", expectedData, string(secret.Data["service-account.json"]))
				}
			},
		},
		{
			name:           "missing credentials secret",
			existingSecret: nil,
			expectedError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup fake client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)

			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			if tc.existingSecret != nil {
				clientBuilder = clientBuilder.WithObjects(tc.existingSecret)
			}
			client := clientBuilder.Build()

			// Create GCP platform instance
			gcp := New("test-image", nil)

			// Create hosted cluster
			hcluster := &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
			}

			// Create upsert function
			createOrUpdate := upsert.New(false)

			// Test credentials reconciliation
			err := gcp.ReconcileCredentials(
				context.TODO(),
				client,
				createOrUpdate,
				hcluster,
				"test-control-plane-namespace",
			)

			// Check expectations
			if tc.expectedError && err == nil {
				t.Fatal("expected an error, but got none")
			}
			if !tc.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate the control plane secret if no error expected
			if !tc.expectedError && tc.validateControlPlane != nil {
				controlPlaneSecret := &corev1.Secret{}
				err := client.Get(context.TODO(), client.ObjectKey{
					Namespace: "test-control-plane-namespace",
					Name:      "gcp-cloud-credentials",
				}, controlPlaneSecret)
				if err != nil {
					t.Fatalf("failed to get control plane secret: %v", err)
				}
				tc.validateControlPlane(t, controlPlaneSecret)
			}
		})
	}
}

func TestGCP_CAPIProviderPolicyRules(t *testing.T) {
	gcp := New("test-image", nil)

	rules := gcp.CAPIProviderPolicyRules()

	// Check that we have the expected number of rules
	if len(rules) != 3 {
		t.Errorf("expected 3 policy rules, got %d", len(rules))
	}

	// Validate specific rules exist
	expectedAPIGroups := map[string]bool{
		"infrastructure.cluster.x-k8s.io": false,
		"":                               false,
		"cluster.x-k8s.io":               false,
	}

	for _, rule := range rules {
		for _, apiGroup := range rule.APIGroups {
			if _, exists := expectedAPIGroups[apiGroup]; exists {
				expectedAPIGroups[apiGroup] = true
			}
		}
	}

	for apiGroup, found := range expectedAPIGroups {
		if !found {
			t.Errorf("expected API group %s not found in policy rules", apiGroup)
		}
	}
}

func TestNew(t *testing.T) {
	testVersion, _ := semver.Parse("4.12.0")

	gcp := New("test-image", testVersion)

	if gcp.capiProviderImage != "test-image" {
		t.Errorf("expected capiProviderImage 'test-image', got %s", gcp.capiProviderImage)
	}

	if gcp.payloadVersion.String() != "4.12.0" {
		t.Errorf("expected payloadVersion '4.12.0', got %s", gcp.payloadVersion.String())
	}
}