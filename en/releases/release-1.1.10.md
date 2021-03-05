---
title: TiDB Operator 1.1.10 Release Notes
---

# TiDB Operator 1.1.10 Release Notes

Release date: January 28, 2021

TiDB Operator version: 1.1.10

## Compatibility Changes

- Due to the changes of [#3638](https://github.com/pingcap/tidb-operator/pull/3638), the `apiVersion` of ClusterRoleBinding, ClusterRole, RoleBinding, and Role created in the TiDB Operator chart is changed from `rbac.authorization .k8s.io/v1beta1` to `rbac.authorization.k8s.io/v1`. In this case, upgrading TiDB Operator through `helm upgrade` may report the following error:

     ```
     Error: UPGRADE FAILED: rendered manifests contain a new resource that already exists. Unable to continue with update: existing resource conflict: namespace:, name: tidb-operator:tidb-controller-manager, existing_kind: rbac.authorization.k8s.io/ v1, Kind=ClusterRole, new_kind: rbac.authorization.k8s.io/v1, Kind=ClusterRole
     ```

     For details, refer to [helm/helm#7697](https://github.com/helm/helm/issues/7697). In this case, you need to delete TiDB Operator through `helm uninstall` and then reinstall it (deleting TiDB Operator will not affect the current TiDB clusters).

## Rolling Update Changes

- Upgrading TiDB Operator will cause the recreation of the TidbMonitor Pod due to [#3684](https://github.com/pingcap/tidb-operator/pull/3684)

## New Features

- Support canary upgrade of TiDB Operator ([#3548](https://github.com/pingcap/tidb-operator/pull/3548), [@shonge](https://github.com/shonge), [#3554](https://github.com/pingcap/tidb-operator/pull/3554), [@cvvz](https://github.com/cvvz))
- TidbMonitor supports `remotewrite` configuration ([#3679](https://github.com/pingcap/tidb-operator/pull/3679), [@mikechengwei](https://github.com/mikechengwei))
- Support configuring init containers for components in the TiDB cluster ([#3713](https://github.com/pingcap/tidb-operator/pull/3713), [@handlerww](https://github.com/handlerww))
- Add local backend support to the TiDB Lightning chart ([#3644](https://github.com/pingcap/tidb-operator/pull/3644), [@csuzhangxc](https://github.com/csuzhangxc))

## Improvements

- Support customizing the storage config for TiDB slow log ([#3731](https://github.com/pingcap/tidb-operator/pull/3731), [@BinChenn](https://github.com/BinChenn))
- Add the `tidb_cluster` label for the scrape jobs in TidbMonitor to support monitoring multiple clusters ([#3750](https://github.com/pingcap/tidb-operator/pull/3750), [@mikechengwei](https://github.com/mikechengwei))
- Supports persisting checkpoint for the TiDB Lightning helm chart ([#3653](https://github.com/pingcap/tidb-operator/pull/3653), [@csuzhangxc](https://github.com/csuzhangxc))
- Change the directory of the customized alert rules in TidbMonitor from `tidb:${tidb_image_version}` to `tidb:${initializer_image_version}` so that when the TiDB cluster is upgraded afterwards, the TidbMonitor Pod will not be recreated ([#3684](https://github.com/pingcap/tidb-operator/pull/3684), [@BinChenn](https://github.com/BinChenn))

## Bug Fixes

- Fix the issue that when TLS is enabled for the TiDB cluster, if `spec.from` or `spec.to` is not configured, backup and restore jobs with BR might fail ([#3707](https://github.com/pingcap/tidb-operator/pull/3707), [@BinChenn](https://github.com/BinChenn))
- Fix the bug that if the advanced StatefulSet is enabled and `delete-slots` annotations are added for PD or TiKV, the Pods whose ordinal is bigger than `replicas - 1` will be terminated directly without any pre-delete operations such as evicting leaders ([#3702](https://github.com/pingcap/tidb-operator/pull/3702), [@cvvz](https://github.com/cvvz))
- Fix the issue that after the Pod has been evicted or killed, the status of backup or restore is not updated to `Failed` ([#3696](https://github.com/pingcap/tidb-operator/pull/3696), [@csuzhangxc](https://github.com/csuzhangxc))
- Fix the issue that when the TiKV cluster is not bootstrapped due to incorrect configuration, the TiKV component could not be recovered by editing `TidbCluster` CR ([#3694](https://github.com/pingcap/tidb-operator/pull/3694), [@cvvz](https://github.com/cvvz))
