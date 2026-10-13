// Package apihttp provides versioned administrative HTTP routes and public
// JSON models. NewHandler borrows explicit execution/read/permission providers;
// administrative routes require an authenticated context principal, while
// operational probes remain public. It does not obtain credentials itself;
// compose server.NewAdministrativeHandler for that boundary.
// Providers own scope, output bounds, cancellation and shared-use safety.
// Revealed record data requires separate permission and durable access-intent
// audit before reading; this does not prove that disclosure reached the client.
package apihttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"reflect"
	"time"

	authentication "github.com/faustbrian/go-authentication/v2"
	controlplane "github.com/faustbrian/go-queue-control-plane/v4"
	controlkubernetes "github.com/faustbrian/go-queue-control-plane/v4/adapters/kubernetes"
	"github.com/faustbrian/go-queue-control-plane/v4/authz"
	"github.com/faustbrian/go-queue-control-plane/v4/control"
	"github.com/faustbrian/go-queue-control-plane/v4/fleet"
	controlpostgres "github.com/faustbrian/go-queue-control-plane/v4/postgres"
	queue "github.com/faustbrian/go-queue/management"
	telemetryhttp "github.com/faustbrian/go-telemetry/v2/instrumentation/nethttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const defaultMaxRequestBytes int64 = 1 << 20

// ErrInvalidConfiguration reports an invalid handler graph or request setting.
var ErrInvalidConfiguration = errors.New("apihttp: invalid configuration")

// CommandExecutor runs an authenticated administrative command.
type CommandExecutor interface {
	// Execute must validate, authorize and durably admit the scoped command;
	// errors can coexist with committed or partial external work.
	Execute(context.Context, controlplane.Command) (controlplane.CommandResult, error)
}

// Readiness checks dependencies without changing process liveness.
type Readiness interface {
	// Ready checks borrowed dependencies using ctx without performing
	// mutations.
	Ready(context.Context) error
}

// WorkerSource provides bounded tenant-scoped worker snapshots.
type WorkerSource interface {
	// SnapshotTenant returns a bounded, tenant-scoped independently owned
	// snapshot; this synchronous interface has no context parameter.
	SnapshotTenant(string, time.Time, time.Duration) fleet.RegistrySnapshot
}

// RemoteWorkerSource provides cancellable tenant-scoped worker snapshots.
type RemoteWorkerSource interface {
	// SnapshotTenant honors ctx and returns an independently owned bounded
	// tenant snapshot; errors must not masquerade as an empty healthy fleet.
	SnapshotTenant(
		context.Context,
		string,
		time.Time,
		time.Duration,
	) (fleet.RegistrySnapshot, error)
}

// WorkloadSource provides bounded tenant-scoped Kubernetes visibility.
type WorkloadSource interface {
	// ListTenantWorkloads returns the requested bounded namespace-scoped page;
	// providers must honor pagination and scope.
	ListTenantWorkloads(context.Context, string, int64, string) (controlkubernetes.Page, error)
}

// Viewer authorizes diagnostic reads against tenant resources.
type Viewer interface {
	// Authorize must fail closed for the authenticated actor, tenant,
	// permission and resource before source access.
	Authorize(
		context.Context,
		string,
		string,
		controlplane.Permission,
		controlplane.Target,
	) error
}

// AuditSource reads bounded tenant audit-history pages.
type AuditSource interface {
	// ListTenant returns a bounded ordered tenant audit page after the
	// exclusive sequence cursor; source integrity checking remains provider-owned.
	ListTenant(context.Context, string, uint64, uint32) (controlpostgres.AuditPage, error)
}

// SensitiveAccessAuditor durably records privileged record reads.
type SensitiveAccessAuditor interface {
	// AuditSensitiveAccess must persist the privileged-access intent before
	// record reading; failure prevents disclosure but success is not proof that
	// bytes reached a client.
	AuditSensitiveAccess(context.Context, controlplane.SensitiveAccess) error
}

// CommandResultSource reads tenant-scoped durable command outcomes.
type CommandResultSource interface {
	// Get returns the durable outcome for the tenant/idempotency key;
	// implementations own validation and diagnostic safety.
	Get(context.Context, string, string) (controlplane.CommandResult, error)
}

