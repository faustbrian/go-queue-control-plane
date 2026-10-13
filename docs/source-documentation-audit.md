# Source documentation audit

## Owner, acceptance and selected proof

This Tier A pass owns comments and directly contradicted maintained prose at
fresh remote main `a4daeab3b172fa80621e0ed2d5d56ceece562886`. Preserve original
and concurrent work. The coordinator owns publication, enforcement and main CI.
Runtime code, SQL/schema, wire fixtures, dependencies, immutable API/history and
desired policies are not changed.

Direct consumers are typed administrative clients/CLI/UI, the control service,
tenant-resolved queue and Kubernetes adapters, PostgreSQL command/desired-state
and audit stores, worker-fleet projections, and the production command's explicit
application/server composition. Named risks are authentication versus permission,
acceptance/dispatch/acknowledgement certainty, idempotency conflict and partial
operations, desired-state revisions versus worker enforcement, cancellation and
shutdown, borrowed callbacks/resources, copies and bounded/redacted diagnostics.

Before comment writes, `git ls-files` and `golib source-docs inventory --json`
identify one module, 152 Go files (64 source, 88 tests) and 18 source directories,
including the browser testserver omitted from the public package catalog. The
initial 528 objective findings comprise 525 exported/member comment omissions
and three package comments. Missing comments are separate from semantic claims
requiring technical investigation. Five tagged integration/disaster-recovery
files, all assurance/fuzz/benchmark sources, SQL/UI embeds and the deprecated
Kubernetes compatibility path remain in scope. No generated Go was detected.

The native API mode is apidiff: retain `api/control-plane.api`, not a prose
snapshot rewrite. Once explicitly granted, use Go 1.27.2 with three owned caches
for all-file exact AST/signature/raw-token/literal/directive proof, formatting,
package GoDoc and the configured native docs operation. That operation runs the
existing desired-state client Example and the native Python docs-gate tests;
do not duplicate the Example or run unchanged service/race/fuzz/benchmark/
consumer/release campaigns. Local docs/source/offline specification checks,
pinned full/changed/staged secrets and ordinary complete final-diff inspection
finish the coherent local commit. Remove all owned temporary resources on every
path. Final semantic inventory, source-only questions and bounded outcomes follow.

## Investigation and findings

All 64 source files were traced across the following boundaries. All 88
assurance files were inventoried by named outcomes, with contract assertions
reviewed for cancellation before dispatch, dispatch/acknowledgement persistence,
scoped authorization/visibility, copying, bounded pagination, digest compatibility,
optional SQL/queue/REST fixtures and process cleanup. The three saved fuzz targets
and eight benchmark operations describe their actual synthetic input/workload
and exclusions; they are not executed by this prose pass. The existing Example
owns a synthetic HTTP server and one caller-scheduled reconciliation invocation.

| Boundary | Source contracts investigated |
| --- | --- |
| Root/control/authz | Bounded but untrimmed envelope identities, optional operation IDs versus tenant idempotency keys, actor/principal binding and deny-by-default decisions, replay source plus destination authorization, atomic admission through a lifecycle journal, pre-dispatch cancellation, acknowledgement of failed/partial business results, unknown persistence outcomes, desired revision planning and borrowed synchronous collaborators. |
| apihttp/server/client/cli | Principal context versus credential middleware, exact origin/CSRF/admission ordering, byte-bounded decoding with unknown/trailing input refusal rather than canonical JSON, provider-owned scoped/page-valid output, independently audited revealed fields, copied option/maps/bytes versus borrowed snapshot collections during synchronous encoding, redirect refusal and shallow transport copying, response closure and client decoding limits, CLI output/error sensitivity, listener shutdown versus application-resource ownership. |
| dataplane/fleet/alerts | Published management envelopes, resolver ownership and partial/unknown acknowledgements, worker HeartbeatAt rather than fetch-time rejuvenation, capped remote traversal, deterministic copied registry snapshots and conflicts/staleness, capability negotiation distinct from permission, identity-cardinality overflow projection, supported versus unavailable measurements, threshold precision and fixed telemetry labels without provider preemption. |
| history/postgres | Unkeyed versioned digest encoding, sequence checks versus hash-link checks and external root trust, tenant anchor locking, UTC microsecond duplicate comparison excluding regenerated CommandID, atomic desired-state/audit admission and lifecycle transitions, repeatable-read pages, bigint bounds after driver scans, contiguous retention prefixes and reference-safe command deletion, borrowed v1/v2 pools and compiler-owned migration input budgets. |
| kubernetes and adapters/kubernetes | Canonical alias/delegation preserves released nominal types, apps/v1 Deployments and autoscaling/v1 scale through SDK v0.37.1, immutable copied tenant maps with borrowed clients, provider namespace/page/context obligations, returned update acknowledgement distinct from observed replica convergence; no watches, worker supervision or HPA/KEDA. |
| Commands/ui/browser helper | Startup serving versus one-shot modes, trusted bounded operator documents and HTTPS management references, bounded regular TLS material reads and cooperative lookup limits, acquisition/cleanup ordering, separate retention batches/legal holds, non-global telemetry and pool shutdown, hand-maintained executable UI assets, isolated synthetic browser surfaces rather than production identity enforcement. |

