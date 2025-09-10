package gcp

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	component "github.com/openshift/hypershift/support/controlplane-component"
)

func adaptConfig(cpContext component.WorkloadContext, cm *corev1.ConfigMap) error {
	hcp := cpContext.HCP
	
	if hcp.Spec.Platform.GCP == nil {
		return fmt.Errorf("GCP platform configuration is missing")
	}

	gcpConfig := hcp.Spec.Platform.GCP

	// Create the GCP cloud controller manager configuration
	config := fmt.Sprintf(`[Global]
project-id = %s
regional = true
region = %s
multizone = true
node-tags = %s
node-instance-prefix = %s
`, 
		gcpConfig.Project,
		gcpConfig.Region,
		fmt.Sprintf("%s-%s", hcp.Namespace, hcp.Name), // node tags for cluster identification
		fmt.Sprintf("%s-%s", hcp.Namespace, hcp.Name), // instance prefix
	)

	// Add network configuration if specified
	if gcpConfig.Network != nil {
		if gcpConfig.Network.Network != "" {
			config += fmt.Sprintf("network-name = %s\n", gcpConfig.Network.Network)
		}
		if gcpConfig.Network.Subnetwork != "" {
			config += fmt.Sprintf("subnetwork-name = %s\n", gcpConfig.Network.Subnetwork)
		}
	}

	// Add load balancer configuration
	config += `
[LoadBalancer]
load-balancer-type = External
`

	cm.Data = map[string]string{
		"gcp.conf": config,
	}

	return nil
}

func adaptCredentials(cpContext component.WorkloadContext, secret *corev1.Secret) error {
	// The credentials secret should be populated by the infrastructure reconciliation
	// This adapter ensures the secret has the correct structure
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	// Set the secret type
	secret.Type = corev1.SecretTypeOpaque

	// The service account JSON should be provided by the platform reconciliation
	// We just ensure the expected key exists
	if _, exists := secret.Data["service-account.json"]; !exists {
		// Add a placeholder - this will be replaced by actual credentials during reconciliation
		secret.Data["service-account.json"] = []byte("{}")
	}

	return nil
}