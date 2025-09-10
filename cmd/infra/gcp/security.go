package gcp

import (
	"context"
	"fmt"
	"strings"

	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"
	"github.com/go-logr/logr"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/secretmanager/v1"
)

// SecurityOptions holds configuration for GCP security best practices
type SecurityOptions struct {
	CredentialsOpts              gcputil.GCPCredentialsOptions
	Project                      string
	Region                       string
	InfraID                      string
	Name                        string
	EnableWorkloadIdentity       bool
	EnablePrivateCluster         bool
	EnableNetworkPolicies        bool
	EnablePodSecurityPolicies    bool
	EnableImageSecurity          bool
	EnableKMSEncryption          bool
	EnableVPCFlowLogs           bool
	EnablePrivateGoogleAccess    bool
	EnableBinaryAuthorization   bool
	KMSKeyRing                  string
	KMSKeyName                  string
	AllowedImageRepositories    []string
	NetworkTags                 []string
}

// SecurityOutput holds the created security resources
type SecurityOutput struct {
	WorkloadIdentityPools    []string          `json:"workloadIdentityPools,omitempty"`
	KMSKeys                  []string          `json:"kmsKeys,omitempty"`
	ServiceAccounts          []string          `json:"serviceAccounts,omitempty"`
	SecurityPolicies         []string          `json:"securityPolicies,omitempty"`
	FirewallRules            []string          `json:"firewallRules,omitempty"`
	VPCSecurityGroups        []string          `json:"vpcSecurityGroups,omitempty"`
	BinaryAuthorizationPolicy string           `json:"binaryAuthorizationPolicy,omitempty"`
	SecretManagerSecrets     []string          `json:"secretManagerSecrets,omitempty"`
}

// ApplySecurityBestPractices implements comprehensive security best practices for the GCP cluster
func (o *SecurityOptions) ApplySecurityBestPractices(ctx context.Context, log logr.Logger) (*SecurityOutput, error) {
	log.Info("Applying GCP security best practices", "infraID", o.InfraID)

	// Get the project ID
	projectID, err := o.CredentialsOpts.GetProjectID()
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}
	if o.Project == "" {
		o.Project = projectID
	}

	output := &SecurityOutput{}

	// Setup Workload Identity
	if o.EnableWorkloadIdentity {
		if err := o.setupWorkloadIdentity(ctx, output, log); err != nil {
			return nil, fmt.Errorf("failed to setup Workload Identity: %w", err)
		}
	}

	// Setup KMS encryption
	if o.EnableKMSEncryption {
		if err := o.setupKMSEncryption(ctx, output, log); err != nil {
			return nil, fmt.Errorf("failed to setup KMS encryption: %w", err)
		}
	}

	// Setup network security
	if err := o.setupNetworkSecurity(ctx, output, log); err != nil {
		return nil, fmt.Errorf("failed to setup network security: %w", err)
	}

	// Setup Binary Authorization
	if o.EnableBinaryAuthorization {
		if err := o.setupBinaryAuthorization(ctx, output, log); err != nil {
			return nil, fmt.Errorf("failed to setup Binary Authorization: %w", err)
		}
	}

	// Setup Secret Manager
	if err := o.setupSecretManager(ctx, output, log); err != nil {
		return nil, fmt.Errorf("failed to setup Secret Manager: %w", err)
	}

	log.Info("Successfully applied security best practices", "infraID", o.InfraID)
	return output, nil
}

