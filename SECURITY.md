# Security policy

## Supported versions

| Release line | Status | End of support |
| --- | --- | --- |
| `v2.0.x` | Supported; latest patch `v2.0.1` | Not scheduled |
| `v1.1.x` | Supported | Not scheduled |
| `v1.0.x` | Unsupported; security update required | 2026-09-09 |

Security fixes are developed on the default branch and shipped in a new
release, using a new major when public contracts require it. Release `v1.1.0`
includes the gRPC denial-of-service fix recorded in
the changelog. Published `v1.0.0` does not include that fix and consumers must
upgrade to the supported `v1.1` line. An advisory will identify affected
versions and any later change to the supported release lines.

Published `v2.0.0` includes the Migrations v2 integration described in
[compatibility guidance](COMPATIBILITY.md#queue-control-plane-v2-migration).
Published `v2.0.1` additionally adopts Queue v1.1.2, rejecting management
HTTP redirects without changing caller-owned clients. Prefer this patch
when adopting the supported v2 line.
The published v1 helper still composes Migrations v1: with a caller-retained
database pool, an uncertain advisory-lock acquisition or release can return a
physical session that still owns the lock to that pool, and default migration
errors can expose driver diagnostics. There is no patched v1 release for this
boundary. Until adopting published v2, isolate migration work in a dedicated
pool that is closed on completion or uncertainty, and do not expose raw
migration errors to logs or untrusted callers. The CLI owns and closes its pool;
that lifecycle differs from the public helper's caller-owned pool.

## Prospective main

Current main prepares root v3 with published Authentication v2.0.0. QCP v3 is
not published yet, and v3 support begins only at publication. The supported
published lines above remain unchanged. See the
[prospective migration guide](COMPATIBILITY.md#queue-control-plane-v3-migration-on-main)
for the nominal import changes and finite startup admission policy.

## Reporting a vulnerability

Do not disclose a suspected vulnerability in a public issue. Use GitHub's
private vulnerability reporting for this repository. If that facility is not
available, contact the maintainer privately through the contact method on the
maintainer's GitHub profile.

Include the affected revision, impact, reproduction conditions, and any known
workaround. Do not include real credentials, queue payloads, tenant data, or
production endpoints. The maintainer will acknowledge the report, assess the
affected versions, coordinate a fix and disclosure, and credit the reporter
when requested.

## Security boundary

The control plane is an administrative system, not a queue backend. Reports
about bypassing authentication, tenant authorization, idempotency, audit-chain
integrity, payload privacy, request bounds, CORS/CSRF admission, Kubernetes
namespace scope, or the no-raw-backend boundary are in scope.

Operational exposure caused solely by deploying without TLS, leaking the
static access file, or granting broader Kubernetes or PostgreSQL permissions
than documented is normally a deployment issue, but reports that reveal an
unsafe default or unclear contract are welcome.
