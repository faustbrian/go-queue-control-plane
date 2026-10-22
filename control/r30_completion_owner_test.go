package control

import (
	"context"
	"errors"
	"testing"
	"time"

	controlplane "github.com/faustbrian/go-queue-control-plane/v5"
)

type r30ContextKey struct{}
type r30BoundedCompletionJournal struct {
	journalStub
	cancel   context.CancelFunc
	fail     bool
	observed bool
}

func (j *r30BoundedCompletionJournal) Accept(ctx context.Context, command controlplane.Command) (controlplane.CommandResult, bool, error) {
	result, created, err := j.journalStub.Accept(ctx, command)
	j.cancel() // Admission succeeded; cancellation now prevents dispatch.
	return result, created, err
}
func (j *r30BoundedCompletionJournal) Complete(ctx context.Context, result controlplane.CommandResult) error {
	_, deadline := ctx.Deadline()
	j.observed = ctx.Err() == nil && ctx.Done() == nil && !deadline && ctx.Value(r30ContextKey{}) == "operation-value"
	if !j.observed {
		return errors.New("completion did not receive cancellation-independent values")
	}
	owned, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	if j.fail {
		<-owned.Done()
		return owned.Err()
	}
	return j.journalStub.Complete(owned, result)
}

// This supplies an explicitly bounded cooperative journal, not proof that the
// Service can preempt arbitrary borrowed I/O or that PostgreSQL has a total bound.
func TestR30PreDispatchCancellationUsesJournalOwnedCompletionBound(t *testing.T) {
	for _, failure := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), r30ContextKey{}, "operation-value"), time.Second)
		journal := &r30BoundedCompletionJournal{journalStub: journalStub{created: true}, cancel: cancel, fail: failure}
		dispatcher := &dispatcherStub{}
		service := NewService(&authorizerStub{}, journal, dispatcher, time.Now)
		result, err := service.Execute(ctx, validCommand())
		cancel()
		if !journal.observed || !errors.Is(err, context.Canceled) || result.Status != controlplane.CommandCanceled || result.Failure != controlplane.FailureCanceled {
			t.Fatalf("accepted cancellation outcome lost: result=%v error=%v observed=%t", result, err, journal.observed)
		}
		if dispatcher.calls != 0 || journal.dispatchCalls != 0 || journal.acknowledgeCalls != 0 {
			t.Fatal("cancelled accepted command crossed dispatch boundary")
		}
		if failure {
			if !errors.Is(err, context.DeadlineExceeded) || journal.completeCalls != 0 {
				t.Fatal("journal-owned completion timeout was hidden or falsely persisted")
			}
		} else if journal.completeCalls != 1 || journal.completed != result {
			t.Fatal("cancellation completion did not persist the actual terminal result")
		}
	}
}
