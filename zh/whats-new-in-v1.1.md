---
title: What's New in TiDB Operator 1.1
aliases: ['/docs-cn/tidb-in-kubernetes/dev/whats-new-in-v1.1/']
---

# What's New in TiDB Operator 1.1

TiDB Operator 1.1 在 1.0 基础上新增 TiDB 4.0 功能特性支持，TiKV 数据加密、TLS 证书配置等。新增 TiFlash、TiCDC 新组件部署支持，同时在易用性上做了许多改进，提供与 Kubernetes 原生资源一致的用户体验。以下是主要变化：

## 扩展性

- TidbCluster CR 支持部署管理 PD Discovery 组件，可完全替代 tidb-cluster chart 管理 TiDB 集群
- TidbCluster CR 新增 Pump、TiFlash、TiCDC、Dashboard 支持
- 新增可选的[准入控制器](enable-admission-webhook.md)改进升级、扩缩容体验，并提供灰度发布功能
- tidb-scheduler 支持任意维度的 HA 调度和调度器 preemption
- 使用 tikv-importer chart [部署、管理 TiKV Importer](restore-data-using-tidb-lightning.md#部署-tikv-importer)

## 易用性

- 新增 TidbMonitor CR 用于部署集群监控
- 新增 TidbInitializer CR 用于初始化集群
- 新增 Backup、BackupSchedule、Restore CR 用于备份恢复集群。备份、恢复支持 S3 和 GCS
- [优雅重启 TiDB 集群组件](restart-a-tidb-cluster.md)

## 安全性

- 支持 TiDB 集群各组件及客户端 TLS 证书配置
- 支持 TiKV 数据存储加密

## 实验性特性

- 新增 `TidbClusterAutoScaler` 实现[集群自动伸缩功能](enable-tidb-cluster-auto-scaling.md)（开启 `AutoScaling` 特性开关后使用）
- 新增可选的[增强型 StatefulSet 控制器](advanced-statefulset.md)，提供对指定 Pod 进行删除的功能（开启 `AdvancedStatefulSet` 特性开关后使用）

完整发布日志参见 [1.1 CHANGE LOG](https://github.com/pingcap/tidb-operator/blob/master/CHANGELOG-1.1.md)。

TiDB Operator 在 Kubernetes 上部署参见[安装文档](deploy-tidb-operator.md)，CRD 文档参见 [API References](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)。
