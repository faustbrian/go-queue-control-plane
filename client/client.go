// Package client provides typed administrative HTTP access. New selects one
// borrowed authentication source, copies HTTP settings with redirects disabled,
// and bounds buffered response bytes. The caller owns transport/TLS trust,
// credential lifetime and collaborator blocking/concurrency; no polling or
// shutdown goroutine is started. Ordinary successful JSON models are decoded,
// not universally semantically validated; GetDesiredState adds exact-target
// validation for worker convergence.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	controlplane "github.com/faustbrian/go-queue-control-plane/v4"
	controlkubernetes "github.com/faustbrian/go-queue-control-plane/v4/adapters/kubernetes"
	"github.com/faustbrian/go-queue-control-plane/v4/apihttp"
	"github.com/faustbrian/go-queue-control-plane/v4/fleet"
	queue "github.com/faustbrian/go-queue/management"
)

const defaultMaxResponseBytes int64 = 4 << 20

const (
	// APIKeyIDHeader carries the non-secret static key identifier accepted by
	// the deployable control-plane server.
	// #nosec G101 -- public HTTP header name, not a credential; credential values come from caller-owned authentication
	APIKeyIDHeader = "X-Queue-Control-Key-ID"
	// APIKeySecretHeader carries the static key credential accepted by the
	// deployable control-plane server.
	// #nosec G101 -- public HTTP header name, not a credential; credential values come from caller-owned authentication
	APIKeySecretHeader = "X-Queue-Control-Key"
)

var (
	// ErrInvalidConfiguration reports an invalid endpoint, authentication
	// choice or response bound.
	ErrInvalidConfiguration = errors.New("control-plane client: invalid configuration")
	// ErrInvalidRequest reports unsupported or unbounded typed request input.
	ErrInvalidRequest = errors.New("control-plane client: invalid request")
	// ErrInvalidToken reports a blank token returned by the borrowed token
	// source.
	ErrInvalidToken = errors.New("control-plane client: invalid bearer token")
	// ErrInvalidAPIKey reports a blank key identifier or secret returned by the
	// borrowed key source.
	ErrInvalidAPIKey = errors.New("control-plane client: invalid API key")
	// ErrResponseTooLarge reports a response exceeding the configured byte
	// bound before JSON decoding.
	ErrResponseTooLarge = errors.New("control-plane client: response too large")
)

// TokenSource acquires a bearer token for one request context.
type TokenSource interface {
	// Token acquires credentials synchronously for ctx; callers own blocking,
	// shared-use safety and credential lifetime.
	Token(context.Context) (string, error)
}

// APIKeySource acquires a static API-key identifier and secret for one request
// context.
type APIKeySource interface {
	// APIKey returns a nonblank identifier and secret for this request;
	// implementations own blocking, concurrency and secret lifetime.
	APIKey(context.Context) (string, string, error)
}

// Config defines the API endpoint, authentication, and response bounds.
type Config struct {
	// BaseURL must be an HTTP(S) origin without credentials, query, fragment or
	// non-root path; TLS trust is caller-configured.
	BaseURL string
	// HTTPClient is shallow-copied with redirects disabled; nil selects a
	// 30-second client. Transport and Jar remain shared caller-owned
	// collaborators.
	HTTPClient *http.Client
	// Tokens is one borrowed authentication source; exactly one of Tokens and
	// APIKeys must be supplied.
	Tokens TokenSource
	// APIKeys is one borrowed authentication source, mutually exclusive with
	// Tokens.
	APIKeys APIKeySource
	// MaxResponseBytes bounds buffered response bytes before JSON parsing; zero
	// selects 4 MiB, negative values fail.
	MaxResponseBytes int64
}

// WorkerQuery contains bounded worker-list filters.
type WorkerQuery struct {
	// Limit bounds one worker page; zero leaves the server default and values
	// above MaxWorkerPageSize fail.
	Limit uint32
	// After is the optional last worker identity cursor passed to the server.
	After string
	// State optionally filters derived fleet state; the server validates
	// supported values.
	State fleet.State
	// Queue optionally selects a queue membership filter.
	Queue string
}

// WorkloadQuery contains bounded Kubernetes pagination.
type WorkloadQuery struct {
	// Limit bounds one workload page; zero leaves the server default, negative
	// or oversized values fail.
	Limit int64
	// Continue is the optional opaque Kubernetes pagination token, not an
	// authorization capability.
	Continue string
}

// AuditQuery contains bounded audit-history pagination.
type AuditQuery struct {
	// After is the exclusive audit sequence cursor; zero starts at retained
	// history.
	After uint64
	// Limit bounds one audit page; zero leaves the server default.
	Limit uint32
}

