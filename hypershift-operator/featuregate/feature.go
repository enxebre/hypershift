package featuregate

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

const (
	// OpenStack is a feature gate for running clusters on OpenStack.
	// owner: @username
	// alpha: v0.1.49
	// beta: x.y.z
	OpenStack featuregate.Feature = "OpenStack"

	// CPOV2      featuregate.Feature = "CPOV2"
	// AWSTenancy featuregate.Feature = "AWSTenancy"
)

func init() {
	runtime.Must(MutableGates.Add(defaultHypershiftFeatureGates))
}

var defaultHypershiftFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	// Every feature should be initiated here:
	OpenStack: {Default: false, PreRelease: featuregate.Alpha},
	// CPOV2:         {Default: false, PreRelease: featuregate.Alpha},
	// AWSTenancy:    {Default: false, PreRelease: featuregate.Alpha},
}
