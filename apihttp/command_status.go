package apihttp

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	authentication "github.com/faustbrian/go-authentication/v2"
	controlplane "github.com/faustbrian/go-queue-control-plane/v3"
	controlpostgres "github.com/faustbrian/go-queue-control-plane/v3/postgres"
)

const (
	defaultCommandPageSize uint32 = 100
	// MaxCommandPageSize bounds one public command-history response.
	MaxCommandPageSize = controlpostgres.MaxCommandPageSize
	// MaxCommandCursorBytes bounds public opaque command-history cursors.
	MaxCommandCursorBytes = controlpostgres.MaxCommandCursorBytes
)

// CommandHistoryPage is one bounded tenant command-history response.
type CommandHistoryPage struct {
	// Commands is a newly projected newest-first page; source providers own
	// size and content validity.
	Commands []CommandHistoryEntry `json:"commands"`
	// NextCursor is opaque pagination state; empty means no additional page was
	// detected.
	NextCursor string `json:"next_cursor,omitempty"`
}

// CommandHistoryEntry is one immutable command envelope and durable outcome.
type CommandHistoryEntry struct {
	// CommandID is the stored operation identity, separate from its idempotency
	// key.
	CommandID string `json:"command_id"`
	// IdempotencyKey is the tenant-scoped journal duplicate/conflict key.
	IdempotencyKey string `json:"idempotency_key"`
	// Actor is the authenticated attribution originally recorded, not body-selected identity.
	Actor string `json:"actor"`
	// AuthenticationMethod is recorded ingress method attribution.
	AuthenticationMethod string `json:"authentication_method"`
	// Reason is stored administrative audit text; callers must exclude
	// sensitive values.
	Reason string `json:"reason"`
	// Action is the originally admitted operation.
	Action controlplane.Action `json:"action"`
	// Capability is recorded operation capability attribution, not a grant.
	Capability string `json:"capability"`
	// Target is the originally admitted resource.
	Target TargetRequest `json:"target"`
	// RequestedAt is the stored request time used in newest-first ordering.
	RequestedAt time.Time `json:"requested_at"`
	// Deadline is the stored bounded enforcement deadline.
	Deadline time.Time `json:"deadline"`
	// Confirmed records the submitted destructive-operation confirmation.
	Confirmed bool `json:"confirmed"`
	// Selection is an independently projected optional bulk bound.
	Selection *SelectionRequest `json:"selection,omitempty"`
	// Replay is an independently projected optional replay destination/policy.
	Replay *ReplayRequest `json:"replay,omitempty"`
	// Scale is an independently projected optional replica request.
	Scale *ScaleRequest `json:"scale,omitempty"`
	// Result is the stored outcome; HTTP projection borrows its optional fields
	// until synchronous encoding.
	Result controlplane.CommandResult `json:"result"`
}

func (h *handler) listCommandHistory(writer http.ResponseWriter, request *http.Request) {
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
		request.Context(), tenant, principal.Subject(), controlplane.PermissionView,
		controlplane.Target{Kind: controlplane.TargetWorkload, Name: "commands"},
	); err != nil {
		writeCommandError(writer, err)
		return
	}
	cursor, limit, err := parseCommandQuery(request.URL.Query())
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	stored, err := h.commandHistory.ListTenant(request.Context(), tenant, cursor, limit)
	if err != nil {
		writeCommandError(writer, err)
		return
	}
	page := CommandHistoryPage{
		Commands:   make([]CommandHistoryEntry, len(stored.Records)),
		NextCursor: stored.NextCursor,
	}
	for index, record := range stored.Records {
		page.Commands[index] = publicCommandRecord(record)
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *handler) getCommandResult(writer http.ResponseWriter, request *http.Request) {
	principal, _ := authentication.PrincipalFromContext(request.Context())
	if principal.IsAnonymous() {
		writeProblem(writer, http.StatusUnauthorized, "unauthenticated")
		return
	}
	tenant := request.PathValue("tenant")
	key := request.PathValue("key")
	if !validIdentity(tenant) || !validIdentity(key) {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.viewer.Authorize(
		request.Context(), tenant, principal.Subject(), controlplane.PermissionView,
		controlplane.Target{Kind: controlplane.TargetWorkload, Name: "commands"},
	); err != nil {
		writeCommandError(writer, err)
		return
	}
	result, err := h.commandResults.Get(request.Context(), tenant, key)
	if err != nil {
		writeCommandError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func parseCommandQuery(values url.Values) (string, uint32, error) {
	for key, entries := range values {
		if (key != "cursor" && key != "limit") || len(entries) != 1 {
			return "", 0, ErrInvalidConfiguration
		}
	}
	cursor := values.Get("cursor")
	if len(cursor) > MaxCommandCursorBytes {
		return "", 0, ErrInvalidConfiguration
	}
	limit := defaultCommandPageSize
	if raw, exists := values["limit"]; exists {
		value, err := strconv.ParseUint(raw[0], 10, 32)
		if err != nil || value == 0 || value > uint64(MaxCommandPageSize) {
			return "", 0, ErrInvalidConfiguration
		}
		limit = uint32(value)
	}

	return cursor, limit, nil
}

func publicCommandRecord(record controlpostgres.CommandRecord) CommandHistoryEntry {
	command := record.Command
	entry := CommandHistoryEntry{
		CommandID:            command.CommandID,
		IdempotencyKey:       command.IdempotencyKey,
		Actor:                command.Actor,
		AuthenticationMethod: command.AuthenticationMethod,
		Reason:               command.Reason,
		Action:               command.Action,
		Capability:           command.Capability,
		Target:               TargetRequest{Kind: command.Target.Kind, Name: command.Target.Name},
		RequestedAt:          command.RequestedAt,
		Deadline:             command.Deadline,
		Confirmed:            command.Confirmed,
		Result:               record.Result,
	}
	if command.Selection != nil {
		entry.Selection = &SelectionRequest{Limit: command.Selection.Limit}
	}
	if command.Replay != nil {
		entry.Replay = &ReplayRequest{
			Destination:       command.Replay.Destination,
			IdempotencyPolicy: command.Replay.IdempotencyPolicy,
		}
	}
	if command.Scale != nil {
		entry.Scale = &ScaleRequest{Replicas: command.Scale.Replicas}
	}

	return entry
}
