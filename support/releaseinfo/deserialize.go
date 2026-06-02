package releaseinfo

import (
	"bytes"
	"encoding/json"
	"fmt"

	imageapi "github.com/openshift/api/image/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func DeserializeImageStream(data []byte) (*imageapi.ImageStream, error) {
	var imageStream imageapi.ImageStream
	if err := json.Unmarshal(data, &imageStream); err != nil {
		return nil, fmt.Errorf("couldn't read image stream data as a serialized ImageStream: %w\nraw data:\n%s", err, string(data))
	}
	return &imageStream, nil
}

// ImageMetadataResult contains the deserialized boot image metadata from the release payload.
type ImageMetadataResult struct {
	// Default is the single-stream metadata from the legacy "stream" key.
	Default *CoreOSStreamMetadata
	// Streams is the multi-stream metadata from the "streams" key, keyed by stream name
	// (e.g., "rhel-9", "rhel-10"). Only present in 5.0+ payloads.
	Streams map[string]*CoreOSStreamMetadata
}

func DeserializeImageMetadata(data []byte) (*CoreOSStreamMetadata, error) {
	result, err := DeserializeImageMetadataMultiStream(data)
	if err != nil {
		return nil, err
	}
	return result.Default, nil
}

// DeserializeImageMetadataMultiStream parses both the legacy single-stream "stream" key
// and the multi-stream "streams" key from the boot image ConfigMap.
func DeserializeImageMetadataMultiStream(data []byte) (*ImageMetadataResult, error) {
	var coreOSMetaCM corev1.ConfigMap
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 100).Decode(&coreOSMetaCM); err != nil {
		return nil, fmt.Errorf("couldn't read image lookup data as serialized ConfigMap: %w\nraw data:\n%s", err, string(data))
	}

	streamData, hasStreamData := coreOSMetaCM.Data["stream"]
	if !hasStreamData {
		return nil, fmt.Errorf("coreos stream metadata configmap is missing the 'stream' key")
	}
	var coreOSMeta CoreOSStreamMetadata
	if err := json.Unmarshal([]byte(streamData), &coreOSMeta); err != nil {
		return nil, fmt.Errorf("couldn't decode stream metadata data: %w\n%s", err, streamData)
	}

	result := &ImageMetadataResult{Default: &coreOSMeta}

	// Parse multi-stream metadata if present (5.0+ payloads).
	if streamsData, hasStreams := coreOSMetaCM.Data["streams"]; hasStreams {
		var streams map[string]*CoreOSStreamMetadata
		if err := json.Unmarshal([]byte(streamsData), &streams); err != nil {
			return nil, fmt.Errorf("couldn't decode multi-stream metadata: %w\n%s", err, streamsData)
		}
		result.Streams = streams
	}

	return result, nil
}
