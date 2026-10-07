package apihttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/faustbrian/go-queue-control-plane/v3/fleet"
)

func TestWorkerResponsePreservesSourceCollectionsAndOrdering(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	queues := []string{"queue-z", "queue-a"}
	capabilities := []fleet.Capability{fleet.CapabilityPause, fleet.CapabilityDrain}
	workers := []fleet.WorkerSnapshot{
		{Heartbeat: workerHeartbeat("tenant-1", "worker-b", now, queues), State: fleet.StateRunning},
		{Heartbeat: workerHeartbeat("tenant-1", "worker-a", now, queues), State: fleet.StateRunning},
	}
	for index := range workers {
		workers[index].Capabilities = slices.Clone(capabilities)
	}
	source := &workerSourceStub{snapshot: fleet.RegistrySnapshot{Workers: workers, Rejected: 3}}
	handler, err := NewHandler(Config{
		Commands: &commandExecutorStub{}, Workers: source, Viewer: &viewerStub{},
		Now: func() time.Time { return now }, StaleAfter: time.Minute,
		Protocol:           fleet.ProtocolRange{Minimum: fleet.ProtocolVersion{Major: 1}, Maximum: fleet.ProtocolVersion{Major: 1}},
		WorkerCapabilities: []fleet.Capability{fleet.CapabilityDrain},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, authenticatedRequest(t, http.MethodGet, "/v1/tenants/tenant-1/workers?limit=2", ""))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d", response.Code)
		}
		var page WorkerPage
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Workers) != 2 || page.Workers[0].WorkerID != "worker-a" || page.Workers[1].WorkerID != "worker-b" || page.Rejected != 3 || page.NextCursor != "" {
			t.Fatalf("incorrect page: %+v", page)
		}
		for _, worker := range page.Workers {
			if !slices.Equal(worker.Queues, queues) || !slices.Equal(worker.Capabilities, capabilities) ||
				!slices.Equal(worker.Compatibility.Enabled, []fleet.Capability{fleet.CapabilityDrain}) {
				t.Fatalf("collections changed in projection: %+v", worker)
			}
		}
		page.Workers[0].Queues[0] = "decoded-response-only"
		page.Workers[0].Capabilities[0] = fleet.CapabilityDrain
		if source.snapshot.Workers[0].WorkerID != "worker-b" || source.snapshot.Workers[1].WorkerID != "worker-a" {
			t.Fatal("response sorting mutated source ordering")
		}
		for _, worker := range source.snapshot.Workers {
			if !slices.Equal(worker.Queues, queues) || !slices.Equal(worker.Capabilities, capabilities) {
				t.Fatal("response construction mutated source collections")
			}
		}
	}
}

func TestWorkerProjectionPreservesNullEmptyCollections(t *testing.T) {
	t.Parallel()
	h := &handler{}
	for _, snapshot := range []fleet.WorkerSnapshot{
		{},
		{Heartbeat: fleet.Heartbeat{Queues: []string{}, Capabilities: []fleet.Capability{}}},
	} {
		worker := h.worker(snapshot)
		if worker.Queues != nil || worker.Capabilities != nil {
			t.Fatalf("empty projection collections changed shape: %+v", worker)
		}
		data, err := json.Marshal(worker)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		if string(fields["queues"]) != "null" || string(fields["capabilities"]) != "null" {
			t.Fatal("empty collection JSON must retain null representation")
		}
	}
}
