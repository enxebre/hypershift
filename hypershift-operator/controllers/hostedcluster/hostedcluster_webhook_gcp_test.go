package hostedcluster

import (
	"context"
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateCreateGCPHostedCluster(t *testing.T) {
	validator := hostedClusterValidator{}

	testCases := []struct {
		name          string
		hc            *hyperv1.HostedCluster
		expectedError bool
		errorContains string
	}{
		{
			name: "valid GCP configuration",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
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
		},
		{
			name: "missing GCP configuration",
			hc: &hyperv1.HostedCluster{
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
			errorContains: "GCP platform configuration is required",
		},
		{
			name: "missing project",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Region: "us-central1",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP project is required",
		},
		{
			name: "missing region",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP region is required",
		},
		{
			name: "invalid project ID - too short",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test",
							Region:  "us-central1",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP project ID must be between 6 and 30 characters",
		},
		{
			name: "invalid project ID - too long",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "this-is-a-very-long-project-id-that-exceeds-the-maximum-length",
							Region:  "us-central1",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP project ID must be between 6 and 30 characters",
		},
		{
			name: "invalid region format",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP region format is invalid",
		},
		{
			name: "invalid zone format",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
							Zone:    "us",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP zone format is invalid",
		},
		{
			name: "invalid label key - too long",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
							Labels: map[string]string{
								"this-is-a-very-long-label-key-that-exceeds-the-maximum-length-allowed": "value",
							},
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP label key",
		},
		{
			name: "invalid label value - too long",
			hc: &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
							Labels: map[string]string{
								"key": "this-is-a-very-long-label-value-that-exceeds-the-maximum-length-allowed",
							},
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP label value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.validateCreateGCPHostedCluster(context.TODO(), tc.hc)

			if tc.expectedError && err == nil {
				t.Fatalf("expected an error, but got none")
			}
			if !tc.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectedError && err != nil && tc.errorContains != "" {
				if !contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error to contain '%s', but got: %v", tc.errorContains, err)
				}
			}
		})
	}
}

func TestValidateUpdateGCPHostedCluster(t *testing.T) {
	validator := hostedClusterValidator{}

	testCases := []struct {
		name          string
		oldHC         *hyperv1.HostedCluster
		newHC         *hyperv1.HostedCluster
		expectedError bool
		errorContains string
	}{
		{
			name: "valid update - mutable fields changed",
			oldHC: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
							Labels: map[string]string{
								"environment": "test",
							},
						},
					},
				},
			},
			newHC: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
							Labels: map[string]string{
								"environment": "production",
								"team":        "platform",
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "invalid update - project changed",
			oldHC: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
						},
					},
				},
			},
			newHC: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "different-project-456",
							Region:  "us-central1",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP project is immutable",
		},
		{
			name: "invalid update - region changed",
			oldHC: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-central1",
						},
					},
				},
			},
			newHC: &hyperv1.HostedCluster{
				Spec: hyperv1.HostedClusterSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPPlatformSpec{
							Project: "test-project-123",
							Region:  "us-east1",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP region is immutable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.validateUpdateGCPHostedCluster(context.TODO(), tc.oldHC, tc.newHC)

			if tc.expectedError && err == nil {
				t.Fatalf("expected an error, but got none")
			}
			if !tc.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectedError && err != nil && tc.errorContains != "" {
				if !contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error to contain '%s', but got: %v", tc.errorContains, err)
				}
			}
		})
	}
}

func TestValidateCreateGCPNodePool(t *testing.T) {
	validator := nodePoolValidator{}

	testCases := []struct {
		name          string
		np            *hyperv1.NodePool
		expectedError bool
		errorContains string
	}{
		{
			name: "valid GCP nodepool configuration",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType:       "n1-standard-4",
							DiskType:          "pd-ssd",
							DiskSizeGb:        100,
							OnHostMaintenance: "MIGRATE",
							Labels: map[string]string{
								"role": "worker",
							},
							Tags: []string{"worker", "production"},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "missing GCP configuration",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP:  nil,
					},
				},
			},
			expectedError: true,
			errorContains: "GCP nodepool platform configuration is required",
		},
		{
			name: "missing machine type",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							DiskType:   "pd-ssd",
							DiskSizeGb: 100,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP machine type is required",
		},
		{
			name: "invalid machine type format",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1",
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP machine type format is invalid",
		},
		{
			name: "invalid disk size - too small",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							DiskSizeGb:  10,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP disk size must be between 20 and 65536 GB",
		},
		{
			name: "invalid disk size - too large",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							DiskSizeGb:  100000,
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP disk size must be between 20 and 65536 GB",
		},
		{
			name: "invalid disk type",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
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
			expectedError: true,
			errorContains: "GCP disk type must be one of: pd-ssd, pd-standard, pd-balanced",
		},
		{
			name: "invalid onHostMaintenance",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
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
			expectedError: true,
			errorContains: "GCP onHostMaintenance must be either MIGRATE or TERMINATE",
		},
		{
			name: "too many tags",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							Tags:        make([]string, 65), // 65 tags, exceeds limit
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP tags cannot exceed 64 items",
		},
		{
			name: "invalid tag - too long",
			np: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodepool",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
						GCP: &hyperv1.GCPNodePoolPlatform{
							MachineType: "n1-standard-4",
							Tags:        []string{"this-is-a-very-long-tag-name-that-exceeds-the-maximum-length-allowed"},
						},
					},
				},
			},
			expectedError: true,
			errorContains: "GCP tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.validateCreateGCPNodePool(context.TODO(), tc.np)

			if tc.expectedError && err == nil {
				t.Fatalf("expected an error, but got none")
			}
			if !tc.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectedError && err != nil && tc.errorContains != "" {
				if !contains(err.Error(), tc.errorContains) {
					t.Fatalf("expected error to contain '%s', but got: %v", tc.errorContains, err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || func() bool {
		for i := 1; i < len(s)-len(substr)+1; i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())))
}