package fleet

import "slices"

// ProtocolVersion identifies a worker/control-plane management protocol.
type ProtocolVersion struct {
	// Major is the reported major version; this model compares the complete
	// nonzero pair rather than validating a universal protocol edition.
	Major uint16 `json:"major"`
	// Minor is the reported minor version used in lexicographic range
	// comparison.
	Minor uint16 `json:"minor"`
}

// ProtocolRange is the inclusive protocol range supported by the control
// plane during rolling upgrades.
type ProtocolRange struct {
	// Minimum is the inclusive lower version; an inverted range yields
	// CompatibilityUnknown.
	Minimum ProtocolVersion
	// Maximum is the inclusive upper version.
	Maximum ProtocolVersion
}

// Capability identifies one advertised data-plane management operation.
// Negotiation enables a shared capability but does not grant authorization.
type Capability string

const (
	// CapabilityWorkerStatus advertises worker-status reads.
	CapabilityWorkerStatus Capability = "worker_status"
	// CapabilityQueueStatus advertises queue-status reads.
	CapabilityQueueStatus Capability = "queue_status"
	// CapabilityPause advertises paused admission.
	CapabilityPause Capability = "pause"
	// CapabilityResume advertises resumed admission.
	CapabilityResume Capability = "resume"
	// CapabilityDrain advertises worker draining.
	CapabilityDrain Capability = "drain"
	// CapabilityTerminate advertises worker termination.
	CapabilityTerminate Capability = "terminate"
	// CapabilityFailures advertises failure-record access.
	CapabilityFailures Capability = "failures"
	// CapabilityDeadLetters advertises dead-letter access.
	CapabilityDeadLetters Capability = "dead_letters"
	// CapabilityRetry advertises single-record retry.
	CapabilityRetry Capability = "retry"
	// CapabilityBulkRetry advertises bounded bulk retries.
	CapabilityBulkRetry Capability = "bulk_retry"
	// CapabilityDelete advertises record deletion.
	CapabilityDelete Capability = "delete"
	// CapabilityPurge advertises confirmed purging.
	CapabilityPurge Capability = "purge"
	// CapabilityReplay advertises record replay.
	CapabilityReplay Capability = "replay"
	// CapabilityRetentionCount advertises count-based retention.
	CapabilityRetentionCount Capability = "retention_count"
	// CapabilityRetentionTime advertises time-based retention.
	CapabilityRetentionTime Capability = "retention_time"
	// CapabilityRetentionBytes advertises byte-based retention.
	CapabilityRetentionBytes Capability = "retention_bytes"
)

// CompatibilityState describes a worker relative to the supported protocol
// range without pretending an incompatible worker is controllable.
type CompatibilityState string

const (
	// CompatibilityCompatible means the reported version lies in the supported
	// inclusive range.
	CompatibilityCompatible CompatibilityState = "compatible"
	// CompatibilityWorkerOlder means the worker is below the minimum supported
	// version.
	CompatibilityWorkerOlder CompatibilityState = "worker_older"
	// CompatibilityWorkerNewer means the worker exceeds the maximum supported
	// version.
	CompatibilityWorkerNewer CompatibilityState = "worker_newer"
	// CompatibilityUnknown means the range is inverted or the worker supplied a
	// zero version.
	CompatibilityUnknown CompatibilityState = "unknown"
)

// Compatibility is the deterministic result of protocol and capability
// negotiation.
type Compatibility struct {
	// State is version compatibility, not authenticated worker identity or
	// liveness.
	State CompatibilityState `json:"state"`
	// Enabled is a newly owned sorted intersection, empty unless versions are
	// compatible.
	Enabled []Capability `json:"enabled"`
	// WorkerOnly is a newly owned sorted set absent from control-plane
	// capabilities.
	WorkerOnly []Capability `json:"worker_only"`
	// ControlPlaneOnly is a newly owned sorted set absent from worker
	// capabilities.
	ControlPlaneOnly []Capability `json:"control_plane_only"`
}

// Negotiate safely compares a worker report with the control-plane range and
// enables only capabilities both sides support at a compatible version.
func Negotiate(
	supported ProtocolRange,
	worker ProtocolVersion,
	workerCapabilities []Capability,
	controlPlaneCapabilities []Capability,
) Compatibility {
	state := classifyCompatibility(supported, worker)
	workerSet := capabilitySet(workerCapabilities)
	controlPlaneSet := capabilitySet(controlPlaneCapabilities)

	result := Compatibility{State: state}
	for capability := range workerSet {
		if _, supportedByControlPlane := controlPlaneSet[capability]; supportedByControlPlane {
			if state == CompatibilityCompatible {
				result.Enabled = append(result.Enabled, capability)
			}

			continue
		}

		result.WorkerOnly = append(result.WorkerOnly, capability)
	}
	for capability := range controlPlaneSet {
		if _, supportedByWorker := workerSet[capability]; !supportedByWorker {
			result.ControlPlaneOnly = append(result.ControlPlaneOnly, capability)
		}
	}

	sortCapabilities(result.Enabled)
	sortCapabilities(result.WorkerOnly)
	sortCapabilities(result.ControlPlaneOnly)

	return result
}

func classifyCompatibility(supported ProtocolRange, worker ProtocolVersion) CompatibilityState {
	if compareVersion(supported.Minimum, supported.Maximum) > 0 || worker == (ProtocolVersion{}) {
		return CompatibilityUnknown
	}
	if compareVersion(worker, supported.Minimum) < 0 {
		return CompatibilityWorkerOlder
	}
	if compareVersion(worker, supported.Maximum) > 0 {
		return CompatibilityWorkerNewer
	}

	return CompatibilityCompatible
}

func compareVersion(left, right ProtocolVersion) int {
	if left.Major < right.Major {
		return -1
	}
	if left.Major > right.Major {
		return 1
	}
	if left.Minor < right.Minor {
		return -1
	}
	if left.Minor > right.Minor {
		return 1
	}

	return 0
}

func capabilitySet(capabilities []Capability) map[Capability]struct{} {
	set := make(map[Capability]struct{}, len(capabilities))
	for _, capability := range capabilities {
		set[capability] = struct{}{}
	}

	return set
}

func sortCapabilities(capabilities []Capability) {
	slices.Sort(capabilities)
}
