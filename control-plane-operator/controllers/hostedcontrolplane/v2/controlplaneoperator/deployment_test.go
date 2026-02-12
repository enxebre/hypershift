package controlplaneoperator

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	assets "github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/v2/assets"
	controlplanecomponent "github.com/openshift/hypershift/support/controlplane-component"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdaptDeployment(t *testing.T) {
	testCases := []struct {
		name                      string
		hcAnnotations             map[string]string
		expectedImageOverridesArg string
	}{
		{
			name:                      "When image-overrides annotation is set it should include the arg with value",
			hcAnnotations:             map[string]string{hyperv1.ImageOverridesAnnotation: "registry.example.com/image=quay.io/image"},
			expectedImageOverridesArg: fmt.Sprintf("--image-overrides=%s", "registry.example.com/image=quay.io/image"),
		},
		{
			name:                      "When image-overrides annotation is empty it should include the arg with empty value",
			hcAnnotations:             nil,
			expectedImageOverridesArg: "--image-overrides=",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGomegaWithT(t)

			hc := &hyperv1.HostedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "hc",
					Namespace:   "hc-namespace",
					Annotations: tc.hcAnnotations,
				},
			}

			hcp := &hyperv1.HostedControlPlane{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hcp",
					Namespace: "hcp-namespace",
				},
				Spec: hyperv1.HostedControlPlaneSpec{
					Platform: hyperv1.PlatformSpec{
						Type: hyperv1.NonePlatform,
					},
				},
			}

			cpo := &ControlPlaneOperatorOptions{
				HostedCluster: hc,
				Image:         "test-image",
			}

			cpContext := controlplanecomponent.WorkloadContext{
				Context: t.Context(),
				HCP:     hcp,
			}

			deployment, err := assets.LoadDeploymentManifest(ComponentName)
			g.Expect(err).ToNot(HaveOccurred())

			err = cpo.adaptDeployment(cpContext, deployment)
			g.Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == ComponentName {
					for _, arg := range container.Args {
						if arg == tc.expectedImageOverridesArg {
							found = true
							break
						}
					}
				}
			}
			g.Expect(found).To(BeTrue(), "expected arg %q not found in container args", tc.expectedImageOverridesArg)
		})
	}
}
