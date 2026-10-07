# queue-control-plane

[![CI](https://github.com/faustbrian/go-queue-control-plane/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-queue-control-plane/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-queue-control-plane/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-queue-control-plane/v2.svg)](https://pkg.go.dev/github.com/faustbrian/go-queue-control-plane/v2)
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

Published v3.0.0 uses root v3 with Authentication v2. The commands below pin
that release and retain the supported historical v1 and v2 choices. See the
[v3 migration guide](COMPATIBILITY.md#queue-control-plane-v3-migration-on-main)
for the coherent nominal import migration and finite authentication admission.
Current main also contains the upcoming PostgreSQL v2 pool integration;
source on main alone is not publication of that later minor release. Confirm
availability in [Releases](https://github.com/faustbrian/go-queue-control-plane/releases)
before adopting that integration.

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
[Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)
and its [Persistence and durability family](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/design-language.md#package-families-and-selection).

## Development

Run the complete local repository gate with:

```sh
make ci
```

`make ci` validates the repository configuration, inventory, cohesion metadata,
repository contract, online specification conformance, workflow contract, and
the complete implementation check. Use `make check` for the implementation
checks alone or `make cohesion` for a focused validation of versioned ecosystem
metadata.

The implementation check covers formatting, module tidiness and checksums, vet,
Staticcheck, strict
golangci-lint, tests, the race detector, exact per-package 100% statement
coverage, and builds. `make nilaway` runs the pinned advisory NilAway profile,
and `make fuzz` runs the bounded fuzz smoke suite. `make integration-postgres`
starts a disposable PostgreSQL 18 container and runs the real persistence
contract under the race detector. `make benchmarks` runs eight single-core
large-fleet, API, payload, audit, reconnect, and backend-outage samples with
enforced allocation budgets. See
[CONTRIBUTING.md](CONTRIBUTING.md) for repository expectations.

`make security` runs the pinned Go vulnerability scanner against the canonical
Go vulnerability database and fails on reachable findings.

`make mutation` requires 100% mutant coverage and efficacy across the public
command contract, authorization mapping, desired state, dispatch, and command
orchestration.

`make disaster-recovery-postgres` creates isolated PostgreSQL 18 source and
restore databases, takes a native logical backup, restores it, and verifies the
complete control and audit state through the production repositories.

The same server image supports separate migrate-only and bounded
retention-only Jobs. See [Deployment and configuration](docs/deployment.md) for
their strict inputs and failure semantics. Retention verifies and advances the
audit anchor before removing unreferenced old terminal commands; active and
current desired-state commands remain durable.

The API, typed client, and CLI expose newest-first tenant command history with
bounded opaque-cursor pagination in addition to point lookup by idempotency
key.

`make api-compatibility` compares the complete exported Go module surface with
the reviewed baseline and fails on compatible or incompatible drift.

## License

This project is licensed under the [MIT License](LICENSE).
