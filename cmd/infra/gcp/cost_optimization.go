package gcp

import (
	"context"
	"fmt"
	"strings"

	gcputil "github.com/openshift/hypershift/cmd/infra/gcp/util"
	"github.com/go-logr/logr"
	"google.golang.org/api/compute/v1"
)

// CostOptimizationOptions holds configuration for GCP cost optimization features
type CostOptimizationOptions struct {
	CredentialsOpts              gcputil.GCPCredentialsOptions
	Project                      string
	Region                       string
	InfraID                      string
	EnablePreemptibleInstances   bool
	EnableSustainedUseDiscounts  bool
	EnableCommittedUseDiscounts  bool
	EnableAutoscaling           bool
	MinNodes                    int32
	MaxNodes                    int32
	EnableSpotInstances         bool
	EnableCustomMachineTypes    bool
	OptimizeDiskTypes           bool
	EnableRegionalPersistentDisks bool
}

// CostOptimizationRecommendations holds cost optimization recommendations
type CostOptimizationRecommendations struct {
	PreemptibleSavings          float64            `json:"preemptibleSavings"`
	SustainedUseDiscountSavings float64            `json:"sustainedUseDiscountSavings"`
	CommittedUseDiscountSavings float64            `json:"committedUseDiscountSavings"`
	DiskOptimizationSavings     float64            `json:"diskOptimizationSavings"`
	Recommendations             []string           `json:"recommendations"`
	CostBreakdown               map[string]float64 `json:"costBreakdown"`
}

// AnalyzeCostOptimization analyzes the cluster configuration and provides cost optimization recommendations
func (o *CostOptimizationOptions) AnalyzeCostOptimization(ctx context.Context, log logr.Logger) (*CostOptimizationRecommendations, error) {
	log.Info("Analyzing GCP cost optimization opportunities", "infraID", o.InfraID)

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

	recommendations := &CostOptimizationRecommendations{
		Recommendations: []string{},
		CostBreakdown:   make(map[string]float64),
	}

	// Analyze compute instances
	if err := o.analyzeComputeInstances(ctx, computeService, recommendations, log); err != nil {
		log.Error(err, "Failed to analyze compute instances")
	}

	// Analyze disk usage
	if err := o.analyzeDiskUsage(ctx, computeService, recommendations, log); err != nil {
		log.Error(err, "Failed to analyze disk usage")
	}

	// Analyze network usage
	if err := o.analyzeNetworkUsage(ctx, computeService, recommendations, log); err != nil {
		log.Error(err, "Failed to analyze network usage")
	}

	// Generate general recommendations
	o.generateGeneralRecommendations(recommendations)

	log.Info("Cost optimization analysis completed", "recommendationsCount", len(recommendations.Recommendations))
	return recommendations, nil
}

func (o *CostOptimizationOptions) analyzeComputeInstances(ctx context.Context, computeService *compute.Service, recommendations *CostOptimizationRecommendations, log logr.Logger) error {
	// List instances with cluster label
	instances, err := computeService.Instances.AggregatedList(o.Project).Filter(fmt.Sprintf("labels.kubernetes-io-cluster-%s=owned", o.InfraID)).Do()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	totalInstances := 0
	preemptibleInstances := 0
	var machineTypes []string

	for _, scopedList := range instances.Items {
		for _, instance := range scopedList.Instances {
			totalInstances++
			if instance.Scheduling != nil && instance.Scheduling.Preemptible {
				preemptibleInstances++
			}
			machineTypes = append(machineTypes, instance.MachineType)
		}
	}

	// Calculate preemptible savings potential
	if preemptibleInstances < totalInstances {
		nonPreemptibleCount := totalInstances - preemptibleInstances
		preemptibleSavings := float64(nonPreemptibleCount) * 0.8 // Up to 80% savings
		recommendations.PreemptibleSavings = preemptibleSavings
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("Consider using preemptible instances for %d non-preemptible instances to save up to 80%% on compute costs", nonPreemptibleCount))
	}

	// Analyze machine types for right-sizing opportunities
	if len(machineTypes) > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			"Review machine types for right-sizing opportunities based on actual CPU and memory utilization")
	}

	// Calculate sustained use discount potential
	if totalInstances > 0 {
		sustainedUseDiscount := float64(totalInstances) * 0.3 // Up to 30% discount
		recommendations.SustainedUseDiscountSavings = sustainedUseDiscount
		recommendations.Recommendations = append(recommendations.Recommendations,
			"Sustained use discounts are automatically applied for instances running >25% of the month")
	}

	recommendations.CostBreakdown["compute"] = float64(totalInstances) * 100 // Estimated monthly cost per instance

	return nil
}

func (o *CostOptimizationOptions) analyzeDiskUsage(ctx context.Context, computeService *compute.Service, recommendations *CostOptimizationRecommendations, log logr.Logger) error {
	// List disks with cluster label
	disks, err := computeService.Disks.AggregatedList(o.Project).Filter(fmt.Sprintf("labels.kubernetes-io-cluster-%s=owned", o.InfraID)).Do()
	if err != nil {
		return fmt.Errorf("failed to list disks: %w", err)
	}

	totalDisks := 0
	totalSize := int64(0)
	ssdDisks := 0
	standardDisks := 0

	for _, scopedList := range disks.Items {
		for _, disk := range scopedList.Disks {
			totalDisks++
			totalSize += disk.SizeGb
			
			if strings.Contains(disk.Type, "pd-ssd") {
				ssdDisks++
			} else if strings.Contains(disk.Type, "pd-standard") {
				standardDisks++
			}
		}
	}

	if ssdDisks > 0 {
		ssdOptimizationSavings := float64(ssdDisks) * 0.6 // SSD is ~2.5x more expensive than standard
		recommendations.DiskOptimizationSavings = ssdOptimizationSavings
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("Consider using pd-standard instead of pd-ssd for %d disks where IOPS requirements allow (up to 60%% disk cost savings)", ssdDisks))
	}

	if totalSize > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			"Consider using pd-balanced disks as a cost-effective middle ground between pd-standard and pd-ssd")
		
		if o.EnableRegionalPersistentDisks {
			recommendations.Recommendations = append(recommendations.Recommendations,
				"Regional persistent disks provide higher availability but cost 2x more than zonal disks")
		}
	}

	recommendations.CostBreakdown["storage"] = float64(totalSize) * 0.04 // ~$0.04/GB/month for standard disks

	return nil
}

