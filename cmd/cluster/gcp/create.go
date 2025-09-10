package gcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"
	gcpinfra "github.com/openshift/hypershift/cmd/infra/gcp"
	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

const (
	SATokenIssuerSecret = "sa-token-issuer-key"
)

type RawCreateOptions struct {
	Credentials                      gcputil.GCPCredentialsOptions
	CredentialSecretName             string
	Region                          string
	Zone                            string
	Project                         string
	MachineType                     string
	DiskType                        string
	DiskSizeGb                      int64
	Network                         string
	Subnetwork                      string
	ControlPlaneSubnetwork          string
	LoadBalancerSubnetwork          string
	KMSKeyName                      string
	IssuerURL                       string
	ServiceAccountTokenIssuerKeyPath string
	Labels                          []string
	ResourceTags                    []string
	ServiceAccountEmail             string
	ServiceAccountScopes            []string
	Preemptible                     bool
	OnHostMaintenance              string
	MultiArch                       bool
}

// validatedCreateOptions is a private wrapper that enforces a call of Validate() before Complete() can be invoked.
type validatedCreateOptions struct {
	*RawCreateOptions
}

type ValidatedCreateOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*validatedCreateOptions
}

func (o *RawCreateOptions) Validate(ctx context.Context, opts *core.CreateOptions) (core.PlatformCompleter, error) {
	if err := validateGCPOptions(ctx, opts, o); err != nil {
		return nil, err
	}

	return &ValidatedCreateOptions{
		validatedCreateOptions: &validatedCreateOptions{
			RawCreateOptions: o,
		},
	}, nil
}

// completedCreateOptions is a private wrapper that enforces a call of Complete() before cluster creation can be invoked.
type completedCreateOptions struct {
	*ValidatedCreateOptions

	infra             *gcpinfra.CreateInfraOutput
	iamInfo           *gcpinfra.CreateIAMOutput
	arch              string
	externalDNSDomain string
	namespace         string
	name              string
}

type CompletedCreateOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedCreateOptions
}

func DefaultOptions() *RawCreateOptions {
	return &RawCreateOptions{
		Region:            "us-central1",
		Zone:              "us-central1-a",
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

func validateGCPOptions(ctx context.Context, opts *core.CreateOptions, gcpOpts *RawCreateOptions) error {
	if gcpOpts.Project == "" {
		return fmt.Errorf("GCP project must be specified")
	}

	if gcpOpts.Region == "" {
		return fmt.Errorf("GCP region must be specified")
	}

	if gcpOpts.MachineType == "" {
		return fmt.Errorf("GCP machine type must be specified")
	}

	validDiskTypes := map[string]bool{
		"pd-standard": true,
		"pd-ssd":      true,
		"pd-balanced": true,
	}
	if !validDiskTypes[gcpOpts.DiskType] {
		return fmt.Errorf("invalid GCP disk type %q, must be one of: pd-standard, pd-ssd, pd-balanced", gcpOpts.DiskType)
	}

	if gcpOpts.DiskSizeGb < 20 || gcpOpts.DiskSizeGb > 65536 {
		return fmt.Errorf("GCP disk size must be between 20 and 65536 GB")
	}

	validMaintenanceOptions := map[string]bool{
		"MIGRATE":   true,
		"TERMINATE": true,
	}
	if !validMaintenanceOptions[gcpOpts.OnHostMaintenance] {
		return fmt.Errorf("invalid GCP onHostMaintenance value %q, must be MIGRATE or TERMINATE", gcpOpts.OnHostMaintenance)
	}

	return nil
}

func (o *ValidatedCreateOptions) Complete(ctx context.Context, opts *core.CreateOptions) (core.Platform, error) {
	output := &CompletedCreateOptions{
		completedCreateOptions: &completedCreateOptions{
			ValidatedCreateOptions: o,
			name:                   opts.Name,
			namespace:              opts.Namespace,
			arch:                   opts.Arch,
		},
	}

	if len(opts.ExternalDNSDomain) > 0 {
		output.externalDNSDomain = opts.ExternalDNSDomain
	}

	// Create GCP infrastructure if needed
	if shouldCreateInfra(opts, o) {
		infraOpts := CreateInfraOptions(o, opts)
		infraOutput, err := infraOpts.CreateInfra(ctx, opts.Log)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP infrastructure: %w", err)
		}
		output.infra = infraOutput
	}

	// Create IAM resources if needed
	if shouldCreateIAM(opts, o) {
		iamOpts := CreateIAMOptions(o, opts)
		iamOutput, err := iamOpts.CreateIAM(ctx, opts.Log)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP IAM resources: %w", err)
		}
		output.iamInfo = iamOutput
	}

	return output, nil
}

