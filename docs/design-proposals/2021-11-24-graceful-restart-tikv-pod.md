# Gracefully restart TiKV pod

TiDB Operator issue: [#4215](https://github.com/pingcap/tidb-operator/issues/4215)

## Summary

Support gracefully restart a TiKV pod. The implementation is necessarily the combination of three operations:

1. Add a evict-leader-scheduler for the target TiKV store in PD to evict the region leaders.
2. When the region leader count drops to zero, delete the pod to let it restart.
3. Remove the evict-leader-scheduler.

## Motivation

### Goals

- Provide a way for a user to gracefully restart a TiKV pod.
- The pod must be gracefully stopped, to be specific, evict region leaders before deleting the pod, and must work normally after recreating the pod.

### Non-Goals

- Provider a way to make TiKV graceful shutting down when just receiving a `SIGTERM` signal.

## Proposal

### User Stories

#### Story 1

Suppose `TiKV-0` is running on node `node0`,  there are many free resources at other nodes. I would like to make `TiKV-0` run at other nodes and shutdown `node0` to reduce the cost. I may apply the flowing operations:

1. Run `kubectl cordon node0` to mark `node0` as `unschedulable`
2. Delete pod `TiKV-0` to let it re-create and reschedule to other nodes, to mitigate the impact of unavailability of some regions, I will do the flowing operations instead of deleting the pod directly:
   1. Add evict-leader-scheduler via `pd-ctl`.
   2. Delete `TiKV-0` pod when the leader count drops to 0.
   3. Remove the evict-leader-scheduler after the Pod is recreated.
3. Drain `node-0` in the normal way as [safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/) and shutdown `node0`.

### Risks and Mitigations

- PD/TiKV fails to evict all leaders and we may have to keep checking it.

## Design Details

Support user to add an annotation to TiKV pod to trigger a graceful restart.

Annotation key: `tidb.pingcap.com/evict-leader`

The controller will do the following operations:

1. Add evict-leader-scheduler for the TiKV store.
2. Delete the pod to make it recreate when the leader count is 0.
3. Remove the evict-leader-scheduler when the new pod becomes ready.

The `Value` of annotation controls the behavior when the leader count drops to zero, the valid value is one of:

- `none`: doing nothing.
- `delete-pod`: delete pod and remove the evict-leader scheduler from PD.

An EvictLeader status will be added to the `TiKVStatus` in `TidbCluster`:

```go
+type EvictLeaderStatus struct {
+       PodCreationTime metav1.Time `json:"podCreationTime,omitempty"`
+       Value         string      `json:"value,omitempty"`
+}

 // TiKVStatus is TiKV status
 type TiKVStatus struct {
        Synced          bool                            `json:"synced,omitempty"`
@@ -1139,6 +1151,7 @@ type TiKVStatus struct {
        TombstoneStores map[string]TiKVStore            `json:"tombstoneStores,omitempty"`
        FailureStores   map[string]TiKVFailureStore     `json:"failureStores,omitempty"`
        Image           string                          `json:"image,omitempty"`
+       EvictLeaderStatus     map[string]*EvictLeaderStatus   `json:"evictLeaderStatus,omitempty"`
 }
```

A new controller is introduced, and the reconciler to handle pod would be like:

```go
func sync(pod *corev1.Pod, tc *v1alpha1.TidbCluster) (ctrl.Result, error) {
    value, ok := pod.Annotations["tidb.pingcap.com/evict-leader"]

    if ok {
            evictStatus := &v1alpha1.EvictLeaderStatus{
                    PodCreationTime: pod.CreationTimestamp,
                    Value:         value,
            }
            nowStatus := tc.Status.TiKV.EvictLeader[pod.Name]
            if nowStatus == nil || *nowStatus != *evictStatus {
                    tc.Status.TiKV.EvictLeaderStatus[pod.Name] = evictStatus
                    // TODO update tc.Status to api-server
            }

            // TODO:
            // 1. add evict-leader scheduler if not added yet.

            if value == "delete-pod" {
                    leaderCount := getLeaderCount(pod)
                    if leaderCount == 0 {
                            // TODO: delete the pod
                    } else {
                            // re-check leader count next time
                            return ctrl.Result{RequeueAfter: time.Second * 15}, nil
                    }
            }
    } else {
            evictStatus := tc.Status.TiKV.EvictLeaderStatus[pod.Name]
            if evictStatus != nil {
                    if evictStatus.Value == "delete-pod" {
                            if IsPodReady(pod) {
                                    // TODO:
                                    // 1. delete evict-leader scheduler
                                    // 2. delete Pod from tc.Status.TiKV.EvictLeaderStatus and update it to api-server
                            }
                    } else if evictStatus.Value == "none" {
                            // TODO:
                            // 1. delete evict-leader scheduler
                            // 2. delete Pod from tc.Status.TiKV.EvictLeader and update it to api-server
                    }
            }
    }

    return ctrl.Result{}, nil
}
```

An example for Story 1 at step 2, a user might add annotation with the key `tidb.pingcap.com/restart`:

```
kubectl annotate pods <TiKV-pod-name> tidb.pingcap.com/evict-leader="delete-pod"
```

when the user observes that the pod is recreated and ready again, it can assume that `TiKV-0` is already gracefully restarted and forward to step 3.

## Drawbacks

- There is still no way to stop TiKV gracefully in a normal way (when receiving a `SIGTERM` signal. TiKV issue [10296](https://github.com/TiKV/TiKV/issues/10296). So the user may need to do some special operations as this document describes when doing some operations that need to evict or restart a TiKV pod, like shutting down a node.
