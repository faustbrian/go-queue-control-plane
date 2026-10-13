package apihttp

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	authentication "github.com/faustbrian/go-authentication/v2"
	controlplane "github.com/faustbrian/go-queue-control-plane/v4"
	queue "github.com/faustbrian/go-queue/management"
)

const (
	defaultRecordPageSize       uint32 = 100
	sensitiveContentDisposition        = `attachment; filename="queue-record.json"`
)

// RecordPage is one stable bounded administrative record page.
type RecordPage struct {
	// Records is the newly projected metadata page with hidden
	// payload/diagnostics.
	Records []Record `json:"records"`
	// NextCursor is opaque backend pagination state.
	NextCursor string `json:"next_cursor,omitempty"`
}

// Record is the stable HTTP representation of a failed or dead-lettered job.
type Record struct {
	// Kind identifies failure or dead-letter storage.
	Kind queue.RecordKind `json:"kind"`
	// ID is the scoped record identity checked for inspection.
	ID string `json:"id"`
	// Backend is source backend attribution, not addressing.
	Backend string `json:"backend"`
	// Queue is the logical source queue identity.
	Queue string `json:"queue"`
	// OccurredAt is source occurrence time normalized to UTC.
	OccurredAt time.Time `json:"occurred_at"`
	// Attempts is reported delivery-attempt count.
	Attempts uint32 `json:"attempts"`
	// FailureCode is the source's public failure category; the provider must
	// keep it secret-safe.
	FailureCode string `json:"failure_code"`
	// Payload is hidden by default; explicit permitted inspection may return
	// copied redacted or revealed bytes.
	Payload RecordPayload `json:"payload"`
	// EnvelopeVersion optionally reports source envelope format, not control-plane API version.
	EnvelopeVersion uint16 `json:"envelope_version,omitempty"`
	// PayloadSchemaVersion optionally identifies the application payload
	// schema.
	PayloadSchemaVersion string `json:"payload_schema_version,omitempty"`
	// OriginalID optionally preserves the original job identity through
	// retries/replay.
	OriginalID string `json:"original_id,omitempty"`
	// Topic optionally identifies source topic routing.
	Topic string `json:"topic,omitempty"`
	// Stream optionally identifies source stream routing.
	Stream string `json:"stream,omitempty"`
	// RoutingKey optionally identifies source routing metadata; providers must
	// classify safe public fields.
	RoutingKey string `json:"routing_key,omitempty"`
	// ConsumerGroup optionally identifies source consumer-group metadata.
	ConsumerGroup string `json:"consumer_group,omitempty"`
	// SourceRecordID optionally identifies the original transport record.
	SourceRecordID string `json:"source_record_id,omitempty"`
	// EnqueuedAt is an optional independently copied UTC enqueue time.
	EnqueuedAt *time.Time `json:"enqueued_at,omitempty"`
	// FirstDeliveryAt is an optional independently copied UTC first delivery
	// time.
	FirstDeliveryAt *time.Time `json:"first_delivery_at,omitempty"`
	// LastDeliveryAt is an optional independently copied UTC last delivery
	// time.
	LastDeliveryAt *time.Time `json:"last_delivery_at,omitempty"`
	// DeadLetteredAt is an optional independently copied UTC dead-letter time.
	DeadLetteredAt *time.Time `json:"dead_lettered_at,omitempty"`
	// RetryPolicy optionally names reported retry policy, not a new enforcement
	// instruction.
	RetryPolicy string `json:"retry_policy,omitempty"`
	// Classification is source failure classification.
	Classification queue.Classification `json:"classification,omitempty"`
	// FailureSummary is public source summary text; HTTP projection does not
	// sanitize its contents.
	FailureSummary string `json:"failure_summary,omitempty"`
	// Diagnostics is hidden unless independently authorized and audited;
	// returned bytes are copied.
	Diagnostics RecordPayload `json:"diagnostics"`
	// HandlerType optionally names the application handler.
	HandlerType string `json:"handler_type,omitempty"`
	// JobType optionally names the application job type.
	JobType string `json:"job_type,omitempty"`
	// Tags is a copied public metadata map; providers must exclude sensitive
	// values.
	Tags map[string]string `json:"tags,omitempty"`
	// TraceID optionally preserves trace correlation, not authorization.
	TraceID string `json:"trace_id,omitempty"`
	// TenantID is optional source metadata; request authorization uses route
	// scope instead.
	TenantID string `json:"tenant_id,omitempty"`
	// ProducerVersion optionally identifies producer implementation version.
	ProducerVersion string `json:"producer_version,omitempty"`
	// WorkerVersion optionally identifies processing worker version.
	WorkerVersion string `json:"worker_version,omitempty"`
	// OriginalDeadLetterID optionally identifies the original dead-letter
	// lineage.
	OriginalDeadLetterID string `json:"original_dead_letter_id,omitempty"`
	// PriorDeadLetterID optionally identifies the immediately preceding dead-letter lineage.
	PriorDeadLetterID string `json:"prior_dead_letter_id,omitempty"`
	// ReplayGeneration is the reported replay generation, not a control-plane
	// dispatch fence.
	ReplayGeneration uint32 `json:"replay_generation,omitempty"`
	// RetentionDeadline is an optional independently copied UTC retention
	// deadline.
	RetentionDeadline *time.Time `json:"retention_deadline,omitempty"`
}

