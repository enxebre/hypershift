package gcp

import (
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	component "github.com/openshift/hypershift/support/controlplane-component"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGCPComponentPredicate(t *testing.T) {
	tests := []struct {
		name        string
		platformType hyperv1.PlatformType
		expected    bool
	}{
		{
			name:        "GCP platform should match",
			platformType: hyperv1.GCPPlatform,
			expected:    true,
		},
		{
			name:        "AWS platform should not match",
			platformType: hyperv1.AWSPlatform,
			expected:    false,
		},
		{
			name:        "Azure platform should not match", 
			platformType: hyperv1.AzurePlatform,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcp := &hyperv1.HostedControlPlane{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hcp",
					Namespace: "test-namespace",
				},
				Spec: hyperv1.HostedControlPlaneSpec{
					Platform: hyperv1.PlatformSpec{
						Type: tt.platformType,
					},
				},
			}

			cpContext := component.WorkloadContext{
				HCP: hcp,
			}

			result, err := predicate(cpContext)
			if err != nil {
				t.Errorf("predicate() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("predicate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewComponent(t *testing.T) {
	component := NewComponent()
	if component == nil {
		t.Error("NewComponent() returned nil")
	}
}