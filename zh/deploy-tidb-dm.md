---
title: 在 Kubernetes 上部署 DM
summary: 了解如何在 Kubernetes 上部署 TiDB DM 集群。
---

# 在 Kubernetes 上部署 DM

[TiDB Data Migration](https://docs.pingcap.com/zh/tidb-data-migration/v2.0) (DM) 是一款支持从 MySQL 或 MariaDB 到 TiDB 的全量数据迁移和增量数据复制的一体化数据迁移任务管理平台。本文介绍如何使用 TiDB Operator 在 Kubernetes 上部署 DM。

## 前置条件

* TiDB Operator [部署](deploy-tidb-operator.md)完成。

> **注意：**
>
> 要求 TiDB Operator 版本 >= 1.2.0。

## 部署配置

通过配置 DMCluster CR 来配置 DM 集群。参考 DMCluster [示例](https://github.com/pingcap/tidb-operator/blob/master/examples/dm/dm-cluster.yaml)和 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md#dmcluster)（示例和 API 文档请切换到当前使用的 TiDB Operator 版本）完成 DMCluster CR (Custom Resource)。

### 集群名称

通过更改 `DMCluster` CR 中的 `metadata.name` 来配置集群名称。

### 版本

正常情况下，集群内的各组件应该使用相同版本，所以一般建议配置 `spec.<master/worker>.baseImage` + `spec.version` 即可。如果需要为不同的组件配置不同的版本，则可以配置 `spec.<master/worker>.version`。

相关参数的格式如下：

- `spec.version`，格式为 `imageTag`，例如 `v2.0.7`
- `spec.<master/worker>.baseImage`，格式为 `imageName`，例如 `pingcap/dm`
- `spec.<master/worker>.version`，格式为 `imageTag`，例如 `v2.0.7`

TiDB Operator 仅支持部署 DM 2.0 及更新版本。

### 集群配置

#### DM-master 配置

DM-master 为 DM 集群必须部署的组件。如果需要高可用部署则至少部署 3 个 DM-master Pod。

可以通过 DMCluster CR 的 `spec.master.config` 来配置 DM-master 配置参数。完整的 DM-master 配置参数，参考[DM-master 配置文件介绍](https://docs.pingcap.com/zh/tidb-data-migration/v2.0/dm-master-configuration-file)。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${dm_cluster_name}
  namespace: ${namespace}
spec:
  version: v2.0.7
  pvReclaimPolicy: Retain
  discovery: {}
  master:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    imagePullPolicy: IfNotPresent
    service:
      type: NodePort
      # 需要将 DM-master service 暴露在一个固定的 NodePort 时配置
      # masterNodePort: 30020
    replicas: 1
    storageSize: "10Gi"
    requests:
      cpu: 1
    config:
      rpc-timeout: 40s
```

#### DM-worker 配置

可以通过 DMCluster CR 的 `spec.worker.config` 来配置 DM-worker 配置参数。完整的 DM-worker 配置参数，参考[DM-worker 配置文件介绍](https://docs.pingcap.com/zh/tidb-data-migration/v2.0/dm-worker-configuration-file)。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${dm_cluster_name}
  namespace: ${namespace}
spec:
  ...
  worker:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    replicas: 1
    storageSize: "100Gi"
    requests:
      cpu: 1
    config:
      keepalive-ttl: 15
```

### 拓扑分布约束

配置 `topologySpreadConstraints` 可以实现同一组件的不同实例在拓扑上的均匀分布。具体配置方法请参阅 [Pod Topology Spread Constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/)。

如需使用 `topologySpreadConstraints`，需要满足以下条件：

* Kubernetes 集群使用 `default-scheduler`，而不是 `tidb-scheduler`。详情可以参考 [tidb-scheduler 与 default-scheduler](tidb-scheduler.md#tidb-scheduler-与-default-scheduler)。
* Kubernetes 集群开启 `EvenPodsSpread` feature gate。如果 Kubernetes 版本低于 v1.16 或集群未开启 `EvenPodsSpread` feature gate，`topologySpreadConstraints` 的配置将不会生效。

`topologySpreadConstraints` 可以设置在整个集群级别 (`spec.topologySpreadConstraints`) 来配置所有组件或者设置在组件级别 (例如 `spec.tidb.topologySpreadConstraints`) 来配置特定的组件。

以下是一个配置示例：

{{< copyable "" >}}

```yaml
topologySpreadConstrains:
- topologyKey: kubernetes.io/hostname
- topologyKey: topology.kubernetes.io/zone
```

该配置能让同一组件的不同实例均匀分布在不同 zone 和节点上。

当前 `topologySpreadConstraints` 仅支持 `topologyKey` 配置。在 Pod spec 中，上述示例配置会自动展开成如下配置：

```yaml
topologySpreadConstrains:
- topologyKey: kubernetes.io/hostname
  maxSkew: 1
  whenUnsatisfiable: DoNotSchedule
  labelSelector: <object>
- topologyKey: topology.kubernetes.io/zone
  maxSkew: 1
  whenUnsatisfiable: DoNotSchedule
  labelSelector: <object>
```

## 部署 DM 集群

按上述步骤配置完 DM 集群的 yaml 文件后，执行以下命令部署 DM 集群：

``` shell
kubectl apply -f ${dm_cluster_name}.yaml -n ${namespace}
```

如果服务器没有外网，需要按下述步骤在有外网的机器上将 DM 集群用到的 Docker 镜像下载下来并上传到服务器上，然后使用 `docker load` 将 Docker 镜像安装到服务器上：

1. 部署一套 DM 集群会用到下面这些 Docker 镜像（假设 DM 集群的版本是 v2.0.7）：

    ```bash
    pingcap/dm:v2.0.7
    ```

2. 通过下面的命令将所有这些镜像下载下来：

    {{< copyable "shell-regular" >}}

    ```bash
    docker pull pingcap/dm:v2.0.7

    docker save -o dm-v2.0.7.tar pingcap/dm:v2.0.7
    ```

3. 将这些 Docker 镜像上传到服务器上，并执行 `docker load` 将这些 Docker 镜像安装到服务器上：

    {{< copyable "shell-regular" >}}

    ```bash
    docker load -i dm-v2.0.7.tar
    ```

部署 DM 集群完成后，通过下面命令查看 Pod 状态：

``` shell
kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${dm_cluster_name}
```

单个 Kubernetes 集群中可以利用 TiDB Operator 部署管理多套 DM 集群，重复以上步骤并将 `${dm_cluster_name}` 替换成不同名字即可。不同集群既可以在相同 `namespace` 中，也可以在不同 `namespace` 中，可根据实际需求进行选择。

## 访问 Kubernetes 上的 DM 集群

在 Kubernetes 集群的 Pod 内访问 DM-master 时，使用 DM-master service 域名 `${cluster_name}-dm-master.${namespace}` 即可。

若需要在集群外访问，则需将 DM-master 服务端口暴露出去。在 `DMCluster` CR 中，通过 `spec.master.service` 字段进行配置：

```yaml
spec:
  ...
  master:
    service:
      type: NodePort
```

即可通过 `${kubernetes_node_ip}:${node_port}` 的地址访问 DM-master 服务。

更多服务暴露方式可参考 [访问 TiDB 集群](access-tidb.md)。

## 探索更多

- 如果你需要在 Kubernetes 上使用 DM 迁移 MySQL 数据到 TiDB 集群，请参考[在 Kubernetes 上部署使用 DM 迁移数据](use-tidb-dm.md)。
- 如果你需要在 Kubernetes 上为 DM 集群组件间开启 TLS，请参考 [为 DM 开启 TLS](enable-tls-for-dm.md)。
