---
title: tidb-cluster chart 配置
summary: 介绍 tidb-cluster chart 配置。
---

# tidb-cluster chart 配置

本文介绍 tidb-cluster chart 配置。

> **注意：**
>
> 对于 TiDBOperator v1.1 及以上版本，不再建议使用 tidb-cluster chart 部署、管理 TiDB 集群，详细信息请参考 [TiDB Operator v1.1 重要注意事项](notes-tidb-operator-v1.1.md)。

## 配置参数

| 参数名 | 说明 | 默认值 |
| :----- | :---- | :----- |
| `rbac.create` | 是否启用 Kubernetes 的 RBAC | `true` |
| `clusterName` | TiDB 集群名，默认不设置该变量，`tidb-cluster` 会直接用执行安装时的 `ReleaseName` 代替 | `nil` |
| `extraLabels` | 添加额外的 labels 到 `TidbCluster` 对象 (CRD) 上，参考：[labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) | `{}` |
| `schedulerName` | TiDB 集群使用的调度器 | `tidb-scheduler` |
| `timezone` | TiDB 集群默认时区 | `UTC` |
| `pvReclaimPolicy` | TiDB 集群使用的 PV (Persistent Volume) 的 reclaim policy | `Retain` |
| `services[0].name` | TiDB 集群对外暴露服务的名字 | `nil` |
| `services[0].type` | TiDB 集群对外暴露服务的类型，（从 `ClusterIP`、`NodePort`、`LoadBalancer` 中选择） | `nil` |
| `discovery.image` | TiDB 集群 PD 服务发现组件的镜像，该组件用于在 PD 集群第一次启动时，为各个 PD 实例提供服务发现功能以协调启动顺序 | `pingcap/tidb-operator:v1.0.0-beta.3` |
| `discovery.imagePullPolicy` | PD 服务发现组件镜像的拉取策略 | `IfNotPresent` |
| `discovery.resources.limits.cpu` | PD 服务发现组件的 CPU 资源限额 | `250m` |
| `discovery.resources.limits.memory` | PD 服务发现组件的内存资源限额 | `150Mi` |
| `discovery.resources.requests.cpu` | PD 服务发现组件的 CPU 资源请求 | `80m` |
| `discovery.resources.requests.memory` | PD 服务发现组件的内存资源请求 | `50Mi` |
| `enableConfigMapRollout` | 是否开启 TiDB 集群自动滚动更新。如果启用，则 TiDB 集群的 ConfigMap 变更时，TiDB 集群自动更新对应组件。该配置只在 TiDB Operator v1.0 及以上版本才支持 | `false` |
| `pd.config` | 配置文件格式的 PD 的配置，请参考 [`pd/conf/config.toml`](https://github.com/pingcap/pd/blob/master/conf/config.toml) 查看默认 PD 配置文件（选择对应 PD 版本的 tag），可以参考 [PD 配置文件描述](https://pingcap.com/docs-cn/stable/pd-configuration-file/)查看配置参数的具体介绍（请选择对应的文档版本），这里只需要**按照配置文件中的格式修改配置** | TiDB Operator 版本 <= v1.0.0-beta.3，默认值为：<br/>`nil`<br/>TiDB Operator 版本 > v1.0.0-beta.3，默认值为：<br/>`[log]`<br/>`level = "info"`<br/>`[replication]`<br/>`location-labels = ["region", "zone", "rack", "host"]`<br/>配置示例：<br/>&nbsp;&nbsp;`config:` \|<br/>&nbsp;&nbsp;&nbsp;&nbsp;`[log]`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`level = "info"`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`[replication]`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`location-labels = ["region", "zone", "rack", "host"]` |
| `pd.replicas` | PD 的 Pod 数 | `3` |
| `pd.image` | PD 镜像 | `pingcap/pd:v3.0.0-rc.1` |
| `pd.imagePullPolicy` | PD 镜像的拉取策略 | `IfNotPresent` |
| `pd.logLevel` | PD 日志级别。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `pd.config` 配置：<br/>`[log]`<br/>`level = "info"` | `info` |
| `pd.storageClassName` | PD 使用的 storageClass，storageClassName 指代一种由 Kubernetes 集群提供的存储类型，不同的类可能映射到服务质量级别、备份策略或集群管理员确定的任意策略。详细参考：[storage-classes](https://kubernetes.io/docs/concepts/storage/storage-classes) | `local-storage` |
| `pd.maxStoreDownTime` | `pd.maxStoreDownTime` 指一个 store 节点断开连接多长时间后状态会被标记为 `down`，当状态变为 `down` 后，store 节点开始迁移数据到其它 store 节点<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `pd.config` 配置：<br/>`[schedule]`<br/>`max-store-down-time = "30m"` | `30m` |
| `pd.maxReplicas` | `pd.maxReplicas` 是 TiDB 集群的数据的副本数。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `pd.config` 配置：<br/>`[replication]`<br/>`max-replicas = 3` | `3` |
| `pd.resources.limits.cpu` | 每个 PD Pod 的 CPU 资源限额 | `nil` |
| `pd.resources.limits.memory` | 每个 PD Pod 的内存资源限额 | `nil` |
| `pd.resources.limits.storage` | 每个 PD Pod 的存储容量限额 | `nil` |
| `pd.resources.requests.cpu` | 每个 PD Pod 的 CPU 资源请求 | `nil` |
| `pd.resources.requests.memory` | 每个 PD Pod 的内存资源请求 | `nil` |
| `pd.resources.requests.storage` | 每个 PD Pod 的存储容量请求 | `1Gi` |
| `pd.affinity` | `pd.affinity` 定义 PD 的调度规则和偏好，详细请参考：[affinity-and-anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) | `{}` |
| `pd.nodeSelector` | `pd.nodeSelector` 确保 PD Pods 只调度到以该键值对作为标签的节点，详情参考：[nodeselector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) | `{}` |
| `pd.tolerations` | `pd.tolerations` 应用于 PD Pods，允许 PD Pods 调度到含有指定 taints 的节点上，详情参考：[taint-and-toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration) | `{}` |
| `pd.annotations` | 为 PD Pods 添加特定的 `annotations` | `{}` |
| `tikv.config` | 配置文件格式的 TiKV 的配置，请参考 [`tikv/etc/config-template.toml`](https://github.com/tikv/tikv/blob/master/etc/config-template.toml) 查看默认 TiKV 配置文件（选择对应 TiKV 版本的 tag），可以参考 [TiKV 配置文件描述](https://pingcap.com/docs-cn/v3.0/reference/configuration/tikv-server/configuration-file/)查看配置参数的具体介绍（请选择对应的文档版本），这里只需要**按照配置文件中的格式修改配置**。<br/><br/>以下两个配置项需要显式配置：<br/><br/>`[storage.block-cache]`<br/>&nbsp;&nbsp;`shared = true`<br/>&nbsp;&nbsp;`capacity = "1GB"`<br/>推荐设置：`capacity` 设置为 `tikv.resources.limits.memory` 的 50%<br/><br/>`[readpool.coprocessor]`<br/>&nbsp;&nbsp;`high-concurrency = 8`<br/>&nbsp;&nbsp;`normal-concurrency = 8`<br/>&nbsp;&nbsp;`low-concurrency = 8`<br/>推荐设置：设置为 `tikv.resources.limits.cpu` 的 80%| TiDB Operator 版本 <= v1.0.0-beta.3，默认值为：<br/>`nil`<br/>TiDB Operator 版本 > v1.0.0-beta.3，默认值为：<br/>`log-level = "info"`<br/>配置示例：<br/>&nbsp;&nbsp;`config:` \|<br/>&nbsp;&nbsp;&nbsp;&nbsp;`log-level = "info"` |
| `tikv.replicas` | TiKV 的 Pod 数 | `3` |
| `tikv.image` | TiKV 的镜像 | `pingcap/tikv:v3.0.0-rc.1` |
| `tikv.imagePullPolicy` | TiKV 镜像的拉取策略 | `IfNotPresent` |
| `tikv.logLevel` | TiKV 的日志级别<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`log-level = "info"` | `info` |
| `tikv.storageClassName` | TiKV 使用的 storageClass，storageClassName 指代一种由 Kubernetes 集群提供的存储类型，不同的类可能映射到服务质量级别、备份策略或集群管理员确定的任意策略。详细参考：[storage-classes](https://kubernetes.io/docs/concepts/storage/storage-classes) | `local-storage` |
| `tikv.syncLog` | syncLog 指是否启用 raft 日志同步功能，启用该功能能保证在断电时数据不丢失。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[raftstore]`<br/>`sync-log = true` | `true` |
| `tikv.grpcConcurrency` | 配置 gRPC server 线程池大小。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[server]`<br/>`grpc-concurrency = 4` | `4` |
| `tikv.resources.limits.cpu` | 每个 TiKV Pod 的 CPU 资源限额 | `nil` |
| `tikv.resources.limits.memory` | 每个 TiKV Pod 的内存资源限额 | `nil` |
| `tikv.resources.limits.storage` | 每个 TiKV Pod 的存储容量限额 | `nil` |
| `tikv.resources.requests.cpu` | 每个 TiKV Pod 的 CPU 资源请求 | `nil` |
| `tikv.resources.requests.memory` | 每个 TiKV Pod 的内存资源请求 | `nil` |
| `tikv.resources.requests.storage` | 每个 TiKV Pod 的存储容量请求 | `10Gi` |
| `tikv.affinity` | `tikv.affinity` 定义 TiKV 的调度规则和偏好，详细请参考：[affinity-and-anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) | `{}` |
| `tikv.nodeSelector` | `tikv.nodeSelector`确保 TiKV Pods 只调度到以该键值对作为标签的节点，详情参考：[nodeselector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) | `{}` |
| `tikv.tolerations` | `tikv.tolerations` 应用于 TiKV Pods，允许 TiKV Pods 调度到含有指定 taints 的节点上，详情参考：[taint-and-toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration) | `{}` |
| `tikv.annotations` | 为 TiKV Pods 添加特定的 `annotations` | `{}` |
| `tikv.defaultcfBlockCacheSize` | 指定 block 缓存大小，block 缓存用于缓存未压缩的 block，较大的 block 缓存设置可以加快读取速度。一般推荐设置为 `tikv.resources.limits.memory` 的 30%-50%<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[rocksdb.defaultcf]`<br/>`block-cache-size = "1GB"`<br/>从 TiKV v3.0.0 开始，不再需要配置 `[rocksdb.defaultcf].block-cache-size` 和 `[rocksdb.writecf].block-cache-size`，改为配置 `[storage.block-cache].capacity` | `1GB` |
| `tikv.writecfBlockCacheSize` | 指定 writecf 的 block 缓存大小，一般推荐设置为 `tikv.resources.limits.memory` 的 10%-30%<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[rocksdb.writecf]`<br/>`block-cache-size = "256MB"`<br/>从 TiKV v3.0.0 开始，不再需要配置 `[rocksdb.defaultcf].block-cache-size` 和 `[rocksdb.writecf].block-cache-size`，改为配置 `[storage.block-cache].capacity` | `256MB` |
| `tikv.readpoolStorageConcurrency` | TiKV 存储的高优先级/普通优先级/低优先级操作的线程池大小<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[readpool.storage]`<br/>`high-concurrency = 4`<br/>`normal-concurrency = 4`<br/>`low-concurrency = 4` | `4` |
| `tikv.readpoolCoprocessorConcurrency` | 一般如果 `tikv.resources.limits.cpu` > 8，则 `tikv.readpoolCoprocessorConcurrency` 设置为`tikv.resources.limits.cpu` * 0.8<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[readpool.coprocessor]`<br/>`high-concurrency = 8`<br/>`normal-concurrency = 8`<br/>`low-concurrency = 8` | `8` |
| `tikv.storageSchedulerWorkerPoolSize` | TiKV 调度程序的工作池大小，应在重写情况下增加，同时应小于总 CPU 核心。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tikv.config` 配置：<br/>`[storage]`<br/>`scheduler-worker-pool-size = 4` | `4` |
| `tidb.config` | 配置文件格式的 TiDB 的配置，请参考[配置文件](https://github.com/pingcap/tidb/blob/master/config/config.toml.example)查看默认 TiDB 配置文件（选择对应 TiDB 版本的 tag），可以参考 [TiDB 配置文件描述](https://pingcap.com/docs-cn/stable/tidb-configuration-file/)查看配置参数的具体介绍（请选择对应的文档版本）。这里只需要**按照配置文件中的格式修改配置**。<br/><br/>以下配置项需要显式配置：<br/><br/>`[performance]`<br/>&nbsp;&nbsp;`max-procs = 0`<br/>推荐设置：`max-procs` 设置为 `tidb.resources.limits.cpu` 对应的核心数 | TiDB Operator 版本 <= v1.0.0-beta.3，默认值为：<br/>`nil`<br/>TiDB Operator 版本 > v1.0.0-beta.3，默认值为：<br/>`[log]`<br/>`level = "info"`<br/>配置示例：<br/>&nbsp;&nbsp;`config:` \|<br/>&nbsp;&nbsp;&nbsp;&nbsp;`[log]`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`level = "info"` |
| `tidb.replicas` | TiDB 的 Pod 数 | `2` |
| `tidb.image` | TiDB 的镜像 | `pingcap/tidb:v3.0.0-rc.1` |
| `tidb.imagePullPolicy` | TiDB 镜像的拉取策略 | `IfNotPresent` |
| `tidb.logLevel` | TiDB 的日志级别。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`[log]`<br/>`level = "info"` | `info` |
| `tidb.resources.limits.cpu` | 每个 TiDB Pod 的 CPU 资源限额 | `nil` |
| `tidb.resources.limits.memory` | 每个 TiDB Pod 的内存资源限额 | `nil` |
| `tidb.resources.requests.cpu` | 每个 TiDB Pod 的 CPU 资源请求 | `nil` |
| `tidb.resources.requests.memory` | 每个 TiDB Pod 的内存资源请求 | `nil` |
| `tidb.passwordSecretName`| 存放 TiDB 用户名及密码的 Secret 的名字，该 Secret 可以使用以下命令创建机密：`kubectl create secret generic tidb secret--from literal=root=${password}--namespace=${namespace}`，如果没有设置，则 TiDB 根密码为空 | `nil` |
| `tidb.initSql`| 在 TiDB 集群启动成功后，会执行的初始化脚本 | `nil` |
| `tidb.affinity` | `tidb.affinity` 定义 TiDB 的调度规则和偏好，详细请参考：[affinity-and-anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) | `{}` |
| `tidb.nodeSelector` | `tidb.nodeSelector`确保 TiDB Pods 只调度到以该键值对作为标签的节点，详情参考：[nodeselector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) | `{}` |
| `tidb.tolerations` | `tidb.tolerations` 应用于 TiDB Pods，允许 TiDB Pods 调度到含有指定 taints 的节点上，详情参考：[taint-and-toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration) | `{}` |
| `tidb.annotations` | 为 TiDB Pods 添加特定的 `annotations` | `{}` |
| `tidb.maxFailoverCount` | TiDB 最大的故障转移数量，假设为 3 即最多支持同时 3 个 TiDB 实例故障转移 | `3` |
| `tidb.service.type` | TiDB 服务对外暴露类型 | `NodePort` |
| `tidb.service.externalTrafficPolicy` | 表示此服务是否希望将外部流量路由到节点本地或集群范围的端点。有两个可用选项：`Cluster`（默认）和 `Local`。`Cluster` 隐藏了客户端源 IP，可能导致流量需要二次跳转到另一个节点，但具有良好的整体负载分布。`Local` 保留客户端源 IP 并避免 LoadBalancer 和 NodePort 类型服务流量的第二次跳转，但存在潜在的不均衡流量传播风险。详细参考：[外部负载均衡器](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) | `nil` |
| `tidb.service.loadBalancerIP` | 指定 TiDB 负载均衡 IP，某些云提供程序允许您指定 loadBalancerIP。在这些情况下，将使用用户指定的 loadBalancerIP 创建负载平衡器。如果未指定 loadBalancerIP 字段，则将使用临时 IP 地址设置 loadBalancer。如果指定 loadBalancerIP 但云提供程序不支持该功能，则将忽略您设置的 loadbalancerIP 字段 | `nil` |
| `tidb.service.mysqlNodePort` | TiDB 服务暴露的 mysql NodePort 端口 |  |
| `tidb.service.exposeStatus` | TiDB 服务是否暴露状态端口 | `true` |
| `tidb.service.statusNodePort` | 指定 TiDB 服务的状态端口暴露的 `NodePort` |  |
| `tidb.separateSlowLog` | 是否以 sidecar 方式运行独立容器输出 TiDB 的 SlowLog | 如果 TiDB Operator 版本 <= v1.0.0-beta.3，默认值为 `false`。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，默认值为 `true` |
| `tidb.slowLogTailer.image` | TiDB 的 slowLogTailer 的镜像，slowLogTailer 是一个 sidecar 类型的容器，用于输出 TiDB 的 SlowLog，该配置仅在 `tidb.separateSlowLog`=`true` 时生效 | `busybox:1.26.2` |
| `tidb.slowLogTailer.resources.limits.cpu` | 每个 TiDB Pod 的 slowLogTailer 的 CPU 资源限额 | `100m` |
| `tidb.slowLogTailer.resources.limits.memory` | 每个 TiDB Pod 的 slowLogTailer 的内存资源限额 | `50Mi` |
| `tidb.slowLogTailer.resources.requests.cpu` | 每个 TiDB Pod 的 slowLogTailer 的 CPU 资源请求 | `20m` |
| `tidb.slowLogTailer.resources.requests.memory` | 每个 TiDB Pod 的 slowLogTailer 的内存资源请求 | `5Mi` |
| `tidb.plugin.enable` | 是否启用 TiDB 插件功能 | `false` |
| `tidb.plugin.directory` | 指定 TiDB 插件所在的目录 | `/plugins` |
| `tidb.plugin.list` | 指定 TiDB 加载的插件列表，plugin ID 命名规则：插件名-版本，例如：'conn_limit-1' | `[]` |
| `tidb.preparedPlanCacheEnabled` | 是否启用 TiDB 的 prepared plan 缓存。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`[prepared-plan-cache]`<br/>`enabled = false` | `false` |
| `tidb.preparedPlanCacheCapacity` | TiDB 的 prepared plan 缓存数量。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`[prepared-plan-cache]`<br/>`capacity = 100` | `100` |
| `tidb.txnLocalLatchesEnabled` | 是否启用事务内存锁，当本地事务冲突比较多时建议开启。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`[txn-local-latches]`<br/>`enabled = false` | `false` |
| `tidb.txnLocalLatchesCapacity` | 事务内存锁的容量，Hash 对应的 slot 数，会自动向上调整为 2 的指数倍。每个 slot 占 32 Bytes 内存。当写入数据的范围比较广时（如导数据），设置过小会导致变慢，性能下降。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`[txn-local-latches]`<br/>`capacity = 10240000` | `10240000` |
| `tidb.tokenLimit` | TiDB 并发执行会话的限制。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`token-limit = 1000` | `1000` |
| `tidb.memQuotaQuery` | TiDB 查询的内存限额，默认 32GB。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`mem-quota-query = 34359738368` | `34359738368` |
| `tidb.checkMb4ValueInUtf8` | 用于控制当字符集为 utf8 时是否检查 mb4 字符。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`check-mb4-value-in-utf8 = true` | `true` |
| `tidb.treatOldVersionUtf8AsUtf8mb4` | 用于升级兼容性。设置为 `true` 将把旧版本的表/列的 `utf8` 字符集视为 `utf8mb4` 字符集<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`treat-old-version-utf8-as-utf8mb4 = true` | `true` |
| `tidb.lease` | `tidb.lease`是 TiDB Schema lease 的期限，对其更改是非常危险的，除非你明确知道可能产生的结果，否则不建议更改。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`lease = "45s"` | `45s` |
| `tidb.maxProcs` | 最大可使用的 CPU 核数，0 代表机器/Pod 上的 CPU 数量。<br/>如果 TiDB Operator 版本 > v1.0.0-beta.3，请通过 `tidb.config` 配置：<br/>`[performance]`<br/>`max-procs = 0` | `0` |
