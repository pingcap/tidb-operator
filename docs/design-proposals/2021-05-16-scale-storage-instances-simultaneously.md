# Scale in/out TiKV/TiFlash instances simultaneously

## Summary

This document presents a design to scale in/out tikv/tiflash instances simultaneously in single schedule round.

## Motivation

Multiple TiKV/TiFlash instances can be scaled simultaneously to speed up schedule.

### Goals

* Support scale multiple TiKV/TiFlash instances in one single schedule round.

### Non-Goals

* Scale pd or other component simultaneously.

## Proposal

### Scale in

* Add `scaleInParallelism` in `spec.tikv` and `spec.tiflash` to specify the max instances can be scaled-in in one single schedule.
* Default value of `scaleInParallelism` would be 1 when it's absent, for backward compatibility.
* Extend `scaleOne` in `scaler.go` to `scaleMulti`, function signature would like:
```javascript
// scaleOne calculates desired replicas and delete slots from actual/desired
// stateful sets by allowing only one pod to be deleted or created
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
* Call PD API to get stores info, which will be used during this operation round.
* For all ordinals waited to be scaled-in in this round:
  * Check if number of stores with `up` state (exclude already deleted store in this round) is more than desired replicas.
  * Call PD API to delete store until its state changes to `offline`.
  * When store become tombstone, add defer deleting annotation to the PVCs of the Pod to delete them.
  * If current store is tombstone and add PVCs defer deleted, mark corresponding ordinal as finished, otherwise failed.
* Call `setReplicasAndDeleteSlots` to delete pod:
  * If without asts enabled:
    * Since native StatefulSet will always scale in pod with the largest order so we should assure the ordinals from largest to smallest __strictly__ are finished.
    * Count the __continuous__ finished ordinal beginning from largest in A, recorded as c, then the final replicas will be ordinal replica - c.
  * If with asts enabled:
    * Since Advanced StatefulSet can scale pod with arbitrary ordinal so we can set replicas and deleteSlots finished in this schedule round.
    * Calculate replicas and deleteSlots from finished and failed ordinals.

### Scale out

* Add `scaleOutParallelism` in `spec.tikv` and `spec.tiflash` to specify the max instances can be scaled-in in one single schedule.
* Default value of `scaleOutParallelism` would be 1 when it's absent, for backward compatibility.
* Call `scaleMulti` to get oridnals to be scaled-out, recorded as A.
* For all oridnals waited to be scaed-out in this round:
  * Call `deleteDeferDeletingPVC` to clean all PVCs with deleted annotation, mark corresponding original as finished if succeed, otherwise failed.
* Call `setReplicasAndDeleteSlots` to create pod:
  * If without asts enabled:
    * Count the __continuous__ finished original beginning from smallest order in A, recorded as c, then the final replicas will be ordinal  + c.
  * If with asts enabled:
    * Calculate replicas and deleteSlots from finished and failed oridnals.

### Test Plan

* [Test Plan](https://docs.google.com/document/d/1XgreMvP6Sx7KrwMwVJn4ZldYWhs5s6oXj4Bcn9FvajI/edit).

## Drawbacks

* Scale in/out multiple instances at the same time may cause region replicas reschedule more frequent.