// RecordPayload is hidden by default and contains bytes only after explicit
// privileged inspection.
type RecordPayload struct {
	// Visibility reports hidden/redacted/revealed representation; hidden
	// projections retain no Data or ContentType.
	Visibility queue.PayloadVisibility `json:"visibility"`
	// ContentType is the representation media type when permitted, not source
	// authenticity.
	ContentType string `json:"content_type,omitempty"`
	// Size reports source payload size even when its bytes are hidden.
	Size int64 `json:"size"`
	// Data is a copied permitted representation encoded as base64 in JSON;
	// empty is not proof that the original payload was empty.
	Data []byte `json:"data,omitempty"`
}

func (h *handler) listFailures(writer http.ResponseWriter, request *http.Request) {
	h.listRecords(writer, request, queue.RecordFailure)
}

func (h *handler) listDeadLetters(writer http.ResponseWriter, request *http.Request) {
	h.listRecords(writer, request, queue.RecordDeadLetter)
}

func (h *handler) listRecords(
	writer http.ResponseWriter,
	request *http.Request,
	kind queue.RecordKind,
) {
	principal, tenant, ok := h.authorizeRecordRead(
		writer, request, kind, collectionName(kind), controlplane.PermissionRecordList,
	)
	if !ok {
		return
	}
	query, err := parseRecordQuery(request.URL.Query())
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}

	var page queue.RecordPage
	if kind == queue.RecordFailure {
		page, err = h.records.ListFailures(request.Context(), tenant, query)
	} else {
		page, err = h.records.ListDeadLetters(request.Context(), tenant, query)
	}
	_ = principal
	if err != nil {
		writeProblem(writer, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(writer, http.StatusOK, recordPage(page))
}

func (h *handler) inspectFailure(writer http.ResponseWriter, request *http.Request) {
	h.inspectRecord(writer, request, queue.RecordFailure)
}

func (h *handler) inspectDeadLetter(writer http.ResponseWriter, request *http.Request) {
	h.inspectRecord(writer, request, queue.RecordDeadLetter)
}

func (h *handler) inspectRecord(
	writer http.ResponseWriter,
	request *http.Request,
	kind queue.RecordKind,
) {
	id := request.PathValue("record")
	principal, tenant, ok := h.authorizeRecordRead(
		writer, request, kind, id, controlplane.PermissionRecordInspect,
	)
	if !ok {
		return
	}
	visibility, revealDiagnostics, err := parseRecordVisibility(request.URL.Query())
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	permissions := sensitiveRecordPermissions(visibility, revealDiagnostics)
	for _, permission := range permissions {
		if h.workflowLimiter != nil && !h.workflowLimiter.Allow(
			request.Context(), workflowRateLimitKey(principal.Subject(), string(permission)),
		) {
			writer.Header().Set("Retry-After", "1")
			writeProblem(writer, http.StatusTooManyRequests, "rate_limited")
			return
		}
	}
	if visibility == queue.PayloadRevealed {
		if err := h.viewer.Authorize(
			request.Context(), tenant, principal.Subject(),
			controlplane.PermissionPayloadView,
			controlplane.Target{Kind: targetKindForRecord(kind), Name: id},
		); err != nil {
			writeCommandError(writer, err)
			return
		}
	}
	if revealDiagnostics {
		if err := h.viewer.Authorize(
			request.Context(), tenant, principal.Subject(),
			controlplane.PermissionDiagnosticsView,
			controlplane.Target{Kind: targetKindForRecord(kind), Name: id},
		); err != nil {
			writeCommandError(writer, err)
			return
		}
	}
	for _, permission := range permissions {
		if h.sensitiveAudit == nil {
			writeProblem(writer, http.StatusServiceUnavailable, "audit_unavailable")
			return
		}
		identifier, err := h.newCommandID()
		if err != nil {
			writeProblem(writer, http.StatusServiceUnavailable, "audit_unavailable")
			return
		}
		if err := h.sensitiveAudit.AuditSensitiveAccess(
			request.Context(),
			controlplane.SensitiveAccess{
				CommandID: identifier, TenantID: tenant, Actor: principal.Subject(),
				Permission: permission,
				Target:     controlplane.Target{Kind: targetKindForRecord(kind), Name: id},
				OccurredAt: h.now().UTC(),
			},
		); err != nil {
			writeProblem(writer, http.StatusServiceUnavailable, "audit_unavailable")
			return
		}
	}
	record, err := h.records.Inspect(request.Context(), tenant, queue.InspectRequest{
		Kind: kind, ID: id, Visibility: visibility,
	})
	if err != nil {
		writeProblem(writer, http.StatusInternalServerError, "internal_error")
		return
	}
	if len(permissions) > 0 {
		writer.Header().Set("Content-Disposition", sensitiveContentDisposition)
	}
	writeJSON(writer, http.StatusOK, recordModel(record, visibility, revealDiagnostics))
}

func sensitiveRecordPermissions(
	visibility queue.PayloadVisibility,
	revealDiagnostics bool,
) []controlplane.Permission {
	permissions := make([]controlplane.Permission, 0, 2)
	if visibility == queue.PayloadRevealed {
		permissions = append(permissions, controlplane.PermissionPayloadView)
	}
	if revealDiagnostics {
		permissions = append(permissions, controlplane.PermissionDiagnosticsView)
	}

	return permissions
}

func (h *handler) authorizeRecordRead(
	writer http.ResponseWriter,
	request *http.Request,
	kind queue.RecordKind,
	name string,
	permission controlplane.Permission,
) (authentication.Principal, string, bool) {
	principal, _ := authentication.PrincipalFromContext(request.Context())
	if principal.IsAnonymous() {
		writeProblem(writer, http.StatusUnauthorized, "unauthenticated")
		return authentication.Principal{}, "", false
	}
	tenant := request.PathValue("tenant")
	if !validIdentity(tenant) || !validIdentity(name) {
		writeProblem(writer, http.StatusBadRequest, "invalid_request")
		return authentication.Principal{}, "", false
	}
	if err := h.viewer.Authorize(
		request.Context(), tenant, principal.Subject(), permission,
		controlplane.Target{Kind: targetKindForRecord(kind), Name: name},
	); err != nil {
		writeCommandError(writer, err)
		return authentication.Principal{}, "", false
	}

	return principal, tenant, true
}

func parseRecordQuery(values url.Values) (queue.PageRequest, error) {
	request := queue.PageRequest{
		Limit: defaultRecordPageSize, Sort: queue.SortOccurredAt, Direction: queue.SortDescending,
	}
	allowed := map[string]bool{
		"cursor": true, "limit": true, "search": true, "sort": true, "direction": true,
	}
	for key, entries := range values {
		if !allowed[key] || len(entries) != 1 {
			return queue.PageRequest{}, ErrInvalidConfiguration
		}
	}
	if raw, exists := values["cursor"]; exists {
		if raw[0] == "" {
			return queue.PageRequest{}, ErrInvalidConfiguration
		}
		request.Cursor = raw[0]
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return queue.PageRequest{}, ErrInvalidConfiguration
		}
		request.Limit = uint32(limit)
	} else if _, exists := values["limit"]; exists {
		return queue.PageRequest{}, ErrInvalidConfiguration
	}
	request.Search = values.Get("search")
	if raw := values.Get("sort"); raw != "" {
		request.Sort = queue.SortField(raw)
	} else if _, exists := values["sort"]; exists {
		return queue.PageRequest{}, ErrInvalidConfiguration
	}
	if raw := values.Get("direction"); raw != "" {
		request.Direction = queue.SortDirection(raw)
	} else if _, exists := values["direction"]; exists {
		return queue.PageRequest{}, ErrInvalidConfiguration
	}
	if err := request.Validate(); err != nil {
		return queue.PageRequest{}, ErrInvalidConfiguration
	}

	return request, nil
}

