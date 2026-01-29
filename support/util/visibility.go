package util

import (
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	"k8s.io/utils/ptr"
)

func IsPrivateHCP(hcp *hyperv1.HostedControlPlane) bool {
	if hcp.Spec.Platform.Type == hyperv1.AWSPlatform {
		access := ptr.Deref(hcp.Spec.Platform.AWS, hyperv1.AWSPlatformSpec{}).EndpointAccess
		return access == hyperv1.PublicAndPrivate || access == hyperv1.Private
	}

	if hcp.Spec.Platform.Type == hyperv1.GCPPlatform {
		access := ptr.Deref(hcp.Spec.Platform.GCP, hyperv1.GCPPlatformSpec{}).EndpointAccess
		return access == hyperv1.GCPEndpointAccessPublicAndPrivate || access == hyperv1.GCPEndpointAccessPrivate
	}
	return false
}

func IsPublicHCP(hcp *hyperv1.HostedControlPlane) bool {
	if hcp.Spec.Platform.Type == hyperv1.AWSPlatform {
		access := ptr.Deref(hcp.Spec.Platform.AWS, hyperv1.AWSPlatformSpec{}).EndpointAccess
		return access == hyperv1.PublicAndPrivate || access == hyperv1.Public
	}
	if hcp.Spec.Platform.Type == hyperv1.GCPPlatform {
		access := ptr.Deref(hcp.Spec.Platform.GCP, hyperv1.GCPPlatformSpec{}).EndpointAccess
		return access == hyperv1.GCPEndpointAccessPublicAndPrivate
	}
	return true
}

func IsPrivateHC(hc *hyperv1.HostedCluster) bool {
	if hc.Spec.Platform.Type == hyperv1.AWSPlatform {
		access := ptr.Deref(hc.Spec.Platform.AWS, hyperv1.AWSPlatformSpec{}).EndpointAccess
		return access == hyperv1.PublicAndPrivate || access == hyperv1.Private
	}
	if hc.Spec.Platform.Type == hyperv1.GCPPlatform {
		access := ptr.Deref(hc.Spec.Platform.GCP, hyperv1.GCPPlatformSpec{}).EndpointAccess
		return access == hyperv1.GCPEndpointAccessPublicAndPrivate || access == hyperv1.GCPEndpointAccessPrivate
	}
	return false
}

func IsPublicHC(hc *hyperv1.HostedCluster) bool {
	if hc.Spec.Platform.Type == hyperv1.AWSPlatform {
		access := ptr.Deref(hc.Spec.Platform.AWS, hyperv1.AWSPlatformSpec{}).EndpointAccess
		return access == hyperv1.PublicAndPrivate || access == hyperv1.Public
	}
	if hc.Spec.Platform.Type == hyperv1.GCPPlatform {
		access := ptr.Deref(hc.Spec.Platform.GCP, hyperv1.GCPPlatformSpec{}).EndpointAccess
		return access == hyperv1.GCPEndpointAccessPublicAndPrivate
	}
	return true
}

// LabelHCPRoutes determines if routes should be labeled for admission by the HCP router.
// Returns true when routes should use the HCP router infrastructure instead of the
// management cluster router.
//
// This returns true when:
// 1. Cluster has no public access (Private-only), OR
// 2. Cluster is Public with dedicated DNS for KAS (KAS uses Route with explicit hostname)
//
// For PublicAndPrivate clusters using LoadBalancer for KAS, external routes should use
// the management cluster router instead of requiring a dedicated public HCP router.
//
// The key insight: HCP router infrastructure availability is determined by KAS publishing
// strategy, not by individual service DNS configurations.
func LabelHCPRoutes(hcp *hyperv1.HostedControlPlane) bool {
	return !IsPublicHCP(hcp) || UseDedicatedDNSForKAS(hcp)
}
