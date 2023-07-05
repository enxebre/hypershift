package routecm

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	configv1 "github.com/openshift/api/config/v1"
	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/imageprovider"
	"github.com/openshift/hypershift/support/config"
	"github.com/openshift/hypershift/support/globalconfig"
)

type OpenShiftRouteControllerManagerParams struct {
	OpenShiftControllerManagerImage string
	APIServer                       *configv1.APIServerSpec

	DeploymentConfig config.DeploymentConfig
	config.OwnerRef
}

func NewOpenShiftRouteControllerManagerParams(hcp *hyperv1.HostedControlPlane, observedConfig *globalconfig.ObservedConfig, releaseImageProvider *imageprovider.ReleaseImageProvider, setDefaultSecurityContext bool) *OpenShiftRouteControllerManagerParams {
	params := &OpenShiftRouteControllerManagerParams{
		OpenShiftControllerManagerImage: releaseImageProvider.GetImage("route-controller-manager"),
	}
	if hcp.Spec.Configuration != nil {
		params.APIServer = hcp.Spec.Configuration.APIServer
	}

	params.DeploymentConfig = *config.NewDeploymentConfig(hcp,
		"openshift-route-controller-manager",
		nil,
		setDefaultSecurityContext,
		false,
		config.DefaultPriorityClass,
		true,
	)

	params.DeploymentConfig.Resources = map[string]corev1.ResourceRequirements{
		routeOCMContainerMain().Name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	params.OwnerRef = config.OwnerRefFrom(hcp)
	return params
}
