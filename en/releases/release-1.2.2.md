---
title: TiDB Operator 1.2.2 Release Notes
---

# TiDB Operator 1.2.2 Release Notes

Release date: September 3, 2021

TiDB Operator version: 1.2.2

## Rolling update changes

- Upgrading TiDB Operator will cause the recreation of the TiDBMonitor Pod due to [#4158](https://github.com/pingcap/tidb-operator/pull/4158)
- Upgrading TiDB Operator will cause the recreation of the TiFlash Pod due to [#4152](https://github.com/pingcap/tidb-operator/pull/4152)

## New features

- TiDBMonitor supports reloading configurations dynamically ([#4158](https://github.com/pingcap/tidb-operator/pull/4158), [@mikechengwei](https://github.com/mikechengwei))

## Bug fixes

- Fix upgrade failures of TiCDC from an earlier version to v5.2.0 ([#4171](https://github.com/pingcap/tidb-operator/pull/4171), [@KanShiori](https://github.com/KanShiori))