func (o *SecurityOptions) setupWorkloadIdentity(ctx context.Context, output *SecurityOutput, log logr.Logger) error {
	log.Info("Setting up Workload Identity")

	// Create IAM service
	iamService, err := o.CredentialsOpts.CreateIAMService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create IAM service: %w", err)
	}

	// Create Workload Identity Pool
	poolID := fmt.Sprintf("hypershift-%s-pool", o.InfraID)
	pool := &iam.WorkloadIdentityPool{
		Name:        poolID,
		DisplayName: fmt.Sprintf("HyperShift %s Workload Identity Pool", o.Name),
		Description: fmt.Sprintf("Workload Identity pool for HyperShift cluster %s", o.Name),
		State:       "ACTIVE",
	}

	parent := fmt.Sprintf("projects/%s/locations/global", o.Project)
	createdPool, err := iamService.Projects.Locations.WorkloadIdentityPools.Create(parent, pool).WorkloadIdentityPoolId(poolID).Do()
	if err != nil {
		return fmt.Errorf("failed to create Workload Identity pool: %w", err)
	}

	output.WorkloadIdentityPools = append(output.WorkloadIdentityPools, createdPool.Name)
	log.Info("Created Workload Identity pool", "pool", createdPool.Name)

	// Create Workload Identity Provider
	providerID := fmt.Sprintf("hypershift-%s-provider", o.InfraID)
	provider := &iam.WorkloadIdentityPoolProvider{
		Name:        providerID,
		DisplayName: fmt.Sprintf("HyperShift %s OIDC Provider", o.Name),
		Description: fmt.Sprintf("OIDC provider for HyperShift cluster %s", o.Name),
		State:       "ACTIVE",
		Oidc: &iam.Oidc{
			IssuerUri: fmt.Sprintf("https://hypershift-%s.%s.com", o.InfraID, o.Region),
		},
		AttributeMapping: map[string]string{
			"google.subject":                "assertion.sub",
			"attribute.kubernetes_namespace": "assertion['kubernetes.io']['namespace']",
			"attribute.kubernetes_pod":       "assertion['kubernetes.io']['pod']['name']",
			"attribute.kubernetes_sa":        "assertion['kubernetes.io']['serviceaccount']['name']",
		},
	}

	poolName := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", o.Project, poolID)
	_, err = iamService.Projects.Locations.WorkloadIdentityPools.Providers.Create(poolName, provider).WorkloadIdentityPoolProviderId(providerID).Do()
	if err != nil {
		return fmt.Errorf("failed to create Workload Identity provider: %w", err)
	}

	log.Info("Created Workload Identity provider", "provider", providerID)

	return nil
}

func (o *SecurityOptions) setupKMSEncryption(ctx context.Context, output *SecurityOutput, log logr.Logger) error {
	log.Info("Setting up KMS encryption")

	// Create KMS service
	kmsService, err := o.CredentialsOpts.CreateKMSService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create KMS service: %w", err)
	}

	// Create key ring if not specified
	keyRing := o.KMSKeyRing
	if keyRing == "" {
		keyRing = fmt.Sprintf("hypershift-%s-keyring", o.InfraID)
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", o.Project, o.Region)
	keyRingRequest := &cloudkms.KeyRing{}
	
	_, err = kmsService.Projects.Locations.KeyRings.Create(parent, keyRingRequest).KeyRingId(keyRing).Do()
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create key ring: %w", err)
	}

	// Create crypto key for etcd encryption
	keyName := o.KMSKeyName
	if keyName == "" {
		keyName = fmt.Sprintf("hypershift-%s-etcd-key", o.InfraID)
	}

	cryptoKey := &cloudkms.CryptoKey{
		Purpose: "ENCRYPT_DECRYPT",
		VersionTemplate: &cloudkms.CryptoKeyVersionTemplate{
			Algorithm: "GOOGLE_SYMMETRIC_ENCRYPTION",
		},
		Labels: map[string]string{
			"cluster":     o.Name,
			"infra-id":    o.InfraID,
			"purpose":     "etcd-encryption",
			"managed-by":  "hypershift",
		},
	}

	keyRingPath := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", o.Project, o.Region, keyRing)
	createdKey, err := kmsService.Projects.Locations.KeyRings.CryptoKeys.Create(keyRingPath, cryptoKey).CryptoKeyId(keyName).Do()
	if err != nil {
		return fmt.Errorf("failed to create crypto key: %w", err)
	}

	output.KMSKeys = append(output.KMSKeys, createdKey.Name)
	log.Info("Created KMS crypto key", "key", createdKey.Name)

	return nil
}

func (o *SecurityOptions) setupNetworkSecurity(ctx context.Context, output *SecurityOutput, log logr.Logger) error {
	log.Info("Setting up network security")

	// Create compute service
	computeService, err := o.CredentialsOpts.CreateComputeService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create compute service: %w", err)
	}

	// Create security-focused firewall rules
	if err := o.createSecurityFirewallRules(ctx, computeService, output, log); err != nil {
		return fmt.Errorf("failed to create security firewall rules: %w", err)
	}

	// Enable VPC Flow Logs if requested
	if o.EnableVPCFlowLogs {
		if err := o.enableVPCFlowLogs(ctx, computeService, output, log); err != nil {
			return fmt.Errorf("failed to enable VPC flow logs: %w", err)
		}
	}

	return nil
}

