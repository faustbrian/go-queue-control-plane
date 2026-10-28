package apihttp

import (
	"encoding/json"
	"github.com/faustbrian/go-queue-control-plane/v6/fleet"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestR30WorkerPagePreservesRegisteredRejectionKnowledge(t *testing.T) {
	now := time.Unix(1, 0)
	registry := fleet.NewRegistry(1)
	if registry.Upsert(workerHeartbeat("registered", "worker", now, []string{"critical"})) != fleet.HeartbeatAccepted {
		t.Fatal("registration control refused")
	}
	if registry.Upsert(workerHeartbeat("unknown", "worker", now, []string{"critical"})) != fleet.HeartbeatCapacityExceeded {
		t.Fatal("unknown capacity control admitted")
	}
	handler, err := NewHandler(Config{Commands: &commandExecutorStub{}, Workers: registry, Viewer: &viewerStub{}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	for _, tenant := range []string{"registered", "unknown"} {
		request := authenticatedRequest(t, http.MethodGet, "/v1/tenants/"+tenant+"/workers", "")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("public worker request refused: %d", response.Code)
		}
		var page WorkerPage
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.Rejected != 0 || page.RejectedKnown != (tenant == "registered") {
			t.Fatalf("public known-state projection lost: %+v", page)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
			t.Fatal(err)
		}
		if _, present := fields["rejected_known"]; !present {
			t.Fatal("unknown was silently encoded without explicit known-state")
		}
	}
}
