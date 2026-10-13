# queue-control-plane

[![CI](https://github.com/faustbrian/go-queue-control-plane/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-queue-control-plane/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-queue-control-plane/actions/workflows/ci.yml)
[![Documentation](https://img.shields.io/badge/docs-structural_checks_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-queue-control-plane/v4.svg)](https://pkg.go.dev/github.com/faustbrian/go-queue-control-plane/v4)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-queue-control-plane?sort=semver)](https://github.com/faustbrian/go-queue-control-plane/releases)
[![Go](https://img.shields.io/badge/go-1.27.0-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`queue-control-plane` is the administrative control plane for
[`queue`](https://pkg.go.dev/github.com/faustbrian/go-queue). It provides durable,
tenant-scoped commands, desired state, audit history, an HTTP API, an
administrative CLI, and an optional narrow Kubernetes Deployment adapter.

See [Releases](https://github.com/faustbrian/go-queue-control-plane/releases)
for the latest published version. A backend-neutral adapter maps
tenant-scoped commands and acknowledgements through published `queue`
management contracts. The optional tenant management document now enables the
authenticated status, command, and record transport; endpoints supply the
managed root `queue` as their native lifecycle controller and a native
`management.RecordReader` for failure workflows.
Kubernetes scale commands work only when the canonical
`adapters/kubernetes` integration is configured. The released `kubernetes`
path remains as a deprecated compatibility path.
Redis Streams and Valkey Streams workers can publish native worker and queue
status through the same `queue` management HTTP handler. Managed queues can
also consume durable desired state through the typed client. See
[Current capability status](docs/compatibility.md) before evaluating a rollout.

## Install

Main now selects the candidate `/v4` module for bounded rejection detail and
explicit known-state reporting. V4 is not published by this source change;
see the [v4 migration guide](COMPATIBILITY.md#queue-control-plane-v4-migration-on-main)
and confirm release availability before adoption. The commands below retain
existing published v3 choices.

Published v3.0.0 uses root v3 with Authentication v2. The commands below pin
that release and retain the supported historical v1 and v2 choices. See the
[v3 migration guide](COMPATIBILITY.md#queue-control-plane-v3-migration-on-main)
for the coherent nominal import migration and finite authentication admission.
Published v3.1.1 also includes PostgreSQL v2 pool integration. Select
that release or a later compatible v3 maintenance release when adopting
the pool integration; the v3.0.0 pins below do not include it.

Use the official major suffix for published v3.0.0:

```sh
go get github.com/faustbrian/go-queue-control-plane/v3@v3.0.0
go install github.com/faustbrian/go-queue-control-plane/v3/cmd/queue-control-plane@v3.0.0
go install github.com/faustbrian/go-queue-control-plane/v3/cmd/queue-control@v3.0.0
```

The published v1 line remains available. Add that module to an application with:

```sh
go get github.com/faustbrian/go-queue-control-plane@latest
```

Install the server and administrative CLI for local evaluation with:

```sh
go install github.com/faustbrian/go-queue-control-plane/cmd/queue-control-plane@latest
go install github.com/faustbrian/go-queue-control-plane/cmd/queue-control@latest
```

Pin the published `v1.1.2` release exactly with:

```sh
go get github.com/faustbrian/go-queue-control-plane@v1.1.2
go install github.com/faustbrian/go-queue-control-plane/cmd/queue-control-plane@v1.1.2
go install github.com/faustbrian/go-queue-control-plane/cmd/queue-control@v1.1.2
```

The supported security-related `v2.0.1` release is published. Use the official
major suffix:

```sh
go get github.com/faustbrian/go-queue-control-plane/v2@v2.0.1
go install github.com/faustbrian/go-queue-control-plane/v2/cmd/queue-control-plane@v2.0.1
go install github.com/faustbrian/go-queue-control-plane/v2/cmd/queue-control@v2.0.1
```

The public migration helpers now compose Migrations v2 types. See the
[v2 migration guide](COMPATIBILITY.md#queue-control-plane-v2-migration) for the
nominal API change and unchanged schema history. The Go floor remains 1.27.0.

The Go-installed server does not carry the release pipeline's commit and build
time metadata, so its `/version` response uses the development build identity.
The public `v2.0.1` assets are source-only, not deployable binaries or images;
see the [release process](docs/releasing.md) for their provenance scope.

## Five-minute local start

Prerequisites: Go 1.27.0 or newer and an empty PostgreSQL database reachable
through `DATABASE_URL`. The installed `queue-control-plane` and `queue-control`
binaries must be on `PATH`.

Create `/tmp/queue-control-access.json` outside version control:

```json
{
  "keys": [
    {"id": "local-cli", "key": "replace-this-secret", "subject": "operator-1"}
  ],
  "acl": [
    {
      "id": "view-audit",
      "subject": "operator-1",
      "tenant": "tenant-1",
      "action": "audit_view",
      "resource_type": "workload",
      "resource_id": "audit",
      "effect": "allow"
    }
  ]
}
```

Start the API and apply its embedded migration:

```sh
export DATABASE_URL='postgres://user:password@localhost/control_plane?sslmode=disable'
export QUEUE_CONTROL_ACCESS_FILE=/tmp/queue-control-access.json
export QUEUE_CONTROL_RUN_MIGRATIONS=true
queue-control-plane
```

In another shell, verify the public probes and authenticated CLI:

```sh
curl --fail http://localhost:8080/health/live
curl --fail http://localhost:8080/health/ready

export QUEUE_CONTROL_URL=http://localhost:8080
export QUEUE_CONTROL_KEY_ID=local-cli
export QUEUE_CONTROL_KEY=replace-this-secret
queue-control audit list --tenant tenant-1
```

Do not commit the local access document. For production, inject it from a
secret volume and run a one-shot `QUEUE_CONTROL_MIGRATE_ONLY=true` Job before
starting serving replicas.

## Documentation

- [Source contract audit and limitations](docs/source-documentation-audit.md)
- [Architecture and trust boundaries](docs/architecture.md)
- [HTTP API reference](docs/api.md)
- [CLI reference](docs/cli.md)
- [Compiler-checked desired-state client example](client/desired_state_example_test.go)
- [Embedded web UI guide](docs/ui.md)
- [Deployment and configuration](docs/deployment.md)
- [Compatibility and current capability status](docs/compatibility.md)
- [Hardening evidence and release gates](docs/hardening.md)
- [Security and privacy](docs/security.md)
- [Operations, retention, backup, and incidents](docs/operations.md)
- [Performance and load benchmarks](docs/performance.md)
- [Kubernetes, HPA, and KEDA](docs/kubernetes.md)
- [Laravel Horizon migration matrix](docs/horizon-migration.md)
- [Troubleshooting and FAQ](docs/faq.md)
- [Release process](docs/releasing.md)
- [Support](SUPPORT.md)
- [Security reporting](SECURITY.md)
- [Changelog](CHANGELOG.md)

For ecosystem-wide package selection and ownership guidance, see the versioned
[Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.8.4/docs/ecosystem/README.md)
and its [Persistence and durability family](https://github.com/faustbrian/go-library-tools/blob/v1.8.4/docs/ecosystem/design-language.md#package-families-and-selection).

## Development

For standalone untagged tests:

```sh
GOWORK=off go test ./... -count=1 -timeout=3m
```

`make check` and `make ci` are bounded public-verifier routes with the actual
PostgreSQL, Redis, Valkey and integration-tag prerequisites. Both retain the
cheap no-Go documentation boundary; hosted Required also needs that boundary
to pass. Explicit `make check-full` retains full integration, disaster recovery,
browser, benchmark and release-risk qualification. See
[CONTRIBUTING.md](CONTRIBUTING.md) for focused fuzzing, cancellation/shutdown
checks, and fixture requirements.

The same server image supports separate migrate-only and bounded
retention-only Jobs. See [Deployment and configuration](docs/deployment.md) for
their strict inputs and failure semantics. Retention verifies and advances the
audit anchor before removing unreferenced old terminal commands; active and
current desired-state commands remain durable.

The API, typed client, and CLI expose newest-first tenant command history with
bounded opaque-cursor pagination in addition to point lookup by idempotency
key.

Full qualification compares the exported Go module surface with the reviewed
API baseline.

## License

This project is licensed under the [MIT License](LICENSE).

Continue with the [documentation index](docs/README.md).