// RecordSource reads tenant-scoped queue failures and dead letters.
type RecordSource interface {
	// ListFailures returns bounded tenant failure metadata; the HTTP projection
	// hides payload and diagnostics.
	ListFailures(context.Context, string, queue.PageRequest) (queue.RecordPage, error)
	// ListDeadLetters returns bounded tenant dead-letter metadata without
	// exceeding the requested page.
	ListDeadLetters(context.Context, string, queue.PageRequest) (queue.RecordPage, error)
	// Inspect must return the exact scoped record at no greater payload
	// visibility than requested; diagnostic authorization remains a separate
	// HTTP boundary.
	Inspect(context.Context, string, queue.InspectRequest) (queue.JobRecord, error)
}

// QueueSource reads tenant-scoped queue status pages.
type QueueSource interface {
	// ListQueues returns a validated bounded tenant queue page with honest
	// supported/unsupported measurements.
	ListQueues(context.Context, string, queue.StatusPageRequest) (queue.QueueStatusPage, error)
}

// DesiredStateSource reads one tenant-scoped durable convergence record.
type DesiredStateSource interface {
	// Get returns current durable state for the exact scoped target, not proof
	// of worker convergence.
	Get(context.Context, string, controlplane.Target) (control.DesiredRecord, error)
}

type commandHistorySource interface {
	ListTenant(context.Context, string, string, uint32) (controlpostgres.CommandPage, error)
}

// BuildInfo is immutable release metadata exposed to automation.
type BuildInfo struct {
	// Version is configured public version text, not a verified release
	// attestation.
	Version string `json:"version"`
	// Commit is configured public source identity text.
	Commit string `json:"commit"`
	// BuiltAt is normalized to UTC when the handler is constructed.
	BuiltAt time.Time `json:"built_at"`
}

// Config defines bounded dependencies for the administrative API.
type Config struct {
	// Commands is the required borrowed mutation executor; the handler does not
	// own its shutdown.
	Commands CommandExecutor
	// MaxRequestBytes bounds JSON mutation input through MaxBytesReader; zero
	// selects 1 MiB and negative values fail.
	MaxRequestBytes int64
	// Readiness is optional borrowed dependency health checking; nil reports
	// ready without checking external services.
	Readiness Readiness
	// Build is copied public startup metadata; it is not cryptographically
	// authenticated.
	Build BuildInfo
	// Capabilities is copied public capability text; advertisement does not
	// grant permission.
	Capabilities []string
	// Workers is optional synchronous fleet state, mutually exclusive with
	// RemoteWorkers; read sources require Viewer.
	Workers WorkerSource
	// RemoteWorkers is optional cancellable remote fleet state, mutually
	// exclusive with Workers.
	RemoteWorkers RemoteWorkerSource
	// Workloads is optional borrowed workload visibility; the source owns
	// provider scope and page bounds.
	Workloads WorkloadSource
	// Viewer is required exactly when ordinary read sources are configured; it
	// must authorize before source access.
	Viewer Viewer
	// Now supplies fleet/sensitive-audit observation time; nil selects time.Now
	// and the callback remains borrowed.
	Now func() time.Time
	// NewCommandID supplies privileged-access audit identities; nil selects
	// controlplane.NewCommandID. Commands are identified by their executor.
	NewCommandID func() (string, error)
	// WorkflowLimiter optionally admits subject/action and sensitive-read
	// workflows synchronously; nil disables this layer.
	WorkflowLimiter RateLimiter
	// StaleAfter is the fleet staleness duration; zero selects thirty seconds
	// and negative values fail.
	StaleAfter time.Duration
	// Protocol selects the advertised inclusive control-plane version range
	// used for negotiation.
	Protocol fleet.ProtocolRange
	// WorkerCapabilities is copied negotiation input, not worker authentication
	// or authorization.
	WorkerCapabilities []fleet.Capability
	// Telemetry optionally supplies borrowed HTTP instrumentation providers;
	// the handler does not shut them down.
	Telemetry *TelemetryConfig
	// Audit is an optional borrowed tenant audit-history source.
	Audit AuditSource
	// SensitiveAudit is required before revealed payload/diagnostic access; nil
	// causes audit_unavailable, not unaudited disclosure.
	SensitiveAudit SensitiveAccessAuditor
	// CommandResults is optional durable lookup; implementations also exposing
	// ListTenant enable command-history pagination.
	CommandResults CommandResultSource
	// Records is optional scoped failure/dead-letter reading; implementations
	// must preserve bounds and disclosure contracts.
	Records RecordSource
	// Queues is optional scoped queue observation; unsupported measurements
	// must remain distinguishable from zero.
	Queues QueueSource
	// DesiredState is optional durable convergence-state reading, separate from
	// worker enforcement.
	DesiredState DesiredStateSource
}

