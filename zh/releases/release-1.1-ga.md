---
title: TiDB Operator 1.1 GA Release Notes
---

# TiDB Operator 1.1 GA Release Notes

发布日期：2020 年 5 月 28 日

TiDB Operator 版本：1.1.0

## 从 v1.0.x 升级

对于 v1.0.x 的用户，请参考[升级 TiDB Operator](https://docs.pingcap.com/zh/tidb-in-kubernetes/stable/upgrade-tidb-operator) 来升级集群中的 TiDB Operator。注意在升级之前应该阅读发布说明（特别是重大变更和需要操作的项目）。

## v1.0.0 后的重大变更

* 将 TiDB Pod 的 `readiness` 探针从 `HTTPGet` 更改为 `TCPSocket` 4000 端口。这将触发 `tidb-server` 组件滚动升级。你可以在升级 TiDB Operator 之前将 `spec.paused` 设置为 `true` 以避免滚动升级，并在准备升级 `tidb-server` 时将其重新设置为 `false` ([#2139](https://github.com/pingcap/tidb-operator/pull/2139), [@weekface](https://github.com/weekface))
* 为 `tidb-server` 配置了 `--advertise-address`，这将触发 `tidb-server` 滚动升级。你可以在升级 TiDB Operator 之前将 `spec.paused` 设置为 `true` 以避免滚动升级，并在准备升级 `tidb-server` 时将其设置为 `false` ([#2076](https://github.com/pingcap/tidb-operator/pull/2076), [@cofyc](https://github.com/cofyc))
* `--default-storage-class-name` 和 `--default-backup-storage-class-name` 标记被废弃，现在存储类型默认为 Kubernetes 默认存储类型。如果你已经设置的默认存储类型不是 Kubernetes 的默认存储类型，请在 TiDB 集群的 Helm 或 YAML 文件中显式设置它们 ([#1581](https://github.com/pingcap/tidb-operator/pull/1581), [@cofyc](https://github.com/cofyc))
* 为所有 Helm chart 添加时区选项 ([#1122](https://github.com/pingcap/tidb-operator/pull/1122), [@weekface](https://github.com/weekface))

    对于 `tidb-cluster` chart，我们已经支持 `timezone` 选项（默认为 UTC）。如果用户没有将其改为其他值 (例如 `Asia/Shanghai`），则不会重新创建任何 Pod。

    如果用户将其改为其他值 (例如 `Asia/Shanghai`），则将重新创建所有相关的 Pod（添加一个 `TZ` 环境变量），即滚动更新。

    相关的 Pod 包括 `pump`、`drainer`、`discovery`、`monitor`、`scheduled backup`、`tidb-initializer` 和 `tikv-importer`。

    TiDB Operator 维护的所有镜像的时区均为 UTC。如果你使用自己的镜像文件，需要保证镜像内的时区为 UTC。

## 其他重要更新

* 修复了同时启用 `PodWebhook` 和 增强型 StatefulSet 时 `TidbCluster` 升级的错误 ([#2507](https://github.com/pingcap/tidb-operator/pull/2507), [@Yisaer](https://github.com/Yisaer))
* `tidb-scheduler` 中支持 preemption ([#2510](https://github.com/pingcap/tidb-operator/pull/2510), [@cofyc](https://github.com/cofyc))
* 将 BR 更新为 v4.0.0-rc.2，包含 `auto_random` 的修复 ([#2508](https://github.com/pingcap/tidb-operator/pull/2508), [@DanielZhangQD](https://github.com/DanielZhangQD))
* 增强型 StatefulSet 支持 TiFlash ([#2469](https://github.com/pingcap/tidb-operator/pull/2469), [@DanielZhangQD](https://github.com/DanielZhangQD))
* 在运行 TiDB 前同步 Pump ([#2515](https://github.com/pingcap/tidb-operator/pull/2515), [@DanielZhangQD](https://github.com/DanielZhangQD))
* 删除 `TidbControl` 锁以提高性能 ([#2489](https://github.com/pingcap/tidb-operator/pull/2489), [@weekface](https://github.com/weekface))
* 在 `TidbCluster` 中支持 TiCDC 功能 ([#2362](https://github.com/pingcap/tidb-operator/pull/2362), [@weekface](https://github.com/weekface))
* 将 TiDB/TiKV/PD 配置更新为 4.0.0 GA 版本 ([#2571](https://github.com/pingcap/tidb-operator/pull/2571), [@Yisaer](https://github.com/Yisaer))
* TiDB Operator 将不会对 Failover 创建的 PD Pod 再次进行 Failover ([#2570](https://github.com/pingcap/tidb-operator/pull/2570), [@Yisaer](https://github.com/Yisaer))
