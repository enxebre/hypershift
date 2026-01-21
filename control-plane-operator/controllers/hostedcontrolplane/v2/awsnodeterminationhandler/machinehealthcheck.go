package awsnodeterminationhandler

import (
	component "github.com/openshift/hypershift/support/controlplane-component"

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func adaptMachineHealthCheck(cpContext component.WorkloadContext, mhc *capiv1.MachineHealthCheck) error {
	hcp := cpContext.HCP

	// Set the namespace to the HCP namespace (management cluster namespace)
	mhc.Namespace = hcp.Namespace

	// Set the cluster name to the infrastructure ID
	mhc.Spec.ClusterName = hcp.Spec.InfraID

	// The selector is already set in the YAML to match interruptible-instance label
	// No need to modify it here, but we can validate it exists
	if mhc.Spec.Selector.MatchLabels == nil {
		mhc.Spec.Selector.MatchLabels = map[string]string{
			"machine.openshift.io/interruptible-instance": "",
		}
	}

	return nil
}
