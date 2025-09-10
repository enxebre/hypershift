package nodepool

import (
	"context"
	"fmt"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/releaseinfo"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	capigcp "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// Default values for GCP compute instances
	defaultGCPDiskType   = "pd-standard"
	defaultGCPDiskSizeGb = 100
	
	// Default service account scopes for GCP instances
	defaultGCPStorageScope = "https://www.googleapis.com/auth/devstorage.read_only"
	defaultGCPLoggingScope = "https://www.googleapis.com/auth/logging.write"
	defaultGCPMonitoringScope = "https://www.googleapis.com/auth/monitoring"
)

// gcpMachineTemplateSpec generates the GCP machine template specification
func gcpMachineTemplateSpec(infraName string, hostedCluster *hyperv1.HostedCluster, nodePool *hyperv1.NodePool, releaseImage *releaseinfo.ReleaseImage) (*capigcp.GCPMachineTemplateSpec, error) {
	// Validate required GCP platform configuration
	if nodePool.Spec.Platform.GCP == nil {
		return nil, fmt.Errorf("GCP platform configuration is required")
	}

	if nodePool.Spec.Platform.GCP.MachineType == "" {
		return nil, fmt.Errorf("GCP machine type is required")
	}

	// Determine the boot image
	var image string
	if nodePool.Spec.Platform.GCP.Image != "" {
		image = nodePool.Spec.Platform.GCP.Image
	} else {
		// Use default image based on the release
		var err error
		image, err = defaultNodePoolGCPImage(hostedCluster.Spec.Platform.GCP.Region, nodePool.Spec.Arch, releaseImage)
		if err != nil {
			return nil, fmt.Errorf("couldn't discover a GCP image for release image: %w", err)
		}
	}

	// Set disk configuration
	diskType := nodePool.Spec.Platform.GCP.DiskType
	if diskType == "" {
		diskType = defaultGCPDiskType
	}

	diskSizeGb := nodePool.Spec.Platform.GCP.DiskSizeGb
	if diskSizeGb == 0 {
		diskSizeGb = defaultGCPDiskSizeGb
	}

	// Set subnetwork configuration
	subnetwork := nodePool.Spec.Platform.GCP.Subnetwork
	if subnetwork == "" && hostedCluster.Spec.Platform.GCP.Network != nil {
		subnetwork = hostedCluster.Spec.Platform.GCP.Network.Subnetwork
	}

	// Configure service account
	var serviceAccount *capigcp.ServiceAccount
	if nodePool.Spec.Platform.GCP.ServiceAccount != nil {
		email := nodePool.Spec.Platform.GCP.ServiceAccount.Email
		scopes := nodePool.Spec.Platform.GCP.ServiceAccount.Scopes
		
		// Set default scopes if none provided
		if len(scopes) == 0 {
			scopes = []string{
				defaultGCPStorageScope,
				defaultGCPLoggingScope,
				defaultGCPMonitoringScope,
			}
		}

		serviceAccount = &capigcp.ServiceAccount{
			Email:  email,
			Scopes: scopes,
		}
	}

	// Configure network tags
	tags := append([]string{}, nodePool.Spec.Platform.GCP.Tags...)
	// Add cluster-specific tags
	tags = append(tags, fmt.Sprintf("kubernetes-io-cluster-%s", infraName))

	// Configure labels
	labels := make(capigcp.Labels)
	for k, v := range nodePool.Spec.Platform.GCP.Labels {
		labels[k] = v
	}
	// Add cluster-specific labels
	labels["kubernetes-io-cluster-"+infraName] = "owned"

	// Configure maintenance behavior
	onHostMaintenance := nodePool.Spec.Platform.GCP.OnHostMaintenance
	if onHostMaintenance == "" {
		onHostMaintenance = "MIGRATE"
	}

	gcpMachineTemplateSpec := &capigcp.GCPMachineTemplateSpec{
		Template: capigcp.GCPMachineTemplateResource{
			Spec: capigcp.GCPMachineSpec{
				InstanceType:          nodePool.Spec.Platform.GCP.MachineType,
				Image:                 &image,
				RootDeviceType:        (*capigcp.DiskType)(&diskType),
				RootDeviceSize:        diskSizeGb,
				Subnet:                &subnetwork,
				ServiceAccount:        serviceAccount,
				AdditionalNetworkTags: tags,
				AdditionalLabels:      labels,
				Preemptible:           nodePool.Spec.Platform.GCP.Preemptible,
				OnHostMaintenance:     (*capigcp.HostMaintenancePolicy)(&onHostMaintenance),
			},
		},
	}

	return gcpMachineTemplateSpec, nil
}

func (c *CAPI) gcpMachineTemplate(templateNameGenerator func(spec any) (string, error)) (*capigcp.GCPMachineTemplate, error) {
	spec, err := gcpMachineTemplateSpec(c.capiClusterName, c.hostedCluster, c.nodePool, c.releaseImage)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GCPMachineTemplateSpec: %w", err)
	}

	templateName, err := templateNameGenerator(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template name: %w", err)
	}

	template := &capigcp.GCPMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
		Spec: *spec,
	}

	return template, nil
}

// defaultNodePoolGCPImage returns the default GCP image for a given region and architecture
// TODO: Implement GCP image discovery from release metadata when GCP support is added to release info
func defaultNodePoolGCPImage(region string, arch string, releaseImage *releaseinfo.ReleaseImage) (string, error) {
	// For now, return an error since GCP image metadata is not yet available in release info
	// Users will need to specify the image manually in the NodePool spec
	return "", fmt.Errorf("GCP image discovery from release metadata is not yet implemented. Please specify the image manually in the NodePool GCP platform configuration")
}

