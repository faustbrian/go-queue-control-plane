//lint:file-ignore SA1019 Delegation preserves the released v1 type identities.

// Package kubernetes exposes the target-oriented entry point for the narrow
// Kubernetes Deployment integration.
package kubernetes

import (
	"io"

	legacy "github.com/faustbrian/go-queue-control-plane/kubernetes"
)

var (
	// ErrInvalidConfiguration reports an unscoped Kubernetes adapter.
	ErrInvalidConfiguration = legacy.ErrInvalidConfiguration
	// ErrInvalidWorkload reports a malformed Kubernetes Deployment name.
	ErrInvalidWorkload = legacy.ErrInvalidWorkload
	// ErrInvalidPage reports an unbounded Deployment list request.
	ErrInvalidPage = legacy.ErrInvalidPage
	// ErrInvalidResponse reports malformed or cross-namespace Kubernetes state.
	ErrInvalidResponse = legacy.ErrInvalidResponse
	// ErrInvalidTenantDocument is a stable Kubernetes mapping error.
	ErrInvalidTenantDocument = legacy.ErrInvalidTenantDocument
	// ErrInvalidDispatcher reports a missing tenant-to-cluster resolver.
	ErrInvalidDispatcher = legacy.ErrInvalidDispatcher
	// ErrUnsupportedCommand reports a non-scaling command at this boundary.
	ErrUnsupportedCommand = legacy.ErrUnsupportedCommand
	// ErrInvalidResolver reports an empty or malformed tenant mapping.
	ErrInvalidResolver = legacy.ErrInvalidResolver
	// ErrTenantNotConfigured reports a tenant without Kubernetes visibility.
	ErrTenantNotConfigured = legacy.ErrTenantNotConfigured
)

const (
	// MaxPageSize bounds one Kubernetes Deployment request.
	MaxPageSize = legacy.MaxPageSize
	// MaxContinueTokenBytes bounds the opaque Kubernetes pagination token.
	MaxContinueTokenBytes = legacy.MaxContinueTokenBytes
)

// DeploymentClient is the deliberately restricted part of a namespace-scoped
// Kubernetes Deployment client used by the control plane.
type DeploymentClient = legacy.DeploymentClient

// Adapter reports Deployment status from one configured namespace.
type Adapter = legacy.Adapter

// Status is the bounded Deployment state presented by the control plane.
type Status = legacy.Status

// Page is one bounded page of Deployment status.
type Page = legacy.Page

// ScaleResult is the Kubernetes acknowledgement of an authorized replica
// update.
type ScaleResult = legacy.ScaleResult

// DeploymentFactory returns the restricted Deployment client for one
// namespace.
type DeploymentFactory = legacy.DeploymentFactory

// Scaler is the scale-only workload mutation exposed to the dispatcher.
type Scaler = legacy.Scaler

// TenantResolver maps a tenant to its namespace-scoped Kubernetes adapter.
type TenantResolver = legacy.TenantResolver

// ScaleDispatcher sends validated scaling commands to a tenant-scoped
// Kubernetes adapter.
type ScaleDispatcher = legacy.ScaleDispatcher

// TenantAdapter is the complete bounded Kubernetes surface for one tenant.
type TenantAdapter = legacy.TenantAdapter

// StaticTenantResolver stores an immutable tenant-to-namespace mapping.
type StaticTenantResolver = legacy.StaticTenantResolver

// New creates an adapter scoped to namespace.
func New(namespace string, client DeploymentClient) (*Adapter, error) {
	return legacy.New(namespace, client)
}

// LoadTenantResolver builds immutable namespace-scoped adapters from one
// strict, bounded tenant document.
func LoadTenantResolver(
	reader io.Reader,
	maxBytes int64,
	factory DeploymentFactory,
) (*StaticTenantResolver, error) {
	return legacy.LoadTenantResolver(reader, maxBytes, factory)
}

// NewScaleDispatcher creates a scale-only Kubernetes command dispatcher.
func NewScaleDispatcher(resolver TenantResolver) (*ScaleDispatcher, error) {
	return legacy.NewScaleDispatcher(resolver)
}

// NewStaticTenantResolver copies a fixed tenant adapter mapping.
func NewStaticTenantResolver(configured map[string]TenantAdapter) (*StaticTenantResolver, error) {
	return legacy.NewStaticTenantResolver(configured)
}
