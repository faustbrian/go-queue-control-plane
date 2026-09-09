# Engineering Policy

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
"OPTIONAL" in this document are to be interpreted as described in BCP 14
[RFC2119] [RFC8174] when, and only when, they appear in all capitals, as
shown here.

## Scope And Authority

- This file is the canonical policy for the complete repository.
- Package policies MAY add stricter domain rules but MUST NOT weaken this file.
- `CLAUDE.md` and tool-specific files MUST point here rather than duplicate it.
- Historical `.ai/GOAL*.md` files are requirements and evidence, not proof of
  completion. Current executable evidence is REQUIRED.

## Repository Structure

- The public root module MUST live at the repository root.
- Intentional optional or test modules MAY live in explicit nested directories.
- Commands MUST live under `cmd/`; private shared code MUST live under
  `internal/`; root automation MUST live under `scripts/`.
- Public module paths MUST match their repository-relative directories beneath
  the module path declared by the root `go.mod`.
- Every module MUST be declared in `modules.json`, and every package MUST be
  declared in `packages.json`.
- Independently releasable modules MUST retain independent `go.mod` files and
  directory-prefixed semantic-version tags.
- Cross-module dependencies MUST remain acyclic and MUST use public contracts.
- Permanent `replace` directives, sibling repositories, and absolute developer
  paths are forbidden in releasable modules.

## Design

- Prefer standard-library interfaces and explicit composition over hidden
  registration, global state, reflection-driven wiring, or service locators.
- Public APIs MUST make ownership, cancellation, retries, timeouts, resource
  limits, error semantics, and concurrency behavior observable.
- Interfaces SHOULD be defined by consumers and MUST remain narrowly scoped.
- Optional integrations SHOULD be adapters or nested modules, not mandatory
  dependencies of a core package.
- Breaking protocol or specification ambiguities MUST be documented as explicit
  decisions and covered by tests.

## Safety And Concurrency

- Shared mutable state MUST have one documented synchronization owner.
- Goroutines MUST have explicit lifetime, cancellation, and shutdown behavior.
  Leak tests MUST run when a changed lifecycle or concurrency risk requires
  them. Fire-and-forget goroutines are forbidden.
- Channels MUST have documented ownership and closure rules.
- Locks MUST NOT be held across caller callbacks, network IO, blocking channel
  operations, or unbounded work.
- Every external operation MUST accept or derive a bounded `context.Context`.
- Response bodies, files, rows, transactions, timers, tickers, connections,
  and temporary resources MUST be closed on every path.
- Integer conversions, sizes, offsets, recursion, decompression, and allocation
  from untrusted input MUST be bounded before allocation or conversion.
- Secrets and credentials MUST NOT appear in errors, logs, traces, snapshots,
  fixtures, mutation reports, or generated artifacts.

## Testing

- Behavioral changes MUST include meaningful tests before completion.
- Tests MUST assert outcomes, invariants, errors, cleanup, and state transitions;
  line execution without behavioral assertions is not acceptable coverage.
- Tier A documentation and metadata changes require only affected structural,
  link, example, and generation checks.
- Tier B internal behavior changes require focused tests, affected module tests,
  applicable static checks, and one complete review.
- Tier C public API, lifecycle, security, persistence, and concurrency changes
  require observable regression evidence, API compatibility where applicable,
  affected module and integration tests, direct owned consumers, and one
  independent complete-diff review.
- Tier D public releases and ecosystem milestones require only the relevant
  compatibility, composition, consumer, and aggregate checks against immutable
  release inputs.
- Race, fuzz, mutation, leak, performance, conformance, and external-service
  checks MUST run only when they exercise a material risk or release boundary.
  They MUST NOT block unrelated changes merely because the check exists.
- Invalid or equivalent mutants require a narrow reviewed record only when
  mutation testing is selected for the affected risk.
- Specification claims MUST be proven against pinned official fixtures and
  independent implementations where applicable.
