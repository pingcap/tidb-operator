---
title: 为已有 TiDB 集群部署异构集群
summary: 本文档介绍如何为已有的 TiDB 集群部署一个异构集群。
---

# 为已有 TiDB 集群部署异构集群

本文档介绍如何为已有的 TiDB 集群部署一个异构集群。

## 前置条件

* 已经存在一个 TiDB 集群，可以参考 [在标准 Kubernetes 上部署 TiDB 集群](deploy-on-general-kubernetes.md)进行部署。

## 部署异构集群

### 什么是异构集群

异构集群是给已经存在的 TiDB 集群创建差异化的实例节点，比如创建不同配置不同 Label 的 TiKV 集群用于热点调度或者创建不同配置的 TiDB 集群分别用于 TP 和 AP 查询。

### 创建一个异构集群

将如下配置存为 `cluster.yaml` 文件，并替换 `${heterogeneous_cluster_name}` 为自己想命名的异构集群名字，`${origin_cluster_name}` 替换为想要加入的已有集群名称:

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: ${heterogeneous_cluster_name}
spec:
  configUpdateStrategy: RollingUpdate
  version: v5.2.0
  timezone: UTC
  pvReclaimPolicy: Delete
  discovery: {}
  cluster:
    name: ${origin_cluster_name}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    # if storageClassName is not set, the default Storage Class of the Kubernetes cluster will be used
    # storageClassName: local-storage
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 1
    replicas: 1
    storageClaims:
      - resources:
          requests:
            storage: 1Gi
        storageClassName: standard
```

执行以下命令创建集群：

{{< copyable "shell-regular" >}}

```shell
kubectl create -f cluster.yaml -n ${namespace}
```

异构集群除了使用 `spec.cluster.name` 字段加入到目标集群，其它字段和正常的 TiDB 集群一样。

### 部署集群监控

将如下配置存为 `tidbmonitor.yaml` 文件，并替换 `${origin_cluster_name}` 为想要加入的集群名称，`${heterogeneous_cluster_name}` 替换为异构集群名称：

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: heterogeneous
spec:
  clusters:
    - name: ${origin_cluster_name}
    - name: ${heterogeneous_cluster_name}
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
  grafana:
    baseImage: grafana/grafana
    version: 6.1.6
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.2.0
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

执行以下命令创建集群：

{{< copyable "shell-regular" >}}

```shell
kubectl create -f tidbmonitor.yaml -n ${namespace}
```

## 部署 TLS 异构集群

开启异构集群 TLS 需要显示声明，需要创建新的 `Secret` 证书文件，使用和目标集群相同的 CA (Certification Authority) 颁发。如果使用 `cert-manager` 方式，需要使用和目标集群相同的 `Issuer` 来创建 `Certificate`。

为异构集群创建证书的详细步骤，可参考以下文档：

- [为 TiDB 组件间开启 TLS](enable-tls-between-components.md)
- [为 MySQL 客户端开启 TLS](enable-tls-for-mysql-client.md)

### 创建一个异构 TLS 集群

将如下配置存为 `cluster.yaml` 文件，并替换 `${heterogeneous_cluster_name}` 为自己想命名的异构集群名字，`${origin_cluster_name}` 替换为想要加入的已有集群名称:

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: ${heterogeneous_cluster_name}
spec:
  tlsCluster:
    enabled: true
  configUpdateStrategy: RollingUpdate
  version: v5.2.0
  timezone: UTC
  pvReclaimPolicy: Delete
  discovery: {}
  cluster:
    name: ${origin_cluster_name}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    # if storageClassName is not set, the default Storage Class of the Kubernetes cluster will be used
    # storageClassName: local-storage
    requests:
      storage: "1Gi"
    config:
      storage:
        # In basic examples, we set this to avoid using too much storage.
        reserve-space: "0MB"
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
    tlsClient:
      enabled: true
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 1
    replicas: 1
    storageClaims:
      - resources:
          requests:
            storage: 1Gi
        storageClassName: standard
```

`spec.tlsCluster.enabled` 表示组件间是否开启 TLS，`spec.tidb.tlsClient.enabled` 表示 MySQL 客户端是否开启 TLS。

执行以下命令创建开启 TLS 的异构集群：

{{< copyable "shell-regular" >}}

```shell
kubectl create -f cluster.yaml -n ${namespace}
```

详细的异构 TLS 集群配置示例，请参阅 ['heterogeneous-tls'](https://github.com/pingcap/tidb-operator/tree/master/examples/heterogeneous-tls)。
