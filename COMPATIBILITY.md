# Compatibility Policy

Each releasable directory is an independent Go module and follows semantic
versioning. Tags use `<module-directory>/v<version>`.

Before `v1`, minor releases MAY contain reviewed breaking changes, but every
break MUST be documented with migration guidance. Patch releases MUST remain
backward compatible. At and after `v1`, incompatible exported API or documented
behavior changes require a new major version.

Compatibility includes exported Go APIs, error classification, serialization,
protocol behavior, persistence schemas, environment variables, command output,
resource ownership, ordering, retry/idempotency semantics, and documented
defaults. A compile-compatible change can still be behaviorally breaking.

Specification-backed modules MUST NOT diverge from their declared standards.
Ambiguities require documented decisions and stable tests. Deprecated APIs
follow [`DEPRECATION.md`](DEPRECATION.md).

## Queue control plane v2 migration

The current main branch prepares `v2.0.0`; it is not a published release yet.
Use the root module `github.com/faustbrian/go-queue-control-plane/v2` and add
`/v2` to all control-plane package imports. The source remains at the repository
root on main; there is no version-specific directory or branch.

`postgres.MigrationSource` now returns `go-migrations/v2.Source`, and
`postgres.NewMigrationRunner` returns `*go-migrations/v2.Runner`. Update direct
Migrations imports, result types, and interfaces used with these helpers to
`github.com/faustbrian/go-migrations/v2`. V1 and v2 nominal types and sentinel
error values are distinct; do not mix them in one helper composition.

The database pool remains caller-owned. The CLI still closes the pool it
opens. Migration acquisition or unlock uncertainty now discards its physical
session, and default migration errors redact private driver data. Explicit
30-second lock and five-minute statement timeouts remain. The v2 producer also
supplies finite operation and cleanup budgets.

Embedded SQL bytes, migration versions, checksums, PostgreSQL ledger formats,
HTTP protocol, and command names remain unchanged; adopting this module major
does not require resetting or replaying an existing schema history. Other
collaborator imports, including Authentication, Authorization, Identifier,
PostgreSQL, Queue, and Telemetry, retain their selected major versions.
