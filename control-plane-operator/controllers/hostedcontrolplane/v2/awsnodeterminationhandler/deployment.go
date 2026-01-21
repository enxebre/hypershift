package awsnodeterminationhandler

import (
	"fmt"
	"net/url"

	component "github.com/openshift/hypershift/support/controlplane-component"
	"github.com/openshift/hypershift/support/util"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func adaptDeployment(cpContext component.WorkloadContext, deployment *appsv1.Deployment) error {
	hcp := cpContext.HCP

	// Get AWS region from HCP spec
	awsRegion := ""
	if hcp.Spec.Platform.AWS != nil {
		awsRegion = hcp.Spec.Platform.AWS.Region
	}

	// Get SQS queue URL from annotation
	queueURL := ""
	if hcp.Annotations != nil {
		queueURL = hcp.Annotations[AnnotationTerminationHandlerQueueURL]
	}

	// Get Kubernetes API endpoint from HCP status
	kubeServiceHost := ""
	kubeServicePort := ""
	if hcp.Status.ControlPlaneEndpoint.Host != "" {
		// Parse the endpoint URL to extract host and port
		endpointURL, err := url.Parse(hcp.Status.ControlPlaneEndpoint.Host)
		if err == nil {
			kubeServiceHost = endpointURL.Hostname()
			kubeServicePort = endpointURL.Port()
			if kubeServicePort == "" {
				kubeServicePort = fmt.Sprintf("%d", hcp.Status.ControlPlaneEndpoint.Port)
			}
		} else {
			// If parsing fails, use the host directly
			kubeServiceHost = hcp.Status.ControlPlaneEndpoint.Host
			kubeServicePort = fmt.Sprintf("%d", hcp.Status.ControlPlaneEndpoint.Port)
		}
	}

	// Update the container environment variables
	util.UpdateContainer(ComponentName, deployment.Spec.Template.Spec.Containers, func(c *corev1.Container) {
		// Update environment variables
		for i := range c.Env {
			switch c.Env[i].Name {
			case "AWS_REGION":
				c.Env[i].Value = awsRegion
			case "QUEUE_URL":
				c.Env[i].Value = queueURL
			case "KUBERNETES_SERVICE_HOST":
				c.Env[i].Value = kubeServiceHost
			case "KUBERNETES_SERVICE_PORT":
				c.Env[i].Value = kubeServicePort
			}
		}
	})

	return nil
}
