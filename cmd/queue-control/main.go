// Package main implements the queue-control client command. Startup reads
// operator-owned endpoint/key settings and delegates one invocation to cli.Runner.
// It owns no server, data-plane loop or durable retry; the typed client's default
// HTTP timeout bounds requests, while output/credential lifetime remains process
// owned. Credential strings are not erased by command completion.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/faustbrian/go-queue-control-plane/v6/cli"
	"github.com/faustbrian/go-queue-control-plane/v6/client"
)

var processExit = os.Exit

func main() {
	processExit(run(context.Background(), os.Args[1:], os.Getenv, os.Stdout, os.Stderr, nil))
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdout io.Writer,
	stderr io.Writer,
	httpClient *http.Client,
) int {
	baseURL := strings.TrimSpace(getenv("QUEUE_CONTROL_URL"))
	keyID := strings.TrimSpace(getenv("QUEUE_CONTROL_KEY_ID"))
	key := strings.TrimSpace(getenv("QUEUE_CONTROL_KEY"))
	if baseURL == "" || keyID == "" || key == "" {
		_, _ = fmt.Fprintln(
			stderr,
			"QUEUE_CONTROL_URL, QUEUE_CONTROL_KEY_ID, and QUEUE_CONTROL_KEY are required",
		)
		return cli.ExitFailure
	}
	api, err := client.New(client.Config{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
		APIKeys:    environmentAPIKeySource{id: keyID, secret: key},
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return cli.ExitFailure
	}

	return (cli.Runner{Client: api, Stdout: stdout, Stderr: stderr}).Run(ctx, args)
}

type environmentAPIKeySource struct {
	id     string
	secret string
}

func (s environmentAPIKeySource) APIKey(context.Context) (string, string, error) {
	return s.id, s.secret, nil
}
