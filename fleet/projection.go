package fleet

import (
	"cmp"
	"slices"
	"sync"
	"time"

	queue "github.com/faustbrian/go-queue/v2"
)

// QueueMetrics is a bounded current-state projection of lifecycle events.
// Runtime is cumulative so a metrics backend can derive honest averages and
// rates without this package retaining an unbounded time series.
type QueueMetrics struct {
	// Enqueued counts observed EventEnqueued events since construction.
	Enqueued uint64
	// Succeeded counts observed successful handler events, not durable
	// settlement.
	Succeeded uint64
	// Failed counts observed failed handler events.
	Failed uint64
	// Runtime accumulates handler success/failure durations; it is not a per-job average.
	Runtime time.Duration
}

// QueueSnapshot identifies one queue's current projected counters.
type QueueSnapshot struct {
	// Backend identifies the event source backend, not a connection handle.
	Backend string
	// Queue identifies the event-reported queue.
	Queue string
	// Metrics contains copied cumulative counters for this identity.
	Metrics QueueMetrics
}

// ProjectionSnapshot is an independently owned point-in-time fleet projection.
type ProjectionSnapshot struct {
	// Queues is a newly owned backend/queue-sorted slice; caller mutation does
	// not alter Projection.
	Queues []QueueSnapshot
	// Overflow aggregates events for identities refused at the configured
	// cardinality limit.
	Overflow QueueMetrics
}

// Projection consumes queue lifecycle events without accessing a backend.
// New queue identities beyond the configured limit are aggregated into the
// overflow bucket to bound retained identity cardinality. Event label bytes and
// numeric overflow are not separately constrained; callers own trusted input.
// Construct with NewProjection and do not copy after the mutex is used.
type Projection struct {
	mu        sync.RWMutex
	maxQueues int
	queues    map[queueKey]QueueMetrics
	overflow  QueueMetrics
}

type queueKey struct {
	backend string
	queue   string
}

// NewProjection creates a projection that retains at most maxQueues distinct
// queue identities. Non-positive limits retain no identities and aggregate all
// events into the overflow bucket.
func NewProjection(maxQueues int) *Projection {
	maxQueues = max(maxQueues, 0)

	return &Projection{
		maxQueues: maxQueues,
		queues:    make(map[queueKey]QueueMetrics, maxQueues),
	}
}

// Observe implements queue.Observer.
func (p *Projection) Observe(event queue.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := queueKey{backend: event.Backend, queue: event.Queue}
	metrics, exists := p.queues[key]
	if !exists && len(p.queues) >= p.maxQueues {
		p.overflow.apply(event)

		return
	}

	metrics.apply(event)
	p.queues[key] = metrics
}

// Snapshot returns a deterministic copy safe for concurrent readers.
func (p *Projection) Snapshot() ProjectionSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	snapshot := ProjectionSnapshot{
		Queues:   make([]QueueSnapshot, 0, len(p.queues)),
		Overflow: p.overflow,
	}
	for key, metrics := range p.queues {
		snapshot.Queues = append(snapshot.Queues, QueueSnapshot{
			Backend: key.backend,
			Queue:   key.queue,
			Metrics: metrics,
		})
	}
	slices.SortFunc(snapshot.Queues, func(left, right QueueSnapshot) int {
		return cmp.Or(
			cmp.Compare(left.Backend, right.Backend),
			cmp.Compare(left.Queue, right.Queue),
		)
	})

	return snapshot
}

func (m *QueueMetrics) apply(event queue.Event) {
	switch event.Kind {
	case queue.EventEnqueued:
		m.Enqueued++
	case queue.EventHandlerSucceeded:
		m.Succeeded++
		m.Runtime += event.Duration
	case queue.EventHandlerFailed:
		m.Failed++
		m.Runtime += event.Duration
	default:
	}
}

var _ queue.Observer = (*Projection)(nil)
