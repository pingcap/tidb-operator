---
title: TiDB Operator 1.2.0-beta.1 Release Notes
---

# TiDB Operator 1.2.0-beta.1 Release Notes

发布日期：2021 年 4 月 7 日

TiDB Operator 版本：1.2.0-beta.1

## 兼容性改动

- 由于 [#3638](https://github.com/pingcap/tidb-operator/pull/3638) 的改动，TiDB Operator chart 中创建的 ClusterRoleBinding、ClusterRole、RoleBinding、Role 的 `apiVersion` 从 `rbac.authorization.k8s.io/v1beta1` 更改为 `rbac.authorization.k8s.io/v1`，此时通过 `helm upgrade` 升级 TiDB Operator 可能会报下面错误：

    ```
    Error: UPGRADE FAILED: rendered manifests contain a new resource that already exists. Unable to continue with update: existing resource conflict: namespace: , name: tidb-operator:tidb-controller-manager, existing_kind: rbac.authorization.k8s.io/v1, Kind=ClusterRole, new_kind: rbac.authorization.k8s.io/v1, Kind=ClusterRole
    ```

    详情可以参考 [helm/helm#7697](https://github.com/helm/helm/issues/7697)，此时需要通过 `helm uninstall` 删除 TiDB Operator 然后重新安装（删除 TiDB Operator 不会影响现有集群）。

## 滚动升级改动

- 由于 [#3785](https://github.com/pingcap/tidb-operator/pull/3785) 的改动，升级 TiDB Operator 会导致 TidbMonitor Pod 删除重建。

## 新功能

- 支持为备份和恢复 Job 设置自定义环境变量 ([#3833](https://github.com/pingcap/tidb-operator/pull/3833)，[@dragonly](https://github.com/dragonly))
- 支持为 TidbMonitor 配置额外的 volume 和 volumeMount ([#3855](https://github.com/pingcap/tidb-operator/pull/3855)，[@mikechengwei](https://github.com/mikechengwei))
- 支持备份恢复 CR 设置 affinity 和 tolerations ([#3835](https://github.com/pingcap/tidb-operator/pull/3835)，[@dragonly](https://github.com/dragonly))
- 设置 `appendReleaseSuffix` 为 `true` 时，支持 tidb-operator chart 使用新的 service account ([#3819](https://github.com/pingcap/tidb-operator/pull/3819)，[@DanielZhangQD](https://github.com/DanielZhangQD))
- 优化 LeaderElection，支持用户配置 LeaderElection 时间 ([#3794](https://github.com/pingcap/tidb-operator/pull/3794)， [@july2993](https://github.com/july2993))
- 为 TidbMonitor 中的 scrape jobs 增加 `tidb_cluster` label 以支持多集群监控 ([#3750](https://github.com/pingcap/tidb-operator/pull/3750)， [@mikechengwei](https://github.com/mikechengwei))
- 支持设置自定义 Store 标签并根据 Node 标签获取值  ([#3784](https://github.com/pingcap/tidb-operator/pull/3784)， [@L3T](https://github.com/L3T))
- TidbMonitor 支持 `remotewrite` ([#3679](https://github.com/pingcap/tidb-operator/pull/3679)， [@mikechengwei](https://github.com/mikechengwei))
- 支持为 TiDB 集群各组件配置 init containers ([#3713](https://github.com/pingcap/tidb-operator/pull/3713)， [@handlerww](https://github.com/handlerww))
- 支持为 TiDB slow log 自定义存储 ([#3731](https://github.com/pingcap/tidb-operator/pull/3731)， [@BinChenn](https://github.com/BinChenn))

## 优化提升

- TiDBInitializer 中增加重试机制，解决 DNS 查询异常处理问题 ([#3884](https://github.com/pingcap/tidb-operator/pull/3884)，[@handlerww](https://github.com/handlerww))
- 优化 Thanos 的 example yaml ([#3726](https://github.com/pingcap/tidb-operator/pull/3726)，[@mikechengwei](https://github.com/mikechengwei))
- 滚动更新过程中，等待 TiKV Pod 升级完成之后再删除 evict leader scheduler  ([#3724](https://github.com/pingcap/tidb-operator/pull/3724)，[@handlerww](https://github.com/handlerww))
- 在 PD 的扩缩容和容灾过程中增加多 PVC 支持 ([#3820](https://github.com/pingcap/tidb-operator/pull/3820)，[@dragonly](https://github.com/dragonly))
- 在 TiKV 的扩缩容过程中增加多 PVC 支持([#3816](https://github.com/pingcap/tidb-operator/pull/3816)，[@dragonly](https://github.com/dragonly))
- 支持调整 TiDB PVC 容量 ([#3891](https://github.com/pingcap/tidb-operator/pull/3891)，[@dragonly](https://github.com/dragonly))
- 添加 TiFlash 滚动更新机制避免升级期间所有 TiFlash Store 同时不可用 ([#3789](https://github.com/pingcap/tidb-operator/pull/3789)， [@handlerww](https://github.com/handlerww))
- 由从 PD 获取 region leader 数量改为从 TiKV 直接获取以确保拿到准确的 region leader 数量 ([#3801](https://github.com/pingcap/tidb-operator/pull/3801)， [@DanielZhangQD](https://github.com/DanielZhangQD))
- 打印 RocksDB 和 Raft 日志到 stdout，以支持这些日志的收集和查询 ([#3768](https://github.com/pingcap/tidb-operator/pull/3768)， [@baurine](https://github.com/baurine))

## Bug 修复

- 修复 PD/TiKV 挂载多 PVC 时容量设置错误的问题 ([#3858](https://github.com/pingcap/tidb-operator/pull/3858)，[@dragonly](https://github.com/dragonly))
- 修复创建 `.spec.tidb` 为空并开启 TLS 的 TidbCluster 导致 tidb-controller-manager panic 的问题 ([#3852](https://github.com/pingcap/tidb-operator/pull/3852)，[@dragonly](https://github.com/dragonly))
- 修复 TidbMonitor 外部标签包含一些无法识别的环境变量的问题 ([#3785](https://github.com/pingcap/tidb-operator/pull/3785)，[@mikechengwei](https://github.com/mikechengwei))
- 修复在集群开启 TLS 的情况下，如果不配置 `spec.from` 或者 `spec.to`，使用 BR 备份或者恢复会失败的问题 ([#3707](https://github.com/pingcap/tidb-operator/pull/3707)， [@BinChenn](https://github.com/BinChenn))
- 修复在开启 Advanced StatefulSet 并且为 PD、TiKV 设置了 `delete-slots` annotations 的情况下，序号大于 `replicas - 1` 的 Pod 在升级过程中不会进行迁移 leader 的问题 ([#3702](https://github.com/pingcap/tidb-operator/pull/3702)， [@cvvz](https://github.com/cvvz))
- 修复备份或者恢复 Pod 被驱逐或者强制停止的情况下，Backup 或者 Restore 状态没有正常更新为 `Failed` 的问题 ([#3696](https://github.com/pingcap/tidb-operator/pull/3696)， [@csuzhangxc](https://github.com/csuzhangxc))
- 修复如果 TiKV 集群由于配置错误无法启动，修改 `TidbCluster` CR 配置后，仍然无法启动的问题 ([#3694](https://github.com/pingcap/tidb-operator/pull/3694)， [@cvvz](https://github.com/cvvz))
