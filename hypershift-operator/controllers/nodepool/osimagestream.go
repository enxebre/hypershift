package nodepool

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	mcfgv1 "github.com/openshift/api/machineconfiguration/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/blang/semver"
)

// getRHELStream resolves the effective RHEL OS image stream for a NodePool.
// The decision logic:
//   - Explicit "rhel-10" + runc -> error (incompatible)
//   - Explicit "rhel-10" + release < 5.0 -> error (unsupported release)
//   - Explicit value set -> return as-is
//   - Unset + release >= 5.0 + runc -> return "rhel-9" (fallback due to runc incompatibility)
//   - Unset + release >= 5.0 -> return "rhel-10" (default for 5.0+)
//   - Unset + release < 5.0 -> return "" (legacy, no stream selection)
func getRHELStream(specStream hyperv1.OSImageStreamName, releaseVersion string, usesRunc bool) (hyperv1.OSImageStreamName, error) {
	version, err := semver.Parse(releaseVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse release version %q: %w", releaseVersion, err)
	}

	isRHEL10Capable := version.GTE(semver.Version{Major: 5, Minor: 0})

	// Explicit stream set by the user.
	if specStream != "" {
		if specStream == hyperv1.OSImageStreamRHEL10 && usesRunc {
			return "", fmt.Errorf("osImageStream %q is incompatible with runc container runtime; use crun or remove the ContainerRuntimeConfig", hyperv1.OSImageStreamRHEL10)
		}
		if specStream == hyperv1.OSImageStreamRHEL10 && !isRHEL10Capable {
			return "", fmt.Errorf("osImageStream %q requires release version >= 5.0, current version is %s", hyperv1.OSImageStreamRHEL10, releaseVersion)
		}
		return specStream, nil
	}

	// Unset: auto-detect.
	if !isRHEL10Capable {
		// Legacy releases do not support stream selection.
		return "", nil
	}

	// Release >= 5.0.
	if usesRunc {
		// RHEL 10 does not ship runc; fall back to RHEL 9.
		return hyperv1.OSImageStreamRHEL9, nil
	}

	// Default for 5.0+ is RHEL 10.
	return hyperv1.OSImageStreamRHEL10, nil
}

// extractUserConfigStrings extracts the raw config YAML strings from user-supplied ConfigMaps.
func extractUserConfigStrings(ctx context.Context, cg *ConfigGenerator) []string {
	var configs []string
	for _, config := range cg.nodePool.Spec.Config {
		configConfigMap := &corev1.ConfigMap{}
		configConfigMap.Name = config.Name
		configConfigMap.Namespace = cg.nodePool.Namespace
		if err := cg.Get(ctx, client.ObjectKeyFromObject(configConfigMap), configConfigMap); err != nil {
			continue
		}
		if data, ok := configConfigMap.Data[TokenSecretConfigKey]; ok {
			configs = append(configs, data)
		}
	}
	return configs
}

// configUsesRunc inspects the user-supplied NodePool config to determine
// whether any ContainerRuntimeConfig specifies runc as the default runtime.
func configUsesRunc(configs []string) bool {
	scheme := runtime.NewScheme()
	_ = mcfgv1.Install(scheme)

	yamlSerializer := serializer.NewSerializerWithOptions(
		serializer.DefaultMetaFactory, scheme, scheme,
		serializer.SerializerOptions{Yaml: true, Pretty: true, Strict: false},
	)

	for _, config := range configs {
		yamlReader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(config)))
		for {
			manifestRaw, err := yamlReader.Read()
			if err != nil && err != io.EOF {
				continue
			}
			if len(manifestRaw) != 0 && strings.TrimSpace(string(manifestRaw)) != "" {
				cr, _, err := yamlSerializer.Decode(manifestRaw, nil, nil)
				if err != nil {
					if err == io.EOF {
						break
					}
					continue
				}
				if crc, ok := cr.(*mcfgv1.ContainerRuntimeConfig); ok {
					if crc.Spec.ContainerRuntimeConfig != nil &&
						string(crc.Spec.ContainerRuntimeConfig.DefaultRuntime) == mcfgv1.ContainerRuntimeDefaultRuntimeRunc {
						return true
					}
				}
			}
			if err == io.EOF {
				break
			}
		}
	}
	return false
}
