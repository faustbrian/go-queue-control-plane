package apihttp

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	authentication "github.com/faustbrian/go-authentication/v2"
	controlplane "github.com/faustbrian/go-queue-control-plane/v4"
	"github.com/faustbrian/go-queue-control-plane/v4/fleet"
)

const (
	defaultWorkerPageSize uint32 = 100
	// MaxWorkerPageSize bounds the public response page, not the provider's
	// entire snapshot allocation.
	MaxWorkerPageSize uint32 = 1_000
)

// WorkerPage is one deterministic bounded worker result page.
type WorkerPage struct {
	// Workers contains sorted/filtered worker projections; the source snapshot
	// is materialized before pagination.
	Workers []Worker `json:"workers"`
	// Rejected is the tenant count only when RejectedKnown is true.
	Rejected uint64 `json:"rejected"`
	// RejectedKnown distinguishes exact registered-tenant counts from unknown
	// detail, including providers that do not expose rejection accounting.
	RejectedKnown bool `json:"rejected_known"`
	// NextCursor is the exclusive last worker identity for another page; empty
	// means no further page was detected.
	NextCursor string `json:"next_cursor,omitempty"`
}

// Worker is the public tenant-scoped worker status representation.
type Worker struct {
	// TenantID is the source's scoped tenant attribution.
	TenantID string `json:"tenant_id"`
	// WorkerID is the worker identity used in deterministic pagination.
	WorkerID string `json:"worker_id"`
	// Version is reported implementation version text.
	Version string `json:"version"`
	// StartedAt is reported worker start time.
	StartedAt time.Time `json:"started_at"`
	// ObservedAt is the heartbeat observation used for staleness, not
	// necessarily request receipt time.
	ObservedAt time.Time `json:"observed_at"`
	// Queues borrows the independently owned source snapshot's queue slice
	// during HTTP encoding.
	Queues []string `json:"queues"`
	// Concurrency is reported configured worker concurrency.
	Concurrency uint32 `json:"concurrency"`
	// State is derived liveness/conflict state, not a verified process
	// supervision result.
	State fleet.State `json:"state"`
	// CurrentJobs is reported in-flight job count.
	CurrentJobs uint32 `json:"current_jobs"`
	// DrainStatus is reported drain progress.
	DrainStatus fleet.DrainState `json:"drain_status"`
	// Backend is reported backend attribution.
	Backend string `json:"backend"`
	// Protocol is the reported management version pair.
	Protocol fleet.ProtocolVersion `json:"protocol"`
	// Capabilities borrows the independently owned snapshot's reported
	// capabilities during encoding.
	Capabilities []fleet.Capability `json:"capabilities"`
	// Compatibility is newly computed version/capability negotiation, not
	// authorization.
	Compatibility fleet.Compatibility `json:"compatibility"`
}

type workerQuery struct {
	limit uint32
	after string
	state fleet.State
	queue string
}