// CommandQuery contains bounded command-history pagination.
type CommandQuery struct {
	// Cursor is optional bounded opaque newest-first pagination state.
	Cursor string
	// Limit bounds one command page; zero leaves the server default.
	Limit uint32
}

// QueueQuery contains bounded queue-status pagination.
type QueueQuery struct {
	// Cursor is optional bounded backend pagination state, not authorization.
	Cursor string
	// Limit bounds one queue-status page; zero leaves the server default.
	Limit uint32
}

// RecordQuery contains bounded failure and dead-letter list controls.
type RecordQuery struct {
	// Cursor is optional bounded backend pagination state.
	Cursor string
	// Limit bounds one record page; zero leaves the server default.
	Limit uint32
	// Search is optional bounded metadata search text; it is not payload
	// search.
	Search string
	// Sort optionally selects occurrence time, queue or attempt ordering.
	Sort queue.SortField
	// Direction optionally selects ascending or descending order; empty leaves
	// the server default.
	Direction queue.SortDirection
}

// RecordInspectOptions controls independently privileged record fields.
type RecordInspectOptions struct {
	// Payload must select hidden, redacted or revealed visibility. Revealed
	// bytes require additional permission/access audit; hidden/redacted
	// inspection still requires record_inspect.
	Payload queue.PayloadVisibility
	// RevealDiagnostics requests separately authorized/audited diagnostics, not
	// blanket payload access.
	RevealDiagnostics bool
}

// APIError is a stable non-success administrative response.
type APIError struct {
	// Status is the non-success HTTP status returned by the remote server.
	Status int
	// Code is decoded from the remote problem body and printed by Error; it is
	// not independently sanitized by Client.
	Code string
}

// Error prints status and server-provided Code, not the entire response body.
func (e *APIError) Error() string {
	return fmt.Sprintf("control-plane API: status %d code %s", e.Status, e.Code)
}

// Client calls the versioned administrative API after New; its zero value is
// invalid. Shared use requires safe borrowed transports and credential sources.
// Methods close response bodies but do not close the transport or credentials.
type Client struct {
	baseURL          *url.URL
	httpClient       *http.Client
	tokens           TokenSource
	apiKeys          APIKeySource
	maxResponseBytes int64
}

// New creates a typed administrative API client.
func New(config Config) (*Client, error) {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || baseURL.Host == "" ||
		(baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
		baseURL.User != nil || (baseURL.Path != "" && baseURL.Path != "/") ||
		baseURL.RawQuery != "" || baseURL.Fragment != "" ||
		(nilInterface(config.Tokens) == nilInterface(config.APIKeys)) ||
		config.MaxResponseBytes < 0 {
		return nil, ErrInvalidConfiguration
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	httpClient := *config.HTTPClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	if config.MaxResponseBytes == 0 {
		config.MaxResponseBytes = defaultMaxResponseBytes
	}
	baseURL.Path = ""

	return &Client{
		baseURL:          baseURL,
		httpClient:       &httpClient,
		tokens:           config.Tokens,
		apiKeys:          config.APIKeys,
		maxResponseBytes: config.MaxResponseBytes,
	}, nil
}

// ExecuteCommand submits one actor-free command request for a tenant.
func (c *Client) ExecuteCommand(
	ctx context.Context,
	tenant string,
	command apihttp.CommandRequest,
) (controlplane.CommandResult, error) {
	if strings.TrimSpace(tenant) == "" {
		return controlplane.CommandResult{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "commands")
	var result controlplane.CommandResult
	if err := c.do(ctx, http.MethodPost, endpoint, command, &result); err != nil {
		return controlplane.CommandResult{}, err
	}

	return result, nil
}

// GetCommand returns one tenant-scoped durable command outcome.
func (c *Client) GetCommand(
	ctx context.Context,
	tenant string,
	key string,
) (controlplane.CommandResult, error) {
	if invalidIdentity(tenant) || invalidIdentity(key) {
		return controlplane.CommandResult{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "commands", key)
	var result controlplane.CommandResult
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &result); err != nil {
		return controlplane.CommandResult{}, err
	}

	return result, nil
}

// ListCommands returns one bounded tenant command-history page.
func (c *Client) ListCommands(
	ctx context.Context,
	tenant string,
	query CommandQuery,
) (apihttp.CommandHistoryPage, error) {
	if invalidIdentity(tenant) || query.Limit > apihttp.MaxCommandPageSize ||
		len(query.Cursor) > apihttp.MaxCommandCursorBytes {
		return apihttp.CommandHistoryPage{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "commands")
	values := endpoint.Query()
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.FormatUint(uint64(query.Limit), 10))
	}
	endpoint.RawQuery = values.Encode()

	var page apihttp.CommandHistoryPage
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return apihttp.CommandHistoryPage{}, err
	}

	return page, nil
}

// ListWorkers returns one bounded tenant worker page.
func (c *Client) ListWorkers(
	ctx context.Context,
	tenant string,
	query WorkerQuery,
) (apihttp.WorkerPage, error) {
	if strings.TrimSpace(tenant) == "" || query.Limit > apihttp.MaxWorkerPageSize {
		return apihttp.WorkerPage{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "workers")
	values := endpoint.Query()
	if query.Limit > 0 {
		values.Set("limit", strconv.FormatUint(uint64(query.Limit), 10))
	}
	if query.After != "" {
		values.Set("after", query.After)
	}
	if query.State != "" {
		values.Set("state", string(query.State))
	}
	if query.Queue != "" {
		values.Set("queue", query.Queue)
	}
	endpoint.RawQuery = values.Encode()

	var page apihttp.WorkerPage
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return apihttp.WorkerPage{}, err
	}

	return page, nil
}

// ListQueues returns one bounded tenant queue-status page.
func (c *Client) ListQueues(
	ctx context.Context,
	tenant string,
	query QueueQuery,
) (apihttp.QueuePage, error) {
	if invalidIdentity(tenant) || query.Limit > queue.MaxStatusPageSize ||
		len(query.Cursor) > queue.MaxCursorBytes {
		return apihttp.QueuePage{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "queues")
	values := endpoint.Query()
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.FormatUint(uint64(query.Limit), 10))
	}
	endpoint.RawQuery = values.Encode()

	var page apihttp.QueuePage
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return apihttp.QueuePage{}, err
	}

	return page, nil
}

