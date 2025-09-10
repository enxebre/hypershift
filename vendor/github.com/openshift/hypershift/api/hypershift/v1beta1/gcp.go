package v1beta1

//go:generate controller-gen object:headerFile="../../../hack/boilerplate.go.txt" paths=./gcp.go

// GCPNodePoolPlatform specifies the configuration of a NodePool when operating
// on Google Cloud Platform.
type GCPNodePoolPlatform struct {
	// machineType is the GCP machine type for node instances (e.g. n1-standard-4).
	//
	// +required
	// +kubebuilder:validation:MaxLength=255
	MachineType string `json:"machineType"`

	// subnetwork is the name of the subnet to use for the nodepool instances.
	// The subnet should be located in the same region as the cluster.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Subnetwork string `json:"subnetwork,omitempty"`

	// image is the boot image to use for the instances.
	// If unspecified, the default is chosen based on the NodePool release payload image.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Image string `json:"image,omitempty"`

	// diskType is the type of disk for the root volume (e.g. pd-ssd, pd-standard).
	//
	// +optional
	// +kubebuilder:default="pd-standard"
	// +kubebuilder:validation:Enum=pd-ssd;pd-standard;pd-balanced
	DiskType string `json:"diskType,omitempty"`

	// diskSizeGb is the size of the disk in GB.
	//
	// +optional
	// +kubebuilder:default=100
	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:Maximum=65536
	DiskSizeGb int64 `json:"diskSizeGb,omitempty"`

	// serviceAccount specifies the service account to use for the instances.
	//
	// +optional
	ServiceAccount *GCPServiceAccount `json:"serviceAccount,omitempty"`

	// tags are additional network tags to apply to the instances.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=64
	Tags []string `json:"tags,omitempty"`

	// labels are additional labels to apply to the instances.
	//
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	Labels map[string]string `json:"labels,omitempty"`

	// preemptible indicates if the instances should be preemptible.
	//
	// +optional
	Preemptible bool `json:"preemptible,omitempty"`

	// onHostMaintenance specifies what to do when Google Compute Engine schedules
	// a maintenance event on the VM.
	//
	// +optional
	// +kubebuilder:validation:Enum=MIGRATE;TERMINATE
	// +kubebuilder:default="MIGRATE"
	OnHostMaintenance string `json:"onHostMaintenance,omitempty"`
}

// GCPServiceAccount specifies the service account configuration for instances.
type GCPServiceAccount struct {
	// email is the email address of the service account.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Email string `json:"email,omitempty"`

	// scopes are the access scopes for the service account.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=50
	Scopes []string `json:"scopes,omitempty"`
}

// GCPPlatformSpec specifies configuration for clusters running on Google Cloud Platform.
type GCPPlatformSpec struct {
	// project is the GCP project ID where the cluster infrastructure will be created.
	//
	// +required
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Project is immutable"
	// +immutable
	Project string `json:"project"`

	// region is the GCP region where the cluster infrastructure will be created.
	//
	// +required
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Region is immutable"
	// +immutable
	Region string `json:"region"`

	// zone is the GCP zone within the region where the cluster control plane will be created.
	// If not specified, the zone will be automatically selected.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Zone string `json:"zone,omitempty"`

	// network specifies the GCP network configuration for the cluster.
	//
	// +optional
	Network *GCPNetworkSpec `json:"network,omitempty"`

	// serviceAccount specifies the service account configuration for the cluster.
	//
	// +optional
	ServiceAccount *GCPServiceAccount `json:"serviceAccount,omitempty"`

	// kmsKeyName is the name of the Cloud KMS key to use for etcd encryption.
	// The key must be in the same project and region as the cluster.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	KMSKeyName string `json:"kmsKeyName,omitempty"`

	// labels are additional labels to apply to GCP resources created for the cluster.
	//
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	Labels map[string]string `json:"labels,omitempty"`

	// resourceTags are additional tags to apply to GCP resources created for the cluster.
	//
	// +optional
	// +kubebuilder:validation:MaxProperties=50
	ResourceTags map[string]string `json:"resourceTags,omitempty"`
}

// GCPNetworkSpec specifies GCP network configuration.
type GCPNetworkSpec struct {
	// network is the name of the VPC network to use for the cluster.
	// If not specified, a new network will be created.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Network string `json:"network,omitempty"`

	// subnetwork is the name of the subnet to use for the cluster.
	// If not specified, a new subnet will be created.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Subnetwork string `json:"subnetwork,omitempty"`

	// controlPlaneSubnet is the subnet to use for the control plane instances.
	// If not specified, the main subnetwork will be used.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	ControlPlaneSubnet string `json:"controlPlaneSubnet,omitempty"`

	// loadBalancerSubnet is the subnet to use for load balancers.
	// If not specified, the main subnetwork will be used.
	//
	// +optional
	// +kubebuilder:validation:MaxLength=255
	LoadBalancerSubnet string `json:"loadBalancerSubnet,omitempty"`
}

// GCPPlatformStatus contains the status information specific to Google Cloud Platform.
type GCPPlatformStatus struct {
	// project is the GCP project ID where the cluster infrastructure is created.
	//
	// +optional
	Project string `json:"project,omitempty"`

	// region is the GCP region where the cluster infrastructure is created.
	//
	// +optional
	Region string `json:"region,omitempty"`

	// zone is the GCP zone where the cluster control plane is created.
	//
	// +optional
	Zone string `json:"zone,omitempty"`

	// networkName is the name of the VPC network used by the cluster.
	//
	// +optional
	NetworkName string `json:"networkName,omitempty"`

	// subnetworkName is the name of the subnet used by the cluster.
	//
	// +optional
	SubnetworkName string `json:"subnetworkName,omitempty"`

	// defaultMachineType is the default machine type used for worker nodes.
	//
	// +optional
	DefaultMachineType string `json:"defaultMachineType,omitempty"`
}