---
title: Kubernetes 上的 TiDB 集群管理常用使用技巧
summary: 介绍 Kubernetes 上 TiDB 集群管理常用使用技巧。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/tips/','/zh/tidb-in-kubernetes/dev/troubleshoot','/docs-cn/tidb-in-kubernetes/dev/troubleshoot/']
---

# Kubernetes 上的 TiDB 集群管理常用使用技巧

本文介绍了 Kubernetes 上 TiDB 集群管理常用使用技巧。

## 诊断模式

当 Pod 处于 `CrashLoopBackoff` 状态时，Pod 内容器不断退出，导致无法正常使用 `kubectl exec`，给诊断带来不便。为了解决这个问题，TiDB in Kubernetes 提供了 PD/TiKV/TiDB Pod 诊断模式。在诊断模式下，Pod 内的容器启动后会直接挂起，不会再进入重复 Crash 的状态，此时，便可以通过 `kubectl exec` 连接 Pod 内的容器进行诊断。

操作方式：

1. 首先，为待诊断的 Pod 添加 Annotation：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl annotate pod ${pod_name} -n ${namespace} runmode=debug
    ```

    在 Pod 内的容器下次重启时，会检测到该 Annotation，进入诊断模式。

2. 等待 Pod 进入 Running 状态即可开始诊断：

    {{< copyable "shell-regular" >}}

    ```bash
    watch kubectl get pod ${pod_name} -n ${namespace}
    ```

    下面是使用 `kubectl exec` 进入容器进行诊断工作的例子：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl exec -it ${pod_name} -n ${namespace} -- /bin/sh
    ```

3. 诊断完毕，修复问题后，删除 Pod：

    ```bash
    kubectl delete pod ${pod_name} -n ${namespace}
    ```

Pod 重建后会自动回到正常运行模式。
