package gcp

import (
	"fmt"
	"strings"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/pflag"
)

type GCPNodePoolPlatformOptions struct {
	MachineType       string
	DiskType          string
	DiskSizeGb        int64
	Subnetwork        string
	Image             string
	ServiceAccountEmail string
	ServiceAccountScopes []string
	Tags              []string
	Labels            []string
	Preemptible       bool
	OnHostMaintenance string
}

func DefaultOptions() *GCPNodePoolPlatformOptions {
	return &GCPNodePoolPlatformOptions{
		MachineType:       "n1-standard-4",
		DiskType:          "pd-standard",
		DiskSizeGb:        100,
		OnHostMaintenance: "MIGRATE",
		ServiceAccountScopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring",
		},
	}
}

func BindOptions(opts *GCPNodePoolPlatformOptions, flags *pflag.FlagSet) {
	flags.StringVar(&opts.MachineType, "gcp-instance-type", opts.MachineType, "GCP machine type for the NodePool")
	flags.StringVar(&opts.DiskType, "gcp-disk-type", opts.DiskType, "Type of disk for the root volume (pd-ssd, pd-standard, pd-balanced)")
	flags.Int64Var(&opts.DiskSizeGb, "gcp-disk-size", opts.DiskSizeGb, "Size of the disk in GB")
	flags.StringVar(&opts.Subnetwork, "gcp-subnetwork", opts.Subnetwork, "Name of the subnet to use for the NodePool instances")
	flags.StringVar(&opts.Image, "gcp-image", opts.Image, "Boot image to use for the instances (if unspecified, default is chosen based on the NodePool release payload image)")
	flags.StringVar(&opts.ServiceAccountEmail, "gcp-service-account-email", opts.ServiceAccountEmail, "Email address of the service account to use for instances")
	flags.StringSliceVar(&opts.ServiceAccountScopes, "gcp-service-account-scopes", opts.ServiceAccountScopes, "Access scopes for the service account")
	flags.StringSliceVar(&opts.Tags, "gcp-tags", opts.Tags, "Additional network tags to apply to the instances")
	flags.StringSliceVar(&opts.Labels, "gcp-labels", opts.Labels, "Additional labels to apply to the instances (e.g. 'key1=value1,key2=value2')")
	flags.BoolVar(&opts.Preemptible, "gcp-preemptible", opts.Preemptible, "Use preemptible instances for cost optimization")
	flags.StringVar(&opts.OnHostMaintenance, "gcp-on-host-maintenance", opts.OnHostMaintenance, "What to do when Google Compute Engine schedules a maintenance event (MIGRATE or TERMINATE)")
}

func (o *GCPNodePoolPlatformOptions) UpdateNodePool(nodePool *hyperv1.NodePool) error {
	if nodePool.Spec.Platform.GCP == nil {
		nodePool.Spec.Platform.GCP = &hyperv1.GCPNodePoolPlatform{}
	}

	// Validate required fields
	if o.MachineType == "" {
		return fmt.Errorf("GCP machine type is required")
	}

	// Update basic configuration
	nodePool.Spec.Platform.GCP.MachineType = o.MachineType
	nodePool.Spec.Platform.GCP.DiskType = o.DiskType
	nodePool.Spec.Platform.GCP.DiskSizeGb = o.DiskSizeGb
	nodePool.Spec.Platform.GCP.Subnetwork = o.Subnetwork
	nodePool.Spec.Platform.GCP.Image = o.Image
	nodePool.Spec.Platform.GCP.Preemptible = o.Preemptible
	nodePool.Spec.Platform.GCP.OnHostMaintenance = o.OnHostMaintenance

	// Set service account if provided
	if o.ServiceAccountEmail != "" {
		nodePool.Spec.Platform.GCP.ServiceAccount = &hyperv1.GCPServiceAccount{
			Email:  o.ServiceAccountEmail,
			Scopes: o.ServiceAccountScopes,
		}
	}

	// Set network tags
	if len(o.Tags) > 0 {
		nodePool.Spec.Platform.GCP.Tags = o.Tags
	}

	// Set labels
	if len(o.Labels) > 0 {
		labels := make(map[string]string)
		for _, label := range o.Labels {
			parts := strings.SplitN(label, "=", 2)
			if len(parts) == 2 {
				labels[parts[0]] = parts[1]
			}
		}
		if len(labels) > 0 {
			nodePool.Spec.Platform.GCP.Labels = labels
		}
	}

	return nil
}

func (o *GCPNodePoolPlatformOptions) Type() hyperv1.PlatformType {
	return hyperv1.GCPPlatform
}

func (o *GCPNodePoolPlatformOptions) Validate() error {
	if o.MachineType == "" {
		return fmt.Errorf("GCP machine type is required")
	}

	// Validate disk type
	validDiskTypes := map[string]bool{
		"pd-standard": true,
		"pd-ssd":      true,
		"pd-balanced": true,
	}
	if o.DiskType != "" && !validDiskTypes[o.DiskType] {
		return fmt.Errorf("invalid GCP disk type %q, must be one of: pd-standard, pd-ssd, pd-balanced", o.DiskType)
	}

	// Validate disk size
	if o.DiskSizeGb < 20 || o.DiskSizeGb > 65536 {
		return fmt.Errorf("GCP disk size must be between 20 and 65536 GB")
	}

	// Validate onHostMaintenance setting
	if o.OnHostMaintenance != "" {
		validMaintenanceOptions := map[string]bool{
			"MIGRATE":   true,
			"TERMINATE": true,
		}
		if !validMaintenanceOptions[o.OnHostMaintenance] {
			return fmt.Errorf("invalid GCP onHostMaintenance value %q, must be MIGRATE or TERMINATE", o.OnHostMaintenance)
		}
	}

	return nil
}