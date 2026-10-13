// Package fleet models reported worker liveness and protocol/capability
// compatibility without supervising processes or implementing delivery.
// NewRegistry retains copied reports under one mutex; NewProjection aggregates
// lifecycle events with a cardinality overflow bucket. Neither authenticates
// reports, persists state, starts a polling loop or owns provider shutdown.
package fleet

import (
	"fmt"
	"strings"
	"time"
)

const (
	// MaxIdentityBytes bounds worker status labels.
	MaxIdentityBytes = 256
	// MaxQueuesPerWorker bounds one heartbeat's queue cardinality.
	MaxQueuesPerWorker = 256
	// MaxCapabilitiesPerWorker bounds negotiated capability cardinality.
	MaxCapabilitiesPerWorker = 64
	// MaxWorkerConcurrency bounds reported goroutine concurrency.
	MaxWorkerConcurrency = 1_000_000
)

// State is the worker state reported to administrative consumers.
type State string

const (
	// StateRunning is a reported worker processing state, not an externally
	// verified health guarantee.
	StateRunning State = "running"
	// StatePaused is a reported admission pause; active work may remain.
	StatePaused State = "paused"
	// StateDraining is a reported drain in progress.
	StateDraining State = "draining"
	// StateStopped is a reported stopped worker.
	StateStopped State = "stopped"
	// StateStale is derived when observation is at or before the staleness
	// cutoff; workers may not report it.
	StateStale State = "stale"
	// StateUnknown is derived for missing/future/unsupported status or registry
	// conflicts; workers may not report it.
	StateUnknown State = "unknown"
)

// DrainState describes progress of a graceful drain request.
type DrainState string

const (
	// DrainNotRequested reports no requested drain.
	DrainNotRequested DrainState = "not_requested"
	// DrainRequested reports acceptance of a drain request.
	DrainRequested DrainState = "requested"
	// DrainInProgress reports active drain work.
	DrainInProgress DrainState = "in_progress"
	// DrainCompleted reports completed draining.
	DrainCompleted DrainState = "completed"
	// DrainTimedOut reports an expired drain.
	DrainTimedOut DrainState = "timed_out"
)

// Heartbeat is the latest status reported by a worker.
type Heartbeat struct {
	// TenantID is required bounded tenant scope; the reporting/authentication
	// boundary must bind it.
	TenantID string
	// WorkerID is required bounded worker identity, paired with TenantID in
	// Registry.
	WorkerID string
	// Version is required bounded implementation version text, not the
	// management Protocol.
	Version string
	// StartedAt is required worker start time and must not follow ObservedAt.
	StartedAt time.Time
	// ObservedAt is required observation time used for ordering and staleness;
	// source adapters choose its clock authority.
	ObservedAt time.Time
	// Queues contains at most MaxQueuesPerWorker bounded names; Registry copies
	// the slice.
	Queues []string
	// Concurrency must be positive and at most MaxWorkerConcurrency.
	Concurrency uint32
	// State must be a reported running/paused/draining/stopped state; derived
	// stale/unknown values are rejected.
	State State
	// CurrentJobs must not exceed Concurrency.
	CurrentJobs uint32
	// DrainStatus must be one supported reported drain state.
	DrainStatus DrainState
	// Backend is required bounded backend attribution.
	Backend string
	// Protocol must be a nonzero version pair; this is a report, not
	// conformance proof.
	Protocol ProtocolVersion
	// Capabilities contains at most MaxCapabilitiesPerWorker bounded capability
	// names; Registry copies the slice.
	Capabilities []Capability
}

// Validate rejects malformed or unbounded worker status reports.
func (h Heartbeat) Validate() error {
	identities := []struct {
		field string
		value string
	}{
		{field: "tenant_id", value: h.TenantID},
		{field: "worker_id", value: h.WorkerID},
		{field: "version", value: h.Version},
		{field: "backend", value: h.Backend},
	}
	for _, identity := range identities {
		if invalidHeartbeatIdentity(identity.value) {
			return invalidHeartbeat(identity.field, "is required and must be bounded")
		}
	}
	if h.ObservedAt.IsZero() {
		return invalidHeartbeat("observed_at", "is required")
	}
	if h.StartedAt.IsZero() || h.StartedAt.After(h.ObservedAt) {
		return invalidHeartbeat("started_at", "is required and cannot follow observation")
	}
	if len(h.Queues) > MaxQueuesPerWorker {
		return invalidHeartbeat("queues", "exceeds the worker queue limit")
	}
	for index, queue := range h.Queues {
		if invalidHeartbeatIdentity(queue) {
			return invalidHeartbeat(fmt.Sprintf("queues[%d]", index), "is invalid")
		}
	}
	if h.Concurrency == 0 || h.Concurrency > MaxWorkerConcurrency {
		return invalidHeartbeat("concurrency", "must be within the worker limit")
	}
	if h.CurrentJobs > h.Concurrency {
		return invalidHeartbeat("current_jobs", "cannot exceed concurrency")
	}
	if !h.State.reported() {
		return invalidHeartbeat("state", "is unsupported")
	}
	if !h.DrainStatus.valid() {
		return invalidHeartbeat("drain_status", "is unsupported")
	}
	if h.Protocol == (ProtocolVersion{}) {
		return invalidHeartbeat("protocol", "is required")
	}
	if len(h.Capabilities) > MaxCapabilitiesPerWorker {
		return invalidHeartbeat("capabilities", "exceeds the worker capability limit")
	}
	for index, capability := range h.Capabilities {
		if invalidHeartbeatIdentity(string(capability)) {
			return invalidHeartbeat(fmt.Sprintf("capabilities[%d]", index), "is invalid")
		}
	}

	return nil
}

// EffectiveState classifies observation time and reported state at now. Missing
// or future times and unsupported states are unknown; this method does not run
// full Heartbeat.Validate. Callers should supply a positive staleAfter.
func (h Heartbeat) EffectiveState(now time.Time, staleAfter time.Duration) State {
	if h.ObservedAt.IsZero() || h.ObservedAt.After(now) || !h.State.reported() {
		return StateUnknown
	}

	if !h.ObservedAt.After(now.Add(-staleAfter)) {
		return StateStale
	}

	return h.State
}

func (s State) reported() bool {
	switch s {
	case StateRunning, StatePaused, StateDraining, StateStopped:
		return true
	case StateStale, StateUnknown:
		return false
	default:
		return false
	}
}

func (s DrainState) valid() bool {
	switch s {
	case DrainNotRequested, DrainRequested, DrainInProgress, DrainCompleted, DrainTimedOut:
		return true
	default:
		return false
	}
}

func invalidHeartbeatIdentity(value string) bool {
	return strings.TrimSpace(value) == "" || len(value) > MaxIdentityBytes
}

// HeartbeatValidationError identifies one invalid worker-status field.
type HeartbeatValidationError struct {
	// Field names the invalid boundary and is printed by Error.
	Field string
	// Problem is printed by Error; producers should supply rule text, not
	// sensitive rejected values.
	Problem string
}

// Error prints the field and rule text, not the rejected heartbeat values.
func (e *HeartbeatValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Problem)
}

func invalidHeartbeat(field string, problem string) error {
	return &HeartbeatValidationError{Field: field, Problem: problem}
}
