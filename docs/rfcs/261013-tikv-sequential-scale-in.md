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

- [ ] (R) This design has been discussed and approved.
- [ ] (R) Unit and integration tests pass.
- [ ] (R) Supported PD versions and store-limit behavior have been validated.
- [ ] (R) A complete scale-in has been tested in kind.

## Summary

Keep TiKVGroup and Updater behavior unchanged. Under `TiKVSequentialScaleIn`, each offlining TiKV adjusts its own incoming store limits, then uses the shared peer context to check whether it is selected before requesting store deletion.

## Motivation

The existing Updater can mark several TiKVs offline together. Their instance controllers independently evict leaders and delete stores, which can cause conflicting leader eviction and repeated peer migration.

### Goals

- Limit incoming peers and leaders on each offlining store.
- Select one store for deletion using the same deterministic ordering in each reconcile.
- Restore the original incoming limits when offline is cleared.

### Non-Goals

- Change Group-level offline marking, Pod handling, finalizers, or normal rolling-update eviction.
- Add locks, removal-request markers, a separate admission coordinator, or new event handlers.
- Add spec/status fields, user-configurable rates, or timing settings.
- Change retained stores, remove-peer limits, or global scheduling settings.

## Proposal

Limit implementation changes to four parts:

1. Add a **Store Limit Adjust task** for offline limits and non-offline restoration.
2. Add **Context Peer Slice** to the TiKV controller to load TiKV peers across Groups in the same Cluster.
3. Add a **self-selection condition at store deletion**. An unselected instance returns Wait.
4. Add **TiKVSequentialScaleIn** to switch between the existing and new offline behavior.

With the gate enabled, offline handling no longer requests leader eviction or waits for eviction. Other controller tasks retain their existing behavior.

### Risks and Mitigations

| Risk | Handling |
| --- | --- |
| PD does not support transfer-leader-in | Fail limit adjustment before reaching store deletion |
| One limit write fails | Keep the saved originals and retry adjustment on the current instance |
| Tiny positive limits are not a strict prohibition | Validate token-bucket, urgent scheduling, and Raft-election behavior with the supported PD version |
| Peers are observed at different times | Validate candidate changes, cancellation, and restart behavior separately; do not add a coordination mechanism in this change |

## Design Details

### Instance Reconcile Flow

Each reconcile operates only on its current instance X.

```text
Reconcile X
    |
    v
Context Peer Slice: load TiKV peers
    |
    v
Store Limit Adjust
    |-- offline: save originals and limit X
    |-- not offline: restore X's limits without waiting for its store to return to Up
    |
    v
Existing offline handling
    |
    |-- gate disabled: keep existing eviction and deletion flow
    |
    v
Before DeleteStore: sort candidates and check self
    |                         |
 X selected              Another TiKV selected
    |                         |
    v                         v
DeleteStore(X)               Wait
    |
    v
Existing store observation and cleanup
```

Candidates are offlining TiKVs in the peer slice, plus Removing stores whose offline flag has been cleared for cancellation. Exclude Removed stores. Sort Removing stores first, then creation timestamp, name, and UID, all ascending. Compare the selected UID with the current instance UID.

Given the same candidate set, parallel reconciles select the same TiKV. Only that instance calls DeleteStore for itself. There is no lock or second check of incoming limits at the deletion point.

### Module Responsibilities

| Module | Change |
| --- | --- |
| TiKV Instance Controller | Load Context Peer Slice and run Store Limit Adjust before offline handling |
| Store Limit Adjust task | Adjust or restore only the current instance's two incoming limits |
| TiKV offline task | Switch eviction behavior with the gate and always call the common offline task with an `isSelected` boolean |
| Common offline task | Check `isSelected` immediately before deleting a store; other offline and cancellation handling remains unchanged |
| PD API Client | Read store limits and write explicitly named incoming limit types |
| Feature Gates | Register the new gate through the existing generation mechanism |

All other controller behavior remains unchanged.

### Incoming Store Limits

Use fixed code constants through the existing per-store limit endpoint:

| Limit | Request body |
| --- | --- |
| Add peer | `{"type":"add-peer","rate":0.000001}` |
| Transfer leader in | `{"type":"transfer-leader-in","rate":0.000001}` |

Both requests use `POST /pd/api/v1/store/{id}/limit`. PD requires positive rates. Transfer-leader-in is a per-store incoming rate, not global scheduling concurrency. Skip writes when the current rate already equals the target.

### Cancellation and Recovery

Before the first limit write, save the original two values in the current instance's `core.pingcap.com/store-limits` annotation, using the format `add-peer:10,transfer-leader-in:10`. Do not overwrite those originals on retries.

When offline is cleared, reuse existing cancellation handling. Restore settings still equal to the reduced rate and remove the annotation. Preserve settings changed by another writer. This annotation stores limit values only; it does not coordinate store deletion.

### Test Plan

- Offline adjustment changes only add-peer and transfer-leader-in on the current store.
- Already limited stores need no additional writes.
- Partial write failures retain the original values for retries and restoration.
- Non-offline restoration proceeds even while the store is Removing and preserves external changes.
- Candidate ordering is deterministic; an unselected reconcile never deletes another store.
- Removing candidates take priority, and Removed candidates no longer delay selection.
- The gate switches offline eviction behavior and preserves the disabled flow.

Integration validation remains necessary for complete scale-in, cancellation, and PD scheduling behavior.

### Feature Gate

`TiKVSequentialScaleIn`: Alpha, disabled by default, no Pod restart required. Register it at the next feature-log revision without changing published history.

```yaml
spec:
  featureGates:
    - name: TiKVSequentialScaleIn
      enabled: true
```

Gate-off retains the existing offline flow. Restoration of saved limits still runs when the instance becomes non-offline. Handling gate changes while instances remain offline is deferred for separate discussion.

## Drawbacks

Sequential removal can take longer. Incoming limits suppress scheduling onto removal targets but do not throttle leader transfers into retained stores or guarantee a particular cache impact.

## Alternatives

- Keep concurrent eviction and deletion: retains the existing scheduling conflicts.
- Change Group-level offline marking: changes Updater behavior beyond this proposal.
- Limit global leader scheduling: affects a different mechanism and the whole Cluster.
