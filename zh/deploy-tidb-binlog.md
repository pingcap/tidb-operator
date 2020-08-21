---
title: 部署 TiDB Binlog
summary: 了解如何在 Kubernetes 上部署 TiDB 集群的 TiDB Binlog。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-tidb-binlog/']
---

# 部署 TiDB Binlog

本文档介绍如何在 Kubernetes 上部署 TiDB 集群的 [TiDB Binlog](https://pingcap.com/docs-cn/stable/tidb-binlog/tidb-binlog-overview/)。

## 部署准备

- [部署 TiDB Operator](deploy-tidb-operator.md)；
- [安装 Helm](tidb-toolkit.md#使用-helm) 并配置 PingCAP 官方 chart 仓库。

## 部署 TiDB 集群的 TiDB Binlog

默认情况下，TiDB Binlog 在 TiDB 集群中处于禁用状态。若要创建一个启用 TiDB Binlog 的 TiDB 集群，或在现有 TiDB 集群中启用 TiDB Binlog，可根据以下步骤进行操作。

### 部署 Pump

可以修改 TidbCluster CR，添加 Pump 相关配置，示例如下：

``` yaml
spec
  ...
  pump:
    baseImage: pingcap/tidb-binlog
    version: v4.0.4
    replicas: 1
    storageClassName: local-storage
    requests:
      storage: 30Gi
    schedulerName: default-scheduler
    config:
      addr: 0.0.0.0:8250
      gc: 7
      heartbeat-interval: 2
```

按照集群实际情况修改 `version`、`replicas`、`storageClassName`、`requests.storage` 等配置。

值得注意的是，如果需要部署企业版的 Pump，需要将 上述 yaml 中 `spec.pump.baseImage` 配置为企业版镜像，格式为 `pingcap/tidb-binlog-enterprise`。

例如:

```yaml
spec:
 pump:
   baseImage: pingcap/tidb-binlog-enterprise
```

如果在生产环境中开启 TiDB Binlog，建议为 TiDB 与 Pump 组件设置亲和性和反亲和性。如果在内网测试环境中尝试使用开启 TiDB Binlog，可以跳过此步。

默认情况下，TiDB 和 Pump 的 affinity 亲和性设置为 `{}`。由于目前 Pump 组件与 TiDB 组件默认并非一一对应，当启用 TiDB Binlog 时，如果 Pump 与 TiDB 组件分开部署并出现网络隔离，而且 TiDB 组件还开启了 `ignore-error`，则会导致 TiDB 丢失 Binlog。推荐通过亲和性特性将 TiDB 组件与 Pump 部署在同一台 Node 上，同时通过反亲和性特性将 Pump 分散在不同的 Node 上，每台 Node 上至多仅需一个 Pump 实例。

* 将 `spec.tidb.affinity` 按照如下设置：

    ```yaml
    spec:
      tidb:
        affinity:
          podAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: "app.kubernetes.io/component"
                    operator: In
                    values:
                    - "pump"
                  - key: "app.kubernetes.io/managed-by"
                    operator: In
                    values:
                    - "tidb-operator"
                  - key: "app.kubernetes.io/name"
                    operator: In
                    values:
                    - "tidb-cluster"
                  - key: "app.kubernetes.io/instance"
                    operator: In
                    values:
                    - ${cluster_name}
                topologyKey: kubernetes.io/hostname
    ```

* 将 `spec.pump.affinity` 按照如下设置：

    ```yaml
    spec:
      pump:
        affinity:
          podAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: "app.kubernetes.io/component"
                    operator: In
                    values:
                    - "tidb"
                  - key: "app.kubernetes.io/managed-by"
                    operator: In
                    values:
                    - "tidb-operator"
                  - key: "app.kubernetes.io/name"
                    operator: In
                    values:
                    - "tidb-cluster"
                  - key: "app.kubernetes.io/instance"
                    operator: In
                    values:
                    - ${cluster_name}
                topologyKey: kubernetes.io/hostname
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: "app.kubernetes.io/component"
                    operator: In
                    values:
                    - "pump"
                  - key: "app.kubernetes.io/managed-by"
                    operator: In
                    values:
                    - "tidb-operator"
                  - key: "app.kubernetes.io/name"
                    operator: In
                    values:
                    - "tidb-cluster"
                  - key: "app.kubernetes.io/instance"
                    operator: In
                    values:
                    - ${cluster_name}
                topologyKey: kubernetes.io/hostname
    ```

> **注意：**
>
> 如果更新了 TiDB 组件的亲和性配置，将引起 TiDB 组件滚动更新。

### 部署 Drainer

可以通过 `tidb-drainer` Helm chart 来为 TiDB 集群部署多个 drainer，示例如下：

1. 确保 PingCAP Helm 库是最新的：

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm search tidb-drainer -l
    ```

2. 获取默认的 `values.yaml` 文件以方便自定义：

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tidb-drainer --version=${chart_version} > values.yaml
    ```

3. 修改 `values.yaml` 文件以指定源 TiDB 集群和 drainer 的下游数据库。示例如下：

    ```yaml
    clusterName: example-tidb
    clusterVersion: v4.0.4
    baseImage: pingcap/tidb-binlog
    storageClassName: local-storage
    storage: 10Gi
    config: |
      detect-interval = 10
      [syncer]
      worker-count = 16
      txn-batch = 20
      disable-dispatch = false
      ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql"
      safe-mode = false
      db-type = "tidb"
      [syncer.to]
      host = "downstream-tidb"
      user = "root"
      password = ""
      port = 4000
    ```

    `clusterName` 和 `clusterVersion` 必须匹配所需的源 TiDB 集群。

    有关完整的配置详细信息，请参阅 [Kubernetes 上的 TiDB Binlog Drainer 配置](configure-tidb-binlog-drainer.md)。

    值得注意的是，如果需要部署企业版的 Drainer，需要将 上述 yaml 中 `baseImage` 配置为企业版镜像，格式为 `pingcap/tidb-binlog-enterprise`。

    例如:

    ```yaml
    ...
    clusterVersion: v4.0.4
    baseImage: pingcap/tidb-binlog-enterprise
    ...
    ```

4. 部署 Drainer：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-drainer --name=${cluster_name} --namespace=${namespace} --version=${chart_version} -f values.yaml
    ```
 
    如果服务器没有外网，请参考 [部署 TiDB 集群](deploy-on-general-kubernetes.md#部署-tidb-集群) 在有外网的机器上将用到的 Docker 镜像下载下来并上传到服务器上。

    > **注意：**
    >
    > 该 chart 必须与源 TiDB 集群安装在相同的命名空间中。

## 开启 TLS

### 为 TiDB 组件间开启 TLS

如果要为 TiDB 集群及 TiDB Binlog 开启 TLS，请参考[为 TiDB 组件间开启 TLS](enable-tls-between-components.md) 进行配置。

创建 secret 并启动包含 Pump 的 TiDB 集群后，修改 `values.yaml` 将 `tlsCluster.enabled` 设置为 true，并配置相应的 `certAllowedCN`：

```yaml
...
tlsCluster:
  enabled: true
  # certAllowedCN:
  #  - TiDB
...
```

### 为 Drainer 和下游数据库间开启 TLS

如果 `tidb-drainer` 的写入下游设置为 `mysql/tidb`，并且希望为 `drainer` 和下游数据库间开启 TLS，可以参考下面步骤进行配置。

首先我们需要创建一个包含下游数据库 TLS 信息的 secret，创建方式如下：

```bash
kubectl create secret generic ${downstream_database_secret_name} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
```

默认情况下，`tidb-drainer` 会将 checkpoint 保存到下游数据库中，所以仅需配置 `tlsSyncer.tlsClientSecretName` 并配置相应的 `certAllowedCN` 即可。

```yaml
tlsSyncer:
  tlsClientSecretName: ${downstream_database_secret_name}
  # certAllowedCN:
  #  - TiDB
```

如果需要将 `tidb-drainer` 的 checkpoint 保存到其他**开启 TLS** 的数据库，需要创建一个包含 checkpoint 数据库的 TLS 信息的 secret，创建方式为：

```bash
kubectl create secret generic ${checkpoint_tidb_client_secret} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
```

修改 `values.yaml` 将 `tlsSyncer.checkpoint.tlsClientSecretName` 设置为 `${checkpoint_tidb_client_secret}`，并配置相应的 `certAllowedCN`：

```yaml
...
tlsSyncer: {}
  tlsClientSecretName: ${downstream_database_secret_name}
  # certAllowedCN:
  #  - TiDB
  checkpoint:
    tlsClientSecretName: ${checkpoint_tidb_client_secret}
    # certAllowedCN:
    #  - TiDB
...
```
