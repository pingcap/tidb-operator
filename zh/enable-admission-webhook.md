---
title: 开启 TiDB-Operator 准入控制器
summary: 介绍如何开启 TiDB-Operator 准入控制器以及它的作用
category: how-to
---

# TiDB Operator 准入控制器

Kubernetes 在 1.9 版本引入了 [动态准入机制](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/)，从而使得拥有对 Kubernetes 中的各类资源进行修改与验证的功能。 在 TiDB Operator 中，我们也同样使用了动态准入机制来帮助我们进行相关资源的修改、验证与运维。

## 开启 TiDB Operator 准入控制器

TiDB Operator 在默认安装情况下不会开启准入控制器，你需要手动开启:

1. 修改 Operator 的 `values.yaml`

    开启 Operator Webhook 特性:

    ```yaml
    admissionWebhook:
      create: true
    ```

2. 配置失败策略

    在 Kubernetes 1.15 版本之前，动态准入机制的管理机制的粒度较粗并且并不方便去使用。所以为了防止 TiDB Operator 的动态准入机制影响全局集群，我们需要配置 [失败策略](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy)。

    对于 Kubernetes 1.15 以下的版本，我们推荐将 TiDB Operator 失败策略配置为 `Ignore`，从而防止 TiDB Operator 的 admission webhook 出现异常时影响整个集群。

    ```yaml
    ......
    failurePolicy:
        validation: Ignore
        mutation: Ignore
    ```

    对于 Kubernetes 1.15 及以上的版本，我们推荐给 TiDB Operator 失败策略配置为 `Failure`，由于 Kubernetes 1.15 及以上的版本中，动态准入机制已经有了基于 Label 的筛选机制，所以不会由于 TiDB Operator 的 admission webhook 出现异常而影响整个集群。

    ```yaml
    ......
    failurePolicy:
        validation: Failure
        mutation: Failure
    ```

3. 安装/更新 TiDB Operator

    修改完 `values.yaml` 文件中的上述配置项以后，进行 TiDB Operator 部署或者更新。安装与更新 TiDB Operator 请参考[在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md)。 


## TiDB Operator 准入控制器功能

TiDB Operator 通过准入控制器的帮助实现了许多功能。我们将在这里介绍各个资源的准入控制器与其相对应的功能。

1. Pod 验证准入控制器:

    Pod 准入控制器提供了对 PD/TiKV/TiDB 组件安全下线与安全上线的保证，通过 Pod 验证准入控制器，我们可以实现 [重启 Kubernetes 上的 TiDB 集群](restart-a-tidb-cluster.md)。该组件在准入控制器开启的情况下默认开启。

    ```yaml
    admissionWebhook:
      validation:
        pods: true
    ```

2. StatefulSet 验证准入控制器:

    StatefulSet 验证准入控制器帮助实现 TiDB 集群中 TiDB/TiKV 组件的灰度发布，该组件在准入控制器开启的情况下默认关闭。

    ```yaml
    admissionWebhook:
      validation:
        statefulSets: false
    ```

    通过 `tidb.pingcap.com/tikv-partition` 和 `tidb.pingcap.com/tidb-partition` 这两个 annotation, 你可以控制 TiDB 集群中 TiDB 与 TiKV 组件的灰度发布。你可以通过以下方式对 TiDB 集群的 TiKV 组件设置灰度发布，其中 `partition=2` 的效果等同于 [StatefulSet 分区](https://kubernetes.io/zh/docs/concepts/workloads/controllers/statefulset/#%E5%88%86%E5%8C%BA)

    ```shell
    $  kubectl annotate tidbcluster <name> -n <namespace> tidb.pingcap.com/tikv-partition=2 
    tidbcluster.pingcap.com/<name> annotated
    ```

    执行以下命令取消灰度发布设置：

    ```shell
    $  kubectl annotate tidbcluster <name> -n <namespace> tidb.pingcap.com/tikv-partition- 
    tidbcluster.pingcap.com/<name> annotated
    ```

    以上设置同样适用于 TiDB 组件。

3. TiDB Operator 资源验证准入控制器:

    TiDB Operator 资源验证准入控制器帮助实现针对 `TidbCluster`、`TidbMonitor` 等 TiDB Operator 自定义资源的验证，该组件在准入控制器开启的情况下默认关闭。

    ```yaml
    admissionWebhook:
      validation:
        pingcapResources: false
    ```

    举个例子，对于 `TidbCluster` 资源，TiDB Operator 资源验证准入控制器将会检查其 `spec` 字段中的必要字段。如果在 `TidbCluster` 创建或者更新时发现检查不通过，比如同时没有定义 `spec.pd.image` 或者 `spec.pd.baseImage` 字段，TiDB Operator 资源验证准入控制器将会拒绝这个请求。

4. Pod 修改准入控制器:

    Pod 修改准入控制器帮助我们在弹性伸缩场景下实现 TiKV 的热点调度功能，在 [启用 TidbCluster 弹性伸缩](enable-tidb-cluster-auto-scaling.md)中需要开启该控制器。该组件在准入控制器开启的情况下默认开启。

    ```yaml
    admissionWebhook:
      mutation:
        pods: true
    ```

5. TiDB Operator 资源修改准入控制器:

    TiDB Operator 资源修改准入控制器帮助我们实现 TiDB Operator 相关自定义资源的默认值填充工作，如 `TidbCluster`，`TidbMonitor`等。该组件在准入控制器开启的情况下默认开启。

    ```yaml
    admissionWebhook:
      mutation:
        pingcapResources: true
    ```
