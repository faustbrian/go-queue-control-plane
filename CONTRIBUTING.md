# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected package's docs.
2. Run focused standalone Go tests for the affected package.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable decision entries, and complete the Specification Decisions
section of the pull request template. An unresolved interpretation or stale
source pin is release-blocking; peer behavior cannot silently select policy.

Explicit full mutation qualification must finish with zero surviving viable
mutants; mutation is not a universal prerequisite for ordinary edits.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Use Go 1.27.2 for verification; the public module minimum remains Go 1.27.0.
Standalone untagged tests require neither sibling repositories nor building
the verifier release from source:

```bash
GOWORK=off go test ./... -count=1 -timeout=3m
```

`make check` and `make ci` consume the checksum-pinned public CLI with bounded
`--local --all` verification: deterministic module tests, saved fuzz seeds,
and short active smoke for affected fuzz targets. They still use the manifest's
PostgreSQL, Redis and Valkey services and `integration` tag; have Docker and
fixtures available. Direct untagged Go tests are separate from that configured
route, not equivalent live-service proof. No goal ledger is a contributor
prerequisite.

The selected local risk checks verify cancellation refuses dispatch, and that
shutdown closes a slow connection within its bound. They run once, with a
combined three-minute declared budget; only the concurrent server case races:

```bash
GOWORK=off go test ./control -run '^TestServiceExecuteCancelsOnlyBeforeDispatch$' -count=1 -timeout=90s
GOWORK=off go test -race ./server -run '^TestServerBoundsSlowShutdown$' -count=1 -timeout=90s
```

Normal Go tests run saved fuzz seeds, not active exploration. Select one
changed target for a two-second active run, and optionally one package for local
mutation:

```bash
GOWORK=off go test ./apihttp -run '^$' -fuzz '^FuzzCommandHandlerFailsClosed$' -fuzztime=2s -timeout=2m
golib mutation --module . --package ./control
```

Agent-run Go commands must use and clean up a task-owned disposable `GOCACHE`.
The cheap `bash scripts/check-docs.sh --local` route retains structure, text and
link refusals without running Go. Ordinary Make and the checkout-only hosted
documentation boundary both run it; Required needs both that boundary and
shared CI to succeed.

PR, main and scheduled shared CI are bounded. Explicit `make check-full` and
`release_dry_run` retain full race, coverage, mutation, API/security checks,
three 10,000-iteration API fuzz targets (two-minute limits), PostgreSQL
integration and disaster recovery, Redis/Valkey integration, Chromium browser
qualification, documentation examples and allocation-budget benchmarks. The
four full test operations and full docs/benchmark operations keep their
20-minute limits. Browser qualification needs npm and Playwright/Chromium;
PostgreSQL scripts default to a disposable PostgreSQL 18 Docker fixture.
`make specification-check` is offline; `make specification-check-online` and
full qualification retain online authority. See the [release guide](docs/releasing.md)
for explicit release and public-consumer qualifications. Report unavailable or
failed gates without describing the ordinary route as release-ready.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
