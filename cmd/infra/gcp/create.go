package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"
	"github.com/openshift/hypershift/cmd/log"
	"github.com/openshift/hypershift/cmd/util"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type CreateInfraOptions struct {
	CredentialsOpts  gcputil.GCPCredentialsOptions
	Region           string
	Zone             string
	Project          string
	InfraID          string
	Name             string
	BaseDomain       string
	BaseDomainPrefix string
	Network          string
	Subnetwork       string
	OutputFile       string
	Labels           []string
	ResourceTags     []string
	
	CredentialsSecretData *util.CredentialsSecretData
}

type CreateInfraOutput struct {
	Name                    string            `json:"name"`
	InfraID                 string            `json:"infraID"`
	Project                 string            `json:"project"`
	Region                  string            `json:"region"`
	Zone                    string            `json:"zone"`
	Network                 string            `json:"network"`
	Subnetwork              string            `json:"subnetwork"`
	ControlPlaneSubnetwork  string            `json:"controlPlaneSubnetwork,omitempty"`
	LoadBalancerSubnetwork  string            `json:"loadBalancerSubnetwork,omitempty"`
	ComputeSubnet           string            `json:"computeSubnet"`
	PrivateDNSZone          string            `json:"privateDNSZone,omitempty"`
	PublicDNSZone           string            `json:"publicDNSZone,omitempty"`
	Labels                  map[string]string `json:"labels,omitempty"`
	ResourceTags            map[string]string `json:"resourceTags,omitempty"`
}

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gcp",
		Short:        "Creates GCP infrastructure resources for a cluster",
		SilenceUsage: true,
	}

	opts := CreateInfraOptions{
		Region: "us-central1",
		Zone:   "us-central1-a",
	}

	cmd.Flags().StringVar(&opts.Region, "region", opts.Region, "GCP region")
	cmd.Flags().StringVar(&opts.Zone, "zone", opts.Zone, "GCP zone")
	cmd.Flags().StringVar(&opts.Project, "project", opts.Project, "GCP project ID")
	cmd.Flags().StringVar(&opts.InfraID, "infra-id", opts.InfraID, "Cluster ID (required)")
	cmd.Flags().StringVar(&opts.Name, "name", opts.Name, "A name for the cluster")
	cmd.Flags().StringVar(&opts.BaseDomain, "base-domain", opts.BaseDomain, "The ingress base domain for the cluster")
	cmd.Flags().StringVar(&opts.BaseDomainPrefix, "base-domain-prefix", opts.BaseDomainPrefix, "The ingress base domain prefix for the cluster, defaults to cluster name. Use 'none' for an empty prefix")
	cmd.Flags().StringVar(&opts.Network, "network", opts.Network, "An existing network name to use for the cluster")
	cmd.Flags().StringVar(&opts.Subnetwork, "subnetwork", opts.Subnetwork, "An existing subnetwork name to use for the cluster")
	cmd.Flags().StringVar(&opts.OutputFile, "output-file", opts.OutputFile, "Path to file that will contain output information from infra resources (optional)")
	cmd.Flags().StringSliceVar(&opts.Labels, "labels", opts.Labels, "Additional labels to apply to GCP resources (e.g. 'key1=value1,key2=value2')")
	cmd.Flags().StringSliceVar(&opts.ResourceTags, "resource-tags", opts.ResourceTags, "Additional tags to apply to GCP resources (e.g. 'key1=value1,key2=value2')")

	opts.CredentialsOpts.BindFlags(cmd.Flags())

	_ = cmd.MarkFlagRequired("infra-id")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		
		if opts.Name == "" {
			opts.Name = opts.InfraID
		}

		_, err := opts.CreateInfra(ctx, log.Log)
		return err
	}

	return cmd
}

