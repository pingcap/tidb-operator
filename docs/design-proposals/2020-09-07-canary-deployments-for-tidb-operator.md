# Canary deployments for TiDB-Operator

Thanks @cvvz @DanielZhangQD

## Summary

This document presents a design to support canary deployments TiDB-Operator.

## Motivation

When upgrade TiDB-Operator, users may want to deploy multiple cluster-scoped TiDB-Operators in the same Kubernetes cluster.
And choose some specific TiDB clusters will be managed by the new version TiDB-Operator, the others are still managed by the old version. When the new version TiDB-Operator is fully tested, replace the old one.

### Goals

* Deploy multiple `tidb-operator` in one Kubernetes cluster
* Manage `tidbcluster` which be selected
* Don't manage `tidbcluster` which be excluded

### Non-Goals

* Upgrading `tidb-operator`
* Upgrading `tidbcluster`

## Proposal

Add `labelSelector`
When sync `tidbcluster` CRDs filter by `labelSelector`

### Design Details
1. Components that need canary deployments: 
- All components deployed by helm as long as `created=ture` need to have canary deployments capabilities.
2. Label `tidbcluster`:
- Label the version like `tidb-operator/managed=v1.1.4` on `tidbcluster` CRs.
3. Specify labelSelector to `tidb-operator`:
- Specify `spec.labelSelector` which be labeled `tidb-operator/managed=v1.1.4`
4. `tidbcluster`-controller:
- Tidb-controller will synchronize label to `tidbcluster` and statefulset, but not to Pod to avoid causing upgrade.
- When a `tidbcluster` event is monitored, judge whether to join the queue by the label.
5. advanced-statefulset-controller:
- When a statefulset event is monitored, then the statefulset's label is used to determine whether to join the queue.
- When listening to Pod events, get `tidbcluster` by ownerReferences, and then judge whether to join the queue by the `tidbcluster` label.
6. tidb-shceduler: 
- When the scheduler monitors the event of the Pod, it obtains the information in statefulset according to ownerReferences and determines whether it should schedule it by itself
7. webhook: 
- Different webhooks are configured with different APIService addresses, and other configurations are the same. 
- When the event of the Pod is monitored, the information in `tidbcluster` is obtained according to ownerReferences, and the judgment to  process or not.
8. apiserver:
- Currently aggregated apiserver is not enabled.
9. Helm: 
- The tidb-controller-manager and tidb-scheduler deployment name need a version suffix.
- The RBAC and other public resources deployed by helm does also need a version suffix, avoiding deleting public resources when helm delete `tidb-operator` release.
- User must set different .Values.clusterName or helm release name between two version of `tidb-operator` releases.

## Example
1. Label all current `tidbcluster` with `tidb-operator/managed=v1.1.3`
2. Update the old version `TiDB-Operator` `spec.labelSelector` filter `tidbcluster` which be labeled `tidb-operator/managed=v1.1.3`
3. Deploy a new version `TiDB-Operator` and specify `spec.labelSelector` which be labeled `tidb-operator/managed=v1.1.4`
4. Deploy `tidbcluster` with label `tidb-operator/managed=v1.1.4`
5. Full test new version `TiDB-Operator`
6. Label all old `tidbcluster` with `tidb-operator/managed=v1.1.4`
7. Offline the old version `TiDB-Operator`

## Test Case
1. When don't specify `spec.labelSelector` in `tidb-operator`, `tidb-operator` will manage all `tidbclusters`
2. When specify `spec.labelSelector` in `tidb-operator`, `tidb-operator` will manage `tidbcluster` that only be labeled to `labelSelector`
3. Deploy two `tidb-operator` with different .Values.clusterName or helm release name in one Kubernetes cluster.
4. Label `tidbcluster` and `tidb-operator` will synchronize label to statefulset
5. Helm delete one of `tidb-operator` release, the others `tidb-operator` will no effect.