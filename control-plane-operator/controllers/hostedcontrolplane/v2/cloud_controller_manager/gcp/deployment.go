package gcp

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	component "github.com/openshift/hypershift/support/controlplane-component"
	"github.com/openshift/hypershift/support/util"
)

func adaptDeployment(cpContext component.WorkloadContext, deployment *appsv1.Deployment) error {
	hcp := cpContext.HCP
	
	// Update image
	util.UpdateContainer(ComponentName, deployment.Spec.Template.Spec.Containers, func(c *corev1.Container) {
		c.Image = cpContext.ReleaseImageProvider.GetImage("gcp-cloud-controller-manager")
	})

	// Update command for GCP
	util.UpdateContainer(ComponentName, deployment.Spec.Template.Spec.Containers, func(c *corev1.Container) {
		c.Command = []string{"/bin/gcp-cloud-controller-manager"}
		c.Args = []string{
			"--cloud-provider=gcp",
			"--use-service-account-credentials=false",
			"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
			"--cloud-config=/etc/cloud/gcp.conf",
			"--configure-cloud-routes=false",
			"--leader-elect=true",
			"--leader-elect-lease-duration=137s",
			"--leader-elect-renew-deadline=107s",
			"--leader-elect-retry-period=26s",
			"--leader-elect-resource-namespace=openshift-cloud-controller-manager",
			"--authentication-kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
			"--authorization-kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
			"--bind-address=0.0.0.0",
			"--secure-port=10258",
			"--tls-cert-file=/etc/kubernetes/certs/tls.crt",
			"--tls-private-key-file=/etc/kubernetes/certs/tls.key",
		}
	})

	// Update environment variables for GCP
	util.UpdateContainer(ComponentName, deployment.Spec.Template.Spec.Containers, func(c *corev1.Container) {
		c.Env = []corev1.EnvVar{
			{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: "/etc/kubernetes/secrets/gcp/service-account.json",
			},
		}
	})

	// Update volume mounts for GCP
	util.UpdateContainer(ComponentName, deployment.Spec.Template.Spec.Containers, func(c *corev1.Container) {
		c.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "cloud-config",
				MountPath: "/etc/cloud",
			},
			{
				Name:      "kubeconfig",
				MountPath: "/etc/kubernetes/kubeconfig",
			},
			{
				Name:      "gcp-credentials",
				MountPath: "/etc/kubernetes/secrets/gcp",
			},
			{
				Name:      "serving-cert",
				MountPath: "/etc/kubernetes/certs",
			},
		}
	})

	// Update volumes for GCP
	deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: ptr.To[int32](420),
					SecretName:  "service-network-admin-kubeconfig",
				},
			},
		},
		{
			Name: "cloud-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: ptr.To[int32](420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "gcp-cloud-config",
					},
				},
			},
		},
		{
			Name: "gcp-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: ptr.To[int32](420),
					SecretName:  "gcp-cloud-credentials",
				},
			},
		},
		{
			Name: "serving-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: ptr.To[int32](420),
					SecretName:  fmt.Sprintf("%s-serving-cert", ComponentName),
				},
			},
		},
	}

	// Set resource requirements
	util.UpdateContainer(ComponentName, deployment.Spec.Template.Spec.Containers, func(c *corev1.Container) {
		c.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("75m"),
				corev1.ResourceMemory: resource.MustParse("60Mi"),
			},
		}
	})

	// Set service account
	deployment.Spec.Template.Spec.ServiceAccountName = "cloud-controller-manager"

	// Set node selector for management nodes
	deployment.Spec.Template.Spec.NodeSelector = map[string]string{
		"hypershift.openshift.io/control-plane": "true",
	}

	// Set tolerations
	deployment.Spec.Template.Spec.Tolerations = []corev1.Toleration{
		{
			Key:    "hypershift.openshift.io/control-plane",
			Effect: corev1.TaintEffectNoSchedule,
		},
		{
			Key:    "hypershift.openshift.io/cluster",
			Effect: corev1.TaintEffectNoSchedule,
			Value:  hcp.Namespace + "-" + hcp.Name,
		},
	}

	// Set priority class
	deployment.Spec.Template.Spec.PriorityClassName = "hypershift-control-plane"

	return nil
}