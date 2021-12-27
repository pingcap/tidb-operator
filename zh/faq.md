---
title: Kubernetes 上的 TiDB 集群常见问题
summary: 介绍 Kubernetes 上的 TiDB 集群常见问题以及解决方案。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/faq/']
---

# Kubernetes 上的 TiDB 集群常见问题

本文介绍 Kubernetes 上的 TiDB 集群常见问题以及解决方案。

## 如何修改时区设置？

默认情况下，在 Kubernetes 集群上部署的 TiDB 集群各组件容器中的时区为 UTC，如果要修改时区配置，有下面两种情况：

### 第一次部署集群

配置 TidbCluster CR 的 `.spec.timezone` 属性，例如：

```bash
...
spec:
  timezone: Asia/Shanghai
...
```

然后部署 TiDB 集群。

### 集群已经在运行

如果 TiDB 集群已经在运行，需要先升级 TiDB 集群，然后再配置 TiDB 集群支持新的时区。

1. 升级 TiDB 集群

    配置 TidbCluster CR 的 `.spec.timezone` 属性，例如：

    ```bash
    ...
    spec:
      timezone: Asia/Shanghai
    ...
    ```

    然后升级 TiDB 集群。

2. 修改 TiDB 支持新的时区

    参考[时区支持](https://pingcap.com/docs-cn/stable/configure-time-zone/)，修改 TiDB 服务时区配置。

## TiDB 相关组件可以配置 HPA 或 VPA 么？

TiDB 集群目前还不支持 HPA（Horizontal Pod Autoscaling，自动水平扩缩容）和 VPA（Vertical Pod Autoscaling，自动垂直扩缩容），因为对于数据库这种有状态应用而言，实现自动扩缩容难度较大，无法仅通过 CPU 和 memory 监控数据来简单地实现。

## 使用 TiDB Operator 编排 TiDB 集群时，有什么场景需要人工介入操作吗？

如果不考虑 Kubernetes 集群本身的运维，TiDB Operator 存在以下可能需要人工介入的场景：

* TiKV 自动故障转移之后的集群调整，参考[自动故障转移](use-auto-failover.md)；
* 维护或下线指定的 Kubernetes 节点，参考[维护节点](maintain-a-kubernetes-node.md)。

## 在公有云上使用 TiDB Operator 编排 TiDB 集群时，推荐的部署拓扑是怎样的？

首先，为了实现高可用和数据安全，我们在推荐生产环境下的 TiDB 集群中至少部署在三个可用区 (Available Zone)。

当考虑 TiDB 集群与业务服务的部署拓扑关系时，TiDB Operator 支持下面几种部署形态。它们有各自的优势与劣势，具体选型需要根据实际业务需求进行权衡：

* 将 TiDB 集群与业务服务部署在同一个 VPC 中的同一个 Kubernetes 集群上；
* 将 TiDB 集群与业务服务部署在同一个 VPC 中的不同 Kubernetes 集群上；
* 将 TiDB 集群与业务服务部署在不同 VPC 中的不同 Kubernetes 集群上。

## TiDB Operator 支持 TiSpark 吗？

TiDB Operator 尚不支持自动编排 TiSpark。

假如要为 TiDB in Kubernetes 添加 TiSpark 组件，你需要在**同一个** Kubernetes 集群中自行维护 Spark，确保 Spark 能够访问到 PD 和 TiKV 实例的 IP 与端口，并为 Spark 安装 TiSpark 插件，TiSpark 插件的安装方式可以参考 [TiSpark](https://pingcap.com/docs-cn/stable/tispark-overview/#已有-spark-集群的部署方式)。

在 Kubernetes 上维护 Spark 可以参考 Spark 的官方文档：[Spark on Kubernetes](http://spark.apache.org/docs/latest/running-on-kubernetes.html)。

## 如何查看 TiDB 集群配置？

如果需要查看当前集群的 PD、TiKV、TiDB 组件的配置信息，可以执行下列命令:

* 查看 PD 配置文件

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl exec -it ${pod_name} -n ${namespace} -- cat /etc/pd/pd.toml
    ```

* 查看 TiKV 配置文件

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl exec -it ${pod_name} -n ${namespace} -- cat /etc/tikv/tikv.toml
    ```

* 查看 TiDB 配置文件

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl exec -it ${pod_name} -c tidb -n ${namespace} -- cat /etc/tidb/tidb.toml
    ```

## 部署 TiDB 集群时调度失败是什么原因？

TiDB Operator 调度 Pod 失败的原因可能有三种情况：

* 资源不足，导致 Pod 一直阻塞在 `Pending` 状态。详细说明参见[集群故障诊断](deploy-failures.md)。

* 部分 Node 被打了 `taint`，导致 Pod 无法调度到对应的 Node 上。详请参考 [taint & toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/)。

* 调度错误，导致 Pod 一直阻塞在 `ContainerCreating` 状态。这种情况发生时请检查 Kubernetes 集群中是否部署过多个 TiDB Operator。重复的 TiDB Operator 自定义调度器的存在，会导致同一个 Pod 的调度周期不同阶段会分别被不同的调度器处理，从而产生冲突。

    执行以下命令，如果有两条及以上记录，就说明 Kubernetes 集群中存在重复的 TiDB Operator，请根据实际情况删除多余的 TiDB Operator。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get deployment --all-namespaces | grep tidb-scheduler
    ```

## TiDB 如何保证数据安全可靠？

TiDB Operator 部署的 TiDB 集群使用 Kubernetes 集群提供的[持久卷](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)作为存储，保证数据的持久化存储。

PD 和 TiKV 使用 [Raft 一致性算法](https://raft.github.io/)将存储的数据在各节点间复制为多副本，以确保某个节点宕机时数据的安全性。

在底层，TiKV 使用复制日志 + 状态机 (State Machine) 的模型来复制数据。对于写入请求，数据被写入 Leader，然后 Leader 以日志的形式将命令复制到它的 Follower 中。当集群中的大多数节点收到此日志时，日志会被提交，状态机会相应作出变更。

## TidbCluster 的 Ready 项为 false 是否代表集群不可用？

执行 `kubectl get tc` 命令后，如果输出中显示某个 TiDBCluster 的 Ready 字段为 false，不代表对应的 TiDBCluster 不可用，集群可能处于以下任一状态：

* 升级中
* 缩扩容中
* PD、TiDB、TiKV、TiFlash 任一 Pod 状态不是 Ready

要判断 TiDB 集群是否真正不可用，你可以尝试连接 TiDB。如果无法连接成功，说明 TiDB 集群真正不可用。
