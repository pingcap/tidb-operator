---
title: 增强型 StatefulSet 控制器
summary: 介绍如何开启、使用增强型 StatefulSet 控制器
aliases: ['/docs-cn/tidb-in-kubernetes/dev/advanced-statefulset/']
---

# 增强型 StatefulSet 控制器

**特性状态**: Alpha

Kubernetes 内置 [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) 为 Pods 分配连续的序号。比如 3 个副本时，Pods 分别为 pod-0, pod-1, pod-2。扩缩容时，必须在尾部增加或删除 Pods。比如扩容到 4 个副本时，会新增 pod-3。缩容到 2 副本时，会删除 pod-2。

在使用本地存储时，Pods 与 Nodes 存储资源绑定，无法自由调度。若希望删除掉中间某个 Pod ，以便维护其所在的 Node 但并没有其他 Node 可以迁移时，或者某个 Pod 故障想直接删除，另起一个序号不一样的 Pod 时，无法通过内置 StatefulSet 实现。

[增强型 StatefulSet 控制器](https://github.com/pingcap/advanced-statefulset) 基于内置 StatefulSet 实现，新增了自由控制 Pods 序号的功能。本文介绍如何在 TiDB Operator 中使用。

## 开启

1. 载入 Advanced StatefulSet 的 CRD 文件：

    * Kubernetes 1.16 之前版本：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/advanced-statefulset-crd.v1beta1.yaml
        ```

    * Kubernetes 1.16 及之后版本:

        {{< copyable "shell-regular" >}}

        ```
        kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/advanced-statefulset-crd.v1.yaml
        ```

2. 在 TiDB Operator chart 的 `values.yaml` 中启用 `AdvancedStatefulSet` 特性：

    {{< copyable "" >}}

    ```yaml
    features:
    - AdvancedStatefulSet=true
    advancedStatefulset:
      create: true
    ```

    然后升级 TiDB Operator，具体可参考[升级 TiDB Operator 文档](upgrade-tidb-operator.md)。

> **注意：**
>
> TiDB Operator 通过开启 `AdvancedStatefulSet` 特性，会将当前 `StatefulSet` 对象转换成 `AdvancedStatefulSet` 对象。但是，TiDB Operator 不支持在关闭 `AdvancedStatefulSet` 特性后，自动从 `AdvancedStatefulSet` 转换为 Kubernetes 内置的 `StatefulSet` 对象。

## 使用

### 通过 kubectl 查看 AdvancedStatefulSet 对象

`AdvancedStatefulSet` 数据格式与 `StatefulSet` 完全一致，但以 CRD 方式实现，别名为 `asts` ，可通过以下方法查看命名空间下的对象。

{{< copyable "shell-regular" >}}

```shell
kubectl get -n ${namespace} asts
```

### 操作 TidbCluster 对象指定 pod 进行缩容

使用增强型 StatefulSet 时，在对 TidbCluster 进行缩容时，除了减少副本数，可同时通过配置 annotations 指定对 PD，TiDB 或 TiKV 组件下任意一个 Pod 进行缩容。

比如：

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: asts
spec:
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 4
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
```

上述配置会部署 4 个 TiKV 实例，分别为 basic-tikv-0，basic-tikv-1，...，basic-tikv-3。若想缩容掉 basic-tikv-1 需要修改 `spec.tikv.replicas` 为 3，同时配置以下 annotations:

{{< copyable "" >}}

```yaml
metadata:
  annotations:
    tikv.tidb.pingcap.com/delete-slots: '[1]'
```

> **注意：**
>
> 对 `replicas` 和 `delete-slots annotation` 的修改需在同一个操作中完成，不然控制器会根据修改一般的期望进行操作。

完整例子如下：

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  annotations:
    tikv.tidb.pingcap.com/delete-slots: '[1]'
  name: asts
spec:
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
```

支持的 annotations 为：

- `pd.tidb.pingcap.com/delete-slots`：指定 PD 组件需要删除的 Pod 序号。
- `tidb.tidb.pingcap.com/delete-slots`：指定 TiDB 组件需要删除的 Pod 序号。
- `tikv.tidb.pingcap.com/delete-slots`：指定 TiKV 组件需要删除的 Pod 序号。

其中 Annotation 值为 JSON 的整数数组，比如 `[0]`, `[0,1]`, `[1,3]` 等。

### 操作 TidbCluster 对象在指定位置进行扩容

对前面缩容进行反向操作，即可恢复 basic-tikv-1。

> **注意：**
>
> 同常规 StatefulSet 缩容一样，并不会主动删除 Pod 关联的 PVC，若想避免使用之前数据，在原位置处扩容之前，需主动删除之前关联的 PVC。

例子如下：

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  annotations:
    tikv.tidb.pingcap.com/delete-slots: '[]'
  name: asts
spec:
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 4
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
```

其中 delete-slots annotations 可留空，也可完全删除。
