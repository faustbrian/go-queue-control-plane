package postgres

import (
	"context"
	"errors"
	"testing"

	gopostgres "github.com/faustbrian/go-postgres/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewRuntimeRejectsMissingPool(t *testing.T) {
	t.Parallel()

	for _, pool := range []*gopostgres.Pool{nil, {}} {
		runtime, err := NewRuntime(pool)
		if runtime != nil || !errors.Is(err, ErrInvalidRuntimePool) {
			t.Fatalf("NewRuntime(invalid) = (%v, %v), want nil and ErrInvalidRuntimePool", runtime, err)
		}
	}
}

func TestNewRuntimeUsesNativePersistenceAndBoundedReadiness(t *testing.T) {
	t.Parallel()
	pool, err := gopostgres.Connect(context.Background(), gopostgres.Config{
		DSN: "postgres://localhost/control_plane",
		ResolveDSN: func(_ context.Context, dsn string) (*gopostgres.PoolConfig, error) {
			return pgxpool.ParseConfig(dsn)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Shutdown(context.Background()) })
	runtime, err := NewRuntime(pool)
	if err != nil {
		t.Fatal(err)
	}
	runner, ok := runtime.Journal.runner.(*postgresTransactionRunner)
	if !ok || runner.beginner != pool.Raw() || runtime.Audit.beginner != pool.Raw() ||
		runtime.Commands.beginner != pool.Raw() || runtime.Desired.queryer != pool.Raw() ||
		runtime.Readiness.pool != pool {
		t.Fatal("runtime must share the native pool for persistence and the bounded pool for readiness")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.Readiness.Ready(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ready(cancelled) = %v", err)
	}
	if err := pool.Liveness().Err; err != nil {
		t.Fatalf("caller-owned pool unexpectedly closed: %v", err)
	}
}
