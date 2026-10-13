# Release process

[Documentation index](README.md)

No production release should be tagged until the attached project acceptance
criteria and every required automation gate pass against the release commit.

## Required evidence

- deterministic format, module checksum, vet, Staticcheck, strict
  golangci-lint, advisory NilAway, test, exact meaningful coverage, race,
  build, fuzz, and leak gates;
- PostgreSQL migration, retention, backup, restore, and upgrade tests;
- fake and real `queue` management conformance;
- Redis and Valkey integration through `queue`, never direct clients;
- rolling worker protocol, disconnect, stale, delayed, duplicate, timeout,
  partial result, restart, and backend-failure tests;
- authorization mutation testing for every action;
- API fuzzing and browser security tests;
- large fleet, reconnect storm, queue, failure, maximum payload, history, and
  backend-outage load benchmarks with enforced allocation budgets;
- vulnerability and dependency scanning;
- documentation and API compatibility validation;
- reproducible multi-platform images, SBOM, provenance, signatures, and
  verification instructions.

Explicit full qualification covers Go formatting, module tidiness and checksum verification,
vet, Staticcheck, strict golangci-lint, advisory NilAway, tests, race, exact
100% statement coverage, builds, a fuzz smoke test, a high-severity browser
dependency audit, real PostgreSQL 18 migration and persistence integration,
an isolated PostgreSQL 18 native
backup-and-restore drill, the production one-shot audit and safe terminal-
command retention path, pinned Go
vulnerability scanning, 100% administrative mutation efficacy and coverage,
targeted HTTP lifecycle leak assertions, public Go API baseline compatibility,
authenticated managed-queue and rolling-protocol HTTP integration, real
Redis 6.2.22 and Valkey 9.1.0 lifecycle/status integration through `queue`,
including a concurrent retry/delete race with exactly one truthful winner,
real Chromium CORS, preflight, CSRF, and defensive-header tests. The current
shared workflow does not define a PostgreSQL 16/17/18 matrix or a
multi-platform OCI build job. Full qualification also runs the
eight 10,000-worker, 100,000-audit-event, maximum-page, maximum-payload,
reconnect-storm, and backend-outage benchmarks with allocation budgets but
without a noisy hosted-runner latency threshold. Ordinary PR, main and scheduled
shared CI use bounded local verification plus the documentation boundary; use
`make check-full` or the explicit release dry-run for these full campaigns.
The configured integration path covers authenticated Redis Streams and
Valkey Streams failure management, but
not the remaining transport-level queue and failure load items above. The
published `v1.0.0` GitHub release includes a source archive, module file,
CycloneDX SBOM, in-toto provenance statement, checksum manifest, SSH signature,
and allowed-signer file.

The published `v2.0.0` release follows that seven-asset source pattern. Its
module archive and module file come from the public Go proxy. The CycloneDX
SBOM and in-toto provenance statement were assembled locally by the maintainer,
not built by CI. The SBOM describes the Darwin/arm64 dependency graph and lacks
detected license evidence for `github.com/pressly/goose/v3`; it is not a complete
legal-license report. No deployable binary or image is attached.

The published `v2.0.1` release retains seven source-only assets. Its module
archive was generated from immutable main source `6da9d0a2` using Go's module
ZIP rules; every archived source file matches the public proxy archive even
though raw ZIP encoding differs. The module file matches the public proxy.
The maintainer-local CycloneDX SBOM covers the application and runtime Go
dependency graph, not tests, standard-library packages, frontend dependencies,
or a complete license report. Its in-toto provenance records local generation,
not CI-built artifacts or a certified deployment. Check the signed checksums
before using these assets.

## Versioning and changelog

The module starts its public release history at `v1.0.0`; published `v2.0.0`
uses the root `/v2` module path. Keep historical v1 release notes and
assets immutable. A new major tag is created from main, not from a version
directory or branch. Update `CHANGELOG.md` in
the same pull request as every user-visible change. Before release, move
Unreleased entries into a dated version section, verify upgrade and rollback
guidance, and confirm `/version` reports the tag, commit, and RFC3339 build time.

Install the checksum-qualified public `golib` v2.1.3 binary for the current platform
from the [tooling release](https://github.com/faustbrian/go-library-tools/releases/tag/v2.1.3).
For example, on macOS arm64:

```sh
TOOLING_VERSION=2.1.3
TOOLING_ARCHIVE=golib_2.1.3_darwin_arm64.tar.gz
TOOLING_DIR=$(mktemp -d)
gh release download "v${TOOLING_VERSION}" \
  --repo faustbrian/go-library-tools \
  --pattern checksums.txt --pattern "$TOOLING_ARCHIVE" \
  --dir "$TOOLING_DIR"
TOOLING_CHECKSUMS_SHA256=$(awk '$1 == "tool_checksums_sha256:" { print $2 }' .golib.yaml)
printf '%s  %s\n' "$TOOLING_CHECKSUMS_SHA256" "$TOOLING_DIR/checksums.txt" | shasum -a 256 --check
awk -v archive="$TOOLING_ARCHIVE" '$2 == archive { print }' \
  "$TOOLING_DIR/checksums.txt" | \
  (cd "$TOOLING_DIR" && shasum -a 256 --check)
tar -xzf "$TOOLING_DIR/$TOOLING_ARCHIVE" -C "$TOOLING_DIR" golib
export PATH="$TOOLING_DIR:$PATH"
golib --version
```

Choose the next semantic version and update the module `version` in
`modules.json` and the changelog. Fetch current remote tags with
`git fetch --tags origin`, then run `golib release check` and
`golib release dry-run` from the repository root. The dry-run rejects an
existing fetched tag. These commands validate release metadata, the module
archive, clean consumer resolution, and the complete repository contract.
Neither command publishes; publication is a separate operation bound to the
reviewed commit and versioned, checksum-bound release assets. Remove the
task-owned tooling download directory after qualification is finished.

For `v1.0.0`, download every asset into an empty directory, check the release's
signed checksum manifest, and verify every listed payload:

```sh
gh release download v1.0.0 --repo faustbrian/go-queue-control-plane
ssh-keygen -Y verify -f ALLOWED_SIGNERS -I brian@cline.sh \
  -n golib-release -s SHA256SUMS.sig < SHA256SUMS
shasum -a 256 --check SHA256SUMS
```

The release assets and notes record the exact source identity and checksums.
Because `ALLOWED_SIGNERS` is distributed with the release, this procedure
checks integrity against the release-declared signer; it does not independently
authenticate the publisher.
Do not treat a locally built image as one of those signed public artifacts.

Continue with [HTTP API reference](api.md).
