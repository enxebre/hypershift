package olm

import (
	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/imageprovider"
	"github.com/openshift/hypershift/support/config"
	"github.com/openshift/hypershift/support/util"
	"k8s.io/utils/pointer"
)

var packageServerLabels = map[string]string{
	"app":                         "packageserver",
	hyperv1.ControlPlaneComponent: "packageserver",
}

type OperatorLifecycleManagerParams struct {
	CLIImage                string
	OLMImage                string
	ProxyImage              string
	OperatorRegistryImage   string
	ReleaseVersion          string
	DeploymentConfig        config.DeploymentConfig
	PackageServerConfig     config.DeploymentConfig
	AvailabilityProberImage string
	NoProxy                 []string
	config.OwnerRef
}

func NewOperatorLifecycleManagerParams(hcp *hyperv1.HostedControlPlane, releaseImageProvider *imageprovider.ReleaseImageProvider, releaseVersion string, setDefaultSecurityContext bool) *OperatorLifecycleManagerParams {
	params := &OperatorLifecycleManagerParams{
		CLIImage:                releaseImageProvider.GetImage("cli"),
		OLMImage:                releaseImageProvider.GetImage("operator-lifecycle-manager"),
		ProxyImage:              releaseImageProvider.GetImage("socks5-proxy"),
		OperatorRegistryImage:   releaseImageProvider.GetImage("operator-registry"),
		ReleaseVersion:          releaseVersion,
		AvailabilityProberImage: releaseImageProvider.GetImage(util.AvailabilityProberImageName),
		NoProxy:                 []string{"kube-apiserver"},
		OwnerRef:                config.OwnerRefFrom(hcp),
	}

	params.DeploymentConfig = *config.NewDeploymentConfig(hcp,
		"olm-operator",
		pointer.Int(1),
		setDefaultSecurityContext,
		false,
		config.DefaultPriorityClass,
		true,
	)

	params.PackageServerConfig = *config.NewDeploymentConfig(hcp,
		"packageserver",
		pointer.Int(1),
		setDefaultSecurityContext,
		false,
		config.DefaultPriorityClass,
		true,
	)

	if hcp.Spec.OLMCatalogPlacement == "management" {
		params.NoProxy = append(params.NoProxy, "certified-operators", "community-operators", "redhat-operators", "redhat-marketplace")
	}

	return params
}