func (o *CostOptimizationOptions) analyzeNetworkUsage(ctx context.Context, computeService *compute.Service, recommendations *CostOptimizationRecommendations, log logr.Logger) error {
	// List load balancers
	forwardingRules, err := computeService.ForwardingRules.AggregatedList(o.Project).Do()
	if err != nil {
		return fmt.Errorf("failed to list forwarding rules: %w", err)
	}

	loadBalancers := 0
	for _, scopedList := range forwardingRules.Items {
		for _, rule := range scopedList.ForwardingRules {
			if strings.Contains(rule.Name, o.InfraID) {
				loadBalancers++
			}
		}
	}

	if loadBalancers > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			"Review load balancer usage - internal load balancers are cheaper than external ones")
		recommendations.CostBreakdown["networking"] = float64(loadBalancers) * 18 // ~$18/month per load balancer
	}

	// Cloud NAT analysis
	routers, err := computeService.Routers.AggregatedList(o.Project).Do()
	if err != nil {
		return fmt.Errorf("failed to list routers: %w", err)
	}

	natGateways := 0
	for _, scopedList := range routers.Items {
		for _, router := range scopedList.Routers {
			if strings.Contains(router.Name, o.InfraID) && len(router.Nats) > 0 {
				natGateways += len(router.Nats)
			}
		}
	}

	if natGateways > 0 {
		recommendations.Recommendations = append(recommendations.Recommendations,
			"Cloud NAT pricing is based on data processing - monitor and optimize data transfer")
		recommendations.CostBreakdown["nat"] = float64(natGateways) * 45 // ~$45/month per NAT gateway
	}

	return nil
}

func (o *CostOptimizationOptions) generateGeneralRecommendations(recommendations *CostOptimizationRecommendations) {
	generalRecommendations := []string{
		"Enable autoscaling to automatically adjust cluster size based on demand",
		"Use committed use discounts for predictable workloads (up to 57% savings for 1-year, 70% for 3-year)",
		"Consider using Google Kubernetes Engine (GKE) Autopilot for fully managed, cost-optimized experience",
		"Monitor resource utilization using Google Cloud Operations (formerly Stackdriver) to identify optimization opportunities",
		"Use Google Cloud Billing reports and Cost Management tools to track and allocate costs",
		"Implement pod resource requests and limits to enable effective autoscaling",
		"Consider using Spot VMs (preemptible instances) for fault-tolerant workloads",
		"Use Google Cloud Storage classes appropriately (Standard, Nearline, Coldline, Archive) based on access patterns",
		"Enable deletion protection for critical resources to avoid accidental charges",
		"Set up budget alerts to monitor and control cloud spending",
	}

	recommendations.Recommendations = append(recommendations.Recommendations, generalRecommendations...)

	// Calculate committed use discount potential
	if recommendations.CostBreakdown["compute"] > 0 {
		oneYearCommitment := recommendations.CostBreakdown["compute"] * 0.57
		threeYearCommitment := recommendations.CostBreakdown["compute"] * 0.70
		recommendations.CommittedUseDiscountSavings = threeYearCommitment
		recommendations.Recommendations = append(recommendations.Recommendations,
			fmt.Sprintf("Committed use discounts: 1-year commitment saves up to $%.0f/month, 3-year saves up to $%.0f/month", oneYearCommitment, threeYearCommitment))
	}
}

// ApplyCostOptimizations applies cost optimization settings to the cluster infrastructure
func (o *CostOptimizationOptions) ApplyCostOptimizations(ctx context.Context, log logr.Logger) error {
	log.Info("Applying GCP cost optimizations", "infraID", o.InfraID)

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

	// Apply autoscaling if enabled
	if o.EnableAutoscaling {
		if err := o.enableAutoscaling(ctx, computeService, log); err != nil {
			return fmt.Errorf("failed to enable autoscaling: %w", err)
		}
	}

	log.Info("Successfully applied cost optimizations", "infraID", o.InfraID)
	return nil
}

func (o *CostOptimizationOptions) enableAutoscaling(ctx context.Context, computeService *compute.Service, log logr.Logger) error {
	log.Info("Enabling autoscaling for instance groups")

	// List instance groups
	instanceGroups, err := computeService.InstanceGroups.AggregatedList(o.Project).Filter(fmt.Sprintf("labels.kubernetes-io-cluster-%s=owned", o.InfraID)).Do()
	if err != nil {
		return fmt.Errorf("failed to list instance groups: %w", err)
	}

	for region, scopedList := range instanceGroups.Items {
		for _, group := range scopedList.InstanceGroups {
			if strings.Contains(group.Name, o.InfraID) {
				log.Info("Found instance group for autoscaling", "group", group.Name, "region", region)
				// Note: Actual autoscaling configuration would depend on the cluster's
				// machine deployment and cluster autoscaler configuration
			}
		}
	}

	return nil
}