func parseRecordVisibility(values url.Values) (queue.PayloadVisibility, bool, error) {
	for key, entries := range values {
		if (key != "payload" && key != "diagnostics") || len(entries) != 1 {
			return queue.PayloadHidden, false, ErrInvalidConfiguration
		}
	}
	visibility := queue.PayloadHidden
	if entries, exists := values["payload"]; exists {
		switch entries[0] {
		case "hidden":
		case string(queue.PayloadRedacted):
			visibility = queue.PayloadRedacted
		case string(queue.PayloadRevealed):
			visibility = queue.PayloadRevealed
		default:
			return queue.PayloadHidden, false, ErrInvalidConfiguration
		}
	}
	revealDiagnostics := false
	if entries, exists := values["diagnostics"]; exists {
		switch entries[0] {
		case "hidden":
		case "revealed":
			revealDiagnostics = true
		default:
			return queue.PayloadHidden, false, ErrInvalidConfiguration
		}
	}

	return visibility, revealDiagnostics, nil
}

func targetKindForRecord(kind queue.RecordKind) controlplane.TargetKind {
	if kind == queue.RecordFailure {
		return controlplane.TargetFailure
	}

	return controlplane.TargetDeadLetter
}

func collectionName(kind queue.RecordKind) string {
	if kind == queue.RecordFailure {
		return "failures"
	}

	return "dead_letters"
}

