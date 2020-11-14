---
title: TiDB Operator 1.1.7 Release Notes
---

# TiDB Operator 1.1.7 Release Notes

Release date: November 13, 2020

TiDB Operator version: 1.1.7

## Compatibility Changes

- The behavior of `prometheus.spec.config.commandOptions` has changed. Any duplicated flags must be removed, or Prometheus will fail to start. ([#3390](https://github.com/pingcap/tidb-operator/pull/3390), [@mightyguava](https://github.com/mightyguava))

    Flags that CANNOT be set are:

    - `--web.enable-admin-api`
    - `--web.enable-lifecycle`
    - `--config.file`
    - `--storage.tsdb.path`
    - `--storage.tsdb.retention`

## New Features

- Support `spec.toolImage` for the `Backup` and `Restore` CR with BR to define the image used to provide the BR binary executables. Defaults to `pingcap/br:${tikv_version}` ([#3471](https://github.com/pingcap/tidb-operator/pull/3471), [@namco1992](https://github.com/namco1992))
- Add `spec.tidb.storageVolumes`, `spec.tikv.storageVolumes`, and `spec.pd.storageVolumes` to support mounting multiple PVs for TiDB, TiKV, and PD ([#3425](https://github.com/pingcap/tidb-operator/pull/3425) [#3444](https://github.com/pingcap/tidb-operator/pull/3444), [@mikechengwei](https://github.com/mikechengwei))
- Add `spec.tidb.readinessProbe` config to support requesting `http://127.0.0.0:10080/status` for TiDB's readiness probe, TiDB version >= v4.0.9 required ([#3438](https://github.com/pingcap/tidb-operator/pull/3438), [@july2993](https://github.com/july2993))
- Support PD leader transfer with advanced StatefulSet controller enabled ([#3395](https://github.com/pingcap/tidb-operator/pull/3395), [@tangwz](https://github.com/tangwz))
- Support setting `OnDelete` update strategies for the StatefulSets via `spec.statefulSetUpdateStrategy` in the TidbCluster CR ([#3408](https://github.com/pingcap/tidb-operator/pull/3408), [@cvvz](https://github.com/cvvz))
- Support HA scheduling when failover happens ([#3419](https://github.com/pingcap/tidb-operator/pull/3419), [@cvvz](https://github.com/cvvz))
- Support smooth migration from TiDB clusters deployed using TiDB Ansible or TiUP or deployed in the same Kubernetes cluster to a new TiDB cluster ([#3226](https://github.com/pingcap/tidb-operator/pull/3226), [@cvvz](https://github.com/cvvz))
- tidb-scheduler supports advanced StatefulSet ([#3388](https://github.com/pingcap/tidb-operator/pull/3388), [@cvvz](https://github.com/cvvz))

## Improvements

- Forbid to scale in TiKV when the number of UP stores is equal to or less than 3 ([#3367](https://github.com/pingcap/tidb-operator/pull/3367), [@cvvz](https://github.com/cvvz))
- `phase` is added in `BackupStatus` and `RestoreStatus`, which will be in sync with the latest condition type and shown when doing `kubectl get` ([#3397](https://github.com/pingcap/tidb-operator/pull/3397), [@namco1992](https://github.com/namco1992))
- Skip setting `tikv_gc_life_time` via SQL for backup and restore with BR when the TiKV version >= v4.0.8 ([#3443](https://github.com/pingcap/tidb-operator/pull/3443), [@namco1992](https://github.com/namco1992))

## Bug Fixes

- Fix the issue that PD cannot scale in to zero if there are other PD members outside of this `TidbCluster` ([#3456](https://github.com/pingcap/tidb-operator/pull/3456), [@dragonly](https://github.com/dragonly))
