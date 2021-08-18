---
title: TiDB Operator 1.2.1 Release Notes
---

# TiDB Operator 1.2.1 Release Notes

Release date: August 18, 2021

TiDB Operator version: 1.2.1

## Rolling update changes

- If [`hostNetwork`](../configure-a-tidb-cluster.md#hostnetwork) is enabled for TiCDC, upgrading TiDB Operator will cause the recreation of the TiCDC Pod due to [#4141](https://github.com/pingcap/tidb-operator/pull/4141)

## Improvements

- Support configuring [`hostNetwork`](../configure-a-tidb-cluster.md#hostnetwork) for all components in TidbCluster so that all components can use host network ([#4141](https://github.com/pingcap/tidb-operator/pull/4141), [@DanielZhangQD](https://github.com/DanielZhangQD))