func recordPage(page queue.RecordPage) RecordPage {
	result := RecordPage{Records: make([]Record, 0, len(page.Items)), NextCursor: page.NextCursor}
	for _, item := range page.Items {
		result.Records = append(
			result.Records,
			recordModel(item, queue.PayloadHidden, false),
		)
	}

	return result
}

func recordModel(
	record queue.JobRecord,
	visibility queue.PayloadVisibility,
	revealDiagnostics bool,
) Record {
	model := Record{
		Kind: record.Kind, ID: record.ID, Backend: record.Backend, Queue: record.Queue,
		OccurredAt: record.OccurredAt.UTC(),
		Attempts:   record.Attempts, FailureCode: record.FailureCode,
		Payload:              hiddenRecordPayloadModel(record.Payload),
		EnvelopeVersion:      record.EnvelopeVersion,
		PayloadSchemaVersion: record.PayloadSchemaVersion,
		OriginalID:           record.OriginalID,
		Topic:                record.Topic,
		Stream:               record.Stream,
		RoutingKey:           record.RoutingKey,
		ConsumerGroup:        record.ConsumerGroup,
		SourceRecordID:       record.SourceRecordID,
		EnqueuedAt:           recordTimeModel(record.EnqueuedAt),
		FirstDeliveryAt:      recordTimeModel(record.FirstDeliveryAt),
		LastDeliveryAt:       recordTimeModel(record.LastDeliveryAt),
		DeadLetteredAt:       recordTimeModel(record.DeadLetteredAt),
		RetryPolicy:          record.RetryPolicy,
		Classification:       record.Classification,
		FailureSummary:       record.FailureSummary,
		Diagnostics:          hiddenRecordPayloadModel(record.Diagnostics),
		HandlerType:          record.HandlerType,
		JobType:              record.JobType,
		Tags:                 cloneRecordTags(record.Tags),
		TraceID:              record.TraceID,
		TenantID:             record.TenantID,
		ProducerVersion:      record.ProducerVersion,
		WorkerVersion:        record.WorkerVersion,
		OriginalDeadLetterID: record.OriginalDeadLetterID,
		PriorDeadLetterID:    record.PriorDeadLetterID,
		ReplayGeneration:     record.ReplayGeneration,
		RetentionDeadline:    recordTimeModel(record.RetentionDeadline),
	}
	if visibility == queue.PayloadRevealed ||
		(visibility == queue.PayloadRedacted && record.Payload.Visibility == queue.PayloadRedacted) {
		model.Payload = recordPayloadModel(record.Payload)
	}
	if revealDiagnostics {
		model.Diagnostics = recordPayloadModel(record.Diagnostics)
	}

	return model
}

func hiddenRecordPayloadModel(payload queue.Payload) RecordPayload {
	return RecordPayload{Visibility: queue.PayloadHidden, Size: payload.Size}
}

func recordPayloadModel(payload queue.Payload) RecordPayload {
	return RecordPayload{
		Visibility: payload.Visibility, ContentType: payload.ContentType,
		Size: payload.Size, Data: append([]byte(nil), payload.Data...),
	}
}

func recordTimeModel(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()

	return &utc
}

func cloneRecordTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(tags))
	for key, value := range tags {
		cloned[key] = value
	}

	return cloned
}
