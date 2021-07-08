---
title: 配置 TiDB 集群
summary: 了解如何在 Kubernetes 中配置 TiDB 集群。
category: how-to
aliases: ['/docs-cn/tidb-in-kubernetes/dev/configure-a-tidb-cluster/','/zh/tidb-in-kubernetes/dev/configure-cluster-using-tidbcluster/','/docs-cn/tidb-in-kubernetes/dev/configure-cluster-using-tidbcluster/']
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

通过配置 `TidbCluster` CR 来配置 TiDB 集群。参考 TidbCluster [示例](https://github.com/pingcap/tidb-operator/blob/master/examples/advanced/tidb-cluster.yaml)和 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)（示例和 API 文档请切换到当前使用的 TiDB Operator 版本）完成 TidbCluster CR(Custom Resource)。

> **注意：**
>
> 建议在 `${cluster_name}` 目录下组织 TiDB 集群的配置，并将其另存为 `${cluster_name}/tidb-cluster.yaml`。默认条件下，修改配置不会自动应用到 TiDB 集群中，只有在 Pod 重启时，才会重新加载新的配置文件。

### 集群名称

通过更改 `TiDBCuster` CR 中的 `metadata.name` 来配置集群名称。

### 版本

正常情况下，集群内的各组件应该使用相同版本，所以一般建议配置 `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.baseImage` + `spec.version` 即可。如果需要为不同的组件配置不同的版本，则可以配置 `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`。

相关参数的格式如下：

- `spec.version`，格式为 `imageTag`，例如 `v5.1.0`
- `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.baseImage`，格式为 `imageName`，例如 `pingcap/tidb`
- `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`，格式为 `imageTag`，例如 `v5.1.0`

### 推荐配置

#### configUpdateStrategy

建议设置 `spec.configUpdateStrategy: RollingUpdate`，开启配置自动更新特性，在每次配置更新时，自动对组件执行滚动更新，将修改后的配置应用到集群中。

#### enableDynamicConfiguration

建议通过设置 `spec.enableDynamicConfiguration: true` 配置 TiKV 的 `--advertise-status-addr` 启动参数。

版本支持：TiDB v4.0.1 及更高版本。

#### pvReclaimPolicy

建议设置 `spec.pvReclaimPolicy: Retain`，确保 PVC 被删除后 PV 仍然保留，保证数据安全。

#### mountClusterClientSecret

PD 和 TiKV 支持配置 `mountClusterClientSecret`。如果开启了[集群组件间 TLS 支持](enable-tls-between-components.md)，建议配置 `spec.pd.mountClusterClientSecret: true` 和 `spec.tikv.mountClusterClientSecret: true`，这样 TiDB Operator 会自动将 `${cluster_name}-cluster-client-secret` 证书挂载到 PD 和 TiKV 容器，方便[使用 `pd-ctl` 和 `tikv-ctl`](enable-tls-between-components.md#第三步配置-pd-ctltikv-ctl-连接集群)。

### 存储

#### 存储类型

如果需要设置存储类型，可以修改 `${cluster_name}/tidb-cluster.yaml` 中各组件的 `storageClassName` 字段。关于 Kubernetes 集群支持哪些[存储类型](https://kubernetes.io/zh/docs/concepts/storage/storage-classes/)，请联系系统管理员确定。

另外，TiDB 集群不同组件对磁盘的要求不一样，所以部署集群前，要根据当前 Kubernetes 集群支持的存储类型以及使用场景，参考[存储配置文档](configure-storage-class.md)为 TiDB 集群各组件选择合适的存储类型。

> **注意：**
>
> 如果创建 TiDB 集群时设置了 Kubernetes 集群中不存在的存储类型，则会导致 TiDB 集群创建处于 Pending 状态，需要[将 TiDB 集群彻底销毁掉](destroy-a-tidb-cluster.md)，再进行重试。

#### 多盘挂载

TiDB Operator 支持为 PD、TiDB、TiKV、TiCDC 挂载多块 PV，可以用于不同用途的数据写入。

每个组件都可以配置 `storageVolumes` 字段，用于描述用户自定义的多个 PV。

> **注意：**
>
> 你需要在集群创建之前配置 `storageVolumes`。集群创建完成后，不支持添加或者删除 `storageVolumes`。对于已经配置的 `storageVolumes`，除增大 `storageVolume.storageSize` 外，其他项不支持修改。如果要增大 `storageVolume.storageSize`，需要对应的 StorageClass 支持[动态扩容](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/)。

相关字段的含义如下：

- `storageVolume.name`：PV 的名称。
- `storageVolume.storageClassName`：PV 使用哪一个 StorageClass。如果不配置，会使用 spec.pd/tidb/tikv.storageClassName。
- `storageVolume.storageSize`：申请 PV 存储容量的大小。
- `storageVolume.mountPath`：将 PV 挂载到容器的哪个目录。

例子:

{{< copyable "" >}}

```yaml
  pd:
    baseImage: pingcap/pd
    replicas: 1
    # if storageClassName is not set, the default Storage Class of the Kubernetes cluster will be used
    # storageClassName: local-storage
    requests:
      storage: "1Gi"
    config:
      log:
        file:
          filename: /var/log/pdlog/pd.log
        level: "warn"
    storageVolumes:
    - name: log
      storageSize: "2Gi"
      mountPath: "/var/log/pdlog"
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config:
      log:
        file:
          filename: /var/log/tidblog/tidb.log
        level: "warn"
    storageVolumes:
    - name: log
      storageSize: "2Gi"
      mountPath: "/var/log/tidblog"
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    # if storageClassName is not set, the default Storage Class of the Kubernetes cluster will be used
    # storageClassName: local-storage
    requests:
      storage: "1Gi"
    config:
      storage:
        # In basic examples, you can set this to avoid using too much storage.
        reserve-space: "0MB"
      rocksdb:
        wal-dir: "/data_sbi/tikv/wal"
      titan:
        dirname: "/data_sbj/titan/data"
    storageVolumes:
    - name: wal
      storageSize: "2Gi"
      mountPath: "/data_sbi/tikv/wal"
    - name: titan
      storageSize: "2Gi"
      mountPath: "/data_sbj/titan/data"
```

> **注意：**
>
> TiDB Operator 默认会使用一些挂载路径，比如会为 TiDB Pod 挂载 `EmptyDir` 到 `/var/log/tidb` 目录。在配置 `storageVolumes` 的时候要避免配置重复的 `mountPath`。

### HostNetwork

PD、TiKV、TiDB、TiFlash、TiCDC 及 Pump 支持配置 Pod 使用宿主机上的网络命名空间 [`HostNetwork`](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#host-namespaces)。可通过配置 `spec.hostNetwork: true` 为所有受支持的组件开启，或通过为特定组件配置 `hostNetwork: true` 为单个或多个组件开启。

### Discovery

TiDB Operator 会为每一个 TiDB 集群启动一个 Discovery 服务。Discovery 服务会为每个 PD Pod 返回相应的启动参数，来辅助 PD 集群启动。你可以通过 `spec.discovery` 配置 Discovery 服务的资源，详见[容器资源管理](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)。

`spec.discovery` 配置示例：

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  version: v5.1.0
  pvReclaimPolicy: Retain
  discovery:
    limits:
      cpu: "0.2"
    requests:
      cpu: "0.2"
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
...
```

### 集群拓扑

#### PD/TiKV/TiDB

默认示例的集群拓扑是：3 个 PD Pod，3 个 TiKV Pod，2 个 TiDB Pod。在该部署拓扑下根据数据高可用原则，TiDB Operator 扩展调度器要求 Kubernetes 集群中至少有 3 个节点。可以修改 `replicas` 配置来更改每个组件的 Pod 数量。

> **注意：**
>
> 如果 Kubernetes 集群节点个数少于 3 个，将会导致有一个 PD Pod 处于 Pending 状态，而 TiKV 和 TiDB Pod 也都不会被创建。Kubernetes 集群节点个数少于 3 个时，为了使 TiDB 集群能启动起来，可以将默认部署的 PD Pod 个数减小到 1 个。

#### 部署 TiFlash

如果要在集群中开启 TiFlash，需要在 `${cluster_name}/tidb-cluster.yaml` 文件中配置 `spec.pd.config.replication.enable-placement-rules: true`，并配置 `spec.tiflash`：

```yaml
  pd:
    config: |
      ...
      [replication]
      enable-placement-rules = true
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

所有 PV 按照配置先后顺序分别挂载到容器内的 `/data0`、`/data1` 等目录。TiFlash 有 4 个日志文件，其中 Proxy 日志打印到容器标准输出，另外 3 个日志存储在硬盘中，默认存储在 `/data0` 目录下，分别为 `/data0/logs/flash_cluster_manager.log`、`/data0/logs/error.log`、`/data0/logs/server.log`，如果要修改日志存储路径，可以参考[配置 TiFlash 配置参数](#配置-tiflash-配置参数)进行修改。

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

本节介绍如何配置 TiDB/TiKV/PD/TiFlash/TiCDC 的配置选项。

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
    image: pingcap/tidb:v5.1.0
    imagePullPolicy: IfNotPresent
    replicas: 1
    service:
      type: ClusterIP
    config: |
      split-table = true
      oom-action = "log"
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
    image: pingcap/tikv:v5.0.1
    config: |
      [storage]
        [storage.block-cache]
          capacity = "16GB"
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
    image: pingcap/pd:v5.0.1
    config: |
      lease = 3
      enable-prevote = true
```

获取所有可以配置的 PD 配置参数，请参考 [PD 配置文档](https://pingcap.com/docs-cn/stable/pd-configuration-file/)

> **注意：**
>
> - 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 PD 组件无法正常启动。
> - PD 部分配置项在首次启动成功后会持久化到 etcd 中且后续将以 etcd 中的配置为准。因此 PD 在首次启动后，这些配置项将无法再通过配置参数来进行修改，而需要使用 SQL、pd-ctl 或 PD server API 来动态进行修改。目前，[在线修改 PD 配置](https://docs.pingcap.com/zh/tidb/stable/dynamic-config#在线修改-pd-配置)文档中所列的配置项中，除 `log.level` 外，其他配置项在 PD 首次启动之后均不再支持通过配置参数进行修改。

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
      config: |
        [flash]
          [flash.flash_cluster]
            log = "/data0/logs/flash_cluster_manager.log"
        [logger]
          count = 10
          level = "information"
          errorlog = "/data0/logs/error.log"
          log = "/data0/logs/server.log"
```

获取所有可以配置的 TiFlash 配置参数，请参考 [TiFlash 配置文档](https://pingcap.com/docs-cn/stable/tiflash/tiflash-configuration/)

#### 配置 TiCDC 启动参数

你可以通过 TidbCluster CR 的 `spec.ticdc.config` 来配置 TiCDC 启动参数。

对于 TiDB Operator v1.2.0-rc.2 及之后版本，请使用 TOML 格式配置：

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  ...
  ticdc:
    config: |
      gc-ttl = 86400
      log-level = "info"
```

对于 TiDB Operator v1.2.0-rc.2 之前版本，请使用 YAML 格式配置：

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

获取所有可以配置的 TiCDC 启动参数，请参考 [TiCDC 启动参数文档](https://github.com/pingcap/ticdc/blob/master/cmd/ticdc.toml)。

### 配置 TiDB 平滑升级

滚动更新 TiDB 集群的过程中，在停止 TiDB Pod 之前，Kubernetes 会向 TiDB server 进程发送一个 [`TERM`](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination) 信号。在收到 `TERM` 信号后，TiDB server 会尝试等待所有的连接关闭，不过 15 秒后会强制关闭所有连接并退出进程。

通过配置下面两个属性来实现平滑升级 TiDB 集群：

- `spec.tidb.terminationGracePeriodSeconds`：滚动更新的时候，删除旧的 TiDB Pod 最多容忍的时间，即过了这个时间，TiDB Pod 会被强制删除；
- `spec.tidb.lifecycle`：设置 TiDB Pod 的 `preStop` Hook，在 TiDB server 停止之前执行的操作。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  version: v5.1.0
  pvReclaimPolicy: Retain
  discovery: {}
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
    terminationGracePeriodSeconds: 60
    lifecycle:
      preStop:
        exec:
          command:
          - /bin/sh
          - -c
          - "sleep 10 && kill -QUIT 1"
```

上述 YAML 文件中：

- 设置了删除 TiDB Pod 的最多容忍时间为 60 秒，如果 60 秒之内客户端仍然没有关闭连接的话，那么这些连接将会强制关闭。这个时间可根据需要进行调整；
- 设置 `preStop` Hook 为 `sleep 10 && kill -QUIT 1`，这里 Pid 1 为 TiDB Pod 内 TiDB server 进程的 Pid。TiDB server 进程收到这个信号之后，会等待所有连接被客户端关闭之后才会退出。

Kubernetes 在删除 TiDB Pod 的同时，也会把该 TiDB 节点从 Service 的 Endpoints 中移除。这样就可以保证新的连接不会连接到该 TiDB 节点，但是由于此过程是异步的，所以可以在发送 Kill 信号之前 sleep 几秒钟，确保该 TiDB 节点从 Endpoints 中去掉。

### 配置 TiKV 平滑升级

TiKV 升级过程中，在重启 TiKV Pod 之前，TiDB Operator 会先驱逐 TiKV Pod 上的所有 Region leader。只有当驱逐完成（即 TiKV Pod 上的 Region leader 个数为 0）或者驱逐超时（默认 10 分钟）后，TiKV Pod 才会重启。

如果驱逐 Region leader 超时，重启 TiKV Pod 会导致部分请求失败或者延时增加。要避免此问题，你可以将超时时间 `spec.tikv.evictLeaderTimeout`（默认 10 分钟）配置为一个更大的值，例如：

```
spec:
  tikv:
    evictLeaderTimeout: 10000m
```

> **警告：**
>
> 如果使用 TiKV 版本小于 4.0.14，或者小于 5.0.3，由于 [TiKV 的 bug](https://github.com/tikv/tikv/pull/10364)，需要将 `spec.tikv.evictLeaderTimeout` 的值设置的尽可能大（推荐大于 `1500m`），以保证 TiKV Pod 上所有的 Region Leader 能在设置的时间内驱逐完毕。

### 配置 TiDB 慢查询日志持久卷

默认配置下，TiDB Operator 会新建名称为 `slowlog` 的 `EmptyDir` 卷来存储慢查询日志，`slowlog` 卷默认挂载到 `/var/log/tidb`，慢查询日志通过 sidecar 容器打印到标准输出。

> **警告：**
>
> 默认配置下，使用 `EmptyDir` 卷存储的慢查询日志会在 Pod 被删除（例如，滚动升级）后丢失。请确保 Kubernetes 集群内已经部署日志收集方案用于收集所有容器的日志。如果没有部署日志收集方案，请**务必**通过下面配置使用持久卷来存储慢查询日志。

如果想使用单独的持久卷来存储慢查询日志，可以通过配置 `spec.tidb.slowLogVolumeName` 单独指定存储慢查询日志的持久卷名称，并在 `spec.tidb.storageVolumes` 或 `spec.tidb.additionalVolumes` 配置持久卷信息。下面分别演示使用 `spec.tidb.storageVolumes` 和 `spec.tidb.additionalVolumes` 配置持久卷。

#### spec.tidb.storageVolumes 配置

按照如下示例配置 `TidbCluster` CR，TiDB Operator 将使用持久卷 `${volumeName}` 存储慢查询日志，日志文件路径为：`${mountPath}/${volumeName}`。`spec.tidb.storageVolumes` 字段的具体配置方式可参考[多盘挂载](#多盘挂载)。

{{< copyable "" >}}

```yaml
  tidb:
    ...
    separateSlowLog: true  # 可省略
    slowLogVolumeName: ${volumeName}
    storageVolumes:
      # name 必须和 slowLogVolumeName 字段的值保持一致
      - name: ${volumeName}
        storageClassName: ${storageClass}
        storageSize: "1Gi"
        mountPath: ${mountPath}
```

#### spec.tidb.additionalVolumes 配置

下面以 NFS 为例配置 `spec.tidb.additionalVolumes`。TiDB Operator 将使用持久卷 `${volumeName}` 存储慢查询日志，日志文件路径为：`${mountPath}/${volumeName}`。具体支持的持久卷类型可参考 [Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes)。

{{< copyable "" >}}

```yaml
  tidb:
    ...
    separateSlowLog: true  # 可省略
    slowLogVolumeName: ${volumeName}
    additionalVolumes:
    # name 必须和 slowLogVolumeName 字段的值保持一致
    - name: ${volumeName}
      nfs:
        server: 192.168.0.2
        path: /nfs
    additionalVolumeMounts:
    # name 必须和 slowLogVolumeName 字段的值保持一致
    - name: ${volumeName}
      mountPath: ${mountPath}
```

### 配置 TiDB 服务

需要配置 `spec.tidb.service`，TiDB Operator 才会为 TiDB 创建 Service。Service 可以根据场景配置不同的类型，比如 `ClusterIP`、`NodePort`、`LoadBalancer` 等。

#### ClusterIP

`ClusterIP` 是通过集群的内部 IP 暴露服务，选择该类型的服务时，只能在集群内部访问，使用 ClusterIP 或者 Service 域名（`${cluster_name}-tidb.${namespace}`）访问。

```yaml
spec:
  ...
  tidb:
    service:
      type: ClusterIP
```

#### NodePort

在没有 LoadBalancer 时，可选择通过 NodePort 暴露。NodePort 是通过节点的 IP 和静态端口暴露服务。通过请求 `NodeIP + NodePort`，可以从集群的外部访问一个 NodePort 服务。

```yaml
spec:
  ...
  tidb:
    service:
      type: NodePort
      # externalTrafficPolicy: Local
```

NodePort 有两种模式：

- `externalTrafficPolicy=Cluster`：集群所有的机器都会给 TiDB 分配 NodePort 端口，此为默认值

    使用 `Cluster` 模式时，可以通过任意一台机器的 IP 加 NodePort 访问 TiDB 服务，如果该机器上没有 TiDB Pod，则相应请求会转发到有 TiDB Pod 的机器上。

    > **注意：**
    >
    > 该模式下 TiDB 服务获取到的请求源 IP 是主机 IP，并不是真正的客户端源 IP，所以基于客户端源 IP 的访问权限控制在该模式下不可用。

- `externalTrafficPolicy=Local`：只有运行 TiDB 的机器会分配 NodePort 端口，用于访问本地的 TiDB 实例

#### LoadBalancer

若运行在有 LoadBalancer 的环境，比如 GCP/AWS 平台，建议使用云平台的 LoadBalancer 特性。

```yaml
spec:
  ...
  tidb:
    service:
      annotations:
        cloud.google.com/load-balancer-type: "Internal"
      externalTrafficPolicy: Local
      type: LoadBalancer
```

访问 [Kubernetes Service 文档](https://kubernetes.io/docs/concepts/services-networking/service/)，了解更多 Service 特性以及云平台 Load Balancer 支持。

## 高可用配置

> **注意：**
>
> TiDB Operator 提供了自定义的调度器，该调度器通过指定的调度算法能在 host 层面保证 TiDB 服务的高可用。目前，TiDB 集群使用该调度器作为默认调度器，可通过 `spec.schedulerName` 配置项进行设置。本节重点介绍如何配置 TiDB 集群以容忍其他级别的故障，例如机架、可用区或 region。本部分可根据使用需求配置，不是必选。

TiDB 是分布式数据库，它的高可用需要做到在任一个物理拓扑节点发生故障时，不仅服务不受影响，还要保证数据也是完整和可用。下面分别具体说明这两种高可用的配置。

### TiDB 服务高可用

#### 通过 affinity 调度实例

配置 `PodAntiAffinity` 能尽量避免同一组件的不同实例部署到同一个物理拓扑节点上，从而达到高可用的目的。关于 Affinity 的使用说明，请参阅 [Affinity & AntiAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)。

下面是一个典型的高可用设置例子：

{{< copyable "" >}}

```yaml
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

#### 通过 topologySpreadConstraints 实现 Pod 均匀分布

配置 `topologySpreadConstraints` 可以实现同一组件的不同实例在拓扑上的均匀分布。具体配置方法请参阅 [Pod Topology Spread Constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/)。

> **注意：**
>
> 配置 `topologySpreadConstraints` 前，你需要开启 `EvenPodsSpread` feature gate。如果 Kubernetes 版本低于 v1.16 或者 `EvenPodsSpread` feature gate 未开启，`topologySpreadConstraints` 的配置将不会生效。

`topologySpreadConstraints` 可以设置在整个集群级别 (`spec.topologySpreadConstraints`) 来配置所有组件或者设置在组件级别 (例如 `spec.tidb.topologySpreadConstraints`) 来配置特定的组件。

以下是一个配置示例：

{{< copyable "" >}}

```yaml
topologySpreadConstraints:
- topologyKey: kubernetes.io/hostname
- topologyKey: topology.kubernetes.io/zone
```

该配置能让同一组件的不同实例均匀分布在不同 zone 和节点上。

当前 `topologySpreadConstraints` 仅支持 `topologyKey` 配置。在 Pod spec 中，上述示例配置会自动展开成如下配置：

```yaml
topologySpreadConstraints:
- topologyKey: kubernetes.io/hostname
  maxSkew: 1
  whenUnsatisfiable: DoNotSchedule
  labelSelector: <object>
- topologyKey: topology.kubernetes.io/zone
  maxSkew: 1
  whenUnsatisfiable: DoNotSchedule
  labelSelector: <object>
```

> **注意：**
>
> 可以用该功能替换 [TiDB Scheduler](tidb-scheduler.md) 来实现均匀调度。

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
