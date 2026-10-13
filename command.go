// Package controlplane defines administrative commands, permissions and result
// validation for the HTTP/CLI/control-service boundaries. Start with Command
// and its Validate method; it validates shape, not authentication or permission.
// Optional PostgreSQL, queue-management and Kubernetes composition own their
// external resources. Domain values containing pointers are not automatically
// immutable or safe to mutate while a borrowed operation is in progress.
package controlplane

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"

	identifierulid "github.com/faustbrian/go-identifier/ulid"
)

// NewCommandID allocates one opaque lowercase ULID operation identifier.
func NewCommandID() (string, error) {
	return newCommandID(rand.Reader)
}

func newCommandID(reader io.Reader) (string, error) {
	identifier, err := identifierulid.NewGenerator(nil, reader).New()
	if err != nil {
		return "", err
	}

	return identifier.StringLower(), nil
}

// Action identifies an administrative mutation.
type Action string

const (
	// ActionPause requests a queue or worker-group admission pause; it does
	// not join active jobs.
	ActionPause Action = "pause"
	// ActionResume requests resumption for a queue or worker group.
	ActionResume Action = "resume"
	// ActionDrain requests worker or worker-group draining without immediate
	// termination.
	ActionDrain Action = "drain"
	// ActionTerminate requests worker or worker-group termination.
	ActionTerminate Action = "terminate"
	// ActionRetry retries one failure or dead-letter record.
	ActionRetry Action = "retry"
	// ActionBulkRetry retries a confirmed bounded record selection.
	ActionBulkRetry Action = "bulk_retry"
	// ActionDelete deletes one failure or dead-letter record.
	ActionDelete Action = "delete"
	// ActionPurge purges a confirmed queue or record target.
	ActionPurge Action = "purge"
	// ActionReplay replays a confirmed record into an explicitly authorized
	// destination.
	ActionReplay Action = "replay"
	// ActionScale requests a bounded Kubernetes workload replica count.
	ActionScale Action = "scale"
)

const (
	// MaxIdentityBytes matches the durable and data-plane identity bounds.
	MaxIdentityBytes = 256
	// MaxReasonBytes bounds actor-supplied administrative audit reasons.
	MaxReasonBytes = 1_024
	// MaxFailureBytes bounds stable public command failure codes.
	MaxFailureBytes = 256
)

// TargetKind identifies the resource affected by a command.
type TargetKind string

const (
	// TargetQueue identifies a tenant-scoped queue.
	TargetQueue TargetKind = "queue"
	// TargetWorker identifies one tenant-scoped worker.
	TargetWorker TargetKind = "worker"
	// TargetWorkerGroup identifies a tenant-scoped worker group.
	TargetWorkerGroup TargetKind = "worker_group"
	// TargetFailure identifies a failure record or its supported administrative
	// selection.
	TargetFailure TargetKind = "failure"
	// TargetDeadLetter identifies a dead-letter record or its supported
	// administrative selection.
	TargetDeadLetter TargetKind = "dead_letter"
	// TargetWorkload identifies a Kubernetes workload, not a queue worker
	// process.
	TargetWorkload TargetKind = "workload"
)

// Target identifies an administrative resource without exposing backend
// addressing or queue serialization details.
type Target struct {
	// Kind selects the resource class; supported classes depend on the action.
	Kind TargetKind
	// Name is required nonblank identity text bounded by MaxIdentityBytes;
	// validation does not trim it.
	Name string
}

