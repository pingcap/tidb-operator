# Sequential TiKV Scale In

Status: Draft.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Instance Reconcile Flow](#instance-reconcile-flow)
  - [Module Responsibilities](#module-responsibilities)
  - [Incoming Store Limits](#incoming-store-limits)
  - [Cancellation and Recovery](#cancellation-and-recovery)
  - [Test Plan](#test-plan)
  - [Feature Gate](#feature-gate)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required prior to targeting a release.

- [ ] (R) This design has been discussed and approved.
- [ ] (R) Test plan is in place, including e2e tests in kind.
- [ ] (R) Supported PD versions and store-limit modes have been validated.
- [ ] (R) Feature graduation criteria are defined.
- [ ] (R) User-facing documentation is available in the TiDB Operator documentation.

## Summary

Change the TiKV Instance Controller's offline handling to suppress incoming peers and leaders, then remove stores one at a time. Reconciles remain parallel and operate only on their own instances. TiKVGroup and Updater behavior stay unchanged. The feature is controlled by `TiKVSequentialScaleIn`.

## Motivation

Currently, the Updater can mark several TiKV instances offline together. Each instance starts leader eviction, waits for eviction or its timeout, then calls DeleteStore independently. Concurrent eviction can leave Regions without an eligible leader target, while rapid leader movement can increase cache warm-up pressure. Migrating peers onto stores that are also being removed causes repeated migration.

### Goals

- Keep existing target selection and offline marking.
- Immediately suppress peers and leaders entering each offlining store.
- Allow only one store removal at a time per Cluster, with each reconcile acting only on itself.
- Remove offline-triggered leader eviction and reuse existing completion and cancellation paths.

### Non-Goals

- Change rolling-update eviction or TiFlash scale-in.
- Limit leader transfers into retained stores or modify global leader/replica scheduling concurrency or remove-peer limits.
- Add spec/status fields, user-configurable rates, or additional timing settings.

## Proposal

Instance reconciles immediately apply their own add-peer and transfer-leader-in limits. Before DeleteStore, each reconcile sorts the offlining instances and checks whether the selected instance is itself. Otherwise, it waits. Offline no longer requests leader eviction.

| Behavior | Current | New |
| --- | --- | --- |
| Select instances and set offline | Updater may mark several instances offline | Unchanged |
| Prepare incoming limits | No scale-in-specific incoming limits | Apply add-peer and transfer-leader-in limits to the current store immediately after observing offline |
| Evict leaders | Offline triggers eviction and an eviction wait | No offline-triggered eviction or eviction wait |
| DeleteStore | Multiple instances may remove stores concurrently | Only the selected instance removes its own store |
| Completion | PD Removed, followed by CR deletion and finalization | Unchanged; Removed allows the next store to proceed |
| Cancel | Clear offline and cancel store removal | Reuse cancellation and restore its original add-peer and transfer-leader-in limits |

### Risks and Mitigations

| Risk | Handling |
| --- | --- |
| Unsupported PD capability | Require transfer-leader-in support introduced by PD PR #11197 and manual v1 store limits; block new removal rather than silently skipping either limit |
| Tiny positive limits are not a strict prohibition | Token buckets, in-flight work, urgent paths, and Raft elections can allow incoming work; validate actual behavior and do not promise absolute zero |
| Concurrent removal attempts | Revalidate admission in a short Cluster-scoped critical section and retain a per-instance removal-request marker |
| Regions whose peers are all on offlining stores may have restricted transfer targets | Validate this layout in integration/e2e tests; do not bypass the limits or remove another store when progress stalls |
| Temporary limits survive interruption | Persist original values before writing PD and restore owned values on cancellation/recovery |

## Design Details

### Instance Reconcile Flow

Every reconcile operates only on its current instance X. Reading other instances does not authorize modifying their CRs or stores. An offlining store is **limited** when both its add-peer and transfer-leader-in limits have been applied and verified.

```text
Reconcile current instance X
         |
         v
Observe X is offline
         |
         v
Ensure X is limited: check and apply
add-peer and transfer-leader-in limits
         |
         v
Read and sort offlining instances
         |
         v
Is X selected? ---------------- No ----------------> Wait
         |
        Yes
         |
         v
Is X Removing?
         |                          |
        Yes                        No
         |                          |
         v                          v
Observe X until Removed    DeleteStore for X only
                                    |
                                    v
                           Observe X until Removed
```

1. **Ensure X is limited immediately.** Check X's add-peer and transfer-leader-in values. If needed, save their originals, apply the fixed limits, and verify both writes. If X is already limited, skip the writes. Complete this step before sorting, including when another store is Removing. Each instance handles its own limits; failures use existing retries and stop this reconcile from reaching DeleteStore.
2. **Read the candidate set.** Include offlining TiKVs in the same namespace and Cluster, across TiKVGroups, until PD confirms Removed. A Removing store still blocks another deletion even if its offline flag was cleared for cancellation.
3. **Sort and check self.** Keep a Removing store first. Otherwise sort by creation timestamp, name, and UID, all ascending. If the selected UID differs from X's UID, return Wait without operating on the selected instance.
4. **Check Removing and delete self.** If X is Removing, observe its progress. Otherwise, call DeleteStore for X directly. The earlier limit step has already completed; do not repeat the limited check here.
5. **Wait for Removed.** A successful DELETE only starts removal. Once PD confirms Removed, exclude X from selection and wake waiting instances. Their own reconciles perform the next deletion. Existing resource cleanup proceeds independently.

For A before B, reconcile(B) applies B's two incoming limits and returns Wait. Reconcile(A) deletes A after its own limit step succeeds and A is selected. After A is Removed, reconcile(B) can delete B. Reconcile(A) never deletes B.

Reconciles and per-instance preparation remain parallel. Use a Cluster-scoped mutex only for the final fresh-read/select/self-check/DeleteStore admission section, never while waiting for removal. Reuse Operator leader election. Record an internal removal-request marker on X before DELETE to retain the selected instance across retries and restarts. If multiple stores are already Removing, admit no additional removal until they finish.

### Module Responsibilities

| Module | Responsibility |
| --- | --- |
| TiKVGroup Controller and Updater | Keep target selection, offline marking, replica accounting, and cancellation intent unchanged |
| TiKV Instance Controller | Apply its own two incoming limits, sort candidates, check self, admit one store removal, and restore its original values |
| TiKV Instance Controller — Pod and leader eviction handling | Do not request eviction because an instance is offline; preserve normal rolling-update eviction |
| TiKV Instance Controller — event handling | Wake waiting instances when offline membership, preparation readiness, or store removal state changes |
| PD Client Manager | Reuse existing Store events, cache, and refresh mechanisms |
| PD API Client | Provide add-peer and transfer-leader-in limit reads/writes; reuse DeleteStore and CancelDeleteStore |
| Feature Gates | Register the new gate through the existing feature-generation mechanism |

The Task framework continues later tasks after Wait or Retry. Pod and leader eviction handling must therefore preserve the rule that offline alone never starts eviction or waits for leaders to be evicted. CR deletion and finalizer handling remain unchanged.

### Incoming Store Limits

Immediately after observing offline, each instance sets these two limits on its own store through `POST /pd/api/v1/store/{id}/limit`:

| Limit | Request body | Purpose |
| --- | --- | --- |
| Add peer | `{"type":"add-peer","rate":0.000001}` | Suppress new peers entering the offlining store |
| Transfer leader in | `{"type":"transfer-leader-in","rate":0.000001}` | Suppress leaders entering the offlining store |

Both values are fixed in code. The API requires positive rates, so use `0.000001` rather than zero and always specify the limit type. Transfer-leader-in is a per-store incoming rate expressed per minute, not a global concurrency limit. Retained stores and other scheduling limits remain unchanged. Existing store observation, deletion, and cancellation operations are reused.

### Cancellation and Recovery

Each instance saves its original add-peer and transfer-leader-in values and removal-request marker in internal annotations bound to its store ID. Save before writing PD, do not replace originals with already reduced values, and reuse the records after restart.

When offline is cleared, only that instance's reconcile restores both original incoming-limit values. If its store is Removing, first reuse CancelDeleteStore and wait for Up. Removed stores remain irreversible and follow existing cleanup. Restore only settings still equal to this feature's writes; preserve external changes.

### Test Plan

Unit tests cover admission, per-instance ownership, request payloads, and restoration. Integration tests validate the PD capability and concurrency behavior. E2e tests in kind cover a complete 6-to-3 scale-in, cancellation, and recovery:

- The existing Updater still marks multiple instances offline; their reconciles apply their own add-peer and transfer-leader-in limits in parallel before checking deletion order.
- A non-selected instance returns Wait and never modifies or deletes another instance.
- The limit step checks X once and skips writes when already limited. After selection, Removing waits and the No branch calls DeleteStore directly without another limited check. No second removal starts while one is Removing.
- Removed wakes the next instance without waiting for resource cleanup. Existing finalization still works.
- Offline never requests leader eviction or waits for eviction; normal rolling-update eviction is unchanged.
- Cancellation, concurrent reconciles, lost responses, restart, and gate changes preserve single-store removal and restore owned add-peer and transfer-leader-in settings. No path writes limits on another instance or changes global leader scheduling, remove-peer, or replica scheduling limits.
- Reject unsupported transfer-leader-in capability or store-limit mode; verify partial preparation cannot admit deletion. Cover Regions whose current peers are all on offlining stores to check that serial removal can still make progress.

### Feature Gate

`TiKVSequentialScaleIn`: Alpha, disabled by default, no Pod restart required. Register it at the next feature-log revision without modifying published history.

```yaml
spec:
  featureGates:
    - name: TiKVSequentialScaleIn
      enabled: true
```

With the gate disabled and no recovery records, retain the current behavior. Disabling during removal stops new admissions through this branch; finish or cancel the outstanding request and restore owned add-peer and transfer-leader-in settings before returning to legacy handling.

Beta graduation requires passing the supported-PD compatibility matrix, the serial-removal and recovery tests, and workload tests showing acceptable cache hit rate, latency, and throughput during scale-in. Validate the fixed rates before graduation.

## Drawbacks

Serial removal can increase total scale-in time. Applying both limits to offlining stores reduces scheduling choices. Because preparation is per-instance and parallel, another offlining store may still be applying its limits when the first removal starts. The proposal suppresses incoming leaders on removal targets but does not directly throttle migration into retained stores, so it cannot guarantee a particular cache impact or leader-transfer rate for the Cluster.

## Alternatives

- **Keep concurrent eviction and deletion:** retains the current scheduling conflicts and migration bursts.
- **Issue one DELETE per reconcile without checking other stores:** concurrent or subsequent reconciles can still start multiple removals.
- **Change Group-level offline marking to one instance at a time:** changes existing Updater behavior and leaves future removal targets without incoming limits.
- **Limit global leader scheduling concurrency:** controls a different mechanism and affects the entire Cluster; use only the requested per-store incoming limits.
