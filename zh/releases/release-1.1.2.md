---
title: TiDB Operator 1.1.2 Release Notes
---

# TiDB Operator 1.1.2 Release Notes

Release date: July 1, 2020

TiDB Operator version: 1.1.2

## Action Required

- An incompatible issue with PD 4.0.2 has been fixed. Please upgrade TiDB Operator to v1.1.2 before deploying TiDB 4.0.2 and later versions ([#2809](https://github.com/pingcap/tidb-operator/pull/2809), [@cofyc](https://github.com/cofyc))

## Other Notable Changes

- Collect metrics for TiCDC, TiDB Lightning and TiKV Importer ([#2835](https://github.com/pingcap/tidb-operator/pull/2835), [@weekface](https://github.com/weekface))
- Update PD/TiDB/TiKV config to v4.0.2 ([#2828](https://github.com/pingcap/tidb-operator/pull/2828), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix the bug that `PD` Member might still exist after scaling-in ([#2793](https://github.com/pingcap/tidb-operator/pull/2793), [@Yisaer](https://github.com/Yisaer))
- Support Auto-Scaler Reference in `TidbCluster` Status when there exists `TidbClusterAutoScaler` ([#2791](https://github.com/pingcap/tidb-operator/pull/2791), [@Yisaer](https://github.com/Yisaer))
- Support configuring container lifecycle hooks and `terminationGracePeriodSeconds` in TiDB spec ([#2810](https://github.com/pingcap/tidb-operator/pull/2810), [@weekface](https://github.com/weekface))
