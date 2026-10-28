package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/faustbrian/go-queue-control-plane/v6/apihttp"
)

func TestR30CLIWorkerViewsPreserveRejectionKnowledge(t *testing.T) {
	for _, known := range []bool{false, true} {
		for _, args := range [][]string{{"workers", "list", "--tenant", "tenant"}, {"retention", "status", "--tenant", "tenant"}} {
			var output bytes.Buffer
			runner := Runner{Client: &apiStub{workers: apihttp.WorkerPage{RejectedKnown: known}}, Stdout: &output, Stderr: &bytes.Buffer{}}
			if got := runner.Run(context.Background(), args); got != ExitOK {
				t.Fatalf("valid worker view failed: %d", got)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			raw, present := fields["rejected_known"]
			if !present {
				t.Fatalf("CLI %s lost explicit rejection knowledge", args[0])
			}
			var got bool
			if err := json.Unmarshal(raw, &got); err != nil || got != known {
				t.Fatalf("CLI %s changed known-state: %t %v", args[0], got, err)
			}
		}
	}
}
