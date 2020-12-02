---
title: TiDB Operator 1.1.4 Release Notes
---

# TiDB Operator 1.1.4 Release Notes

发布日期：2020 年 8 月 21 日

TiDB Operator 版本：1.1.4

## 重大变化

- 将 `TableFilter` 添加到 `BackupSpec` 和 `RestoreSpec`。`TableFilter` 支持使用 Dumpling 或 BR 备份特定的数据库或表，并支持使用 BR 恢复特定的数据库或表。
  从 v1.1.4 开始，已弃用 `BackupSpec.Dumpling.TableFilter`，请改为配置 `BackupSpec.TableFilter`。
  从 TiDB v4.0.3 开始，可以通过配置 `BackupSpec.TableFilter` 来替换 `BackupSpec.BR.DB` 和 `BackupSpec.BR.Table` 字段，并且通过配置 `RestoreSpec.TableFilter` 来替换 `RestoreSpec.BR.DB` 和 `RestoreSpec.BR.Table` 字段 ([#3134](https://github.com/pingcap/tidb-operator/pull/3134)，[@sstubbs](https://github.com/sstubbs))
- 更新 TiDB 和配套工具的版本为 v4.0.4 ([#3135](https://github.com/pingcap/tidb-operator/pull/3135)，[@lichunzhu](https://github.com/lichunzhu))
- 支持在 TidbMonitor CR 中为初始化容器自定义环境变量 ([#3109](https://github.com/pingcap/tidb-operator/pull/3109)，[@kolbe](https://github.com/kolbe))
- 支持增加存储请求 ([#3096](https://github.com/pingcap/tidb-operator/pull/3096)，[@cofyc](https://github.com/cofyc))
- 为使用 Dumpling 和 TiDB Lightning 进行备份恢复添加 TLS 支持 ([#3100](https://github.com/pingcap/tidb-operator/pull/3100)，[@lichunzhu](https://github.com/lichunzhu))
- 支持 TiFlash 中的 `cert-allowed-cn` 配置项 ([#3101](https://github.com/pingcap/tidb-operator/pull/3101)，[@DanielZhangQD](https://github.com/DanielZhangQD))
- 在 TidbCluster CRD 中添加对 TiDB 配置项 [`max-index-length`](https://docs.pingcap.com/tidb/stable/tidb-configuration-file#max-index-length) 的支持 ([#3076](https://github.com/pingcap/tidb-operator/pull/3076)，[@kolbe](https://github.com/kolbe))
- 修复了启用 TLS 时的 goroutine 泄漏 ([#3081](https://github.com/pingcap/tidb-operator/pull/3081)，[@DanielZhangQD](https://github.com/DanielZhangQD))
- 修复了启用 TLS 时由 etcd 客户端引起的内存泄漏问题 ([#3064](https://github.com/pingcap/tidb-operator/pull/3064), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 为 TiFlash 添加 TLS 支持 ([#3049](https://github.com/pingcap/tidb-operator/pull/3049)，[@DanielZhangQD](https://github.com/DanielZhangQD))
- 为 tidb-operator chart 中部署的 Admission Webhook 和增强型 Statefulset 控制器配置 TZ 环境 ([#3034](https://github.com/pingcap/tidb-operator/pull/3034)，[@cofyc](https://github.com/cofyc))
