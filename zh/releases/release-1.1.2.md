---
title: TiDB Operator 1.1.2 Release Notes
---

# TiDB Operator 1.1.2 Release Notes

发布日期：2020 年 7 月 1 日

TiDB Operator 版本：1.1.2

## 需要采取的行动

- 修复了与 PD 4.0.2 不兼容的问题。在部署 TiDB 4.0.2 及更高版本之前，请将 TiDB Operator 升级到 v1.1.2 ([#2809](https://github.com/pingcap/tidb-operator/pull/2809)， [@cofyc](https://github.com/cofyc))

## 其他需要注意的变更

- 抓取 TiCDC, TiDB Lightning 和 TiKV Importer 的监控指标 ([#2835](https://github.com/pingcap/tidb-operator/pull/2835)，[@weekface](https://github.com/weekface))
- 更新 PD/TiDB/TiKV 配置为 v4.0.2 ([#2828](https://github.com/pingcap/tidb-operator/pull/2828)，[@DanielZhangQD](https://github.com/DanielZhangQD))
- 修复了缩容后 PD 成员可能仍然存在的错误 ([#2793](https://github.com/pingcap/tidb-operator/pull/2793)，[@Yisaer](https://github.com/Yisaer))
- 如果存在 `TidbClusterAutoScaler` CR，同步 `TidbClusterAutoScaler` 信息到 `TidbCluster` `Status` 字段 ([#2791](https://github.com/pingcap/tidb-operator/pull/2791)，[@Yisaer](https://github.com/Yisaer))
- 支持在 TiDB 参数中配置容器生命周期 hook 和 `terminationGracePeriodSeconds` ([#2810](https://github.com/pingcap/tidb-operator/pull/2810)，[@weekface](https://github.com/weekface))
