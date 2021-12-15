---
title: 维护 TiDB 集群所在的 Kubernetes 节点
summary: 介绍如何维护 TiDB 集群所在的 Kubernetes 节点。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/maintain-a-kubernetes-node/']
---

# 维护 TiDB 集群所在的 Kubernetes 节点

TiDB 是高可用数据库，可以在部分数据库节点下线的情况下正常运行，因此，我们可以安全地对底层 Kubernetes 节点进行停机维护。在具体操作时，针对 PD、TiKV 和 TiDB Pod 的不同特性，我们需要采取不同的操作策略。

本文档将详细介绍如何对 Kubernetes 节点进行临时或长期的维护操作。

环境准备：

- [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [`jq`](https://stedolan.github.io/jq/download/)

> **注意：**
>
> 维护节点前，需要保证 Kubernetes 集群的剩余资源足够运行 TiDB 集群。

## 维护短期内可恢复的节点

1. 使用 `kubectl cordon` 命令标记待维护节点为不可调度，防止新的 Pod 调度到待维护节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl cordon ${node_name}
    ```

2. 检查待维护节点上是否有 TiKV Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep tikv
    ```

    假如存在 TiKV Pod，针对每一个 Pod，进行以下操作：
    
    1. 参考[迁移 TiKV Region Leader](#迁移-tikv-region-leader) 将 Region Leader 迁移到其他 Pod。

    2. 通过调整 PD 的 `max-store-down-time` 配置来增大集群所允许的 TiKV Pod 下线时间，在此时间内维护完毕并恢复 Kubernetes 节点后，所有该节点上的 TiKV Pod 会自动恢复。
    
        以调整 `max-store-down-time` 为 `60m` 为例，请使用以下命令：

        {{< copyable "shell-regular" >}}

        ```shell
        pd-ctl config set max-store-down-time 60m
        ```

        调整 `max-store-down-time` 到合理的值。

3. 检查待维护节点上是否有 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep pd
    ```

    假如存在 PD Pod，针对每一个 Pod，参考[迁移 PD Leader](#迁移-pd-leader) 将 Leader 迁移到其他 Pod。

4. 确认待维护节点上不再有 TiKV 和 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name}
    ```

5. 使用 `kubectl drain` 命令将待维护节点上的 Pod 迁移到其它节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl drain ${node_name} --ignore-daemonsets
    ```

    运行后，该节点上的 Pod 会自动迁移到其它可用节点上。

6. 再次确认节点不再有任何 TiKV、TiDB 和 PD Pod 运行：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name}
    ```

7. 如果节点维护结束，在恢复节点后确认其健康状态：

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl get node ${node_name}
    ```

    观察到节点进入 `Ready` 状态后，继续操作。

8. 使用 `kubectl uncordon` 命令解除节点的调度限制：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl uncordon ${node_name}
    ```

9. 观察 Pod 是否全部恢复正常运行：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name}
    ```

    Pod 恢复正常运行后，操作结束。

## 维护短期内不可恢复的节点

1. 检查待维护节点上是否有 TiKV Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep tikv
    ```

    假如存在 TiKV Pod，针对每一个 Pod，参考[重调度 TiKV Pod](#重调度-tikv-pod) 将 Pod 重调度到其他节点。    

2. 检查待维护节点上是否有 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep pd
    ```

    假如存在 PD Pod，针对每一个 Pod，参考[重调度 PD Pod](#重调度-pd-pod) 将 Pod 重调度到其他节点。

3. 确认待维护节点上不再有 TiKV 和 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name}
    ```

4. 使用 `kubectl drain` 命令将待维护节点上的 Pod 迁移到其它节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl drain ${node_name} --ignore-daemonsets
    ```

    运行后，该节点上的 Pod 会自动迁移到其它可用节点上。

5. 再次确认节点不再有任何 TiKV、TiDB 和 PD Pod 运行：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name}
    ```

6. 最后（可选），假如是长期下线节点，建议将节点从 Kubernetes 集群中删除：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete node ${node_name}
    ```

## 重调度 PD Pod

针对节点长期下线等情形，为了尽可能减少业务受到的影响，可以将该节点上的 PD Pod 预先调度到其他节点。

### 如果节点存储可自动迁移

如果节点存储可以自动迁移（比如使用 EBS），你不需要删除 PD Member，只需要迁移 Leader 到其他 Pod 后删除原来的 Pod 就可以实现重调度。

1. 使用 `kubectl cordon` 命令标记待维护节点为不可调度，防止新的 Pod 调度到待维护节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl cordon ${node_name}
    ```

2. 查看待维护节点上的 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep pd
    ```

3. 参考[迁移 PD Leader](#迁移-pd-leader) 将 Leader 迁移到其他 Pod。

4. 删除 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

5. 确认该 PD Pod 正常调度到其它节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

### 如果节点存储不可自动迁移

如果节点存储不可以自动迁移（比如使用本地存储），你需要删除 PD Member 以实现重调度。

1. 使用 `kubectl cordon` 命令标记待维护节点为不可调度，防止新的 Pod 调度到待维护节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl cordon ${node_name}
    ```

2. 查看待维护节点上的 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep pd
    ```

3. 参考[迁移 PD Leader](#迁移-pd-leader) 将 Leader 迁移到其他 Pod。

4. 下线 PD Pod。

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl member delete name ${pod_name}
    ```

5. 确认 PD Member 已删除：

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl member
    ```

6. 解除 PD Pod 与节点本地盘的绑定。

    查询 Pod 使用的 `PesistentVolumeClaim`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pvc -l tidb.pingcap.com/pod-name=${pod_name}
    ```

    删除该 `PesistentVolumeClaim`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pvc ${pvc_name} --wait=false
    ```

7. 删除 PD Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

8. 观察该 PD Pod 是否正常调度到其它节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

## 重调度 TiKV Pod

针对节点长期下线等情形，为了尽可能减少业务受到的影响，可以将该节点上的 TiKV Pod 预先调度到其他节点。

### 如果节点存储可自动迁移

如果节点存储可以自动迁移（比如使用 EBS），你不需要删除整个 TiKV Store，只需要迁移 Region Leader 到其他 Pod 后删除原来的 Pod 就可以实现重调度。

1. 使用 `kubectl cordon` 命令标记待维护节点为不可调度，防止新的 Pod 调度到待维护节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl cordon ${node_name}
    ```

2. 查看待维护节点上的 TiKV Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep tikv
    ```

3. 参考[迁移 TiKV Region Leader](#迁移-tikv-region-leader) 将 Leader 迁移到其他 Pod。

4. 删除 TiKV Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

5. 确认该 TiKV Pod 正常调度到其它节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

6. 移除 evict-leader-scheduler，等待 Region Leader 自动调度回来：

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl scheduler remove evict-leader-scheduler-${ID}
    ```

### 如果节点存储不可自动迁移

如果节点存储不可以自动迁移（比如使用本地存储），你需要删除整个 TiKV Store 以实现重调度。

1. 使用 `kubectl cordon` 命令标记待维护节点为不可调度，防止新的 Pod 调度到待维护节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl cordon ${node_name}
    ```

2. 查看待维护节点上的 TiKV Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pod --all-namespaces -o wide | grep ${node_name} | grep tikv
    ```

3. 参考[迁移 TiKV Region Leader](#迁移-tikv-region-leader) 将 Leader 迁移到其他 Pod。

4. 下线 TiKV Pod。

    > **注意：**
    >
    > 下线 TiKV Pod 前，需要保证集群中剩余的 TiKV Pod 数不少于 PD 配置中的 TiKV 数据副本数（配置项：`max-replicas`，默认值 3）。假如不符合该条件，需要先操作扩容 TiKV。

    查看 TiKV Pod 的 `store-id`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get tc ${cluster_name} -ojson | jq ".status.tikv.stores | .[] | select ( .podName == \"${pod_name}\" ) | .id"
    ```

    下线 Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl store delete ${ID}
    ```

5. 等待 store 状态（`state_name`）转移为 `Tombstone`：

    {{< copyable "shell-regular" >}}

    ```shell
    watch pd-ctl store ${ID}
    ```

6. 解除 TiKV Pod 与节点本地盘的绑定。

    查询 Pod 使用的 `PesistentVolumeClaim`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pvc -l tidb.pingcap.com/pod-name=${pod_name}
    ```

    删除该 `PesistentVolumeClaim`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pvc ${pvc_name} --wait=false
    ```

7. 删除 TiKV Pod：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

8. 观察该 TiKV Pod 是否正常调度到其它节点上：

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

9. 移除 evict-leader-scheduler，等待 Region Leader 自动调度回来：

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl scheduler remove evict-leader-scheduler-${ID}
    ```

## 迁移 PD Leader

1. 查看 PD Leader：

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl member leader show
    ```

2. 如果 Leader Pod 所在节点是要维护的节点，则需要将 Leader 先迁移到其他节点上的 Pod。

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl member leader transfer ${pod_name}
    ```

    其中 `${pod_name}` 是其他节点上的 PD Pod。

## 迁移 TiKV Region Leader

1. 查看 TiKV Pod 的 `store-id`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get tc ${cluster_name} -ojson | jq ".status.tikv.stores | .[] | select ( .podName == \"${pod_name}\" ) | .id"
    ```

2. 驱逐 Region Leader：

    {{< copyable "shell-regular" >}}

    ```shell
    pd-ctl scheduler add evict-leader-scheduler ${ID}
    ```

3. 检查 Region Leader 已经全部被迁移走:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get tc ${cluster_name} -ojson | jq ".status.tikv.stores | .[] | select ( .podName == \"${pod_name}\" ) | .leaderCount"
    ```
