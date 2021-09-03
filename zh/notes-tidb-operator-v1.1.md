---
title: TiDB Operator v1.1 重要注意事项
aliases: ['/docs-cn/tidb-in-kubernetes/dev/notes-tidb-operator-v1.1/']
---

# TiDB Operator v1.1 重要注意事项

本文介绍 TiDB Operator v1.1 版本重要注意事项。

## PingCAP 不再继续更新维护 tidb-cluster chart

从 TiDB Operator v1.1.0 开始，PingCAP 不再继续更新 tidb-cluster chart，原来由 tidb-cluster chart 负责管理的组件或者功能在 v1.1 中的变更如下：

| 组件、功能 | v1.1 |
| :--- | :--- |
| TiDB Cluster (PD, TiDB, TiKV) | [TidbCluster CR](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md) |
| TiDB Monitor | [TidbMonitor CR](https://github.com/pingcap/tidb-operator/blob/master/manifests/monitor/tidb-monitor.yaml) |
| TiDB Initializer | [TidbInitializer CR](https://github.com/pingcap/tidb-operator/blob/master/manifests/initializer/tidb-initializer.yaml) |
| Scheduled Backup | [BackupSchedule CR](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-schedule-aws-s3-br.yaml) |
| Pump | [TidbCluster CR](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md) |
| Drainer | [tidb-drainer chart](https://github.com/pingcap/tidb-operator/tree/master/charts/tidb-drainer) |
| Importer | [tikv-importer chart](https://github.com/pingcap/tidb-operator/tree/master/charts/tikv-importer) |

- tidb-cluster chart 会继续发布，但是 PingCAP 不再添加新功能，社区贡献者仍然可以为 tidb-cluster chart 添加新功能。
- 通过 v1.0.x TiDB Operator 部署的 TiDB 集群，在 TiDB Operator 升级到 v1.1 之后，仍然可以通过 v1.1 版本的 tidb-cluster chart 升级和管理 TiDB 集群。

## tidb-cluster chart 管理的组件或者功能切换到 v1.1 完整支持的方式

虽然 TiDB Operator v1.1 版本仍然支持用户使用 Helm 和 tidb-cluster chart 管理集群，但是由于 tidb-cluster chart 不再新增功能，用户可以自己为 tidb-cluster chart 贡献新功能或者切换到 TiDB Operator v1.1 完整支持的方式。

下面介绍如何将 tidb-cluster chart 管理的组件或者功能切换到 v1.1 完整支持的方式。

### Discovery

Discovery 服务直接由 TiDB Operator 内部生成，不再需要用户做任何配置。

### PD、TiDB、TiKV

在 tidb-cluster chart 中，PD、TiDB、TiKV 配置由 Helm 渲染成 ConfigMap，从 TiDB Operator v1.1 开始，PD、TiDB、TiKV 配置也可以直接在 TiDBCluster CR 中配置，具体配置方法可以参考 [通过 TidbCluster 配置 TiDB 集群](configure-a-tidb-cluster.md)。

> **注意：**
>
> 由于 TiDB Operator 渲染配置的方式和 Helm 渲染配置的方式不同，从 tidb-cluster chart values.yaml 中的配置迁移到 CR 中的配置，会引起对应组件滚动升级。

### Monitor

可以参考 [TiDB 集群的监控与告警](monitor-a-tidb-cluster.md)创建 TidbMonitor CR，管理 Monitor 组件。

> **注意：**
>
> * TidbMonitor CR 中的 `metadata.name` 需要和集群中 TidbCluster CR 的名字保持一致。
> * 由于 TiDB Operator 渲染资源的方式和 Helm 渲染资源的方式不同，从 tidb-cluster chart values.yaml 中的配置迁移到 TidbMonitor CR，会引起 Monitor 组件滚动升级。

### Initializer

- 如果在升级到 TiDB Operator v1.1 之前，初始化 Job 已经执行，初始化 Job 不需要从 tidb-cluster chart 中迁移到 TidbInitializer CR。
- 如果在升级到 TiDB Operator v1.1 之前，没有执行过初始化 Job，也没有修改过 TiDB 服务 root 用户的密码，升级到 TiDB Operator v1.1 之后，需要执行初始化，可以参考 [Kubernetes 上的集群初始化配置](initialize-a-cluster.md)进行配置。

### Pump

升级到 TiDB Operator v1.1 之后，可以修改 TidbCluster CR，添加 Pump 相关配置，通过 TidbCluster CR 管理 Pump 组件：

``` yaml
spec
  ...
  pump:
    baseImage: pingcap/tidb-binlog
    version: v5.2.0
    replicas: 1
    storageClassName: local-storage
    requests:
      storage: 30Gi
    schedulerName: default-scheduler
    config:
      addr: 0.0.0.0:8250
      gc: 7
      heartbeat-interval: 2
```

按照集群实际情况修改 `version`、`replicas`、`storageClassName`、`requests.storage` 等配置。

> **注意：**
>
> 由于 TiDB Operator 渲染资源的方式和 Helm 渲染资源的方式不同，从 tidb-cluster chart values.yaml 中的配置迁移到 TidbCluster CR，会引起 Pump 组件滚动升级。

### Scheduled Backup

升级到 TiDB Operator v1.1 之后，可以通过 BackupSchedule CR 配置定时全量备份：

- 如果 TiDB 集群版本 < v3.1，可以参考 [Dumpling 定时全量备份](backup-to-s3.md#定时全量备份)
- 如果 TiDB 集群版本 >= v3.1，可以参考 [BR 定时全量备份](backup-to-aws-s3-using-br.md#定时全量备份)

> **注意：**
>
> * BackupSchedule CR Dumpling 方式目前只支持备份到 s3、gcs，BR 方式只支持备份到 s3，如果升级之前的定时全量备份是备份到本地 PVC，则升级后不能切换到 CR 方式管理。
> * 如果切换到 CR 方式管理，请删除原有定时全量备份的 Cronjob，以防止重复备份。

### Drainer

- 如果在升级到 TiDB Operator v1.1 之前，没有部署 Drainer，现在需要新部署，可以参考 [Drainer 部署](deploy-tidb-binlog.md#部署-drainer)。
- 如果在升级到 TiDB Operator v1.1 之前，已经通过 `tidb-drainer` chart 部署 Drainer，继续用 `tidb-drainer` chart 管理。
- 如果在升级到 TiDB Operator v1.1 之前，已经通过 `tidb-cluster` chart 部署 Drainer，建议直接用 kubectl 管理。

### TiKV Importer

- 如果在升级到 TiDB Operator v1.1 之前，没有部署 TiKV Importer，现在需要新部署，可以参考 [TiKV Importer 部署](restore-data-using-tidb-lightning.md#部署-tikv-importer)。
- 如果在升级到 TiDB Operator v1.1 之前，已经部署 TiKV Importer，建议直接用 kubectl 管理。

## 其他由 chart 管理的组件或者功能切换到 v1.1 支持的方式

### Ad-hoc 备份

升级到 TiDB Operator v1.1 之后，可以通过 Backup CR 进行备份。Dumpling 方式只支持全量备份，BR 方式支持全量备份与增量备份：

- 如果 TiDB 集群版本 < v3.1，可以参考 [Dumpling Ad-hoc 全量备份](backup-to-s3.md#ad-hoc-全量备份)
- 如果 TiDB 集群版本 >= v3.1，可以参考 [BR Ad-hoc 备份](backup-to-aws-s3-using-br.md#ad-hoc-备份)

> **注意：**
>
> Backup CR Dumpling 方式和 BR 方式目前只支持备份到 S3、GCS。如果升级之前的 Ad-hoc 全量备份是备份到本地 PVC，则不能切换到 CR 方式管理。

### 备份恢复

升级到 TiDB Operator v1.1 之后，可以通过 Restore CR 进行备份恢复：

- 如果 TiDB 集群版本 < v3.1，可以参考 [使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据](restore-from-s3.md)。
- 如果 TiDB 集群版本 >= v3.1，可以参考 [使用 BR 恢复 S3 兼容存储上的备份数据](restore-from-aws-s3-using-br.md)。

> **注意：**
>
> Restore CR Lightning 方式和 BR 方式 目前只支持从 S3、GCS 获取备份数据进行恢复。如果需要从本地 PVC 获取备份数据进行恢复，则不能切换到 CR 方式管理。
