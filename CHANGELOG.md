# Changelog

All notable changes to this project will be documented in this file. The
format follows Keep a Changelog, and releases use Semantic
Versioning.

## Unreleased

## 3.1.1 - 2026-10-10

### Changed

- Update the coupled Kubernetes client, API and machinery dependencies to
  0.37.1 while preserving namespace isolation, bounded Deployment pages,
  scale acknowledgements and cancellation/error classification.
- Adopt Telemetry v2.0.1 and patched HTTP/2 dependencies; build with
  Go 1.27.2 to address standard-library vulnerabilities while retaining
  the public Go 1.27.0 language minimum.

### Fixed

- Keep documentation fence and stale-credential checks effective on CI
  runners without optional search tooling.

## 3.1.0 - 2026-10-08

### Added

- Add `postgres.NewRuntimeWithPool` for caller-owned PostgreSQL v2 pools,
  sharing the native persistence runtime and bounded readiness checks. The
  existing `NewRuntime` entrypoint continues accepting PostgreSQL v1 pools
  ([55e1e5cd96]).

### Changed

- Avoid redundant worker snapshot collection copies during synchronous HTTP
  response encoding, retaining deterministic ordering and empty-field shapes
  ([9a11dd3a26]).
- Adopt published PostgreSQL v2.0.0 and pgx v5.11.0 for server and retention
  pool acquisition, with application-owned DSN resolution and explicit bounded
  startup connectivity checks. Migration history and persistence formats are
  unchanged; PostgreSQL v1 remains available for legacy public collaborators.
- Keep retained audit sequences and command deletion counts checked against
  their native and batch bounds, with preparation owning post-resolution
  cancellation ([85fd63886c], [751bab22a7]).

### Documentation

- Distinguish the published v3 baseline from additive pool adoption and its
  upgrade guidance ([97e0cfd799]).

### Maintenance

- Cover TLS material admission and inclusive migration-source budgets while
  retaining lint-compatible redaction assertions ([43e536e813], [6d83664679],
  [3f210f5d3a]).

[55e1e5cd96]: https://github.com/faustbrian/go-queue-control-plane/commit/55e1e5cd9687e8f39e59e9e23c48f2d9671d16a0
[85fd63886c]: https://github.com/faustbrian/go-queue-control-plane/commit/85fd63886c4cfe0bd2f7e3383f3d4b3839d5abda
[751bab22a7]: https://github.com/faustbrian/go-queue-control-plane/commit/751bab22a732a17c5269b67583e778be4ca90834
[43e536e813]: https://github.com/faustbrian/go-queue-control-plane/commit/43e536e813d4bd5ce797a4cb2dbf0b5fb7070fbc
[6d83664679]: https://github.com/faustbrian/go-queue-control-plane/commit/6d836646790b4115b87489571e735b7359384498
[3f210f5d3a]: https://github.com/faustbrian/go-queue-control-plane/commit/3f210f5d3ae460870600af642f342eab3b50551e
[97e0cfd799]: https://github.com/faustbrian/go-queue-control-plane/commit/97e0cfd799cc3ae1600a5e21f1fc73e85dd15214
[9a11dd3a26]: https://github.com/faustbrian/go-queue-control-plane/commit/9a11dd3a268e90041dc174856ca8ac01b0bc6a56

## 3.0.0 - 2026-10-07

### Changed

- Move the root module and all package imports to
  `github.com/faustbrian/go-queue-control-plane/v3` to compose Authentication v2
  principal, credential, challenge, API-key and HTTP adapter types. Update
  both module majors together; oversized authentication identities and keys
  now follow the producer's finite admission policy ([15a18d1280]).
- Adopt Queue v1.1.3 while retaining its management protocol and nominal types.
  Authorization's core ACL and PostgreSQL retain their existing major versions.

### Documentation

- Align release guidance with the configured PostgreSQL 18 and shared CI
  operations; identify the root v3 rehearsal and its fresh mutation evidence
  without implying publication, additional platform coverage, or deployment.
