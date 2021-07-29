---
title: TiDB Operator 1.2.0 Release Notes
---

# TiDB Operator 1.2.0 Release Notes

发布日期：2021 年 7 月 29 日

TiDB Operator 版本：1.2.0

## 滚动升级改动

- 由于 [#4085](https://github.com/pingcap/tidb-operator/pull/4085) 的改动，升级 TiDB Operator 会导致 TidbMonitor Pod 删除重建

## 新功能

- 支持为 `TiDBMonitor` 的 `Prometheus` 设置比 `reserveDays` 更细粒度的 `retentionTime`，两者都配置的情况下仅使用 `retentionTime` ([#4085](https://github.com/pingcap/tidb-operator/pull/4085), [@better0332](https://github.com/better0332))
- 支持 `Backup` CR 通过 `priorityClassName` 设置备份 Job 优先级 ([#4078](https://github.com/pingcap/tidb-operator/pull/4078), [@mikechengwei](https://github.com/mikechengwei))

## 优化提升

- 将升级过程中驱逐 TiKV 的 Region Leader 超时的默认值调整为 1500 分钟，避免驱逐尚未完成时停止 Pod 导致数据损坏 ([#4071](https://github.com/pingcap/tidb-operator/pull/4071), [@KanShiori](https://github.com/KanShiori))

## Bug 修复

- 修复解析 `TiDBMonitor` 定义中 `Prometheus.RemoteWrite` 的 URL 可能失败的问题 ([#4087](https://github.com/pingcap/tidb-operator/pull/4087), [@better0332](https://github.com/better0332))
