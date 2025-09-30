package nodepool

import (
	"testing"

	. "github.com/onsi/gomega"
	imageapi "github.com/openshift/api/image/v1"
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/releaseinfo"
	"k8s.io/utils/ptr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultAzureMarketplaceImageFromRelease(t *testing.T) {
	testCases := []struct {
		name                  string
		nodePool              *hyperv1.NodePool
		releaseImage          *releaseinfo.ReleaseImage
		arch                  string
		expectedMarketplace   *hyperv1.AzureMarketplaceImage
		expectedErr           bool
		expectedErrSubstring  string
	}{
		{
			name: "OCP 4.20 with Gen2 default (imageGeneration not specified)",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{
							// ImageGeneration is nil, should default to V2
						},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{
						"x86_64": {
							RHCOS: releaseinfo.CoreRHCOSImage{
								AzureMarketplace: releaseinfo.CoreAzureMarketplace{
									Azure: releaseinfo.CoreAzureMarketplaceChannels{
										NoPurchasePlan: releaseinfo.CoreAzureMarketplacePlan{
											HyperVGen2: &releaseinfo.CoreAzureMarketplaceImage{
												Publisher: "azureopenshift",
												Offer:     "aro4",
												SKU:       "aro_420_rhel_93_gen2",
												Version:   "420.93.202501010000",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			arch: "x86_64",
			expectedMarketplace: &hyperv1.AzureMarketplaceImage{
				Publisher: "azureopenshift",
				Offer:     "aro4",
				SKU:       "aro_420_rhel_93_gen2",
				Version:   "420.93.202501010000",
			},
			expectedErr: false,
		},
		{
			name: "OCP 4.20 with explicit Gen1",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{
							ImageGeneration: ptr.To(hyperv1.AzureVMImageGenerationV1),
						},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{
						"x86_64": {
							RHCOS: releaseinfo.CoreRHCOSImage{
								AzureMarketplace: releaseinfo.CoreAzureMarketplace{
									Azure: releaseinfo.CoreAzureMarketplaceChannels{
										NoPurchasePlan: releaseinfo.CoreAzureMarketplacePlan{
											HyperVGen1: &releaseinfo.CoreAzureMarketplaceImage{
												Publisher: "azureopenshift",
												Offer:     "aro4",
												SKU:       "aro_420_rhel_93_gen1",
												Version:   "420.93.202501010000",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			arch: "x86_64",
			expectedMarketplace: &hyperv1.AzureMarketplaceImage{
				Publisher: "azureopenshift",
				Offer:     "aro4",
				SKU:       "aro_420_rhel_93_gen1",
				Version:   "420.93.202501010000",
			},
			expectedErr: false,
		},
		{
			name: "OCP 4.20 with explicit Gen2",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{
							ImageGeneration: ptr.To(hyperv1.AzureVMImageGenerationV2),
						},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{
						"x86_64": {
							RHCOS: releaseinfo.CoreRHCOSImage{
								AzureMarketplace: releaseinfo.CoreAzureMarketplace{
									Azure: releaseinfo.CoreAzureMarketplaceChannels{
										NoPurchasePlan: releaseinfo.CoreAzureMarketplacePlan{
											HyperVGen2: &releaseinfo.CoreAzureMarketplaceImage{
												Publisher: "azureopenshift",
												Offer:     "aro4",
												SKU:       "aro_420_rhel_93_gen2",
												Version:   "420.93.202501010000",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			arch: "x86_64",
			expectedMarketplace: &hyperv1.AzureMarketplaceImage{
				Publisher: "azureopenshift",
				Offer:     "aro4",
				SKU:       "aro_420_rhel_93_gen2",
				Version:   "420.93.202501010000",
			},
			expectedErr: false,
		},
		{
			name: "OCP 4.19 - version gating prevents defaulting",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.19.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{
						"x86_64": {
							RHCOS: releaseinfo.CoreRHCOSImage{},
						},
					},
				},
			},
			arch:                 "x86_64",
			expectedErr:          true,
			expectedErrSubstring: "Azure Marketplace image defaulting is only supported for OCP >= 4.20",
		},
		{
			name: "Missing architecture metadata",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{},
				},
			},
			arch:                 "arm64",
			expectedErr:          true,
			expectedErrSubstring: "couldn't find OS metadata for architecture",
		},
		{
			name: "Missing Gen2 marketplace image in release",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{
							// Defaults to Gen2
						},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{
						"x86_64": {
							RHCOS: releaseinfo.CoreRHCOSImage{
								AzureMarketplace: releaseinfo.CoreAzureMarketplace{
									Azure: releaseinfo.CoreAzureMarketplaceChannels{
										NoPurchasePlan: releaseinfo.CoreAzureMarketplacePlan{
											// No Gen2 image
											HyperVGen2: nil,
										},
									},
								},
							},
						},
					},
				},
			},
			arch:                 "x86_64",
			expectedErr:          true,
			expectedErrSubstring: "no Azure Marketplace Gen2 image found in release payload",
		},
		{
			name: "Missing StreamMetadata in release",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: nil,
			},
			arch:                 "x86_64",
			expectedErr:          true,
			expectedErrSubstring: "has no stream metadata",
		},
		{
			name: "Missing Gen1 marketplace image in release when explicitly requested",
			nodePool: &hyperv1.NodePool{
				Spec: hyperv1.NodePoolSpec{
					Platform: hyperv1.NodePoolPlatform{
						Type: hyperv1.AzurePlatform,
						Azure: &hyperv1.AzureNodePoolPlatform{
							ImageGeneration: ptr.To(hyperv1.AzureVMImageGenerationV1),
						},
					},
				},
			},
			releaseImage: &releaseinfo.ReleaseImage{
				ImageStream: &imageapi.ImageStream{
					ObjectMeta: metav1.ObjectMeta{Name: "4.20.0"},
				},
				StreamMetadata: &releaseinfo.CoreOSStreamMetadata{
					Architectures: map[string]releaseinfo.CoreOSArchitecture{
						"x86_64": {
							RHCOS: releaseinfo.CoreRHCOSImage{
								AzureMarketplace: releaseinfo.CoreAzureMarketplace{
									Azure: releaseinfo.CoreAzureMarketplaceChannels{
										NoPurchasePlan: releaseinfo.CoreAzureMarketplacePlan{
											// No Gen1 image
											HyperVGen1: nil,
										},
									},
								},
							},
						},
					},
				},
			},
			arch:                 "x86_64",
			expectedErr:          true,
			expectedErrSubstring: "no Azure Marketplace Gen1 image found in release payload",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGomegaWithT(t)

			result, err := defaultAzureMarketplaceImageFromRelease(tc.nodePool, tc.releaseImage, tc.arch)

			if tc.expectedErr {
				g.Expect(err).ToNot(BeNil())
				if tc.expectedErrSubstring != "" {
					g.Expect(err.Error()).To(ContainSubstring(tc.expectedErrSubstring))
				}
			} else {
				g.Expect(err).To(BeNil())
				g.Expect(result).To(Equal(tc.expectedMarketplace))
			}
		})
	}
}
