---
title: TiDB Operator 0.2.1 Release Notes
---

# TiDB Operator 0.2.1 Release Notes

Release date: September 20, 2018

TiDB Operator version: 0.2.1

## Bug Fixes

- Fix retry on conflict logic ([#87](https://github.com/pingcap/tidb-operator/pull/87))
- Fix TiDB timezone configuration by setting TZ environment variable ([#96](https://github.com/pingcap/tidb-operator/pull/96))
- Fix failover by keeping spec replicas unchanged ([#95](https://github.com/pingcap/tidb-operator/pull/95))
- Fix repeated updating pod and pd/tidb StatefulSet ([#101](https://github.com/pingcap/tidb-operator/pull/101))
