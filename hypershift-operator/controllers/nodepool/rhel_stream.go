package nodepool

import (
	"fmt"

	"github.com/blang/semver"
)

// rhelStreamRHEL9 is the RHEL 9 stream identifier.
const rhelStreamRHEL9 = "rhel-9"

// rhelStreamRHEL10 is the RHEL 10 stream identifier.
const rhelStreamRHEL10 = "rhel-10"

// minRHEL10Version is the minimum release version that carries RHEL 10 images.
var minRHEL10Version = semver.Version{Major: 5, Minor: 0, Patch: 0}

// getRHELStream resolves the RHEL stream for a NodePool.
// Returns the stream name ("rhel-9" or "rhel-10") or an error for
// invalid combinations. Returns "" for pre-5.0 releases with no explicit
// stream, indicating legacy single-stream behavior.
func getRHELStream(specStream string, releaseVersion semver.Version, usesRunc bool) (string, error) {
	// Strip pre-release and build metadata for comparison.
	rv := semver.Version{Major: releaseVersion.Major, Minor: releaseVersion.Minor, Patch: releaseVersion.Patch}

	switch {
	// Explicit rhel-10 with runc is always an error — RHEL 10 does not ship runc.
	case specStream == rhelStreamRHEL10 && usesRunc:
		return "", fmt.Errorf("OS stream %s is incompatible with default_runtime=runc; RHEL 10 does not ship runc", rhelStreamRHEL10)

	// Explicit rhel-10 on release < 5.0 is an error — those payloads don't carry RHEL 10 images.
	case specStream == rhelStreamRHEL10 && rv.LT(minRHEL10Version):
		return "", fmt.Errorf("OS stream %s requires release version >= %d.%d", rhelStreamRHEL10, minRHEL10Version.Major, minRHEL10Version.Minor)

	// Explicit stream set — return it as-is.
	case specStream != "":
		return specStream, nil

	// Unset + release >= 5.0 + runc — fallback to RHEL 9.
	case !rv.LT(minRHEL10Version) && usesRunc:
		return rhelStreamRHEL9, nil

	// Unset + release >= 5.0 — default to RHEL 10.
	case !rv.LT(minRHEL10Version):
		return rhelStreamRHEL10, nil

	// Unset + release < 5.0 — legacy single-stream behavior, no OSImageStream CR.
	default:
		return "", nil
	}
}