func shouldCreateInfra(opts *core.CreateOptions, gcpOpts *ValidatedCreateOptions) bool {
	return gcpOpts.Network == "" || gcpOpts.Subnetwork == ""
}

func shouldCreateIAM(opts *core.CreateOptions, gcpOpts *ValidatedCreateOptions) bool {
	return gcpOpts.ServiceAccountEmail == ""
}

func (o *CompletedCreateOptions) CreatePlatform(ctx context.Context, opts *core.CreateOptions) (*hyperv1.HostedCluster, error) {
	hcluster := &hyperv1.HostedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HostedCluster",
			APIVersion: hyperv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.namespace,
			Name:      o.name,
		},
		Spec: hyperv1.HostedClusterSpec{
			Platform: hyperv1.PlatformSpec{
				Type: hyperv1.GCPPlatform,
				GCP: &hyperv1.GCPPlatformSpec{
					Project: o.Project,
					Region:  o.Region,
					Zone:    o.Zone,
				},
			},
		},
	}

	// Set network configuration
	if o.Network != "" || o.Subnetwork != "" || o.ControlPlaneSubnetwork != "" || o.LoadBalancerSubnetwork != "" {
		hcluster.Spec.Platform.GCP.Network = &hyperv1.GCPNetworkSpec{
			Network:               o.Network,
			Subnetwork:            o.Subnetwork,
			ControlPlaneSubnet:    o.ControlPlaneSubnetwork,
			LoadBalancerSubnet:    o.LoadBalancerSubnetwork,
		}
	}

	// Set service account if provided
	if o.ServiceAccountEmail != "" {
		hcluster.Spec.Platform.GCP.ServiceAccount = &hyperv1.GCPServiceAccount{
			Email:  o.ServiceAccountEmail,
			Scopes: o.ServiceAccountScopes,
		}
	}

	// Set KMS key if provided
	if o.KMSKeyName != "" {
		hcluster.Spec.Platform.GCP.KMSKeyName = o.KMSKeyName
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
			hcluster.Spec.Platform.GCP.Labels = labels
		}
	}

	// Set resource tags
	if len(o.ResourceTags) > 0 {
		tags := make(map[string]string)
		for _, tag := range o.ResourceTags {
			parts := strings.SplitN(tag, "=", 2)
			if len(parts) == 2 {
				tags[parts[0]] = parts[1]
			}
		}
		if len(tags) > 0 {
			hcluster.Spec.Platform.GCP.ResourceTags = tags
		}
	}

	// Use values from created infrastructure if available
	if o.infra != nil {
		if o.infra.Network != "" {
			if hcluster.Spec.Platform.GCP.Network == nil {
				hcluster.Spec.Platform.GCP.Network = &hyperv1.GCPNetworkSpec{}
			}
			hcluster.Spec.Platform.GCP.Network.Network = o.infra.Network
		}
		if o.infra.Subnetwork != "" {
			if hcluster.Spec.Platform.GCP.Network == nil {
				hcluster.Spec.Platform.GCP.Network = &hyperv1.GCPNetworkSpec{}
			}
			hcluster.Spec.Platform.GCP.Network.Subnetwork = o.infra.Subnetwork
		}
	}

	// Use values from created IAM if available
	if o.iamInfo != nil {
		if o.iamInfo.ServiceAccountEmail != "" {
			if hcluster.Spec.Platform.GCP.ServiceAccount == nil {
				hcluster.Spec.Platform.GCP.ServiceAccount = &hyperv1.GCPServiceAccount{}
			}
			hcluster.Spec.Platform.GCP.ServiceAccount.Email = o.iamInfo.ServiceAccountEmail
		}
	}

	return hcluster, nil
}

func (o *CompletedCreateOptions) CreateCredentialSecret(ctx context.Context, c client.Client, opts *core.CreateOptions) (*corev1.Secret, error) {
	credentialsData, err := o.Credentials.LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load GCP credentials: %w", err)
	}

	credentialSecretName := o.CredentialSecretName
	if credentialSecretName == "" {
		credentialSecretName = o.name + "-gcp-creds"
	}

	credentialsSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.namespace,
			Name:      credentialSecretName,
		},
		Data: credentialsData,
		Type: corev1.SecretTypeOpaque,
	}

	return credentialsSecret, nil
}

