---
title: TiDB Operator 1.2.1 Release Notes
---

# TiDB Operator 1.2.1 Release Notes

发布日期：2021 年 8 月 18 日

TiDB Operator 版本：1.2.1

## 滚动升级改动

- 由于 [#4141](https://github.com/pingcap/tidb-operator/pull/4141) 的改动，如果你部署 TiCDC 时配置了 [`hostNetwork`](../configure-a-tidb-cluster.md#hostnetwork) 为 `true`，那么升级 TiDB Operator 后会导致 TiCDC Pod 删除重建

## 优化提升

- 支持为 TidbCluster 的所有组件配置 [`hostNetwork`](../configure-a-tidb-cluster.md#hostnetwork)，使所有组件都可以使用宿主机网络 ([#4141](https://github.com/pingcap/tidb-operator/pull/4141), [@DanielZhangQD](https://github.com/DanielZhangQD))
