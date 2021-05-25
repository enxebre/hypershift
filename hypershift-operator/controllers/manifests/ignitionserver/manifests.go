package ignitionserver

import (
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sutilspointer "k8s.io/utils/pointer"
)

const (
	resourceName = "ignition-server"
)

func Route(namespace string) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      resourceName,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: resourceName,
			},
		},
	}
}

func Service(namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      resourceName,
		},
	}
}

func ReconcileServiceClusterIP(mcsService *corev1.Service) error {
	mcsService.Spec.Ports = ServicePorts()
	mcsService.Spec.Selector = ServiceSelector()
	mcsService.Spec.Type = corev1.ServiceTypeClusterIP
	return nil
}

func ServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Name:       "http",
			Protocol:   corev1.ProtocolTCP,
			Port:       80,
			TargetPort: intstr.FromInt(8080),
		},
	}
}

func ServiceSelector() map[string]string {
	return map[string]string{
		"app": resourceName,
	}
}

func Deployment(namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      resourceName,
		},
	}
}

func ReconcileDeployment(deployment *appsv1.Deployment, image string) error {
	deployment.Spec = appsv1.DeploymentSpec{
		Replicas: k8sutilspointer.Int32Ptr(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": resourceName,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": resourceName,
				},
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: k8sutilspointer.Int64Ptr(10),
				Tolerations: []corev1.Toleration{
					{
						Key:    "node-role.kubernetes.io/master",
						Effect: corev1.TaintEffectNoSchedule,
					},
				},
				Containers: []corev1.Container{
					{
						Name:            resourceName,
						Image:           image,
						ImagePullPolicy: corev1.PullAlways,
						Command:         []string{"/ignition-server"},
					},
				},
			},
		},
	}

	return nil
}
