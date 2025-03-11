package oapi

import (
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/config"

	configv1 "github.com/openshift/api/config/v1"

	corev1 "k8s.io/api/core/v1"
)

type OpenShiftAPIServerParams struct {
	APIServer               *configv1.APIServerSpec
	Proxy                   *configv1.ProxySpec
	IngressSubDomain        string
	EtcdURL                 string
	ServiceAccountIssuerURL string

	OpenShiftAPIServerDeploymentConfig      config.DeploymentConfig
	OpenShiftOAuthAPIServerDeploymentConfig config.DeploymentConfig
	config.OwnerRef
	OpenShiftAPIServerImage string
	OAuthAPIServerImage     string
	ProxyImage              string
	AvailabilityProberImage string
	Availability            hyperv1.AvailabilityPolicy
	Ingress                 *configv1.IngressSpec
	Image                   *configv1.ImageSpec
	Project                 *configv1.Project
	AuditWebhookRef         *corev1.LocalObjectReference
	InternalOAuthDisable    bool
}

type OpenShiftAPIServerServiceParams struct {
	OwnerRef config.OwnerRef `json:"ownerRef"`
}

func NewOpenShiftAPIServerServiceParams(hcp *hyperv1.HostedControlPlane) *OpenShiftAPIServerServiceParams {
	return &OpenShiftAPIServerServiceParams{
		OwnerRef: config.OwnerRefFrom(hcp),
	}
}
