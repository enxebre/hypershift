package featuregate

import (
	"context"
	"time"
	featuregate "github.com/openshift/hypershift/hypershift-operator/featuregate"
	hypershiftclient "github.com/openshift/hypershift/client/clientset/clientset"
	"github.com/openshift/hypershift/control-plane-operator/hostedclusterconfigoperator/operator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const ControllerName = "featuregate"

func Setup(ctx context.Context, opts *operator.HostedClusterConfigOperatorConfig) error {
	!featuregate.Gates.Enabled(featuregate.MinimumKubeletVersion) {
		return nil
	}

	hypershiftClient, err := hypershiftclient.NewForConfig(opts.CPCluster.GetConfig())
	if err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(opts.Manager).
		Named(ControllerName).
		For(&corev1.Node{}).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 10*time.Second),
		}).Complete(&minimumKubeletVersionReconciler{
		hcpName:            opts.HCPName,
		hcpNamespace:       opts.Namespace,
		client:             hypershiftClient,
		lister:             opts.CPCluster.GetClient(),
		guestClusterClient: opts.Manager.GetClient(),
	}); err != nil {
		return err
	}
}
