package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type r30CompletionValue struct{}
type r30AdmissionDeadlineProbe struct {
	observed context.Context
	bounded  bool
	deadline time.Time
	value    any
	cause    error
}

func (p *r30AdmissionDeadlineProbe) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	p.observed = ctx
	deadline, ok := ctx.Deadline()
	p.bounded = ok && time.Until(deadline) > 0
	p.deadline = deadline
	p.value = ctx.Value(r30CompletionValue{})
	return nil, p.cause
}

// The actual public SQL journal must supply a cooperative completion bound
// even when Service intentionally detached request cancellation. No database
// or fabricated private transaction state is required for this I/O admission.
func TestR30SQLJournalOwnsDetachedCompletionAdmissionDeadline(t *testing.T) {
	cause := errors.New("owned admission probe refused transaction")
	probe := &r30AdmissionDeadlineProbe{cause: cause}
	journal, err := NewJournal(probe)
	if err != nil {
		t.Fatal(err)
	}
	request, cancel := context.WithCancel(context.WithValue(context.Background(), r30CompletionValue{}, "operation-value"))
	cancel()
	started := time.Now()
	err = journal.Complete(context.WithoutCancel(request), journalResult())
	if !errors.Is(err, cause) {
		t.Fatalf("SQL admission cause lost: %v", err)
	}
	if probe.observed.Err() != context.Canceled {
		t.Fatal("completion retained its owned timer context after admission failure")
	}
	if !probe.bounded || probe.deadline.After(started.Add(5*time.Second+100*time.Millisecond)) || probe.value != "operation-value" {
		t.Fatalf("detached completion acquired unbounded SQL admission or lost values: bounded=%t value=%v", probe.bounded, probe.value)
	}
}

func TestR30SQLJournalPreservesShorterCallerDeadlineAndValues(t *testing.T) {
	cause := errors.New("owned shorter admission probe refused transaction")
	probe := &r30AdmissionDeadlineProbe{cause: cause}
	journal, err := NewJournal(probe)
	if err != nil {
		t.Fatal(err)
	}
	caller, cancel := context.WithTimeout(context.WithValue(context.Background(), r30CompletionValue{}, "operation-value"), 500*time.Millisecond)
	defer cancel()
	deadline, _ := caller.Deadline()
	err = journal.Complete(caller, journalResult())
	if !errors.Is(err, cause) || !probe.bounded || !probe.deadline.Equal(deadline) || probe.value != "operation-value" {
		t.Fatalf("shorter caller deadline/cause/values changed: error=%v bounded=%t deadline=%v", err, probe.bounded, probe.deadline)
	}
	if caller.Err() != nil {
		t.Fatal("completion cancelled or consumed caller context")
	}
}
