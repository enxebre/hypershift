package gcp

import (
	"context"
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGCPOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *RawCreateOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			opts: &RawCreateOptions{
				Project:           "test-project",
				Region:            "us-central1",
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: false,
		},
		{
			name: "missing project",
			opts: &RawCreateOptions{
				Region:            "us-central1",
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP project must be specified",
		},
		{
			name: "missing region",
			opts: &RawCreateOptions{
				Project:           "test-project",
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP region must be specified",
		},
		{
			name: "missing machine type",
			opts: &RawCreateOptions{
				Project:           "test-project",
				Region:            "us-central1",
				DiskType:          "pd-standard",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP machine type must be specified",
		},
		{
			name: "invalid disk type",
			opts: &RawCreateOptions{
				Project:           "test-project",
				Region:            "us-central1",
				MachineType:       "n1-standard-4",
				DiskType:          "invalid-type",
				DiskSizeGb:        100,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "invalid GCP disk type",
		},
		{
			name: "invalid disk size - too small",
			opts: &RawCreateOptions{
				Project:           "test-project",
				Region:            "us-central1",
				MachineType:       "n1-standard-4",
				DiskType:          "pd-standard",
				DiskSizeGb:        10,
				OnHostMaintenance: "MIGRATE",
			},
			wantErr: true,
			errMsg:  "GCP disk size must be between 20 and 65536 GB",
		},
		{
			name: "invalid disk size - too large",
			opts: &RawCreateOptions{
				Project:           "test-project",
				Region:            "us-central1",
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
			opts: &RawCreateOptions{
				Project:           "test-project",
				Region:            "us-central1",
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
			coreOpts := &core.CreateOptions{}
			err := validateGCPOptions(context.Background(), coreOpts, tt.opts)
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePlatform(t *testing.T) {
	tests := []struct {
		name    string
		opts    *CompletedCreateOptions
		wantErr bool
		verify  func(t *testing.T, hc *hyperv1.HostedCluster)
	}{
		{
			name: "basic cluster creation",
			opts: &CompletedCreateOptions{
				completedCreateOptions: &completedCreateOptions{
					ValidatedCreateOptions: &ValidatedCreateOptions{
						validatedCreateOptions: &validatedCreateOptions{
							RawCreateOptions: &RawCreateOptions{
								Project:           "test-project",
								Region:            "us-central1",
								Zone:              "us-central1-a",
								MachineType:       "n1-standard-4",
								DiskType:          "pd-standard",
								DiskSizeGb:        100,
								OnHostMaintenance: "MIGRATE",
							},
						},
					},
					name:      "test-cluster",
					namespace: "test-namespace",
				},
			},
			wantErr: false,
			verify: func(t *testing.T, hc *hyperv1.HostedCluster) {
				assert.Equal(t, "test-cluster", hc.Name)
				assert.Equal(t, "test-namespace", hc.Namespace)
				assert.Equal(t, hyperv1.GCPPlatform, hc.Spec.Platform.Type)
				assert.NotNil(t, hc.Spec.Platform.GCP)
				assert.Equal(t, "test-project", hc.Spec.Platform.GCP.Project)
				assert.Equal(t, "us-central1", hc.Spec.Platform.GCP.Region)
				assert.Equal(t, "us-central1-a", hc.Spec.Platform.GCP.Zone)
			},
		},
		{
			name: "cluster with network configuration",
			opts: &CompletedCreateOptions{
				completedCreateOptions: &completedCreateOptions{
					ValidatedCreateOptions: &ValidatedCreateOptions{
						validatedCreateOptions: &validatedCreateOptions{
							RawCreateOptions: &RawCreateOptions{
								Project:                "test-project",
								Region:                 "us-central1",
								Zone:                   "us-central1-a",
								MachineType:            "n1-standard-4",
								DiskType:               "pd-standard",
								DiskSizeGb:             100,
								OnHostMaintenance:      "MIGRATE",
								Network:                "test-network",
								Subnetwork:             "test-subnet",
								ControlPlaneSubnetwork: "test-control-plane-subnet",
								LoadBalancerSubnetwork: "test-lb-subnet",
							},
						},
					},
					name:      "test-cluster",
					namespace: "test-namespace",
				},
			},
			wantErr: false,
			verify: func(t *testing.T, hc *hyperv1.HostedCluster) {
				assert.NotNil(t, hc.Spec.Platform.GCP.Network)
				assert.Equal(t, "test-network", hc.Spec.Platform.GCP.Network.Network)
				assert.Equal(t, "test-subnet", hc.Spec.Platform.GCP.Network.Subnetwork)
				assert.Equal(t, "test-control-plane-subnet", hc.Spec.Platform.GCP.Network.ControlPlaneSubnet)
				assert.Equal(t, "test-lb-subnet", hc.Spec.Platform.GCP.Network.LoadBalancerSubnet)
			},
		},
		{
			name: "cluster with labels and tags",
			opts: &CompletedCreateOptions{
				completedCreateOptions: &completedCreateOptions{
					ValidatedCreateOptions: &ValidatedCreateOptions{
						validatedCreateOptions: &validatedCreateOptions{
							RawCreateOptions: &RawCreateOptions{
								Project:           "test-project",
								Region:            "us-central1",
								Zone:              "us-central1-a",
								MachineType:       "n1-standard-4",
								DiskType:          "pd-standard",
								DiskSizeGb:        100,
								OnHostMaintenance: "MIGRATE",
								Labels:            []string{"env=test", "team=platform"},
								ResourceTags:      []string{"cost-center=engineering", "project=hypershift"},
							},
						},
					},
					name:      "test-cluster",
					namespace: "test-namespace",
				},
			},
			wantErr: false,
			verify: func(t *testing.T, hc *hyperv1.HostedCluster) {
				assert.NotNil(t, hc.Spec.Platform.GCP.Labels)
				assert.Equal(t, "test", hc.Spec.Platform.GCP.Labels["env"])
				assert.Equal(t, "platform", hc.Spec.Platform.GCP.Labels["team"])
				
				assert.NotNil(t, hc.Spec.Platform.GCP.ResourceTags)
				assert.Equal(t, "engineering", hc.Spec.Platform.GCP.ResourceTags["cost-center"])
				assert.Equal(t, "hypershift", hc.Spec.Platform.GCP.ResourceTags["project"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coreOpts := &core.CreateOptions{}
			hc, err := tt.opts.CreatePlatform(context.Background(), coreOpts)
			
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, hc)
				tt.verify(t, hc)
			}
		})
	}
}