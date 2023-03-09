package aws

import (
	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/kas"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/manifests"
	"github.com/openshift/hypershift/support/config"
	"github.com/openshift/hypershift/support/releaseinfo"
	"github.com/openshift/hypershift/support/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ReconcileCCMServiceAccount(sa *corev1.ServiceAccount, ownerRef config.OwnerRef) error {
	ownerRef.ApplyTo(sa)
	return nil
}

func ReconcileCCMRole(role *rbacv1.Role, ownerRef config.OwnerRef) error {
	ownerRef.ApplyTo(role)
	role.Rules = []rbacv1.PolicyRule{
		{Verbs: , APIGroups: , Resources: , ResourceNames: , NonResourceURLs: },
	}
	return nil
}

func ReconcileCCMRoleBinding(roleBinding *rbacv1.RoleBinding, ownerRef config.OwnerRef, sa *corev1.ServiceAccount, role *rbacv1.Role) error {
	ownerRef.ApplyTo(roleBinding)
	roleBinding.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     role.Name,
	}
	roleBinding.Subjects = []rbacv1.Subject{
		{
			Namespace: sa.Namespace,
			Kind:      rbacv1.ServiceAccountKind,
			Name:      sa.Name,
		},
	}
	return nil
}

func ReconcileDeployment(deployment *appsv1.Deployment, hcp *hyperv1.HostedControlPlane, serviceAccountName string, releaseImage *releaseinfo.ReleaseImage) error {
	deploymentConfig := newDeploymentConfig()
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ccmLabels(),
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: ccmLabels(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					util.BuildContainer(CCMContainer(), buildCCMContainer(releaseImage)),
				},
				Volumes:            []corev1.Volume{},
				ServiceAccountName: serviceAccountName,
			},
		},
	}

	addVolumes(deployment)

	config.OwnerRefFrom(hcp).ApplyTo(deployment)
	deploymentConfig.ApplyTo(deployment)
	util.ApplyCloudProviderCreds(&deployment.Spec.Template.Spec, p.CloudProvider, p.CloudProviderCreds, p.TokenMinterImage, kcmContainerMain().Name)
	util.AvailabilityProber(kas.InClusterKASReadyURL(deployment.Namespace, apiPort), p.AvailabilityProberImage, &deployment.Spec.Template.Spec)
	return nil
}

func addVolumes(deployment *appsv1.Deployment) {

	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		util.BuildVolume(ccmVolumeKubeconfig(), buildCCMVolumeKubeconfig),
	)
	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		util.BuildVolume(ccmCloudConfig(), buildCCMCloudConfig),
	)
}

func podVolumeMounts() util.PodVolumeMounts {
	return util.PodVolumeMounts{
		CCMContainer().Name: util.ContainerVolumeMounts{
			ccmVolumeKubeconfig().Name: "/etc/kubernetes/kubeconfig",
			ccmCloudConfig().Name:      "/etc/cloud",
		},
	}
}

func buildCCMContainer(releaseImage *releaseinfo.ReleaseImage) func(c *corev1.Container) {
	return func(c *corev1.Container) {
		c.Image = releaseImage.ComponentImages()["aws-cloud-controller-manager"]
		c.ImagePullPolicy = corev1.PullIfNotPresent
		c.Command = []string{"/bin/aws-cloud-controller-manager"}
		c.Args = []string{
			"--cloud-provider=aws",
			"--cloud-config=/etc/cloud/cloud-config",
			"--use-service-account-credentials=true",
			"--configure-cloud-routes=false",
			"--leader-elect=true",
			"--leader-elect-lease-duration=137s",
			"--leader-elect-renew-deadline=107s",
			"--leader-elect-retry-period=26s",
			"--leader-elect-resource-namespace=openshift-cloud-controller-manager",
		}
		c.VolumeMounts = podVolumeMounts().ContainerMounts(c.Name)
	}
}

func buildCCMVolumeKubeconfig(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{
		SecretName: manifests.KASServiceKubeconfigSecret("").Name,
	}
}

func buildCCMCloudConfig(v *corev1.Volume) {
	v.ConfigMap = &corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: CCMConfigMap("").Name,
		},
	}
}

func newDeploymentConfig() config.DeploymentConfig {
	result := config.DeploymentConfig{}
	result.Resources = config.ResourcesSpec{
		CCMContainer().Name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("60Mi"),
				corev1.ResourceCPU:    resource.MustParse("75m"),
			},
		},
	}
	result.AdditionalLabels = additionalLabels()
	result.Scheduling.PriorityClass = config.DefaultPriorityClass

	result.Replicas = 1

	return result
}

func ccmLabels() map[string]string {
	return map[string]string{
		"app": "cloud-controller-manager",
	}
}

func additionalLabels() map[string]string {
	return map[string]string{
		hyperv1.ControlPlaneComponent: "cloud-controller-manager",
	}
}
