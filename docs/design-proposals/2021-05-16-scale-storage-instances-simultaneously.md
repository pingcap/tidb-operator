# Scale in/out multiple TiKV/TiFlash instances simultaneously

## Summary

This document presents a design to scale in/out multiple TiKV/TiFlash instances simultaneously in one sync loop.

## Motivation

Multiple TiKV/TiFlash instances can be scaled simultaneously to speed up the schedule.

### Goals

* Support scale multiple TiKV/TiFlash instances in one sync loop.

### Non-Goals

* Scale PD or other components simultaneously.

## Proposal

### Scale in

* Add `scalePolicy` in `spec.tikv` and `spec.tiflash` to define scale related arguments.
* Add `scaleInParallelism` in `scalePolicy` to specify the max instances can be scaled-in in one sync loop.
* Default value of `scaleInParallelism` would be 1 when it's not set, for backward compatibility.
* Extend `scaleOne` in `scaler.go` to `scaleMulti`, function signature would be like:
```go
// scaleMulti calculates desired replicas and delete slots from actual/desired
// StatefulSets by allowing multiple pods to be deleted or created
// it returns following values:
// - scaling:
//   - 0: no scaling required
//   - 1: scaling out
//   - -1: scaling in
// - ordinals: pod ordinals to create or delete
// - replicas/deleteSlots: desired replicas and deleteSlots by allowing no more than maxCount pods to be deleted or created
func scaleMulti(actual *apps.StatefulSet, desired *apps.StatefulSet, maxCount int) (scaling int, ordinals []int32, replicas int32, deleteSlots sets.Int32)
`````````
* Call `scaleMulti` to get ordinals to be scaled-in, recorded as A.
* Call PD API to get store info, which will be used during this sync loop.
* For all ordinals to be scaled-in in this loop:
  * Check if the number of stores with `up` state (exclude already deleted store in this round) is more than `Replication.MaxReplicas` in PD config.
  * Call PD API to delete store until its state changes to `offline`.
  * When store becomes tombstone, add defer deleting annotation to the PVCs of the corresponding pod to be deleted.
  * If the current store is `tombstone` and the defer deleting annotation has been added to the PVCs, mark the corresponding ordinal as finished, otherwise ongoing.
* Call `setReplicasAndDeleteSlotsByFinished` to delete pod:
  * Since native StatefulSet will always scale in Pod with the largest order so we should assure the ordinals from largest to smallest __strictly__ are finished.
  * Count the __continuous__ finished ordinal beginning from largest in A, recorded as c, then the final replicas will be `replicas - c`.

### Scale out

* Add `scaleOutParallelism` in `scalePolicy` to specify the max instances can be scaled-in in one sync loop.
* Default value of `scaleOutParallelism` would be 1 when it's not set, for backward compatibility.
* Call `scaleMulti` to get ordinals to be scaled-out, recorded as A.
* For all ordinals to be scaled-out in this loop:
  * Call `deleteDeferDeletingPVC` to clean all PVCs with the defer deleting annotation, mark the corresponding original as finished if succeed, otherwise ongoing.
* Call `setReplicasAndDeleteSlotsByFinished` to create pod:
  * Count the __continuous__ finished original beginning from smallest order in A, recorded as c, then the final replicas will be `replicas  + c`.

### Test Plan

* [Test Plan](https://docs.google.com/document/d/1XgreMvP6Sx7KrwMwVJn4ZldYWhs5s6oXj4Bcn9FvajI/edit).

## Drawbacks

* Scale in/out multiple instances simultaneously will cause more region replicas to be rescheduled at the same time.
