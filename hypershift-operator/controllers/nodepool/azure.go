package nodepool

import (
	"fmt"

	"github.com/blang/semver"
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/azureutil"
	"github.com/openshift/hypershift/support/releaseinfo"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	capiazure "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
)

// dummySSHKey is a base64 encoded dummy SSH public key.
// The CAPI AzureMachineTemplate requires an SSH key to be set, so we provide a dummy one here.
const dummySSHKey = "c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDTGFjOTR4dUE4QjkyMEtjejhKNjhUdmZCRjQyR2UwUllXSUx3Lzd6dDhUQlU5ell5Q0Q2K0ZlekFwWndLRjB1V3luMGVBQmlBWVdIV0tKbENxS0VIT2hOQmV2Mkx3S0dnZHFqM0dvcHV2N3RpZFVqSVpqYi9DVWtjQVRZUWhMWkxVTCs3eWkzRThKNHdhYkxEMWVNS1p1U3ZmMUsxT0RwVUFXYTkwbWVmR0FBOVdIVEhMcnF1UUpWdC9JT0JLN1ROZFNwMDVuM0Ywa29xZlE2empwRlFYMk8zaWJUc29yR3ZEekdhYS9yUENxQWhTSjRJaEhnMDNVb3FBbVlraW51NTFvVEcxRlRXaTh2b00vRVJ4TlduamNUSElET1JmYmo2bFVyZ3Zkci9MZGtqc2dFcENiNEMxUS9IbW5MRHVpTEdPM2tNZ2cyOHFzZ0ZmTHloUjl3ay8K"

// minVersionForMarketplaceDefaulting is the minimum OCP version that supports Azure Marketplace image defaulting.
var minVersionForMarketplaceDefaulting = semver.Version{Major: 4, Minor: 20, Patch: 0}

// defaultAzureMarketplaceImageFromRelease returns the default Azure Marketplace image from the release payload.
// It applies version gating (>= 4.20) and defaults to Gen2 if imageGeneration is not specified.
func defaultAzureMarketplaceImageFromRelease(nodePool *hyperv1.NodePool, releaseImage *releaseinfo.ReleaseImage, arch string) (*hyperv1.AzureMarketplaceImage, error) {
	// Parse release version
	releaseVersion, err := semver.Parse(releaseImage.Version())
	if err != nil {
		return nil, fmt.Errorf("failed to parse release version %q: %w", releaseImage.Version(), err)
	}

	// Version gating: only apply marketplace defaulting for OCP >= 4.20
	if releaseVersion.LT(minVersionForMarketplaceDefaulting) {
		return nil, fmt.Errorf("Azure Marketplace image defaulting is only supported for OCP >= 4.20, current version is %s", releaseVersion.String())
	}

	// Validate StreamMetadata is present
	if releaseImage.StreamMetadata == nil {
		return nil, fmt.Errorf("release image %q has no stream metadata", releaseImage.Version())
	}

	// Get architecture metadata
	archMeta, found := releaseImage.StreamMetadata.Architectures[arch]
	if !found {
		return nil, fmt.Errorf("couldn't find OS metadata for architecture %q", arch)
	}

	// Default to Gen2 if imageGeneration is not specified
	imageGen := hyperv1.AzureVMImageGenerationV2
	if nodePool.Spec.Platform.Azure.ImageGeneration != nil {
		imageGen = *nodePool.Spec.Platform.Azure.ImageGeneration
	}

	// Get the appropriate marketplace image based on generation
	var marketplaceImage *releaseinfo.CoreAzureMarketplaceImage
	switch imageGen {
	case hyperv1.AzureVMImageGenerationV1:
		marketplaceImage = archMeta.RHCOS.AzureMarketplace.Azure.NoPurchasePlan.HyperVGen1
		if marketplaceImage == nil {
			return nil, fmt.Errorf("no Azure Marketplace Gen1 image found in release payload for architecture %q", arch)
		}
	case hyperv1.AzureVMImageGenerationV2:
		marketplaceImage = archMeta.RHCOS.AzureMarketplace.Azure.NoPurchasePlan.HyperVGen2
		if marketplaceImage == nil {
			return nil, fmt.Errorf("no Azure Marketplace Gen2 image found in release payload for architecture %q", arch)
		}
	default:
		return nil, fmt.Errorf("unsupported image generation %q", imageGen)
	}

	return &hyperv1.AzureMarketplaceImage{
		Publisher: marketplaceImage.Publisher,
		Offer:     marketplaceImage.Offer,
		SKU:       marketplaceImage.SKU,
		Version:   marketplaceImage.Version,
	}, nil
}

