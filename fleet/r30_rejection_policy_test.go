package fleet

import (
	"fmt"
	"testing"
	"time"
)

// Selected v4 policy: detail belongs only to registered worker tenants. The
// public views prove conservation and outcome semantics; the owned map count
// checks retained resource cardinality without manufacturing internal state.
func TestR30RejectedTenantDetailIsBoundedByRegisteredRoster(t *testing.T) {
	registry := NewRegistry(2)
	at := time.Unix(1, 0)
	for _, tenant := range []string{"registered-a", "registered-b"} {
		if registry.Upsert(validHeartbeat(tenant, "same-worker", at)) != HeartbeatAccepted {
			t.Fatal("registered roster control not admitted")
		}
	}
	// Worker identity is (tenant, worker), not a reassignment of the first
	// tenant's registration. Both original registrations remain retained.
	if registry.Upsert(validHeartbeat("registered-a", "another-worker", at)) != HeartbeatCapacityExceeded {
		t.Fatal("full registry admitted extra registered-tenant worker")
	}
	var unknownDetails uint64
	for index := range 32 {
		tenant := fmt.Sprintf("never-registered-%02d", index)
		if registry.Upsert(validHeartbeat(tenant, "same-worker", at)) != HeartbeatCapacityExceeded {
			t.Fatal("full roster admitted a new tenant generation")
		}
		view := registry.SnapshotTenant(tenant, at, time.Minute)
		if len(view.Workers) != 0 {
			t.Fatal("refused tenant acquired worker storage")
		}
		unknownDetails += view.Rejected
	}
	all := registry.Snapshot(at, time.Minute)
	if len(all.Workers) != 2 || all.Rejected != 33 {
		t.Fatalf("global rejection conservation/registered roster changed: %+v", all)
	}
	known := registry.SnapshotTenant("registered-a", at, time.Minute)
	if len(known.Workers) != 1 || known.Rejected != 1 {
		t.Fatal("registered tenant's exact detail lost")
	}
	if len(registry.rejected) > 2 {
		t.Fatalf("rejected metadata grew beyond registered tenant roster: %d keys", len(registry.rejected))
	}
	if unknownDetails != 0 {
		t.Fatalf("unregistered tenants retained rejection details: %d", unknownDetails)
	}
	if registry.Upsert(validHeartbeat("registered-a", "same-worker", at.Add(time.Second))) != HeartbeatAccepted {
		t.Fatal("same-identity later heartbeat was not reusable")
	}
	if len(registry.Snapshot(at.Add(time.Second), time.Minute).Workers) != 2 || len(registry.rejected) > 2 {
		t.Fatal("re-registration changed roster/detail cardinality")
	}
}

func TestR30ZeroWorkerRosterKeepsOnlyGlobalRejectionAccounting(t *testing.T) {
	registry := NewRegistry(0)
	at := time.Unix(1, 0)
	for index := range 16 {
		if registry.Upsert(validHeartbeat(fmt.Sprintf("unknown-%d", index), "worker", at)) != HeartbeatCapacityExceeded {
			t.Fatal("zero-capacity registry admitted a worker")
		}
	}
	if got := registry.Snapshot(at, time.Minute); len(got.Workers) != 0 || got.Rejected != 16 {
		t.Fatalf("zero-roster global accounting changed: %+v", got)
	}
	if len(registry.rejected) != 0 || registry.SnapshotTenant("unknown-0", at, time.Minute).Rejected != 0 {
		t.Fatal("zero roster retained unknown-tenant detail")
	}
}

func TestR30RejectionKnowledgeDistinguishesRegisteredZeroFromUnknown(t *testing.T) {
	registry := NewRegistry(1)
	at := time.Unix(1, 0)
	if registry.Upsert(validHeartbeat("registered", "worker", at)) != HeartbeatAccepted {
		t.Fatal("valid registration refused")
	}
	known := registry.SnapshotTenant("registered", at, time.Minute)
	unknown := registry.SnapshotTenant("never-registered", at, time.Minute)
	global := registry.Snapshot(at, time.Minute)
	if !known.RejectedKnown || known.Rejected != 0 || unknown.RejectedKnown || unknown.Rejected != 0 || !global.RejectedKnown || global.Rejected != 0 {
		t.Fatal("exact zero and unknown detail were conflated")
	}
	if registry.Upsert(validHeartbeat("never-registered", "worker", at)) != HeartbeatCapacityExceeded {
		t.Fatal("capacity control admitted unknown")
	}
	if registry.SnapshotTenant("never-registered", at, time.Minute).RejectedKnown || registry.Snapshot(at, time.Minute).Rejected != 1 {
		t.Fatal("unknown rejection lost global conservation or became known")
	}
}
