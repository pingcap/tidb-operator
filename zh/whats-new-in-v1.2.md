---
title: What's New in TiDB Operator 1.2
---

# What's New in TiDB Operator 1.2

TiDB Operator 1.2 引入了以下关键特性，从扩展性、易用性、安全性等方面帮助你更轻松地管理 TiDB 集群及其周边工具。

## 扩展性

- 支持为 TidbCluster 中的 Pod 与 service 自定义 label 及 annotation
- 支持为所有 TiDB 组件设置 podSecurityContext、topologySpreadConstraints
- 支持对 [Pump](https://docs.pingcap.com/zh/tidb/stable/tidb-binlog-overview#pump) 的完整生命周期管理
- TidbMonitor 支持[监控多个 TidbCluster](monitor-a-tidb-cluster.md#多集群监控)
- TidbMonitor 支持 remotewrite
- TidbMonitor 支持[配置 Thanos sidecar](aggregate-multiple-cluster-monitor-data.md)
- TidbMonitor 管理资源从 Deployment 变为 StatefulSet，以支持更多灵活性
- 支持在[仅有 namespace 权限时部署 TiDB Operator](deploy-tidb-operator.md#自定义部署-tidb-operator)
- 支持[部署多套 TiDB Operator](deploy-multiple-tidb-operator.md) 分别管理不同的 TiDB 集群

## 易用性

- 新增 [DMCluster CR 用于管理 DM 2.0](deploy-tidb-dm.md)
- 支持为 TiKV、TiFlash 自定义 Store 标签
- 支持为 [TiDB slow log 自定义存储](configure-a-tidb-cluster.md#配置-tidb-慢查询日志持久卷)
- TiDB Lightning chart [支持 local backend、持久化 checkpoint](restore-data-using-tidb-lightning.md)

## 安全性

- [TiDB Lightning chart 和 TiKV Importer chart 支持配置 TLS](enable-tls-between-components.md)

## 实验性特性

- [跨多个 Kubernetes 集群部署一个 TiDB 集群](deploy-tidb-cluster-across-multiple-kubernetes.md)
