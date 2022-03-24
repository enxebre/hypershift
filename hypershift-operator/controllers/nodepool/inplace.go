package nodepool

import (
	"context"
	"fmt"
	"strconv"

	api "github.com/openshift/hypershift/api"
	hyperv1 "github.com/openshift/hypershift/api/v1alpha1"
	hyperutil "github.com/openshift/hypershift/hypershift-operator/controllers/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8sutilspointer "k8s.io/utils/pointer"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// reconcileInPlaceUpgrade loops over all Nodes that belong to a NodePool and performs and upgrade if necessary.
func (r *NodePoolReconciler) reconcileInPlaceUpgrade(ctx context.Context, hc *hyperv1.HostedCluster, nodePool *hyperv1.NodePool, machineSet *capiv1.MachineSet, targetConfigHash, targetVersion, targetConfigVersionHash string) error {
	log := ctrl.LoggerFrom(ctx)

	// If there's no guest cluster yet return early.
	if hc.Status.KubeConfig == nil {
		return nil
	}

	remoteClient, err := newRemoteClient(ctx, r.Client, hc)
	if err != nil {
		return fmt.Errorf("failed to create remote client: %v", err)
	}

	// Watch guest cluster for Nodes.
	if !r.remoteCaches[client.ObjectKeyFromObject(nodePool)] {
		remoteCache, err := newRemoteCache(ctx, r.Client, hc)
		if err != nil {
			return fmt.Errorf("failed to create remote client: %v", err)
		}

		// TODO: cancel the ctx on exit.
		go remoteCache.Start(ctx)
		if !remoteCache.WaitForCacheSync(ctx) {
			return fmt.Errorf("failed waiting for cache for remote cluster to sync: %w", err)
		}

		if err := r.controller.Watch(source.NewKindWithCache(&corev1.Node{}, remoteCache), handler.EnqueueRequestsFromMapFunc(r.nodeToNodePool)); err != nil {
			return fmt.Errorf("error creating watch: %v", err)
		}

		// TODO: index by HC here instead, figure clean up and use locks.
		if r.remoteCaches == nil {
			r.remoteCaches = make(map[client.ObjectKey]bool)
		}
		r.remoteCaches[client.ObjectKeyFromObject(nodePool)] = true
		log.Info("Created remote cache")
	}

	nodes, err := getNodesForNodePool(ctx, r.Client, remoteClient, machineSet)
	if err != nil {
		return err
	}

	if len(nodes) < 1 {
		log.Info("Processing: no nodes available yet")
	}

	// TODO (alberto): Debug, drop.
	for _, node := range nodes {
		log.Info("Processing", "node", node.Name)
	}

	// If all Nodes are atVersion
	if inPlaceUpgradeComplete(nodes, targetConfigVersionHash) {
		if nodePool.Status.Version != targetVersion {
			log.Info("Version update complete",
				"previous", nodePool.Status.Version, "new", targetVersion)
			nodePool.Status.Version = targetVersion
		}

		if nodePool.Annotations[nodePoolAnnotationCurrentConfig] != targetConfigHash {
			log.Info("Config update complete",
				"previous", nodePool.Annotations[nodePoolAnnotationCurrentConfig], "new", targetConfigHash)
			nodePool.Annotations[nodePoolAnnotationCurrentConfig] = targetConfigHash
		}
		nodePool.Annotations[nodePoolAnnotationCurrentConfigVersion] = targetConfigVersionHash
		return nil
	}

	// Otherwise:
	// Order Nodes deterministically.
	// Check state: AtVersionConfig, Upgrading, wantVersionConfig.
	// If AtVersionConfig then next Node.
	// If Upgrading then no-op, return.
	// If wantVersionConfig then:
	// Check maxUnavailable/MaxSurge.
	// Drain.
	// Create Namespace/RBAC/ConfigMap/Pod in guest cluster.
	// Mark Node as Upgrading.
	return nil
}

func (r *NodePoolReconciler) nodeToNodePool(o client.Object) []reconcile.Request {
	node, ok := o.(*corev1.Node)
	if !ok {
		panic(fmt.Sprintf("Expected a Node but got a %T", o))
	}

	machineName, ok := node.GetAnnotations()[capiv1.MachineAnnotation]
	if !ok {
		return nil
	}

	// Match by namespace when the node has the annotation.
	machineNamespace, ok := node.GetAnnotations()[capiv1.ClusterNamespaceAnnotation]
	if !ok {
		return nil
	}

	// Match by nodeName and status.nodeRef.name.
	machine := &capiv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: machineNamespace,
			Name:      machineName,
		},
	}
	if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(machine), machine); err != nil {
		return nil
	}

	machineOwner := metav1.GetControllerOf(machine)
	if machineOwner.Kind != "MachineSet" {
		return nil
	}

	machineSet := &capiv1.MachineSet{ObjectMeta: metav1.ObjectMeta{
		Name:      machineOwner.Name,
		Namespace: machineNamespace,
	}}
	if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(machineSet), machineSet); err != nil {
		return nil
	}
	nodePoolName := machineSet.GetAnnotations()[nodePoolAnnotation]
	if nodePoolName == "" {
		return nil
	}

	return []reconcile.Request{
		{NamespacedName: hyperutil.ParseNamespacedName(nodePoolName)},
	}
}

