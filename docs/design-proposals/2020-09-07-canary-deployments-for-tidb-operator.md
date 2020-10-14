# Canary Upgrade for TiDB-Operator

Thanks @cvvz @DanielZhangQD

## Summary

This document presents a design to support canary deployments TiDB-Operator.

## Motivation

When upgrading TiDB-Operator, users may want to deploy multiple cluster-scoped TiDB-Operators in the same Kubernetes cluster.
And choose some specific TiDB clusters to be managed by the new version TiDB Operator, the others are still managed by the old version. When the new version TiDB Operator is fully tested, the old version can be removed safely.

### Goals

* Deploy multiple clusterScoped `tidb-operator` in one Kubernetes cluster
* Different `tidb-operator` manage different `TidbCluster` with label selector

### Non-Goals

* Upgrading `tidb-operator`
* Upgrading `TidbCluster`

## Proposal

1. Add a flag to the operator like this `--selector` <label expression> where the label expression is key-value pairs of labels and values, the handling of `--selector` will keep consistent with `kubectl`. 
2. Add `spec.labelSelector` in values.yaml of `tidb-operator` helm chart.
3. TiDB Operator syncs the `TidbCluster` CRs filtered by `spec.labelSelector`.

### Design Details
1. Components that need a canary upgrade: 
- All components that can be deployed by the `tidb-operator` chart.
2. Label `TidbCluster`:
- Add version label to `TidbCluster` CRs, for example, `tidb.pingcap.com/managed-by=v1.1.4`, this label needs to be synced to the StatefulSets created for this CR.
3. Support specifying labelSelector for the `tidb-operator` chart.
4. tidb-controller-manager:
- tidb-controller-manager will synchronize labels of `TidbCluster` CR to StatefulSets, but not to Pod to avoid causing upgrade.
- List-watch the custom resources with the labelSelector
- List-watch all the Kubernetes native resources but filter them with the labelSelector before enqueueing
5. advanced-statefulset-controller:
- List-watch the asts with the labelSelector
- When receiving Pod events, get its ownerReferences, and then filter them with the labelSelector before enqueueing
6. tidb-shceduler: 
- When receiving Pod events, get its ownerReferences, and then filter them with the labelSelector before enqueueing
7. webhook: 
- Different webhooks are configured with different APIService addresses, and other configurations are the same. 
- When receiving Pod events, get its ownerReferences, and then filter them with the labelSelector before enqueueing
8. Helm: 
- The deployment name of tidb-controller-manager and tidb-scheduler needs a version suffix.
- The RBAC and other public resources deployed by helm also need a version suffix to avoid being deleted when run `helm delete` to remove the old version TiDB Operator.

## Example
1. Label all current `TidbCluster` with `tidb.pingcap.com/managed-by=v1.1.3`
2. Update the `spec.labelSelector` for old version TiDB Operator to manage the `TidbCluster` labeled with `tidb.pingcap.com/managed-by=v1.1.3`
3. Deploy a new version TiDB Operator and specify `spec.labelSelector` to `tidb.pingcap.com/managed-by=v1.1.4`
4. Deploy `TidbCluster` with label `tidb.pingcap.com/managed-by=v1.1.4`
5. Full test new version TiDB Operator
6. Label all the old `TidbCluster` with `tidb.pingcap.com/managed-by=v1.1.4`
7. Remove the old version TiDB Operator

## Test Case
1. When don't specify `spec.labelSelector` in `tidb-operator`, `tidb-operator` will manage all `TidbCluster`
2. When specify `spec.labelSelector` in `tidb-operator`, `tidb-operator` will manage `TidbCluster` that only labeled with the value of `labelSelector`
3. Deploy two `tidb-operator` with different helm release names in one Kubernetes cluster, check that tidb-controller-manager, tidb-scheduler, advanced statefulset controller and webhook work as expected.
4. Label `TidbCluster` and `tidb-operator` will synchronize labels to StatefulSets
5. Helm delete one of the `tidb-operator` releases, the other `tidb-operator` will work as expected.
