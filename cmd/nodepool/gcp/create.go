package gcp

import (
	"context"
	"fmt"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/nodepool/core"

	"github.com/spf13/cobra"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type GCPPlatformCreateOptions struct {
	*GCPNodePoolPlatformOptions
}

func NewCreateCommand(opts *core.CreateNodePoolOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gcp",
		Short:        "Create a NodePool with a GCP platform",
		SilenceUsage: true,
	}

	platformOpts := &GCPPlatformCreateOptions{
		GCPNodePoolPlatformOptions: DefaultOptions(),
	}

	BindOptions(platformOpts.GCPNodePoolPlatformOptions, cmd.Flags())

	cmd.RunE = opts.CreateRunFunc(platformOpts)

	return cmd
}

func (o *GCPPlatformCreateOptions) UpdateNodePool(ctx context.Context, nodePool *hyperv1.NodePool, hcluster *hyperv1.HostedCluster, client crclient.Client) error {
	nodePool.Spec.Platform.Type = hyperv1.GCPPlatform

	if err := o.GCPNodePoolPlatformOptions.UpdateNodePool(nodePool); err != nil {
		return fmt.Errorf("failed to update nodepool with GCP platform options: %w", err)
	}

	// Set defaults from HostedCluster if not specified
	if hcluster.Spec.Platform.GCP != nil {
		if nodePool.Spec.Platform.GCP.Subnetwork == "" && hcluster.Spec.Platform.GCP.Network != nil {
			nodePool.Spec.Platform.GCP.Subnetwork = hcluster.Spec.Platform.GCP.Network.Subnetwork
		}
		
		// Use cluster service account if none specified for nodepool
		if nodePool.Spec.Platform.GCP.ServiceAccount == nil && hcluster.Spec.Platform.GCP.ServiceAccount != nil {
			nodePool.Spec.Platform.GCP.ServiceAccount = &hyperv1.GCPServiceAccount{
				Email:  hcluster.Spec.Platform.GCP.ServiceAccount.Email,
				Scopes: hcluster.Spec.Platform.GCP.ServiceAccount.Scopes,
			}
		}
	}

	return nil
}

func (o *GCPPlatformCreateOptions) Type() hyperv1.PlatformType {
	return hyperv1.GCPPlatform
}

func (o *GCPPlatformCreateOptions) Validate(ctx context.Context, opts *core.CreateNodePoolOptions) error {
	return o.GCPNodePoolPlatformOptions.Validate()
}