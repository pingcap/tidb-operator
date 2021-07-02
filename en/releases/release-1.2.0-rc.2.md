---
title: TiDB Operator 1.2.0-rc.2 Release Notes
---

# TiDB Operator 1.2.0-rc.2 Release Notes

Release date: July 2, 2021

TiDB Operator version: 1.2.0-rc.2

## New features

- Support passing raw TOML config for TiCDC ([#4010](https://github.com/pingcap/tidb-operator/pull/4010), [@july2993](https://github.com/july2993))
- Support setting `StorageVolumes`, `AdditionalVolumes`, and `AdditionalVolumeMounts` for TiCDC ([#4004](https://github.com/pingcap/tidb-operator/pull/4004), [@csuzhangxc](https://github.com/csuzhangxc))
- Support setting custom `labels` and `annotations` for Discovery, TidbMonitor, and TidbInitializer ([#4029](https://github.com/pingcap/tidb-operator/pull/4029), [@csuzhangxc](https://github.com/csuzhangxc))
- Support modifying Grafana dashboard ([#4035](https://github.com/pingcap/tidb-operator/pull/4035), [@mikechengwei](https://github.com/mikechengwei))

## Improvements

- Support using the TiKV version as the tag for BR `toolImage` if no tag is specified ([#4048](https://github.com/pingcap/tidb-operator/pull/4048), [@KanShiori](https://github.com/KanShiori))
- Support handling PVC during scaling of TiDB ([#4033](https://github.com/pingcap/tidb-operator/pull/4033), [@csuzhangxc](https://github.com/csuzhangxc))
- Add the liveness and readiness probes for TiDB Operator to check the TiDB Operator status ([#4002](https://github.com/pingcap/tidb-operator/pull/4002), [@mikechengwei](https://github.com/mikechengwei))

## Bug fixes

- Fix the issue that TiDB Operator may panic during the deployment of heterogeneous clusters ([#4054](https://github.com/pingcap/tidb-operator/pull/4054) [#3965](https://github.com/pingcap/tidb-operator/pull/3965), [@KanShiori](https://github.com/KanShiori) [@july2993](https://github.com/july2993))
- Fix the issue that the status of TiDB service and TidbCluster are updated continuously even when no change is made to the TidbCluster spec ([#4008](https://github.com/pingcap/tidb-operator/pull/4008), [@july2993](https://github.com/july2993))