// ListWorkloads returns one bounded tenant workload page.
func (c *Client) ListWorkloads(
	ctx context.Context,
	tenant string,
	query WorkloadQuery,
) (controlkubernetes.Page, error) {
	if invalidIdentity(tenant) || query.Limit < 0 || query.Limit > controlkubernetes.MaxPageSize ||
		len(query.Continue) > controlkubernetes.MaxContinueTokenBytes {
		return controlkubernetes.Page{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "workloads")
	values := endpoint.Query()
	if query.Limit > 0 {
		values.Set("limit", strconv.FormatInt(query.Limit, 10))
	}
	if query.Continue != "" {
		values.Set("continue", query.Continue)
	}
	endpoint.RawQuery = values.Encode()

	var page controlkubernetes.Page
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return controlkubernetes.Page{}, err
	}

	return page, nil
}

// ListAudit returns one bounded tenant audit-history page.
func (c *Client) ListAudit(
	ctx context.Context,
	tenant string,
	query AuditQuery,
) (apihttp.AuditPage, error) {
	if strings.TrimSpace(tenant) == "" || query.Limit > apihttp.MaxAuditPageSize {
		return apihttp.AuditPage{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, "audit")
	values := endpoint.Query()
	if query.After > 0 {
		values.Set("after", strconv.FormatUint(query.After, 10))
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.FormatUint(uint64(query.Limit), 10))
	}
	endpoint.RawQuery = values.Encode()

	var page apihttp.AuditPage
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return apihttp.AuditPage{}, err
	}

	return page, nil
}

// ListFailures returns one bounded tenant failure page.
func (c *Client) ListFailures(
	ctx context.Context,
	tenant string,
	query RecordQuery,
) (apihttp.RecordPage, error) {
	return c.listRecords(ctx, tenant, "failures", query)
}

// ListDeadLetters returns one bounded tenant dead-letter page.
func (c *Client) ListDeadLetters(
	ctx context.Context,
	tenant string,
	query RecordQuery,
) (apihttp.RecordPage, error) {
	return c.listRecords(ctx, tenant, "dead-letters", query)
}

// InspectFailure returns one tenant failure at the requested visibility.
func (c *Client) InspectFailure(
	ctx context.Context,
	tenant string,
	id string,
	visibility queue.PayloadVisibility,
) (apihttp.Record, error) {
	return c.InspectFailureWithOptions(ctx, tenant, id, RecordInspectOptions{Payload: visibility})
}

// InspectFailureWithOptions returns a failure with explicit privileged fields.
func (c *Client) InspectFailureWithOptions(
	ctx context.Context,
	tenant string,
	id string,
	options RecordInspectOptions,
) (apihttp.Record, error) {
	return c.inspectRecord(ctx, tenant, "failures", id, options)
}

