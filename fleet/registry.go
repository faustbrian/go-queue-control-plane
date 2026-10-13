package fleet

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"
)

// HeartbeatDisposition describes how the registry handled one report.
type HeartbeatDisposition string

const (
	// HeartbeatAccepted retains a copied new or later validated heartbeat.
	HeartbeatAccepted HeartbeatDisposition = "accepted"
	// HeartbeatDuplicate leaves an identical same-time report unchanged.
	HeartbeatDuplicate HeartbeatDisposition = "duplicate"
	// HeartbeatReordered ignores an older observation.
	HeartbeatReordered HeartbeatDisposition = "reordered"
	// HeartbeatConflict marks same-time conflicting reports unknown until a
	// later valid report arrives.
	HeartbeatConflict HeartbeatDisposition = "conflict"
	// HeartbeatInvalid refuses a malformed report without retaining it.
	HeartbeatInvalid HeartbeatDisposition = "invalid"
	// HeartbeatCapacityExceeded refuses a new worker identity at capacity and
	// increments the tenant rejection counter.
	HeartbeatCapacityExceeded HeartbeatDisposition = "capacity_exceeded"
)

// WorkerSnapshot contains a defensively copied heartbeat and its derived
// fail-safe state.
type WorkerSnapshot struct {
	Heartbeat
	// State is derived liveness/conflict state, shadowing the embedded reported
	// State.
	State State
}

// RegistrySnapshot is a deterministic point-in-time view of retained workers.
type RegistrySnapshot struct {
	// Workers is a newly owned tenant/worker-sorted slice with copied
	// queue/capability slices.
	Workers []WorkerSnapshot
	// Rejected counts capacity refusals in the selected tenant or whole-registry view.
	Rejected uint64
}

// Registry synchronizes a bounded set of latest copied worker heartbeats.
// Construct with NewRegistry and do not copy after use. There is no eviction or
// close operation. Worker cardinality is bounded, but rejection-counter tenant
// cardinality is not: this is not a total memory bound for arbitrary tenants.
type Registry struct {
	mu         sync.RWMutex
	maxWorkers int
	workers    map[workerKey]workerRecord
	rejected   map[string]uint64
}

type workerKey struct {
	tenant string
	worker string
}

type workerRecord struct {
	heartbeat  Heartbeat
	conflicted bool
}

// NewRegistry creates a registry that retains at most maxWorkers identities.
func NewRegistry(maxWorkers int) *Registry {
	maxWorkers = max(maxWorkers, 0)

	return &Registry{
		maxWorkers: maxWorkers,
		workers:    make(map[workerKey]workerRecord, maxWorkers),
		rejected:   make(map[string]uint64),
	}
}

// Upsert ingests one heartbeat without allowing delayed, duplicate, or
// conflicting reports to falsely advance worker state.
func (r *Registry) Upsert(heartbeat Heartbeat) HeartbeatDisposition {
	if strings.TrimSpace(heartbeat.TenantID) == "" || strings.TrimSpace(heartbeat.WorkerID) == "" {
		return HeartbeatInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := workerKey{tenant: heartbeat.TenantID, worker: heartbeat.WorkerID}
	record, exists := r.workers[key]
	if !exists && len(r.workers) >= r.maxWorkers {
		r.rejected[heartbeat.TenantID]++

		return HeartbeatCapacityExceeded
	}
	if err := heartbeat.Validate(); err != nil {
		return HeartbeatInvalid
	}
	if !exists {
		r.workers[key] = workerRecord{heartbeat: cloneHeartbeat(heartbeat)}

		return HeartbeatAccepted
	}

	if heartbeat.ObservedAt.Before(record.heartbeat.ObservedAt) {
		return HeartbeatReordered
	}
	if heartbeat.ObservedAt.Equal(record.heartbeat.ObservedAt) {
		if reflect.DeepEqual(heartbeat, record.heartbeat) {
			return HeartbeatDuplicate
		}

		record.conflicted = true
		r.workers[key] = record

		return HeartbeatConflict
	}

	r.workers[key] = workerRecord{heartbeat: cloneHeartbeat(heartbeat)}

	return HeartbeatAccepted
}

// Snapshot returns sorted workers with liveness derived at now.
func (r *Registry) Snapshot(now time.Time, staleAfter time.Duration) RegistrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := r.snapshotLocked("", now, staleAfter)
	for _, rejected := range r.rejected {
		snapshot.Rejected += rejected
	}

	return snapshot
}

// SnapshotTenant returns only one tenant's workers and rejection count.
func (r *Registry) SnapshotTenant(
	tenant string,
	now time.Time,
	staleAfter time.Duration,
) RegistrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := r.snapshotLocked(tenant, now, staleAfter)
	snapshot.Rejected = r.rejected[tenant]

	return snapshot
}

func (r *Registry) snapshotLocked(tenant string, now time.Time, staleAfter time.Duration) RegistrySnapshot {
	snapshot := RegistrySnapshot{
		Workers: make([]WorkerSnapshot, 0, len(r.workers)),
	}
	for _, record := range r.workers {
		if tenant != "" && record.heartbeat.TenantID != tenant {
			continue
		}
		state := record.heartbeat.EffectiveState(now, staleAfter)
		if record.conflicted {
			state = StateUnknown
		}
		snapshot.Workers = append(snapshot.Workers, WorkerSnapshot{
			Heartbeat: cloneHeartbeat(record.heartbeat),
			State:     state,
		})
	}
	slices.SortFunc(snapshot.Workers, func(left, right WorkerSnapshot) int {
		return cmp.Or(
			cmp.Compare(left.TenantID, right.TenantID),
			cmp.Compare(left.WorkerID, right.WorkerID),
		)
	})

	return snapshot
}

func cloneHeartbeat(heartbeat Heartbeat) Heartbeat {
	heartbeat.Queues = append([]string(nil), heartbeat.Queues...)
	heartbeat.Capabilities = append([]Capability(nil), heartbeat.Capabilities...)

	return heartbeat
}
