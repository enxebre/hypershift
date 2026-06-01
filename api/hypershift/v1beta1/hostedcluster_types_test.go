package v1beta1

import (
	"encoding/json"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// These types represent the N-1 (previous) version of the API structs,
// before the omitempty changes. They are used to verify that JSON
// produced by the current types can be deserialized by previous
// versions of the code, and vice versa.
type clusterUpdateHistoryNMinus1 struct {
	State          configv1.UpdateState `json:"state"`
	StartedTime    metav1.Time          `json:"startedTime"`
	CompletionTime *metav1.Time         `json:"completionTime"`
	Version        string               `json:"version"`
	Image          string               `json:"image"`
	Verified       bool                 `json:"verified"`
	AcceptedRisks  string               `json:"acceptedRisks,omitempty"`
}

type clusterVersionStatusNMinus1 struct {
	Desired            configv1.Release              `json:"desired"`
	History            []clusterUpdateHistoryNMinus1 `json:"history,omitempty"`
	ObservedGeneration int64                         `json:"observedGeneration"`
	AvailableUpdates   []configv1.Release            `json:"availableUpdates"`
	ConditionalUpdates []configv1.ConditionalUpdate  `json:"conditionalUpdates,omitempty"`
}

func TestClusterUpdateHistorySerializationCompatibility(t *testing.T) {
	now := metav1.NewTime(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))

	tests := []struct {
		name          string
		current       ClusterUpdateHistory
		nMinus1Result clusterUpdateHistoryNMinus1
	}{
		{
			name: "When CompletionTime is set it should round-trip to N-1",
			current: ClusterUpdateHistory{
				State:          configv1.CompletedUpdate,
				StartedTime:    now,
				CompletionTime: &now,
				Version:        ptr.To("4.17.0"),
				Image:          ptr.To("quay.io/ocp:4.17.0"),
			},
			nMinus1Result: clusterUpdateHistoryNMinus1{
				State:          configv1.CompletedUpdate,
				StartedTime:    now,
				CompletionTime: &now,
				Version:        "4.17.0",
				Image:          "quay.io/ocp:4.17.0",
			},
		},
		{
			name: "When CompletionTime is nil it should be omitted and N-1 should deserialize as nil",
			current: ClusterUpdateHistory{
				State:       configv1.PartialUpdate,
				StartedTime: now,
				Version:     ptr.To("4.17.0"),
				Image:       ptr.To("quay.io/ocp:4.17.0"),
			},
			nMinus1Result: clusterUpdateHistoryNMinus1{
				State:       configv1.PartialUpdate,
				StartedTime: now,
				Version:     "4.17.0",
				Image:       "quay.io/ocp:4.17.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal current (N) version
			data, err := json.Marshal(tt.current)
			if err != nil {
				t.Fatalf("failed to marshal current struct: %v", err)
			}

			// Deserialize into N-1 struct
			var nMinus1 clusterUpdateHistoryNMinus1
			if err := json.Unmarshal(data, &nMinus1); err != nil {
				t.Fatalf("N-1 failed to unmarshal JSON from N: %v", err)
			}
			if nMinus1.State != tt.nMinus1Result.State {
				t.Errorf("State mismatch: got %v, want %v", nMinus1.State, tt.nMinus1Result.State)
			}
			if nMinus1.Version != tt.nMinus1Result.Version {
				t.Errorf("Version mismatch: got %v, want %v", nMinus1.Version, tt.nMinus1Result.Version)
			}
			if nMinus1.Image != tt.nMinus1Result.Image {
				t.Errorf("Image mismatch: got %v, want %v", nMinus1.Image, tt.nMinus1Result.Image)
			}
			if (nMinus1.CompletionTime == nil) != (tt.nMinus1Result.CompletionTime == nil) {
				t.Errorf("CompletionTime nil mismatch: got %v, want %v", nMinus1.CompletionTime, tt.nMinus1Result.CompletionTime)
			}
			if nMinus1.CompletionTime != nil && tt.nMinus1Result.CompletionTime != nil &&
				!nMinus1.CompletionTime.Equal(tt.nMinus1Result.CompletionTime) {
				t.Errorf("CompletionTime mismatch: got %v, want %v", nMinus1.CompletionTime, tt.nMinus1Result.CompletionTime)
			}

			// Reverse: marshal N-1 and deserialize into current (N)
			nMinus1Data, err := json.Marshal(tt.nMinus1Result)
			if err != nil {
				t.Fatalf("failed to marshal N-1 struct: %v", err)
			}
			var roundTripped ClusterUpdateHistory
			if err := json.Unmarshal(nMinus1Data, &roundTripped); err != nil {
				t.Fatalf("N failed to unmarshal JSON from N-1: %v", err)
			}
			if roundTripped.State != tt.current.State {
				t.Errorf("State mismatch after N-1 round-trip: got %v, want %v", roundTripped.State, tt.current.State)
			}
			if (roundTripped.Version == nil) != (tt.current.Version == nil) {
				t.Errorf("Version nil mismatch after N-1 round-trip: got %v, want %v", roundTripped.Version, tt.current.Version)
			} else if roundTripped.Version != nil && *roundTripped.Version != *tt.current.Version {
				t.Errorf("Version mismatch after N-1 round-trip: got %v, want %v", *roundTripped.Version, *tt.current.Version)
			}
			if (roundTripped.CompletionTime == nil) != (tt.current.CompletionTime == nil) {
				t.Errorf("CompletionTime nil mismatch after round-trip: got %v, want %v", roundTripped.CompletionTime, tt.current.CompletionTime)
			}
		})
	}
}

