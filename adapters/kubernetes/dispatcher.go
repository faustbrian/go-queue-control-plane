package kubernetes

import (
	"context"
	"errors"
	"time"

	controlplane "github.com/faustbrian/go-queue-control-plane/v6"
	"github.com/faustbrian/go-queue-control-plane/v6/control"
)

var (
	// ErrInvalidDispatcher reports a missing tenant-to-cluster resolver.
	ErrInvalidDispatcher = errors.New("kubernetes: invalid dispatcher")
	// ErrUnsupportedCommand reports a non-scaling command at this boundary.
	ErrUnsupportedCommand = errors.New("kubernetes: unsupported command")
)

// Scaler is the scale-only workload mutation exposed to the dispatcher.
type Scaler interface {
	// Scale updates the named workload replicas honoring ctx;
	// authorization/confirmation must happen before this borrowed provider
	// call.
	Scale(context.Context, string, uint32) (ScaleResult, error)
}

// TenantResolver maps a tenant to its namespace-scoped Kubernetes adapter.
type TenantResolver interface {
	// ResolveTenant returns the borrowed namespace-scoped scaler for a tenant;
	// errors refuse dispatch.
	ResolveTenant(context.Context, string) (Scaler, error)
}

// ScaleDispatcher sends validated scaling commands to a tenant-scoped
// Kubernetes adapter.
type ScaleDispatcher struct {
	resolver TenantResolver
}

// NewScaleDispatcher creates a scale-only Kubernetes command dispatcher.
func NewScaleDispatcher(resolver TenantResolver) (*ScaleDispatcher, error) {
	if nilInterface(resolver) {
		return nil, ErrInvalidDispatcher
	}

	return &ScaleDispatcher{resolver: resolver}, nil
}

// DispatchResult resolves the tenant and updates only the workload scale
// subresource. Kubernetes acknowledges the accepted replica update, so a
// returned scale reports success completed when the update returned.
func (dispatcher *ScaleDispatcher) DispatchResult(
	ctx context.Context,
	command controlplane.Command,
) (control.DispatchOutcome, error) {
	if err := command.Validate(); err != nil {
		return control.DispatchOutcome{}, err
	}
	if command.Action != controlplane.ActionScale {
		return control.DispatchOutcome{}, ErrUnsupportedCommand
	}

	scaler, err := dispatcher.resolver.ResolveTenant(ctx, command.TenantID)
	if err != nil {
		return control.DispatchOutcome{}, err
	}
	if nilInterface(scaler) {
		return control.DispatchOutcome{}, ErrInvalidDispatcher
	}
	if _, err := scaler.Scale(ctx, command.Target.Name, command.Scale.Replicas); err != nil {
		return control.DispatchOutcome{}, err
	}

	return control.DispatchOutcome{Status: controlplane.CommandSucceeded, CompletedAt: time.Now()}, nil
}
