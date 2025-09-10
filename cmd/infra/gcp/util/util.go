package util

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"google.golang.org/api/option"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/monitoring/v1"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/secretmanager/v1"
)

// GCPCredentialsOptions holds the credentials configuration for GCP
type GCPCredentialsOptions struct {
	CredentialsFile string
	ProjectID       string
}

// BindFlags binds command line flags for GCP credentials
func (o *GCPCredentialsOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.CredentialsFile, "gcp-creds", o.CredentialsFile, "Path to GCP service account key JSON file (required)")
	flags.StringVar(&o.ProjectID, "project-id", o.ProjectID, "GCP project ID (can be inferred from credentials)")
}

// LoadCredentials loads GCP credentials and returns the data for use in secrets
func (o *GCPCredentialsOptions) LoadCredentials() (map[string][]byte, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file must be specified")
	}

	credentialsData, err := os.ReadFile(o.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read GCP credentials file %s: %w", o.CredentialsFile, err)
	}

	// Validate that it's valid JSON
	var creds map[string]interface{}
	if err := json.Unmarshal(credentialsData, &creds); err != nil {
		return nil, fmt.Errorf("invalid JSON in GCP credentials file: %w", err)
	}

	// Extract project ID if not provided
	if o.ProjectID == "" {
		if projectID, ok := creds["project_id"].(string); ok && projectID != "" {
			o.ProjectID = projectID
		}
	}

	return map[string][]byte{
		"service_account.json": credentialsData,
	}, nil
}

// GetProjectID returns the project ID, either from the flag or from the credentials
func (o *GCPCredentialsOptions) GetProjectID() (string, error) {
	if o.ProjectID != "" {
		return o.ProjectID, nil
	}

	if o.CredentialsFile == "" {
		return "", fmt.Errorf("either project-id flag or credentials file must be provided")
	}

	credentialsData, err := os.ReadFile(o.CredentialsFile)
	if err != nil {
		return "", fmt.Errorf("failed to read GCP credentials file: %w", err)
	}

	var creds struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(credentialsData, &creds); err != nil {
		return "", fmt.Errorf("failed to parse credentials JSON: %w", err)
	}

	if creds.ProjectID == "" {
		return "", fmt.Errorf("project_id not found in credentials file")
	}

	return creds.ProjectID, nil
}

// CreateComputeService creates a GCP Compute Engine service
func (o *GCPCredentialsOptions) CreateComputeService(ctx context.Context) (*compute.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := compute.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Compute service: %w", err)
	}

	return service, nil
}

// CreateResourceManagerService creates a GCP Resource Manager service
func (o *GCPCredentialsOptions) CreateResourceManagerService(ctx context.Context) (*cloudresourcemanager.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := cloudresourcemanager.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Resource Manager service: %w", err)
	}

	return service, nil
}

// CreateIAMService creates a GCP IAM service
func (o *GCPCredentialsOptions) CreateIAMService(ctx context.Context) (*iam.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := iam.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP IAM service: %w", err)
	}

	return service, nil
}

// CreateDNSService creates a GCP DNS service
func (o *GCPCredentialsOptions) CreateDNSService(ctx context.Context) (*dns.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := dns.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP DNS service: %w", err)
	}

	return service, nil
}

// GenerateInfraID generates a unique infrastructure ID for the cluster
func GenerateInfraID(name string) string {
	return fmt.Sprintf("%s", name)
}

// GetAvailabilityZones returns a list of available zones in the given region
func GetAvailabilityZones(ctx context.Context, computeService *compute.Service, project, region string) ([]string, error) {
	zonesCall := computeService.Zones.List(project)
	zonesCall = zonesCall.Filter(fmt.Sprintf("region eq %s", region))
	
	zones, err := zonesCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones in region %s: %w", region, err)
	}

	var zoneNames []string
	for _, zone := range zones.Items {
		if zone.Status == "UP" {
			zoneNames = append(zoneNames, zone.Name)
		}
	}

	return zoneNames, nil
}

// ValidateRegion checks if the given region exists and is available
func ValidateRegion(ctx context.Context, computeService *compute.Service, project, region string) error {
	regionsCall := computeService.Regions.Get(project, region)
	
	regionInfo, err := regionsCall.Do()
	if err != nil {
		return fmt.Errorf("failed to validate region %s: %w", region, err)
	}

	if regionInfo.Status != "UP" {
		return fmt.Errorf("region %s is not available (status: %s)", region, regionInfo.Status)
	}

	return nil
}

// ValidateProject checks if the project exists and is accessible
func ValidateProject(ctx context.Context, rmService *cloudresourcemanager.Service, projectID string) error {
	projectCall := rmService.Projects.Get(projectID)
	
	project, err := projectCall.Do()
	if err != nil {
		return fmt.Errorf("failed to validate project %s: %w", projectID, err)
	}

	if project.LifecycleState != "ACTIVE" {
		return fmt.Errorf("project %s is not active (state: %s)", projectID, project.LifecycleState)
	}

	return nil
}

// CreateLoggingService creates a GCP Logging service
func (o *GCPCredentialsOptions) CreateLoggingService(ctx context.Context) (*logging.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := logging.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Logging service: %w", err)
	}

	return service, nil
}

// CreateMonitoringService creates a GCP Monitoring service
func (o *GCPCredentialsOptions) CreateMonitoringService(ctx context.Context) (*monitoring.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := monitoring.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Monitoring service: %w", err)
	}

	return service, nil
}
// CreateKMSService creates a GCP KMS service
func (o *GCPCredentialsOptions) CreateKMSService(ctx context.Context) (*cloudkms.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := cloudkms.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP KMS service: %w", err)
	}

	return service, nil
}

// CreateSecretManagerService creates a GCP Secret Manager service
func (o *GCPCredentialsOptions) CreateSecretManagerService(ctx context.Context) (*secretmanager.Service, error) {
	if o.CredentialsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is required")
	}

	service, err := secretmanager.NewService(ctx, option.WithCredentialsFile(o.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Secret Manager service: %w", err)
	}

	return service, nil
}