// TelemetryConfig supplies standard OpenTelemetry APIs owned by telemetry.
type TelemetryConfig struct {
	// TracerProvider is a borrowed tracing provider; application composition
	// owns processor/exporter shutdown.
	TracerProvider trace.TracerProvider
	// MeterProvider is a borrowed metrics provider used by HTTP
	// instrumentation.
	MeterProvider metric.MeterProvider
	// Propagator is borrowed propagation behavior; trace context is not
	// authentication.
	Propagator propagation.TextMapPropagator
	// TrustedInbound opts into inbound context trust only for requests carrying
	// a non-anonymous authenticated principal.
	TrustedInbound bool
}

// Problem is the stable secret-safe API error envelope.
type Problem struct {
	// Code is a stable public error category selected by the handler, never a
	// raw backend diagnostic.
	Code string `json:"code"`
}

type handler struct {
	commands           CommandExecutor
	maxRequestBytes    int64
	readiness          Readiness
	build              BuildInfo
	capabilities       []string
	workers            WorkerSource
	remoteWorkers      RemoteWorkerSource
	workloads          WorkloadSource
	viewer             Viewer
	now                func() time.Time
	newCommandID       func() (string, error)
	workflowLimiter    RateLimiter
	staleAfter         time.Duration
	protocol           fleet.ProtocolRange
	workerCapabilities []fleet.Capability
	audit              AuditSource
	sensitiveAudit     SensitiveAccessAuditor
	commandResults     CommandResultSource
	commandHistory     commandHistorySource
	records            RecordSource
	queues             QueueSource
	desiredState       DesiredStateSource
}

// CommandRequest is the actor-free JSON mutation accepted by the API.
type CommandRequest struct {
	// IdempotencyKey is required scoped duplicate/conflict identity; actor and
	// tenant come from trusted ingress, not this body.
	IdempotencyKey string `json:"idempotency_key"`
	// Reason is required bounded audit text; callers must omit credentials and
	// sensitive payloads.
	Reason string `json:"reason"`
	// Action selects the requested supported administrative mutation.
	Action controlplane.Action `json:"action"`
	// Target identifies the action-compatible resource without backend
	// addressing.
	Target TargetRequest `json:"target"`
	// RequestedAt is required caller time anchoring the enforcement window, not
	// the server receipt time.
	RequestedAt time.Time `json:"requested_at"`
	// Confirmed is required for destructive bulk/purge/replay and scale-to-zero
	// operations.
	Confirmed bool `json:"confirmed"`
	// Selection is required for bulk retry and is copied into the domain
	// command.
	Selection *SelectionRequest `json:"selection,omitempty"`
	// Replay is required for replay with explicit destination and duplicate
	// policy.
	Replay *ReplayRequest `json:"replay,omitempty"`
	// Scale is required for scaling, including an explicit zero replica count.
	Scale *ScaleRequest `json:"scale,omitempty"`
}

// TargetRequest identifies a command target without backend addressing.
type TargetRequest struct {
	// Kind selects a supported resource class compatible with the action.
	Kind controlplane.TargetKind `json:"kind"`
	// Name is required bounded identity text; it is not a backend endpoint.
	Name string `json:"name"`
}

// SelectionRequest bounds a bulk administrative mutation.
type SelectionRequest struct {
	// Limit must be positive and at most controlplane.MaxBulkSelection.
	Limit uint32 `json:"limit"`
}

// ReplayRequest declares explicit replay destination and idempotency policy.
type ReplayRequest struct {
	// Destination is the explicit destination separately authorized by the
	// command service.
	Destination string `json:"destination"`
	// IdempotencyPolicy selects rejection or replacement of destination
	// duplicates.
	IdempotencyPolicy controlplane.ReplayPolicy `json:"idempotency_policy"`
}

// ScaleRequest declares the desired workload replica count.
type ScaleRequest struct {
	// Replicas is at most controlplane.MaxScaleReplicas; zero additionally
	// requires confirmation.
	Replicas uint32 `json:"replicas"`
}

