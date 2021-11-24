# gracefully reschedule tikv pod

TiDB-Operator issue: [#4215](https://github.com/pingcap/tidb-operator/issues/4215)

## Summary

Support gracefully reschedule a tikv pod. The implemtation is necessarrily the combination of three operations:

1. Add a PD evict-leader-scheduler to transfer the leader.
2. When leader count drop down to zero, delete the pod to let it re-create and reschedule.
3. Remove the evict-leader-scheduler.

## Motivation

### Goals

- Provide a way for user to gracefully reschedule a tikv pod.
- The pod must be gracefully stop, to be specify, evict leaders before delete the pod and must work nomal when recreate the pod.

### Non-Goals

- Provider a way to make tikv graceful stop when just recive a `SIGTERM` signal.

## Proposal

### User Stories

#### Story 1

suppose `tikv-0` is running at node `node0`,  there are many free resource at other nodes, I would like to make tikv-0 run at other node and shutdown `node0` to reduce the cost. I may apply the flowing operations:

1. run `kubectl cordon node0` to mark `node0`as unschedulable.
2. delete pod `tikv-0` to let it re-create and reschedule to other nodes.
3. draine `node-0` in the nomal way as [safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/) and shotdown `node0`

### Risks and Mitigations

- PD/TiKV fails to evict all leader and we may just keep checking it.

## Design Details

Support user to add an annotation to CRD `TidbCluster` to trigger an gracefule reschedule.

annotation key: `tikv.tidb.pingcap.com/reschedule`

annotation value is JSON in the flowing struct:

```go
{
  // the flowing field set by user.
  NodeName *string
  Slots []int 
  
  // the flowing filed set by controller.
  RunningSlot *RunningSlot
  FinishedSlots []int
  Status string // Running|Finished
  Message string // optial message of the current status for human readible 
}

RunningSlot struct {
  Slot int
  Status string // Evicting|ReCreating
  StartTimestamp metav1.Time // etc. 2021-11-23T13:09:32Z
}
```

One of `NodeName` and `Slots` is required, user is able to specify all pods in a node or specifiy the pods directly.

At controller side, when ever observe this annotation is set, it will do the flowing step to handle an un finished action:

```go
if RunningSlot is set {
   switch RunningSlotStatus {
     case "Evicting"
       // 1. make sure evict-scheduler is added
       // 2. update RunningSlot.Status as ReCreating if leader-count is 0
     case "ReCreating"
     if the creationTimestamp of pod is less than RunningSlot.startTimestamp {
        delete the pod.
     }
     if the pod is already running normal {
        append RunningSlot.Slot in RunningSlot and set RunningSlot as nil
     }
   }
} else {
  // Find a slot to hanle by comaring `FinishedSlots` and `Slots` or slots belong to the specify NodeName.
  // set RunningSlot or mark Status as Finished
}


```

An example for Story 1 step 2, user might add annotation with the flowing value for key `tikv.tidb.pingcap.com/reschedule`:

```
{
	nodeName: "node0",
}
```

when user observe that statue is `Finished`, it can assume that `tikv-0` is already gracefully reschedule to other node and forward to step 3.

## Drawbacks

- The are still no way to stop tikv gracefully in nomal way(graceful stop when receive a `SIGTERM` signal.TiKV issue [10296](https://github.com/tikv/tikv/issues/10296). So user may need do some special operaion as this document describe when doing some operations that need to evict ot re-create tikv pod like take down a node.

