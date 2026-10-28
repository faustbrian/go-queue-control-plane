package kubernetes

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLoadTenantResolverBuildsNamespaceScopedAdapters(t *testing.T) {
	t.Parallel()

	document := `{"tenants":[{"id":"tenant-1","namespace":"workers-one"},{"id":"tenant-2","namespace":"workers-two"}]}`
	namespaces := make([]string, 0, 2)
	resolver, err := LoadTenantResolver(strings.NewReader(document), 1024, func(namespace string) DeploymentClient {
		namespaces = append(namespaces, namespace)

		return &deploymentClient{}
	})
	if err != nil || resolver == nil {
		t.Fatalf("LoadTenantResolver() = (%v, %v), want resolver and nil", resolver, err)
	}
	if strings.Join(namespaces, ",") != "workers-one,workers-two" {
		t.Fatalf("factory namespaces = %v, want document order", namespaces)
	}
	for _, tenant := range []string{"tenant-1", "tenant-2"} {
		if _, err := resolver.ResolveTenant(t.Context(), tenant); err != nil {
			t.Fatalf("ResolveTenant(%q) error = %v", tenant, err)
		}
	}
}

func TestLoadTenantResolverRejectsUnsafeDocuments(t *testing.T) {
	t.Parallel()

	validFactory := func(string) DeploymentClient { return &deploymentClient{} }
	var typedNil *strings.Reader
	tests := []struct {
		reader  io.Reader
		limit   int64
		factory DeploymentFactory
	}{
		{},
		{reader: typedNil, limit: 100, factory: validFactory},
		{reader: strings.NewReader(`{}`), limit: 100, factory: validFactory},
		{reader: strings.NewReader(`{"tenants":[]}`), limit: 100, factory: validFactory},
		{reader: strings.NewReader(`{"unknown":[]}`), limit: 100, factory: validFactory},
		{reader: strings.NewReader(`{"tenants":[]}{}`), limit: 100, factory: validFactory},
		{reader: strings.NewReader(`{"tenants":[{"id":"tenant-1","namespace":"workers"}]}`), limit: 8, factory: validFactory},
		{reader: failingTenantReader{}, limit: 100, factory: validFactory},
		{reader: strings.NewReader(`{"tenants":[{"id":"tenant-1","namespace":"workers"},{"id":"tenant-1","namespace":"other"}]}`), limit: 1000, factory: validFactory},
		{reader: strings.NewReader(`{"tenants":[{"id":"tenant-1","namespace":"INVALID"}]}`), limit: 1000, factory: validFactory},
		{reader: strings.NewReader(`{"tenants":[{"id":"tenant-1","namespace":"workers"}]}`), limit: 1000},
		{
			reader: strings.NewReader(`{"tenants":[{"id":"tenant-1","namespace":"workers"}]}`),
			limit:  1000,
			factory: func(string) DeploymentClient {
				var client *deploymentClient

				return client
			},
		},
	}
	for _, test := range tests {
		resolver, err := LoadTenantResolver(test.reader, test.limit, test.factory)
		if resolver != nil || !errors.Is(err, ErrInvalidTenantDocument) {
			t.Fatalf("LoadTenantResolver() = (%v, %v), want nil and stable error", resolver, err)
		}
	}
}

func TestLoadTenantResolverEnforcesByteBoundaries(t *testing.T) {
	t.Parallel()

	document := `{"tenants":[{"id":"tenant-1","namespace":"workers"}]}`
	factory := func(string) DeploymentClient { return &deploymentClient{} }
	if _, err := LoadTenantResolver(strings.NewReader(document), int64(len(document)), factory); err != nil {
		t.Fatalf("LoadTenantResolver() at exact byte limit error = %v", err)
	}
	if _, err := LoadTenantResolver(strings.NewReader(document+" "), int64(len(document)), factory); !errors.Is(err, ErrInvalidTenantDocument) {
		t.Fatalf("LoadTenantResolver() over byte limit error = %v, want ErrInvalidTenantDocument", err)
	}
	if _, err := LoadTenantResolver(strings.NewReader(document), 0, factory); !errors.Is(err, ErrInvalidTenantDocument) {
		t.Fatalf("LoadTenantResolver() with zero byte limit error = %v, want ErrInvalidTenantDocument", err)
	}
}

func TestLoadTenantResolverDoesNotRejectPositiveByteLimitEarly(t *testing.T) {
	t.Parallel()

	reader := &countingReader{reader: strings.NewReader(`{}`)}
	if _, err := LoadTenantResolver(reader, 1, func(string) DeploymentClient { return &deploymentClient{} }); !errors.Is(err, ErrInvalidTenantDocument) {
		t.Fatalf("LoadTenantResolver() error = %v, want ErrInvalidTenantDocument", err)
	}
	if reader.reads == 0 {
		t.Fatal("LoadTenantResolver() did not read a document with a positive byte limit")
	}
}

func TestLoadTenantResolverRejectsReaderErrorWithValidDocument(t *testing.T) {
	t.Parallel()

	document := `{"tenants":[{"id":"tenant-1","namespace":"workers"}]}`
	reader := &readerWithError{content: []byte(document)}
	if _, err := LoadTenantResolver(reader, int64(len(document)), func(string) DeploymentClient { return &deploymentClient{} }); !errors.Is(err, ErrInvalidTenantDocument) {
		t.Fatalf("LoadTenantResolver() error = %v, want ErrInvalidTenantDocument", err)
	}
}

type failingTenantReader struct{}

func (failingTenantReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

type readerWithError struct {
	content []byte
	read    bool
}

type countingReader struct {
	reader io.Reader
	reads  int
}

func (reader *countingReader) Read(target []byte) (int, error) {
	reader.reads++

	return reader.reader.Read(target)
}

func (reader *readerWithError) Read(target []byte) (int, error) {
	if reader.read {
		return 0, io.EOF
	}
	reader.read = true
	copy(target, reader.content)

	return len(reader.content), errors.New("read failed after content")
}
