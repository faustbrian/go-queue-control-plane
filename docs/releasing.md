# Release process

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

The current CI covers Go formatting, module tidiness and checksum verification,
vet, Staticcheck, strict golangci-lint, advisory NilAway, tests, race, exact
100% statement coverage, builds, a fuzz smoke test, a high-severity browser
dependency audit, real PostgreSQL 16, 17, and
18 migration and persistence integration, an isolated PostgreSQL 18 native
backup-and-restore drill, the production one-shot audit and safe terminal-
command retention path, pinned Go
vulnerability scanning, 100% administrative mutation efficacy and coverage,
targeted HTTP lifecycle leak assertions, public Go API baseline compatibility,
authenticated managed-queue and rolling-protocol HTTP integration, real
Redis 6.2.22 and Valkey 9.1.0 lifecycle/status integration through `queue`,
including a concurrent retry/delete race with exactly one truthful winner,
real Chromium CORS, preflight, CSRF, and defensive-header tests, Dockerfile
checks, and a multi-platform OCI build. It also smoke-runs the
eight 10,000-worker, 100,000-audit-event, maximum-page, maximum-payload,
reconnect-storm, and backend-outage benchmarks with allocation budgets but
without a noisy hosted-runner latency threshold.
The OCI build path can produce BuildKit SBOM and provenance attestations. It
covers authenticated Redis Streams and Valkey Streams failure management, but
not the remaining transport-level queue and failure load items above. The
published `v1.0.0` GitHub release includes a source archive, module file,
CycloneDX SBOM, in-toto provenance statement, checksum manifest, SSH signature,
and allowed-signer file.

## Versioning and changelog

The module starts its public release history at `v1.0.0`. Update `CHANGELOG.md` in
the same pull request as every user-visible change. Before release, move
Unreleased entries into a dated version section, verify upgrade and rollback
guidance, and confirm `/version` reports the tag, commit, and RFC3339 build time.

Install the checksum-verified `golib` v1.6.2 binary for the current platform
from the [tooling release](https://github.com/faustbrian/go-library-tools/releases/tag/v1.6.2).
For example, on macOS arm64:

```sh
TOOLING_VERSION=1.6.2
TOOLING_ARCHIVE=golib_1.6.2_darwin_arm64.tar.gz
TOOLING_DIR=$(mktemp -d)
gh release download "v${TOOLING_VERSION}" \
  --repo faustbrian/go-library-tools \
  --pattern checksums.txt --pattern "$TOOLING_ARCHIVE" \
  --dir "$TOOLING_DIR"
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
reviewed commit and versioned, checksum-bound release assets.

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
