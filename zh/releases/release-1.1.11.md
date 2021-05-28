---
title: TiDB Operator 1.1.11 Release Notes
---

# TiDB Operator 1.1.11 Release Notes

发布日期：2021 年 2 月 26 日

TiDB Operator 版本：1.1.11

## 新功能

- 优化 LeaderElection，支持用户配置 LeaderElection 时间 ([#3794](https://github.com/pingcap/tidb-operator/pull/3794), [@july2993](https://github.com/july2993))
- 支持设置自定义 Store 标签并根据 Node 标签获取值  ([#3784](https://github.com/pingcap/tidb-operator/pull/3784), [@L3T](https://github.com/L3T))

## 优化提升

- 添加 TiFlash 滚动更新机制避免升级期间所有 TiFlash Store 同时不可用 ([#3789](https://github.com/pingcap/tidb-operator/pull/3789), [@handlerww](https://github.com/handlerww))
- 由从 PD 获取 region leader 数量改为从 TiKV 直接获取以确保拿到准确的 region leader 数量 ([#3801](https://github.com/pingcap/tidb-operator/pull/3801), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 打印 RocksDB 和 Raft 日志到 stdout，以支持这些日志的收集和查询 ([#3768](https://github.com/pingcap/tidb-operator/pull/3768), [@baurine](https://github.com/baurine))
