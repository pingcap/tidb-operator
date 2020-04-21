---
title: Kubernetes 上的 TiDB 集群的资源配置以及容灾配置
summary: 介绍 Kubernetes 上 TiDB 集群的的资源配置以及容灾配置。
category: reference
---

<!-- markdownlint-disable MD007 -->

# Kubernetes 上的 TiDB 集群的资源配置以及容灾配置

本文介绍 Kubernetes 上 TiDB 集群的资源配置以及容灾配置。

## 资源配置说明

部署前需要根据实际情况和需求，为 TiDB 集群各个组件配置资源，其中 PD、TiKV、TiDB 是 TiDB 集群的核心服务组件，在生产环境下它们的资源配置还需要按组件要求指定，具体参考：[资源配置推荐](https://pingcap.com/docs-cn/v3.0/how-to/deploy/hardware-recommendations)。

为了保证 TiDB 集群的组件在 Kubernetes 中合理的调度和稳定的运行，建议为其设置 Guaranteed 级别的 QoS，通过在配置资源时让 limits 等于 requests 来实现, 具体参考：[配置 QoS](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/)。

如果使用 NUMA 架构的 CPU，为了获得更好的性能，需要在节点上开启 `Static` 的 CPU 管理策略。为了 TiDB 集群组件能独占相应的 CPU 资源，除了为其设置上述 Guaranteed 级别的 QoS 外，还需要保证 CPU 的配额必须是大于或等于 1 的整数。具体参考: [CPU 管理策略](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies)。

## 容灾配置说明

TiDB 是分布式数据库，它的容灾需要做到在任一个物理拓扑节点发生故障时，不仅服务不受影响，还要保证数据也是完整和可用。下面分别具体说明这两种容灾的配置。

### TiDB 服务的容灾

TiDB 服务容灾本质上基于 Kubernetes 的调度功能来实现的，为了优化调度，TiDB Operator 提供了自定义的调度器，该调度器通过指定的调度算法能在 host 层面，保证 TiDB 服务的容灾，而且目前 TiDB Cluster 使用该调度器作为默认调度器，设置项是上述列表中的 `schedulerName` 配置项。

其它层面的容灾（例如 rack，zone，region）是通过 Affinity 的 `PodAntiAffinity` 来保证，通过 `PodAntiAffinity` 能尽量避免同一组件的不同实例部署到同一个物理拓扑节点上，从而达到容灾的目的，Affinity 的使用参考：[Affinity & AntiAffinity](https://kubernetes.io/docs/concepts/configuration/assign-Pod-node/#affinity-and-anti-affinity)。

下面是一个典型的容灾设置例子：

{{< copyable "shell-regular" >}}

```shell
affinity:
 podAntiAffinity:
   preferredDuringSchedulingIgnoredDuringExecution:
   # this term works when the nodes have the label named region
   - weight: 10
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "region"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named zone
   - weight: 20
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "zone"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named rack
   - weight: 40
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "rack"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named kubernetes.io/hostname
   - weight: 80
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "kubernetes.io/hostname"
       namespaces:
       - ${namespace}
```

### 数据的容灾

在开始数据容灾配置前，首先请阅读[集群拓扑信息配置](https://pingcap.com/docs-cn/v3.0/how-to/deploy/geographic-redundancy/location-awareness)。该文档描述了 TiDB 集群数据容灾的实现原理。

在 Kubernetes 上支持数据容灾的功能，需要如下操作：

* 为 PD 设置拓扑位置 Label 集合

    用 Kubernetes 集群 Node 节点上描述拓扑位置的 Label 集合替换 `pd.config` 配置项中里的 `location-labels` 信息。

    > **注意：**
    >
    > * PD 版本 < v3.0.9 不支持名字中带 `/` 的 Label。
    > * 如果在 `location-labels` 中配置 `hostname`，TiDB Operator 会从 Node Label 中的 `kubernetes.io/hostname` 获取值。

* 为 TiKV 节点设置所在的 Node 节点的拓扑信息

    TiDB Operator 会自动为 TiKV 获取其所在 Node 节点的拓扑信息，并调用 PD 接口将这些信息设置为 TiKV 的 store labels 信息，这样 TiDB 集群就能基于这些信息来调度数据副本。

    如果当前 Kubernetes 集群的 Node 节点没有表示拓扑位置的 Label，或者已有的拓扑 Label 名字中带有 `/`，可以通过下面的命令手动给 Node 增加标签：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl label node ${node_name} region=${region_name} zone=${zone_name} rack=${rack_name} kubernetes.io/hostname=${host_name}
    ```

    其中 `region`、`zone`、`rack`、`kubernetes.io/hostname` 只是举例，要添加的 Label 名字和数量可以任意定义，只要符合规范且和 `pd.config` 里的 `location-labels` 设置的 Labels 保持一致即可。
