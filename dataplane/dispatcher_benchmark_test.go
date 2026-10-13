package dataplane

import (
	"errors"
	"testing"
	"time"

	queue "github.com/faustbrian/go-queue/management"
)

// BenchmarkControllerDispatchBackendOutage measures validation and a synthetic
// resolver refusal after fixture setup; it does not measure a live backend,
// remote outage detection or timeout latency.
func BenchmarkControllerDispatchBackendOutage(b *testing.B) {
	command := validControlCommand()
	dispatcher, err := NewControllerDispatcher(
		&controllerResolverStub{err: errors.New("backend unavailable")},
		queue.ProtocolVersion{Major: 1},
		time.Minute,
		func() time.Time { return command.RequestedAt.Add(time.Second) },
	)
	if err != nil {
		b.Fatalf("NewControllerDispatcher() error = %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := dispatcher.DispatchResult(b.Context(), command); err == nil {
			b.Fatal("DispatchResult() error = nil")
		}
	}
}