func (o *CreateInfraOptions) CreateInfra(ctx context.Context, log logr.Logger) (*CreateInfraOutput, error) {
	// Get the project ID
	projectID, err := o.CredentialsOpts.GetProjectID()
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}
	if o.Project == "" {
		o.Project = projectID
	}

	// Create GCP services
	computeService, err := o.CredentialsOpts.CreateComputeService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	dnsService, err := o.CredentialsOpts.CreateDNSService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS service: %w", err)
	}

	// Validate project and region
	rmService, err := o.CredentialsOpts.CreateResourceManagerService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager service: %w", err)
	}

	if err := gcputil.ValidateProject(ctx, rmService, o.Project); err != nil {
		return nil, fmt.Errorf("project validation failed: %w", err)
	}

	if err := gcputil.ValidateRegion(ctx, computeService, o.Project, o.Region); err != nil {
		return nil, fmt.Errorf("region validation failed: %w", err)
	}

	// Convert labels and tags to maps
	labels := make(map[string]string)
	for _, label := range o.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	resourceTags := make(map[string]string)
	for _, tag := range o.ResourceTags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 2 {
			resourceTags[parts[0]] = parts[1]
		}
	}

	// Add cluster-specific labels
	labels["kubernetes-io-cluster-"+o.InfraID] = "owned"
	labels["hypershift.openshift.io/cluster"] = o.Name

	output := &CreateInfraOutput{
		Name:         o.Name,
		InfraID:      o.InfraID,
		Project:      o.Project,
		Region:       o.Region,
		Zone:         o.Zone,
		Labels:       labels,
		ResourceTags: resourceTags,
	}

	// Create or validate network
	if o.Network == "" {
		networkName := o.InfraID + "-network"
		log.Info("Creating VPC network", "name", networkName)
		if err := o.createNetwork(ctx, computeService, networkName, labels); err != nil {
			return nil, fmt.Errorf("failed to create network: %w", err)
		}
		output.Network = networkName
	} else {
		log.Info("Using existing network", "name", o.Network)
		output.Network = o.Network
	}

	// Create or validate subnetwork
	if o.Subnetwork == "" {
		subnetworkName := o.InfraID + "-subnet"
		log.Info("Creating subnetwork", "name", subnetworkName)
		if err := o.createSubnetwork(ctx, computeService, subnetworkName, output.Network, "10.0.0.0/16", labels); err != nil {
			return nil, fmt.Errorf("failed to create subnetwork: %w", err)
		}
		output.Subnetwork = subnetworkName
		output.ComputeSubnet = "10.0.0.0/16"
	} else {
		log.Info("Using existing subnetwork", "name", o.Subnetwork)
		output.Subnetwork = o.Subnetwork
		// Get the existing subnetwork's CIDR
		subnet, err := computeService.Subnetworks.Get(o.Project, o.Region, o.Subnetwork).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get existing subnetwork: %w", err)
		}
		output.ComputeSubnet = subnet.IpCidrRange
	}

	// Create firewall rules
	log.Info("Creating firewall rules")
	if err := o.createFirewallRules(ctx, computeService, output.Network, labels); err != nil {
		return nil, fmt.Errorf("failed to create firewall rules: %w", err)
	}

	// Create Cloud NAT if needed
	log.Info("Creating Cloud NAT gateway")
	if err := o.createCloudNAT(ctx, computeService, output.Network, labels); err != nil {
		return nil, fmt.Errorf("failed to create Cloud NAT: %w", err)
	}

	// Create DNS zones if base domain is provided
	if o.BaseDomain != "" {
		log.Info("Creating DNS zones")
		if err := o.createDNSZones(ctx, dnsService, output, labels); err != nil {
			return nil, fmt.Errorf("failed to create DNS zones: %w", err)
		}
	}

	// Output results
	if o.OutputFile != "" {
		if err := o.writeOutput(output); err != nil {
			return nil, fmt.Errorf("failed to write output file: %w", err)
		}
	}

	log.Info("Successfully created GCP infrastructure", "infraID", o.InfraID)
	return output, nil
}

func (o *CreateInfraOptions) createNetwork(ctx context.Context, computeService *compute.Service, networkName string, labels map[string]string) error {
	network := &compute.Network{
		Name:                  networkName,
		AutoCreateSubnetworks: false,
		RoutingConfig: &compute.NetworkRoutingConfig{
			RoutingMode: "REGIONAL",
		},
		// Note: Labels are not supported in this API version
	}

	op, err := computeService.Networks.Insert(o.Project, network).Do()
	if err != nil {
		return fmt.Errorf("failed to insert network: %w", err)
	}

	// Wait for operation to complete
	return o.waitForGlobalOperation(ctx, computeService, op.Name)
}

func (o *CreateInfraOptions) createSubnetwork(ctx context.Context, computeService *compute.Service, subnetworkName, networkName, cidr string, labels map[string]string) error {
	subnetwork := &compute.Subnetwork{
		Name:         subnetworkName,
		Network:      fmt.Sprintf("projects/%s/global/networks/%s", o.Project, networkName),
		IpCidrRange:  cidr,
		Region:       o.Region,
		PrivateIpGoogleAccess: true,
	}

	op, err := computeService.Subnetworks.Insert(o.Project, o.Region, subnetwork).Do()
	if err != nil {
		return fmt.Errorf("failed to insert subnetwork: %w", err)
	}

	// Wait for operation to complete
	return o.waitForRegionalOperation(ctx, computeService, op.Name)
}