// NewHandler creates the versioned administrative HTTP handler.
func NewHandler(config Config) (http.Handler, error) {
	if nilInterface(config.Commands) || config.MaxRequestBytes < 0 || config.StaleAfter < 0 ||
		(config.Workers != nil && nilInterface(config.Workers)) ||
		(config.RemoteWorkers != nil && nilInterface(config.RemoteWorkers)) ||
		(config.Workers != nil && config.RemoteWorkers != nil) ||
		(config.Workloads != nil && nilInterface(config.Workloads)) ||
		(config.Audit != nil && nilInterface(config.Audit)) ||
		(config.SensitiveAudit != nil && nilInterface(config.SensitiveAudit)) ||
		(config.CommandResults != nil && nilInterface(config.CommandResults)) ||
		(config.Records != nil && nilInterface(config.Records)) ||
		(config.Queues != nil && nilInterface(config.Queues)) ||
		(config.DesiredState != nil && nilInterface(config.DesiredState)) ||
		(config.Viewer != nil && nilInterface(config.Viewer)) ||
		(config.WorkflowLimiter != nil && nilInterface(config.WorkflowLimiter)) ||
		(config.Workers == nil && config.RemoteWorkers == nil &&
			config.Workloads == nil && config.Audit == nil &&
			config.CommandResults == nil && config.Records == nil &&
			config.Queues == nil && config.DesiredState == nil) != (config.Viewer == nil) {
		return nil, ErrInvalidConfiguration
	}
	if config.MaxRequestBytes == 0 {
		config.MaxRequestBytes = defaultMaxRequestBytes
	}
	config.Build.BuiltAt = config.Build.BuiltAt.UTC()
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.NewCommandID == nil {
		config.NewCommandID = controlplane.NewCommandID
	}
	if config.StaleAfter == 0 {
		config.StaleAfter = 30 * time.Second
	}

	var commandHistory commandHistorySource
	if source, ok := config.CommandResults.(commandHistorySource); ok {
		commandHistory = source
	}
	handler := &handler{
		commands:           config.Commands,
		maxRequestBytes:    config.MaxRequestBytes,
		readiness:          config.Readiness,
		build:              config.Build,
		capabilities:       append([]string(nil), config.Capabilities...),
		workers:            config.Workers,
		remoteWorkers:      config.RemoteWorkers,
		workloads:          config.Workloads,
		viewer:             config.Viewer,
		now:                config.Now,
		newCommandID:       config.NewCommandID,
		workflowLimiter:    config.WorkflowLimiter,
		staleAfter:         config.StaleAfter,
		protocol:           config.Protocol,
		workerCapabilities: append([]fleet.Capability(nil), config.WorkerCapabilities...),
		audit:              config.Audit,
		sensitiveAudit:     config.SensitiveAudit,
		commandResults:     config.CommandResults,
		commandHistory:     commandHistory,
		records:            config.Records,
		queues:             config.Queues,
		desiredState:       config.DesiredState,
	}
	router := http.NewServeMux()
	router.HandleFunc("GET /health/live", handler.liveness)
	router.HandleFunc("GET /health/ready", handler.ready)
	router.HandleFunc("GET /version", handler.version)
	router.HandleFunc("GET /v1/capabilities", handler.listCapabilities)
	router.HandleFunc("POST /v1/tenants/{tenant}/commands", handler.executeCommand)
	if handler.workers != nil || handler.remoteWorkers != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/workers", handler.listWorkers)
	}
	if handler.workloads != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/workloads", handler.listWorkloads)
	}
	if handler.audit != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/audit", handler.listAudit)
	}
	if handler.commandHistory != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/commands", handler.listCommandHistory)
	}
	if handler.commandResults != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/commands/{key}", handler.getCommandResult)
	}
	if handler.records != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/failures", handler.listFailures)
		router.HandleFunc("GET /v1/tenants/{tenant}/dead-letters", handler.listDeadLetters)
		router.HandleFunc("GET /v1/tenants/{tenant}/failures/{record}", handler.inspectFailure)
		router.HandleFunc("GET /v1/tenants/{tenant}/dead-letters/{record}", handler.inspectDeadLetter)
	}
	if handler.queues != nil {
		router.HandleFunc("GET /v1/tenants/{tenant}/queues", handler.listQueues)
	}
	if handler.desiredState != nil {
		router.HandleFunc(
			"GET /v1/tenants/{tenant}/desired-state/{kind}/{name}",
			handler.getDesiredState,
		)
	}

	if config.Telemetry == nil {
		return router, nil
	}

	return telemetryhttp.NewHandler(router, telemetryhttp.ServerConfig{
		Operation:      "queue_control_plane.http.server",
		TrustedInbound: config.Telemetry.TrustedInbound,
		TrustInbound: func(request *http.Request) bool {
			principal, ok := authentication.PrincipalFromContext(request.Context())
			return ok && !principal.IsAnonymous()
		},
		TracerProvider: config.Telemetry.TracerProvider,
		MeterProvider:  config.Telemetry.MeterProvider,
		Propagator:     config.Telemetry.Propagator,
	})
}

