---
title: TiDB Operator 1.2.2 Release Notes
---

# TiDB Operator 1.2.2 Release Notes

发布日期：2021 年 9 月 3 日

TiDB Operator 版本：1.2.2

## 滚动升级改动

- 由于 [#4158](https://github.com/pingcap/tidb-operator/pull/4158) 的改动，升级 TiDB Operator 会导致 TiDBMonitor Pod 删除重建
- 由于 [#4152](https://github.com/pingcap/tidb-operator/pull/4152) 的改动，升级 TiDB Operator 会导致 TiFlash Pod 删除重建

## 新功能

- TiDBMonitor 支持动态重新加载配置 ([#4158](https://github.com/pingcap/tidb-operator/pull/4158), [@mikechengwei](https://github.com/mikechengwei))

## Bug 修复

- 修复 TiCDC 无法从低版本升级到 v5.2.0 的问题 ([#4171](https://github.com/pingcap/tidb-operator/pull/4171), [@KanShiori](https://github.com/KanShiori))
