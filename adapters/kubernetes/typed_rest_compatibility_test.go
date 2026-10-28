package kubernetes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// This server exercises generated client serialization and response decoding,
// rather than substituting the adapter's DeploymentClient interface.
func TestTypedDeploymentRESTCompatibility(t *testing.T) {
	for _, format := range []string{"json", "protobuf"} {
		t.Run(format, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				parts := strings.Split(r.URL.Path, "/")
				if len(parts) < 7 || strings.Join(parts[:5], "/") != "/apis/apps/v1/namespaces" || parts[6] != "deployments" {
					t.Errorf("unexpected Kubernetes route: %s", r.URL.Path)
					http.Error(w, "unexpected route", http.StatusBadRequest)
					return
				}
				namespace := parts[5]
				if namespace != "queues-a" && namespace != "queues-b" {
					t.Errorf("unexpected namespace: %s", namespace)
					http.Error(w, "unknown namespace", http.StatusForbidden)
					return
				}
				if format == "protobuf" && !strings.Contains(r.Header.Get("Accept"), "application/vnd.kubernetes.protobuf") {
					t.Errorf("default client did not negotiate protobuf: %s", r.Header.Get("Accept"))
				}
				switch {
				case len(parts) == 7 && r.Method == http.MethodGet:
					if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("continue") != "before/+?=" {
						t.Errorf("pagination query: %s", r.URL.RawQuery)
					}
					typedRESTReply(t, w, format, fmt.Sprintf(`{"apiVersion":"apps/v1","kind":"DeploymentList","metadata":{"continue":"after/+?=","remainingItemCount":2},"items":[{"metadata":{"name":"workers","namespace":%q,"generation":9},"spec":{"replicas":4},"status":{"observedGeneration":8,"updatedReplicas":3,"readyReplicas":2,"availableReplicas":1,"unavailableReplicas":1}}]}`, namespace))
				case len(parts) == 8 && parts[7] == "workers" && r.Method == http.MethodGet:
					typedRESTReply(t, w, format, fmt.Sprintf(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"workers","namespace":%q,"generation":9},"spec":{"replicas":4},"status":{"observedGeneration":8,"updatedReplicas":3,"readyReplicas":2,"availableReplicas":1,"unavailableReplicas":1}}`, namespace))
				case len(parts) == 9 && parts[7] == "workers" && parts[8] == "scale":
					if r.Method == http.MethodGet {
						typedRESTReply(t, w, format, fmt.Sprintf(`{"apiVersion":"autoscaling/v1","kind":"Scale","metadata":{"name":"workers","namespace":%q,"resourceVersion":"41"},"spec":{"replicas":3},"status":{"replicas":2}}`, namespace))
						return
					}
					if r.Method != http.MethodPut {
						t.Errorf("scale method: %s", r.Method)
						http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
						return
					}
					body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
					if err != nil {
						t.Error(err)
						http.Error(w, "read failed", http.StatusInternalServerError)
						return
					}
					if format == "protobuf" && (!strings.HasPrefix(r.Header.Get("Content-Type"), "application/vnd.kubernetes.protobuf") || !bytes.HasPrefix(body, []byte("k8s\x00"))) {
						t.Errorf("scale update is not the production protobuf request")
					}
					object, _, err := scheme.Codecs.UniversalDeserializer().Decode(body, nil, nil)
					if err != nil {
						t.Error(err)
						http.Error(w, "decode failed", http.StatusBadRequest)
						return
					}
					scale, ok := object.(*autoscalingv1.Scale)
					if !ok || scale.Name != "workers" || scale.Namespace != namespace || scale.ResourceVersion != "41" || scale.Spec.Replicas != 7 || scale.Status.Replicas != 2 {
						t.Errorf("scale update lost concurrency or replica state: %#v", object)
					}
					typedRESTReply(t, w, format, fmt.Sprintf(`{"apiVersion":"autoscaling/v1","kind":"Scale","metadata":{"name":"workers","namespace":%q,"resourceVersion":"42"},"spec":{"replicas":6},"status":{"replicas":5}}`, namespace))
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
				}
			}))
			defer server.Close()
			config := &rest.Config{Host: server.URL, Timeout: 3 * time.Second}
			if format == "json" {
				config.ContentType = "application/json"
				config.AcceptContentTypes = "application/json"
			}
			client, err := kubernetesclient.NewForConfig(config)
			if err != nil {
				t.Fatal(err)
			}
			adapters := map[string]TenantAdapter{}
			for _, namespace := range []string{"queues-a", "queues-b"} {
				adapter, err := New(namespace, client.AppsV1().Deployments(namespace))
				if err != nil {
					t.Fatal(err)
				}
				adapters[namespace] = adapter
			}
			resolver, err := NewStaticTenantResolver(adapters)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			for _, namespace := range []string{"queues-a", "queues-b"} {
				page, err := resolver.ListTenantWorkloads(ctx, namespace, 2, "before/+?=")
				expected := Status{Namespace: namespace, Name: "workers", Generation: 9, ObservedGeneration: 8, DesiredReplicas: 4, UpdatedReplicas: 3, ReadyReplicas: 2, AvailableReplicas: 1, UnavailableReplicas: 1}
				if err != nil || len(page.Items) != 1 || page.Items[0] != expected || page.Continue != "after/+?=" || page.Remaining != 2 {
					t.Fatalf("typed list = %#v, %v", page, err)
				}
				adapter := adapters[namespace].(*Adapter)
				got, err := adapter.Get(ctx, "workers")
				if err != nil || got != expected {
					t.Fatalf("typed get = %#v, %v", got, err)
				}
				scaled, err := adapter.Scale(ctx, "workers", 7)
				if err != nil || scaled != (ScaleResult{Namespace: namespace, Name: "workers", DesiredReplicas: 6, CurrentReplicas: 5, ResourceVersion: "42"}) {
					t.Fatalf("typed scale acknowledgement = %#v, %v", scaled, err)
				}
			}
			before := calls.Load()
			if _, err := resolver.ListTenantWorkloads(ctx, "unknown", 2, ""); !errors.Is(err, ErrTenantNotConfigured) || calls.Load() != before {
				t.Fatalf("unknown tenant reached Kubernetes: %v", err)
			}
			if before != 8 {
				t.Fatalf("expected one list/get/scale GET/PUT per namespace, got %d", before)
			}
		})
	}
}

func typedRESTReply(t *testing.T, w http.ResponseWriter, format, fixture string) {
	t.Helper()
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, fixture); err != nil {
			t.Error(err)
		}
		return
	}
	// Hand-authored API objects keep the oracle independent of adapter output;
	// the official protobuf codec only implements the wire encoding.
	object, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(fixture), nil, nil)
	if err != nil {
		t.Error(err)
		http.Error(w, "invalid fixture", http.StatusInternalServerError)
		return
	}
	var payload bytes.Buffer
	if err := protobuf.NewSerializer(scheme.Scheme, scheme.Scheme).Encode(object, &payload); err != nil {
		t.Error(err)
		http.Error(w, "encode fixture", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.kubernetes.protobuf")
	if _, err := w.Write(payload.Bytes()); err != nil {
		t.Error(err)
	}
}

func TestTypedDeploymentRESTStatusErrors(t *testing.T) {
	for _, test := range []struct {
		code    int
		reason  string
		matches func(error) bool
	}{
		{403, "Forbidden", apierrors.IsForbidden}, {404, "NotFound", apierrors.IsNotFound}, {409, "Conflict", apierrors.IsConflict},
	} {
		t.Run(test.reason, func(t *testing.T) {
			var writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					writes.Add(1)
				}
				if test.code == 409 && r.Method == http.MethodGet {
					typedRESTReply(t, w, "json", `{"apiVersion":"autoscaling/v1","kind":"Scale","metadata":{"name":"workers","namespace":"queues","resourceVersion":"41"},"spec":{"replicas":3}}`)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.code)
				_, _ = fmt.Fprintf(w, `{"apiVersion":"v1","kind":"Status","status":"Failure","reason":%q,"code":%d,"message":"fixture refusal"}`, test.reason, test.code)
			}))
			defer server.Close()
			client, err := kubernetesclient.NewForConfig(&rest.Config{Host: server.URL, Timeout: 3 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			adapter, err := New("queues", client.AppsV1().Deployments("queues"))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			if test.code == 409 {
				_, err = adapter.Scale(ctx, "workers", 7)
			} else {
				_, err = adapter.Get(ctx, "workers")
			}
			if !test.matches(err) {
				t.Fatalf("typed API status was not preserved: %v", err)
			}
			if test.code == 409 && writes.Load() != 1 {
				t.Fatalf("conflict must not replay scale update: %d", writes.Load())
			}
		})
	}
}

func TestTypedDeploymentRESTCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	client, err := kubernetesclient.NewForConfig(&rest.Config{Host: server.URL, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New("queues", client.AppsV1().Deployments("queues"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := adapter.Get(ctx, "workers"); result <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("transport cancellation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("request did not cancel")
	}
	limiter := flowcontrol.NewTokenBucketRateLimiter(0.001, 1)
	if !limiter.TryAccept() {
		t.Fatal("could not consume initial rate-limit token")
	}
	client, err = kubernetesclient.NewForConfig(&rest.Config{Host: server.URL, Timeout: 3 * time.Second, RateLimiter: limiter})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err = New("queues", client.AppsV1().Deployments("queues"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Get(ctx, "workers"); !errors.Is(err, context.Canceled) {
		t.Fatalf("rate-limiter cancellation: %v", err)
	}
}