func (o *CreateInfraOptions) createFirewallRules(ctx context.Context, computeService *compute.Service, networkName string, labels map[string]string) error {
	networkURL := fmt.Sprintf("projects/%s/global/networks/%s", o.Project, networkName)

	// Allow internal traffic
	internalRule := &compute.Firewall{
		Name:    o.InfraID + "-allow-internal",
		Network: networkURL,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{"0-65535"},
			},
			{
				IPProtocol: "udp",
				Ports:      []string{"0-65535"},
			},
			{
				IPProtocol: "icmp",
			},
		},
		SourceRanges: []string{"10.0.0.0/16"},
		TargetTags:   []string{o.InfraID + "-worker", o.InfraID + "-master"},
	}

	op, err := computeService.Firewalls.Insert(o.Project, internalRule).Do()
	if err != nil {
		return fmt.Errorf("failed to create internal firewall rule: %w", err)
	}
	if err := o.waitForGlobalOperation(ctx, computeService, op.Name); err != nil {
		return err
	}

	// Allow SSH
	sshRule := &compute.Firewall{
		Name:    o.InfraID + "-allow-ssh",
		Network: networkURL,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{"22"},
			},
		},
		SourceRanges: []string{"0.0.0.0/0"},
		TargetTags:   []string{o.InfraID + "-worker", o.InfraID + "-master"},
	}

	op, err = computeService.Firewalls.Insert(o.Project, sshRule).Do()
	if err != nil {
		return fmt.Errorf("failed to create SSH firewall rule: %w", err)
	}
	if err := o.waitForGlobalOperation(ctx, computeService, op.Name); err != nil {
		return err
	}

	// Allow API server access
	apiRule := &compute.Firewall{
		Name:    o.InfraID + "-allow-api-server",
		Network: networkURL,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{"6443"},
			},
		},
		SourceRanges: []string{"0.0.0.0/0"},
		TargetTags:   []string{o.InfraID + "-master"},
	}

	op, err = computeService.Firewalls.Insert(o.Project, apiRule).Do()
	if err != nil {
		return fmt.Errorf("failed to create API server firewall rule: %w", err)
	}
	return o.waitForGlobalOperation(ctx, computeService, op.Name)
}

func (o *CreateInfraOptions) createCloudNAT(ctx context.Context, computeService *compute.Service, networkName string, labels map[string]string) error {
	// Create router first
	routerName := o.InfraID + "-router"
	router := &compute.Router{
		Name:    routerName,
		Network: fmt.Sprintf("projects/%s/global/networks/%s", o.Project, networkName),
		Region:  o.Region,
	}

	op, err := computeService.Routers.Insert(o.Project, o.Region, router).Do()
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}
	if err := o.waitForRegionalOperation(ctx, computeService, op.Name); err != nil {
		return err
	}

	// Create NAT
	natName := o.InfraID + "-nat"
	nat := &compute.RouterNat{
		Name:                          natName,
		SourceSubnetworkIpRangesToNat: "ALL_SUBNETWORKS_ALL_IP_RANGES",
		NatIpAllocateOption:           "AUTO_ONLY",
	}

	// Add NAT to router
	router.Nats = []*compute.RouterNat{nat}
	op, err = computeService.Routers.Patch(o.Project, o.Region, routerName, router).Do()
	if err != nil {
		return fmt.Errorf("failed to add NAT to router: %w", err)
	}

	return o.waitForRegionalOperation(ctx, computeService, op.Name)
}

func (o *CreateInfraOptions) createDNSZones(ctx context.Context, dnsService *dns.Service, output *CreateInfraOutput, labels map[string]string) error {
	// Create private DNS zone
	privateZoneName := strings.ReplaceAll(o.InfraID+"-private-zone", ".", "-")
	privateDNS := fmt.Sprintf("%s.%s", o.InfraID, o.BaseDomain)
	
	privateZone := &dns.ManagedZone{
		Name:        privateZoneName,
		DnsName:     privateDNS + ".",
		Description: fmt.Sprintf("Private DNS zone for %s", o.Name),
		Visibility:  "private",
		PrivateVisibilityConfig: &dns.ManagedZonePrivateVisibilityConfig{
			Networks: []*dns.ManagedZonePrivateVisibilityConfigNetwork{
				{
					NetworkUrl: fmt.Sprintf("projects/%s/global/networks/%s", o.Project, output.Network),
				},
			},
		},
		// Note: Labels are not supported in this API version
	}

	_, err := dnsService.ManagedZones.Create(o.Project, privateZone).Do()
	if err != nil {
		return fmt.Errorf("failed to create private DNS zone: %w", err)
	}
	output.PrivateDNSZone = privateZoneName

	return nil
}

func (o *CreateInfraOptions) waitForGlobalOperation(ctx context.Context, computeService *compute.Service, operationName string) error {
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

func (o *CreateInfraOptions) waitForRegionalOperation(ctx context.Context, computeService *compute.Service, operationName string) error {
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

func (o *CreateInfraOptions) writeOutput(output *CreateInfraOutput) error {
	outputBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	if err := os.WriteFile(o.OutputFile, outputBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}