# Release process

No production release should be tagged until the attached project acceptance
criteria and every required automation gate pass against the release commit.

## Required evidence

Follow the assurance tiers in [AGENTS.md](../AGENTS.md). `modules.json` selects
the native gates and `.golib.yaml` binds the affected integration operations.
The v2 maintenance release selects API compatibility, documentation, lint,
security and tests, including the existing PostgreSQL persistence/recovery,
queue integration and browser-security checks. Its worker-clock regression
also needs the focused race check.

Exhaustive mutation, fuzz campaigns, repository-wide race, benchmarks and an
exact statement-coverage target are not gates for this bounded patch. Select
them only for changes whose material risks require them; do not remove the
underlying checks or claim that unselected suites passed.

CI uses the checksum-locked tool version below and Go 1.27.2. The public v2
module still supports Go 1.27.0. Verify the immutable release source, public
consumer resolution and signed source assets before claiming publication.
Source releases do not establish deployed image provenance.

The OCI build path can produce BuildKit SBOM and provenance attestations. The
published `v1.0.0` GitHub release includes a source archive, module file,
CycloneDX SBOM, in-toto provenance statement, checksum manifest, SSH signature,
and allowed-signer file.

The published `v2.0.0` release follows that seven-asset source pattern. Its
module archive and module file come from the public Go proxy. The CycloneDX
SBOM and in-toto provenance statement were assembled locally by the maintainer,
not built by CI. The SBOM describes the Darwin/arm64 dependency graph and lacks
detected license evidence for `github.com/pressly/goose/v3`; it is not a complete
legal-license report. No deployable binary or image is attached.

## Versioning and changelog

The module starts its public release history at `v1.0.0`; published `v2.0.0`
uses the root `/v2` module path. Keep historical v1 release notes and
assets immutable. A new major tag is created from main, not from a version
directory or branch. Compatible v2 maintenance tags use the reviewed v2
maintenance branch; do not merge that branch over the v3 main module. Update
`CHANGELOG.md` alongside every user-visible change. Before release, move
Unreleased entries into a dated version section, verify upgrade and rollback
guidance, and confirm `/version` reports the tag, commit, and RFC3339 build time.

Install the checksum-verified `golib` v2.2.0 binary for the current platform
from the [tooling release](https://github.com/faustbrian/go-library-tools/releases/tag/v2.2.0).
For example, on macOS arm64:

```sh
TOOLING_VERSION=2.2.0
TOOLING_ARCHIVE=golib_2.2.0_darwin_arm64.tar.gz
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