func (o *CompletedCreateOptions) CreateNodePool(ctx context.Context, c client.Client, hcluster *hyperv1.HostedCluster, opts *core.CreateOptions) (*hyperv1.NodePool, error) {
	nodePool := &hyperv1.NodePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodePool",
			APIVersion: hyperv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hcluster.Namespace,
			Name:      hcluster.Name,
		},
		Spec: hyperv1.NodePoolSpec{
			ClusterName: hcluster.Name,
			Replicas:    ptr.To[int32](int32(opts.NodePoolReplicas)),
			Platform: hyperv1.NodePoolPlatform{
				Type: hyperv1.GCPPlatform,
				GCP: &hyperv1.GCPNodePoolPlatform{
					MachineType: o.MachineType,
					DiskType:    o.DiskType,
					DiskSizeGb:  o.DiskSizeGb,
					Subnetwork:  o.Subnetwork,
					Preemptible: o.Preemptible,
					OnHostMaintenance: o.OnHostMaintenance,
				},
			},
		},
	}

	// Set service account if provided
	if o.ServiceAccountEmail != "" {
		nodePool.Spec.Platform.GCP.ServiceAccount = &hyperv1.GCPServiceAccount{
			Email:  o.ServiceAccountEmail,
			Scopes: o.ServiceAccountScopes,
		}
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

	return nodePool, nil
}

func (o *CompletedCreateOptions) CreateServiceAccountTokenIssuerKey(ctx context.Context, c client.Client, opts *core.CreateOptions) (*corev1.Secret, error) {
	if o.ServiceAccountTokenIssuerKeyPath == "" {
		return nil, nil
	}

	privateKeyBytes, err := os.ReadFile(o.ServiceAccountTokenIssuerKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account token issuer private key: %w", err)
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.namespace,
			Name:      SATokenIssuerSecret,
		},
		Data: map[string][]byte{
			"service-account.key": privateKeyBytes,
		},
		Type: corev1.SecretTypeOpaque,
	}

	return secret, nil
}

// ApplyPlatformSpecifics applies GCP-specific configurations to the cluster
func (o *CompletedCreateOptions) ApplyPlatformSpecifics(hcluster *hyperv1.HostedCluster) error {
	// Apply any GCP-specific post-creation configurations here
	// For now, this is a no-op but can be extended for GCP-specific logic
	return nil
}

