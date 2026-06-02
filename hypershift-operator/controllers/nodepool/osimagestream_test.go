package nodepool

import (
	"testing"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	. "github.com/onsi/gomega"
)

func TestGetRHELStream(t *testing.T) {
	tests := []struct {
		name           string
		specStream     hyperv1.OSImageStreamName
		releaseVersion string
		usesRunc       bool
		expected       hyperv1.OSImageStreamName
		expectErr      bool
	}{
		{
			name:           "When explicit rhel-9 on 4.x it should return rhel-9",
			specStream:     hyperv1.OSImageStreamRHEL9,
			releaseVersion: "4.19.0",
			expected:       hyperv1.OSImageStreamRHEL9,
		},
		{
			name:           "When explicit rhel-9 on 5.0 it should return rhel-9",
			specStream:     hyperv1.OSImageStreamRHEL9,
			releaseVersion: "5.0.0",
			expected:       hyperv1.OSImageStreamRHEL9,
		},
		{
			name:           "When explicit rhel-10 on 5.0 it should return rhel-10",
			specStream:     hyperv1.OSImageStreamRHEL10,
			releaseVersion: "5.0.0",
			expected:       hyperv1.OSImageStreamRHEL10,
		},
		{
			name:           "When explicit rhel-10 on 4.x it should return error",
			specStream:     hyperv1.OSImageStreamRHEL10,
			releaseVersion: "4.19.0",
			expectErr:      true,
		},
		{
			name:           "When explicit rhel-10 with runc it should return error",
			specStream:     hyperv1.OSImageStreamRHEL10,
			releaseVersion: "5.0.0",
			usesRunc:       true,
			expectErr:      true,
		},
		{
			name:           "When unset on 4.x it should return empty (legacy)",
			specStream:     "",
			releaseVersion: "4.19.0",
			expected:       "",
		},
		{
			name:           "When unset on 5.0 it should return rhel-10",
			specStream:     "",
			releaseVersion: "5.0.0",
			expected:       hyperv1.OSImageStreamRHEL10,
		},
		{
			name:           "When unset on 5.0 with runc it should fallback to rhel-9",
			specStream:     "",
			releaseVersion: "5.0.0",
			usesRunc:       true,
			expected:       hyperv1.OSImageStreamRHEL9,
		},
		{
			name:           "When unset on 5.1 it should return rhel-10",
			specStream:     "",
			releaseVersion: "5.1.0",
			expected:       hyperv1.OSImageStreamRHEL10,
		},
		{
			name:           "When explicit rhel-9 with runc on 5.0 it should return rhel-9",
			specStream:     hyperv1.OSImageStreamRHEL9,
			releaseVersion: "5.0.0",
			usesRunc:       true,
			expected:       hyperv1.OSImageStreamRHEL9,
		},
		{
			name:           "When invalid version it should return error",
			specStream:     "",
			releaseVersion: "invalid",
			expectErr:      true,
		},
		{
			name:           "When pre-release version 5.0.0-rc.1 it should return empty (pre-release < 5.0.0 per semver)",
			specStream:     "",
			releaseVersion: "5.0.0-rc.1",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result, err := getRHELStream(tt.specStream, tt.releaseVersion, tt.usesRunc)
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(result).To(Equal(tt.expected))
			}
		})
	}
}

func TestConfigUsesRunc(t *testing.T) {
	tests := []struct {
		name     string
		configs  []string
		expected bool
	}{
		{
			name:     "When no configs it should return false",
			configs:  nil,
			expected: false,
		},
		{
			name: "When ContainerRuntimeConfig with runc it should return true",
			configs: []string{`apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: set-runc
spec:
  containerRuntimeConfig:
    defaultRuntime: runc`},
			expected: true,
		},
		{
			name: "When ContainerRuntimeConfig with crun it should return false",
			configs: []string{`apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: set-crun
spec:
  containerRuntimeConfig:
    defaultRuntime: crun`},
			expected: false,
		},
		{
			name: "When MachineConfig (not CRC) it should return false",
			configs: []string{`apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  name: some-mc
spec:
  config:
    ignition:
      version: 3.2.0`},
			expected: false,
		},
		{
			name: "When ContainerRuntimeConfig without defaultRuntime it should return false",
			configs: []string{`apiVersion: machineconfiguration.openshift.io/v1
kind: ContainerRuntimeConfig
metadata:
  name: no-default
spec:
  containerRuntimeConfig:
    overlaySize: 10G`},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(configUsesRunc(tt.configs)).To(Equal(tt.expected))
		})
	}
}
