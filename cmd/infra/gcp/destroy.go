package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"
	"github.com/openshift/hypershift/cmd/log"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
	"k8s.io/apimachinery/pkg/util/wait"
)

type DestroyInfraOptions struct {
	CredentialsOpts gcputil.GCPCredentialsOptions
	Project         string
	Region          string
	InfraID         string
	Name            string
}

type DestroyIAMOptions struct {
	CredentialsOpts gcputil.GCPCredentialsOptions
	Project         string
	InfraID         string
}

func NewDestroyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gcp",
		Short:        "Destroys GCP infrastructure resources for a cluster",
		SilenceUsage: true,
	}

	opts := DestroyInfraOptions{}

	cmd.Flags().StringVar(&opts.Project, "project", opts.Project, "GCP project ID")
	cmd.Flags().StringVar(&opts.Region, "region", opts.Region, "GCP region")
	cmd.Flags().StringVar(&opts.InfraID, "infra-id", opts.InfraID, "Cluster ID (required)")
	cmd.Flags().StringVar(&opts.Name, "name", opts.Name, "A name for the cluster")

	opts.CredentialsOpts.BindFlags(cmd.Flags())

	_ = cmd.MarkFlagRequired("infra-id")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		
		if opts.Name == "" {
			opts.Name = opts.InfraID
		}

		return opts.DestroyInfra(ctx, log.Log)
	}

	return cmd
}

func (o *DestroyInfraOptions) DestroyInfra(ctx context.Context, log logr.Logger) error {
	// Get the project ID
	projectID, err := o.CredentialsOpts.GetProjectID()
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}
	if o.Project == "" {
		o.Project = projectID
	}

	// Create GCP services
	computeService, err := o.CredentialsOpts.CreateComputeService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create compute service: %w", err)
	}

	dnsService, err := o.CredentialsOpts.CreateDNSService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create DNS service: %w", err)
	}

	log.Info("Starting GCP infrastructure destruction", "infraID", o.InfraID)

	// Destroy resources in reverse order of creation
	if err := o.destroyDNSZones(ctx, dnsService, log); err != nil {
		log.Error(err, "Failed to destroy DNS zones")
		// Continue with other resources
	}

	if err := o.destroyCloudNAT(ctx, computeService, log); err != nil {
		log.Error(err, "Failed to destroy Cloud NAT")
		// Continue with other resources
	}

	if err := o.destroyFirewallRules(ctx, computeService, log); err != nil {
		log.Error(err, "Failed to destroy firewall rules")
		// Continue with other resources
	}

	if err := o.destroySubnetworks(ctx, computeService, log); err != nil {
		log.Error(err, "Failed to destroy subnetworks")
		// Continue with other resources
	}

	if err := o.destroyNetworks(ctx, computeService, log); err != nil {
		log.Error(err, "Failed to destroy networks")
		// Continue with other resources
	}

	log.Info("Successfully destroyed GCP infrastructure", "infraID", o.InfraID)
	return nil
}

func (o *DestroyInfraOptions) destroyDNSZones(ctx context.Context, dnsService *dns.Service, log logr.Logger) error {
	log.Info("Destroying DNS zones")

	// List all managed zones and filter by cluster
	zones, err := dnsService.ManagedZones.List(o.Project).Do()
	if err != nil {
		return fmt.Errorf("failed to list DNS zones: %w", err)
	}

	for _, zone := range zones.ManagedZones {
		if strings.Contains(zone.Name, o.InfraID) {
			log.Info("Deleting DNS zone", "name", zone.Name)
			
			// Delete all resource record sets first (except NS and SOA)
			rrsets, err := dnsService.ResourceRecordSets.List(o.Project, zone.Name).Do()
			if err != nil {
				log.Error(err, "Failed to list resource record sets", "zone", zone.Name)
				continue
			}

			for _, rrset := range rrsets.Rrsets {
				if rrset.Type != "NS" && rrset.Type != "SOA" {
					log.Info("Deleting resource record set", "name", rrset.Name, "type", rrset.Type)
					if _, err := dnsService.ResourceRecordSets.Delete(o.Project, zone.Name, rrset.Name, rrset.Type).Do(); err != nil {
						log.Error(err, "Failed to delete resource record set", "name", rrset.Name)
					}
				}
			}

			// Delete the zone
			if err := dnsService.ManagedZones.Delete(o.Project, zone.Name).Do(); err != nil {
				if !isNotFoundError(err) {
					log.Error(err, "Failed to delete DNS zone", "name", zone.Name)
				}
			}
		}
	}

	return nil
}

func (o *DestroyInfraOptions) destroyCloudNAT(ctx context.Context, computeService *compute.Service, log logr.Logger) error {
	log.Info("Destroying Cloud NAT and router")

	routerName := o.InfraID + "-router"
	
	// Delete router (this will also delete associated NATs)
	op, err := computeService.Routers.Delete(o.Project, o.Region, routerName).Do()
	if err != nil {
		if !isNotFoundError(err) {
			return fmt.Errorf("failed to delete router: %w", err)
		}
		return nil
	}

	return o.waitForRegionalOperation(ctx, computeService, op.Name)
}

