package awsnodeterminationhandler

import (
	component "github.com/openshift/hypershift/support/controlplane-component"
	"github.com/openshift/hypershift/support/infra"

	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func adaptMachineHealthCheck(cpContext component.WorkloadContext, mhc *capiv1.MachineHealthCheck) error {
	hcp := cpContext.HCP

	// Set the namespace to the HCP namespace (management cluster namespace)
	mhc.Namespace = hcp.Namespace

	// Set the cluster name to the infrastructure ID
	infraStatus := cpContext.InfraStatus
	mhc.Spec.ClusterName = infraStatus.InfraID()

	// The selector is already set in the YAML to match interruptible-instance label
	// No need to modify it here, but we can validate it exists
	if mhc.Spec.Selector.MatchLabels == nil {
		mhc.Spec.Selector.MatchLabels = map[string]string{
			"machine.openshift.io/interruptible-instance": "",
		}
	}

	return nil
}
