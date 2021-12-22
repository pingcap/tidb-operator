---
title: TiDB Operator 架构
summary: 了解 TiDB Operator 架构及其工作原理。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/architecture/']
---

# TiDB Operator 架构

本文档介绍 TiDB Operator 的架构及其工作原理。

## 架构

下图是 TiDB Operator 的架构概览。

![TiDB Operator Overview](/media/tidb-operator-overview-1.1.png)

其中，`TidbCluster`、`TidbMonitor`、`TidbInitializer`、`Backup`、`Restore`、`BackupSchedule`、`TidbClusterAutoScaler` 是由 CRD（`CustomResourceDefinition`）定义的自定义资源：

* `TidbCluster` 用于描述用户期望的 TiDB 集群
* `TidbMonitor` 用于描述用户期望的 TiDB 集群监控组件
* `TidbInitializer` 用于描述用户期望的 TiDB 集群初始化 Job
* `Backup` 用于描述用户期望的 TiDB 集群备份
* `Restore` 用于描述用户期望的 TiDB 集群恢复
* `BackupSchedule` 用于描述用户期望的 TiDB 集群周期性备份
* `TidbClusterAutoScaler` 用于描述用户期望的 TiDB 集群自动伸缩

TiDB 集群的编排和调度逻辑则由下列组件负责：

* `tidb-controller-manager` 是一组 Kubernetes 上的自定义控制器。这些控制器会不断对比 `TidbCluster` 对象中记录的期望状态与 TiDB 集群的实际状态，并调整 Kubernetes 中的资源以驱动 TiDB 集群满足期望状态，并根据其他 CR 完成相应的控制逻辑；
* `tidb-scheduler` 是一个 Kubernetes 调度器扩展，它为 Kubernetes 调度器注入 TiDB 集群特有的调度逻辑。
* `tidb-admission-webhook` 是一个 Kubernetes 动态准入控制器，完成 Pod、StatefulSet 等相关资源的修改、验证与运维。

> **注意：**
>
> `tidb-scheduler` 并不是必须使用，详情可以参考 [tidb-scheduler 与 default-scheduler](tidb-scheduler.md#tidb-scheduler-与-default-scheduler)。

## 流程解析

下图是 TiDB Operator 的控制流程解析。从 TiDB Operator v1.1 开始，TiDB 集群、监控、初始化、备份等组件，都通过 CR 进行部署、管理。

![TiDB Operator Control Flow](/media/tidb-operator-control-flow-1.1.png)

整体的控制流程如下：

1. 用户通过 kubectl 创建 `TidbCluster` 和其他 CR 对象，比如 `TidbMonitor` 等；
2. TiDB Operator 会 watch `TidbCluster` 以及其它相关对象，基于集群的实际状态不断调整 PD、TiKV、TiDB 或者 Monitor 等组件的 `StatefulSet`、`Deployment` 和 `Service` 等对象；
3. Kubernetes 的原生控制器根据 `StatefulSet`、`Deployment`、`Job` 等对象创建更新或删除对应的 `Pod`；
4. 如果 `TidbCluster` 中配置组件使用 `tidb-scheduler`，PD、TiKV、TiDB 的 `Pod` 声明中会指定使用 `tidb-scheduler` 调度器。在调度对应 `Pod` 时，`tidb-scheduler` 会应用 TiDB 的特定调度逻辑。

基于上述的声明式控制流程，TiDB Operator 能够自动进行集群节点健康检查和故障恢复。部署、升级、扩缩容等操作也可以通过修改 `TidbCluster` 对象声明“一键”完成。
