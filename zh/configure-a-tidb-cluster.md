---
title: 配置 TiDB 集群
summary: 了解如何在 Kubernetes 中配置 TiDB 集群。
category: how-to
aliases: ['/zh/tidb-in-kubernetes/dev/configure-cluster-using-tidbcluster/','/docs-cn/tidb-in-kubernetes/dev/configure-cluster-using-tidbcluster/']
---

<!-- markdownlint-disable MD007 -->

# 在 Kubernetes 中配置 TiDB 集群

本文档介绍了如何配置生产可用的 TiDB 集群。涵盖以下内容：

- [资源配置](#资源配置)
- [部署配置](#部署配置)
- [高可用配置](#高可用配置)

## 资源配置

部署前需要根据实际情况和需求，为 TiDB 集群各个组件配置资源，其中 PD、TiKV、TiDB 是 TiDB 集群的核心服务组件，在生产环境下它们的资源配置还需要按组件要求指定，具体参考：[资源配置推荐](https://pingcap.com/docs-cn/stable/how-to/deploy/hardware-recommendations)。

为了保证 TiDB 集群的组件在 Kubernetes 中合理的调度和稳定的运行，建议为其设置 Guaranteed 级别的 QoS，通过在配置资源时让 limits 等于 requests 来实现, 具体参考：[配置 QoS](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/)。

如果使用 NUMA 架构的 CPU，为了获得更好的性能，需要在节点上开启 `Static` 的 CPU 管理策略。为了 TiDB 集群组件能独占相应的 CPU 资源，除了为其设置上述 Guaranteed 级别的 QoS 外，还需要保证 CPU 的配额必须是大于或等于 1 的整数。具体参考: [CPU 管理策略](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies)。

## 部署配置

通过配置 `TidbCluster` CR 来配置 TiDB 集群。参考 TidbCluster [示例](https://github.com/pingcap/tidb-operator/blob/master/examples/tiflash/tidb-cluster.yaml)和 [API 文档](https://github.com/pingcap/docs-tidb-operator/blob/master/en/api-references.md)（示例和 API 文档请切换到当前使用的 TiDB Operator 版本）完成 TidbCluster CR(Custom Resource)。

> **注意：**
>
> 建议在 `${cluster_name}` 目录下组织 TiDB 集群的配置，并将其另存为 `${cluster_name}/tidb-cluster.yaml`。默认条件下，修改配置不会自动应用到 TiDB 集群中，只有在 Pod 重启时，才会重新加载新的配置文件。

建议设置 `spec.configUpdateStrategy` 为 `RollingUpdate` 开启配置自动更新特性，在每次配置更新时，自动对组件执行滚动更新，将修改后的配置应用到集群中。

### 集群名称

通过更改 `TiDBCuster` CR 中的 `metadata.name` 来配置集群名称。

### 版本

正常情况下，集群内的各组件应该使用相同版本，所以一般建议配置 `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.baseImage` + `spec.version` 即可。如果需要为不同的组件配置不同的版本，则可以配置 `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`。

相关参数的格式如下：

- `spec.version`，格式为 `imageTag`，例如 `v4.0.0`
- `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.baseImage`，格式为 `imageName`，例如 `pingcap/tidb`
- `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`，格式为 `imageTag`，例如 `v4.0.0`

### Storage class

如果需要设置存储类型，可以修改 `${cluster_name}/tidb-cluster.yaml` 中各组件的 `storageClassName` 字段。关于 Kubernetes 集群支持哪些[存储类型](configure-storage-class.md)，请联系系统管理员确定。

另外，TiDB 集群不同组件对磁盘的要求不一样，所以部署集群前，要根据当前 Kubernetes 集群支持的存储类型以及使用场景，为 TiDB 集群各组件选择合适的存储类型，

生产环境推荐使用本地存储，但实际 Kubernetes 集群中本地存储可能按磁盘类型进行了分类，例如 `nvme-disks`，`sas-disks`。

对于演示环境或功能性验证，可以使用网络存储，例如 `ebs`，`nfs` 等。

> **注意：**
>
> 如果创建集群时设置了集群中不存在的存储类型，则会导致集群创建处于 Pending 状态，需要[将集群彻底销毁掉](destroy-a-tidb-cluster.md)。

### 集群拓扑

#### PD/TiKV/TiDB

默认示例的集群拓扑是：3 个 PD Pod，3 个 TiKV Pod，2 个 TiDB Pod。在该部署拓扑下根据数据高可用原则，TiDB Operator 扩展调度器要求 Kubernetes 集群中至少有 3 个节点。可以修改 `replicas` 配置来更改每个组件的 Pod 数量。

> **注意：**
>
> 如果 Kubernetes 集群节点个数少于 3 个，将会导致有一个 PD Pod 处于 Pending 状态，而 TiKV 和 TiDB Pod 也都不会被创建。Kubernetes 集群节点个数少于 3 个时，为了使 TiDB 集群能启动起来，可以将默认部署的 PD 和 TiKV Pod 个数都减小到 1 个。

#### 部署 TiFlash

如果要在集群中开启 TiFlash，需要在 `${cluster_name}/tidb-cluster.yaml` 文件中配置 `spec.pd.config.replication.enable-placement-rules: "true"`，并配置 `spec.tiflash`：

```yaml
  pd:
    config:
      ...
      replication:
        enable-placement-rules: "true"
        ...
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 3
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

TiFlash 支持挂载多个 PV，如果要为 TiFlash 配置多个 PV，可以在 `tiflash.storageClaims` 下面配置多项，每一项可以分别配置 `storage reqeust` 和 `storageClassName`，例如：

```yaml
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 3
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

> **警告：**
>
> 由于 TiDB Operator 会按照 `storageClaims` 列表中的配置**按顺序**自动挂载 PV，如果需要为 TiFlash 增加磁盘，请确保只在列表原有配置**最后添加**，并且**不能**修改列表中原有配置的顺序。

#### 部署 TiCDC

如果要在集群中开启 TiCDC，需要在 `${cluster_name}/tidb-cluster.yaml` 文件中配置 `spec.ticdc`：

```yaml
  ticdc:
    baseImage: pingcap/ticdc
    replicas: 3
    config:
      logLevel: info
```

值得注意的是，如果需要部署企业版的 TiDB/PD/TiKV/TiFlash/TiCDC，需要将 db.yaml 中 `spec.<tidb/pd/tikv/tiflash/ticdc>.baseImage` 配置为企业版镜像，格式为 `pingcap/<tidb/pd/tikv/tiflash/ticdc>-enterprise`。

例如:

```yaml
spec:
  ...
  pd:
    baseImage: pingcap/pd-enterprise
  ...
  tikv:
    baseImage: pingcap/tikv-enterprise
```

### 配置 TiDB 组件

本节介绍如何配置 TiDB/TiKV/PD/TiFlash/TiCDC 的配置选项，目前 TiDB Operator 1.1 版本支持了 TiDB 集群 4.0 版本参数。

#### 配置 TiDB 配置参数

你可以通过 TidbCluster CR 的 `spec.tidb.config` 来配置 TiDB 配置参数。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
....
  tidb:
    image: pingcap.com/tidb:v4.0.2
    imagePullPolicy: IfNotPresent
    replicas: 1
    service:
      type: ClusterIP
    config:
      split-table: true
      oom-action: "log"
    requests:
      cpu: 1
```

获取所有可以配置的 TiDB 配置参数，请参考 [TiDB 配置文档](https://pingcap.com/docs-cn/stable/tidb-configuration-file/)。

> **注意：**
>
> 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 TiDB 组件无法正常启动。

#### 配置 TiKV 配置参数

你可以通过 TidbCluster CR 的 `spec.tikv.config` 来配置 TiKV 配置参数。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
....
  tikv:
    image: pingcap.com/tikv:v4.0.2
    config: {}
    replicas: 1
    requests:
      cpu: 2
```

获取所有可以配置的 TiKV 配置参数，请参考 [TiKV 配置文档](https://pingcap.com/docs-cn/stable/tikv-configuration-file/)

> **注意：**
>
> 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 TiKV 组件无法正常启动。

#### 配置 PD 配置参数

你可以通过 TidbCluster CR 的 `spec.pd.config` 来配置 PD 配置参数。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
.....
  pd:
    image: pingcap.com/pd:v4.0.2
    config:
      lease: 3
      enable-prevote: true
```

获取所有可以配置的 PD 配置参数，请参考 [PD 配置文档](https://pingcap.com/docs-cn/stable/pd-configuration-file/)

> **注意：**
>
> 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 PD 组件无法正常启动。

#### 配置 TiFlash 配置参数

你可以通过 TidbCluster CR 的 `spec.tiflash.config` 来配置 TiFlash 配置参数。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  ...
  tiflash:
    config:
      config:
        logger:
          count: 5
          level: information
```

获取所有可以配置的 TiFlash 配置参数，请参考 [TiFlash 配置文档](https://pingcap.com/docs-cn/stable/tiflash/tiflash-configuration/)

#### 配置 TiCDC 启动参数

你可以通过 TidbCluster CR 的 `spec.ticdc.config` 来配置 TiCDC 启动参数。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  ...
  ticdc:
    config:
      timezone: UTC
      gcTTL: 86400
      logLevel: info
```

获取所有可以配置的 TiCDC 启动参数，请参考 [TiCDC 启动参数文档](https://pingcap.com/docs-cn/stable/ticdc/deploy-ticdc/#手动在原有-tidb-集群上新增-ticdc-组件)。

## 高可用配置

> **注意：**
>
> TiDB Operator 提供了自定义的调度器，该调度器通过指定的调度算法能在 host 层面保证 TiDB 服务的高可用。目前，TiDB 集群使用该调度器作为默认调度器，可通过 `spec.schedulerName` 配置项进行设置。本节重点介绍如何配置 TiDB 集群以容忍其他级别的故障，例如机架、可用区或 region。本部分可根据使用需求配置，不是必选。

TiDB 是分布式数据库，它的高可用需要做到在任一个物理拓扑节点发生故障时，不仅服务不受影响，还要保证数据也是完整和可用。下面分别具体说明这两种高可用的配置。

### TiDB 服务高可用

其它层面的高可用（例如 rack，zone，region）是通过 Affinity 的 `PodAntiAffinity` 来保证，通过 `PodAntiAffinity` 能尽量避免同一组件的不同实例部署到同一个物理拓扑节点上，从而达到高可用的目的，Affinity 的使用参考：[Affinity & AntiAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)。

下面是一个典型的高可用设置例子：

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

### 数据的高可用

在开始数据高可用配置前，首先请阅读[集群拓扑信息配置](https://pingcap.com/docs-cn/stable/schedule-replicas-by-topology-labels/)。该文档描述了 TiDB 集群数据高可用的实现原理。

在 Kubernetes 上支持数据高可用的功能，需要如下操作：

* 为 PD 设置拓扑位置 Label 集合

    用 Kubernetes 集群 Node 节点上描述拓扑位置的 Label 集合替换 `pd.config` 配置项中里的 `location-labels` 信息。

    > **注意：**
    >
    > * PD 版本 < v3.0.9 不支持名字中带 `/` 的 Label。
    > * 如果在 `location-labels` 中配置 `host`，TiDB Operator 会从 Node Label 中的 `kubernetes.io/hostname` 获取值。

* 为 TiKV 节点设置所在的 Node 节点的拓扑信息

    TiDB Operator 会自动为 TiKV 获取其所在 Node 节点的拓扑信息，并调用 PD 接口将这些信息设置为 TiKV 的 store labels 信息，这样 TiDB 集群就能基于这些信息来调度数据副本。

    如果当前 Kubernetes 集群的 Node 节点没有表示拓扑位置的 Label，或者已有的拓扑 Label 名字中带有 `/`，可以通过下面的命令手动给 Node 增加标签：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl label node ${node_name} region=${region_name} zone=${zone_name} rack=${rack_name} kubernetes.io/hostname=${host_name}
    ```

    其中 `region`、`zone`、`rack`、`kubernetes.io/hostname` 只是举例，要添加的 Label 名字和数量可以任意定义，只要符合规范且和 `pd.config` 里的 `location-labels` 设置的 Labels 保持一致即可。
