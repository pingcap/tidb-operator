# Reduce Group Volume Requests

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Example](#example)
- [Design Details](#design-details)
  - [Group and Instance Specifications](#group-and-instance-specifications)
  - [PVC Synchronization](#pvc-synchronization)
  - [Instance Condition](#instance-condition)
  - [Condition Reconciliation](#condition-reconciliation)
  - [Scale-In Preference](#scale-in-preference)
  - [Implementation Areas](#implementation-areas)
  - [Compatibility and Feature Gate](#compatibility-and-feature-gate)
  - [Test Plan](#test-plan)
- [Risks and Mitigations](#risks-and-mitigations)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)

<!-- /toc -->

## Release Signoff Checklist

- [ ] This design doc has been discussed and approved.
- [ ] Unit and integration tests have been implemented.
- [ ] The end-to-end scenario has been verified.
- [ ] User-facing documentation has been updated.

## Summary

When VolumeAttributesClass (VAC) is enabled, allow users to decrease `spec.template.spec.volumes[].storage` on a group. Existing instances receive the smaller request in their specifications while retaining their existing PVC capacity. Newly created PVCs use the smaller request.

Add an instance condition, `VolumeCapacityExceedsRequest`, to report that at least one volume has more allocated capacity than the instance requests. This is a normal state that can persist indefinitely. It does not indicate a failure, pending work, or a promise to shrink or replace the volume.

Add `PreferVolumeCapacityExceedsRequest()` to the shared scale-in preferences so that subsequent scale-in operations prefer these instances within the candidates selected by existing topology policies.

## Motivation

A group may need less storage per instance after workload or capacity requirements change. Users should be able to lower the request for future instances while continuing to run existing instances with their current storage. When users later scale the group in, removing instances with excess capacity helps the group move toward its new storage configuration.

### Goals

- Accept a smaller group volume request and propagate it to instance specifications.
- Create new PVCs with the new request.
- Preserve existing PVCs and avoid submitting a smaller size to PVC APIs.
- Expose excess allocated capacity as an informational instance condition.
- Prefer instances with excess capacity during scale-in, subject to existing topology and lifecycle rules.
- Support the VolumeAttributesClass (VAC) synchronization path.

### Non-Goals

- Change legacy PVC synchronization or cloud volume modifiers.
- Shrink an existing disk or filesystem in place.
- Automatically replace instances or migrate data when storage requests decrease.
- Trigger scale-in or change replica counts based on this condition.
- Rank instances by the number of excess bytes.
- Change PVC retention, store offlining, or scale-in cancellation semantics.

## Proposal

### Example

A group starts with three instances, each requesting `100Gi`. The user changes the group's volume request to `50Gi` without changing replicas.

After the instance specifications and conditions converge:

| Instances | Instance volume request | PVC capacity | VolumeCapacityExceedsRequest |
| --- | --- | --- | --- |
| Three existing instances | `50Gi` | `100Gi` | `True` |

The existing instances continue running. The operator does not shrink their PVCs or initiate replacement because of this state.

The user then scales out to four replicas. Assuming the provisioner allocates exactly the requested capacity, the new instance has a `50Gi` PVC and reports `False`. If the user subsequently scales back to three replicas, an old instance with a `100Gi` PVC is preferred for removal when topology policies allow that choice.

A new PVC may be allocated more capacity than requested. Such an instance also reports `True`; the condition describes the current capacity relationship, regardless of whether a request was previously decreased.

## Design Details

### Group and Instance Specifications

The shared `Volume.Storage` API currently has no validation requiring requests to increase monotonically. No new group spec field is needed.

Keep the requested value in both the group template and the instance spec. Use the existing group revision and instance update mechanism to propagate changes. Do not clamp the instance spec to the existing PVC size: doing so would hide the user's desired request and prevent the condition from expressing it.

A storage-only decrease must converge without replacing the instance or restarting its Pod. Preserve the existing revision mechanism and verify that updating revision metadata alone does not cause a Pod restart. Other concurrent template changes retain their normal update behavior.

PVC overlays retain their existing precedence for PVC generation. The condition compares `instance.spec.volumes[].storage` with capacity, as its API definition states. If an overlay explicitly requests more storage, the condition can therefore be `True` even for a newly created PVC. Users who want new PVCs to request the smaller size must also adjust any storage override in the overlay.

### PVC Synchronization

Keep the user request separate from the value sent to storage APIs. Normalize the generated desired PVC directly; never mutate the instance volume spec. Get reads into a separate empty object so transformers receive the original expected object without a deep copy.

For a PVC that does not exist, create it using the generated desired request, including the existing overlay behavior.

For an existing PVC, retain its current request when either the current request or reported capacity already covers the generated desired request. Otherwise, apply the larger desired request:

```text
if desiredRequest <= currentPVCRequest or desiredRequest <= reportedCapacity:
    syncPVCRequest = currentPVCRequest
else:
    syncPVCRequest = desiredRequest
```

Compare quantities with `resource.Quantity.Cmp`, so equivalent values such as `1Ti` and `1024Gi` compare equally. Keeping the current PVC request preserves an expansion that has already been requested but has not completed. Do not raise a PVC request solely to match excess allocated capacity: a storage class without expansion support can reject that request change even though no disk resize is needed.

Apply this normalization before `SyncPVCs` applies the desired PVC, while retaining the existing immutable storage-class handling. `LegacySyncPVCs` retains its existing behavior and is outside this proposal.

Normalize only storage size. Continue reconciling labels, annotations, and VAC. A reduced user request must not suppress an otherwise valid attribute update or cause a permanent wait for capacity to equal that user request. Existing expansion and PVC condition handling still apply to unrelated work in progress.

The apply client exposes `Transformers []Transformer`, where `Transform(current, expected client.Object) client.Object` returns the next expected object without an error. After Get, transformers run in registration order before serialization and diff; current is read-only and nil for a new object. VAC capacity protection uses this interface.

Normalization must use current PVC data at the write boundary. Implement it alongside the apply path's existing-object read, or use a resource-version-guarded write and re-read on conflict. A separate stale read followed by an unconditional apply must not overwrite a concurrently increased request.

### Instance Condition

Add the shared condition constant `CondVolumeCapacityExceedsRequest` with value `VolumeCapacityExceedsRequest`.

For each declared instance volume, resolve its corresponding PVC using the same volume-to-PVC naming logic as PVC generation. Compare:

```text
PVC.status.capacity.storage > instance.spec.volumes[i].storage
```

Use the reported PVC capacity, not its request, for this condition. A larger PVC request alone may represent an expansion that has not completed.

Aggregate results across volumes as follows, in order:

| Condition status | Rule | Reason |
| --- | --- | --- |
| `True` | At least one resolved PVC has capacity greater than its volume request | `CapacityExceedsRequest` |
| `Unknown` | No known excess capacity, but at least one PVC is missing or has no reported storage capacity | `CapacityUnknown` |
| `False` | All declared volumes can be evaluated and none has excess capacity | `CapacityDoesNotExceedRequest` |
| `False` | The instance declares no volumes | `NoVolumes` |

`False` does not assert that expansion is complete or that capacity equals the request. A volume whose capacity is smaller than its request does not satisfy this condition; existing volume reconciliation handles expansion.

Example:

```yaml
status:
  conditions:
    - type: VolumeCapacityExceedsRequest
      status: "True"
      observedGeneration: 7
      reason: CapacityExceedsRequest
      message: 'Volume data (PVC example-data): capacity 100Gi exceeds request 50Gi.'
      lastTransitionTime: "2026-09-15T00:00:00Z"
```

For multiple volumes, list all known excess-capacity volumes in a deterministic order with volume name, PVC name, request, and capacity. Unknown messages identify the volumes whose capacity cannot be determined.

Use the existing condition helpers to maintain `observedGeneration` and `lastTransitionTime`. Recompute the condition when the spec or PVC capacity changes. Increasing the request to the existing capacity clears `True`; a request can also remain below capacity and keep the condition `True`.

This condition does not make an instance unready, unsynced, or abnormal, and must not independently emit warning events or trigger an abnormal-instance alert.

### Condition Reconciliation

Implement comparison and aggregation in shared code, and expose a shared instance condition task. Like `TaskInstanceConditionSynced`, `TaskInstanceConditionVolumeCapacityExceedsRequest` updates the in-memory condition and calls `SetStatusChanged()` only when it changes. The existing controller status task persists the change; the condition task does not write status directly. Reuse PVC observations within a reconcile where practical. Read errors must remain retryable errors; missing PVCs or missing capacity are the explicitly defined `Unknown` cases.

Integrate the task with the status persistence path for each instance controller. Ensure a PVC sync wait does not prevent the new condition from being persisted. When reconciliation handles suspended instances, report the relationship for retained PVCs without creating or modifying storage solely to calculate the condition. Preserve the existing pause behavior.

Verify that each affected instance controller watches its owned PVCs and that instance status changes reach the group controller. These events update the condition and scale-in preference without polling merely because excess capacity exists.

### Scale-In Preference

Add `PreferVolumeCapacityExceedsRequest[R runtime.Instance]()` as a shared `PreferPolicy` in `pkg/updater`.

The policy returns candidates whose condition has both:

- `status == True`;
- `observedGeneration == instance.metadata.generation`.

Missing, stale, `False`, and `Unknown` conditions receive no preference. If there are no preferred candidates, return an empty result so the selector continues with its existing fallback behavior. If several candidates qualify, lower-priority policies break the tie.

Register the policy in the shared builder's default scale-in policies after `PreferNotRunning`, before the policies supplied through `WithScaleInPreferPolicy`:

```go
scaleInPolicies := []PreferPolicy[R]{
    PreferPriority[R](),
    PreferUnready[R](),
    PreferNotRunning[R](),
    PreferVolumeCapacityExceedsRequest[R](),
}
scaleInPolicies = append(scaleInPolicies, b.scaleInPreferPolicies...)
```

The selector evaluates the last policy first. With the existing group topology policy appended, precedence from highest to lowest is:

```text
Topology > VolumeCapacityExceedsRequest > NotRunning > Unready > Priority
```

Topology remains a preference with its existing fallback semantics. Within its selected candidates, excess capacity takes precedence over the existing default preferences, including the priority annotation. This ordering is intentional and must be covered by tests.

Only the scale-in selector receives the new policy. Update selection and cancellation of offlining retain their existing policies. The executor still chooses an updated or outdated candidate set before running the selector; the new preference does not select across those sets or alter availability limits and deletion hooks.

This is a best-effort preference over observed instance state. If the user reduces storage and replicas in the same update, some instance specs or conditions may not yet reflect the new request when scale-in starts. Scale-in proceeds using existing fallback behavior rather than waiting for this informational condition. To obtain the preference for the new request, reduce storage first and allow the instance specs and conditions to converge before reducing replicas.

### Implementation Areas

| Area | Planned change |
| --- | --- |
| `api/core/v1alpha1/common_types.go` | Condition and reason constants; document reduced volume requests |
| `pkg/volumes/sync.go`, `pkg/volumes/capacity.go` | Normalize existing PVC requests in the VAC path |
| `pkg/client`, if needed | Support normalization against the current object during apply |
| `pkg/apiutil/core/v1alpha1` | Shared condition construction and volume/PVC comparison helpers |
| `pkg/controllers/common` and instance controller builders | Compute and persist the condition across supported instances |
| `pkg/updater/selector.go`, `pkg/updater/builder.go` | Add and register the scale-in preference |
| Relevant generated files and manifests | Regenerate only as required by API comments or interface changes |

### Compatibility and Feature Gate

No new feature gate or user-configurable scale-in field is proposed. Support for reduced volume requests requires the existing VAC feature gate to be enabled. The informational condition and scale-in preference use shared instance state; legacy PVC synchronization is not extended.

Existing instance objects without the condition continue to use the previous scale-in preferences until reconciled. Groups without volumes receive no new preference. Excess capacity is permitted indefinitely; no migration or automatic replacement runs on upgrade.

Downgrading to an operator version without this behavior can restore attempts to apply smaller requests to existing PVCs. Before downgrading, users should restore requests sufficient for retained PVCs where necessary. The additive condition itself requires no data migration.

### Test Plan

**Unit tests**

- Quantity normalization: desired smaller, equal, and larger than request/capacity; missing capacity; equivalent units; expansion already in progress.
- VAC synchronization: new PVC uses the desired request; existing PVC never receives a smaller request; concurrent attribute changes still synchronize.
- Conflict handling: a concurrent request increase is preserved after retry.
- Condition aggregation: zero, one, and multiple volumes; partial excess capacity; missing PVC/capacity; known excess capacity plus an unknown volume; deterministic messages; recovery after increasing the request; current observed generation.
- PVC overlays: preserve overlay behavior while comparing capacity with the declared volume request.
- Preference: true/current condition, stale condition, false/unknown/missing conditions, multiple preferred instances, fallback, topology precedence, and ordering against existing default policies.
- Confirm the condition does not independently change Ready, Synced, or abnormal-instance classification.

**Controller integration tests**

- Decrease a group request and verify propagation to existing instance specs, retained PVC requests/capacity, condition persistence, and converged group status.
- Verify instance and Pod UIDs remain unchanged for a storage-only decrease.
- Verify condition persistence when PVC synchronization waits and when compute is suspended with retained PVCs.
- Exercise the VAC path and shared task wiring across supported instance kinds.
- Exercise API writes against an API server so fake-client acceptance cannot hide an invalid PVC update.
- Verify that simultaneous storage and replica changes do not block scale-in on missing or stale conditions.

**End-to-end scenario**

1. Create three instances with `100Gi` PVCs.
2. Change the group request to `50Gi` and wait for instance specs and conditions to converge.
3. Verify the three existing PVCs retain `100Gi`, their instances report `True`, and Pods remain running without restart.
4. Scale out to four instances and verify the new PVC requests `50Gi`.
5. With equal reported capacity and request on the new instance, verify its condition is `False`.
6. Scale in to three instances in a topology that allows either size to be selected; verify an instance with excess capacity is chosen and the normal offlining/deletion flow completes.
7. Verify PVC retention follows the existing policy.

## Risks and Mitigations

- **Storage size leaks into another write path:** protect VAC PVC writes at the apply boundary.
- **Capacity reporting and condition lag:** treat missing capacity as unknown and stale conditions as unpreferred. Do not make scale-in depend on condition convergence.
- **Over-allocation by the provisioner:** a freshly created volume can report excess capacity. This is consistent with the condition definition; no action is triggered solely by this state.
- **Persistent reconciliation or status churn:** separate user request from sync request, order messages deterministically, and update status only when condition content changes.
- **Changed scale-in choices:** preserve topology precedence and existing executor lifecycle rules; explicitly test the new ordering against readiness and manual priority.

## Drawbacks

- Storage savings are gradual and depend on later instance removal and the existing PVC retention policy.
- The condition and preference add PVC observation and status reconciliation work.
- The preference does not guarantee selection of the largest volume or a particular instance when topology, revision partitions, or stale observations limit the candidates.

## Alternatives

- **Keep old instance specs at the larger request:** loses the desired request on existing instances and prevents a direct spec-to-capacity condition.
- **Replace instances immediately after a request decrease:** introduces automatic data movement and disruption beyond changing requests for future instances.
- **Report a pending shrink or an error condition:** misrepresents a valid steady state and implies follow-up work that the operator does not schedule.
- **Compare only with PVC request:** can describe an incomplete expansion as already allocated capacity. Use capacity for reporting and consider both request and capacity when deciding whether a larger PVC request is needed.
- **Prefer excess capacity over topology:** may remove an instance from an undesirable topology. Preserve topology precedence.
- **Wait for all conditions before any scale-in:** makes an informational condition a prerequisite for replica reconciliation. Keep the preference best effort.
