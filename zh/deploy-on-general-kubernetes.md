---
title: 在标准 Kubernetes 上部署 TiDB 集群
summary: 介绍如何在标准 Kubernetes 集群上通过 TiDB Operator 部署 TiDB 集群。
category: how-to
---

# 在标准 Kubernetes 上部署 TiDB 集群

本文主要描述了如何在标准的 Kubernetes 集群上通过 TiDB Operator 部署 TiDB 集群。

## 前置条件

* TiDB Operator [部署](deploy-tidb-operator.md)完成。

## 配置 TiDB 集群

参考 TidbCluster [示例](https://github.com/pingcap/tidb-operator/blob/master/examples/basic/tidb-cluster.yaml)和 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)（示例和 API 文档请切换到当前使用的 TiDB Operator 版本）完成 TidbCluster CR(Custom Resource)，并保存到文件 `${cluster_name}/tidb-cluster.yaml`。

需要注意的是，TidbCluster CR 中关于镜像配置有多个参数：

- `spec.version`，格式为 `imageTag`，例如 `v3.1.0`
- `spec.<pd/tidb/tikv/pump>.baseImage`，格式为 `imageName`，例如 `pingcap/tidb`
- `spec.<pd/tidb/tikv/pump>.version`，格式为 `imageTag`，例如 `v3.1.0`
- `spec.<pd/tidb/tikv/pump>.image`，格式为 `imageName:imageTag`，例如 `pingcap/tidb:v3.1.0`

镜像配置获取的优先级为：

`spec.<pd/tidb/tikv/pump>.baseImage` + `spec.<pd/tidb/tikv/pump>.version` > `spec.<pd/tidb/tikv/pump>.baseImage` + `spec.version` > `spec.<pd/tidb/tikv/pump>.image`。

正常情况下，集群内的各组件应该使用相同版本，所以一般建议配置 `spec.<pd/tidb/tikv/pump>.baseImage` + `spec.version` 即可。

默认条件下，修改配置不会自动应用到 TiDB 集群中，只有在 Pod 重启时，才会重新加载新的配置文件，建议设置 `spec.configUpdateStrategy` 为 `RollingUpdate` 开启配置自动更新特性，在每次配置更新时，自动对组件执行滚动更新，将修改后的配置应用到集群中。

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

如果要在集群中开启 TiCDC，需要在 `${cluster_name}/tidb-cluster.yaml` 文件中配置 `spec.ticdc`：

```yaml
  ticdc:
    baseImage: pingcap/ticdc
    replicas: 3
    config:
      logLevel: info
```

如果要部署 TiDB 集群监控，请参考 TidbMonitor [示例](https://github.com/pingcap/tidb-operator/blob/master/manifests/monitor/tidb-monitor.yaml)和 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)（示例和 API 文档请切换到当前使用的 TiDB Operator 版本）完成 TidbMonitor CR，并保存到文件 `${cluster_name}/tidb-monitor.yaml`。

### 存储类型

- 生产环境：推荐使用本地存储，但实际 Kubernetes 集群中本地存储可能按磁盘类型进行了分类，例如 `nvme-disks`，`sas-disks`。
- 演示环境或功能性验证：可以使用网络存储，例如 `ebs`，`nfs` 等。

另外 TiDB 集群不同组件对磁盘的要求不一样，所以部署集群前要根据当前 Kubernetes 集群支持的存储类型以及使用场景为 TiDB 集群各组件选择合适的存储类型，通过修改 `${cluster_name}/tidb-cluster.yaml` 和 `${cluster_name}/tidb-monitor.yaml` 中各组件的 `storageClassName` 字段设置存储类型。关于 Kubernetes 集群支持哪些[存储类型](configure-storage-class.md)，请联系系统管理员确定。

> **注意：**
>
> 如果创建集群时设置了集群中不存在的存储类型，则会导致集群创建处于 Pending 状态，需要[将集群彻底销毁掉](destroy-a-tidb-cluster.md)。

### 集群拓扑

默认示例的集群拓扑是：3 个 PD Pod，3 个 TiKV Pod，2 个 TiDB Pod。在该部署拓扑下根据数据高可用原则，TiDB Operator 扩展调度器要求 Kubernetes 集群中至少有 3 个节点。如果 Kubernetes 集群节点个数少于 3 个，将会导致有一个 PD Pod 处于 Pending 状态，而 TiKV 和 TiDB Pod 也都不会被创建。

Kubernetes 集群节点个数少于 3 个时，为了使 TiDB 集群能启动起来，可以将默认部署的 PD 和 TiKV Pod 个数都减小到 1 个。

## 部署 TiDB 集群

TiDB Operator 部署并配置完成后，可以通过下面命令部署 TiDB 集群：

1. 创建 `Namespace`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create namespace ${namespace}
    ```

    > **注意：**
    >
    > `namespace` 是[命名空间](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)，可以起一个方便记忆的名字，比如和 `cluster-name` 相同的名称。

2. 部署 TiDB 集群：
    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ${cluster_name} -n ${namespace}
    ```

3. 通过下面命令查看 Pod 状态：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${cluster_name}
    ```

单个 Kubernetes 集群中可以利用 TiDB Operator 部署管理多套 TiDB 集群，重复以上步骤并将 `cluster-name` 替换成不同名字即可。不同集群既可以在相同 `namespace` 中，也可以在不同 `namespace` 中，可根据实际需求进行选择。

### 初始化 TiDB 集群

如果要在部署完 TiDB 集群后做一些初始化工作，参考 [Kubernetes 上的集群初始化配置](initialize-a-cluster.md)进行配置。
