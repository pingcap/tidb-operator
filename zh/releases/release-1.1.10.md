---
title: TiDB Operator 1.1.10 Release Notes
---

# TiDB Operator 1.1.10 Release Notes

发布日期：2021 年 1 月 28 日

TiDB Operator 版本：1.1.10

## 兼容性改动

- 由于 [#3638](https://github.com/pingcap/tidb-operator/pull/3638) 的改动，TiDB Operator chart 中创建的 ClusterRoleBinding、ClusterRole、RoleBinding、Role 的 `apiVersion` 从 `rbac.authorization.k8s.io/v1beta1` 更改为 `rbac.authorization.k8s.io/v1`，此时通过 `helm upgrade` 升级 TiDB Operator 可能会报下面错误：

    ```
    Error: UPGRADE FAILED: rendered manifests contain a new resource that already exists. Unable to continue with update: existing resource conflict: namespace: , name: tidb-operator:tidb-controller-manager, existing_kind: rbac.authorization.k8s.io/v1, Kind=ClusterRole, new_kind: rbac.authorization.k8s.io/v1, Kind=ClusterRole
    ```

    详情可以参考 [helm/helm#7697](https://github.com/helm/helm/issues/7697)，此时需要通过 `helm uninstall` 删除 TiDB Operator 然后重新安装（删除 TiDB Operator 不会影响现有集群）。

## 滚动升级改动

- 由于 [#3684](https://github.com/pingcap/tidb-operator/pull/3684) 的改动，升级 TiDB Operator 会导致 TidbMonitor Pod 删除重建

## 新功能

- TiDB Operator 灰度升级 ([#3548](https://github.com/pingcap/tidb-operator/pull/3548), [@shonge](https://github.com/shonge), [#3554](https://github.com/pingcap/tidb-operator/pull/3554), [@cvvz](https://github.com/cvvz))
- TidbMonitor 支持 `remotewrite` ([#3679](https://github.com/pingcap/tidb-operator/pull/3679), [@mikechengwei](https://github.com/mikechengwei))
- 支持为 TiDB 集群各组件配置 init containers ([#3713](https://github.com/pingcap/tidb-operator/pull/3713), [@handlerww](https://github.com/handlerww))
- TiDB Lightning chart 支持 local backend ([#3644](https://github.com/pingcap/tidb-operator/pull/3644), [@csuzhangxc](https://github.com/csuzhangxc))

## 优化提升

- 支持为 TiDB slow log 自定义存储 ([#3731](https://github.com/pingcap/tidb-operator/pull/3731), [@BinChenn](https://github.com/BinChenn))
- 为 TidbMonitor 中的 scrape jobs 增加 `tidb_cluster` label 以支持多集群监控 ([#3750](https://github.com/pingcap/tidb-operator/pull/3750), [@mikechengwei](https://github.com/mikechengwei))
- TiDB Lightning chart 支持持久化 checkpoint ([#3653](https://github.com/pingcap/tidb-operator/pull/3653), [@csuzhangxc](https://github.com/csuzhangxc))
- 将 TidbMonitor 自定义告警规则的存储路径从 `tidb:${tidb_image_version}` 修改为 `tidb:${initializer_image_version}`，确保后续 TiDB 集群升级时不会导致 TidbMonitor Pod 重建 ([#3684](https://github.com/pingcap/tidb-operator/pull/3684), [@BinChenn](https://github.com/BinChenn))

## Bug 修复

- 修复在集群开启 TLS 的情况下，如果不配置 `spec.from` 或者 `spec.to`，使用 BR 备份或者恢复会失败的问题 ([#3707](https://github.com/pingcap/tidb-operator/pull/3707), [@BinChenn](https://github.com/BinChenn))
- 修复在开启 Advanced StatefulSet 并且为 PD、TiKV 设置了 `delete-slots` annotations 的情况下，序号大于 `replicas - 1` 的 Pod 在升级过程中不会进行迁移 leader 的问题 ([#3702](https://github.com/pingcap/tidb-operator/pull/3702), [@cvvz](https://github.com/cvvz))
- 修复备份或者恢复 Pod 被驱逐或者强制停止的情况下，Backup 或者 Restore 状态没有正常更新为 `Failed` 的问题 ([#3696](https://github.com/pingcap/tidb-operator/pull/3696), [@csuzhangxc](https://github.com/csuzhangxc))
- 修复如果 TiKV 集群由于配置错误无法启动，修改 `TidbCluster` CR 配置后，仍然无法启动的问题 ([#3694](https://github.com/pingcap/tidb-operator/pull/3694), [@cvvz](https://github.com/cvvz))