// Command is the mandatory envelope for every administrative mutation.
type Command struct {
	// CommandID is an optional bounded operation identity; Service/Journal
	// generate it when absent.
	CommandID string
	// IdempotencyKey is the required tenant-scoped duplicate/conflict key, not
	// authorization.
	IdempotencyKey string
	// TenantID is required bounded scope; a trusted ingress must authenticate
	// and authorize it.
	TenantID string
	// Actor is required bounded attribution; authenticated ingress owns its
	// binding to a principal.
	Actor string
	// AuthenticationMethod is optional bounded audit attribution; orchestration
	// defaults it to internal, without authenticating anyone.
	AuthenticationMethod string
	// Reason is required nonblank audit text bounded by MaxReasonBytes; callers
	// must omit credentials.
	Reason string
	// Action selects a supported mutation and its required
	// target/confirmation/options.
	Action Action
	// Capability is optional bounded audit attribution; orchestration defaults
	// it to the action text.
	Capability string
	// Target identifies the scoped resource; action/target combinations are
	// validated.
	Target Target
	// RequestedAt is required and anchors the enforcement lifetime; validation
	// does not replace caller time.
	RequestedAt time.Time
	// Deadline is optional; orchestration defaults it to RequestedAt plus
	// DefaultCommandLifetime, and supplied values must follow RequestedAt
	// within MaxCommandLifetime.
	Deadline time.Time
	// Confirmed must be true for purge, bulk retry, replay and scaling to zero.
	Confirmed bool
	// Selection is required for bulk retry; its pointer remains caller-owned
	// outside explicit copying boundaries.
	Selection *Selection
	// Replay is required for replay, including an explicit destination and
	// duplicate policy.
	Replay *Replay
	// Scale is required for scaling; zero replicas additionally require
	// confirmation.
	Scale *Scale
}

// MaxBulkSelection is the largest destructive selection accepted by one
// command. Callers must paginate larger administrative workflows.
const MaxBulkSelection uint32 = 1_000

// Selection bounds a bulk administrative mutation.
type Selection struct {
	// Limit must be positive and at most MaxBulkSelection; it is a per-command
	// selection bound.
	Limit uint32
}

// ReplayPolicy declares how duplicate destination identities are handled.
type ReplayPolicy string

const (
	// ReplayRejectDuplicate refuses an existing destination identity.
	ReplayRejectDuplicate ReplayPolicy = "reject_duplicate"
	// ReplayReplaceDuplicate explicitly permits replacement of an existing
	// destination identity.
	ReplayReplaceDuplicate ReplayPolicy = "replace_duplicate"
)

// Replay contains the explicit destination and idempotency semantics required
// for a replay command.
type Replay struct {
	// Destination is a required nonblank queue identity; Service separately
	// authorizes this destination.
	Destination string
	// IdempotencyPolicy must explicitly select rejection or replacement of
	// destination duplicates.
	IdempotencyPolicy ReplayPolicy
}

// MaxScaleReplicas bounds one explicitly authorized scaling request.
const MaxScaleReplicas uint32 = 10_000

const (
	// DefaultCommandLifetime is the enforcement window assigned when a caller
	// supplies no deadline; explicit windows may extend to MaxCommandLifetime.
	DefaultCommandLifetime time.Duration = 30_000_000_000
	// MaxCommandLifetime prevents administrative work from remaining live
	// indefinitely across request and adapter boundaries.
	MaxCommandLifetime time.Duration = 300_000_000_000
)

// Scale declares the desired Kubernetes workload replica count.
type Scale struct {
	// Replicas is at most MaxScaleReplicas; zero requires command confirmation.
	Replicas uint32
}

// Permission names an authorization capability required by an operation.
// Naming a permission does not authenticate a principal or grant access.
type Permission string

