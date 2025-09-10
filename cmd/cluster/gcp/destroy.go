package gcp

import (
	"context"
	"fmt"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"
	gcpinfra "github.com/openshift/hypershift/cmd/infra/gcp"
	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"

	"github.com/spf13/cobra"
)

type DestroyOptions struct {
	Credentials          gcputil.GCPCredentialsOptions
	Project              string
	Region               string
	PreserveIAM          bool
	DestroyCloudResources bool
}

func NewDestroyCommand(opts *core.DestroyOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gcp",
		Short:        "Destroys a HostedCluster and its associated infrastructure on GCP",
		SilenceUsage: true,
	}

	destroyOpts := &DestroyOptions{
		DestroyCloudResources: true,
	}

	cmd.Flags().StringVar(&destroyOpts.Project, "project", destroyOpts.Project, "GCP project ID where the cluster infrastructure is located")
	cmd.Flags().StringVar(&destroyOpts.Region, "region", destroyOpts.Region, "GCP region where the cluster infrastructure is located")
	cmd.Flags().BoolVar(&destroyOpts.PreserveIAM, "preserve-iam", destroyOpts.PreserveIAM, "Preserve IAM resources during destruction")
	destroyOpts.Credentials.BindFlags(cmd.Flags())

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		return destroyOpts.Run(ctx, opts)
	}

	return cmd
}

func (o *DestroyOptions) Run(ctx context.Context, opts *core.DestroyOptions) error {
	hostedCluster, err := core.GetCluster(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to get HostedCluster: %w", err)
	}

	if hostedCluster != nil && hostedCluster.Spec.Platform.Type != hyperv1.GCPPlatform {
		return fmt.Errorf("hostedCluster %s is not a GCP cluster", hostedCluster.Name)
	}

	// Extract platform information from the cluster
	if hostedCluster != nil && hostedCluster.Spec.Platform.GCP != nil {
		if o.Project == "" {
			o.Project = hostedCluster.Spec.Platform.GCP.Project
		}
		if o.Region == "" {
			o.Region = hostedCluster.Spec.Platform.GCP.Region
		}
	}

	// Validate required parameters
	if o.Project == "" {
		return fmt.Errorf("GCP project must be specified via --project flag or from the HostedCluster")
	}
	if o.Region == "" {
		return fmt.Errorf("GCP region must be specified via --region flag or from the HostedCluster")
	}

	// Extract platform info from the cluster for infrastructure deletion
	if hostedCluster != nil {
		opts.InfraID = hostedCluster.Spec.InfraID
		if hostedCluster.Spec.Platform.GCP != nil {
			o.Project = hostedCluster.Spec.Platform.GCP.Project  
			o.Region = hostedCluster.Spec.Platform.GCP.Region
		}
	}

	// Destroy GCP infrastructure if requested
	if o.DestroyCloudResources {
		if err := o.destroyInfrastructure(ctx, opts); err != nil {
			return fmt.Errorf("failed to destroy GCP infrastructure: %w", err)
		}

		// Destroy IAM resources if not preserving them
		if !o.PreserveIAM {
			if err := o.destroyIAM(ctx, opts); err != nil {
				return fmt.Errorf("failed to destroy GCP IAM resources: %w", err)
			}
		}
	}

	opts.Log.Info("Successfully destroyed cluster")
	return nil
}

func (o *DestroyOptions) destroyInfrastructure(ctx context.Context, opts *core.DestroyOptions) error {
	opts.Log.Info("Destroying GCP infrastructure")

	destroyInfraOpts := gcpinfra.DestroyInfraOptions{
		CredentialsOpts: o.Credentials,
		Project:         o.Project,
		Region:          o.Region,
		InfraID:         opts.InfraID,
		Name:            opts.Name,
	}

	return destroyInfraOpts.DestroyInfra(ctx, opts.Log)
}

func (o *DestroyOptions) destroyIAM(ctx context.Context, opts *core.DestroyOptions) error {
	opts.Log.Info("Destroying GCP IAM resources")

	destroyIAMOpts := gcpinfra.DestroyIAMOptions{
		CredentialsOpts: o.Credentials,
		Project:         o.Project,
		InfraID:         opts.InfraID,
	}

	return destroyIAMOpts.DestroyIAM(ctx, opts.Log)
}