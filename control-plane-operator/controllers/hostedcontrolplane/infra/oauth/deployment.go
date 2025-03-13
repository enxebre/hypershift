package oauth

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	configHashAnnotation                 = "oauth.hypershift.openshift.io/config-hash"
	KubeadminSecretHashAnnotation        = "hypershift.openshift.io/kubeadmin-secret-hash"
	oauthNamedCertificateMountPathPrefix = "/etc/kubernetes/certs/named"
	socks5ProxyContainerName             = "socks-proxy"
)

func oauthContainerMain() *corev1.Container {
	return &corev1.Container{
		Name: "oauth-server",
	}
}