func (h *handler) listWorkers(writer http.ResponseWriter, request *http.Request) {
	principal, _ := authentication.PrincipalFromContext(request.Context())
	if principal.IsAnonymous() {
		writeProblem(writer, http.StatusUnauthorized, "unauthenticated")
		return
	}
	tenant := request.PathValue("tenant")
	if !validIdentity(tenant) {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.viewer.Authorize(
		request.Context(),
		tenant,
		principal.Subject(),
		controlplane.PermissionView,
		controlplane.Target{Kind: controlplane.TargetWorkload, Name: "fleet"},
	); err != nil {
		writeCommandError(writer, err)
		return
	}

	query, err := parseWorkerQuery(request.URL.Query())
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	var snapshot fleet.RegistrySnapshot
	if h.remoteWorkers != nil {
		snapshot, err = h.remoteWorkers.SnapshotTenant(
			request.Context(), tenant, h.now(), h.staleAfter,
		)
		if err != nil {
			writeProblem(writer, http.StatusServiceUnavailable, "worker_status_unavailable")
			return
		}
	} else {
		snapshot = h.workers.SnapshotTenant(tenant, h.now(), h.staleAfter)
	}
	workers := append([]fleet.WorkerSnapshot(nil), snapshot.Workers...)
	sort.SliceStable(workers, func(i, j int) bool {
		return workerComesBefore(workers[i], workers[j])
	})

	page := WorkerPage{Workers: make([]Worker, 0, len(workers)), Rejected: snapshot.Rejected, RejectedKnown: snapshot.RejectedKnown}
	for _, snapshot := range workers {
		if !query.matches(snapshot) {
			continue
		}
		page.Workers = append(page.Workers, h.worker(snapshot))
	}
	if len(page.Workers) > int(query.limit) {
		page.Workers = page.Workers[:query.limit]
		page.NextCursor = page.Workers[len(page.Workers)-1].WorkerID
	}

	writeJSON(writer, http.StatusOK, page)
}

func workerComesBefore(left, right fleet.WorkerSnapshot) bool {
	return left.WorkerID < right.WorkerID
}

func (h *handler) worker(snapshot fleet.WorkerSnapshot) Worker {
	// The snapshot owns defensive copies. This private projection only reads
	// them until the response is synchronously encoded by listWorkers.
	queues := snapshot.Queues
	if len(queues) == 0 {
		queues = nil
	}
	capabilities := snapshot.Capabilities
	if len(capabilities) == 0 {
		capabilities = nil
	}

	return Worker{
		TenantID:     snapshot.TenantID,
		WorkerID:     snapshot.WorkerID,
		Version:      snapshot.Version,
		StartedAt:    snapshot.StartedAt,
		ObservedAt:   snapshot.ObservedAt,
		Queues:       queues,
		Concurrency:  snapshot.Concurrency,
		State:        snapshot.State,
		CurrentJobs:  snapshot.CurrentJobs,
		DrainStatus:  snapshot.DrainStatus,
		Backend:      snapshot.Backend,
		Protocol:     snapshot.Protocol,
		Capabilities: capabilities,
		Compatibility: fleet.Negotiate(
			h.protocol,
			snapshot.Protocol,
			snapshot.Capabilities,
			h.workerCapabilities,
		),
	}
}

func parseWorkerQuery(values url.Values) (workerQuery, error) {
	query := workerQuery{limit: defaultWorkerPageSize}
	allowed := map[string]bool{"limit": true, "after": true, "state": true, "queue": true}
	for key, entries := range values {
		if !allowed[key] || len(entries) != 1 {
			return workerQuery{}, ErrInvalidConfiguration
		}
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return workerQuery{}, ErrInvalidConfiguration
		}
		if limit == 0 || limit > uint64(MaxWorkerPageSize) {
			return workerQuery{}, ErrInvalidConfiguration
		}
		query.limit = uint32(limit)
	}
	query.after = values.Get("after")
	if _, exists := values["after"]; exists && strings.TrimSpace(query.after) == "" {
		return workerQuery{}, ErrInvalidConfiguration
	}
	query.queue = values.Get("queue")
	if _, exists := values["queue"]; exists && strings.TrimSpace(query.queue) == "" {
		return workerQuery{}, ErrInvalidConfiguration
	}
	if raw := values.Get("state"); raw != "" {
		query.state = fleet.State(raw)
		if !publicWorkerState(query.state) {
			return workerQuery{}, ErrInvalidConfiguration
		}
	} else if _, exists := values["state"]; exists {
		return workerQuery{}, ErrInvalidConfiguration
	}

	return query, nil
}

func (q workerQuery) matches(snapshot fleet.WorkerSnapshot) bool {
	if snapshot.WorkerID <= q.after || (q.state != "" && snapshot.State != q.state) {
		return false
	}
	if q.queue == "" {
		return true
	}
	for _, queue := range snapshot.Queues {
		if queue == q.queue {
			return true
		}
	}

	return false
}

func publicWorkerState(state fleet.State) bool {
	switch state {
	case fleet.StateRunning,
		fleet.StatePaused,
		fleet.StateDraining,
		fleet.StateStopped,
		fleet.StateStale,
		fleet.StateUnknown:
		return true
	default:
		return false
	}
}

func validIdentity(value string) bool {
	return strings.TrimSpace(value) != "" && len(value) <= controlplane.MaxIdentityBytes
}
