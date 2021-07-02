---
title: TiDB Operator 1.2.0-rc.2 Release Notes
---

# TiDB Operator 1.2.0-rc.2 Release Notes

发布日期：2021 年 7 月 2 日

TiDB Operator 版本：1.2.0-rc.2

## 新功能

- 支持透传 TiCDC 的 TOML 格式配置 ([#4010](https://github.com/pingcap/tidb-operator/pull/4010), [@july2993](https://github.com/july2993))
- 支持为 TiCDC 设置 `StorageVolumes`、`AdditionalVolumes` 和 `AdditionalVolumeMounts` ([#4004](https://github.com/pingcap/tidb-operator/pull/4004), [@csuzhangxc](https://github.com/csuzhangxc))
- 支持为 Discovery、TidbMonitor 和 TidbInitializer 设置自定义的 `labels` 与 `annotations` ([#4029](https://github.com/pingcap/tidb-operator/pull/4029), [@csuzhangxc](https://github.com/csuzhangxc))
- 支持修改 Grafana dashboard ([#4035](https://github.com/pingcap/tidb-operator/pull/4035), [@mikechengwei](https://github.com/mikechengwei))

## 优化提升

- 支持在未为 BR `toolImage` 指定 tag 时将 TiKV 版本作为 tag ([#4048](https://github.com/pingcap/tidb-operator/pull/4048), [@KanShiori](https://github.com/KanShiori))
- 支持在扩缩容 TiDB 过程中协调 PVC ([#4033](https://github.com/pingcap/tidb-operator/pull/4033), [@csuzhangxc](https://github.com/csuzhangxc))
- 为 TiDB Operator 增加 liveness 与 readiness 探测器，便于查看 TiDB Operator 状态 ([#4002](https://github.com/pingcap/tidb-operator/pull/4002), [@mikechengwei](https://github.com/mikechengwei))

## Bug 修复

- 修复部署异构集群时 TiDB Operator 可能 panic 的问题 ([#4054](https://github.com/pingcap/tidb-operator/pull/4054) [#3965](https://github.com/pingcap/tidb-operator/pull/3965), [@KanShiori](https://github.com/KanShiori) [@july2993](https://github.com/july2993))
- 修复当 TidbCluster spec 未更改时 TiDB service 与 TidbCluster 状态仍然持续被更新的问题 ([#4008](https://github.com/pingcap/tidb-operator/pull/4008), [@july2993](https://github.com/july2993))