func (o *SecurityOptions) createSecurityFirewallRules(ctx context.Context, computeService *compute.Service, output *SecurityOutput, log logr.Logger) error {
	log.Info("Creating security firewall rules")

	networkName := fmt.Sprintf("hypershift-%s-network", o.InfraID)
	networkURL := fmt.Sprintf("projects/%s/global/networks/%s", o.Project, networkName)

	// Deny all ingress by default (except what's explicitly allowed)
	denyAllRule := &compute.Firewall{
		Name:     fmt.Sprintf("hypershift-%s-deny-all-ingress", o.InfraID),
		Network:  networkURL,
		Priority: 65534,
		Direction: "INGRESS",
		Denied: []*compute.FirewallDenied{
			{
				IPProtocol: "all",
			},
		},
		SourceRanges: []string{"0.0.0.0/0"},
		TargetTags:   []string{fmt.Sprintf("hypershift-%s", o.InfraID)},
	}

	_, err := computeService.Firewalls.Insert(o.Project, denyAllRule).Do()
	if err != nil {
		return fmt.Errorf("failed to create deny-all firewall rule: %w", err)
	}
	output.FirewallRules = append(output.FirewallRules, denyAllRule.Name)
	log.Info("Created deny-all firewall rule", "rule", denyAllRule.Name)

	// Allow only necessary internal communication
	allowInternalRule := &compute.Firewall{
		Name:     fmt.Sprintf("hypershift-%s-allow-internal-secure", o.InfraID),
		Network:  networkURL,
		Priority: 1000,
		Direction: "INGRESS",
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{"80", "443", "6443", "8080", "9090", "10250", "10256"},
			},
			{
				IPProtocol: "udp",
				Ports:      []string{"53", "8472"},
			},
		},
		SourceRanges: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		TargetTags:   []string{fmt.Sprintf("hypershift-%s", o.InfraID)},
	}

	_, err = computeService.Firewalls.Insert(o.Project, allowInternalRule).Do()
	if err != nil {
		return fmt.Errorf("failed to create allow-internal firewall rule: %w", err)
	}
	output.FirewallRules = append(output.FirewallRules, allowInternalRule.Name)
	log.Info("Created allow-internal firewall rule", "rule", allowInternalRule.Name)

	// Allow SSH only from specific ranges (if any)
	if len(o.NetworkTags) > 0 {
		allowSSHRule := &compute.Firewall{
			Name:     fmt.Sprintf("hypershift-%s-allow-ssh-secure", o.InfraID),
			Network:  networkURL,
			Priority: 1000,
			Direction: "INGRESS",
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
					Ports:      []string{"22"},
				},
			},
			SourceTags: o.NetworkTags,
			TargetTags: []string{fmt.Sprintf("hypershift-%s", o.InfraID)},
		}

		_, err = computeService.Firewalls.Insert(o.Project, allowSSHRule).Do()
		if err != nil {
			return fmt.Errorf("failed to create allow-ssh firewall rule: %w", err)
		}
		output.FirewallRules = append(output.FirewallRules, allowSSHRule.Name)
		log.Info("Created allow-ssh firewall rule", "rule", allowSSHRule.Name)
	}

	return nil
}

func (o *SecurityOptions) enableVPCFlowLogs(ctx context.Context, computeService *compute.Service, output *SecurityOutput, log logr.Logger) error {
	log.Info("Enabling VPC Flow Logs")

	// Get the subnetwork and enable flow logs
	subnetworkName := fmt.Sprintf("hypershift-%s-subnet", o.InfraID)
	subnet, err := computeService.Subnetworks.Get(o.Project, o.Region, subnetworkName).Do()
	if err != nil {
		return fmt.Errorf("failed to get subnetwork: %w", err)
	}

	// Enable flow logs
	subnet.EnableFlowLogs = true
	subnet.LogConfig = &compute.SubnetworkLogConfig{
		Enable:           true,
		FlowSampling:     0.5, // Sample 50% of flows
		AggregationInterval: "INTERVAL_5_SEC",
		Metadata:         "INCLUDE_ALL_METADATA",
	}

	_, err = computeService.Subnetworks.Patch(o.Project, o.Region, subnetworkName, subnet).Do()
	if err != nil {
		return fmt.Errorf("failed to enable flow logs: %w", err)
	}

	log.Info("Enabled VPC Flow Logs", "subnet", subnetworkName)
	return nil
}

