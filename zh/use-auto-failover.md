---
title: Kubernetes 上的 TiDB 集群故障自动转移
summary: 介绍 Kubernetes 上的 TiDB 集群故障自动转移的功能。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/use-auto-failover/']
---

# Kubernetes 上的 TiDB 集群故障自动转移

TiDB Operator 基于 [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) 管理 Pod 的部署和扩缩容，但 `StatefulSet` 在某些 Pod 或者节点发生故障时不会自动创建新 Pod 来替换旧 Pod。为解决此问题，TiDB Operator 支持通过自动扩容 Pod 实现故障自动转移功能。

## 配置故障自动转移

故障自动转移功能在 TiDB Operator 中默认开启。

部署 TiDB Operator 时，可以在 `charts/tidb-operator/values.yaml` 文件中配置 TiDB 集群中 PD、TiKV、TiDB 和 TiFlash 组件故障转移的等待超时时间。示例如下：

```yaml
controllerManager:
 ...
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

其中，`pdFailoverPeriod`、`tikvFailoverPeriod`、`tiflashFailoverPeriod` 和 `tidbFailoverPeriod` 代表在确认实例故障后的等待超时时间，默认均为 5 分钟。超过这个时间后，TiDB Operator 就开始做故障自动转移。

另外，在配置 TiDB 集群时，可以通过 `spec.${component}.maxFailoverCount` 指定 TiDB Operator 在各组件故障自动转移时能扩容的 Pod 数量阈值，详情请参考 [TiDB 组件配置文档](configure-a-tidb-cluster.md#配置-pdtidbtikvtiflash-故障自动转移阈值)。

> **注意：**
> 
> 如果集群中没有足够的资源以供 TiDB Operator 扩容新 Pod，则扩容出的 Pod 会处于 Pending 状态。

## 实现原理

TiDB 集群包括 PD、TiKV、TiDB、TiFlash、TiCDC 和 Pump 六个组件。目前 TiCDC 和 Pump 并不支持故障自动转移，PD、TiKV、TiDB 和 TiFlash 的故障转移策略有所不同，本节将详细介绍这几种策略。

### PD 故障转移策略

TiDB Operator 通过 `pd/health` PD API 获取 PD members 健康状况，并记录到 TidbCluster CR 的 `.status.pd.members` 字段中。

以一个有 3 个 Pod 的 PD 集群为例，如果其中一个 Pod 不健康超过 5 分钟（`pdFailoverPeriod` 可配置），TiDB Operator 将自动进行以下操作：

1. TiDB Operator 将此 Pod 信息记录到 TidbCluster CR 的 `.status.pd.failureMembers` 字段中。
2. TiDB Operator 将此 Pod 下线：TiDB Operator 调用 PD API 将此 Pod 从 member 列表中删除，然后删掉 Pod 及其 PVC。
3. StatefulSet controller 会重新创建此 Pod 并以新的 member 身份加入集群。
4. 在计算 PD StatefulSet 的 Replicas 时，TiDB Operator 会将已经被删除过的 `.status.pd.failureMembers` 考虑在内，因此会扩容一个新的 Pod。此时将有 4 个 Pod 同时存在。

当原来集群中所有不健康的 Pod 都恢复正常时，TiDB Operator 会将新扩容的 Pod 自动缩容掉，恢复成原来的 Pod 数量。

> **注意：**
>
> - TiDB Operator 会为每个 PD 集群最多扩容 `spec.pd.maxFailoverCount` (默认 `3`) 个 Pod，超过这个阈值后不会再进行故障转移。
> - 如果 PD 集群多数 member 已经不健康，导致 PD 集群不可用，TiDB Operator 不会为这个 PD 集群进行故障自动转移。

### TiDB 故障转移策略

TiDB Operator 通过访问每个 TiDB Pod 的 `/status` 接口确认 Pod 健康状况，并记录到 TidbCluster CR 的 `.status.tidb.members` 字段中。

以一个有 3 个 Pod 的 TiDB 集群为例，如果一个 Pod 不健康超过 5 分钟（`tidbFailoverPeriod` 可配置），TiDB Operator 将自动进行以下操作：

1. TiDB Operator 将此 Pod 信息记录到 TidbCluster CR 的 `.status.tidb.failureMembers` 字段中。
2. 在计算 TiDB StatefulSet 的 Replicas 时，TiDB Operator 会将 `.status.tidb.failureMembers` 考虑在内，因此会扩容一个新的 Pod。此时会有 4 个 Pod 同时存在。

当原来集群中不健康的 Pod 恢复正常时，TiDB Operator 会将新扩容的 Pod 缩容掉，恢复成原来的 3 个 Pod。

> **注意：**
>
> TiDB Operator 会为每个 TiDB 集群最多扩容 `spec.tidb.maxFailoverCount` (默认 `3`) 个 Pod，超过这个阈值后不会再进行故障转移。

### TiKV 故障转移策略

TiDB Operator 通过访问 PD API 获取 TiKV store 健康状况，并记录到 TidbCluster CR 的 `.status.tikv.stores` 字段中。

以一个有 3 个 Pod 的 TiKV 集群为例，当一个 TiKV Pod 无法正常工作时，该 Pod 对应的 Store 状态会变为 `Disconnected`。默认 30 分钟（可以通过 `pd.config` 中 `[schedule]` 部分的 `max-store-down-time = "30m"` 来修改）后会变成 `Down` 状态，然后 TiDB Operator 将自动进行以下操作：

1. 在此基础上再等待 5 分钟（可以通过 `tikvFailoverPeriod` 配置），如果此 TiKV Pod 仍未恢复，TiDB Operator 会将此 Pod 信息记录到 TidbCluster CR 的 `.status.tikv.failureStores` 字段中。
2. 在计算 TiKV StatefulSet 的 Replicas 时，TiDB Operator 会将 `.status.tikv.failureStores` 考虑在内，因此会扩容一个新的 Pod。此时会有 4 个 Pod 同时存在。

当原来集群中不健康的 Pod 恢复正常时，考虑到缩容 Pod 需要迁移数据，可能会对集群性能有一定影响，TiDB Operator 并**不会**将新扩容的 Pod 缩容掉，而是继续保持 4 个 Pod。

> **注意：**
>
> TiDB Operator 会为每个 TiKV 集群最多扩容 `spec.tikv.maxFailoverCount` (默认 `3`) 个 Pod，超过这个阈值后不会再进行故障转移。

如果**所有**异常的 TiKV Pod 都已经恢复，这时如果需要缩容新起的 Pod，请参考以下步骤：

配置 `spec.tikv.recoverFailover: true` (从 TiDB Operator v1.1.5 开始支持)：

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator 会自动将新起的 TiKV Pod 缩容，请在集群缩容完成后，配置 `spec.tikv.recoverFailover: false`，避免下次发生故障转移并恢复后自动缩容。

### TiFlash 故障转移策略

TiDB Operator 通过访问 PD API 获取 TiFlash store 健康状况，并记录到 TidbCluster CR 的 `.status.tiflash.stores` 字段中。

以一个有 3 个 Pod 的 TiFlash 集群为例，当一个 TiFlash Pod 无法正常工作时，该 Pod 对应的 Store 状态会变为 `Disconnected`。默认 30 分钟（可以通过 `pd.config` 中 `[schedule]` 部分的 `max-store-down-time = "30m"` 来修改）后会变成 `Down` 状态，然后 TiDB Operator 将自动进行以下操作：

1. 在此基础上再等待 5 分钟（`tiflashFailoverPeriod` 可配置），如果此 TiFlash Pod 仍未恢复，TiDB Operator 会将此 Pod 信息记录到 TidbCluster CR 的 `.status.tiflash.failureStores` 字段中。
2. 在计算 TiFlash StatefulSet 的 Replicas 时，TiDB Operator 会将 `.status.tiflash.failureStores` 考虑在内，因此会扩容一个新的 Pod。此时会有 4 个 Pod 同时存在。

当原来集群中不健康的 Pod 恢复正常时，考虑到缩容 Pod 需要迁移数据，可能会对集群性能有一定影响，TiDB Operator 并**不会**将新扩容的 Pod 缩容掉，而是继续保持 4 个 Pod。

> **注意：**
>
> TiDB Operator 会为每个 TiFlash 集群最多扩容 `spec.tiflash.maxFailoverCount` (默认 `3`) 个 Pod，超过这个阈值后不会再进行故障转移。

如果**所有**异常的 TiFlash Pod 都已经恢复，这时如果需要缩容新起的 Pod，请参考以下步骤：

配置 `spec.tiflash.recoverFailover: true` (从 TiDB Operator v1.1.5 开始支持)：

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator 会自动将新起的 TiFlash Pod 缩容，请在集群缩容完成后，配置 `spec.tiflash.recoverFailover: false`，避免下次发生故障转移并恢复后自动缩容。

## 关闭故障自动转移

你可以在集群或组件级别关闭故障自动转移功能。

- 如需在集群级别关闭故障自动转移功能，在部署 TiDB Operator 时，请将 `charts/tidb-operator/values.yaml` 文件的 `controllerManager.autoFailover` 字段值配置为 `false`。示例如下：

    ```yaml
    controllerManager:
    ...
    # autoFailover is whether tidb-operator should auto failover when failure occurs
    autoFailover: false
    ```

- 如需在组件级别关闭故障自动转移功能，在创建 TiDB 集群时，请将 TidbCluster CR 中对应组件的 `spec.${component}.maxFailoverCount` 字段值配置为 `0`。
