# Security, authorization, and privacy

## Version and source applicability

This model describes the published root `v2.0.1` contract and explicitly
identified v3 changes in current source.
The immutable release source is
[`6da9d0a2`](https://github.com/faustbrian/go-queue-control-plane/tree/v2.0.1).
This patch adopts Queue v1.1.2's management HTTP redirect protection without
changing caller-owned clients or the Migrations v2 nominal integration.
Common deployment responsibilities also apply to supported v1.1 deployments,
but the Migrations v2 session and default-error protections below do not apply
to published v1 helpers. See the [supported-version policy](../SECURITY.md)
and [migration guidance](../COMPATIBILITY.md#queue-control-plane-v2-migration).
This model does not certify a deployment, caller-supplied collaborator, or
unpublished change merely because it is on main.

Current v3 source composes Authentication v2 nominal types and Queue v1.1.3.
Authentication v2.0.0 is published. Confirm QCP v3 availability in
[Releases](https://github.com/faustbrian/go-queue-control-plane/releases);
main source alone does not establish release qualification. The
[v3 migration guide](../COMPATIBILITY.md#queue-control-plane-v3-migration-on-main)
describes adoption without implying release qualification or deployment.

## Threat model

The control plane accepts high-impact administrative commands and stores their
history. Protect it as a privileged internal service. Primary risks are stolen
credentials, cross-tenant authorization, command replay with changed content,
destructive operations without confirmation, audit tampering, browser-origin
confusion, denial of service through unbounded input, Kubernetes privilege
expansion, and disclosure through errors or payloads.

TLS termination and network policy are deployment responsibilities. Do not
expose the plain HTTP listener directly to an untrusted network.

OTLP export uses TLS by default and supports custom CA and mTLS file mounts.
Plaintext is explicit and should be limited to a protected same-pod or
cluster-local path. Inbound trace context follows a bounded untrusted policy;
trusted extraction additionally requires an authenticated request and an
ingress that removes attacker-supplied propagation headers.
Never put tenant, actor, target, reason, payload, key, or idempotency values in
telemetry attributes.

Custom TLS material paths are a trusted-operator deployment boundary. Their
owner is the control-plane operator: provision local, read-only regular files
and exclude network, FUSE, and UNC mounts. The reader caps retained material at
1 MiB and closes an opened file on cancellation, but a blocked pathname lookup
cannot be preempted before a file descriptor exists. This residual is accepted
because the portable Go file API cannot cancel that lookup without an unowned
worker; review it if non-local TLS mounts become a supported deployment need or
the file-access implementation changes.

## Migration session and source boundary

On current main, the public PostgreSQL migration helpers use Migrations v2.
Uncertain advisory-lock acquisition or release discards the physical database
session instead of returning it to a retained caller pool. Healthy sessions
remain reusable. Default migration error formatting redacts private driver
data; applications must not deliberately extract and expose private causes.
The public helper leaves the pool caller-owned, whereas the CLI closes the
pool it opens. Lock attempts retain a 30-second budget and statements a
five-minute budget; the producer also applies finite operation and cleanup
deadlines. Context cancellation is not evidence that SQL was rolled back:
inspect the migration ledger and database before retrying an uncertain result.

| Residual | Owner | Rationale | Mitigation | Review trigger |
| --- | --- | --- | --- | --- |
| Compiler-owned embedded SQL | Repository maintainers | The provider serves immutable build-time files, not an arbitrary caller filesystem. | Enforce count, individual and aggregate name, and file-byte budgets before retaining metadata or copying bytes. | Embedded-history changes or support for external files. |
| Driver cancellation and physical close | Application operators and driver maintainers | Portable contexts cannot forcibly preempt an uncooperative driver. | Use a context-aware PostgreSQL driver, bounded database/network operations, and a dedicated migration pool when isolation is required. | Driver changes or delayed cancellation/close. |
| Explicit private diagnostic extraction | Application error-handling owners | Classification is available without default disclosure; deliberate cause extraction leaves that default boundary. | Log redacted classifications, never raw driver causes or SQL. | New error adapters or diagnostic logging. |

These are conditional trusted-collaborator residuals, not guarantees of forced
callback or driver preemption. Published v1 helpers retain the old producer
session behavior; use a dedicated migration pool closed after work or
uncertainty, redact errors, and migrate to published `v2.0.1`. Existing embedded
SQL and ledger formats remain unchanged; no history reset is required.

## Authentication and key handling

Current v3 source uses Authentication v2's finite principal and static
API-key admission policy. Oversized startup identities and keys fail rather
than being truncated; review access documents before migration. Its principal,
credential, challenge and canonical HTTP adapter types must all use the v2
module. Published QCP v2 retains its existing Authentication v1 composition.

The production server uses the static API-key implementation from
`authentication`. A request supplies a public key ID and secret in separate
headers. Authentication is optional only at middleware level so public probes
can work; every administrative route rejects the anonymous principal.
Rejected static credentials return `401` with the
`WWW-Authenticate: QueueControlKey` challenge; provider unavailability remains
a distinct `503` response.

- Generate high-entropy secrets and store the access document in a secret
  manager-backed read-only volume.
- Never put secrets in CLI arguments, container image layers, Git, logs, or
  command reasons.
- Rotate by deploying an overlapping new key, moving callers, then removing
  the old key. The access file is immutable at runtime, so restart replicas.
- Keep subjects stable across rotation so audit attribution remains coherent.

The file stores static secrets in plaintext. Hash-at-rest key storage and
dynamic identity-provider integration are not yet implemented.

Worker management status uses a separate per-tenant bearer token loaded from
a bounded read-only file. The tenant document contains only HTTPS endpoints
and token-file paths; inline tokens, URL credentials, plaintext endpoints, and
endpoint paths or queries are rejected. Keep this internal transport behind
namespace-scoped network policy, verify worker certificates, and rotate tokens
independently from operator API keys.

Worker desired-state reads use the administrative API-key boundary, not the
management bearer token. Issue a dedicated worker subject and allow `view`
only for its exact tenant and queue, worker, or worker-group targets. Keep the
key in a read-only secret volume and never reuse an operator mutation key.

The typed administrative client never follows HTTP redirects. This applies to
both bearer tokens and API keys, including caller-supplied HTTP clients. Treat
a redirect as an endpoint or ingress misconfiguration; correct the configured
base URL instead of allowing credentials to cross to the redirect target.

## Authorization

Authorization uses `authorization` with deny-overrides ACL evaluation. Each
decision includes subject, action, tenant, resource type, and resource ID. A
caller allowed to pause one queue is not implicitly allowed to pause another,
view audit history, or scale a Deployment.

Use least privilege. Separate read-only, queue-operation, failure-operation,
and workload-scaling subjects. Add explicit deny entries for emergency
guardrails where an allow pattern could otherwise be too broad. Test the full
authorization matrix before rollout.

## Mutation safeguards

Every mutation requires an authenticated actor, an authorization decision, a
bounded reason, a caller-selected idempotency key, a target, and a request
timestamp. Purge, bulk retry, and replay require confirmation; scaling to zero
also requires confirmation. Replay requires an explicit destination and
duplicate policy. It is authorized twice before persistence or dispatch: once
for the exact failure or dead-letter source and once for the destination as a
queue resource. Granting replay on a source does not grant replay into an
otherwise unauthorized queue.

An idempotency key is scoped to a tenant. Reusing it with changed command
content fails with HTTP 409. Dispatch and persistence errors are redacted.
Never treat a timeout or `outcome_unknown` as permission to issue a new key;
inspect the stored result and the target system first.

## Browser protections

Allowed origins are exact matches. Requests with unapproved origins and
preflights with unapproved methods or headers fail before reaching the API.
Responses deny framing, MIME sniffing, referrers, caching, and all content by
default Content Security Policy.

Cookie-bearing unsafe methods require a matching CSRF cookie and header unless
bearer authentication is in use. The shipped server uses header API keys and
does not issue cookies. The optional embedded UI consumes only the public API,
keeps its API key in page memory, clears the password input after connection,
and never uses local or session storage. Its narrower policy permits only
same-origin scripts, styles, and API connections while retaining frame,
referrer, MIME, and cache protections.

The browser suite starts isolated approved, hostile, UI, and API surfaces and
uses real Chromium to prove CORS, automatic preflight, CSRF, defensive headers,
ephemeral credentials, hidden payload requests, literal rendering of hostile
values, keyboard-reachable labeled controls, status rendering, and audited
command envelopes. The HTTP server suite separately sends conflicting request
framing and proves rejection before handler execution. Run the browser gate
with `npm run test:browser` after installing dependencies and Chromium:

```sh
cd _browser
npm ci
npx playwright install chromium
npm run test:browser
```

## Bounds and availability

The server bounds request bodies, access and tenant documents, identities,
reasons, page sizes, continuation tokens, bulk selections, scale values, rate
limit keys, HTTP timeouts, fleet size, and audit batches. General reads use a
subject-wide admission counter. Each mutation action and the privileged
`payload_view` and `diagnostics_view` reads use independent subject counters,
so one workflow cannot consume another workflow's allowance. The limiter is
process-local, so aggregate capacity increases with replica count. Place a
shared ingress limiter in front of multi-replica deployments when needed.

The shipped server creates no login session and issues no cookie. If an
identity-aware ingress adds a session, the ingress owns secure cookie flags,
fixation prevention, rotation, expiry, and logout. Unsafe cookie-bearing API
requests still require the matching CSRF cookie and header.

## Payload and data privacy

Failure and dead-letter lists expose metadata with payloads hidden. Inspection
also defaults to hidden; redacted inspection can expose bounded content type
and size but no bytes. Revealed inspection requires both `record_inspect` and
the separate `payload_view` permission on the exact record before the adapter is
called. The stable `queue` contract caps revealed bytes at one mebibyte,
and the control plane rejects malformed, mismatched, or over-disclosed adapter
output.

Any response containing privileged payload or diagnostics bytes uses a
constant attachment filename. The bytes remain base64-encoded inside bounded
JSON with `no-store` and `nosniff`; a backend-supplied content type is inert
metadata and never becomes the HTTP response type.

Privileged diagnostics remain hidden unless the caller explicitly requests
them and has `diagnostics_view` on the exact record. `payload_view` and
`diagnostics_view` are independent decisions, and an over-disclosing adapter
response is masked to the requested visibility before serialization.

After authorization and before calling the record adapter, the API appends a
tenant-chained audit event for each privileged field. PostgreSQL audit failure
returns `audit_unavailable` and prevents the backend read. These access events
record a generated command ID, actor, exact record target, permission, time,
and `authorized` result without recording payload or diagnostics bytes.
Audit hash version 2 seals the command ID; version 1 remains readable so the
migration does not invalidate pre-upgrade chains.

Payloads are never persisted in control-plane PostgreSQL or attached to
telemetry. They pass through the authenticated response only. Configure
adapter-side redaction for sensitive fields and deny `payload_view` and
`diagnostics_view` by default. Command targets, subjects,
reasons, timestamps, and outcomes are also sensitive metadata; never put
credentials or payload fragments in identifiers or reasons.

## Audit integrity

Audit hashes detect modification, deletion, insertion, and reordering within a
tenant's retained chain when verification starts from the trusted anchor. They
do not prevent an attacker with database write access from replacing both data
and an unprotected backup. Restrict database writes, export verification
results, protect backups separately, and investigate any chain failure as a
security incident.

## Accepted deployment and collaborator residuals

The following are conditional ownership boundaries, not acceptance of an
authentication, authorization, redaction, or resource-limit bypass. Deployments
must supply the stated mitigations; a missing mitigation is not certified safe.
The migration-specific residuals above remain separately scoped to their
published producer versions.

| Residual | Owner | Rationale | Mitigation | Review trigger |
| --- | --- | --- | --- | --- |
| Plaintext static secrets at rest | Deployment secret-management owner | The shipped immutable access document contains plaintext secrets; hash-at-rest and dynamic identity integration are not implemented. | Mount a secret-manager-backed read-only volume, restrict filesystem access, exclude images/logs/arguments, and rotate overlapping keys with replica restarts. | Storage or identity-provider changes, or evidence of secret exposure. |
| Inbound TLS and protected plaintext OTLP paths | Network and ingress operators | The listener relies on deployment TLS; explicit insecure OTLP is limited to a protected local path. | Terminate TLS before untrusted traffic, restrict network access, verify worker certificates, and prefer default TLS for export. | New exposure paths, ingress/Collector changes, or insecure export outside the protected path. |
| Operator-owned TLS file lookup | Deployment filesystem owner | Portable file APIs cannot cancel a blocked pathname lookup before opening a descriptor. | Use local read-only regular files within the 1 MiB cap; exclude network, FUSE, and UNC mounts. | Non-local mount requirements or file-reader changes. |
| Ingress-added sessions and trusted trace extraction | Ingress authentication owner | The server issues no cookies or login sessions; trusted propagation depends on authenticated ingress. | Own secure cookie flags, fixation prevention, rotation, expiry, and logout; strip untrusted propagation headers and retain API CSRF checks for unsafe cookie requests. | Session-provider, cookie, or propagation-policy changes. |
| Process-local rate limiting | Deployment capacity owner | Each replica owns its counters, so aggregate admission grows with replica count. | Size replica budgets and use a shared ingress limiter when aggregate limits are required. | Replica-count, quota, tenant-isolation, or ingress-limiter changes. |
| Trusted telemetry and application collaborators | Application integration and provider/driver owners | Standard telemetry providers and synchronous authorization, persistence, and dispatch collaborators are trusted code; contexts cannot forcibly preempt arbitrary implementations. | Use bounded context-aware collaborators, fixed telemetry labels, bounded exporter flush/shutdown, and controlled endpoints; do not substitute blocking custom instrumentation. | Provider/driver/adapter changes, new callbacks, or delayed operations/shutdown. |
| Audit anchors and backup trust | Database security and recovery operators | Hashes detect chain changes only from a trusted anchor; database write access can replace data and an unprotected backup together. | Restrict writes, export verification heads, protect anchors/backups separately, verify restored chains, and investigate failures. | Retention/anchor changes, backup or restore procedures, or a chain failure. |

See the [hardening evidence and threat matrix](hardening.md) for exact test and
release-gate ownership.