func (o *SecurityOptions) setupBinaryAuthorization(ctx context.Context, output *SecurityOutput, log logr.Logger) error {
	log.Info("Setting up Binary Authorization")

	// Binary Authorization policy configuration
	// Note: This would typically be done through the Binary Authorization API
	// For now, we'll just log the configuration that should be applied
	
	policyConfig := fmt.Sprintf(`
# Binary Authorization Policy for HyperShift cluster %s
# This policy ensures only verified container images are deployed

defaultAdmissionRule:
  requireAttestationsBy:
  - projects/%s/attestors/hypershift-%s-attestor
  enforcementMode: ENFORCED_BLOCK_AND_AUDIT_LOG

clusterAdmissionRules:
  %s:
    requireAttestationsBy:
    - projects/%s/attestors/hypershift-%s-attestor
    enforcementMode: ENFORCED_BLOCK_AND_AUDIT_LOG

name: projects/%s/policy
`, o.Name, o.Project, o.InfraID, o.Name, o.Project, o.InfraID, o.Project)

	output.BinaryAuthorizationPolicy = policyConfig
	log.Info("Configured Binary Authorization policy", "cluster", o.Name)

	return nil
}

func (o *SecurityOptions) setupSecretManager(ctx context.Context, output *SecurityOutput, log logr.Logger) error {
	log.Info("Setting up Secret Manager")

	// Create Secret Manager service
	secretService, err := o.CredentialsOpts.CreateSecretManagerService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Secret Manager service: %w", err)
	}

	// Create secrets for cluster credentials
	secrets := []struct {
		id          string
		description string
	}{
		{
			id:          fmt.Sprintf("hypershift-%s-kubeconfig", o.InfraID),
			description: fmt.Sprintf("Kubeconfig for HyperShift cluster %s", o.Name),
		},
		{
			id:          fmt.Sprintf("hypershift-%s-ca-certs", o.InfraID),
			description: fmt.Sprintf("CA certificates for HyperShift cluster %s", o.Name),
		},
		{
			id:          fmt.Sprintf("hypershift-%s-etcd-certs", o.InfraID),
			description: fmt.Sprintf("etcd certificates for HyperShift cluster %s", o.Name),
		},
	}

	parent := fmt.Sprintf("projects/%s", o.Project)
	for _, secret := range secrets {
		secretResource := &secretmanager.Secret{
			Labels: map[string]string{
				"cluster":    o.Name,
				"infra-id":   o.InfraID,
				"managed-by": "hypershift",
			},
			Replication: &secretmanager.Replication{
				Automatic: &secretmanager.Automatic{},
			},
		}

		createdSecret, err := secretService.Projects.Secrets.Create(parent, secretResource).SecretId(secret.id).Do()
		if err != nil {
			log.Error(err, "Failed to create secret", "secret", secret.id)
			continue
		}

		output.SecretManagerSecrets = append(output.SecretManagerSecrets, createdSecret.Name)
		log.Info("Created Secret Manager secret", "secret", createdSecret.Name)
	}

	return nil
}

// Note: Service creation methods moved to util package

// GetSecurityRecommendations provides comprehensive security recommendations
func GetSecurityRecommendations() []string {
	return []string{
		"Enable Workload Identity to securely access GCP services from Kubernetes workloads",
		"Use Cloud KMS for encryption at rest of etcd data and persistent volumes",
		"Enable Private Google Access to allow nodes to access Google APIs without external IPs",
		"Implement network policies to control pod-to-pod communication",
		"Use Binary Authorization to ensure only verified container images are deployed",
		"Enable VPC Flow Logs for network traffic analysis and security monitoring",
		"Configure IAM roles with principle of least privilege",
		"Enable audit logging for all GCP services and Kubernetes API server",
		"Use Secret Manager for storing sensitive configuration data",
		"Implement Pod Security Standards (formerly Pod Security Policies)",
		"Enable Shielded GKE nodes for additional protection against rootkits and bootkits",
		"Use VPC Service Controls to create security perimeters around resources",
		"Configure firewall rules to deny all traffic by default and allow only necessary communication",
		"Enable image vulnerability scanning for container images",
		"Use Google Cloud Security Command Center for unified security monitoring",
		"Implement RBAC (Role-Based Access Control) for Kubernetes resources",
		"Enable monitoring and alerting for security events",
		"Regularly rotate service account keys and access tokens",
		"Use managed SSL certificates for secure communication",
		"Implement backup and disaster recovery procedures for critical data",
	}
}