const (
	// PermissionView permits ordinary administrative views.
	PermissionView Permission = "view"
	// PermissionPause permits pausing supported resources.
	PermissionPause Permission = "pause"
	// PermissionResume permits resuming supported resources.
	PermissionResume Permission = "resume"
	// PermissionDrain permits worker draining.
	PermissionDrain Permission = "drain"
	// PermissionTerminate permits worker termination.
	PermissionTerminate Permission = "terminate"
	// PermissionRetry permits one record retry.
	PermissionRetry Permission = "retry"
	// PermissionBulkRetry permits bounded bulk retries.
	PermissionBulkRetry Permission = "bulk_retry"
	// PermissionDelete permits record deletion.
	PermissionDelete Permission = "delete"
	// PermissionPurge permits confirmed purging.
	PermissionPurge Permission = "purge"
	// PermissionReplay permits replay, including separate destination
	// authorization.
	PermissionReplay Permission = "replay"
	// PermissionScale permits workload scaling.
	PermissionScale Permission = "scale"
	// PermissionRecordList permits safe record metadata listings.
	PermissionRecordList Permission = "record_list"
	// PermissionRecordInspect permits safe record metadata inspection.
	PermissionRecordInspect Permission = "record_inspect"
	// PermissionPayloadView permits separately audited payload disclosure.
	PermissionPayloadView Permission = "payload_view"
	// PermissionDiagnosticsView permits separately audited diagnostic
	// disclosure.
	PermissionDiagnosticsView Permission = "diagnostics_view"
	// PermissionAuditView permits tenant audit history.
	PermissionAuditView Permission = "audit_view"
	// PermissionRetentionConfigure permits retention-policy configuration.
	PermissionRetentionConfigure Permission = "retention_configure"
)

// SensitiveAccess is one fail-closed audit record for privileged record data.
type SensitiveAccess struct {
	// CommandID is a required bounded identity linking this privileged access
	// to its audit record.
	CommandID string
	// TenantID is required tenant scope; the record itself does not
	// authenticate it.
	TenantID string
	// Actor is required bounded authenticated attribution supplied by ingress.
	Actor string
	// Permission must be PermissionPayloadView or PermissionDiagnosticsView.
	Permission Permission
	// Target must identify a bounded failure or dead-letter record.
	Target Target
	// OccurredAt is required observation time for the privileged access.
	OccurredAt time.Time
}

// Validate rejects unscoped or non-sensitive access audit records.
func (a SensitiveAccess) Validate() error {
	if invalidIdentity(a.CommandID) {
		return invalid("command_id", "is required and must be bounded")
	}
	if invalidIdentity(a.TenantID) {
		return invalid("tenant_id", "is required and must be bounded")
	}
	if invalidIdentity(a.Actor) {
		return invalid("actor", "is required and must be bounded")
	}
	if a.Permission != PermissionPayloadView && a.Permission != PermissionDiagnosticsView {
		return invalid("permission", "is unsupported")
	}
	if (a.Target.Kind != TargetFailure && a.Target.Kind != TargetDeadLetter) ||
		invalidIdentity(a.Target.Name) {
		return invalid("target", "must identify a bounded record")
	}
	if a.OccurredAt.IsZero() {
		return invalid("occurred_at", "is required")
	}

	return nil
}

// CommandStatus is the durable administrative outcome presented to clients.
type CommandStatus string

const (
	// CommandPending identifies pending dispatch; the native Journal stores
	// this state as its initial durable admission result.
	CommandPending CommandStatus = "pending"
	// CommandAccepted identifies durable acceptance before dispatch.
	CommandAccepted CommandStatus = "accepted"
	// CommandDispatched identifies the durable pre-dispatch recovery boundary,
	// not enforcement proof.
	CommandDispatched CommandStatus = "dispatched"
	// CommandAcknowledged identifies recorded backend acknowledgement before
	// completion.
	CommandAcknowledged CommandStatus = "acknowledged"
	// CommandSucceeded identifies a successful terminal result.
	CommandSucceeded CommandStatus = "succeeded"
	// CommandFailed identifies an unsuccessful terminal result; backend effects
	// may be partial.
	CommandFailed CommandStatus = "failed"
	// CommandUnsupported identifies a terminal unsupported capability.
	CommandUnsupported CommandStatus = "unsupported"
	// CommandTimedOut identifies an expired terminal enforcement outcome.
	CommandTimedOut CommandStatus = "timed_out"
	// CommandPartial identifies a terminal operation with partial effects.
	CommandPartial CommandStatus = "partial"
	// CommandUnknown identifies a terminal result without reliable enforcement
	// certainty.
	CommandUnknown CommandStatus = "unknown"
	// CommandCanceled identifies cancellation before dispatch, not cancellation
	// of previously dispatched work.
	CommandCanceled CommandStatus = "canceled"
)

