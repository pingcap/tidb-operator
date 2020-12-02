---
title: TiDB Operator 1.1.3 Release Notes
---

# TiDB Operator 1.1.3 Release Notes

发布日期：2020 年 7 月 27 日

TiDB Operator 版本：1.1.3

## 需要采取的行动

- 在 `BackupSpec` 中添加字段 `cleanPolicy`，表示从集群中删除 `Backup` CR 时对备份数据采取的清理策略（默认为 `Retain`）。需要注意的是，在 v1.1.3 之前的版本中，TiDB Operator 将在删除 `Backup` CR 时清除远程存储中的备份数据。因此，如果想像以前一样清除备份数据，请在 `Backup` CR 的 `spec.cleanPolicy` 或 `BackupSchedule` CR 中的 `spec.backupTemplate.cleanPolicy` 设置为 `Delete` ([#3002](https://github.com/pingcap/tidb-operator/pull/3002)，[@lichunzhu](https://github.com/lichunzhu))
- 将 `mydumper` 替换为 `dumpling` 进行备份。

    如果在 `Backup` CR 中配置了 `spec.mydumper`，或者在 `BackupSchedule` CR 中配置了 `spec.backupTemplate.mydumper`，需要将该配置迁移至 `spec.dumpling` 或 `spec.backupTemplate.dumpling`。请注意，TiDB Operator 升级到 v1.1.3 后，`spec.mydumper` 或 `spec.backupTemplate.mydumper` 配置会丢失 ([#2870](https://github.com/pingcap/tidb-operator/pull/2870)， [@lichunzhu](https://github.com/lichunzhu))

## 其他需要注意的变更

- 将 backup manager 中的工具更新到 v4.0.3 ([#3019](https://github.com/pingcap/tidb-operator/pull/3019)，[@lichunzhu](https://github.com/lichunzhu))
- 在 `Backup` CR 中添加 `cleanPolicy` 字段，表示从集群中删除 `Backup` CR 时对远端存储的备份数据采取的清理策略 ([#3002](https://github.com/pingcap/tidb-operator/pull/3002)，[@lichunzhu](https://github.com/lichunzhu))
- 为 TiCDC 添加 TLS 支持 ([#3011](https://github.com/pingcap/tidb-operator/pull/3011)，[@weekface](https://github.com/weekface))
- 在 Drainer 和下游数据库服务器之间添加 TLS 支持 ([#2993](https://github.com/pingcap/tidb-operator/pull/2993)，[@lichunzhu](https://github.com/lichunzhu))
- 支持为 TiDB Service 指定 `mysqlNodePort` 和 `statusNodePort` ([#2941](https://github.com/pingcap/tidb-operator/pull/2941)，[@lichunzhu](https://github.com/lichunzhu))
- 修复 Drainer `values.yaml` 文件中的 `initialCommitTs` 错误 ([#2857](https://github.com/pingcap/tidb-operator/pull/2857)，[@weekface](https://github.com/weekface))
- 为 TiKV 添加 `backup` 配置，为 PD 添加 `enable-telemetry` 并弃用 `disable-telemetry` 配置 ([#2964](https://github.com/pingcap/tidb-operator/pull/2964)，[@lichunzhu](https://github.com/lichunzhu))
- 在 `get restore` 命令中添加 commitTS 列 ([#2926](https://github.com/pingcap/tidb-operator/pull/2926)，[@lichunzhu](https://github.com/lichunzhu))
- 将 Grafana 版本从 v6.0.1 更新到 v6.1.6 ([#2923](https://github.com/pingcap/tidb-operator/pull/2923)， [@lichunzhu](https://github.com/lichunzhu))
- 在 `Restore` CR 中的 `Status` 字段下增加 commitTS 字段 ([#2899](https://github.com/pingcap/tidb-operator/pull/2899)，[@lichunzhu](https://github.com/lichunzhu))
- 如果用户尝试清除的备份数据不存在，则不报错并退出 ([#2916](https://github.com/pingcap/tidb-operator/pull/2916)，[@lichunzhu](https://github.com/lichunzhu))
- 支持在 `TidbClusterAutoScaler` 中设置 TiKV 根据剩余存储容量自动扩容 ([#2884](https://github.com/pingcap/tidb-operator/pull/2884)，[@Yisaer](https://github.com/Yisaer))
- 清理 `Dumpling` 备份 Job 中的临时文件，以节省空间 ([#2897](https://github.com/pingcap/tidb-operator/pull/2897)，[@lichunzhu](https://github.com/lichunzhu))
- 如果现有 PVC 的大小小于 `Backup` Job 中的存储请求，则 `Backup` Job 失败 ([#2894](https://github.com/pingcap/tidb-operator/pull/2894)，[@lichunzhu](https://github.com/lichunzhu))
- 支持在 TiKV store 升级失败时扩缩容和自动故障转移 ([#2886](https://github.com/pingcap/tidb-operator/pull/2886)，[@cofyc](https://github.com/cofyc))
- 修复了无法设置 `TidbMonitor` 资源的问题 ([#2878](https://github.com/pingcap/tidb-operator/pull/2878)，[@weekface](https://github.com/weekface))
- 修复了 tidb-cluster chart 中监控组件创建失败的问题 ([#2869](https://github.com/pingcap/tidb-operator/pull/2869)，[@8398a7](https://github.com/8398a7))
- 在 `TidbClusterAutoScaler` 中删除 `readyToScaleThresholdSeconds`；TiDB Operator 不支持 `TidbClusterAutoScaler` 中的降噪 ([#2862](https://github.com/pingcap/tidb-operator/pull/2862)，[@Yisaer](https://github.com/Yisaer))
- 将 tidb-backup-manager 中使用的 TiDB Lightning 版本从 v3.0.15 更新到 v4.0.2 ([#2865](https://github.com/pingcap/tidb-operator/pull/2865)，[@lichunzhu](https://github.com/lichunzhu))
