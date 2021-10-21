---
title: TiDB Operator 1.2.4 Release Notes
---

# TiDB Operator 1.2.4 Release Notes

Release date: October 21, 2021

TiDB Operator version: 1.2.4

## Rolling update changes

- Upgrading TiDB Operator will cause the recreation of the TiDBMonitor Pod due to [#4180](https://github.com/pingcap/tidb-operator/pull/4180)

## New features

- TidbMonitor supports customizing prometheus rules and reloading configurations dynamically ([#4180](https://github.com/pingcap/tidb-operator/pull/4180), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor supports the `enableRules` field. When AlertManager is not configured, you can configure this field to `true` to add Prometheus rules ([#4115](https://github.com/pingcap/tidb-operator/pull/4115), [@mikechengwei](https://github.com/mikechengwei))

## Improvements

- Optimize `TiFlash` rolling upgrade process ([#4193](https://github.com/pingcap/tidb-operator/pull/4193), [@KanShiori](https://github.com/KanShiori))
- Support deleting backup data in batches ([#4095](https://github.com/pingcap/tidb-operator/pull/4095), [@KanShiori](https://github.com/KanShiori))

## Bug fixes

- Fix the security vulnerabilities in the `tidb-backup-manager` and `tidb-operator` images ([#4217](https://github.com/pingcap/tidb-operator/pull/4217), [@KanShiori](https://github.com/KanShiori))
- Fix the issue that some backup data might retain if the `Backup` CR is deleted when the `Backup` job is running ([#4133](https://github.com/pingcap/tidb-operator/pull/4133), [@KanShiori](https://github.com/KanShiori))
