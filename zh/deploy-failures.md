---
title: Kubernetes 上的 TiDB 常见部署错误
summary: 介绍 Kubernetes 上 TiDB 部署的常见错误以及处理办法。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-failures/']
---

# Kubernetes 上的 TiDB 常见部署错误

本文介绍了 Kubernetes 上 TiDB 常见部署错误以及处理办法。

## Pod 未正常创建

创建集群后，如果 Pod 没有创建，则可以通过以下方式进行诊断：

{{< copyable "shell-regular" >}}

```shell
kubectl get tidbclusters -n ${namespace}
kubectl describe tidbclusters -n ${namespace} ${cluster_name}
kubectl get statefulsets -n ${namespace}
kubectl describe statefulsets -n ${namespace} ${cluster_name}-pd
```

创建备份恢复任务后，如果 Pod 没有创建，则可以通过以下方式进行诊断：

{{< copyable "shell-regular" >}}

```shell
kubectl get backups -n ${namespace}
kubectl get jobs -n ${namespace}
kubectl describe backups -n ${namespace} ${backup_name}
kubectl describe backupschedules -n ${namespace} ${backupschedule_name}
kubectl describe jobs -n ${namespace} ${backupjob_name}
kubectl describe restores -n ${namespace} ${restore_name}
```

## Pod 处于 Pending 状态

Pod 处于 Pending 状态，通常都是资源不满足导致的，比如：

* 使用持久化存储的 PD、TiKV、TiFlash、Pump、Monitor、Backup、Restore Pod 使用的 PVC 的 StorageClass 不存在或 PV 不足
* Kubernetes 集群中没有节点能满足 Pod 申请的 CPU 或内存
* PD 或者 TiKV Replicas 数量和集群内节点数量不满足 tidb-scheduler 高可用调度策略

此时，可以通过 `kubectl describe pod` 命令查看 Pending 的具体原因：

{{< copyable "shell-regular" >}}

```
kubectl describe po -n ${namespace} ${pod_name}
```

### CPU 或内存资源不足

如果是 CPU 或内存资源不足，可以通过降低对应组件的 CPU 或内存资源申请，使其能够得到调度，或是增加新的 Kubernetes 节点。

### PVC 的 StorageClass 不存在

如果是 PVC 的 StorageClass 找不到，可采取以下步骤：

1. 通过以下命令获取集群中可用的 StorageClass：

    {{< copyable "shell-regular" >}}

    ```
    kubectl get storageclass
    ```

2. 将 `storageClassName` 修改为集群中可用的 StorageClass 名字。

3. 使用下述方式更新配置文件：

   * 如果是启动 tidbcluster 集群，运行 `kubectl edit tc ${cluster_name} -n ${namespace}` 进行集群更新。
   * 如果是运行 backup/restore 的备份/恢复任务，首先需要运行 `kubectl delete bk ${backup_name} -n ${namespace}` 删掉老的备份/恢复任务，再运行 `kubectl apply -f backup.yaml` 重新创建新的备份/恢复任务。

4. 将 Statefulset 删除，并且将对应的 PVC 也都删除。

    {{< copyable "shell-regular" >}}

    ```
    kubectl delete pvc -n ${namespace} ${pvc_name}
    kubectl delete sts -n ${namespace} ${statefulset_name}
    ```

### 可用 PV 不足

如果集群中有 StorageClass，但可用的 PV 不足，则需要添加对应的 PV 资源。对于 Local PV，可以参考[本地 PV 配置](configure-storage-class.md#本地-pv-配置)进行扩充。

### 不满足 tidb-scheduler 高可用策略

tidb-scheduler 针对 PD 和 TiKV 定制了高可用调度策略。对于同一个 TiDB 集群，假设 PD 或者 TiKV 的 Replicas 数量为 N，那么可以调度到每个节点的 PD Pod 数量最多为 `M=(N-1)/2`（如果 N<3，M=1），可以调度到每个节点的 TiKV Pod 数量最多为 `M=ceil(N/3)`（ceil 表示向上取整，如果 N<3，M=1）。

如果 Pod 因为不满足高可用调度策略而导致状态为 Pending，需要往集群内添加节点。

## Pod 处于 CrashLoopBackOff 状态

Pod 处于 CrashLoopBackOff 状态意味着 Pod 内的容器重复地异常退出（异常退出后，容器被 Kubelet 重启，重启后又异常退出，如此往复）。定位方法有很多种。

### 查看 Pod 内当前容器的日志

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} logs -f ${pod_name}
```

### 查看 Pod 内容器上次启动时的日志信息

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} logs -p ${pod_name}
```

确认日志中的错误信息后，可以根据 [tidb-server 启动报错](https://pingcap.com/docs-cn/v3.0/how-to/troubleshoot/cluster-setup/#tidb-server-启动报错)，[tikv-server 启动报错](https://pingcap.com/docs-cn/v3.0/how-to/troubleshoot/cluster-setup/#tikv-server-启动报错)，[pd-server 启动报错](https://pingcap.com/docs-cn/v3.0/how-to/troubleshoot/cluster-setup/#pd-server-启动报错)中的指引信息进行进一步排查解决。

### cluster id mismatch

若是 TiKV Pod 日志中出现 "cluster id mismatch" 信息，则 TiKV Pod 使用的数据可能是其他或之前的 TiKV Pod 的旧数据。在集群配置本地存储时未清除机器上本地磁盘上的数据，或者强制删除了 PV 导致数据并没有被 local volume provisioner 程序回收，可能导致 PV 遗留旧数据，导致错误。

在确认该 TiKV 应作为新节点加入集群、且 PV 上的数据应该删除后，可以删除该 TiKV Pod 和关联 PVC。TiKV Pod 将自动重建并绑定新的 PV 来使用。集群本地存储配置中，应对机器上的本地存储删除，避免 Kubernetes 使用机器上遗留的数据。集群运维中，不可强制删除 PV ，应由 local volume provisioner 程序管理。用户通过创建、删除 PVC 以及设置 PV 的 reclaimPolicy 来管理 PV 的生命周期。

### ulimit 不足

另外，TiKV 在 ulimit 不足时也会发生启动失败的状况，对于这种情况，可以修改 Kubernetes 节点的 `/etc/security/limits.conf` 调大 ulimit：

```
root        soft        nofile        1000000
root        hard        nofile        1000000
root        soft        core          unlimited
root        soft        stack         10240
```

### 其他原因

假如通过日志无法确认失败原因，ulimit 也设置正常，那么可以通过[诊断模式](tips.md#诊断模式)进行进一步排查。
