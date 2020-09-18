---
title: Kubernetes 上的 TiDB 集群故障自动转移
summary: 介绍 Kubernetes 上的 TiDB 集群故障自动转移的功能。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/use-auto-failover/']
---

# Kubernetes 上的 TiDB 集群故障自动转移

故障自动转移是指在 TiDB 集群的某些节点出现故障时，TiDB Operator 会自动添加一个节点，保证 TiDB 集群的高可用，类似于 K8s 的 `Deployment` 行为。

由于 TiDB Operator 基于 `StatefulSet` 来管理 Pod，但 `StatefulSet` 在某些 Pod 发生故障时不会自动创建新节点来替换旧节点，所以，TiDB Operator 扩展了 `StatefulSet` 的这种行为，添加了 Auto Failover 功能。

Auto Failover 功能在 TiDB Operator 中默认开启。部署 TiDB Operator 时，可通过设置 `charts/tidb-operator/values.yaml` 文件的 `controllerManager.autoFailover` 为 `false` 关闭该功能：

```yaml
controllerManager:
 serviceAccount: tidb-controller-manager
 logLevel: 2
 replicas: 1
 resources:
   limits:
     cpu: 250m
     memory: 150Mi
   requests:
     cpu: 80m
     memory: 50Mi
 # autoFailover is whether tidb-operator should auto failover when failure occurs
 autoFailover: true
 # pd failover period default(5m)
 pdFailoverPeriod: 5m
 # tikv failover period default(5m)
 tikvFailoverPeriod: 5m
 # tidb failover period default(5m)
 tidbFailoverPeriod: 5m
 # tiflash failover period default(5m)
 tiflashFailoverPeriod: 5m
```

`pdFailoverPeriod`、`tikvFailoverPeriod`、`tiflashFailoverPeriod` 和 `tidbFailoverPeriod` 默认均为 5 分钟，它们的含义是在确认实例故障后的等待超时时间，超过这个时间后，TiDB Operator 就开始做自动的故障转移。

## 实现原理

TiDB 集群有 PD、TiKV 和 TiDB 三个组件，它们的故障转移策略有所不同，本节将详细介绍这三种策略。

### PD 故障转移策略

假设 PD 集群有 3 个节点，如果一个 PD 节点挂掉超过 5 分钟（`pdFailoverPeriod` 可配置），TiDB Operator 首先会将这个 PD 节点下线，然后再添加一个新的 PD 节点。此时会有 4 个 Pod 同时存在，待挂掉的 PD 节点恢复后，TiDB Operator 会将新启动的节点删除掉，恢复成原来的 3 个节点。

### TiKV 故障转移策略

当一个 TiKV Pod 无法正常工作后，该 Pod 对应的 Store 状态会变为 `Disconnected`，30 分钟（通过 `pd.config` 文件中 `[schedule]` 部分的 `max-store-down-time = "30m"` 来配置）后会变成 `Down` 状态，TiDB Operator 会在此基础上再等待 5 分钟（`tikvFailoverPeriod` 可配置），如果该 TiKV Pod 仍不能恢复，就会新起一个 TiKV Pod。异常的 TiKV Pod 恢复后，考虑到自动缩容会引起数据的迁移，TiDB Operator 不会自动缩容新起的 Pod。

如果**所有**异常的 TiKV Pod 都已经恢复，这时如果需要缩容新起的 Pod，请参考以下步骤：

配置 `spec.tikv.recoverFailover: true` (从 TiDB Operator v1.1.5 开始支持)：

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator 会自动将新起的 TiKV Pod 缩容，请在集群缩容完成后，配置 `spec.tikv.recoverFailover: false`，避免下次发生故障转移并恢复后自动缩容。

### TiDB 故障转移策略

假设 TiDB 集群有 3 个节点，TiDB 的故障转移策略跟 Kubernetes 中的 `Deployment` 的是一致的。如果一个 TiDB 节点挂掉超过 5 分钟（`tidbFailoverPeriod` 可配置），TiDB Operator 会添加一个新的 TiDB 节点。此时会有 4 个 Pod 同时存在，待挂掉的 TiDB 节点恢复后，TiDB Operator 会将新启动的节点删除掉，恢复成原来的 3 个节点。

### TiFlash 故障转移策略

当一个 TiFlash Pod 无法正常工作后，该 Pod 对应的 Store 状态会变为 `Disconnected`，30 分钟（通过 `pd.config` 文件中 `[schedule]` 部分的 `max-store-down-time = "30m"` 来配置）后会变成 `Down` 状态，TiDB Operator 会在此基础上再等待 5 分钟（`tiflashFailoverPeriod` 可配置），如果该 TiFlash Pod 仍不能恢复，就会新起一个 TiFlash Pod。异常的 TiFlash Pod 恢复后，考虑到自动缩容会引起数据的迁移，TiDB Operator 不会自动缩容新起的 Pod。

如果**所有**异常的 TiFlash Pod 都已经恢复，这时如果需要缩容新起的 Pod，请参考以下步骤：

配置 `spec.tiflash.recoverFailover: true` (从 TiDB Operator v1.1.5 开始支持)：

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator 会自动将新起的 TiFlash Pod 缩容，请在集群缩容完成后，配置 `spec.tiflash.recoverFailover: false`，避免下次发生故障转移并恢复后自动缩容。