- Benchmarks selected for a performance claim MUST compare equivalent behavior
  and publish latency, throughput, allocations, environment, corpus, and
  statistical method.

## Required Commands

- `make inventory` validates repository and package manifests.
- `golib check --module <directory>` runs one affected module contract.
- `make check` runs every repository module and is reserved for repository-wide
  changes and release milestones.
- `make ci` runs the complete repository release contract.
- Local and CI invocations of the same selected gate MUST use the same scripts
  and thresholds.
- Missing tools, services, packages, profiles, or reports selected by the
  applicable risk tier MUST fail.
- NilAway is advisory; its findings MUST remain visible and tracked against a
  no-regression baseline.

## Evidence Validity And Reuse

- Evidence MUST directly exercise the observable claim and affected risk.
- Evidence bound to unchanged immutable source, dependency, tool, and
  environment inputs SHOULD be reused instead of rerun.
- A commit hash MAY identify an execution but MUST NOT force a rerun after a
  history-only or unrelated change.
- After a change, rerun only the affected modules, checks, and direct consumers.
- Missing or ambiguous applicability requires fresh execution of the affected
  check, not a repository-wide restart.
- Routine logs, coverage files, mutation reports, fuzz corpora, caches, and
  temporary environments MUST remain disposable unless they are maintained
  inputs or public release artifacts.

## CI And Workflows

- `.github/workflows/ci.yml` is the only owned GitHub Actions workflow.
- Package-local workflows MUST NOT be added.
- Actions and external tools MUST be pinned to immutable versions.
- Every module selected by the applicable assurance tier MUST have an
  attributable result. Durable evidence artifacts are required only when the
  selected gate or public release contract requires them.
- The stable required job MUST fail for failed, cancelled, skipped, or missing
  module results.
- Required checks MUST NOT use `continue-on-error`, `|| true`, permissive
  thresholds, or warning substitutions.

## Dependencies And Supply Chain

- Dependencies MUST be necessary, maintained, license-compatible, and pinned to
  reviewed current versions.
- Standard-library functionality MUST NOT be wrapped merely to create an owned
  abstraction; wrappers require a stable policy or portability boundary.
- Generated code and vendored corpora MUST record source, version, checksum,
  license, generation command, and update procedure.
- Vulnerability, secret, license, SBOM, provenance, and clean-consumer checks
  MUST run at a public release boundary only when they exercise an applicable
  supply-chain or consumer risk.

## Documentation

- Public identifiers MUST have useful Go documentation describing semantics,
  invariants, ownership, errors, concurrency, and caveats where relevant.
- Comments MUST explain why a constraint or non-obvious implementation exists;
  they MUST NOT narrate obvious syntax.
- Public module documentation MUST cover the setup, API, examples, adoption,
  limitations, security, and release information applicable to its audience.
  It MUST NOT require empty or duplicative sections solely to satisfy a fixed
  documentation bundle.
- Documentation and examples selected by the applicable assurance tier MUST be
  checked in CI.

## Changelogs

- Every user-visible change MUST update the affected module `CHANGELOG.md` in
  the same commit.
- Entries MUST describe behavior and migration impact, not internal activity.
- Changes to multiple modules MUST update each changelog whose users observe
  the change.
- Unreleased entries MUST NOT be silently rewritten or removed.
- Generated and dependency changes require changelog entries only when they
  alter public behavior, adoption, compatibility, security, or release output.

## Completion

- Classify each change by its actual assurance tier and run the narrowest gates
  that prove its observable contract and material risks.
- Run release, composition, and clean-consumer checks only at the applicable
  public delivery boundary.
- One complete independent review is the default for a meaningful public
  contract or release batch; additional reviews require a distinct named risk.
- Re-run affected gates after the final source, test, dependency, documentation,
  workflow, or generated-file change.
- Report exact selected commands and results. A skipped, blocked, stale, or
  warning-only selected gate is not a pass.