func (h *handler) liveness(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "live"})
}

func (h *handler) ready(writer http.ResponseWriter, request *http.Request) {
	if h.readiness != nil {
		if err := h.readiness.Ready(request.Context()); err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
	}

	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *handler) version(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, h.build)
}

func (h *handler) listCapabilities(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, struct {
		Capabilities []string `json:"capabilities"`
	}{Capabilities: append([]string(nil), h.capabilities...)})
}

func (h *handler) executeCommand(writer http.ResponseWriter, request *http.Request) {
	principal, _ := authentication.PrincipalFromContext(request.Context())
	if principal.IsAnonymous() {
		writeProblem(writer, http.StatusUnauthorized, "unauthenticated")
		return
	}

	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeProblem(writer, http.StatusUnsupportedMediaType, "unsupported_media_type")
		return
	}

	var input CommandRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, h.maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeDecodeProblem(writer, err)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}

	command := input.command(
		request.PathValue("tenant"), principal.Subject(), principal.Method(),
	)
	if h.workflowLimiter != nil && !h.workflowLimiter.Allow(
		request.Context(), workflowRateLimitKey(principal.Subject(), string(command.Action)),
	) {
		writer.Header().Set("Retry-After", "1")
		writeProblem(writer, http.StatusTooManyRequests, "rate_limited")
		return
	}
	result, err := h.commands.Execute(request.Context(), command)
	if err != nil {
		writeCommandError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, result)
}

func workflowRateLimitKey(subject string, operation string) string {
	return "subject:" + subject + "|workflow:" + operation
}

func (r CommandRequest) command(
	tenant string,
	actor string,
	authenticationMethod string,
) controlplane.Command {
	command := controlplane.Command{
		IdempotencyKey:       r.IdempotencyKey,
		TenantID:             tenant,
		Actor:                actor,
		AuthenticationMethod: authenticationMethod,
		Reason:               r.Reason,
		Action:               r.Action,
		Target: controlplane.Target{
			Kind: r.Target.Kind,
			Name: r.Target.Name,
		},
		RequestedAt: r.RequestedAt,
		Confirmed:   r.Confirmed,
	}
	if r.Selection != nil {
		command.Selection = &controlplane.Selection{Limit: r.Selection.Limit}
	}
	if r.Replay != nil {
		command.Replay = &controlplane.Replay{
			Destination:       r.Replay.Destination,
			IdempotencyPolicy: r.Replay.IdempotencyPolicy,
		}
	}
	if r.Scale != nil {
		command.Scale = &controlplane.Scale{Replicas: r.Scale.Replicas}
	}

	return command
}

func writeDecodeProblem(writer http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeProblem(writer, http.StatusRequestEntityTooLarge, "request_too_large")
		return
	}

	writeProblem(writer, http.StatusBadRequest, "invalid_request")
}

func writeCommandError(writer http.ResponseWriter, err error) {
	var validationError *controlplane.ValidationError
	switch {
	case errors.As(err, &validationError):
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, controlpostgres.ErrInvalidCommandRequest):
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, authz.ErrUnauthenticated):
		writeProblem(writer, http.StatusUnauthorized, "unauthenticated")
	case errors.Is(err, authz.ErrDenied), errors.Is(err, authz.ErrActorMismatch):
		writeProblem(writer, http.StatusForbidden, "forbidden")
	case errors.Is(err, controlpostgres.ErrIdempotencyConflict):
		writeProblem(writer, http.StatusConflict, "idempotency_conflict")
	case errors.Is(err, controlpostgres.ErrCommandNotFound):
		writeProblem(writer, http.StatusNotFound, "command_not_found")
	case errors.Is(err, control.ErrOutcomeUnknown):
		writeProblem(writer, http.StatusServiceUnavailable, "outcome_unknown")
	default:
		writeProblem(writer, http.StatusInternalServerError, "internal_error")
	}
}

func writeProblem(writer http.ResponseWriter, status int, code string) {
	writeJSON(writer, status, Problem{Code: code})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Pointer && reflected.IsNil()
}