func getNodesForNodePool(ctx context.Context, c client.Reader, remoteClient client.Client, machineSet *capiv1.MachineSet) ([]*corev1.Node, error) {
	selectorMap, err := metav1.LabelSelectorAsMap(&machineSet.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to convert MachineSet %q label selector to a map: %v", machineSet.Name, err)
	}

	// Get all Machines linked to this MachineSet.
	allMachines := &capiv1.MachineList{}
	if err = c.List(ctx,
		allMachines,
		client.InNamespace(machineSet.Namespace),
		client.MatchingLabels(selectorMap),
	); err != nil {
		return nil, fmt.Errorf("failed to list machines: %v", err)
	}

	var machineSetOwnedMachines []capiv1.Machine
	for i, machine := range allMachines.Items {
		if metav1.GetControllerOf(&machine) != nil && metav1.IsControlledBy(&machine, machineSet) {
			machineSetOwnedMachines = append(machineSetOwnedMachines, allMachines.Items[i])
		}
	}

	var nodes []*corev1.Node
	for _, machine := range machineSetOwnedMachines {
		if machine.Status.NodeRef != nil {
			node := &corev1.Node{}
			if err := remoteClient.Get(ctx, client.ObjectKey{Name: machine.Status.NodeRef.Name}, node); err != nil {
				return nil, fmt.Errorf("error getting node: %v", err)
			}
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// TODO (alberto): implement.
func inPlaceUpgradeComplete(nodes []*corev1.Node, targetVersionConfig string) bool {
	return false
}

func targetRESTConfig(ctx context.Context, c client.Reader, hc *hyperv1.HostedCluster) (*restclient.Config, error) {
	log := ctrl.LoggerFrom(ctx)

	if hc.Status.KubeConfig == nil {
		log.Info("No kubeConfig available yet")
		return nil, nil
	}

	// Resolve the kubeconfig secret for CAPI.
	// TODO (alberto): Provider adhoc kubeconfig.
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hc.Namespace,
			Name:      fmt.Sprintf("%s-admin-kubeconfig", hc.Name),
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(kubeconfigSecret), kubeconfigSecret); err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig secret %q: %w", kubeconfigSecret.Name, err)
	}

	// TODO (alberto): not hardcode key here.
	kubeConfig, ok := kubeconfigSecret.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("failed to get kubeconfig from secret %q", kubeconfigSecret.Name)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config kubeconfig from secret %q", kubeconfigSecret.Name)
	}

	restConfig.UserAgent = "nodepool-controller"
	// restConfig.Timeout = defaultClientTimeout

	return restConfig, nil
}

