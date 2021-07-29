---
title: TiDB Operator 1.2.0 Release Notes
---

# TiDB Operator 1.2.0 Release Notes

Release date: July 29, 2021

TiDB Operator version: 1.2.0

## Rolling update changes

- Upgrading TiDB Operator will cause the recreation of the TidbMonitor Pod due to [#4085](https://github.com/pingcap/tidb-operator/pull/4085)

## New features

- Support setting Prometheus `retentionTime`, which is more fine-grained than `reserveDays`, and only `retentionTime` takes effect if both are configured ([#4085](https://github.com/pingcap/tidb-operator/pull/4085), [@better0332](https://github.com/better0332))
- Support setting `priorityClassName` in the `Backup` CR to specify the priority of the backup job ([#4078](https://github.com/pingcap/tidb-operator/pull/4078), [@mikechengwei](https://github.com/mikechengwei))

## Improvements

- Changes the default Region leader eviction timeout of TiKV to 1500 minutes. The change prevents the Pod from being deleted when the Region leaders are not transferred completely to the other stores, which will cause data corruption ([#4071](https://github.com/pingcap/tidb-operator/pull/4071), [@KanShiori](https://github.com/KanShiori))

## Bug fixes

- Fix the issue that the URL in `Prometheus.RemoteWrite` may be parsed incorrectly in `TiDBMonitor` ([#4087](https://github.com/pingcap/tidb-operator/pull/4087), [@better0332](https://github.com/better0332))