// CommandResult is the durable result associated with an idempotency key.
type CommandResult struct {
	// CommandID is an optional bounded operation identity retained for legacy
	// result compatibility.
	CommandID string `json:"command_id"`
	// IdempotencyKey is the required scoped journal lookup key.
	IdempotencyKey string `json:"idempotency_key"`
	// TenantID is the required bounded journal scope.
	TenantID string `json:"tenant_id"`
	// Status selects a validated lifecycle state; a returned unknown result
	// need not itself have been durably saved.
	Status CommandStatus `json:"status"`
	// Failure is required bounded nonblank text for unsuccessful terminal
	// states and absent otherwise; use secret-safe public codes.
	Failure string `json:"failure,omitempty"`
	// WorkerID is optional bounded acknowledgement attribution and must
	// accompany Protocol.
	WorkerID string `json:"worker_id,omitempty"`
	// Protocol is optional worker protocol attribution; when present its major
	// must be nonzero and WorkerID must be present.
	Protocol *ProtocolVersion `json:"protocol,omitempty"`
	// CapabilityAvailable optionally distinguishes a reported capability from
	// an unspecified acknowledgement; nil means not supplied.
	CapabilityAvailable *bool `json:"capability_available,omitempty"`
	// DispatchedAt records the pre-dispatch boundary, not proof that a backend
	// accepted the operation.
	DispatchedAt time.Time `json:"dispatched_at,omitempty"`
	// AcknowledgedAt records acknowledgement after dispatch;
	// accepted/pending/dispatched states may not carry it.
	AcknowledgedAt time.Time `json:"acknowledged_at,omitempty"`
	// CompletedAt is required for terminal states and absent before completion.
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// ProtocolVersion identifies the data-plane protocol that acknowledged a
// command without coupling this package to an adapter implementation.
type ProtocolVersion struct {
	// Major is the nonzero major identity required when attached to a
	// CommandResult.
	Major uint16 `json:"major"`
	// Minor is the worker's reported minor protocol revision.
	Minor uint16 `json:"minor"`
}

const (
	// FailureDispatch is the redacted public code for a data-plane dispatch
	// failure. Raw adapter errors must not be exposed through administrative API
	// models because they may contain credentials or backend endpoints.
	FailureDispatch = "dispatch_failed"
	// FailureOutcomeUnknown reports that enforcement may have occurred but no
	// reliable acknowledgement was available.
	FailureOutcomeUnknown = "outcome_unknown"
	// FailureInvalidDispatchResult reports malformed adapter output without
	// exposing its contents.
	FailureInvalidDispatchResult = "invalid_dispatch_result"
	// FailureDeadlineExceeded reports a command whose enforcement window closed
	// before it reached a tenant controller.
	FailureDeadlineExceeded = "deadline_exceeded"
	// FailureCanceled reports cancellation before a command crossed the
	// durable dispatch boundary. Once dispatched, cancellation is not claimed.
	FailureCanceled = "canceled"
)

// Validate rejects malformed or internally inconsistent durable results.
func (r CommandResult) Validate() error {
	if r.CommandID != "" && invalidIdentity(r.CommandID) {
		return invalid("command_id", "must be bounded when supplied")
	}
	if invalidIdentity(r.IdempotencyKey) {
		return invalid("idempotency_key", "is required and must be bounded")
	}
	if invalidIdentity(r.TenantID) {
		return invalid("tenant_id", "is required and must be bounded")
	}
	if !r.Status.valid() {
		return invalid("status", "is unsupported")
	}
	if r.WorkerID != "" && invalidIdentity(r.WorkerID) {
		return invalid("worker_id", "must be bounded when supplied")
	}
	if r.Protocol != nil && r.Protocol.Major == 0 {
		return invalid("protocol", "major version is required when supplied")
	}
	if (r.WorkerID == "") != (r.Protocol == nil) {
		return invalid("worker_id", "worker and protocol must be supplied together")
	}
	if (r.Status == CommandPending || r.Status == CommandAccepted) &&
		(!r.DispatchedAt.IsZero() || !r.AcknowledgedAt.IsZero() || !r.CompletedAt.IsZero()) {
		return invalid("completed_at", "transition timestamps must be empty for pending results")
	}
	if r.Status == CommandDispatched &&
		(r.DispatchedAt.IsZero() || !r.AcknowledgedAt.IsZero() || !r.CompletedAt.IsZero()) {
		return invalid("dispatched_at", "must be the only transition timestamp for dispatched results")
	}
	if r.Status == CommandAcknowledged &&
		(r.DispatchedAt.IsZero() || r.AcknowledgedAt.IsZero() || !r.CompletedAt.IsZero()) {
		return invalid("acknowledged_at", "requires dispatch and must precede completion")
	}
	if r.Status.terminal() && r.CompletedAt.IsZero() {
		return invalid("completed_at", "is required for terminal results")
	}
	if !r.DispatchedAt.IsZero() && !r.AcknowledgedAt.IsZero() &&
		r.AcknowledgedAt.Before(r.DispatchedAt) {
		return invalid("acknowledged_at", "must not precede dispatch")
	}
	if !r.AcknowledgedAt.IsZero() && !r.CompletedAt.IsZero() &&
		r.CompletedAt.Before(r.AcknowledgedAt) {
		return invalid("completed_at", "must not precede acknowledgement")
	}
	if r.Status.requiresFailure() &&
		(strings.TrimSpace(r.Failure) == "" || len(r.Failure) > MaxFailureBytes) {
		return invalid("failure", "is required and must be bounded for unsuccessful results")
	}
	if !r.Status.requiresFailure() && r.Failure != "" {
		return invalid("failure", "must be empty for non-failed results")
	}

	return nil
}

func (s CommandStatus) valid() bool {
	switch s {
	case CommandPending, CommandAccepted, CommandDispatched, CommandAcknowledged,
		CommandSucceeded, CommandFailed, CommandUnsupported, CommandTimedOut,
		CommandPartial, CommandUnknown, CommandCanceled:
		return true
	default:
		return false
	}
}

func (s CommandStatus) requiresFailure() bool {
	switch s {
	case CommandFailed, CommandUnsupported, CommandTimedOut, CommandPartial,
		CommandUnknown, CommandCanceled:
		return true
	case CommandPending, CommandAccepted, CommandDispatched, CommandAcknowledged,
		CommandSucceeded:
		return false
	default:
		return true
	}
}

func (s CommandStatus) terminal() bool {
	switch s {
	case CommandSucceeded, CommandFailed, CommandUnsupported, CommandTimedOut,
		CommandPartial, CommandUnknown, CommandCanceled:
		return true
	default:
		return false
	}
}

// Validate rejects incomplete and unsupported mutation envelopes.
func (c Command) Validate() error {
	if c.CommandID != "" && invalidIdentity(c.CommandID) {
		return invalid("command_id", "must be bounded when supplied")
	}
	if invalidIdentity(c.IdempotencyKey) {
		return invalid("idempotency_key", "is required and must be bounded")
	}
	if invalidIdentity(c.TenantID) {
		return invalid("tenant_id", "is required and must be bounded")
	}
	if invalidIdentity(c.Actor) {
		return invalid("actor", "is required and must be bounded")
	}
	if c.AuthenticationMethod != "" && invalidIdentity(c.AuthenticationMethod) {
		return invalid("authentication_method", "must be bounded when supplied")
	}
	if strings.TrimSpace(c.Reason) == "" || len(c.Reason) > MaxReasonBytes {
		return invalid("reason", "is required and must be bounded")
	}
	if !c.Action.valid() {
		return invalid("action", "is unsupported")
	}
	if c.Capability != "" && invalidIdentity(c.Capability) {
		return invalid("capability", "must be bounded when supplied")
	}
	if !c.Target.Kind.valid() {
		return invalid("target.kind", "is unsupported")
	}
	if invalidIdentity(c.Target.Name) {
		return invalid("target.name", "is required and must be bounded")
	}
	if c.RequestedAt.IsZero() {
		return invalid("requested_at", "is required")
	}
	if !c.Deadline.IsZero() && (!c.Deadline.After(c.RequestedAt) ||
		c.Deadline.Sub(c.RequestedAt) > MaxCommandLifetime) {
		return invalid("deadline", "must follow requested_at within the command lifetime")
	}
	if !c.Action.supports(c.Target.Kind) {
		return invalid("target.kind", "is unsupported for the action")
	}

	switch c.Action {
	case ActionPurge:
		if !c.Confirmed {
			return invalid("confirmed", "is required for purge")
		}
	case ActionBulkRetry:
		if !c.Confirmed {
			return invalid("confirmed", "is required for bulk retry")
		}
		if c.Selection == nil {
			return invalid("selection", "is required for bulk retry")
		}
		if c.Selection.Limit == 0 || c.Selection.Limit > MaxBulkSelection {
			return invalid("selection.limit", "must be within the bulk limit")
		}
	case ActionReplay:
		if !c.Confirmed {
			return invalid("confirmed", "is required for replay")
		}
		if c.Replay == nil {
			return invalid("replay", "is required")
		}
		if strings.TrimSpace(c.Replay.Destination) == "" {
			return invalid("replay.destination", "is required")
		}
		if !c.Replay.IdempotencyPolicy.valid() {
			return invalid("replay.idempotency_policy", "is unsupported")
		}
	case ActionScale:
		if c.Scale == nil {
			return invalid("scale", "is required")
		}
		if c.Scale.Replicas > MaxScaleReplicas {
			return invalid("scale.replicas", "must be within the replica limit")
		}
		if c.Scale.Replicas == 0 && !c.Confirmed {
			return invalid("confirmed", "is required to scale to zero")
		}
	default:
	}

	return nil
}

func (p ReplayPolicy) valid() bool {
	switch p {
	case ReplayRejectDuplicate, ReplayReplaceDuplicate:
		return true
	default:
		return false
	}
}

func (a Action) valid() bool {
	switch a {
	case ActionPause,
		ActionResume,
		ActionDrain,
		ActionTerminate,
		ActionRetry,
		ActionBulkRetry,
		ActionDelete,
		ActionPurge,
		ActionReplay,
		ActionScale:
		return true
	default:
		return false
	}
}

func (a Action) supports(target TargetKind) bool {
	switch a {
	case ActionPause, ActionResume:
		return target == TargetQueue || target == TargetWorkerGroup
	case ActionDrain, ActionTerminate:
		return target == TargetWorker || target == TargetWorkerGroup
	case ActionRetry, ActionBulkRetry, ActionDelete, ActionReplay:
		return target == TargetFailure || target == TargetDeadLetter
	case ActionPurge:
		return target == TargetQueue || target == TargetFailure || target == TargetDeadLetter
	case ActionScale:
		return target == TargetWorkload
	default:
		return false
	}
}

func (k TargetKind) valid() bool {
	switch k {
	case TargetQueue,
		TargetWorker,
		TargetWorkerGroup,
		TargetFailure,
		TargetDeadLetter,
		TargetWorkload:
		return true
	default:
		return false
	}
}

func invalidIdentity(value string) bool {
	return strings.TrimSpace(value) == "" || len(value) > MaxIdentityBytes
}

// ValidationError is a machine-readable public contract for invalid input.
type ValidationError struct {
	// Field names the invalid input boundary and is printed by Error; keep
	// labels non-sensitive.
	Field string
	// Problem is printed by Error; use rule descriptions rather than rejected
	// values.
	Problem string
}

// Error implements error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Problem)
}

func invalid(field, problem string) error {
	return &ValidationError{Field: field, Problem: problem}
}