- Point installation and security guidance to published v2.0.1, including its
  Queue management redirect protection and source-only artifact scope
  ([570074f61c]).

### Maintenance

- Refresh the pinned shared CI workflow while retaining repository tooling
  and required module gates ([d631afaf2a]).
- Patch the development-tool parser to smol-toml v1.9.0 and clarify the
  coherent v3 and Authentication v2 adoption boundary ([52b6e15c40]).

[15a18d1280]: https://github.com/faustbrian/go-queue-control-plane/commit/15a18d12802e15eae37155c607311c816c0961d2
[570074f61c]: https://github.com/faustbrian/go-queue-control-plane/commit/570074f61c6b8a655ea9d1d859c42f47d2cfde52
[d631afaf2a]: https://github.com/faustbrian/go-queue-control-plane/commit/d631afaf2a3d50121dee194743ee49146d7a4c8d
[52b6e15c40]: https://github.com/faustbrian/go-queue-control-plane/commit/52b6e15c40f57fc8d9b191853f0c2eec1b94c06f

## 2.0.1 - 2026-10-01

### Security

- Adopt Queue v1.1.2 for the owned management HTTP client, preserving tenant
  routing and Queue type identities while rejecting redirects without changing
  caller-owned HTTP clients.

### Documentation

- Clarify published v2 security-model applicability, accepted deployment and
  collaborator risk ownership, and the configured hosted release checks.

## 2.0.0 - 2026-09-30

### Changed

- Move the root module and all package imports to
  `github.com/faustbrian/go-queue-control-plane/v2`. The public PostgreSQL
  `MigrationSource` and `NewMigrationRunner` helpers now return Migrations v2
  types; callers using their results must also import `go-migrations/v2`.
  Other collaborator major versions, HTTP contracts, embedded SQL, migration
  identities, checksums, and ledger formats remain unchanged.

### Security

- Adopt Migrations v2 so uncertain advisory-lock acquisition or release
  discards the physical session instead of returning it to a caller-owned
  pool, and default migration error formatting does not expose private driver
  diagnostics. Keep the 30-second lock and five-minute statement budgets.
- Enforce producer inventory, filename, aggregate-name, and file-byte budgets
  before retaining metadata or copying compiler-owned embedded migrations;
  canceled or over-budget loads return no partial input.

## 1.1.2 - 2026-09-28

### Changed

- Adopt Telemetry v2 for the owned OTLP runtime and HTTP instrumentation while
  preserving non-global trace and metric export. TLS material reads are bounded,
  custom CA files replace system roots, and trusted inbound propagation now
  requires an authenticated request.

## 1.1.1 - 2026-09-13

### Changed

