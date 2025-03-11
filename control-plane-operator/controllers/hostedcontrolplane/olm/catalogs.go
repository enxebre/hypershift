package olm

import (
	"context"
	"fmt"
	"strings"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/util"

	"github.com/blang/semver"
)

func GetCatalogImages(ctx context.Context, hcp hyperv1.HostedControlPlane, pullSecret []byte, imageMetadataProvider util.ImageMetadataProvider, registryOverrides map[string][]string) (map[string]string, error) {
	imageRef := hcp.Spec.ReleaseImage
	imageConfig, _, _, err := imageMetadataProvider.GetMetadata(ctx, imageRef, pullSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get image metadata: %w", err)
	}

	version, err := semver.Parse(imageConfig.Config.Labels["io.openshift.release"])
	if err != nil {
		return nil, fmt.Errorf("invalid OpenShift release version format: %s", imageConfig.Config.Labels["io.openshift.release"])
	}

	registries := []string{
		"registry.redhat.io/redhat",
	}
	if len(registryOverrides) > 0 {
		for registrySource, registryDest := range registryOverrides {
			if registries[0] == registrySource {
				registries = registryDest
				break
			}
		}
	}

	//check catalogs of last 4 supported version in case new version is not available
	supportedVersions := 4
	imageRegistry := ""
	for i := 0; i < supportedVersions; i++ {
		for _, registry := range registries {
			testImage := fmt.Sprintf("%s/certified-operator-index:v%d.%d", registry, version.Major, version.Minor)

			_, dockerImage, err := imageMetadataProvider.GetDigest(ctx, testImage, pullSecret)
			if err == nil {
				imageRegistry = fmt.Sprintf("%s/%s", dockerImage.Registry, dockerImage.Namespace)
				break
			}

			// Manifest unknown error is expected if the image is not available.
			if !strings.Contains(err.Error(), "manifest unknown") {
				return nil, err // Return if it's an unexpected error
			}
		}
		if imageRegistry != "" {
			break
		}
		if i == supportedVersions-1 {
			return nil, fmt.Errorf("failed to get image digest for 4 previous versions of certified-operator-index: %w", err)
		}
		version.Minor--
	}

	operators := map[string]string{
		"certified-operators": fmt.Sprintf("%s/certified-operator-index:v%d.%d", imageRegistry, version.Major, version.Minor),
		"community-operators": fmt.Sprintf("%s/community-operator-index:v%d.%d", imageRegistry, version.Major, version.Minor),
		"redhat-marketplace":  fmt.Sprintf("%s/redhat-marketplace-index:v%d.%d", imageRegistry, version.Major, version.Minor),
		"redhat-operators":    fmt.Sprintf("%s/redhat-operator-index:v%d.%d", imageRegistry, version.Major, version.Minor),
	}

	return operators, nil
}
