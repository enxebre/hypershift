package configoperator

import (
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

func IsExternalInfraKv(hcp *hyperv1.HostedControlPlane) bool {
	if hcp.Spec.Platform.Kubevirt != nil &&
		hcp.Spec.Platform.Kubevirt.Credentials != nil &&
		hcp.Spec.Platform.Kubevirt.Credentials.InfraKubeConfigSecret != nil &&
		hcp.Spec.Platform.Kubevirt.Credentials.InfraNamespace != "" {
		return true
	} else {
		return false
	}
}
