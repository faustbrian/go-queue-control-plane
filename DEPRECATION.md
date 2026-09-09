# Deprecation Policy

Deprecations MUST identify the replacement, reason, migration steps, and
earliest removal version. Public Go identifiers use a valid `Deprecated:` doc
paragraph and corresponding changelog entry.

At `v1` and later, a supported replacement SHOULD exist for at least one minor
release before removal. Security or correctness defects MAY require faster
removal when continued support would be unsafe; the release notes must explain
the exception.

Silent behavior changes, undocumented aliases, and indefinite deprecated code
are prohibited. Deprecations are checked during compatibility and release
review.

## Kubernetes adapter import path

The released `github.com/faustbrian/go-queue-control-plane/kubernetes` package
is deprecated in favor of
`github.com/faustbrian/go-queue-control-plane/adapters/kubernetes`. The new
path makes the external target explicit and aligns package selection with the
rest of Golib without changing the adapter's narrow Deployment contract.

Migration requires only changing the import path; exported types, constants,
error sentinels, constructors, and behavior remain compatible. The legacy path
will not be removed before `v2.0.0`.
