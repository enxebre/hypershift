package infra

import (
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/config"

	configv1 "github.com/openshift/api/config/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

const (
	OpenShiftAPIServerPort      = 8443
	OpenShiftOAuthAPIServerPort = 8443
	OpenShiftServicePort        = 443
	OLMPackageServerPort        = 5443
)

var (
	oauthAPIServerLabels   = map[string]string{"app": "openshift-oauth-apiserver", hyperv1.ControlPlaneComponentLabel: "openshift-oauth-apiserver"}
	olmPackageServerLabels = map[string]string{"app": "packageserver", hyperv1.ControlPlaneComponentLabel: "packageserver"}
)

func openshiftAPIServerLabels() map[string]string {
	return map[string]string{"app": "openshift-apiserver", hyperv1.ControlPlaneComponentLabel: "openshift-apiserver"}
}

func ReconcileOpenShiftAPIService(svc *corev1.Service, ownerRef config.OwnerRef) error {
	return reconcileAPIService(svc, ownerRef, openshiftAPIServerLabels(), OpenShiftAPIServerPort)
}

func ReconcileOAuthAPIService(svc *corev1.Service, ownerRef config.OwnerRef) error {
	return reconcileAPIService(svc, ownerRef, oauthAPIServerLabels, OpenShiftAPIServerPort)
}

func ReconcileOLMPackageServerService(svc *corev1.Service, ownerRef config.OwnerRef) error {
	return reconcileAPIService(svc, ownerRef, olmPackageServerLabels, OLMPackageServerPort)
}

func reconcileAPIService(svc *corev1.Service, ownerRef config.OwnerRef, labels map[string]string, targetPort int) error {
	ownerRef.ApplyTo(svc)
	svc.Labels = openshiftAPIServerLabels()
	if svc.Spec.Selector == nil {
		svc.Spec.Selector = labels
	}
	var portSpec corev1.ServicePort
	if len(svc.Spec.Ports) > 0 {
		portSpec = svc.Spec.Ports[0]
	} else {
		svc.Spec.Ports = []corev1.ServicePort{portSpec}
	}
	portSpec.Name = "https"
	portSpec.Port = int32(OpenShiftServicePort)
	portSpec.Protocol = corev1.ProtocolTCP
	portSpec.TargetPort = intstr.FromInt(targetPort)
	svc.Spec.Type = corev1.ServiceTypeClusterIP
	svc.Spec.Ports[0] = portSpec
	return nil
}
