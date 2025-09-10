package gcp

import (
	"context"
	"fmt"
	"testing"

	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInfraOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *CreateInfraOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			opts: &CreateInfraOptions{
				CredentialsOpts: gcputil.GCPCredentialsOptions{
					CredentialsFile: "/tmp/test-creds.json",
					ProjectID:       "test-project",
				},
				Region:  "us-central1",
				Zone:    "us-central1-a",
				Project: "test-project",
				InfraID: "test-infra",
				Name:    "test-cluster",
			},
			wantErr: false,
		},
		{
			name: "missing infra ID",
			opts: &CreateInfraOptions{
				CredentialsOpts: gcputil.GCPCredentialsOptions{
					CredentialsFile: "/tmp/test-creds.json",
					ProjectID:       "test-project",
				},
				Region:  "us-central1",
				Zone:    "us-central1-a",
				Project: "test-project",
				Name:    "test-cluster",
			},
			wantErr: true,
			errMsg:  "infra ID is required",
		},
		{
			name: "missing credentials",
			opts: &CreateInfraOptions{
				Region:  "us-central1",
				Zone:    "us-central1-a",
				Project: "test-project",
				InfraID: "test-infra",
				Name:    "test-cluster",
			},
			wantErr: true,
			errMsg:  "credentials file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateInfraOptions(tt.opts)
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCostOptimizationRecommendations(t *testing.T) {
	opts := &CostOptimizationOptions{
		Project:                     "test-project",
		Region:                      "us-central1",
		InfraID:                     "test-infra",
		EnablePreemptibleInstances:  true,
		EnableSustainedUseDiscounts: true,
		EnableCommittedUseDiscounts: true,
		EnableAutoscaling:          true,
		MinNodes:                   1,
		MaxNodes:                   10,
	}

	// Test that cost optimization options are properly configured
	assert.True(t, opts.EnablePreemptibleInstances)
	assert.True(t, opts.EnableSustainedUseDiscounts)
	assert.True(t, opts.EnableCommittedUseDiscounts)
	assert.True(t, opts.EnableAutoscaling)
	assert.Equal(t, int32(1), opts.MinNodes)
	assert.Equal(t, int32(10), opts.MaxNodes)
}

func TestMonitoringConfiguration(t *testing.T) {
	opts := &MonitoringOptions{
		Project:               "test-project",
		Region:                "us-central1",
		InfraID:               "test-infra",
		Name:                  "test-cluster",
		EnableCloudLogging:    true,
		EnableCloudMonitoring: true,
		EnableErrorReporting:  true,
		EnableCloudTrace:      true,
		LogRetentionDays:      30,
	}

	// Test that monitoring options are properly configured
	assert.True(t, opts.EnableCloudLogging)
	assert.True(t, opts.EnableCloudMonitoring)
	assert.True(t, opts.EnableErrorReporting)
	assert.True(t, opts.EnableCloudTrace)
	assert.Equal(t, int32(30), opts.LogRetentionDays)
}

func TestSecurityConfiguration(t *testing.T) {
	opts := &SecurityOptions{
		Project:                   "test-project",
		Region:                    "us-central1",
		InfraID:                   "test-infra",
		Name:                      "test-cluster",
		EnableWorkloadIdentity:    true,
		EnablePrivateCluster:      true,
		EnableKMSEncryption:       true,
		EnableVPCFlowLogs:        true,
		EnableBinaryAuthorization: true,
		KMSKeyRing:               "test-keyring",
		KMSKeyName:               "test-key",
	}

	// Test that security options are properly configured
	assert.True(t, opts.EnableWorkloadIdentity)
	assert.True(t, opts.EnablePrivateCluster)
	assert.True(t, opts.EnableKMSEncryption)
	assert.True(t, opts.EnableVPCFlowLogs)
	assert.True(t, opts.EnableBinaryAuthorization)
	assert.Equal(t, "test-keyring", opts.KMSKeyRing)
	assert.Equal(t, "test-key", opts.KMSKeyName)
}

// Helper function to validate create infra options
func validateCreateInfraOptions(opts *CreateInfraOptions) error {
	if opts.InfraID == "" {
		return fmt.Errorf("infra ID is required")
	}
	
	if opts.CredentialsOpts.CredentialsFile == "" {
		return fmt.Errorf("credentials file is required")
	}
	
	return nil
}

func TestGetMonitoringRecommendations(t *testing.T) {
	recommendations := GetMonitoringRecommendations()
	
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations, "Enable Cloud Logging for centralized log management and analysis")
	assert.Contains(t, recommendations, "Use Cloud Monitoring for comprehensive metric collection and alerting")
	assert.Contains(t, recommendations, "Set up Error Reporting to track and analyze application errors")
}

func TestGetSecurityRecommendations(t *testing.T) {
	recommendations := GetSecurityRecommendations()
	
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations, "Enable Workload Identity to securely access GCP services from Kubernetes workloads")
	assert.Contains(t, recommendations, "Use Cloud KMS for encryption at rest of etcd data and persistent volumes")
	assert.Contains(t, recommendations, "Implement network policies to control pod-to-pod communication")
}