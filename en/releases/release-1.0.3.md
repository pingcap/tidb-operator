---
title: TiDB Operator 1.0.3 Release Notes
---

# TiDB Operator 1.0.3 Release Notes

Release date: November 13, 2019

TiDB Operator version: 1.0.3

## v1.0.3 What's New

### Action Required

ACTION REQUIRED: This release upgrades default TiDB version to `v3.0.5` which fixed a serious [bug](https://github.com/pingcap/tidb/pull/12597) in TiDB. So if you are using TiDB `v3.0.4` or prior versions, you **must** upgrade to `v3.0.5`.

ACTION REQUIRED: This release adds the `timezone` support for [all charts](https://github.com/pingcap/tidb-operator/tree/master/charts).

For existing TiDB clusters. If the `timezone` in `tidb-cluster/values.yaml` has been customized to other timezones instead of the default `UTC`, then upgrading tidb-operator will trigger a rolling update for the related pods.

The related pods include `pump`, `drainer`, `discovery`, `monitor`, `scheduled backup`, `tidb-initializer`, and `tikv-importer`.

The time zone for all images maintained by `tidb-operator` should be `UTC`. If you use your own images, you need to make sure that the corresponding time zones are `UTC`.

### Improvements

- Add the `timezone` support for all containers of the TiDB cluster
- Support configuring resource requests and limits for all containers of the TiDB cluster

## Detailed Bug Fixes and Changes

- Upgrade default TiDB version to `v3.0.5` ([#1132](https://github.com/pingcap/tidb-operator/pull/1132))
- Add the `timezone` support for all containers of the TiDB cluster ([#1122](https://github.com/pingcap/tidb-operator/pull/1122))
- Support configuring resource requests and limits for all containers of the TiDB cluster ([#853](https://github.com/pingcap/tidb-operator/pull/853))