// NewRemoteClient returns a Client for interacting with a remote Cluster using the given scheme for encoding and decoding objects.
func newRemoteClient(ctx context.Context, c client.Client, hc *hyperv1.HostedCluster) (client.Client, error) {
	restConfig, err := targetRESTConfig(ctx, c, hc)
	if err != nil {
		return nil, err
	}

	remoteClient, err := client.New(restConfig, client.Options{Scheme: c.Scheme()})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return remoteClient, nil
}

// NewRemoteClient returns a Client for interacting with a remote Cluster using the given scheme for encoding and decoding objects.
func newRemoteCache(ctx context.Context, c client.Client, hc *hyperv1.HostedCluster) (cache.Cache, error) {
	restConfig, err := targetRESTConfig(ctx, c, hc)
	if err != nil {
		return nil, err
	}

	remoteCache, err := cache.New(restConfig, cache.Options{Scheme: c.Scheme()})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return remoteCache, nil
}

func (r *NodePoolReconciler) reconcileMachineSet(ctx context.Context,
	machineSet *capiv1.MachineSet,
	hc *hyperv1.HostedCluster,
	nodePool *hyperv1.NodePool,
	userDataSecret *corev1.Secret,
	machineTemplateCR client.Object,
	CAPIClusterName string,
	targetVersion,
	targetConfigHash, targetConfigVersionHash, machineTemplateSpecJSON string) error {

	log := ctrl.LoggerFrom(ctx)
	// Set annotations and labels
	if machineSet.GetAnnotations() == nil {
		machineSet.Annotations = map[string]string{}
	}
	machineSet.Annotations[nodePoolAnnotation] = client.ObjectKeyFromObject(nodePool).String()
	if machineSet.GetLabels() == nil {
		machineSet.Labels = map[string]string{}
	}
	machineSet.Labels[capiv1.ClusterLabelName] = CAPIClusterName

	resourcesName := generateName(CAPIClusterName, nodePool.Spec.ClusterName, nodePool.GetName())
	machineSet.Spec.MinReadySeconds = int32(0)

	gvk, err := apiutil.GVKForObject(machineTemplateCR, api.Scheme)
	if err != nil {
		return err
	}

	// Set selector and template
	machineSet.Spec.ClusterName = CAPIClusterName
	if machineSet.Spec.Selector.MatchLabels == nil {
		machineSet.Spec.Selector.MatchLabels = map[string]string{}
	}
	machineSet.Spec.Selector.MatchLabels[resourcesName] = resourcesName
	machineSet.Spec.Template = capiv1.MachineTemplateSpec{
		ObjectMeta: capiv1.ObjectMeta{
			Labels: map[string]string{
				resourcesName:           resourcesName,
				capiv1.ClusterLabelName: CAPIClusterName,
			},
			Annotations: map[string]string{
				// TODO (alberto): Use conditions to signal an in progress rolling upgrade
				// similar to what we do with nodePoolAnnotationCurrentConfig
				nodePoolAnnotationPlatformMachineTemplate: machineTemplateSpecJSON, // This will trigger a deployment rolling upgrade when its value changes.
			},
		},

		Spec: capiv1.MachineSpec{
			ClusterName: CAPIClusterName,
			Bootstrap: capiv1.Bootstrap{
				// Keep current user data for later check.
				DataSecretName: machineSet.Spec.Template.Spec.Bootstrap.DataSecretName,
			},
			InfrastructureRef: corev1.ObjectReference{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
				Namespace:  machineTemplateCR.GetNamespace(),
				Name:       machineTemplateCR.GetName(),
			},
			// Keep current version for later check.
			Version:          machineSet.Spec.Template.Spec.Version,
			NodeDrainTimeout: nodePool.Spec.NodeDrainTimeout,
		},
	}

	// Propagate version and userData Secret to the machineDeployment.
	if userDataSecret.Name != k8sutilspointer.StringPtrDerefOr(machineSet.Spec.Template.Spec.Bootstrap.DataSecretName, "") {
		log.Info("New user data Secret has been generated",
			"current", machineSet.Spec.Template.Spec.Bootstrap.DataSecretName,
			"target", userDataSecret.Name)

		// TODO (alberto): possibly compare with NodePool here instead so we don't rely on impl details to drive decisions.
		if targetVersion != k8sutilspointer.StringPtrDerefOr(machineSet.Spec.Template.Spec.Version, "") {
			log.Info("Starting version update: Propagating new version to the MachineSet",
				"releaseImage", nodePool.Spec.Release.Image, "target", targetVersion)
		}

		if targetConfigHash != nodePool.Annotations[nodePoolAnnotationCurrentConfig] {
			log.Info("Starting config update: Propagating new config to the MachineDeployment",
				"current", nodePool.Annotations[nodePoolAnnotationCurrentConfig], "target", targetConfigHash)
		}
		machineSet.Spec.Template.Spec.Version = &targetVersion
		machineSet.Spec.Template.Spec.Bootstrap.DataSecretName = k8sutilspointer.StringPtr(userDataSecret.Name)

		// We return early here during a version/config update to persist the resource with new user data Secret,
		// so in the next reconciling loop we get a new machineSet.Generation
		// and we can do a legit MachineDeploymentComplete/MachineDeployment.Status.ObservedGeneration check.
		// Before persisting, if the NodePool is brand new we want to make sure the replica number is set so the machineDeployment controller
		// does not panic.
		if machineSet.Spec.Replicas == nil {
			machineSet.Spec.Replicas = k8sutilspointer.Int32Ptr(k8sutilspointer.Int32PtrDerefOr(nodePool.Spec.NodeCount, 0))
		}
		return nil
	}

	setMachineSetReplicas(nodePool, machineSet)

	// Bubble up AvailableReplicas and Ready condition from MachineDeployment.
	nodePool.Status.NodeCount = machineSet.Status.AvailableReplicas
	for _, c := range machineSet.Status.Conditions {
		// This condition should aggregate and summarise readiness from underlying MachineSets and Machines
		// https://github.com/kubernetes-sigs/cluster-api/issues/3486.
		if c.Type == capiv1.ReadyCondition {
			// this is so api server does not complain
			// invalid value: \"\": status.conditions.reason in body should be at least 1 chars long"
			reason := hyperv1.NodePoolAsExpectedConditionReason
			if c.Reason != "" {
				reason = c.Reason
			}

			setStatusCondition(&nodePool.Status.Conditions, hyperv1.NodePoolCondition{
				Type:               hyperv1.NodePoolReadyConditionType,
				Status:             c.Status,
				ObservedGeneration: nodePool.Generation,
				Message:            c.Message,
				Reason:             reason,
			})
			break
		}
	}

	return nil
}

