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
    version: v5.2.1
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

自 v1.1.6 版本起支持透传 TOML 配置给组件:

```yaml
spec
  ...
  pump:
    baseImage: pingcap/tidb-binlog
    version: v5.2.1
    replicas: 1
    storageClassName: local-storage
    requests:
      storage: 30Gi
    schedulerName: default-scheduler
    config: |
      addr = "0.0.0.0:8250"
      gc = 7
      heartbeat-interval = 2
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
    helm search repo tidb-drainer -l
    ```

2. 获取默认的 `values.yaml` 文件以方便自定义：

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tidb-drainer --version=${chart_version} > values.yaml
    ```

3. 修改 `values.yaml` 文件以指定源 TiDB 集群和 drainer 的下游数据库。示例如下：

    ```yaml
    clusterName: example-tidb
    clusterVersion: v5.2.1
    baseImage: pingcap/tidb-binlog
    storageClassName: local-storage
    storage: 10Gi
    initialCommitTs: "-1"
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

    `initialCommitTs` 为 drainer 没有 checkpoint 时数据同步的起始 commit timestamp。该参数值必须以 string 类型配置，如 `"424364429251444742"`。

    有关完整的配置详细信息，请参阅 [Kubernetes 上的 TiDB Binlog Drainer 配置](configure-tidb-binlog-drainer.md)。

    值得注意的是，如果需要部署企业版的 Drainer，需要将 上述 yaml 中 `baseImage` 配置为企业版镜像，格式为 `pingcap/tidb-binlog-enterprise`。

    例如:

    ```yaml
    ...
    clusterVersion: v5.2.1
    baseImage: pingcap/tidb-binlog-enterprise
    ...
    ```

4. 部署 Drainer：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${release_name} pingcap/tidb-drainer --namespace=${namespace} --version=${chart_version} -f values.yaml
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

## 缩容/移除 Pump/Drainer 节点

