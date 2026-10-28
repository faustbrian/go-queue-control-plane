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

Version 6 removed the deprecated top-level `kubernetes` package. Import
`github.com/faustbrian/go-queue-control-plane/v6/adapters/kubernetes`, which now
owns the implementation; exported types, constants, error sentinels,
constructors and behaviour are unchanged. Published v1 to v5 keep the old path
and receive no further releases.
