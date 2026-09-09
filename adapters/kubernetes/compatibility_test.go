//lint:file-ignore SA1019 This test intentionally verifies the deprecated path.

package kubernetes_test

import (
	"errors"
	"testing"

	canonical "github.com/faustbrian/go-queue-control-plane/adapters/kubernetes"
	legacy "github.com/faustbrian/go-queue-control-plane/kubernetes"
)

func TestCanonicalAndLegacyPathsShareConstructionContract(t *testing.T) {
	t.Parallel()

	if _, err := canonical.New("", nil); !errors.Is(err, canonical.ErrInvalidConfiguration) {
		t.Fatalf("canonical New() error = %v, want ErrInvalidConfiguration", err)
	}
	if _, err := legacy.New("", nil); !errors.Is(err, canonical.ErrInvalidConfiguration) {
		t.Fatalf("legacy New() error = %v, want canonical ErrInvalidConfiguration", err)
	}
	if !errors.Is(legacy.ErrInvalidConfiguration, canonical.ErrInvalidConfiguration) {
		t.Fatal("legacy and canonical configuration sentinels differ")
	}

	if _, err := canonical.LoadTenantResolver(nil, 0, nil); !errors.Is(err, legacy.ErrInvalidTenantDocument) {
		t.Fatalf("LoadTenantResolver() error = %v, want legacy sentinel", err)
	}
	if _, err := canonical.NewScaleDispatcher(nil); !errors.Is(err, legacy.ErrInvalidDispatcher) {
		t.Fatalf("NewScaleDispatcher() error = %v, want legacy sentinel", err)
	}
	if _, err := canonical.NewStaticTenantResolver(nil); !errors.Is(err, legacy.ErrInvalidResolver) {
		t.Fatalf("NewStaticTenantResolver() error = %v, want legacy sentinel", err)
	}
}
