---
title: TiDB Operator 1.1.3 Release Notes
---

# TiDB Operator 1.1.3 Release Notes

Release date: July 27, 2020

TiDB Operator version: 1.1.3

## Action Required

- Add a field `cleanPolicy` in `BackupSpec` to denote the clean policy for backup data when the Backup CR is deleted from the cluster (default to `Retain`). Note that before v1.1.3, TiDB Operator will clean the backup data in the remote storage when the Backup CR is deleted, so if you want to clean backup data as before, set `spec.cleanPolicy` in `Backup` CR or `spec.backupTemplate.cleanPolicy` in `BackupSchedule` CR to `Delete`. ([#3002](https://github.com/pingcap/tidb-operator/pull/3002), [@lichunzhu](https://github.com/lichunzhu))
- Replace `mydumper` with `dumpling` for backup.

    If `spec.mydumper` is configured in the `Backup` CR or `spec.backupTemplate.mydumper` is configured in the `BackupSchedule` CR, migrate it to `spec.dumpling` or `spec.backupTemplate.dumpling`. After you upgrade TiDB Operator to v1.1.3, note that `spec.mydumper` or `spec.backupTemplate.mydumper` will be lost after the upgrade. ([#2870](https://github.com/pingcap/tidb-operator/pull/2870), [@lichunzhu](https://github.com/lichunzhu))

## Other Notable Changes

- Update tools in backup manager to v4.0.3 ([#3019](https://github.com/pingcap/tidb-operator/pull/3019), [@lichunzhu](https://github.com/lichunzhu))
- Support `cleanPolicy` for the `Backup` CR to define the clean behavior of the backup data in the remote storage when the `Backup` CR is deleted ([#3002](https://github.com/pingcap/tidb-operator/pull/3002), [@lichunzhu](https://github.com/lichunzhu))
- Add TLS support for TiCDC ([#3011](https://github.com/pingcap/tidb-operator/pull/3011), [@weekface](https://github.com/weekface))
- Add TLS support between Drainer and the downstream database server ([#2993](https://github.com/pingcap/tidb-operator/pull/2993), [@lichunzhu](https://github.com/lichunzhu))
- Support specifying `mysqlNodePort` and `statusNodePort` for TiDB Service Spec ([#2941](https://github.com/pingcap/tidb-operator/pull/2941), [@lichunzhu](https://github.com/lichunzhu))
- Fix the `initialCommitTs` bug in Drainer's `values.yaml` ([#2857](https://github.com/pingcap/tidb-operator/pull/2857), [@weekface](https://github.com/weekface))
- Add `backup` config for TiKV server, add `enable-telemetry`, and deprecate `disable-telemetry` config for PD server ([#2964](https://github.com/pingcap/tidb-operator/pull/2964), [@lichunzhu](https://github.com/lichunzhu))
- Add commitTS info column in `get restore` command ([#2926](https://github.com/pingcap/tidb-operator/pull/2926), [@lichunzhu](https://github.com/lichunzhu))
- Update the used Grafana version from v6.0.1 to v6.1.6 ([#2923](https://github.com/pingcap/tidb-operator/pull/2923), [@lichunzhu](https://github.com/lichunzhu))
- Support showing commitTS in restore status ([#2899](https://github.com/pingcap/tidb-operator/pull/2899), [@lichunzhu](https://github.com/lichunzhu))
- Exit without error if the backup data the user tries to clean does not exist ([#2916](https://github.com/pingcap/tidb-operator/pull/2916), [@lichunzhu](https://github.com/lichunzhu))
- Support auto-scaling by storage for TiKV in `TidbClusterAutoScaler` ([#2884](https://github.com/pingcap/tidb-operator/pull/2884), [@Yisaer](https://github.com/Yisaer))
- Clean temporary files in `Backup` job with `Dumpling` to save space ([#2897](https://github.com/pingcap/tidb-operator/pull/2897), [@lichunzhu](https://github.com/lichunzhu))
- Fail the backup job if existing PVC's size is smaller than the storage request in the backup job ([#2894](https://github.com/pingcap/tidb-operator/pull/2894), [@lichunzhu](https://github.com/lichunzhu))
- Support scaling and auto-failover even if a TiKV store fails in upgrading ([#2886](https://github.com/pingcap/tidb-operator/pull/2886), [@cofyc](https://github.com/cofyc))
- Fix a bug that the `TidbMonitor` resource could not be set ([#2878](https://github.com/pingcap/tidb-operator/pull/2878), [@weekface](https://github.com/weekface))
- Fix an error for the monitor creation in the tidb-cluster chart ([#2869](https://github.com/pingcap/tidb-operator/pull/2869), [@8398a7](https://github.com/8398a7))
- Remove `readyToScaleThresholdSeconds` in `TidbClusterAutoScaler`; TiDB Operator won't support de-noise in `TidbClusterAutoScaler` ([#2862](https://github.com/pingcap/tidb-operator/pull/2862), [@Yisaer](https://github.com/Yisaer))
- Update the version of TiDB Lightning used in tidb-backup-manager from v3.0.15 to v4.0.2 ([#2865](https://github.com/pingcap/tidb-operator/pull/2865), [@lichunzhu](https://github.com/lichunzhu))
