package router

import (
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	oapiv2 "github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/v2/oapi"
	"github.com/openshift/hypershift/hypershift-operator/controllers/sharedingress"
	component "github.com/openshift/hypershift/support/controlplane-component"
	"github.com/openshift/hypershift/support/util"
)

const (
	ComponentName = "router"
)

var _ component.ComponentOptions = &router{}

type router struct {
}

// IsRequestServing implements controlplanecomponent.ComponentOptions.
func (k *router) IsRequestServing() bool {
	return true
}

// MultiZoneSpread implements controlplanecomponent.ComponentOptions.
func (k *router) MultiZoneSpread() bool {
	return true
}

// NeedsManagementKASAccess implements controlplanecomponent.ComponentOptions.
func (k *router) NeedsManagementKASAccess() bool {
	return false
}

func NewComponent() component.ControlPlaneComponent {
	return component.NewDeploymentComponent(ComponentName, &router{}).
		WithPredicate(useHCPRouter).
		WithManifestAdapter(
			"config.yaml",
			component.WithAdaptFunction(adaptConfig),
		).
		WithManifestAdapter(
			"pdb.yaml",
			component.AdaptPodDisruptionBudget(),
		).
		WithDependencies(oapiv2.ComponentName).
		Build()
}

// useHCPRouter returns true when the HCP routes should be served by a dedicated
// HCP router, as determined by util.LabelHCPRoutes (private-only or public with
// dedicated KAS DNS), excluding shared ingress and IBM Cloud.
func useHCPRouter(cpContext component.WorkloadContext) (bool, error) {
	if sharedingress.UseSharedIngress() {
		return false, nil
	}
	if cpContext.HCP.Spec.Platform.Type == hyperv1.IBMCloudPlatform {
		return false, nil
	}
	return util.LabelHCPRoutes(cpContext.HCP), nil
}