// setGCPConditions sets GCP-specific conditions for the NodePool
func (r *NodePoolReconciler) setGCPConditions(ctx context.Context, nodePool *hyperv1.NodePool, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string, releaseImage *releaseinfo.ReleaseImage) error {
	if nodePool.Spec.Platform.Type == hyperv1.GCPPlatform {
		if hcluster.Spec.Platform.GCP == nil {
			return fmt.Errorf("the HostedCluster for this NodePool has no .Spec.Platform.GCP, this is unsupported")
		}

		if nodePool.Spec.Platform.GCP.Image != "" {
			// User-defined images cannot be validated
			removeStatusCondition(&nodePool.Status.Conditions, hyperv1.NodePoolValidPlatformImageType)
		} else {
			// Validate default image
			image, err := defaultNodePoolGCPImage(hcluster.Spec.Platform.GCP.Region, nodePool.Spec.Arch, releaseImage)
			if err != nil {
				SetStatusCondition(&nodePool.Status.Conditions, hyperv1.NodePoolCondition{
					Type:               hyperv1.NodePoolValidPlatformImageType,
					Status:             corev1.ConditionFalse,
					Reason:             hyperv1.NodePoolValidationFailedReason,
					Message:            fmt.Sprintf("Couldn't discover a GCP image for release image %q: %s", nodePool.Spec.Release.Image, err.Error()),
					ObservedGeneration: nodePool.Generation,
				})
				return fmt.Errorf("couldn't discover a GCP image for release image: %w", err)
			}
			SetStatusCondition(&nodePool.Status.Conditions, hyperv1.NodePoolCondition{
				Type:               hyperv1.NodePoolValidPlatformImageType,
				Status:             corev1.ConditionTrue,
				Reason:             hyperv1.AsExpectedReason,
				Message:            fmt.Sprintf("Bootstrap GCP image is %q", image),
				ObservedGeneration: nodePool.Generation,
			})
		}
	}
	return nil
}

// validateGCPPlatformConfig validates GCP-specific NodePool configuration
func (r NodePoolReconciler) validateGCPPlatformConfig(ctx context.Context, nodePool *hyperv1.NodePool, hc *hyperv1.HostedCluster, oldCondition *hyperv1.NodePoolCondition) error {
	// Validate machine type is provided
	if nodePool.Spec.Platform.GCP.MachineType == "" {
		return fmt.Errorf("GCP machine type is required")
	}

	// Validate disk configuration
	if nodePool.Spec.Platform.GCP.DiskSizeGb < 20 || nodePool.Spec.Platform.GCP.DiskSizeGb > 65536 {
		return fmt.Errorf("GCP disk size must be between 20 and 65536 GB")
	}

	validDiskTypes := map[string]bool{
		"pd-standard": true,
		"pd-ssd":      true,
		"pd-balanced": true,
	}
	if nodePool.Spec.Platform.GCP.DiskType != "" && !validDiskTypes[nodePool.Spec.Platform.GCP.DiskType] {
		return fmt.Errorf("invalid GCP disk type %q, must be one of: pd-standard, pd-ssd, pd-balanced", nodePool.Spec.Platform.GCP.DiskType)
	}

	// Validate onHostMaintenance setting
	if nodePool.Spec.Platform.GCP.OnHostMaintenance != "" {
		validMaintenanceOptions := map[string]bool{
			"MIGRATE":   true,
			"TERMINATE": true,
		}
		if !validMaintenanceOptions[nodePool.Spec.Platform.GCP.OnHostMaintenance] {
			return fmt.Errorf("invalid GCP onHostMaintenance value %q, must be MIGRATE or TERMINATE", nodePool.Spec.Platform.GCP.OnHostMaintenance)
		}
	}

	return nil
}

// reconcileGCPMachines reconciles GCP machines to ensure labels and tags are up to date
func (c *CAPI) reconcileGCPMachines(ctx context.Context) error {
	gcpMachines := &capigcp.GCPMachineList{}
	if err := c.List(ctx, gcpMachines, client.InNamespace(c.controlplaneNamespace), client.MatchingLabels{
		capiv1.MachineDeploymentNameLabel: c.nodePool.Name,
	}); err != nil {
		return fmt.Errorf("failed to list GCPMachines for NodePool %s: %w", c.nodePool.Name, err)
	}

	var errs []error
	for _, machine := range gcpMachines.Items {
		if _, err := controllerutil.CreateOrPatch(ctx, c.Client, &machine, func() error {
			// Update additional labels if specified
			if c.nodePool.Spec.Platform.GCP.Labels != nil {
				labels := make(capigcp.Labels)
				for k, v := range c.nodePool.Spec.Platform.GCP.Labels {
					labels[k] = v
				}
				// Add cluster-specific labels
				labels["kubernetes-io-cluster-"+c.capiClusterName] = "owned"
				machine.Spec.AdditionalLabels = labels
			}

			// Update network tags if specified
			if c.nodePool.Spec.Platform.GCP.Tags != nil {
				tags := append([]string{}, c.nodePool.Spec.Platform.GCP.Tags...)
				// Add cluster-specific tags
				tags = append(tags, fmt.Sprintf("kubernetes-io-cluster-%s", c.capiClusterName))
				machine.Spec.AdditionalNetworkTags = tags
			}

			return nil
		}); err != nil {
			errs = append(errs, fmt.Errorf("failed to reconcile GCPMachine %s: %w", machine.Name, err))
		}
	}

	return errors.NewAggregate(errs)
}