// setMachineSetReplicas sets wanted replicas:
// If autoscaling is enabled we reconcile min/max annotations and leave replicas untouched.
func setMachineSetReplicas(nodePool *hyperv1.NodePool, machineSet *capiv1.MachineSet) {
	if machineSet.Annotations == nil {
		machineSet.Annotations = make(map[string]string)
	}

	if isAutoscalingEnabled(nodePool) {
		if k8sutilspointer.Int32PtrDerefOr(machineSet.Spec.Replicas, 0) == 0 {
			// if autoscaling is enabled and the machineDeployment does not exist yet or it has 0 replicas
			// we set it to 1 replica as the autoscaler does not support scaling from zero yet.
			machineSet.Spec.Replicas = k8sutilspointer.Int32Ptr(int32(1))
		}
		machineSet.Annotations[autoscalerMaxAnnotation] = strconv.Itoa(int(nodePool.Spec.AutoScaling.Max))
		machineSet.Annotations[autoscalerMinAnnotation] = strconv.Itoa(int(nodePool.Spec.AutoScaling.Min))
	}

	// If autoscaling is NOT enabled we reset min/max annotations and reconcile replicas.
	if !isAutoscalingEnabled(nodePool) {
		machineSet.Annotations[autoscalerMaxAnnotation] = "0"
		machineSet.Annotations[autoscalerMinAnnotation] = "0"
		machineSet.Spec.Replicas = k8sutilspointer.Int32Ptr(k8sutilspointer.Int32PtrDerefOr(nodePool.Spec.NodeCount, 0))
	}
}
