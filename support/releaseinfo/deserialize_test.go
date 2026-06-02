package releaseinfo

import (
	"testing"

	. "github.com/onsi/gomega"

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
		coreOSMetadata, err := DeserializeImageMetadata(imageMetadata)
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := coreOSMetadata.Architectures["x86_64"]; !ok {
			t.Fatal(err)
		}

		if coreOSMetadata.Architectures["x86_64"].RHCOS.AzureDisk.URL == "" {
			t.Fatal(err)
		}

	}
}

func TestDeserializeImageMetadataMultiStream(t *testing.T) {
	t.Run("When parsing a legacy single-stream payload it should return default metadata with no streams", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_4_10)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Default).ToNot(BeNil())
		g.Expect(result.Default.Stream).To(Equal("rhcos-4.10"))
		g.Expect(result.Streams).To(BeNil())
	})

	t.Run("When parsing a 5.0 multi-stream payload it should return both default and streams", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_5_0)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(result.Default).ToNot(BeNil())
		g.Expect(result.Default.Stream).To(Equal("rhcos-4.21"))

		g.Expect(result.Streams).To(HaveLen(2))
		g.Expect(result.Streams).To(HaveKey("rhel-9"))
		g.Expect(result.Streams).To(HaveKey("rhel-10"))

		g.Expect(result.Streams["rhel-9"].Stream).To(Equal("rhcos-4.21"))
		g.Expect(result.Streams["rhel-10"].Stream).To(Equal("rhcos-5.0"))
	})

	t.Run("When parsing a 5.0 payload it should have distinct AMIs per stream", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_5_0)
		g.Expect(err).ToNot(HaveOccurred())

		rhel9AMI := result.Streams["rhel-9"].Architectures["x86_64"].Images.AWS.Regions["us-east-1"].Image
		rhel10AMI := result.Streams["rhel-10"].Architectures["x86_64"].Images.AWS.Regions["us-east-1"].Image
		g.Expect(rhel9AMI).To(Equal("ami-rhel9-us-east-1"))
		g.Expect(rhel10AMI).To(Equal("ami-rhel10-us-east-1"))
		g.Expect(rhel9AMI).ToNot(Equal(rhel10AMI))
	})
}

func TestStreamMetadataForStream(t *testing.T) {
	t.Run("When requesting rhel-10 from a multi-stream payload it should return rhel-10 metadata", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_5_0)
		g.Expect(err).ToNot(HaveOccurred())

		ri := &ReleaseImage{
			StreamMetadata:  result.Default,
			StreamsMetadata: result.Streams,
		}
		meta := ri.StreamMetadataForStream("rhel-10")
		g.Expect(meta).ToNot(BeNil())
		g.Expect(meta.Stream).To(Equal("rhcos-5.0"))
		g.Expect(meta.Architectures["x86_64"].Images.AWS.Regions["us-east-1"].Image).To(Equal("ami-rhel10-us-east-1"))
	})

	t.Run("When requesting rhel-9 from a multi-stream payload it should return rhel-9 metadata", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_5_0)
		g.Expect(err).ToNot(HaveOccurred())

		ri := &ReleaseImage{
			StreamMetadata:  result.Default,
			StreamsMetadata: result.Streams,
		}
		meta := ri.StreamMetadataForStream("rhel-9")
		g.Expect(meta).ToNot(BeNil())
		g.Expect(meta.Stream).To(Equal("rhcos-4.21"))
	})

	t.Run("When requesting empty stream it should fall back to default", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_5_0)
		g.Expect(err).ToNot(HaveOccurred())

		ri := &ReleaseImage{
			StreamMetadata:  result.Default,
			StreamsMetadata: result.Streams,
		}
		meta := ri.StreamMetadataForStream("")
		g.Expect(meta).ToNot(BeNil())
		g.Expect(meta.Stream).To(Equal("rhcos-4.21"))
	})

	t.Run("When requesting a stream from a legacy payload it should fall back to default", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_4_10)
		g.Expect(err).ToNot(HaveOccurred())

		ri := &ReleaseImage{
			StreamMetadata:  result.Default,
			StreamsMetadata: result.Streams,
		}
		meta := ri.StreamMetadataForStream("rhel-10")
		g.Expect(meta).ToNot(BeNil())
		g.Expect(meta.Stream).To(Equal("rhcos-4.10"))
	})

	t.Run("When requesting unknown stream from a multi-stream payload it should fall back to default", func(t *testing.T) {
		g := NewWithT(t)
		result, err := DeserializeImageMetadataMultiStream(fixtures.CoreOSBootImagesYAML_5_0)
		g.Expect(err).ToNot(HaveOccurred())

		ri := &ReleaseImage{
			StreamMetadata:  result.Default,
			StreamsMetadata: result.Streams,
		}
		meta := ri.StreamMetadataForStream("rhel-11")
		g.Expect(meta).ToNot(BeNil())
		g.Expect(meta.Stream).To(Equal("rhcos-4.21"))
	})
}
