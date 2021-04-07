---
title: TiDB Operator 1.2.0-beta.1 Release Notes
---

# TiDB Operator 1.2.0-beta.1 Release Notes

Release date: April 7, 2021

TiDB Operator version: 1.2.0-beta.1

## Compatibility Changes

- Due to the changes of [#3638](https://github.com/pingcap/tidb-operator/pull/3638), the `apiVersion` of ClusterRoleBinding, ClusterRole, RoleBinding, and Role created in the TiDB Operator chart is changed from `rbac.authorization .k8s.io/v1beta1` to `rbac.authorization.k8s.io/v1`. In this case, upgrading TiDB Operator through `helm upgrade` may report the following error:

     ```
     Error: UPGRADE FAILED: rendered manifests contain a new resource that already exists. Unable to continue with update: existing resource conflict: namespace:, name: tidb-operator:tidb-controller-manager, existing_kind: rbac.authorization.k8s.io/ v1, Kind=ClusterRole, new_kind: rbac.authorization.k8s.io/v1, Kind=ClusterRole
     ```

     For details, refer to [helm/helm#7697](https://github.com/helm/helm/issues/7697). In this case, you need to delete TiDB Operator through `helm uninstall` and then reinstall it (deleting TiDB Operator will not affect the current TiDB clusters).

## Rolling Update Changes

- Upgrading TiDB Operator will cause the recreation of the TidbMonitor Pod due to [#3785](https://github.com/pingcap/tidb-operator/pull/3785)

## New Features

- Support setting customized environment variables for backup and restore job containers ([#3833](https://github.com/pingcap/tidb-operator/pull/3833), [@dragonly](https://github.com/dragonly))
- Add additional volume and volumeMount configurations to TidbMonitor([#3855](https://github.com/pingcap/tidb-operator/pull/3855), [@mikechengwei](https://github.com/mikechengwei))
- Support affinity and tolerations in backup/restore CR ([#3835](https://github.com/pingcap/tidb-operator/pull/3835), [@dragonly](https://github.com/dragonly))
- The resources in the tidb-operator chart use the new service account when `appendReleaseSuffix` is set to `true` ([#3819](https://github.com/pingcap/tidb-operator/pull/3819), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support configuring durations for leader election ([#3794](https://github.com/pingcap/tidb-operator/pull/3794), [@july2993](https://github.com/july2993))
- Add the `tidb_cluster` label for the scrape jobs in TidbMonitor to support monitoring multiple clusters ([#3750](https://github.com/pingcap/tidb-operator/pull/3750), [@mikechengwei](https://github.com/mikechengwei))
- Support setting customized store labels according to the node labels ([#3784](https://github.com/pingcap/tidb-operator/pull/3784), [@L3T](https://github.com/L3T))
- Support customizing the storage config for TiDB slow log ([#3731](https://github.com/pingcap/tidb-operator/pull/3731), [@BinChenn](https://github.com/BinChenn))
- TidbMonitor supports `remotewrite` configuration ([#3679](https://github.com/pingcap/tidb-operator/pull/3679), [@mikechengwei](https://github.com/mikechengwei))
- Support configuring init containers for components in the TiDB cluster ([#3713](https://github.com/pingcap/tidb-operator/pull/3713), [@handlerww](https://github.com/handlerww))

## Improvements

- Add retry for DNS lookup failure exception in TiDBInitializer ([#3884](https://github.com/pingcap/tidb-operator/pull/3884), [@handlerww](https://github.com/handlerww))
- Optimize thanos example yaml files ([#3726](https://github.com/pingcap/tidb-operator/pull/3726), [@mikechengwei](https://github.com/mikechengwei))
- Delete the evict leader scheduler after TiKV Pod is recreated during the rolling update ([#3724](https://github.com/pingcap/tidb-operator/pull/3724), [@handlerww](https://github.com/handlerww))
- Support multiple PVCs for PD during scaling and failover ([#3820](https://github.com/pingcap/tidb-operator/pull/3820), [@dragonly](https://github.com/dragonly))
- Support multiple PVCs for TiKV during scaling ([#3816](https://github.com/pingcap/tidb-operator/pull/3816), [@dragonly](https://github.com/dragonly))
- Support PVC resizing for TiDB ([#3891](https://github.com/pingcap/tidb-operator/pull/3891), [@dragonly](https://github.com/dragonly))
- Add TiFlash rolling upgrade logic to avoid all TiFlash stores being unavailable at the same time during the upgrade ([#3789](https://github.com/pingcap/tidb-operator/pull/3789), [@handlerww](https://github.com/handlerww))
- Retrieve the region leader count from TiKV Pod directly instead of from PD to get the accurate count ([#3801](https://github.com/pingcap/tidb-operator/pull/3801), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Print RocksDB and Raft logs to stdout to support collecting and querying the logs in Grafana ([#3768](https://github.com/pingcap/tidb-operator/pull/3768), [@baurine](https://github.com/baurine))

## Bug Fixes

- Fix the issue that PVCs will be set to incorrect size if multiple PVCs are configured for PD/TiKV ([#3858](https://github.com/pingcap/tidb-operator/pull/3858), [@dragonly](https://github.com/dragonly))
- Fix the panic issue when `.spec.tidb` is not set in the TidbCluster CR with TLS enabled ([#3852](https://github.com/pingcap/tidb-operator/pull/3852), [@dragonly](https://github.com/dragonly))
- Fix the issue that some unrecognized environment variables are included in the external labels of the TidbMonitor ([#3785](https://github.com/pingcap/tidb-operator/pull/3785), [@mikechengwei](https://github.com/mikechengwei))
- Fix the issue that after the Pod has been evicted or killed, the status of backup or restore is not updated to `Failed` ([#3696](https://github.com/pingcap/tidb-operator/pull/3696), [@csuzhangxc](https://github.com/csuzhangxc))
- Fix the bug that if the advanced StatefulSet is enabled and `delete-slots` annotations are added for PD or TiKV, the Pods whose ordinal is bigger than `replicas - 1` will be terminated directly without any pre-delete operations such as evicting leaders ([#3702](https://github.com/pingcap/tidb-operator/pull/3702), [@cvvz](https://github.com/cvvz))
- Fix the issue that when TLS is enabled for the TiDB cluster, if `spec.from` or `spec.to` is not configured, backup and restore jobs with BR might fail ([#3707](https://github.com/pingcap/tidb-operator/pull/3707), [@BinChenn](https://github.com/BinChenn))
- Fix the issue that when the TiKV cluster is not bootstrapped due to incorrect configuration, the TiKV component could not be recovered by editing `TidbCluster` CR ([#3694](https://github.com/pingcap/tidb-operator/pull/3694), [@cvvz](https://github.com/cvvz))
