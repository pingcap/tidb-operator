---
title: 使用 PD Recover 恢复 PD 集群
summary: 了解如何使用 PD Recover 恢复 PD 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/pd-recover/']
---

# 使用 PD Recover 恢复 PD 集群

[PD Recover](https://pingcap.com/docs-cn/stable/reference/tools/pd-recover) 是对 PD 进行灾难性恢复的工具，用于恢复无法正常启动或服务的 PD 集群。

## 下载 PD Recover

1. 下载 TiDB 官方安装包：

    {{< copyable "shell-regular" >}}

    ```shell
    wget https://download.pingcap.org/tidb-${version}-linux-amd64.tar.gz
    ```

    `${version}` 是 TiDB 集群版本，例如，`v5.2.1`。

2. 解压安装包：

    {{< copyable "shell-regular" >}}

    ```shell
    tar -xzf tidb-${version}-linux-amd64.tar.gz
    ```

    `pd-recover` 在 `tidb-${version}-linux-amd64/bin` 目录下。

## 使用 PD Recover 恢复 PD 集群

本小节详细介绍如何使用 PD Recover 来恢复 PD 集群。

### 获取 Cluster ID

{{< copyable "shell-regular" >}}

```shell
kubectl get tc ${cluster_name} -n ${namespace} -o='go-template={{.status.clusterID}}{{"\n"}}'
```

示例：

```
kubectl get tc test -n test -o='go-template={{.status.clusterID}}{{"\n"}}'
6821434242797747735
```

### 获取 Alloc ID

使用 `pd-recover` 恢复 PD 集群时，需要指定 `alloc-id`。`alloc-id` 的值需要是一个比当前已经分配的最大的 `Alloc ID` 更大的值。

1. 参考[访问 Prometheus 监控数据](monitor-a-tidb-cluster.md#访问-prometheus-监控数据)打开 TiDB 集群的 Prometheus 访问页面。

2. 在输入框中输入 `pd_cluster_id` 并点击 `Execute` 按钮查询数据，获取查询结果中的最大值。

3. 将查询结果中的最大值乘以 `100`，作为使用 `pd-recover` 时指定的 `alloc-id`。

### 恢复 PD 集群 Pod

1. 删除 PD 集群 Pod。

    通过如下命令设置 `spec.pd.replicas` 为 `0`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

    由于此时 PD 集群异常，TiDB Operator 无法将上面的改动同步到 PD StatefulSet，所以需要通过如下命令设置 PD StatefulSet `spec.replicas` 为 `0`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit sts ${cluster_name}-pd -n ${namespace}
    ```

    通过如下命令确认 PD Pod 已经被删除：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod -n ${namespace}
    ```

2. 确认所有 PD Pod 已经被删除后，通过如下命令删除 PD Pod 绑定的 PVC：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete pvc -l app.kubernetes.io/component=pd,app.kubernetes.io/instance=${cluster_name} -n ${namespace}
    ```

3. PVC 删除完成后，扩容 PD 集群至一个 Pod。

    通过如下命令设置 `spec.pd.replicas` 为 `1`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

    由于此时 PD 集群异常，TiDB Operator 无法将上面的改动同步到 PD StatefulSet，所以需要通过如下命令设置 PD StatefulSet `spec.replicas` 为 `1`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit sts ${cluster_name}-pd -n ${namespace}
    ```

    通过如下命令确认 PD Pod 已经启动：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod -n ${namespace}
    ```

### 使用 PD Recover 恢复集群

1. 通过 `port-forward` 暴露 PD 服务：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379
    ```

2. 打开一个**新**终端标签或窗口，进入到 `pd-recover` 所在的目录，使用 `pd-recover` 恢复 PD 集群：

    {{< copyable "shell-regular" >}}

    ```shell
    ./pd-recover -endpoints http://127.0.0.1:2379 -cluster-id ${cluster_id} -alloc-id ${alloc_id}
    ```

    `${cluster_id}` 是[获取 Cluster ID](#获取-cluster-id) 步骤中获取的 Cluster ID，`${alloc_id}` 是[获取 Alloc ID](#获取-alloc-id) 步骤中获取的 `pd_cluster_id` 的最大值再乘以 `100`。

    `pd-recover` 命令执行成功后，会打印如下输出：

    ```shell
    recover success! please restart the PD cluster
    ```

3. 回到 `port-forward` 命令所在窗口，按 <kbd>Ctrl</kbd>+<kbd>C</kbd> 停止并退出。

### 重启 PD Pod

1. 删除 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete pod ${cluster_name}-pd-0 -n ${namespace}
    ```

2. Pod 正常启动后，通过 `port-forward` 暴露 PD 服务：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379
    ```

3. 打开一个**新**终端标签或窗口，通过如下命令确认 Cluster ID 为[获取 Cluster ID](#获取-cluster-id) 步骤中获取的 Cluster ID：

    {{< copyable "shell-regular" >}}

    ```shell
    curl 127.0.0.1:2379/pd/api/v1/cluster
    ```

4. 回到 `port-forward` 命令所在窗口，按 <kbd>Ctrl</kbd>+<kbd>C</kbd> 停止并退出。

### 扩容 PD 集群

通过如下命令设置 `spec.pd.replicas` 为期望的 Pod 数量：

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc ${cluster_name} -n ${namespace}
```

### 重启 TiDB 和 TiKV

{{< copyable "shell-regular" >}}

```shell
kubectl delete pod -l app.kubernetes.io/component=tidb,app.kubernetes.io/instance=${cluster_name} -n ${namespace} && 
kubectl delete pod -l app.kubernetes.io/component=tikv,app.kubernetes.io/instance=${cluster_name} -n ${namespace}
```