// GenerateNodePools generates the default NodePool for the cluster
func (o *CompletedCreateOptions) GenerateNodePools(constructor core.DefaultNodePoolConstructor) []*hyperv1.NodePool {
	var instanceType string
	if o.MachineType != "" {
		instanceType = o.MachineType
	} else {
		// Default GCP instance type
		instanceType = "n1-standard-4"
	}

	// Use the constructor to create the nodepool with the proper defaults
	nodePool := constructor(hyperv1.GCPPlatform, "")
	if nodePool.Spec.Management.UpgradeType == "" {
		nodePool.Spec.Management.UpgradeType = hyperv1.UpgradeTypeReplace
	}

	nodePool.Spec.Platform.GCP = &hyperv1.GCPNodePoolPlatform{
		MachineType:       instanceType,
		DiskType:          o.DiskType,
		DiskSizeGb:        o.DiskSizeGb,
		Subnetwork:        o.Subnetwork,
		Preemptible:       o.Preemptible,
		OnHostMaintenance: o.OnHostMaintenance,
	}

	// Set service account if provided
	if o.ServiceAccountEmail != "" {
		nodePool.Spec.Platform.GCP.ServiceAccount = &hyperv1.GCPServiceAccount{
			Email:  o.ServiceAccountEmail,
			Scopes: o.ServiceAccountScopes,
		}
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

	return []*hyperv1.NodePool{nodePool}
}

// GenerateResources generates additional resources needed for GCP cluster creation
func (o *CompletedCreateOptions) GenerateResources() ([]client.Object, error) {
	var result []client.Object
	
	// For now, return empty slice - this can be extended later for GCP-specific resources
	// such as secrets, configmaps, etc.
	
	return result, nil
}

func BindOptions(opts *RawCreateOptions, flags *flag.FlagSet) {
	bindCoreOptions(opts, flags)
}

func bindCoreOptions(opts *RawCreateOptions, flags *flag.FlagSet) {
	flags.StringVar(&opts.Project, "project", opts.Project, "GCP project ID where the cluster infrastructure will be created (required)")
	flags.StringVar(&opts.Region, "region", opts.Region, "GCP region where the cluster infrastructure will be created")
	flags.StringVar(&opts.Zone, "zone", opts.Zone, "GCP zone within the region where the cluster control plane will be created")
	flags.StringVar(&opts.MachineType, "instance-type", opts.MachineType, "GCP machine type for worker nodes")
	flags.StringVar(&opts.DiskType, "disk-type", opts.DiskType, "Type of disk for the root volume (pd-ssd, pd-standard, pd-balanced)")
	flags.Int64Var(&opts.DiskSizeGb, "disk-size", opts.DiskSizeGb, "Size of the disk in GB")
	flags.StringVar(&opts.Network, "network", opts.Network, "Name of the VPC network to use for the cluster")
	flags.StringVar(&opts.Subnetwork, "subnetwork", opts.Subnetwork, "Name of the subnet to use for the cluster")
	flags.StringVar(&opts.ControlPlaneSubnetwork, "control-plane-subnet", opts.ControlPlaneSubnetwork, "Subnet to use for the control plane instances")
	flags.StringVar(&opts.LoadBalancerSubnetwork, "load-balancer-subnet", opts.LoadBalancerSubnetwork, "Subnet to use for load balancers")
	flags.StringVar(&opts.KMSKeyName, "kms-key-name", opts.KMSKeyName, "Name of the Cloud KMS key to use for etcd encryption")
	flags.StringVar(&opts.IssuerURL, "oidc-issuer-url", "", "The OIDC provider issuer URL")
	flags.StringVar(&opts.ServiceAccountTokenIssuerKeyPath, "sa-token-issuer-private-key-path", "", "The file to the private key for the service account token issuer")
	flags.StringSliceVar(&opts.Labels, "labels", opts.Labels, "Additional labels to apply to GCP resources (e.g. 'key1=value1,key2=value2')")
	flags.StringSliceVar(&opts.ResourceTags, "resource-tags", opts.ResourceTags, "Additional tags to apply to GCP resources (e.g. 'key1=value1,key2=value2')")
	flags.StringVar(&opts.ServiceAccountEmail, "service-account-email", opts.ServiceAccountEmail, "Email address of the service account to use for instances")
	flags.StringSliceVar(&opts.ServiceAccountScopes, "service-account-scopes", opts.ServiceAccountScopes, "Access scopes for the service account")
	flags.BoolVar(&opts.Preemptible, "preemptible", opts.Preemptible, "Use preemptible instances for cost optimization")
	flags.StringVar(&opts.OnHostMaintenance, "on-host-maintenance", opts.OnHostMaintenance, "What to do when Google Compute Engine schedules a maintenance event (MIGRATE or TERMINATE)")

	_ = flags.MarkDeprecated("multi-arch", "Multi-arch validation is now performed automatically based on the release image and signaled in the HostedCluster.Status.PayloadArch.")
}

func BindDeveloperOptions(opts *RawCreateOptions, flags *flag.FlagSet) {
	bindCoreOptions(opts, flags)
	opts.Credentials.BindFlags(flags)
}

var _ core.Platform = (*CompletedCreateOptions)(nil)

func NewCreateCommand(opts *core.RawCreateOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gcp",
		Short:        "Creates basic functional HostedCluster resources on GCP",
		SilenceUsage: true,
	}

	gcpOpts := DefaultOptions()
	BindDeveloperOptions(gcpOpts, cmd.Flags())
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if opts.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
		}

		if err := core.CreateCluster(ctx, opts, gcpOpts); err != nil {
			opts.Log.Error(err, "Failed to create cluster")
			return err
		}
		return nil
	}

	return cmd
}

func CreateInfraOptions(gcpOpts *ValidatedCreateOptions, opts *core.CreateOptions) gcpinfra.CreateInfraOptions {
	return gcpinfra.CreateInfraOptions{
		Region:     gcpOpts.Region,
		Zone:       gcpOpts.Zone,
		Project:    gcpOpts.Project,
		InfraID:    opts.InfraID,
		Name:       opts.Name,
		BaseDomain: opts.BaseDomain,
		Network:    gcpOpts.Network,
		Subnetwork: gcpOpts.Subnetwork,
		CredentialsOpts: gcpOpts.Credentials,
		Labels:     gcpOpts.Labels,
		ResourceTags: gcpOpts.ResourceTags,
	}
}

func CreateIAMOptions(gcpOpts *ValidatedCreateOptions, opts *core.CreateOptions) gcpinfra.CreateIAMOptions {
	return gcpinfra.CreateIAMOptions{
		Project:         gcpOpts.Project,
		InfraID:         opts.InfraID,
		CredentialsOpts: gcpOpts.Credentials,
	}
}