如需详细了解如何维护 TiDB Binlog 集群节点状态信息，可以参考 [Pump/Drainer 的启动、退出流程](https://docs.pingcap.com/zh/tidb/stable/maintain-tidb-binlog-cluster#pumpdrainer-的启动退出流程)。

如果需要完整移除 TiDB Binlog 组件，最好是先移除 Pump 节点，再移除 Drainer 节点。

如果需要移除的 TiDB Binlog 组件开启了 TLS，则需要先将下述文件写入 `binlog.yaml`，并使用 `kubectl apply -f binlog.yaml` 启动一个挂载了 TLS 文件和 binlogctl 工具的 Pod。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: binlogctl
spec:
  containers:
  - name: binlogctl
    image: pingcap/tidb-binlog:${tidb_version}
    command: ['/bin/sh']
    stdin: true
    stdinOnce: true
    tty: true
    volumeMounts:
      - name: binlog-tls
        mountPath: /etc/binlog-tls
  volumes:
    - name: binlog-tls
      secret:
        secretName: ${cluster_name}-cluster-client-secret
```

### 缩容 Pump 节点

缩容 Pump 需要先将单个 Pump 节点从集群中下线，然后运行 `kubectl edit tc ${cluster_name} -n ${namespace}` 命令将 Pump 对应的 replica 数量减 1，并对每个节点重复上述步骤。具体操作步骤如下：

1. 下线 Pump 节点：

    假设现在有 3 个 Pump 节点，我们需要下线第 3 个 Pump 节点，将 `${ordinal_id}` 替换成 `2`，操作方式如下（`${tidb_version}` 为当前 TiDB 的版本）。

    如果 Pump 没有开启 TLS，使用下述指令新建 Pod 下线 Pump。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl run offline-pump-${ordinal_id} --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd offline-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250
    ```

    如果 Pump 开启了 TLS，通过下述指令使用前面开启的 Pod 来下线 Pump。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls "https://${cluster_name}-pd:2379" -cmd offline-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250 -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
    ```

    然后查看 Pump 的日志输出，输出 `pump offline, please delete my pod` 后即可确认该节点已经成功下线。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl logs -f -n ${namespace} ${release_name}-pump-${ordinal_id}
    ```

2. 删除对应的 Pump Pod：

    运行 `kubectl edit tc ${cluster_name} -n ${namespace}` 修改文件中 `spec.pump.replicas` 为 `2`，然后等待 Pump Pod 自动下线被删除。

3. (可选项) 强制下线 Pump

    如果在下线 Pump 节点时遇到下线失败的情况，即执行下线操作后仍未看到 Pump pod 输出可以删除 pod 的日志，可以先进行步骤 2 调小 `replicas`， 等待 Pump Pod 被完全删除后，标注 Pump 状态为 offline。
    
    没有开启 TLS 时，使用下述指令标注状态为 offline。
    
    {{< copyable "shell-regular" >}}
    
    ```shell
    kubectl run update-pump-${ordinal_id} --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd update-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250 --state offline
    ```
    
    如果开启了 TLS，通过下述指令使用前面开启的 pod 来标注状态为 offline。
    
    {{< copyable "shell-regular" >}}
    
    ```shell
    kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls=https://${cluster_name}-pd:2379 -cmd update-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250 --state offline -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
    ```

### 完全移除 Pump 节点

> **注意：**
>
> 执行如下步骤之前，集群内需要至少存在一个 Pump 节点。如果此时 Pump 节点已经缩容到 0，需要先至少扩容到 1，再进行下面的移除操作。如果需要扩容至 1，使用命令 `kubectl edit tc ${tidb-cluster} -n ${namespace}`，修改 `spec.pump.replicas` 为 1 即可。

1. 移除 Pump 节点前，必须首先需要执行 `kubectl edit tc ${cluster_name} -n ${namespace}` 设置其中的 `spec.tidb.binlogEnabled` 为 `false`，等待 TiDB Pod 完成重启更新后再移除 Pump 节点。如果直接移除 Pump 节点会导致 TiDB 没有可以写入的 Pump 而无法使用。
2. 参考[缩容 Pump 节点步骤](#缩容-pump-节点)缩容 Pump 到 0。
3. `kubectl edit tc ${cluster_name} -n ${namespace}` 将 `spec.pump` 部分配置项全部删除。
4. `kubectl delete sts ${cluster_name}-pump -n ${namespace}` 删除 Pump StatefulSet 资源。
5. 通过 `kubectl get pvc -n ${namespace} -l app.kubernetes.io/component=pump` 查看 Pump 集群使用过的 PVC，随后使用 `kubectl delete pvc -l app.kubernetes.io/component=pump -n ${namespace}` 指令删除 Pump 的所有 PVC 资源。

### 移除 Drainer 节点

1. 下线 Drainer 节点：

    使用下述指令下线 Drainer 节点，`${drainer_node_id}` 为需要下线的 Drainer 的 node ID。如果在 Helm 的 `values.yaml` 中配置了 `drainerName` 选项，则 `${drainer_node_id}` 为 `${drainer_name}-0`，否则 `${drainer_node_id}` 为 `${cluster_name}-${release_name}-drainer-0`。

    如果 Drainer 没有开启 TLS，使用下述指令新建 pod 下线 Drainer。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl run offline-drainer-0 --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd offline-drainer -node-id ${drainer_node_id}:8249
    ```

    如果 Drainer 开启了 TLS，通过下述指令使用前面开启的 pod 来下线 Drainer。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls "https://${cluster_name}-pd:2379" -cmd offline-drainer -node-id ${drainer_node_id}:8249 -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
    ```

    然后查看 Drainer 的日志输出，输出 `drainer offline, please delete my pod` 后即可确认该节点已经成功下线。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl logs -f -n ${namespace} ${drainer_node_id}
    ```

2. 删除对应的 Drainer Pod：

    运行 `helm uninstall ${release_name} -n ${namespace}` 指令即可删除 Drainer Pod。

    如果不再使用 Drainer，使用 `kubectl delete pvc data-${drainer_node_id} -n ${namespace}` 指令删除该 Drainer 的 PVC 资源。

3. (可选项) 强制下线 Drainer

    如果在下线 Drainer 节点时遇到下线失败的情况，即执行下线操作后仍未看到 Drainer pod 输出可以删除 pod 的日志，可以先进行步骤 2 删除 Drainer Pod 后，再运行下述指令标注 Drainer 状态为 offline：
    
    没有开启 TLS 时，使用下述指令标注状态为 offline。
    
    {{< copyable "shell-regular" >}}
    
    ```shell
    kubectl run update-drainer-${ordinal_id} --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd update-drainer -node-id ${drainer_node_id}:8249 --state offline
    ```
    
    如果开启了 TLS，通过下述指令使用前面开启的 pod 来下线 Drainer。
    
    {{< copyable "shell-regular" >}}
    
    ```shell
    kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls=https://${cluster_name}-pd:2379 -cmd update-drainer -node-id ${drainer_node_id}:8249 --state offline -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
    ```
