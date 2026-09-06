# Security policy

## Supported versions

| Release line | Status | End of support |
| --- | --- | --- |
| `v1.0.0` | Supported; security update pending | Not scheduled |

Security fixes are developed on the default branch and shipped in a new `v1`
release. Published `v1.0.0` does not include the gRPC denial-of-service fix
recorded under Unreleased in the changelog and must not be treated as containing
that fix. An advisory will identify affected versions and any change to the
supported release lines.

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
