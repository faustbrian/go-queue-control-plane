# Postal v2 Worker Observation Repair

## Contract And Scope

Backport the remote worker classification correction to the published v2.0.1
baseline without changing module identity, PostgreSQL migration ownership,
timestamps, or public API. Remote acquisition can produce heartbeats after the
initial request clock; classify against the completed acquisition clock while
still rejecting truly future timestamps and preserving stale boundaries.

The authenticated HTTP handler and remote FleetSource are the affected owners.
Postal consumes v2 and must not require the unrelated v3 migration API change.
The human authorized a scoped v2 maintenance branch and hosted CI on 2026-10-12.

## Verification And Delivery

The native HTTP regression exercises paginated remote acquisition, every state
filter, nanosecond future/stale boundaries and preservation of timestamps.
Capture its failure against unchanged v2, then run affected packages, targeted
race, public API compatibility and the native release checks. One independent
complete-diff review is required before publication.

The native regression fails against v2.0.1: newly retrieved workers are unknown
and the exact stale boundary is incorrectly classified as running. The compatible
patch version is v2.0.2; the module and migration API remain v2.

Publication and Postal adoption remain pending. A local fix or hosted workflow
start does not establish a published module or deployed consumer.

The affected package tests and targeted race checks pass. The pinned Go 1.27.0
local contract, public API compatibility and release metadata check pass.
Independent complete-diff review approved the compatible patch.

The maintenance baseline's v1.6.2 release rehearsal runs exhaustive mutation
campaigns for every package, contrary to the current proportional Golib workflow.
Use checksum-locked v2.2.0 tooling without copying v3 source or migration changes.
Native bounded checks and hosted
release rehearsal remain required; exhaustive mutation is not an ordinary gate.
The manifest now explicitly retains API compatibility, documentation, lint,
security and tests as release gates. Exhaustive mutation/fuzz/benchmark, universal
coverage and repository-wide race are not selected for this bounded fix. The
changed authenticated handler's deterministic regression and targeted race
remain required; this does not assert unexecuted full qualification.

The updated native security scanner identified five reachable vulnerabilities
in inherited `golang.org/x/net` v0.58.0, all fixed by v0.60.0. Adopt that patched
minor version and the necessary x/sync, x/sys, x/term and x/text resolution;
keep all owned Go library majors and migration dependencies unchanged. Rerun
security, compatibility, affected behavior and the complete final-diff review.

Backport the existing upstream main's rule-specific G101/G304 annotations for
public protocol header names and operator-owned bounded file reads. These
replace the v2 baseline's existing broad `nolint:gosec` declarations, preserving
the same runtime behavior and recording why those specific findings do not
identify credentials or request-controlled filesystem access.

The Go 1.27.0 standard library also has twelve reachable advisories fixed in
Go 1.27.2. CI and qualification use that patched compiler; module API identity
and the v2 migration dependency remain unchanged. Tooling v2.2.0 builds the
released security analyzers with its owned x/tools v0.50.0 importer graph,
supporting Go 1.27.2 rather than suppressing analysis failures. Earlier v2.1.3
and pinned-Go-1.27.0 checks are not security-readiness evidence.

Native G115 analysis could not follow the already validated retention bounds.
Use the existing checked numeric helpers at both conversion sites, preserving
the stricter batch limit, transaction behavior, SQL and public result types.
No G115 exclusion is needed. The native analyzer failure is the regression
artifact; the existing numeric/retention tests cover the unchanged behavior.
Release/container compiler metadata is aligned to Go 1.27.2 while the public
module retains its Go 1.27.0 minimum.