func TestClusterUpdateHistoryNMinus1NullCompletionTime(t *testing.T) {
	t.Run("When N-1 serializes null CompletionTime it should deserialize into N as nil pointer", func(t *testing.T) {
		// N-1 format: completionTime is present as explicit null (no omitempty on pointer)
		nMinus1JSON := `{"state":"Partial","startedTime":"2025-01-15T10:30:00Z","completionTime":null,"version":"4.17.0","image":"quay.io/ocp:4.17.0","verified":false}`

		var current ClusterUpdateHistory
		if err := json.Unmarshal([]byte(nMinus1JSON), &current); err != nil {
			t.Fatalf("N failed to unmarshal N-1 JSON with null completionTime: %v", err)
		}
		if current.CompletionTime != nil {
			t.Errorf("CompletionTime should be nil when deserializing null, got %v", current.CompletionTime)
		}
		if current.State != configv1.PartialUpdate {
			t.Errorf("State should be Partial, got %v", current.State)
		}
	})
}

func TestClusterUpdateHistoryDroppedFields(t *testing.T) {
	t.Run("When N-1 serializes Verified and AcceptedRisks they should be ignored by N", func(t *testing.T) {
		nMinus1JSON := `{"state":"Completed","startedTime":"2025-01-15T10:30:00Z","completionTime":"2025-01-15T11:00:00Z","version":"4.17.0","image":"quay.io/ocp:4.17.0","verified":true,"acceptedRisks":"risk1"}`

		var current ClusterUpdateHistory
		if err := json.Unmarshal([]byte(nMinus1JSON), &current); err != nil {
			t.Fatalf("N should unmarshal N-1 JSON with extra fields: %v", err)
		}
		if current.State != configv1.CompletedUpdate {
			t.Errorf("State should be Completed, got %v", current.State)
		}
		if current.CompletionTime == nil {
			t.Fatal("CompletionTime should not be nil")
		}
	})

	t.Run("When N serializes without Verified and AcceptedRisks N-1 should get zero values", func(t *testing.T) {
		now := metav1.NewTime(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
		current := ClusterUpdateHistory{
			State:       configv1.CompletedUpdate,
			StartedTime: now,
			Version:     ptr.To("4.17.0"),
			Image:       ptr.To("quay.io/ocp:4.17.0"),
		}

		data, err := json.Marshal(current)
		if err != nil {
			t.Fatalf("failed to marshal current: %v", err)
		}

		var nMinus1 clusterUpdateHistoryNMinus1
		if err := json.Unmarshal(data, &nMinus1); err != nil {
			t.Fatalf("N-1 failed to unmarshal: %v", err)
		}
		if nMinus1.Verified != false {
			t.Errorf("Verified should default to false, got %v", nMinus1.Verified)
		}
		if nMinus1.AcceptedRisks != "" {
			t.Errorf("AcceptedRisks should default to empty, got %v", nMinus1.AcceptedRisks)
		}
	})
}

func TestClusterVersionStatusSerializationCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		current       ClusterVersionStatus
		nMinus1Result clusterVersionStatusNMinus1
	}{
		{
			name: "When AvailableUpdates is nil it should be omitted and N-1 should deserialize as nil",
			current: ClusterVersionStatus{
				Desired:            configv1.Release{Version: "4.17.0", Image: "quay.io/ocp:4.17.0"},
				ObservedGeneration: ptr.To(int64(1)),
			},
			nMinus1Result: clusterVersionStatusNMinus1{
				Desired:            configv1.Release{Version: "4.17.0", Image: "quay.io/ocp:4.17.0"},
				ObservedGeneration: 1,
			},
		},
		{
			name: "When AvailableUpdates is populated it should round-trip to N-1",
			current: ClusterVersionStatus{
				Desired:            configv1.Release{Version: "4.17.0", Image: "quay.io/ocp:4.17.0"},
				ObservedGeneration: ptr.To(int64(1)),
				AvailableUpdates: []configv1.Release{
					{Version: "4.17.1", Image: "quay.io/ocp:4.17.1"},
				},
			},
			nMinus1Result: clusterVersionStatusNMinus1{
				Desired:            configv1.Release{Version: "4.17.0", Image: "quay.io/ocp:4.17.0"},
				ObservedGeneration: 1,
				AvailableUpdates: []configv1.Release{
					{Version: "4.17.1", Image: "quay.io/ocp:4.17.1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal current (N) version
			data, err := json.Marshal(tt.current)
			if err != nil {
				t.Fatalf("failed to marshal current struct: %v", err)
			}

			// Deserialize into N-1 struct
			var nMinus1 clusterVersionStatusNMinus1
			if err := json.Unmarshal(data, &nMinus1); err != nil {
				t.Fatalf("N-1 failed to unmarshal JSON from N: %v", err)
			}
			if len(nMinus1.AvailableUpdates) != len(tt.nMinus1Result.AvailableUpdates) {
				t.Errorf("AvailableUpdates length mismatch: got %d, want %d",
					len(nMinus1.AvailableUpdates), len(tt.nMinus1Result.AvailableUpdates))
			}
			if nMinus1.ObservedGeneration != tt.nMinus1Result.ObservedGeneration {
				t.Errorf("ObservedGeneration mismatch: got %d, want %d",
					nMinus1.ObservedGeneration, tt.nMinus1Result.ObservedGeneration)
			}

			// Reverse: marshal N-1 and deserialize into current (N)
			nMinus1Data, err := json.Marshal(tt.nMinus1Result)
			if err != nil {
				t.Fatalf("failed to marshal N-1 struct: %v", err)
			}
			var roundTripped ClusterVersionStatus
			if err := json.Unmarshal(nMinus1Data, &roundTripped); err != nil {
				t.Fatalf("N failed to unmarshal JSON from N-1: %v", err)
			}
			if len(roundTripped.AvailableUpdates) != len(tt.current.AvailableUpdates) {
				t.Errorf("AvailableUpdates length mismatch after round-trip: got %d, want %d",
					len(roundTripped.AvailableUpdates), len(tt.current.AvailableUpdates))
			}
		})
	}
}

func TestClusterVersionStatusNMinus1NullAvailableUpdates(t *testing.T) {
	t.Run("When N-1 serializes null AvailableUpdates it should deserialize into N as nil slice", func(t *testing.T) {
		// N-1 format: availableUpdates is present as explicit null (no omitempty)
		nMinus1JSON := `{"desired":{"version":"4.17.0","image":"quay.io/ocp:4.17.0"},"observedGeneration":1,"availableUpdates":null}`

		var current ClusterVersionStatus
		if err := json.Unmarshal([]byte(nMinus1JSON), &current); err != nil {
			t.Fatalf("N failed to unmarshal N-1 JSON with null availableUpdates: %v", err)
		}
		if current.AvailableUpdates != nil {
			t.Errorf("AvailableUpdates should be nil when deserializing null, got %v", current.AvailableUpdates)
		}
		if current.Desired.Version != "4.17.0" {
			t.Errorf("Desired.Version should be 4.17.0, got %v", current.Desired.Version)
		}
	})
}
