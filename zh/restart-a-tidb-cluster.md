---
title: 重启 Kubernetes 上的 TiDB 集群
summary: 了解如何重启 Kubernetes 集群上的 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restart-a-tidb-cluster/']
---

# 重启 Kubernetes 上的 TiDB 集群

在使用 TiDB 集群的过程中，如果你发现某个 Pod 存在内存泄漏等问题，需要对集群进行重启，本文描述了如何优雅滚动重启 TiDB 集群内某个组件的所有 Pod，或优雅重启单个 TiKV Pod。

> **警告：**
>
> 在生产环境中，未经过优雅重启而手动删除某个 TiDB 集群 Pod 节点是一件极其危险的事情，虽然 StatefulSet 控制器会将 Pod 节点再次拉起，但这依旧可能会引起部分访问 TiDB 集群的请求失败。

## 优雅滚动重启 TiDB 集群组件的所有 Pod

[在标准 Kubernetes 上部署 TiDB 集群](deploy-on-general-kubernetes.md)之后，通过 `kubectl edit tc ${name} -n ${namespace}` 修改集群配置，为期望优雅滚动重启的 TiDB 集群组件 Spec 添加 annotation `tidb.pingcap.com/restartedAt`，Value 设置为当前时间。以下示例中，为组件 `pd`、`tikv`、`tidb` 都设置了 annotation，表示将优雅滚动重启以上三个 TiDB 集群组件的所有 Pod。可以根据实际情况，只为某个组件设置 annotation。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  version: v5.2.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
  tikv:
    baseImage: pingcap/tikv
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
```

## 优雅重启单个 TiKV Pod

从 v1.2.5 起，TiDB Operator 支持给 TiKV Pod 添加 annotation 来触发优雅重启单个 TiKV Pod。

添加一个 key 为 `tidb.pingcap.com/evict-leader` 的 annotation，触发优雅重启：

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} annotate pod ${tikv_pod_name} tidb.pingcap.com/evict-leader="delete-pod"
```

当 TiKV region leader 数掉到 0 时，根据 annotation 的不同值，TiDB Operator 会采取不同的行为。合法的 annotation 值如下：

- `none`: 无对应行为。
- `delete-pod`: 删除 Pod，TiDB Operator 的具体行为如下：
    1. 调用 PD API，为对应 TiKV store 添加 evict-leader-scheduler。
    2. 当 TiKV region leader 数掉到 0 时，删除 Pod 并重建 Pod。
    3. 当新的 Pod Ready 后，调用 PD API 删除对应 TiKV store 的 evict-leader-scheduler。
