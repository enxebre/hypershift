package gcp

import (
	"context"
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/nodepool/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGCPNodePoolPlatformOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *GCPNodePoolPlatformOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: false,
		},
		{
			name: "missing machine type",
			opts: &GCPNodePoolPlatformOptions{
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP machine type is required",
		},
		{
			name: "invalid disk type",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:       "n1-standard-4",
				DiskType:          "invalid-type",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "invalid GCP disk type",
		},
		{
			name: "disk size too small",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        10,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP disk size must be between 20 and 65536 GB",
		},
		{
			name: "disk size too large",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        70000,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP disk size must be between 20 and 65536 GB",
		},
		{
			name: "invalid maintenance option",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "INVALID",
			},
			wantErr: true,
			errMsg:  "invalid GCP onHostMaintenance value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGCPNodePoolPlatformOptions_UpdateNodePool(t *testing.T) {
	tests := []struct {
		name     string
		opts     *GCPNodePoolPlatformOptions
		nodePool *hyperv1.NodePool
		verify   func(t *testing.T, np *hyperv1.NodePool)
	}{
		{
			name: "basic configuration",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				Subnetwork:        "test-subnet",
				OnHostMaintenance: "MIGRATE",
			},
			nodePool: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-nodepool",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
					},
				},
			},
			verify: func(t *testing.T, np *hyperv1.NodePool) {
				assert.NotNil(t, np.Spec.Platform.GCP)
				assert.Equal(t, "n1-standard-4", np.Spec.Platform.GCP.MachineType)
				assert.Equal(t, "pd-standard", np.Spec.Platform.GCP.DiskType)
				assert.Equal(t, int64(100), np.Spec.Platform.GCP.DiskSizeGb)
				assert.Equal(t, "test-subnet", np.Spec.Platform.GCP.Subnetwork)
				assert.Equal(t, "MIGRATE", np.Spec.Platform.GCP.OnHostMaintenance)
			},
		},
		{
			name: "with service account and labels",
			opts: &GCPNodePoolPlatformOptions{
				MachineType:         "n1-standard-4",
				DiskType:            "pd-ssd",
				DiskSizeGb:          200,
				ServiceAccountEmail: "test@test-project.iam.gserviceaccount.com",
				ServiceAccountScopes: []string{
					"https://www.googleapis.com/auth/devstorage.read_only",
					"https://www.googleapis.com/auth/logging.write",
				},
				Labels: []string{"env=test", "team=platform"},
				Tags:   []string{"worker", "hypershift"},
				Preemptible: true,
			},
			nodePool: &hyperv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-nodepool",
				},
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.GCPPlatform,
					},
				},
			},
			verify: func(t *testing.T, np *hyperv1.NodePool) {
				assert.NotNil(t, np.Spec.Platform.GCP)
				assert.Equal(t, "n1-standard-4", np.Spec.Platform.GCP.MachineType)
				assert.Equal(t, "pd-ssd", np.Spec.Platform.GCP.DiskType)
				assert.Equal(t, int64(200), np.Spec.Platform.GCP.DiskSizeGb)
				assert.True(t, np.Spec.Platform.GCP.Preemptible)
				
				// Check service account
				assert.NotNil(t, np.Spec.Platform.GCP.ServiceAccount)
				assert.Equal(t, "test@test-project.iam.gserviceaccount.com", np.Spec.Platform.GCP.ServiceAccount.Email)
				assert.Equal(t, 2, len(np.Spec.Platform.GCP.ServiceAccount.Scopes))
				
				// Check labels
				assert.NotNil(t, np.Spec.Platform.GCP.Labels)
				assert.Equal(t, "test", np.Spec.Platform.GCP.Labels["env"])
				assert.Equal(t, "platform", np.Spec.Platform.GCP.Labels["team"])
				
				// Check tags
				assert.Equal(t, 2, len(np.Spec.Platform.GCP.Tags))
				assert.Contains(t, np.Spec.Platform.GCP.Tags, "worker")
				assert.Contains(t, np.Spec.Platform.GCP.Tags, "hypershift")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.UpdateNodePool(tt.nodePool)
			require.NoError(t, err)
			tt.verify(t, tt.nodePool)
		})
	}
}

