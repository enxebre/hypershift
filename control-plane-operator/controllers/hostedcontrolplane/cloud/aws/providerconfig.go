package aws

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/openshift/hypershift/support/util"
)

const (
	Provider          = util.AWSCloudProviderName
	ProviderConfigKey = "aws.conf"
)

const configTemplate = `[Global]
Zone = %s
VPC = %s
KubernetesClusterID = %s
SubnetID = %s`

func (p *AWSParams) ReconcileCloudConfig(cm *corev1.ConfigMap) error {
	util.EnsureOwnerRef(cm, p.OwnerRef)
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[ProviderConfigKey] = fmt.Sprintf(configTemplate, p.Zone, p.VPC, p.ClusterID, p.SubnetID)
	return nil
}

func KubeCloudControllerCredsSecret(controlPlaneNamespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: controlPlaneNamespace,
			Name:      "cloud-controller-creds",
		},
	}
}
