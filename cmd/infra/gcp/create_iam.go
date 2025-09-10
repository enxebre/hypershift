package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"
	"github.com/openshift/hypershift/cmd/log"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
)

type CreateIAMOptions struct {
	CredentialsOpts gcputil.GCPCredentialsOptions
	Project         string
	InfraID         string
	OutputFile      string
}

type CreateIAMOutput struct {
	ServiceAccountEmail string `json:"serviceAccountEmail"`
	ServiceAccountKey   string `json:"serviceAccountKey,omitempty"`
	WorkerPoolRoles     []string `json:"workerPoolRoles"`
	ControlPlaneRoles   []string `json:"controlPlaneRoles"`
}

func NewCreateIAMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "iam",
		Short:        "Creates GCP IAM resources for a cluster",
		SilenceUsage: true,
	}

	opts := CreateIAMOptions{}

	cmd.Flags().StringVar(&opts.Project, "project", opts.Project, "GCP project ID")
	cmd.Flags().StringVar(&opts.InfraID, "infra-id", opts.InfraID, "Cluster ID (required)")
	cmd.Flags().StringVar(&opts.OutputFile, "output-file", opts.OutputFile, "Path to file that will contain output information from IAM resources (optional)")

	opts.CredentialsOpts.BindFlags(cmd.Flags())

	_ = cmd.MarkFlagRequired("infra-id")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		_, err := opts.CreateIAM(ctx, log.Log)
		return err
	}

	return cmd
}

func (o *CreateIAMOptions) CreateIAM(ctx context.Context, log logr.Logger) (*CreateIAMOutput, error) {
	// Get the project ID
	projectID, err := o.CredentialsOpts.GetProjectID()
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}
	if o.Project == "" {
		o.Project = projectID
	}

	// Create IAM service
	iamService, err := o.CredentialsOpts.CreateIAMService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM service: %w", err)
	}

	// Create Resource Manager service for IAM policy binding
	rmService, err := o.CredentialsOpts.CreateResourceManagerService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager service: %w", err)
	}

	output := &CreateIAMOutput{}

	// Create service account for the cluster
	log.Info("Creating cluster service account")
	serviceAccountEmail, err := o.createServiceAccount(ctx, iamService, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}
	output.ServiceAccountEmail = serviceAccountEmail

	// Assign necessary roles to the service account
	log.Info("Assigning roles to service account")
	controlPlaneRoles, workerPoolRoles, err := o.assignRoles(ctx, rmService, serviceAccountEmail, log)
	if err != nil {
		return nil, fmt.Errorf("failed to assign roles: %w", err)
	}
	output.ControlPlaneRoles = controlPlaneRoles
	output.WorkerPoolRoles = workerPoolRoles

	// Output results
	if o.OutputFile != "" {
		if err := o.writeOutput(output); err != nil {
			return nil, fmt.Errorf("failed to write output file: %w", err)
		}
	}

	log.Info("Successfully created GCP IAM resources", "serviceAccount", serviceAccountEmail)
	return output, nil
}

func (o *CreateIAMOptions) createServiceAccount(ctx context.Context, iamService *iam.Service, log logr.Logger) (string, error) {
	accountID := o.InfraID + "-cluster-sa"
	displayName := fmt.Sprintf("HyperShift cluster %s service account", o.InfraID)

	serviceAccount := &iam.ServiceAccount{
		DisplayName: displayName,
		Description: fmt.Sprintf("Service account for HyperShift cluster %s", o.InfraID),
	}

	request := &iam.CreateServiceAccountRequest{
		AccountId:      accountID,
		ServiceAccount: serviceAccount,
	}

	projectPath := "projects/" + o.Project
	createdSA, err := iamService.Projects.ServiceAccounts.Create(projectPath, request).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create service account: %w", err)
	}

	log.Info("Created service account", "email", createdSA.Email)
	return createdSA.Email, nil
}

func (o *CreateIAMOptions) assignRoles(ctx context.Context, rmService *cloudresourcemanager.Service, serviceAccountEmail string, log logr.Logger) ([]string, []string, error) {
	// Define roles needed for control plane
	controlPlaneRoles := []string{
		"roles/compute.admin",
		"roles/storage.admin",
		"roles/dns.admin",
		"roles/iam.serviceAccountUser",
		"roles/cloudsql.admin",
		"roles/logging.admin",
		"roles/monitoring.admin",
	}

	// Define roles needed for worker pools
	workerPoolRoles := []string{
		"roles/compute.instanceAdmin",
		"roles/storage.objectViewer",
		"roles/logging.logWriter",
		"roles/monitoring.metricWriter",
	}

	// Get current IAM policy
	policy, err := rmService.Projects.GetIamPolicy(o.Project, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get IAM policy: %w", err)
	}

	// Create member string for the service account
	member := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)

	// Add bindings for control plane roles
	for _, role := range controlPlaneRoles {
		log.Info("Adding control plane role binding", "role", role)
		policy = addRoleBinding(policy, role, member)
	}

	// Add bindings for worker pool roles
	for _, role := range workerPoolRoles {
		log.Info("Adding worker pool role binding", "role", role)
		policy = addRoleBinding(policy, role, member)
	}

	// Set the updated policy
	setRequest := &cloudresourcemanager.SetIamPolicyRequest{
		Policy: policy,
	}

	_, err = rmService.Projects.SetIamPolicy(o.Project, setRequest).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set IAM policy: %w", err)
	}

	return controlPlaneRoles, workerPoolRoles, nil
}

func addRoleBinding(policy *cloudresourcemanager.Policy, role, member string) *cloudresourcemanager.Policy {
	// Check if binding already exists
	for i, binding := range policy.Bindings {
		if binding.Role == role {
			// Check if member is already in the binding
			for _, existingMember := range binding.Members {
				if existingMember == member {
					return policy // Member already has this role
				}
			}
			// Add member to existing binding
			policy.Bindings[i].Members = append(binding.Members, member)
			return policy
		}
	}

	// Create new binding
	newBinding := &cloudresourcemanager.Binding{
		Role:    role,
		Members: []string{member},
	}
	policy.Bindings = append(policy.Bindings, newBinding)
	return policy
}

func (o *CreateIAMOptions) writeOutput(output *CreateIAMOutput) error {
	outputBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	if err := os.WriteFile(o.OutputFile, outputBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}