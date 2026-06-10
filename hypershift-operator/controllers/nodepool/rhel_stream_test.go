package nodepool

import (
	"strings"
	"testing"

	"github.com/blang/semver"
)

func TestGetRHELStream(t *testing.T) {
	tests := []struct {
		name           string
		specStream     string
		releaseVersion semver.Version
		usesRunc       bool
		expectedStream string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "When explicit rhel-10 is set with runc it should return error",
			specStream:     "rhel-10",
			releaseVersion: semver.Version{Major: 5, Minor: 0, Patch: 0},
			usesRunc:       true,
			expectError:    true,
			errorContains:  "incompatible with default_runtime=runc",
		},
		{
			name:           "When explicit rhel-10 is set on release < 5.0 it should return error",
			specStream:     "rhel-10",
			releaseVersion: semver.Version{Major: 4, Minor: 19, Patch: 0},
			usesRunc:       false,
			expectError:    true,
			errorContains:  "requires release version >= 5.0",
		},
		{
			name:           "When explicit rhel-10 is set on release >= 5.0 without runc it should return rhel-10",
			specStream:     "rhel-10",
			releaseVersion: semver.Version{Major: 5, Minor: 0, Patch: 0},
			usesRunc:       false,
			expectedStream: "rhel-10",
		},
		{
			name:           "When explicit rhel-9 is set on any release it should return rhel-9",
			specStream:     "rhel-9",
			releaseVersion: semver.Version{Major: 4, Minor: 19, Patch: 0},
			usesRunc:       false,
			expectedStream: "rhel-9",
		},
		{
			name:           "When explicit rhel-9 is set on release >= 5.0 it should return rhel-9",
			specStream:     "rhel-9",
			releaseVersion: semver.Version{Major: 5, Minor: 1, Patch: 0},
			usesRunc:       false,
			expectedStream: "rhel-9",
		},
		{
			name:           "When no stream is set on release >= 5.0 with runc it should fallback to rhel-9",
			specStream:     "",
			releaseVersion: semver.Version{Major: 5, Minor: 0, Patch: 0},
			usesRunc:       true,
			expectedStream: "rhel-9",
		},
		{
			name:           "When no stream is set on release >= 5.0 without runc it should default to rhel-10",
			specStream:     "",
			releaseVersion: semver.Version{Major: 5, Minor: 0, Patch: 0},
			usesRunc:       false,
			expectedStream: "rhel-10",
		},
		{
			name:           "When no stream is set on release < 5.0 it should return empty string for legacy behavior",
			specStream:     "",
			releaseVersion: semver.Version{Major: 4, Minor: 19, Patch: 0},
			usesRunc:       false,
			expectedStream: "",
		},
		{
			name:           "When no stream is set on release < 5.0 with runc it should return empty string",
			specStream:     "",
			releaseVersion: semver.Version{Major: 4, Minor: 19, Patch: 0},
			usesRunc:       true,
			expectedStream: "",
		},
		{
			name:           "When explicit rhel-10 is set on release 5.1.0 without runc it should return rhel-10",
			specStream:     "rhel-10",
			releaseVersion: semver.Version{Major: 5, Minor: 1, Patch: 0},
			usesRunc:       false,
			expectedStream: "rhel-10",
		},
		{
			name:           "When release has pre-release metadata it should strip it for comparison",
			specStream:     "",
			releaseVersion: semver.Version{Major: 5, Minor: 0, Patch: 0, Pre: []semver.PRVersion{{VersionStr: "rc", IsNum: false}, {VersionNum: 1, IsNum: true}}},
			usesRunc:       false,
			expectedStream: "rhel-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := getRHELStream(tt.specStream, tt.releaseVersion, tt.usesRunc)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stream != tt.expectedStream {
				t.Errorf("expected stream %q, got %q", tt.expectedStream, stream)
			}
		})
	}
}
