package control

import (
	"context"
	"errors"
	"reflect"

	controlplane "github.com/faustbrian/go-queue-control-plane/v6"
)

var (
	// ErrInvalidDispatcherConfiguration reports a missing dispatch boundary.
	ErrInvalidDispatcherConfiguration = errors.New("control: invalid dispatcher configuration")
	// ErrDataPlaneUnavailable reports that no queue controller is reachable.
	ErrDataPlaneUnavailable = errors.New("control: data plane unavailable")
)

// UnavailableDispatcher explicitly represents a disconnected data plane.
type UnavailableDispatcher struct{}

// DispatchResult fails closed without issuing any backend operation.
func (UnavailableDispatcher) DispatchResult(context.Context, controlplane.Command) (DispatchOutcome, error) {
	return DispatchOutcome{}, ErrDataPlaneUnavailable
}

// RoutingDispatcher keeps Kubernetes scaling separate from queue control
// commands.
type RoutingDispatcher struct {
	dataPlane Dispatcher
	workloads Dispatcher
}

// NewRoutingDispatcher creates an action router with explicit boundaries.
func NewRoutingDispatcher(dataPlane Dispatcher, workloads Dispatcher) (*RoutingDispatcher, error) {
	if nilDispatcher(dataPlane) || nilDispatcher(workloads) {
		return nil, ErrInvalidDispatcherConfiguration
	}

	return &RoutingDispatcher{dataPlane: dataPlane, workloads: workloads}, nil
}

func nilDispatcher(dispatcher Dispatcher) bool {
	if dispatcher == nil {
		return true
	}
	reflected := reflect.ValueOf(dispatcher)

	return reflected.Kind() == reflect.Pointer && reflected.IsNil()
}

// DispatchResult routes scaling to Kubernetes and every data-plane command to
// its queue adapter.
func (dispatcher *RoutingDispatcher) DispatchResult(
	ctx context.Context,
	command controlplane.Command,
) (DispatchOutcome, error) {
	return dispatcher.selected(command).DispatchResult(ctx, command)
}

func (dispatcher *RoutingDispatcher) selected(command controlplane.Command) Dispatcher {
	if command.Action == controlplane.ActionScale {
		return dispatcher.workloads
	}

	return dispatcher.dataPlane
}
