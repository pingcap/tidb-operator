---
title: TiDB Operator 1.1.4 Release Notes
---

# TiDB Operator 1.1.4 Release Notes

Release date: August 21, 2020

TiDB Operator version: 1.1.4

## Notable changes

- `TableFilter` is added to the `BackupSpec` and `RestoreSpec`. `TableFilter` supports backing up specific databases or tables with Dumpling or BR and supports restoring specific databases or tables with BR.
  `BackupSpec.Dumpling.TableFilter` is deprecated since v1.1.4. Please configure `BackupSpec.TableFilter` instead.
  Since TiDB v4.0.3, you can configure `BackupSpec.TableFilter` to replace the `BackupSpec.BR.DB` and `BackupSpec.BR.Table` fields and configure `RestoreSpec.TableFilter` to replace the `RestoreSpec.BR.DB` and `RestoreSpec.BR.Table` fields ([#3134](https://github.com/pingcap/tidb-operator/pull/3134), [@sstubbs](https://github.com/sstubbs))
- Update the version of TiDB and tools to v4.0.4 ([#3135](https://github.com/pingcap/tidb-operator/pull/3135), [@lichunzhu](https://github.com/lichunzhu))
- Support customizing environment variables for the initializer container in the TidbMonitor CR ([#3109](https://github.com/pingcap/tidb-operator/pull/3109), [@kolbe](https://github.com/kolbe))
- Support patching PVCs when the storage request is increased ([#3096](https://github.com/pingcap/tidb-operator/pull/3096), [@cofyc](https://github.com/cofyc))
- Support TLS for Backup & Restore with Dumpling & TiDB Lightning ([#3100](https://github.com/pingcap/tidb-operator/pull/3100), [@lichunzhu](https://github.com/lichunzhu))
- Support `cert-allowed-cn` for TiFlash ([#3101](https://github.com/pingcap/tidb-operator/pull/3101), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add support for the [`max-index-length`](https://docs.pingcap.com/tidb/stable/tidb-configuration-file#max-index-length) TiDB config option to the TidbCluster CRD ([#3076](https://github.com/pingcap/tidb-operator/pull/3076), [@kolbe](https://github.com/kolbe))
- Fix goroutine leak when TLS is enabled ([#3081](https://github.com/pingcap/tidb-operator/pull/3081), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix a memory leak issue caused by etcd client when TLS is enabled ([#3064](https://github.com/pingcap/tidb-operator/pull/3064), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support TLS for TiFlash ([#3049](https://github.com/pingcap/tidb-operator/pull/3049), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Configure TZ environment for admission webhook and advanced statefulset controller deployed in tidb-operator chart ([#3034](https://github.com/pingcap/tidb-operator/pull/3034), [@cofyc](https://github.com/cofyc))
