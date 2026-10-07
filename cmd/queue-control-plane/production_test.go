package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	controlplane "github.com/faustbrian/go-queue-control-plane/v3"
	"github.com/faustbrian/go-queue-control-plane/v3/control"
	"github.com/faustbrian/go-queue-control-plane/v3/server"
)

func TestProductionDependenciesUseBoundedImplementations(t *testing.T) {
	t.Parallel()

	dependencies := productionDependencies()
	if invalidProcessDependencies(dependencies) {
		t.Fatal("productionDependencies() returned an incomplete graph")
	}
	build, err := dependencies.buildInfo()
	if err != nil || build.Version != "dev" || build.Commit != "unknown" {
		t.Fatalf("buildInfo() = (%+v, %v)", build, err)
	}
	runtime, err := dependencies.buildTelemetry(context.Background(), Config{}, build)
	if err != nil || runtime != nil {
		t.Fatalf("disabled buildTelemetry() = (%v, %v), want nil runtime", runtime, err)
	}
	telemetryConfig := Config{
		TelemetryEnabled:  true,
		TelemetryEndpoint: "127.0.0.1:4317",
		TelemetryProtocol: "grpc",
		TelemetryInsecure: true,
	}
	cancelledBuild, cancelBuild := context.WithCancel(context.Background())
	cancelBuild()
	runtime, err = dependencies.buildTelemetry(cancelledBuild, telemetryConfig, build)
	if runtime != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("buildTelemetry(cancelled) = (%v, %v), want context canceled", runtime, err)
	}
	runtime, err = dependencies.buildTelemetry(context.Background(), telemetryConfig, build)
	if err != nil || runtime == nil {
		t.Fatalf("buildTelemetry() = (%v, %v), want runtime", runtime, err)
	}
	cancelledTelemetry, cancelTelemetry := context.WithCancel(context.Background())
	cancelTelemetry()
	if err := runtime.Shutdown(cancelledTelemetry); err == nil {
		t.Fatal("cancelled telemetry Shutdown() returned nil")
	}
	cleanup, cancelCleanup := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelCleanup()
	if err := runtime.Shutdown(cleanup); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown(after canceled attempt) error = %v", err)
	}

	limiter, err := dependencies.buildRateLimiter()
	if err != nil || limiter == nil {
		t.Fatalf("buildRateLimiter() = (%v, %v), want limiter and nil", limiter, err)
	}
	for range 120 {
		if !limiter.Allow(context.Background(), "subject:operator") {
			t.Fatal("production limiter rejected within configured allowance")
		}
	}
	if limiter.Allow(context.Background(), "subject:operator") {
		t.Fatal("production limiter exceeded configured allowance")
	}
	if routed, err := dependencies.routeDispatchers(
		control.UnavailableDispatcher{},
		applicationDispatcher{},
	); err != nil || routed == nil {
		t.Fatalf("routeDispatchers() = (%v, %v), want dispatcher and nil", routed, err)
	}

	if pool, err := dependencies.openPool(context.Background(), postgresPoolConfig("://")); err == nil || pool != nil {
		t.Fatalf("openPool(malformed) = (%v, %v), want safe parse failure", pool, err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := dependencies.migrate(cancelled, "postgres://localhost/control_plane"); !errors.Is(err, context.Canceled) {
		t.Fatalf("migrate(cancelled) error = %v, want context canceled", err)
	}

	listener, err := dependencies.listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen() error = %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	process, err := dependencies.buildServer(listener, http.NotFoundHandler(), server.Config{})
	if err != nil || process == nil {
		t.Fatalf("buildServer() = (%v, %v), want server and nil", process, err)
	}
	if err := dependencies.dispatcher.Dispatch(context.Background(), controlplane.Command{}); !errors.Is(err, control.ErrDataPlaneUnavailable) {
		t.Fatalf("dispatcher error = %v, want unavailable", err)
	}
}