- Require Go 1.27.0 for module consumers, local builds, and the container image
  ([383c7561d3](https://github.com/faustbrian/go-queue-control-plane/commit/383c7561d35baf59b360264c5b1bbcdcff68e6e2)).
- Document the maintained public API baseline and typed compatibility
  operations while retaining the repository-owned API compatibility script
  ([076c00db94](https://github.com/faustbrian/go-queue-control-plane/commit/076c00db940641c6dc7b1be8bcca4b8b122a604b)).
- Update gRPC to 1.83.2, miniredis to 2.39.0, and OpenTelemetry SDK to 1.46.0
  ([8bcd2669b2](https://github.com/faustbrian/go-queue-control-plane/commit/8bcd2669b26fe0ebda7286e516d2766a2741c328),
  [6f86aa24d7](https://github.com/faustbrian/go-queue-control-plane/commit/6f86aa24d7cc28b4f67843029dd1938889f53b63),
  [f17040c10d](https://github.com/faustbrian/go-queue-control-plane/commit/f17040c10d33327cbdc5e80d373022c587bcb923)).

### Documentation

- Adopt and scope proportional assurance policy for repository checks
  ([db14e98948](https://github.com/faustbrian/go-queue-control-plane/commit/db14e98948be0593cbfa7c23e4c373905f59dca3),
  [c432bc31da](https://github.com/faustbrian/go-queue-control-plane/commit/c432bc31dae82e690530936c123d41268a1d00e7)).

## 1.1.0 - 2026-09-09

### Added

- Add `adapters/kubernetes` as the target-oriented entry point for the narrow,
  namespace-scoped Kubernetes Deployment status and scale integration.

### Changed

- Adopt the public Authentication v1.2, Authorization v1.1, Migrations v1.1,
  PostgreSQL v1.1, Queue v1.1, and Telemetry v1.2 contracts used by the
  control-plane composition while retaining its existing domain behavior.
- Use Authentication's canonical HTTP adapter, Queue's canonical Redis
  Streams adapter, and PostgreSQL's explicit `Connect` and bounded `Shutdown`
  lifecycle. Authentication's released `authhttp` aliases remain source
  compatible even though API documentation now renders the canonical path.
- Adopt `go-library-tools` v1.6.2 so declaration-only compatibility packages
  with no viable mutants complete the repository gate without synthetic logic.

- Replace the copied repository verification implementation with the
  checksum-verified `go-library-tools` v1.0.7 contract while retaining the
  package-owned manifests, fixtures, browser checks, and mutation evidence.

- Upgrade `go-telemetry` to v1.1.1 and select gRPC v1.83.1 for the
  control-plane telemetry runtime.
- Adopt the `go-library-tools` v1.3.0 schema-v2 cohesion contract and local
  `make cohesion` gate without changing public APIs or runtime behavior.
- Pin reusable CI to the immutable v1.3.0 workflow and enforce cohesion
  metadata in the repository's required CI contract.

- Upgrade the checksum-verified repository tooling to the `go-library-tools`
  v1.3.0 contract while retaining the
  package-owned manifests, fixtures, browser checks, and mutation evidence.

- Advance repository tooling to the `go-library-tools` v1.4.0 schema-v2
  cohesion contract, local `make cohesion` gate, and online specification
  validation in `make ci`. This tooling-only adoption does not change public
  APIs or runtime behavior.
- Pin reusable CI to the immutable v1.4.0 workflow, prefer public module
  releases before bootstrap-only fallbacks, and enforce cohesion
  metadata in the repository's required CI contract.

- Upgrade the checksum-verified repository tooling to the `go-library-tools`
  v1.4.0 contract while retaining the
  package-owned manifests, fixtures, browser checks, and mutation evidence.

### Documentation

- Complete the public package, executable-example, and support navigation;
  bind documentation checks and module metadata to the compiler-checked client
  example; and publish direct issue and discussion routes.

- Describe the supported stable `v1` release, add version-pinned installation
  commands, make the local quick start independent of a source checkout, and
  align release guidance with the published signed checksums, archive, SBOM,
  and provenance assets while documenting the Go-installed server's build
  identity boundary, the supported `v1.1` security line, and active delivery
  work.

- Publish family, capability, ownership, lifecycle, support, and package
  selection metadata, with links to the immutable ecosystem index and family
  guidance.
- Remove the archived monorepo documentation link; package guidance remains in
  the repository-owned documentation.

### Deprecated

- Retain `kubernetes` as a compatibility path for `adapters/kubernetes` so
  existing v1 consumers keep the same public types, errors, construction, and
  runtime behavior while new code uses the target-oriented path.

### Security

- Resolve the gRPC HTTP/2 DATA-frame fragmentation denial-of-service exposure
  in the reachable OTLP/gRPC telemetry exporter path.

## 1.0.0 - 2026-08-25

### Fixed

- Bind the reviewed zero-mutant embedded UI handler to its exact standalone
  source identity.

### Changed

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Correct stale package, standalone, and authoritative-source links in public
  documentation.

### Fixed

- Enforce exact fleet capacity, runtime accumulation, rejection aggregation,
  deterministic ordering, and telemetry saturation boundaries under mutation
  verification.
- Validate structured data-plane results through an explicit terminal-outcome
  whitelist, preserving fail-closed cancellation and lifecycle-state handling
  without redundant status conditions.
- Strengthen command-orchestration verification around desired-state target
  alternatives, caller-supplied metadata, and every nonterminal dispatch
  outcome.

### Documentation

- Replace obsolete standalone-repository links and workflow claims with
  standalone-repository targets and current release guidance.

- Replace the invalid GitHub package URL with a portable repository-relative
  link to `queue`.

### Compatibility

- Added a pinned module export baseline so incompatible public API changes
  fail the canonical repository gate.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-queue-control-plane` identity while preserving its documented API and behavior.
- Require each CLI connection credential independently, with mutation-precise
  regression coverage for every missing environment value.
- Replace obsolete owned-module pseudo-version pins with the monorepo's local
  `v0.0.0` source-proxy coordinates; release tooling continues to emit exact
  `v1.0.0` dependency versions.
- New durable command identifiers are lowercase ULIDs. The PostgreSQL upgrade
  preserves historical UUID command identifiers while changing command and
  reference columns to text so existing audit history remains addressable.
- Require owned sibling modules at local `v0.0.0`; clean external consumers
  pin each module to an exact main pseudo-version.

- Upgraded the resolved gRPC dependency to 1.82.1 and refreshed owned-module
  checksums after the shared telemetry dependency update.
- Align indirect dependency metadata with the resolved graph so clean
  consumers and CI obtain reproducible module metadata.
- Refresh owned-module checksums against the final consolidated archives.
- Made the documented idempotency example unambiguously synthetic so strict
  secret scanning does not mistake it for a credential.
- Normalized standalone module metadata against the canonical owned dependency
  graph, including complete checksums for clean consumer resolution.
- Refreshed the canonical authentication checksum after its test archive
  changed, preserving isolated module verification.
- Refreshed owned authentication, authorization, and PostgreSQL checksums after
  their API compatibility baselines were normalized to module boundaries.
- Pinned the completed `queue` dead-letter v1 contracts, stable operational
  failure codes, retention negotiation, and bounded Redis/Valkey redrive
  lineage used by rolling worker fleets.
- Strengthened the deterministic local and hosted quality gate with module
  checksum verification, Staticcheck, strict golangci-lint, advisory NilAway,
  and exact per-package statement coverage.
- Added a global CLI `--output human` mode while retaining compact JSON as the
  stable default and keeping terminal control characters escaped.

### Added

- Fixed-label bounded alert telemetry export for up to 12,000 validated alerts
  without tenant, resource, or error-derived metric labels.
- Attachment disposition on privileged payload and diagnostics responses.
- Real Redis and Valkey concurrent retry/delete evidence proving exactly one
  truthful winner for a contested failed record.
- PostgreSQL child-process death evidence during authorization and live
  dispatcher execution, preserving zero or durably auditable state.
- Allocation-budgeted maximum-payload, 10,000-worker reconnect-storm, and
  backend-outage dispatch benchmarks in the local and hosted smoke gate.
- Distinct durable `unsupported`, `timed_out`, and `partial` command outcomes
  so `queue` acknowledgements cannot be flattened into a clean failure.
- Restart-safe `pending`, `dispatched`, and `acknowledged` command boundaries,
  transition timestamps, pre-dispatch cancellation, and fault-injected
  PostgreSQL recovery evidence.
- Durable bounded command deadlines, authentication methods, required
  capabilities, and worker/protocol acknowledgement snapshots.
- Durable opaque command identifiers distinct from caller-owned idempotency keys,
  including migration backfill and end-to-end result propagation.
- Separate record-list, record-inspection, payload-view, and audit-view
  permissions with exact tenant and object scope.
- Truthful CLI retention status derived from negotiated worker capabilities,
  with unknown numeric limits represented explicitly.
- Independently authorized diagnostics visibility with fail-closed masking
  across the HTTP API, typed client, and CLI.
- Fail-closed chained audit persistence before every privileged payload or
  diagnostics backend read.
- Public administrative command, result, permission, and validation contracts.
- Durable PostgreSQL command, desired-state, and tamper-evident audit storage.
- Bounded worker fleet and rolling-protocol compatibility models.
- Authenticated and authorized HTTP API with health, readiness, version,
  capability, command, audit, worker, and Kubernetes workload surfaces.
- Typed Go client and administrative CLI.
- Narrow namespace-scoped Kubernetes Deployment visibility and scaling.
- Hardened server and CLI container targets for amd64 and arm64.
- Deterministic Go quality, race, coverage, fuzz-smoke, and container CI gates.
- Real PostgreSQL 16, 17, and 18 migration, persistence, idempotency, tenant
  isolation, audit verification, and retention integration gates.
- Reproducible 10,000-worker and 100,000-audit-event load benchmarks with a
  documented development baseline and CI smoke execution.
- Maximum 1,000-worker authenticated HTTP response benchmark with every worker
  advertising the bounded 256-queue limit.
- Maximum-page queue and failure/dead-letter conversion benchmarks in the
  hosted load smoke gate.
- Pinned `govulncheck` source scanning against the canonical Go vulnerability
  database in local and hosted quality workflows.
- BuildKit SBOM and maximal provenance attestations on the multi-platform OCI
  artifact produced by CI.
- Semver-tagged GHCR publication and keyless Sigstore signing for both server
  and CLI images, with immediate workflow-identity verification.
- Targeted goroutine-leak assertions for graceful and forced HTTP server
  shutdown paths.
- Pinned mutation testing with 100% coverage and efficacy across all current
  public command, authorization, desired-state, dispatch, and orchestration
  mutants.
- Migration-only server mode for one-shot production schema Jobs that exit
  before loading serving dependencies.
- Isolated PostgreSQL 18 disaster-recovery gate using native dump and restore
  with repository-level verification of the recovered state.
- Pinned semantic API-diff gate against a reviewed full-module export baseline
  in both CI and release quality.
- Reproducible standalone archives for six operating-system and architecture
  targets with signed checksums and GitHub Release publication.
- Optional secure OTLP runtime wiring for HTTP and bounded control-command
  telemetry with explicit lifecycle ownership and trusted-context policy.
- One-shot production audit retention with strict per-tenant policies, legal
  holds, pre/post verification, bounded batches, and real PostgreSQL coverage.
- Real Chromium security coverage for exact-origin CORS, automatic preflight,
  cookie-backed CSRF, credential admission, and defensive response headers.
- Real Chromium command-envelope coverage for every public mutation and each
  action-specific selection, replay, and scale field.
- Mutation scope excludes installed browser dependencies and test-server code,
  keeping release efficacy measurements limited to owned production decisions.
- Server and CLI container builds now include the imported data-plane adapter
  and embedded UI packages in their restricted build context.
- Newest-first tenant command history with bounded opaque-cursor pagination
  across PostgreSQL, the authenticated API, typed client, and CLI.
- Safe bounded terminal-command retention after verified audit cleanup, while
  preserving active commands and current desired-state references.
- Tenant-scoped `queue` management adaptation for every control action,
  bounded bulk retry, protocol deadlines, and structured terminal outcomes.
- Authenticated failure and dead-letter list and inspect APIs, typed client,
  and CLI with bounded search/sort pagination and privileged payload viewing.
- Authenticated bounded queue-status API, typed client, CLI, and application
  capability wiring with explicit unsupported-measurement representation.
- Optional production worker and queue-status composition through a strict
  tenant-to-HTTPS management document and separate bounded per-tenant
  bearer-token files.
- Optional production tenant command dispatch through the same authenticated
  `queue` management endpoints, with protocol-v1 translation, a bounded
  acknowledgement deadline, and structured terminal outcomes.
- Optional production failure and dead-letter reads through the same tenant
  management client, with validated hidden lists and least-privilege payload
  inspection.
- Pinned native Valkey failed-attempt and dead-letter reads through the
  authenticated management transport, including hidden lists and explicitly
  authorized payload inspection.
- Pinned native Valkey retry, bounded bulk retry, delete, and record-purge
  enforcement with atomic active-failure redrive and structured ambiguity
  outcomes.
- Pinned native Valkey allowlisted replay with durable reject-or-replace
  duplicate handling, plus source and destination authorization before command
  acceptance.
- Pinned native Redis Streams failure and dead-letter reads plus retry,
  bounded bulk retry, allowlisted replay, delete, and record-purge dispatch
  through authenticated `queue` contracts without native client access.
- Authenticated `queue` HTTP integration coverage for managed pause, resume,
  status, and duplicate command enforcement, plus older, current, newer,
  partitioned, and reconnecting protocol observations.
- Pinned Redis 6.2.22 and Valkey 9.1.0 CI services proving real backend status,
  pause, and resume through `queue` and authenticated management HTTP.
- The real Valkey gate creates a failed delivery, dispatches replay, rejects a
  duplicate, preserves the source, and consumes the destination through a
  second public `queue` worker.
- Published `queue` Redis Streams and Valkey Streams status providers, with
  honest backend-version capability reporting, are pinned for the production
  worker and queue-status transport.
- Pinned `queue` retention capability negotiation, including exact-count
  reporting through Redis Streams and rolling-version HTTP integration.
- Authenticated tenant desired-state reads, a typed tenant-bound `queue`
  reader, and native queue-owned pause, resume, drain, and terminate
  convergence through the pinned data-plane lifecycle.
- Bounded alert-input evaluation for queue wait, failures, stale workers, dead
  letters, and failed or unknown control commands.
- Optional embedded same-origin operator console for current worker, queue,
  failure, dead-letter, audit, command-history, and audited mutation workflows,
  with ephemeral credentials and Chromium end-to-end coverage.
- Architecture, API, CLI, deployment, security, operations, compatibility,
  Horizon migration, troubleshooting, contribution, and release documentation.
- Protocol/failure, crash-boundary, administrative threat, scale, Horizon
  ownership, and release-gate hardening matrices with executable evidence.

### Fixed

- Return an authentication-scheme challenge with rejected administrative
  credentials so invalid API keys remain a `401` response instead of being
  misclassified as authentication infrastructure unavailability.

### Security

- Blocked HTTP redirects in the typed administrative client so bearer tokens
  and custom API-key headers cannot be forwarded to another origin.
- Added exhaustive deployed permission scope, ambiguous request-framing,
  hostile-value XSS, and keyboard accessibility regression coverage.
- Added failure injection for duplicate sensitive mutations, control endpoint
  loss during ordinary delivery, telemetry shutdown, PostgreSQL saturation,
  and 10,000-worker stale/reconnect storms.
- Added real PostgreSQL process-death injection across command, desired-state,
  admission-audit, dispatch, acknowledgement, result, and completion-audit
  writes to prove atomic rollback and honest recovery state.
- Added exact-origin CORS, strict preflight, CSRF checks, defensive response
  headers, request bounds, process-local rate limiting, and secret-safe errors.
- Aligned the shipped CLI with the server's static API-key authentication and
  allowed those headers through approved browser preflights.
- Moved rate-limit admission behind authentication so valid administrative
  requests are isolated by stable subject instead of shared source address.
- Canonicalized persisted timestamps to UTC so idempotent responses and audit
  hashes do not depend on database session or caller timezone offsets.

### Known gaps

- Native queue purge remains unavailable because `queue` intentionally
  exposes only failure/dead-letter record purge for these stream backends.
- Production alert collection/export, desired-state cleanup, historical UI
  charts, retention reconfiguration, and deployment-specific ingress and
  PostgreSQL capacity tests remain intentionally external or incomplete.
