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
