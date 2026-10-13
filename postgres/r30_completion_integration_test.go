//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	gopostgres "github.com/faustbrian/go-postgres/v2"
	controlplane "github.com/faustbrian/go-queue-control-plane/v4"
	controlpostgres "github.com/faustbrian/go-queue-control-plane/v4/postgres"
)

// A real row lock causes cooperative completion timeout before result/audit
// publication. Reuse after rollback proves this is not a falsely known commit.
func TestR30PostgresCompletionTimeoutRollsBackAndDetachedCompletionPersists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL required for owned integration")
	}
	if err := migrateIntegrationDatabase(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := gopostgres.Connect(ctx, integrationPoolConfig(dsn))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err := pool.Shutdown(cleanup); err != nil {
			t.Error(err)
		}
	}()
	runtime, err := controlpostgres.NewRuntimeWithPool(pool)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	command := controlplane.Command{TenantID: "r30-owned-completion", IdempotencyKey: "completion", Actor: "operator", Reason: "owned completion integration", Action: controlplane.ActionPause, Target: controlplane.Target{Kind: controlplane.TargetQueue, Name: "critical"}, RequestedAt: at}
	accepted, created, err := runtime.Journal.Accept(ctx, command)
	if err != nil || !created {
		t.Fatalf("admission control failed: created=%t error=%v", created, err)
	}
	before, err := runtime.Audit.VerifyTenant(ctx, command.TenantID, 10)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := pool.Raw().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = lock.Rollback(cleanup)
	}()
	if _, err := lock.Exec(ctx, "SELECT 1 FROM queue_control_commands WHERE tenant_id=$1 AND idempotency_key=$2 FOR UPDATE", command.TenantID, command.IdempotencyKey); err != nil {
		t.Fatal(err)
	}
	terminal := controlplane.CommandResult{CommandID: accepted.CommandID, TenantID: command.TenantID, IdempotencyKey: command.IdempotencyKey, Status: controlplane.CommandCanceled, Failure: controlplane.FailureCanceled, CompletedAt: at.Add(time.Second)}
	short, finish := context.WithTimeout(ctx, 150*time.Millisecond)
	err = runtime.Journal.Complete(short, terminal)
	finish()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked completion did not preserve caller timeout: %v", err)
	}
	if err := lock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	stored, err := runtime.Commands.Get(ctx, command.TenantID, command.IdempotencyKey)
	if err != nil || stored.Status != accepted.Status {
		t.Fatalf("timed out completion falsely published terminal state: %+v %v", stored, err)
	}
	after, err := runtime.Audit.VerifyTenant(ctx, command.TenantID, 10)
	if err != nil || after != before {
		t.Fatalf("timed out completion changed audit head: %+v %v", after, err)
	}
	request, stop := context.WithCancel(ctx)
	stop()
	if err := runtime.Journal.Complete(context.WithoutCancel(request), terminal); err != nil {
		t.Fatal(err)
	}
	stored, err = runtime.Commands.Get(ctx, command.TenantID, command.IdempotencyKey)
	if err != nil || stored.Status != terminal.Status || stored.Failure != terminal.Failure {
		t.Fatalf("detached cancellation completion lost durable result: %+v %v", stored, err)
	}
	after, err = runtime.Audit.VerifyTenant(ctx, command.TenantID, 10)
	if err != nil || after.Events != before.Events+1 || after.HeadHash == before.HeadHash {
		t.Fatalf("durable completion did not atomically append audit: %+v %v", after, err)
	}
}