func azureMachineTemplateSpec(nodePool *hyperv1.NodePool, releaseImage *releaseinfo.ReleaseImage) (*capiazure.AzureMachineTemplateSpec, error) {
	subnetName, err := azureutil.GetSubnetNameFromSubnetID(nodePool.Spec.Platform.Azure.SubnetID)
	if err != nil {
		return nil, fmt.Errorf("failed to determine subnet name for Azure machine: %w", err)
	}

	// Handle image defaulting: if no explicit image is provided, try to default from release payload
	image := nodePool.Spec.Platform.Azure.Image
	if image.ImageID == nil && image.AzureMarketplace == nil {
		// Attempt to default from release payload
		// TODO(CNTRLPLANE-475): Determine architecture from NodePool or HostedCluster spec.
		// Currently hardcoded to x86_64. ARM64 nodepools will fail if they rely on automatic defaulting.
		// For now, ARM64 users must explicitly specify Image.AzureMarketplace.
		arch := "x86_64"
		defaultMarketplaceImage, err := defaultAzureMarketplaceImageFromRelease(nodePool, releaseImage, arch)
		if err != nil {
			return nil, fmt.Errorf("failed to default Azure Marketplace image from release: %w", err)
		}
		image = hyperv1.AzureVMImage{
			Type:             hyperv1.AzureMarketplace,
			AzureMarketplace: defaultMarketplaceImage,
		}
	}

	azureMachineTemplate := &capiazure.AzureMachineTemplateSpec{Template: capiazure.AzureMachineTemplateResource{Spec: capiazure.AzureMachineSpec{
		VMSize: nodePool.Spec.Platform.Azure.VMSize,
		OSDisk: capiazure.OSDisk{
			DiskSizeGB: ptr.To(nodePool.Spec.Platform.Azure.OSDisk.SizeGiB),
			ManagedDisk: &capiazure.ManagedDiskParameters{
				StorageAccountType: string(nodePool.Spec.Platform.Azure.OSDisk.DiskStorageAccountType),
			},
		},
		NetworkInterfaces: []capiazure.NetworkInterface{{
			SubnetName: subnetName,
		}},
		FailureDomain: failureDomain(nodePool),
	}}}

	switch image.Type {
	case hyperv1.ImageID:
		azureMachineTemplate.Template.Spec.Image = &capiazure.Image{
			ID: image.ImageID,
		}
	case hyperv1.AzureMarketplace:
		azureMachineTemplate.Template.Spec.Image = &capiazure.Image{
			Marketplace: &capiazure.AzureMarketplaceImage{
				ImagePlan: capiazure.ImagePlan{
					Publisher: image.AzureMarketplace.Publisher,
					Offer:     image.AzureMarketplace.Offer,
					SKU:       image.AzureMarketplace.SKU,
				},
				Version: image.AzureMarketplace.Version,
			},
		}
	}

	if nodePool.Spec.Platform.Azure.OSDisk.EncryptionSetID != "" {
		azureMachineTemplate.Template.Spec.OSDisk.ManagedDisk.DiskEncryptionSet = &capiazure.DiskEncryptionSetParameters{
			ID: nodePool.Spec.Platform.Azure.OSDisk.EncryptionSetID,
		}
	}

	if nodePool.Spec.Platform.Azure.EncryptionAtHost == "Enabled" {
		azureMachineTemplate.Template.Spec.SecurityProfile = &capiazure.SecurityProfile{
			EncryptionAtHost: to.Ptr(true),
		}
	}

	if nodePool.Spec.Platform.Azure.OSDisk.Persistence == hyperv1.EphemeralDiskPersistence {
		// This is set to "None" if not explicitly set - https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/f44d953844de58e4b6fe8f51d88b0bf75a04e9ec/api/v1beta1/azuremachine_default.go#L54
		// "VMs and VM Scale Set Instances using an ephemeral OS disk support only Readonly caching."
		azureMachineTemplate.Template.Spec.OSDisk.CachingType = "ReadOnly"
		azureMachineTemplate.Template.Spec.OSDisk.DiffDiskSettings = &capiazure.DiffDiskSettings{Option: "Local"}
	}

	if nodePool.Spec.Platform.Azure.Diagnostics != nil && nodePool.Spec.Platform.Azure.Diagnostics.StorageAccountType != "" {
		azureMachineTemplate.Template.Spec.Diagnostics = &capiazure.Diagnostics{
			Boot: &capiazure.BootDiagnostics{
				StorageAccountType: capiazure.BootDiagnosticsStorageAccountType(nodePool.Spec.Platform.Azure.Diagnostics.StorageAccountType),
			},
		}
		if nodePool.Spec.Platform.Azure.Diagnostics.StorageAccountType == "UserManaged" {
			azureMachineTemplate.Template.Spec.Diagnostics.Boot.UserManaged = &capiazure.UserManagedBootDiagnostics{
				StorageAccountURI: nodePool.Spec.Platform.Azure.Diagnostics.UserManaged.StorageAccountURI,
			}
		}
	}

	azureMachineTemplate.Template.Spec.SSHPublicKey = dummySSHKey

	return azureMachineTemplate, nil
}

func (c *CAPI) azureMachineTemplate(templateNameGenerator func(spec any) (string, error)) (*capiazure.AzureMachineTemplate, error) {
	spec, err := azureMachineTemplateSpec(c.nodePool, c.releaseImage)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AzureMachineTemplateSpec: %w", err)
	}

	templateName, err := templateNameGenerator(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template name: %w", err)
	}

	template := &capiazure.AzureMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
		Spec: *spec,
	}

	return template, nil
}

func failureDomain(nodepool *hyperv1.NodePool) *string {
	if nodepool.Spec.Platform.Azure.AvailabilityZone == "" {
		return nil
	}
	return ptr.To(fmt.Sprintf("%v", nodepool.Spec.Platform.Azure.AvailabilityZone))
}
