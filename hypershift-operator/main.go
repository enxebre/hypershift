/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"

	capiaws "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	routev1 "github.com/openshift/api/route/v1"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	hyperv1 "openshift.io/hypershift/api/v1alpha1"
	"openshift.io/hypershift/hypershift-operator/controllers"
	"openshift.io/hypershift/hypershift-operator/releaseinfo"

	capiv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	capiaws.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	hyperv1.AddToScheme(scheme)
	capiv1.AddToScheme(scheme)
	configv1.AddToScheme(scheme)
	securityv1.AddToScheme(scheme)
	operatorv1.AddToScheme(scheme)
	routev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	cmd := &cobra.Command{
		Use: "hypershift-operator",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}
	cmd.AddCommand(NewStartCommand())

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs the Hypershift operator",
	}

	var metricsAddr string
	var enableLeaderElection bool
	var controlPlaneOperatorImage string

	cmd.Flags().StringVar(&metricsAddr, "metrics-addr", "0", "The address the metric endpoint binds to.")
	cmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	cmd.Flags().StringVar(&controlPlaneOperatorImage, "control-plane-operator-image", "", "A control plane operator image.")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:             scheme,
			MetricsBindAddress: metricsAddr,
			Port:               9443,
			LeaderElection:     enableLeaderElection,
			LeaderElectionID:   "b2ed43ca.hypershift.openshift.io",
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		// Add some flexibility to getting the control plane operator image. Use the
		// flag if given, but if that's empty and we're running in a deployment, use the
		// hypershift operator's image for the control plane by default.
		lookupControlPlaneOperatorImage := func(kubeClient client.Client) (string, error) {
			if len(controlPlaneOperatorImage) > 0 {
				return controlPlaneOperatorImage, nil
			}
			deployment := appsv1.Deployment{}
			err := kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: "hypershift", Name: "operator"}, &deployment)
			if err != nil {
				if errors.IsNotFound(err) {
					return "", nil
				}
				return "", fmt.Errorf("failed to get operator deployment: %w", err)
			}
			var image string
			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == "operator" {
					image = container.Image
					break
				}
			}
			return image, nil
		}

		kubeClient, err := kubernetes.NewForConfig(mgr.GetConfig())
		if err != nil {
			setupLog.Error(err, "unable to create kube client")
			os.Exit(1)
		}

		if err = (&controllers.OpenShiftClusterReconciler{
			Client: mgr.GetClient(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "OpenShiftCluster")
			os.Exit(1)
		}

		if err := (&controllers.HostedControlPlaneReconciler{
			Client:                          mgr.GetClient(),
			LookupControlPlaneOperatorImage: lookupControlPlaneOperatorImage,
			ReleaseProvider: &releaseinfo.PodProvider{
				Pods: kubeClient.CoreV1().Pods("hypershift"),
			},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "hostedControlPlane")
			os.Exit(1)
		}

		if err := (&controllers.ExternalInfraClusterReconciler{
			Client: mgr.GetClient(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ExternalInfraCluster")
			os.Exit(1)
		}

		if err := (&controllers.NodePoolReconciler{
			Client: mgr.GetClient(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "nodePool")
			os.Exit(1)
		}

		// +kubebuilder:scaffold:builder

		setupLog.Info("starting manager")

		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}

	return cmd
}
