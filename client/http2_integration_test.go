package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	controlplane "github.com/faustbrian/go-queue-control-plane/v3"
)

func TestTypedClientHTTP2RoundTripAndCancellation(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 || r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-bearer" {
			t.Errorf("request protocol/method/auth = %s %s %t", r.Proto, r.Method, r.Header.Get("Authorization") == "Bearer test-bearer")
		}
		switch r.URL.Path {
		case "/v1/tenants/tenant-1/commands/request-1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(controlplane.CommandResult{TenantID: "tenant-1", IdempotencyKey: "request-1", Status: controlplane.CommandSucceeded})
		case "/v1/tenants/tenant-1/commands/blocked":
			close(started)
			select {
			case <-r.Context().Done():
				close(canceled)
			case <-time.After(2 * time.Second):
				t.Error("HTTP/2 server did not observe cancellation")
			}
		default:
			t.Errorf("unexpected typed command path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	httpClient := server.Client()
	httpClient.Timeout = 3 * time.Second
	defer httpClient.CloseIdleConnections()
	api, err := New(Config{BaseURL: server.URL, HTTPClient: httpClient, Tokens: &tokenSourceStub{token: "test-bearer"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	result, err := api.GetCommand(ctx, "tenant-1", "request-1")
	if err != nil || result.TenantID != "tenant-1" || result.IdempotencyKey != "request-1" || result.Status != controlplane.CommandSucceeded {
		t.Fatalf("typed HTTP/2 command = %+v, %v", result, err)
	}
	requestCtx, cancelRequest := context.WithCancel(ctx)
	defer cancelRequest()
	done := make(chan error, 1)
	go func() { _, requestErr := api.GetCommand(requestCtx, "tenant-1", "blocked"); done <- requestErr }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("HTTP/2 request did not reach server")
	}
	cancelRequest()
	select {
	case requestErr := <-done:
		if !errors.Is(requestErr, context.Canceled) {
			t.Fatalf("HTTP/2 cancellation = %v", requestErr)
		}
	case <-ctx.Done():
		t.Fatal("HTTP/2 client did not finish cancellation")
	}
	select {
	case <-canceled:
	case <-ctx.Done():
		t.Fatal("HTTP/2 stream did not cancel server request")
	}
}
