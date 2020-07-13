---
title: Kubernetes 上的 TiDB 集群常见异常
summary: 介绍 TiDB 集群运行过程中常见异常以及处理办法。
---

# Kubernetes 上的 TiDB 集群常见异常

本文介绍 TiDB 集群运行过程中常见异常以及处理办法。

## TiKV Store 异常进入 Tombstone 状态

正常情况下，当 TiKV Pod 处于健康状态时（Pod 状态为 `Running`），对应的 TiKV Store 状态也是健康的（Store 状态为 `UP`）。但并发进行 TiKV 组件的扩容和缩容可能会导致部分 TiKV Store 异常并进入 Tombstone 状态。此时，可以按照以下步骤进行修复：

1. 查看 TiKV Store 状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get -n ${namespace} tidbcluster ${cluster_name} -ojson | jq '.status.tikv.stores'
    ```

2. 查看 TiKV Pod 运行状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get -n ${namespace} po -l app.kubernetes.io/component=tikv
    ```

3. 对比 Store 状态与 Pod 运行状态。假如某个 TiKV Pod 所对应的 Store 处于 `Offline` 状态，则表明该 Pod 的 Store 正在异常下线中。此时，可以通过下面的命令取消下线进程，进行恢复：

    1. 打开到 PD 服务的连接：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl port-forward -n ${namespace} svc/${cluster_name}-pd ${local_port}:2379 &>/tmp/portforward-pd.log &
        ```

    2. 上线对应 Store：

        {{< copyable "shell-regular" >}}

        ```shell
        curl -X POST http://127.0.0.1:2379/pd/api/v1/store/${store_id}/state?state=Up
        ```

4. 假如某个 TiKV Pod 所对应的 `lastHeartbeatTime` 最新的 Store 处于 `Tombstone` 状态 ，则表明异常下线已经完成。此时，需要重建 Pod 并绑定新的 PV 进行恢复：

    1. 将该 Store 对应 PV 的 `reclaimPolicy` 调整为 `Delete`：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl patch $(kubectl get pv -l app.kubernetes.io/instance=${cluster_name},tidb.pingcap.com/store-id=${store_id} -o name) -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}
        ```

    2. 删除 Pod 使用的 PVC：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl delete -n ${namespace} pvc tikv-${pod_name} --wait=false
        ```

    3. 删除 Pod，等待 Pod 重建：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl delete -n ${namespace} pod ${pod_name}
        ```

    Pod 重建后，会以在集群中注册一个新的 Store，恢复完成。

## TiDB 长连接被异常中断

许多负载均衡器 (Load Balancer) 会设置连接空闲超时时间。当连接上没有数据传输的时间超过设定值，负载均衡器会主动将连接中断。若发现 TiDB 使用过程中，长查询会被异常中断，可检查客户端与 TiDB 服务端之间的中间件程序。若其连接空闲超时时间较短，可尝试增大该超时时间。若不可修改，可打开 TiDB `tcp-keep-alive` 选项，启用 TCP keepalive 特性。

默认情况下，Linux 发送 keepalive 探测包的等待时间为 7200 秒。若需减少该时间，可通过 `podSecurityContext` 字段配置 `sysctls`。

- 如果 Kubernetes 集群内的 [kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) 允许配置 `--allowed-unsafe-sysctls=net.*`，请为 `kubelet` 配置该参数，并按如下方式配置 TiDB：

    {{< copyable "" >}}

    ```yaml
    tidb:
      ...
      podSecurityContext:
        sysctls:
        - name: net.ipv4.tcp_keepalive_time
          value: "300"
    ```

- 如果 Kubernetes 集群内的 [kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) 不允许配置 `--allowed-unsafe-sysctls=net.*`，请按如下方式配置 TiDB：

    {{< copyable "" >}}

    ```yaml
    tidb:
      annotations:
        tidb.pingcap.com/sysctl-init: "true"
      podSecurityContext:
        sysctls:
        - name: net.ipv4.tcp_keepalive_time
          value: "300"
      ...
    ```

> **注意：**
>
> 进行以上配置要求 TiDB Operator 1.1 及以上版本。
