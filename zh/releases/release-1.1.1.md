---
title: TiDB Operator 1.1.1 Release Notes
---

# TiDB Operator 1.1.1 Release Notes

发布日期：2020 年 6 月 19 日

TiDB Operator 版本：1.1.1

## 重大变化

- 添加 `additionalContainers` 和 `additionalVolumes` 字段，以便 TiDB Operator 添加 `sidecar` 到 TiDB、TiKV、PD 等 ([#2229](https://github.com/pingcap/tidb-operator/pull/2229), [@yeya24](https://github.com/yeya24))
- 添加交叉检查以确保 TiKV 不会同时被扩缩容和升级 ([#2705](https://github.com/pingcap/tidb-operator/pull/2705), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 修复了当未设置 `ClusterRef` 中的 `namespace` 时，`TidbMonitor` 会在不同的 `namespace` 中抓取多个具有相同名字的 `TidbCluster` 的监控数据的问题 ([#2746](https://github.com/pingcap/tidb-operator/pull/2746), [@Yisaer](https://github.com/Yisaer))
- 更新 TiDB Operator 部署 TiDB Cluster 4.0.0 版本镜像的示例 ([#2600](https://github.com/pingcap/tidb-operator/pull/2600), [@kolbe](https://github.com/kolbe))
- 为 TidbMonitor 添加 `alertMangerAlertVersion` 配置 ([#2744](https://github.com/pingcap/tidb-operator/pull/2744), [@weekface](https://github.com/weekface))
- 修复滚动升级后监控的告警规则丢失的问题 ([#2715](https://github.com/pingcap/tidb-operator/pull/2715), [@weekface](https://github.com/weekface))
- 修复了先缩容再扩容时 Pod 可能长时间停留在 `Pending` 状态的问题 ([#2709](https://github.com/pingcap/tidb-operator/pull/2709), [@cofyc](https://github.com/cofyc))
- 在 `PDSpec` 中添加 `EnableDashboardInternalProxy` 配置以使用户可以直接访问 PD Dashboard ([#2713](https://github.com/pingcap/tidb-operator/pull/2713), [@Yisaer](https://github.com/Yisaer))
- 修复了当 `TidbMonitor` 和 `TidbCluster` 的 `reclaimPolicy` 值不同时，PV 同步错误的问题 ([#2707](https://github.com/pingcap/tidb-operator/pull/2707), [@Yisaer](https://github.com/Yisaer))
- 更新 TiDB Operator 的配置版本到 v4.0.1 ([#2702](https://github.com/pingcap/tidb-operator/pull/2702), [@Yisaer](https://github.com/Yisaer))
- 将 tidb-discovery 策略类型更改为 `Recreate`，以修复可能同时存在多个 discovery pod 的问题 ([#2701](https://github.com/pingcap/tidb-operator/pull/2701), [@weekface](https://github.com/weekface))
- 支持在集群开启 TLS 时暴露 `Dashboard` 服务 ([#2684](https://github.com/pingcap/tidb-operator/pull/2684), [@Yisaer](https://github.com/Yisaer))
- 添加 `.tikv.dataSubDir` 字段以指定数据卷下子目录来存储 TiKV 数据 ([#2682](https://github.com/pingcap/tidb-operator/pull/2682), [@cofyc](https://github.com/cofyc))
- 支持向所有组件添加 `imagePullSecrets` 属性 ([#2679](https://github.com/pingcap/tidb-operator/pull/2679), [@weekface](https://github.com/weekface))
- 支持 StatefulSet 和 Pod 验证 webhook 同时工作 ([#2664](https://github.com/pingcap/tidb-operator/pull/2664), [@Yisaer](https://github.com/Yisaer))
- 在 TiDB Operator 同步标签到 TiKV stores 失败时提交一个事件 ([#2587](https://github.com/pingcap/tidb-operator/pull/2587), [@PengJi](https://github.com/PengJi))
- 在 `Backup` 和 `Restore` jobs 日志中隐藏 `datasource` 信息 ([#2652](https://github.com/pingcap/tidb-operator/pull/2652), [@Yisaer](https://github.com/Yisaer))
- 支持 TidbCluster Spec 配置 `enableDynamicConfiguration` ([#2539](https://github.com/pingcap/tidb-operator/pull/2539), [@Yisaer](https://github.com/Yisaer))
- 在 `TidbCluster` 和 `TidbMonitor` 的 `ServiceSpec` 支持 `LoadBalancerSourceRanges` 配置 ([#2610](https://github.com/pingcap/tidb-operator/pull/2610), [@shonge](https://github.com/shonge))
- 当部署了 `TidbMonitor` 时，支持 `TidbCluster` 的 Dashboard metrics 功能 ([#2483](https://github.com/pingcap/tidb-operator/pull/2483), [@Yisaer](https://github.com/Yisaer))
- 更新 TiDB Operator 中的 DM 的版本到 v2.0.0-beta.1 ([#2615](https://github.com/pingcap/tidb-operator/pull/2615), [@tennix](https://github.com/tennix))
- 支持配置 discovery 资源 ([#2434](https://github.com/pingcap/tidb-operator/pull/2434), [@shonge](https://github.com/shonge))
- 支持 `TidbCluster` 自动扩缩容的降噪 ([#2307](https://github.com/pingcap/tidb-operator/pull/2307), [@vincent178](https://github.com/vincent178))
- 支持在 `TidbMonitor` 中抓取 `Pump` 和 `Drainer` 监控指标 ([#2750](https://github.com/pingcap/tidb-operator/pull/2750), [@Yisaer](https://github.com/Yisaer))