// InspectDeadLetter returns one tenant dead letter at the requested visibility.
func (c *Client) InspectDeadLetter(
	ctx context.Context,
	tenant string,
	id string,
	visibility queue.PayloadVisibility,
) (apihttp.Record, error) {
	return c.InspectDeadLetterWithOptions(ctx, tenant, id, RecordInspectOptions{Payload: visibility})
}

// InspectDeadLetterWithOptions returns a dead letter with explicit privileged fields.
func (c *Client) InspectDeadLetterWithOptions(
	ctx context.Context,
	tenant string,
	id string,
	options RecordInspectOptions,
) (apihttp.Record, error) {
	return c.inspectRecord(ctx, tenant, "dead-letters", id, options)
}

func (c *Client) listRecords(
	ctx context.Context,
	tenant string,
	collection string,
	query RecordQuery,
) (apihttp.RecordPage, error) {
	if invalidIdentity(tenant) || !validRecordQuery(query) {
		return apihttp.RecordPage{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, collection)
	values := endpoint.Query()
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.FormatUint(uint64(query.Limit), 10))
	}
	if query.Search != "" {
		values.Set("search", query.Search)
	}
	if query.Sort != "" {
		values.Set("sort", string(query.Sort))
	}
	if query.Direction != "" {
		values.Set("direction", string(query.Direction))
	}
	endpoint.RawQuery = values.Encode()

	var page apihttp.RecordPage
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return apihttp.RecordPage{}, err
	}

	return page, nil
}

func (c *Client) inspectRecord(
	ctx context.Context,
	tenant string,
	collection string,
	id string,
	options RecordInspectOptions,
) (apihttp.Record, error) {
	if invalidIdentity(tenant) || invalidIdentity(id) || !validPayloadVisibility(options.Payload) {
		return apihttp.Record{}, ErrInvalidRequest
	}
	endpoint := c.baseURL.JoinPath("v1", "tenants", tenant, collection, id)
	if options.Payload != queue.PayloadHidden || options.RevealDiagnostics {
		values := endpoint.Query()
		if options.Payload != queue.PayloadHidden {
			values.Set("payload", string(options.Payload))
		}
		if options.RevealDiagnostics {
			values.Set("diagnostics", "revealed")
		}
		endpoint.RawQuery = values.Encode()
	}
	var record apihttp.Record
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &record); err != nil {
		return apihttp.Record{}, err
	}

	return record, nil
}

func validRecordQuery(query RecordQuery) bool {
	if query.Limit > queue.MaxPageSize || len(query.Cursor) > queue.MaxCursorBytes ||
		len(query.Search) > queue.MaxSearchBytes {
		return false
	}
	if query.Sort != "" && query.Sort != queue.SortOccurredAt &&
		query.Sort != queue.SortQueue && query.Sort != queue.SortAttempts {
		return false
	}

	return query.Direction == "" || query.Direction == queue.SortAscending ||
		query.Direction == queue.SortDescending
}

func validPayloadVisibility(visibility queue.PayloadVisibility) bool {
	return visibility == queue.PayloadHidden || visibility == queue.PayloadRedacted ||
		visibility == queue.PayloadRevealed
}

func (c *Client) do(
	ctx context.Context,
	method string,
	endpoint *url.URL,
	input any,
	output any,
) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("control-plane client: encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("control-plane client: create request: %w", err)
	}
	if err := c.authenticate(ctx, request); err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("control-plane client: request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	encoded, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("control-plane client: read response: %w", err)
	}
	if int64(len(encoded)) > c.maxResponseBytes {
		return ErrResponseTooLarge
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		problem := apihttp.Problem{Code: "http_error"}
		_ = json.Unmarshal(encoded, &problem)
		return &APIError{Status: response.StatusCode, Code: problem.Code}
	}
	if err := json.Unmarshal(encoded, output); err != nil {
		return fmt.Errorf("control-plane client: decode response: %w", err)
	}

	return nil
}

func (c *Client) authenticate(ctx context.Context, request *http.Request) error {
	if !nilInterface(c.apiKeys) {
		id, secret, err := c.apiKeys.APIKey(ctx)
		if err != nil {
			return err
		}
		if strings.TrimSpace(id) == "" || strings.TrimSpace(secret) == "" {
			return ErrInvalidAPIKey
		}
		request.Header.Set(APIKeyIDHeader, id)
		request.Header.Set(APIKeySecretHeader, secret)

		return nil
	}

	token, err := c.tokens.Token(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return ErrInvalidToken
	}
	request.Header.Set("Authorization", "Bearer "+token)

	return nil
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Pointer && reflected.IsNil()
}

func invalidIdentity(value string) bool {
	return strings.TrimSpace(value) == "" || len(value) > controlplane.MaxIdentityBytes
}