func TestGCPPlatformCreateOptions_UpdateNodePool(t *testing.T) {
	opts := &GCPPlatformCreateOptions{
		GCPNodePoolPlatformOptions: &GCPNodePoolPlatformOptions{
			MachineType:       "n1-standard-4",
			DiskType:          "pd-standard",
			DiskSizeGb:        100,
			OnHostMaintenance: "MIGRATE",
		},
	}

	nodePool := &hyperv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-nodepool",
		},
		Spec: hyperv1.NodePoolSpec{
			Platform: hyperv1.NodePoolPlatform{},
		},
	}

	hostedCluster := &hyperv1.HostedCluster{
		Spec: hyperv1.HostedClusterSpec{
			Platform: hyperv1.PlatformSpec{
				Type: hyperv1.GCPPlatform,
				GCP: &hyperv1.GCPPlatformSpec{
					Project: "test-project",
					Region:  "us-central1",
					Network: &hyperv1.GCPNetworkSpec{
						Subnetwork: "cluster-subnet",
					},
					ServiceAccount: &hyperv1.GCPServiceAccount{
						Email: "cluster@test-project.iam.gserviceaccount.com",
						Scopes: []string{
							"https://www.googleapis.com/auth/devstorage.read_only",
						},
					},
				},
			},
		},
	}

	err := opts.UpdateNodePool(context.Background(), nodePool, hostedCluster)
	require.NoError(t, err)

	// Verify platform type is set
	assert.Equal(t, hyperv1.GCPPlatform, nodePool.Spec.Platform.Type)

	// Verify GCP configuration
	assert.NotNil(t, nodePool.Spec.Platform.GCP)
	assert.Equal(t, "n1-standard-4", nodePool.Spec.Platform.GCP.MachineType)
	assert.Equal(t, "pd-standard", nodePool.Spec.Platform.GCP.DiskType)
	assert.Equal(t, int64(100), nodePool.Spec.Platform.GCP.DiskSizeGb)

	// Verify defaults from HostedCluster
	assert.Equal(t, "cluster-subnet", nodePool.Spec.Platform.GCP.Subnetwork)
	assert.NotNil(t, nodePool.Spec.Platform.GCP.ServiceAccount)
	assert.Equal(t, "cluster@test-project.iam.gserviceaccount.com", nodePool.Spec.Platform.GCP.ServiceAccount.Email)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	
	assert.NotNil(t, opts)
	assert.Equal(t, "n1-standard-4", opts.MachineType)
	assert.Equal(t, "pd-standard", opts.DiskType)
	assert.Equal(t, int64(100), opts.DiskSizeGb)
	assert.Equal(t, "MIGRATE", opts.OnHostMaintenance)
	assert.NotEmpty(t, opts.ServiceAccountScopes)
	assert.Contains(t, opts.ServiceAccountScopes, "https://www.googleapis.com/auth/devstorage.read_only")
	assert.Contains(t, opts.ServiceAccountScopes, "https://www.googleapis.com/auth/logging.write")
	assert.Contains(t, opts.ServiceAccountScopes, "https://www.googleapis.com/auth/monitoring")
}

func TestGCPPlatformCreateOptions_Type(t *testing.T) {
	opts := &GCPPlatformCreateOptions{}
	assert.Equal(t, hyperv1.GCPPlatform, opts.Type())
}

func TestGCPPlatformCreateOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *GCPPlatformCreateOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &GCPPlatformCreateOptions{
				GCPNodePoolPlatformOptions: &GCPNodePoolPlatformOptions{
					MachineType:       "n1-standard-4",
					DiskType:          "pd-standard",
					DiskSizeGb:        100,
					OnHostMaintenance: "MIGRATE",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid options",
			opts: &GCPPlatformCreateOptions{
				GCPNodePoolPlatformOptions: &GCPNodePoolPlatformOptions{
					DiskType:          "pd-standard",
					DiskSizeGb:        100,
					OnHostMaintenance: "MIGRATE",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coreOpts := &core.CreateNodePoolOptions{}
			err := tt.opts.Validate(context.Background(), coreOpts)
			
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}