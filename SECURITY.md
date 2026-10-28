# Security policy

## Supported versions

| Release line | Status | End of support |
| --- | --- | --- |
| `v6.x` | Supported; first release `v6.0.0` | Not scheduled |
| `v1.x` to `v5.x` | Published, no further releases | 2026-10-28 |

Only the latest patch of the latest major receives security fixes.

Security fixes are developed on the default branch and shipped in a new
release of the latest major, using a new major when public contracts require
it. An advisory will identify affected versions.

The following history describes published releases that now receive no
further releases. Release `v1.1.0` includes the gRPC denial-of-service fix
recorded in the changelog; published `v1.0.0` does not. Published `v2.0.0`
includes the Migrations v2 integration described in
[compatibility guidance](COMPATIBILITY.md#queue-control-plane-v2-migration),
and `v2.0.1` additionally adopts Queue v1.1.2, rejecting management HTTP
redirects without changing caller-owned clients.
The published v1 helper still composes Migrations v1: with a caller-retained
database pool, an uncertain advisory-lock acquisition or release can return a
physical session that still owns the lock to that pool, and default migration
errors can expose driver diagnostics. There is no patched v1 release for this
boundary. Until adopting published v2, isolate migration work in a dedicated
pool that is closed on completion or uncertainty, and do not expose raw
migration errors to logs or untrusted callers. The CLI owns and closes its pool;
that lifecycle differs from the public helper's caller-owned pool.

## Current source and new major releases

Current main is the `/v6` line. Source on main alone does not establish
publication or security qualification; confirm each version in
[Releases](https://github.com/faustbrian/go-queue-control-plane/releases).

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
