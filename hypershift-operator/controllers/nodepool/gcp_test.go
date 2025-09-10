package nodepool

import (
	"context"
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/releaseinfo"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	capigcp "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"

	"github.com/google/go-cmp/cmp"
)

const gcpImageName = "test-image"
const gcpInfraName = "test-gcp"

func TestGCPMachineTemplateSpec(t *testing.T) {
	testCases := []struct {
		name         string
		cluster      hyperv1.HostedClusterSpec
		nodePool     hyperv1.NodePoolSpec
		expectedSpec *capigcp.GCPMachineTemplateSpec
		expectError  bool
	}{
		{
			name: "minimal valid configuration",
			cluster: hyperv1.HostedClusterSpec{
				Platform: hyperv1.PlatformSpec{
					Type: hyperv1.GCPPlatform,
					GCP: &hyperv1.GCPPlatformSpec{
						Project: "test-project",
						Region:  "us-central1",
					},
				},
			},
			nodePool: hyperv1.NodePoolSpec{
				Platform: hyperv1.NodePoolPlatform{
					Type: hyperv1.GCPPlatform,
					GCP: &hyperv1.GCPNodePoolPlatform{
						MachineType: "n1-standard-4",
						Image:       gcpImageName,
					},
				},
			},
			expectedSpec: &capigcp.GCPMachineTemplateSpec{
				Template: capigcp.GCPMachineTemplateResource{
					Spec: capigcp.GCPMachineSpec{
						InstanceType:          "n1-standard-4",
						Image:                 ptr.To(gcpImageName),
						RootDeviceType:        (*capigcp.DiskType)(ptr.To("pd-standard")),
						RootDeviceSize:        int64(100),
						Subnet:                ptr.To(""),
						AdditionalNetworkTags: []string{"kubernetes-io-cluster-" + gcpInfraName},
						AdditionalLabels:      capigcp.Labels{"kubernetes-io-cluster-" + gcpInfraName: "owned"},
						Preemptible:           false,
						OnHostMaintenance:     (*capigcp.HostMaintenancePolicy)(ptr.To("MIGRATE")),
					},
				},
			},
			expectError: false,
		},
		{
			name: "with custom configuration",
			cluster: hyperv1.HostedClusterSpec{
				Platform: hyperv1.PlatformSpec{
					Type: hyperv1.GCPPlatform,
					GCP: &hyperv1.GCPPlatformSpec{
						Project: "test-project",
						Region:  "us-central1",
						Network: &hyperv1.GCPNetworkSpec{
							Subnetwork: "test-subnet",
						},
					},
				},
			},
			nodePool: hyperv1.NodePoolSpec{
				Platform: hyperv1.NodePoolPlatform{
					Type: hyperv1.GCPPlatform,
					GCP: &hyperv1.GCPNodePoolPlatform{
						MachineType:       "n1-standard-2",
						Image:             gcpImageName,
						DiskType:          "pd-ssd",
						DiskSizeGb:        200,
						Subnetwork:        "custom-subnet",
						Preemptible:       true,
						OnHostMaintenance: "TERMINATE",
						Tags:              []string{"custom-tag"},
						Labels:            map[string]string{"env": "test"},
						ServiceAccount: &hyperv1.GCPServiceAccount{
							Email: "test@test-project.iam.gserviceaccount.com",
							Scopes: []string{
								"https://www.googleapis.com/auth/cloud-platform",
							},
						},
					},
				},
			},
			expectedSpec: &capigcp.GCPMachineTemplateSpec{
				Template: capigcp.GCPMachineTemplateResource{
					Spec: capigcp.GCPMachineSpec{
						InstanceType:   "n1-standard-2",
						Image:          ptr.To(gcpImageName),
						RootDeviceType: (*capigcp.DiskType)(ptr.To("pd-ssd")),
						RootDeviceSize: int64(200),
						Subnet:         ptr.To("custom-subnet"),
						ServiceAccount: &capigcp.ServiceAccount{
							Email: "test@test-project.iam.gserviceaccount.com",
							Scopes: []string{
								"https://www.googleapis.com/auth/cloud-platform",
							},
						},
						AdditionalNetworkTags: []string{"custom-tag", "kubernetes-io-cluster-" + gcpInfraName},
						AdditionalLabels: capigcp.Labels{
							"env":                                           "test",
							"kubernetes-io-cluster-" + gcpInfraName: "owned",
						},
						Preemptible:       true,
						OnHostMaintenance: (*capigcp.HostMaintenancePolicy)(ptr.To("TERMINATE")),
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing machine type should error",
			cluster: hyperv1.HostedClusterSpec{
				Platform: hyperv1.PlatformSpec{
					Type: hyperv1.GCPPlatform,
					GCP: &hyperv1.GCPPlatformSpec{
						Project: "test-project",
						Region:  "us-central1",
					},
				},
			},
			nodePool: hyperv1.NodePoolSpec{
				Platform: hyperv1.NodePoolPlatform{
					Type: hyperv1.GCPPlatform,
					GCP: &hyperv1.GCPNodePoolPlatform{
						Image: gcpImageName,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hostedCluster := &hyperv1.HostedCluster{
				Spec: tc.cluster,
			}
			nodePool := &hyperv1.NodePool{
				Spec: tc.nodePool,
			}

			// Mock release image - in real implementation this would come from release metadata
			releaseImage := &releaseinfo.ReleaseImage{}

			spec, err := gcpMachineTemplateSpec(gcpInfraName, hostedCluster, nodePool, releaseImage)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(tc.expectedSpec, spec); diff != "" {
				t.Errorf("GCP machine template spec differs from expected:\n%s", diff)
			}
		})
	}
}

func TestGCPConditions(t *testing.T) {
	testCases := []struct {
		name                 string
		nodePool             *hyperv1.NodePool
		hostedCluster        *hyperv1.HostedCluster
		expectCondition      bool
		expectedConditionType string
		expectedStatus       corev1.ConditionStatus
	}{
		{
			name: "valid GCP configuration",
			nodePool: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-nodepool",
					Generation: 1,
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							Image:       gcpImageName,
						},
					},
					Release: hyperv1.Release{
						Image: "test-release-image",
					},
				},
			},
			hostedCluster: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project",
							Region:  "us-central1",
						},
					},
				},
			},
			expectCondition:       false, // Currently we remove conditions for user-defined images
			expectedConditionType: string(hyperv1.NodePoolValidPlatformImageType),
			expectedStatus:        corev1.ConditionTrue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reconciler := &NodePoolReconciler{}
			
			// Mock release image - in real tests this would be properly mocked
			releaseImage := &releaseinfo.ReleaseImage{}
			
			err := reconciler.setGCPConditions(context.TODO(), tc.nodePool, tc.hostedCluster, "test-namespace", releaseImage)
			
			if err != nil {
				t.Errorf("unexpected error setting GCP conditions: %v", err)
				return
			}

			if tc.expectCondition {
				found := false
				for _, condition := range tc.nodePool.Status.Conditions {
					if string(condition.Type) == tc.expectedConditionType {
						found = true
						if condition.Status != tc.expectedStatus {
							t.Errorf("expected condition status %v, got %v", tc.expectedStatus, condition.Status)
						}
						break
					}
				}
				if !found {
					t.Errorf("expected condition %v not found", tc.expectedConditionType)
				}
			}
		})
	}
}

func TestValidateGCPPlatformConfig(t *testing.T) {
	testCases := []struct {
		name        string
		nodePool    *hyperv1.NodePool
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							DiskType:    "pd-ssd",
							DiskSizeGb:  100,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing machine type",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP:  &hyperv1.GCPNodePoolPlatform{},
					},
				},
			},
			expectError: true,
			errorMsg:    "GCP machine type is required",
		},
		{
			name: "invalid disk type",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							DiskType:    "invalid-disk-type",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid GCP disk type",
		},
		{
			name: "disk size too small",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							DiskSizeGb:  10, // Below minimum of 20
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "GCP disk size must be between 20 and 65536 GB",
		},
		{
			name: "invalid onHostMaintenance",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType:       "n1-standard-4",
							OnHostMaintenance: "INVALID",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid GCP onHostMaintenance value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reconciler := &NodePoolReconciler{}
			
			err := reconciler.validateGCPPlatformConfig(context.TODO(), tc.nodePool, &hyperv1.HostedCluster{}, nil)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tc.errorMsg != "" && !contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}