The 525 exported/member omissions and three package omissions are resolved;
structural zero is not semantic certification. Every source directory has one
canonical package comment. The five build-constrained files are assurance only
(four integration, one disasterrecovery); there are no production platform/tag
variants or generated Go. Both embed directives retain their bytes. Nine SQL
migrations and all UI HTML/JavaScript/CSS, authenticated fixtures/dependencies,
`api/control-plane.api` and historical authorities are unchanged. There is no
shipped Go generator; embedding compiler-owned hand-maintained inputs is not a
regeneration recipe. No specification decision set is configured.

Directly contradicted current descriptions were repaired for ingress/dispatch
ordering, default versus supplied command IDs, relative audit-page verification,
provider/context borrowing, total-memory claims and derived heartbeat validation.
Existing tested redacted/redacted payload inspection copies provider-redacted
bytes: `apihttp.TestRecordModelRevealsOnlyExplicitCompatibleVisibility` explicitly
expects data for that case. Current API/security prose now states this boundary;
original-byte reveal still requires separate permission/audit. Desired privacy
and release requirements, test expectations and dated release pins are not relaxed.

## Source-only questions and limits

- Registry retains at most its declared workers, but its rejected-tenant counter
  map has no cardinality cap; capacity refusal occurs before complete heartbeat
  validation. Arbitrary-tenant total-memory safety is not established.
- Custom HTTP/Kubernetes providers must honor page sizes, scope and safe public
  metadata. Kubernetes List does not cap already materialized response counts or
  continuation metadata. HTTP projections do not universally revalidate sources.
- Audit ListTenant checks from the first entry's stored predecessor; VerifyTenant
  traverses from its database retention anchor. Neither authenticates a root
  against an actor able to replace database state, and page size is not a bound
  on complete-history traversal.
- Pre-dispatch cancellation completion uses WithoutCancel: values remain, but
  the request deadline/cancellation no longer bounds the journal call. Arbitrary
  callbacks/providers can block or panic; no universal isolation is claimed.
- Retention loading multiplies signed seconds into a nanosecond Duration before
  validating the converted range. No pre-conversion overflow check was found;
  integer arithmetic shows `2^55 + 3600` seconds wraps to a one-hour duration.
  This is source/arithmetic evidence, not a newly executed loader regression.
  The intended finite retention policy remains unchanged and needs its owner
  to resolve this separately; no runtime repair is bundled here.
- Source/API errors can retain driver/provider diagnostics; safe HTTP problem
  codes do not sanitize every metadata/failure field or deliberate cause access.
  Actual content redaction remains a provider obligation, not a visibility label.

No alternative rationale is invented for these boundaries. Existing stronger
security/resource/release requirements remain requirements, not evidence that
unchanged source satisfies every deployment or fault scenario.

## Local proof and delivery boundary

The owned Go 1.27.2 session passed formatting and exact AST/signatures, raw
tokens (including implicit semicolons), literals and directives across all 152
Go files. All 971 final comment groups parsed/rendered as HTML, the 17 catalog
package and command GoDocs were inspected, and the ignored browser helper was
rendered through the standard-library AST/doc packages without executing it. The final
rate-limiter prose refinement passed a short all-file preservation/comment proof
and affected apihttp GoDoc refresh; the other package renders and unchanged
native Example/docs-gate results were retained from the original session.

The configured `verification/package.mk docs` operation passed its one existing
output-bearing client Example and native Python docs-gate tests. No second
Example invocation was needed. Native API mode is apidiff, so its frozen baseline
was preserved rather than rewritten as prose. SQL, executable UI assets,
fixtures, dependencies, configuration, API and historical inputs remained exact.
Local source/docs/offline-specification checks passed; specification reported
zero applicable modules/decisions, not a conformance result. Pinned baseline,
current and changed-content scans found no secrets, with ignore annotations
disabled and fully redacted output. Final full/changed and staged snapshots are
scanned before the commit; ordinary complete final-diff truth/usefulness
inspection covers every owned source/prose change. All session-owned caches,
helpers and renders were removed immediately; no parent-owned tool or
shared/user resource was removed.
The coordinator owns exact-main CI and publication. No live PostgreSQL, queue,
broker, Chromium, active fuzz, race, benchmark, consumer or release campaign is
claimed by this source pass.

## Objective Enforcement

The checksum-pinned public Tools v2.4.0 verifier now supplies
`make source-docs-check` as a prerequisite of the existing local and CI Make
routes. Ordinary shared CI selects strict source-documentation checking and
requires its result. Native prerequisites, operation budgets, security and
release controls are retained. Hosted exact-head evidence is recorded by the
coordination ledger; configuration alone does not establish green delivery.
