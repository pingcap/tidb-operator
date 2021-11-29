# gracefully reschedule tikv pod

TiDB-Operator issue: [#4215](https://github.com/pingcap/tidb-operator/issues/4215)

## Summary

Support gracefully reschedule a tikv pod. The implantation is neccessarrily the combination of three operations:

1. Add a PD evict-leader-scheduler to transfer the leader.
2. When leader count drop down to zero, delete the pod to let it re-create and reschedule.
3. Remove the evict-leader-scheduler.

## Motivation

### Goals

- Provide a way for user to gracefully reschedule a tikv pod.
- The pod must be gracefully stop, to be specify, evict leaders before delete the pod and must work normal when recreate the pod.

### Non-Goals

- Provider a way to make tikv graceful stop when just recive a `SIGTERM` signal.

## Proposal

### User Stories

#### Story 1

suppose `tikv-0` is running at node `node0`,  there are many free resource at other nodes, I would like to make tikv-0 run at other node and shutdown `node0` to reduce the cost. I may apply the flowing operations:

1. run `kubectl cordon node0` to mark `node0` as `unschedulable`.
2. delete pod `tikv-0` to let it re-create and reschedule to other nodes, to mitigate the impact of unavailability of some region, I will do the flowing operations insted of delete the pod directly:
   1. add evict-leader-scheduler by `pd-ctl`.
   2. delete `tikv-0` pod when leader count drop down to 0.
   3. remove the evict-leader-scheduler by `pd-ctl`
3. drain `node-0` in the normal way as [safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/) and shotdown `node0`

### Risks and Mitigations

- PD/TiKV fails to evict all leader and we may just keep checking it.

## Design Details

Support user to add an annotation to tikv pod to trigger an graceful reschedule.

Annotation key `EvictLeaderAnnKey`: `tikv.tidb.pingcap.com/evict-leader`

the value controler the behavier when the leander count drop to zero, valid value is one of:

- `none`: doing nothing
- `delete-pod`: delete pod and remove the evict-leader scheduler from PD.



An EvictLeader status willbe added to the `TiKVStatus`:

```go
+type EvictLeaderStatus struct {
+       PodCreateTime metav1.Time `json:"podCreateTime,omitempty"`
+       Value         string      `json:"value,omitempty"`
+}

 // TiKVStatus is TiKV status
 type TiKVStatus struct {
        Synced          bool                            `json:"synced,omitempty"`
@@ -1139,6 +1151,7 @@ type TiKVStatus struct {
        TombstoneStores map[string]TiKVStore            `json:"tombstoneStores,omitempty"`
        FailureStores   map[string]TiKVFailureStore     `json:"failureStores,omitempty"`
        Image           string                          `json:"image,omitempty"`
+       EvictLeader     map[string]*EvictLeaderStatus   `json:"evictLeaderStatus,omitempty"`
 }
```



At controller side, the reconciler to handle pod will be like:

```go
func sync(pod *corev1.Pod, tc *v1alpha1.TidbCluster) (ctrl.Result, error) {
        value, ok := pod.Annotations[v1alpha1.EvictLeaderAnnKey]

        if ok {
                evictStatus := &v1alpha1.EvictLeaderStatus{
                        PodCreateTime: pod.CreationTimestamp,
                        Value:         value,
                }
                nowStatus := tc.Status.TiKV.EvictLeader[pod.Name]
                if nowStatus == nil || *nowStatus != evictStatus {
                        tc.Status.TiKV.EvictLeader[pod.Name] = evictStatus
                        // TODO update tc.Status to api-server
                }

                // TODO:
                // 1. add evict-leader scheduler if not added yet.

                if value == v1alpha1.EvictLeaderAnnValueDeletePod {
                        leaderCount := getLeaderCount(pod)
                        if leaderCount == 0 {
                                // TODO: delete the pod
                        } else {
                                // re-check leader count next time
                                return ctrl.Result{RequeueAfter: time.Second * 15}, nil
                        }
                }
        } else {
                evictStatus := tc.Status.TiKV.EvictLeader[pod.Name]
                if evictStatus != nil {
                        if evictStatus.Value == v1alpha1.EvictLeaderAnnValueDeletePod {
                                if IsPodReady(pod) {
                                        // TODO:
                                        // 1. delete evict-leader scheduler
                                        // 2. delete pod from tc.Status.TiKV.EvictLeader and update it to api-server
                                }
                        } else if evictStatus.Value == v1alpha1.EvictLeaderAnnValueNone {
                                // TODO:
                                // 1. delete evict-leader scheduler
                                // 2. delete pod from tc.Status.TiKV.EvictLeader and update it to api-server
                        }
                }
        }

        return ctrl.Result{}, nil
}
```

An example for Story 1 at step 2, user might add annotation with the flowing value for key `tikv.tidb.pingcap.com/reschedule`:

```
kubectl annotate pods <tikv-pod-name> tikv.tidb.pingcap.com/evict-leader=delete-pod
```

when user observe that pod is re-create and ready again, it can assume that `tikv-0` is already gracefully reschedule to other node and forward to step 3.

## Drawbacks

- The are still no way to stop tikv gracefully in normal way(graceful stop when receive a `SIGTERM` signal.TiKV issue [10296](https://github.com/tikv/tikv/issues/10296). So user may need do some special operations as this document describe when doing some operations that need to evict or re-create tikv pod like take down a node.