func (o *DestroyInfraOptions) destroyFirewallRules(ctx context.Context, computeService *compute.Service, log logr.Logger) error {
	log.Info("Destroying firewall rules")

	// List all firewall rules and filter by cluster
	firewalls, err := computeService.Firewalls.List(o.Project).Do()
	if err != nil {
		return fmt.Errorf("failed to list firewall rules: %w", err)
	}

	for _, firewall := range firewalls.Items {
		if strings.Contains(firewall.Name, o.InfraID) {
			log.Info("Deleting firewall rule", "name", firewall.Name)
			op, err := computeService.Firewalls.Delete(o.Project, firewall.Name).Do()
			if err != nil {
				if !isNotFoundError(err) {
					log.Error(err, "Failed to delete firewall rule", "name", firewall.Name)
				}
				continue
			}

			if err := o.waitForGlobalOperation(ctx, computeService, op.Name); err != nil {
				log.Error(err, "Failed to wait for firewall deletion", "name", firewall.Name)
			}
		}
	}

	return nil
}

func (o *DestroyInfraOptions) destroySubnetworks(ctx context.Context, computeService *compute.Service, log logr.Logger) error {
	log.Info("Destroying subnetworks")

	// List all subnetworks in the region and filter by cluster
	subnetworks, err := computeService.Subnetworks.List(o.Project, o.Region).Do()
	if err != nil {
		return fmt.Errorf("failed to list subnetworks: %w", err)
	}

	for _, subnetwork := range subnetworks.Items {
		if strings.Contains(subnetwork.Name, o.InfraID) {
			log.Info("Deleting subnetwork", "name", subnetwork.Name)
			op, err := computeService.Subnetworks.Delete(o.Project, o.Region, subnetwork.Name).Do()
			if err != nil {
				if !isNotFoundError(err) {
					log.Error(err, "Failed to delete subnetwork", "name", subnetwork.Name)
				}
				continue
			}

			if err := o.waitForRegionalOperation(ctx, computeService, op.Name); err != nil {
				log.Error(err, "Failed to wait for subnetwork deletion", "name", subnetwork.Name)
			}
		}
	}

	return nil
}

func (o *DestroyInfraOptions) destroyNetworks(ctx context.Context, computeService *compute.Service, log logr.Logger) error {
	log.Info("Destroying networks")

	// List all networks and filter by cluster
	networks, err := computeService.Networks.List(o.Project).Do()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks.Items {
		if strings.Contains(network.Name, o.InfraID) {
			log.Info("Deleting network", "name", network.Name)
			op, err := computeService.Networks.Delete(o.Project, network.Name).Do()
			if err != nil {
				if !isNotFoundError(err) {
					log.Error(err, "Failed to delete network", "name", network.Name)
				}
				continue
			}

			if err := o.waitForGlobalOperation(ctx, computeService, op.Name); err != nil {
				log.Error(err, "Failed to wait for network deletion", "name", network.Name)
			}
		}
	}

	return nil
}

func (o *DestroyInfraOptions) waitForGlobalOperation(ctx context.Context, computeService *compute.Service, operationName string) error {
	return wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		op, err := computeService.GlobalOperations.Get(o.Project, operationName).Do()
		if err != nil {
			return false, fmt.Errorf("failed to get operation status: %w", err)
		}

		if op.Status == "DONE" {
			if op.Error != nil {
				return false, fmt.Errorf("operation failed: %v", op.Error)
			}
			return true, nil
		}

		return false, nil
	})
}

func (o *DestroyInfraOptions) waitForRegionalOperation(ctx context.Context, computeService *compute.Service, operationName string) error {
	return wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		op, err := computeService.RegionOperations.Get(o.Project, o.Region, operationName).Do()
		if err != nil {
			return false, fmt.Errorf("failed to get operation status: %w", err)
		}

		if op.Status == "DONE" {
			if op.Error != nil {
				return false, fmt.Errorf("operation failed: %v", op.Error)
			}
			return true, nil
		}

		return false, nil
	})
}

func (o *DestroyIAMOptions) DestroyIAM(ctx context.Context, log logr.Logger) error {
	log.Info("Destroying GCP IAM resources", "infraID", o.InfraID)

	// Get the project ID
	projectID, err := o.CredentialsOpts.GetProjectID()
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}
	if o.Project == "" {
		o.Project = projectID
	}

	// Create IAM service
	iamService, err := o.CredentialsOpts.CreateIAMService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create IAM service: %w", err)
	}

	// List service accounts and filter by cluster
	serviceAccounts, err := iamService.Projects.ServiceAccounts.List("projects/" + o.Project).Do()
	if err != nil {
		return fmt.Errorf("failed to list service accounts: %w", err)
	}

	for _, sa := range serviceAccounts.Accounts {
		if strings.Contains(sa.DisplayName, o.InfraID) || strings.Contains(sa.Email, o.InfraID) {
			log.Info("Deleting service account", "email", sa.Email)
			_, err := iamService.Projects.ServiceAccounts.Delete(sa.Name).Do()
			if err != nil {
				if !isNotFoundError(err) {
					log.Error(err, "Failed to delete service account", "email", sa.Email)
				}
			}
		}
	}

	log.Info("Successfully destroyed GCP IAM resources", "infraID", o.InfraID)
	return nil
}

func isNotFoundError(err error) bool {
	if googleAPIError, ok := err.(*googleapi.Error); ok {
		return googleAPIError.Code == 404
	}
	return false
}