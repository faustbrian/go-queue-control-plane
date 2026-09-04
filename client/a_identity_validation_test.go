package client

import (
	"errors"
	"strings"
	"testing"

	controlplane "github.com/faustbrian/go-queue-control-plane"
)

func TestClientValidatesTenantIdentityBeforeCreatingReader(t *testing.T) {
	client := &Client{}

	for _, test := range []struct {
		name   string
		tenant string
	}{
		{name: "empty"},
		{name: "whitespace", tenant: " \t\n"},
		{name: "too long", tenant: strings.Repeat("t", controlplane.MaxIdentityBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := client.DesiredStateReader(test.tenant); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("DesiredStateReader() error = %v, want ErrInvalidRequest", err)
			}
		})
	}

	reader, err := client.DesiredStateReader(strings.Repeat("t", controlplane.MaxIdentityBytes))
	if err != nil {
		t.Fatalf("DesiredStateReader(maximum identity) error = %v", err)
	}
	if reader == nil {
		t.Fatal("DesiredStateReader(maximum identity) reader = nil")
	}
}
