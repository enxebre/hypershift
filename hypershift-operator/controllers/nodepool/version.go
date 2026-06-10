package nodepool

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	corev1 "k8s.io/api/core/v1"

	capiv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

// versionKey is used as a map key for grouping machines by version.
type versionKey struct {
	ocpVersion     string
	kubeletVersion string
}

// nodeVersionsFromMachines aggregates version and health information from CAPI Machines.
// It groups machines by (ocpVersion, kubeletVersion) and counts ready/unready nodes
// based on the CAPI NodeHealthy condition.
func (r *NodePoolReconciler) nodeVersionsFromMachines(_ context.Context, machines []*capiv1.Machine, nodePool *hyperv1.NodePool) []hyperv1.NodeVersion {
	type counts struct {
		ready   int32
		unready int32
	}
	versionCounts := make(map[versionKey]*counts)

	for _, machine := range machines {
		// Skip machines that haven't registered a node yet.
		if machine.Status.NodeInfo == nil {
			continue
		}

		kubeletVersion := machine.Status.NodeInfo.KubeletVersion

		// Resolve OCP version from Machine annotation.
		// For replace upgrades, the annotation is propagated via the MachineDeployment template at Machine creation.
		// For in-place upgrades, the annotation is set by the in-place upgrader (sourced from the token secret)
		// after each node completes its upgrade.
		// Fallback to nodePool.Status.Version for machines created before this annotation existed.
		ocpVersion := machine.Annotations[hyperv1.NodePoolReleaseVersionAnnotation]
		if ocpVersion == "" {
			ocpVersion = nodePool.Status.Version
		}

		key := versionKey{ocpVersion: ocpVersion, kubeletVersion: kubeletVersion}
		if _, exists := versionCounts[key]; !exists {
			versionCounts[key] = &counts{}
		}

		// Determine node health from CAPI NodeHealthy condition.
		condition := findMachineStatusCondition(machine, string(capiv1.MachineNodeHealthyCondition))
		if condition != nil && condition.Status == corev1.ConditionTrue {
			versionCounts[key].ready++
		} else {
			versionCounts[key].unready++
		}
	}

	if len(versionCounts) == 0 {
		return nil
	}

	result := make([]hyperv1.NodeVersion, 0, len(versionCounts))
	for key, c := range versionCounts {
		result = append(result, hyperv1.NodeVersion{
			OCPVersion:       key.ocpVersion,
			KubeletVersion:   key.kubeletVersion,
			ReadyNodeCount:   &c.ready,
			UnreadyNodeCount: &c.unready,
		})
	}

	// Sort for deterministic output: by ocpVersion, then kubeletVersion.
	sort.Slice(result, func(i, j int) bool {
		if result[i].OCPVersion != result[j].OCPVersion {
			return result[i].OCPVersion < result[j].OCPVersion
		}
		return result[i].KubeletVersion < result[j].KubeletVersion
	})

	return result
}

// setNodesInfoStatus aggregates node version and health information from CAPI Machines
// and sets it on nodePool.Status.NodesInfo.
func (r *NodePoolReconciler) setNodesInfoStatus(ctx context.Context, nodePool *hyperv1.NodePool) error {
	machines, err := r.getMachinesForNodePool(ctx, nodePool)
	if err != nil {
		return fmt.Errorf("failed to get Machines for NodesInfo: %w", err)
	}

	nodeVersions := r.nodeVersionsFromMachines(ctx, machines, nodePool)
	nodePool.Status.NodesInfo = hyperv1.NodePoolNodesInfo{
		NodeVersions: nodeVersions,
	}

	// Set status.osImageStream from observed node OS images.
	// When no stream is detected (empty string), the zero-value reference
	// omits the field from JSON (via omitzero tag), indicating default stream.
	nodePool.Status.OSImageStream = hyperv1.OSImageStreamReference{
		Name: osImageStreamFromMachines(machines),
	}

	return nil
}

// osImageStreamFromMachines detects the RHEL stream (rhel-9 or rhel-10) from CAPI Machines'
// NodeInfo.OSImage field. The RHCOS version string contains a major version prefix:
// 4xx = RHEL 9 (e.g., "Red Hat Enterprise Linux CoreOS 419.97..."),
// 5xx = RHEL 10 (e.g., "Red Hat Enterprise Linux CoreOS 510.97...").
// Returns the stream name if a majority of nodes report a consistent stream,
// or "" if no stream can be determined.
func osImageStreamFromMachines(machines []*capiv1.Machine) string {
	streamCounts := make(map[string]int)
	total := 0

	for _, machine := range machines {
		if machine.Status.NodeInfo == nil {
			continue
		}
		osImage := machine.Status.NodeInfo.OSImage
		if osImage == "" {
			continue
		}
		stream := detectRHELStreamFromOSImage(osImage)
		if stream != "" {
			streamCounts[stream]++
			total++
		}
	}

	if total == 0 {
		return ""
	}

	// Return the stream with the majority of nodes.
	for stream, count := range streamCounts {
		if count > total/2 {
			return stream
		}
	}
	return ""
}

// detectRHELStreamFromOSImage extracts the RHEL stream from a CoreOS version string.
// The expected format is "Red Hat Enterprise Linux CoreOS NNN.NN.YYYYMMDDHHMMSS-X (Name)"
// where NNN is the major RHCOS version: 4xx maps to rhel-9, 5xx maps to rhel-10.
func detectRHELStreamFromOSImage(osImage string) string {
	// Extract the version number after "CoreOS "
	const prefix = "CoreOS "
	idx := strings.Index(osImage, prefix)
	if idx < 0 {
		return ""
	}
	versionStr := osImage[idx+len(prefix):]
	// The version starts with digits like "419.97" or "510.97"
	if len(versionStr) < 3 {
		return ""
	}

	// Parse the major version (first 1-3 digits before the first dot).
	dotIdx := strings.Index(versionStr, ".")
	if dotIdx <= 0 {
		return ""
	}
	majorStr := versionStr[:dotIdx]

	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return ""
	}

	// RHCOS 4xx = RHEL 9, RHCOS 5xx = RHEL 10
	switch {
	case major >= 500:
		return "rhel-10"
	case major >= 400:
		return "rhel-9"
	default:
		return ""
	}
}
