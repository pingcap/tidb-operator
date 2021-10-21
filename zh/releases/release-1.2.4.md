---
title: TiDB Operator 1.2.4 Release Notes
---

# TiDB Operator 1.2.4 Release Notes

发布日期：2021 年 10 月 21 日

TiDB Operator 版本：1.2.4

## 滚动升级改动

- 由于 [#4180](https://github.com/pingcap/tidb-operator/pull/4180) 的改动，升级 TiDB Operator 会导致 TidbMonitor Pod 删除重建

## 新功能

- TidbMonitor 支持用户自定义 Prometheus 告警规则，并且可以动态重新加载告警规则 ([#4180](https://github.com/pingcap/tidb-operator/pull/4180), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor 支持 `enableRules` 字段。当没有配置 AlterManager 时，可以配置该字段为 `true` 来为 Prometheus 添加告警规则 ([#4115](https://github.com/pingcap/tidb-operator/pull/4115), [@mikechengwei](https://github.com/mikechengwei))

## 优化提升

- 优化 `TiFlash` 滚动升级流程 ([#4193](https://github.com/pingcap/tidb-operator/pull/4193), [@KanShiori](https://github.com/KanShiori))
- 支持批量删除备份数据 ([#4095](https://github.com/pingcap/tidb-operator/pull/4095), [@KanShiori](https://github.com/KanShiori))

## Bug 修复

- 修复 `tidb-backup-manager` 和 `tidb-operator` 镜像中的安全漏洞 ([#4217](https://github.com/pingcap/tidb-operator/pull/4217), [@KanShiori](https://github.com/KanShiori))
- 修复当 `Backup` 备份任务正在运行时，如果 `Backup` CR 被删除，备份数据可能残留的问题 ([#4133](https://github.com/pingcap/tidb-operator/pull/4133), [@KanShiori](https://github.com/KanShiori))
