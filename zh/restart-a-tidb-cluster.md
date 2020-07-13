---
title: 重启 Kubernetes 上的 TiDB 集群
summary: 了解如何重启 Kubernetes 集群上的 TiDB 集群。
---

# 重启 Kubernetes 上的 TiDB 集群

在使用 TiDB 集群的过程中，如果你发现某个 Pod 存在内存泄漏等问题，需要对集群进行重启，本文描述了如何优雅滚动重启 TiDB 集群内某个组件的所有 Pod 或通过优雅重启指令来将 TiDB 集群内某个 Pod 优雅下线然后再进行重新启动。

> **警告：**
>
> 在生产环境中，未经过优雅重启而手动删除某个 TiDB 集群 Pod 节点是一件极其危险的事情，虽然 StatefulSet 控制器会将 Pod 节点再次拉起，但这依旧可能会引起部分访问 TiDB 集群的请求失败。

## 优雅滚动重启 TiDB 集群组件的所有 Pod 节点

1. 参考[在标准 Kubernetes 上部署 TiDB 集群](deploy-on-general-kubernetes.md)，修改 `${cluster_name}/tidb-cluster.yaml` 文件，为期望优雅滚动重启的 TiDB 集群组件 Spec 添加 annotation `tidb.pingcap.com/restartedAt`，Value 设置为当前时间。以下示例中，为组件 `pd`，`tikv`，`tidb` 都设置了 annotation，表示将优雅滚动重启以上三个 TiDB 集群组件的所有 Pod。可以根据实际情况，只为某个组件设置 annotation。

    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      name: basic
    spec:
      version: v3.0.8
      timezone: UTC
      pvReclaimPolicy: Delete
      pd:
        baseImage: pingcap/pd
        replicas: 3
        requests:
          storage: "1Gi"
        config: {}
        annotations:
          tidb.pingcap.com/restartedAt: "202004201200"
      tikv:
        baseImage: pingcap/tikv
        replicas: 3
        requests:
          storage: "1Gi"
        config: {}
        annotations:
          tidb.pingcap.com/restartedAt: "202004201200"
      tidb:
        baseImage: pingcap/tidb
        replicas: 2
        service:
          type: ClusterIP
        config: {}
        annotations:
          tidb.pingcap.com/restartedAt: "202004201200"
    ```

2. 应用更新

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ${cluster_name} -n ${namespace}
    ```

## 优雅重启 TiDB 集群组件的单个 Pod

### 开启相关设置

开启优雅下线功能，需要打开 Webhook 相关设置。默认情况下 Webhook 相关配置是关闭的，你需要手动开启:
    
1. 修改 Operator 的 `values.yaml`

    开启 Operator Webhook 特性:

    ```yaml
    admissionWebhook:
      create: true
    ```

    关于 Operator Webhook 详情，请参考[开启 TiDB Operator 准入控制器](enable-admission-webhook.md)

2. 安装/更新 TiDB Operator

    修改完 `values.yaml` 文件中的上述配置项以后，进行 TiDB Operator 部署或者更新。安装与更新 TiDB Operator 请参考[在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md)。 

### 使用 annotate 标记目标 Pod 节点

我们通过 `kubectl annotate` 的方式来标记目标 TiDB 集群 Pod 节点组件，当 `annotate` 标记完成以后，TiDB Operator 会自动进行 Pod 节点的优雅下线并重启。你可以通过以下方式来进行标记:

{{< copyable "shell-regular" >}}

```sh
kubectl annotate ${pod_name} -n ${namespace} tidb.pingcap.com/pod-defer-deleting=true
```
