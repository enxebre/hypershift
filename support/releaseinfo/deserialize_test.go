package releaseinfo

import (
	"testing"

	"github.com/openshift/hypershift/support/releaseinfo/fixtures"
)

func TestDeserializeImageStream(t *testing.T) {
	for _, imageStream := range [][]byte{fixtures.ImageReferencesJSON_4_8, fixtures.ImageReferencesJSON_4_10} {
		if _, err := DeserializeImageStream(imageStream); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeserializeImageMetadata(t *testing.T) {
	for _, imageMetadata := range [][]byte{fixtures.CoreOSBootImagesYAML_4_8, fixtures.CoreOSBootImagesYAML_4_10} {
		var coreOSMetadata *CoreOSStreamMetadata
		coreOSMetadata, _, err := DeserializeImageMetadata(imageMetadata)
		if err != nil {
			t.Fatal(err)
		}

		arch, ok := coreOSMetadata.Architectures["x86_64"]
		if !ok {
			t.Fatal("missing x86_64 architecture")
		}

		if arch.RHCOS.AzureDisk.URL == "" {
			t.Fatal("missing azure disk URL")
		}

	}
}
