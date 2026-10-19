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

Keep TiKVGroup and Updater behavior unchanged. Under `TiKVSequentialScaleIn`, each offlining TiKV adjusts its own incoming store limits, then checks the selection annotation before requesting store deletion.

## Motivation

The existing Updater can mark several TiKVs offline together. Their instance controllers independently evict leaders and delete stores, which can cause conflicting leader eviction and repeated peer migration.

### Goals

- Limit incoming peers and leaders on each offlining store.
- Select one store for deletion using the same deterministic ordering in each reconcile.
- Restore the original incoming limits when offline is cleared.

### Non-Goals

- Change Group-level offline marking, Pod handling, finalizers, or normal rolling-update eviction.
- Add a separate admission controller.
- Add spec/status fields, user-configurable rates, or timing settings.
- Change retained stores, remove-peer limits, or global scheduling settings.

## Proposal

Limit implementation changes to four parts:

1. Add a **Store Limit Adjust task** for offline limits and non-offline restoration.
2. Add a **Select TiKV task** to the Cluster controller. Pass its shared Order directly to the task.
3. Pass all offline TiKV instances from cache to Order.Sort, and Patch its `core.pingcap.com/selected-to-delete` annotation. The instance controller checks that annotation before DeleteStore.
4. Add **TiKVSequentialScaleIn** to switch between the existing and new offline behavior.

With the gate enabled, offline handling no longer requests leader eviction or waits for eviction. Other controller tasks retain their existing behavior.

### Risks and Mitigations

| Risk | Handling |
| --- | --- |
| PD does not support transfer-leader-in | Fail limit adjustment before reaching store deletion |
| One limit write fails | Keep the saved originals and retry adjustment on the current instance |
| Tiny positive limits are not a strict prohibition | Validate token-bucket, urgent scheduling, and Raft-election behavior with the supported PD version |
| Peers are observed at different times | Keep a successful selection pinned across observations and persist it before DeleteStore |

## Design Details

### Instance Reconcile Flow

The Cluster controller watches TiKV create, update, and delete events.

```text
Cluster reconcile
    |
    v
List TiKV instances from cache
    |
    v
Wait if a non-offline instance still has selection annotation
    |
    v
Order.Sort(cluster, all spec.offline=true instances)
    |
    v
Returns the selected TiKV
    |
    v
Patch selected-to-delete=true on the selected instance

TiKV instance reconcile
    |
    v
Store Limit Adjust
    |
    v
Before DeleteStore: check selected-to-delete
    |                         |
 Present                    Absent
    |                         |
    v                         v
DeleteStore(self)            Wait
```

Order is shared by Cluster reconciles and scoped by namespace and Cluster. Its constructor accepts the sorting and key functions. Sort persisted selections first, then Removing stores, creation timestamp, name, and UID.

Sort receives all TiKV instances with `spec.offline=true`, regardless of store state. It returns the instance with the previously selected name from the current input if present; otherwise it sorts the supplied slice in place and selects a new top. Empty input clears the Cluster’s selection. Order stores only selected keys, not candidate lists, and exposes only Sort.

If a non-offline instance still has a selection annotation, the Cluster controller keeps the removal slot occupied and does not call Sort. The TiKV instance controller clears that annotation with Patch only after its PD-backed offline task confirms cancellation or completed removal. A stale CR store state cannot release the slot. A failed Patch returns an error. An annotation write failure retains the in-memory selection for retry. On restart, persisted selection annotations take priority.

The instance controller checks the annotation before DeleteStore and clears it after confirmed cancellation. Selection remains in the Cluster controller.

### Module Responsibilities

| Module | Change |
| --- | --- |
| Cluster Controller | Watch TiKV create/delete, generation changes, and selection-annotation changes; pass Order to Select TiKV |
| Select TiKV task | Read TiKV instances from cache through apicall, wait for pending cancellation, call Sort with offline peers, and Patch the selected annotation |
| Utils Order | Return Sort from the supplied instances, retaining only the selected key while present |
| TiKV Instance Controller | Adjust its own limits and check its selection annotation before deletion |
| Store Limit Adjust task | Adjust or restore only the current instance's two incoming limits |
| TiKV offline task | Check the selection annotation, switch eviction behavior with the gate, and always call the common offline task with an `isSelected` boolean |
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

When offline is cleared, reuse existing cancellation handling. Restore settings still equal to the reduced rate and remove the annotation. Preserve settings changed by another writer. The store-limits annotation stores limit values only; the separate selected-to-delete annotation persists deletion selection.

### Test Plan

- Offline adjustment changes only add-peer and transfer-leader-in on the current store.
- Already limited stores need no additional writes.
- Partial write failures retain the original values for retries and restoration.
- Non-offline restoration proceeds even while the store is Removing and preserves external changes.
- Candidate ordering is deterministic; an unselected reconcile never deletes another store.
- Persisted selections take priority on restart; successful selections remain stable across observations.
- Failed annotation writes prevent DeleteStore; successful writes precede DeleteStore.
- Selection remains stable while its name is observed; pending cancellation retains the annotation until PD confirms completion.
- Stale instance feature gates cannot bypass selection or limit adjustment.
- Status-only TiKV updates do not enqueue the Cluster; selection-annotation changes do.
- The gate switches offline eviction behavior and preserves the disabled flow.

Integration validation remains necessary for complete scale-in, cancellation, and PD scheduling behavior.

### Feature Gate

`TiKVSequentialScaleIn`: Alpha, disabled by default, no Pod restart required. Selection, store-limit adjustment, and offline handling all use the Cluster gate directly, including during instance feature propagation. Register it at the next feature-log revision without changing published history.

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
