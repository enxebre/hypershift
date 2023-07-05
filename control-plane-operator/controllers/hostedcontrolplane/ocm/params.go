package ocm

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	configv1 "github.com/openshift/api/config/v1"
	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/imageprovider"
	"github.com/openshift/hypershift/support/config"
	"github.com/openshift/hypershift/support/globalconfig"
)

type OpenShiftControllerManagerParams struct {
	OpenShiftControllerManagerImage string
	DockerBuilderImage              string
	DeployerImage                   string
	APIServer                       *configv1.APIServerSpec
	Network                         *configv1.NetworkSpec
	Build                           *configv1.Build
	Image                           *configv1.Image

	DeploymentConfig config.DeploymentConfig
	config.OwnerRef
}

func NewOpenShiftControllerManagerParams(hcp *hyperv1.HostedControlPlane, observedConfig *globalconfig.ObservedConfig, releaseImageProvider *imageprovider.ReleaseImageProvider, setDefaultSecurityContext bool) *OpenShiftControllerManagerParams {
	params := &OpenShiftControllerManagerParams{
		OpenShiftControllerManagerImage: releaseImageProvider.GetImage("openshift-controller-manager"),
		DockerBuilderImage:              releaseImageProvider.GetImage("docker-builder"),
		DeployerImage:                   releaseImageProvider.GetImage("deployer"),
		Build:                           observedConfig.Build,
		Image:                           observedConfig.Image,
	}
	if hcp.Spec.Configuration != nil {
		params.APIServer = hcp.Spec.Configuration.APIServer
		params.Network = hcp.Spec.Configuration.Network
	}

	params.DeploymentConfig = *config.NewDeploymentConfig(hcp,
		"openshift-controller-manager",
		nil,
		setDefaultSecurityContext,
		false,
		config.DefaultPriorityClass,
		true,
	)

	params.DeploymentConfig.Resources = map[string]corev1.ResourceRequirements{
		ocmContainerMain().Name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	params.OwnerRef = config.OwnerRefFrom(hcp)
	return params
}

func (p *OpenShiftControllerManagerParams) MinTLSVersion() string {
	if p.APIServer != nil {
		return config.MinTLSVersion(p.APIServer.TLSSecurityProfile)
	}
	return config.MinTLSVersion(nil)
}

func (p *OpenShiftControllerManagerParams) CipherSuites() []string {
	if p.APIServer != nil {
		return config.CipherSuites(p.APIServer.TLSSecurityProfile)
	}
	return config.CipherSuites(nil)
}
