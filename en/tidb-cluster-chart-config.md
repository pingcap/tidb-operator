---
title: Configuration of tidb-cluster Chart
summary: Learn the configurations of the tidb-cluster chart.
aliases: ['/docs/tidb-in-kubernetes/dev/tidb-cluster-chart-config/']
---

# Configuration of tidb-cluster Chart

This document describes the configuration of the tidb-cluster chart.

> **Note:**
>
> For TiDB Operator v1.1 and later versions, it is recommended not to use the tidb-cluster chart to deploy and manage the TiDB cluster. For details, refer to [TiDB Operator v1.1 Notes](notes-tidb-operator-v1.1.md).

## Parameters

| Parameter | Description | Default value |
| :----- | :---- | :----- |
| `rbac.create` | Whether to enable RBAC for Kubernetes | `true` |
| `clusterName` | TiDB cluster name. This variable is not set by default, and `tidb-cluster` directly uses `ReleaseName` during installation instead of the TiDB cluster name. | `nil` |
| `extraLabels` | Add extra labels to the `TidbCluster` object (CRD). Refer to [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) | `{}` |
| `schedulerName` | Scheduler used by TiDB cluster | `tidb-scheduler` |
| `timezone` | The default time zone of the TiDB cluster | `UTC` |
| `pvReclaimPolicy` | Reclaim policy for PVs (Persistent Volumes) used by the TiDB cluster | `Retain` |
| `services[0].name` | Name of the service exposed by the TiDB cluster | `nil` |
| `services[0].type` | Type of the service exposed by the TiDB cluster (selected from `ClusterIP`, `NodePort`, `LoadBalancer`) | `nil` |
| `discovery.image` | Mirror image of the PD service discovery component in the TiDB cluster, which is used to provide service discovery for each PD instance to coordinate the startup sequence when the PD cluster is first started | `pingcap/tidb-operator:v1.0.0-beta.3` |
| `discovery.imagePullPolicy` | Pull strategy for the PD service discovery component image | `IfNotPresent` |
| `discovery.resources.limits.cpu` | CPU resource limit for PD service discovery component | `250m` |
| `discovery.resources.limits.memory` | Memory resource limit for PD service discovery component | `150Mi` |
| `discovery.resources.requests.cpu` | CPU resource request for PD service discovery component | `80m` |
| `discovery.resources.requests.memory` | Memory resource request for PD service discovery component | `50Mi` |
| `enableConfigMapRollout` | Whether to enable automatic rolling update for the TiDB cluster. If enabled, the TiDB cluster automatically updates the corresponding components when the ConfigMap of the TiDB cluster changes. This configuration is only supported in TiDB Operator v1.0 and later versions | `false` |
| `pd.config` | Configuration of PD in configuration file format. To view the default PD configuration file, refer to [`pd/conf/config.toml`](https://github.com/pingcap/pd/blob/master/conf/config.toml) and select the tag of the corresponding PD version. To view the descriptions of parameters, refer to [PD configuration description](https://pingcap.com/docs/stable/reference/configuration/pd-server/configuration-file) and select the corresponding document version. Here you only need to modify the configuration according to the format in the configuration file. | If the TiDB Operator version <= v1.0.0-beta.3, the default value is <br/>`nil`<br/> If the TiDB Operator version > v1.0.0-beta.3, the default value is <br/>`[log]`<br/>`level = "info"`<br/>`[replication]`<br/>`location-labels = ["region", "zone", "rack", "host"]`<br/> For example, <br/>&nbsp;&nbsp;`config:` \|<br/>&nbsp;&nbsp;&nbsp;&nbsp;`[log]`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`level = "info"`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`[replication]`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`location-labels = ["region", "zone", "rack", "host"]` |
| `pd.replicas` | Number of Pods in PD | `3` |
| `pd.image` | PD image | `pingcap/pd:v3.0.0-rc.1` |
| `pd.imagePullPolicy` | Pull strategy for PD mirror | `IfNotPresent` |
| `pd.logLevel` | If TiDB Operator version> v1.0.0-beta.3, configure in `pd.config`: <br/>`[log]`<br/>`level = "info"` | `info` |
| `pd.storageClassName` | `storageClass` used by PD. `storageClassName` refers to a type of storage provided by the Kubernetes cluster. Different classes may be mapped to service quality levels, backup policies, or any policies determined by the cluster administrator. For details, refer to [storage-classes](https://kubernetes.io/docs/concepts/storage/storage-classes) | `local-storage` |
| `pd.maxStoreDownTime` | `pd.maxStoreDownTime` refers to the time that a store node is disconnected before the node is marked as `down`. When the status becomes `down`, the store node starts to migrate its data to other store nodes.<br/>If TiDB Operator version > v1.0.0-beta.3, configure in `pd.config`:<br/>`[schedule]`<br/>`max-store-down-time = "30m"` | `30m` |
| `pd.maxReplicas` | `pd.maxReplicas` is the number of replicas in the TiDB cluster.<br/>If TiDB Operator version > v1.0.0-beta.3, configure in `pd.config`:<br/>`[replication]`<br/>`max-replicas = 3` | `3` |
| `pd.resources.limits.cpu` | CPU resource limit for each PD Pod | `nil` |
| `pd.resources.limits.memory` | Memory resource limit for each PD Pod | `nil` |
| `pd.resources.limits.storage` | Storage capacity limit for each PD Pod | `nil` |
| `pd.resources.requests.cpu` | CPU resource request for each PD Pod | `nil` |
| `pd.resources.requests.memory` | Memory resource request for each PD Pod | `nil` |
| `pd.resources.requests.storage` | Storage capacity request for each PD Pod | `1Gi` |
| `pd.affinity` | `pd.affinity` defines PD scheduling rules and preferences. For details, refer to [affinity-and-anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) | `{}` |
| `pd.nodeSelector` | `pd.nodeSelector` makes sure that PD Pods are dispatched only to nodes that have this key-value pair as a label. For details, refer to [nodeselector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) | `{}` |
| `pd.tolerations` | `pd.tolerations` applies to PD Pods, allowing PD Pods to be dispatched to nodes with specified taints. For details, refer to [taint-and-toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration) | `{}` |
| `pd.annotations` | Add specific `annotations` to PD Pods | `{}` |
| `tikv.config` | Configuration of TiKV in configuration file format. To view the default TiKV configuration file, refer to [`tikv/etc/config-template.toml`](https://github.com/tikv/tikv/blob/master/etc/config-template.toml) and select the tag of the corresponding TiKV version. To view the descriptions of parameters, refer to [TiKV configuration description](https://pingcap.com/docs/stable/reference/configuration/tikv-server/configuration-file/) and select the corresponding document version. Here you only need to modify the configuration according to the format in the configuration file.<br/><br/>The following two configuration items need to be configured explicitly:<br/><br/>`[storage.block-cache]`<br/>&nbsp;&nbsp;`shared = true`<br/>&nbsp;&nbsp;`capacity = "1GB"`<br/>Recommended: set `capacity` to 50% of `tikv.resources.limits.memory`<br/><br/>`[readpool.coprocessor]`<br/>&nbsp;&nbsp;`high-concurrency = 8`<br/>&nbsp;&nbsp;`normal-concurrency = 8`<br/>&nbsp;&nbsp;`low-concurrency = 8`<br/>Recommended: set to 80% of `tikv.resources.limits.cpu` | If the TiDB Operator version <= v1.0.0-beta.3, the default value is<br/>`nil`<br/>If the TiDB Operator version > v1.0.0-beta.3, the default value is<br/>`log-level = "info"`<br/>For example:<br/>&nbsp;&nbsp;`config:` \|<br/>&nbsp;&nbsp;&nbsp;&nbsp;`log-level = "info"` |
| `tikv.replicas` | Number of Pods in TiKV | `3` |
| `tikv.image` | Image of TiKV | `pingcap/tikv:v3.0.0-rc.1` |
| `tikv.imagePullPolicy` | Pull strategy for TiKV image| `IfNotPresent` |
| `tikv.logLevel` | TiKV log level.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`log-level = "info"` | `info` |
| `tikv.storageClassName` | `storageClass` used by TiKV. `storageClassName` refers to a type of storage provided by the Kubernetes cluster. Different classes may be mapped to service quality levels, backup policies, or any policies determined by the cluster administrator. For details, refer to [storage-classes](https://kubernetes.io/docs/concepts/storage/storage-classes) | `local-storage` |
| `tikv.syncLog` | `syncLog` indicates whether to enable the raft log replication feature. If enabled, the feature ensure that data is not lost during poweroff.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[raftstore]`<br/>`sync-log = true`| `true` |
| `tikv.grpcConcurrency` | Configure the size of the gRPC server thread pool.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[server]`<br/>`grpc-concurrency = 4` | `4` |
| `tikv.resources.limits.cpu` | CPU resource limit for each TiKV Pod | `nil` |
| `tikv.resources.limits.memory` | Memory resource limit for each TiKV Pod | `nil` |
| `tikv.resources.limits.storage` | Storage capacity limit for each TiKV Pod | `nil` |
| `tikv.resources.requests.cpu` | CPU resource request for each TiKV Pod | `nil` |
| `tikv.resources.requests.memory` | Memory resource request for each TiKV Pod | `nil` |
| `tikv.resources.requests.storage` | Storage capacity request for each TiKV Pod | `10Gi` |
| `tikv.affinity` | `tikv.affinity` defines TiKV's scheduling rules and preferences. For details, refer to [affinity-and-anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) | `{}` |
| `tikv.nodeSelector` | `tikv.nodeSelector` makes sure that TiKV Pods are dispatched only to nodes that have this key-value pair as a label. For details, refer to [nodeselector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) | `{}` |
| `tikv.tolerations` | `tikv.tolerations` applies to TiKV Pods, allowing TiKV Pods to be dispatched to nodes with specified taints. For details, refer to [taint-and-toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration) | `{}` |
| `tikv.annotations` | Add specific `annotations` for TiKV Pods | `{}` |
| `tikv.defaultcfBlockCacheSize` | Specifies the block cache size. The block cache is used to cache uncompressed blocks. A large block cache setting can speed up reading. Generally  it is recommended to set the block cache size to 30%-50% of `tikv.resources.limits.memory`<br/>If TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[rocksdb.defaultcf]`<br/>`block-cache-size = "1GB"`<br/>Since TiKV v3.0.0, it is no longer necessary to configure `[rocksdb.defaultcf].block-cache-size` and `[rocksdb.writecf].block-cache-size`. Instead, you can configure `[storage.block-cache].capacity`. | `1GB` |
| `tikv.writecfBlockCacheSize` | Specifies the block cache size of `writecf`. It is generally recommended to be 10%-30% of `tikv.resources.limits.memory`<br/> If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[rocksdb.writecf]`<br/>`block-cache-size = "256MB"`<br/>Since TiKV v3.0.0, it is no longer necessary to configure `[rocksdb.defaultcf].block-cache-size` and `[rocksdb.writecf].block-cache-size`. Instead, you can configure `[storage.block-cache].capacity` | `256MB` |
| `tikv.readpoolStorageConcurrency` | Size of TiKV storage thread pool for high/medium/low priority operations<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[readpool.storage]`<br/>`high-concurrency = 4`<br/>`normal-concurrency = 4`<br/>`low-concurrency = 4` | `4` |
| `tikv.readpoolCoprocessorConcurrency` | Usually, if `tikv.resources.limits.cpu` > 8, set `tikv.readpoolCoprocessorConcurrency` to `tikv.resources.limits.cpu` * 0.8<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[readpool.coprocessor]`<br/>`high-concurrency = 8`<br/>`normal-concurrency = 8`<br/>`low-concurrency = 8` | `8` |
| `tikv.storageSchedulerWorkerPoolSize` | Size of the working pool of the TiKV scheduler, which should be increased in the case of rewriting, and should be smaller than the total CPU cores.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tikv.config`:<br/>`[storage]`<br/>`scheduler-worker-pool-size = 4` | `4` |
| `tidb.config` | Configuration of TiDB in configuration file format. To view the default TiDB configuration file, refer to [configuration file](https://github.com/pingcap/tidb/blob/master/config/config.toml.example) and select the tag of the corresponding TiDB version. To view the descriptions of parameters, refer to [TiDB configuration description](https://pingcap.com/docs/stable/reference/configuration/tidb-server/configuration-file) and select the corresponding document version. Here you only need to modify the configuration **according to the format in the configuration file**.<br/><br/>The following configuration items need to be configured explicitly:<br/><br/>`[performance]`<br/>&nbsp;&nbsp;`max-procs = 0`<br/>Recommended: set `max-procs` to the number of cores that correspondes to `tidb.resources.limits.cpu` | If the TiDB Operator version <= v1.0.0-beta.3, the default value is:<br/>`nil`<br/>If the TiDB Operator version > v1.0.0-beta.3, the default value is:<br/>`[log]`<br/>`level = "info"`<br/>For example:<br/>&nbsp;&nbsp;`config:` \|<br/>&nbsp;&nbsp;&nbsp;&nbsp;`[log]`<br/>&nbsp;&nbsp;&nbsp;&nbsp;`level = "info"` |
| `tidb.replicas` | Number of Pods in TiDB | `2` |
| `tidb.image` | Image of TiDB | `pingcap/tidb:v3.0.0-rc.1` |
| `tidb.imagePullPolicy` | Pull strategy for TiDB image | `IfNotPresent` |
| `tidb.logLevel` | TiDB log level.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`[log]`<br/>`level = "info"` | `info` |
| `tidb.resources.limits.cpu` | CPU resource limits for each TiDB Pod | `nil` |
| `tidb.resources.limits.memory` | Memory resource limit for each TiDB Pod | `nil` |
| `tidb.resources.requests.cpu` | Storage capacity limit for each TiDB Pod | `nil` |
| `tidb.resources.requests.memory` | Memory resource request for each TiDB Pod | `nil` |
| `tidb.passwordSecretName`| The name of the Secret which stores the TiDB username and password. This Secret can be created with the following command: `kubectl create secret generic tidb secret--from literal=root=${password}--namespace=${namespace}`. If the parameter is not set, the TiDB root password is empty. | `nil` |
| `tidb.initSql`| The initialization script that will be executed after the TiDB cluster is successfully started. | `nil` |
| `tidb.affinity` | `tidb.affinity` defines the scheduling rules and preferences of TiDB. For details, refer to [affinity-and-anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity). | `{}` |
| `tidb.nodeSelector` | `tidb.nodeSelector` makes sure that TiDB Pods are only dispatched to nodes that have this key-value pair as a label. For details, refer to[nodeselector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector). | `{}` |
| `tidb.tolerations` | `tidb.tolerations` applies to TiKV Pods, allowing TiKV Pods to be dispatched to nodes with specified taints. For details, refer to [taint-and-toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration). | `{}` |
| `tidb.annotations` | Add specific `annotations` for TiDB Pods | `{}` |
| `tidb.maxFailoverCount` | The maximum number of failover in TiDB. If it is set to 3, then TiDB supports at most 3 instances failover at the same time. | `3` |
| `tidb.service.type` | Type of the service exposed by TiDB | `NodePort` |
| `tidb.service.externalTrafficPolicy` | Indicates whether this service wants to route external traffic to a local node or a cluster-wide endpoint. Two options are available: `Cluster` (default) and `Local`. `Cluster` hides the client's source IP. In such case, the traffic might need to be redirected to another node, but has a good overall load distribution. `Local` retains the client's source IP and avoids the redirection of traffic of the LoadBalancer and NodePort service, but a potential risk of unbalanced traffic propagation exists. For details, refer to [External Load Balancer](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip). | `nil` |
| `tidb.service.loadBalancerIP` | Specifies TiDB load balancing IP. Some cloud service providers allow you to specify the loadBalancer IP. In these cases, a load balancer is created using a user-specified loadBalancerIP. If the loadBalancerIP field is not specified, the loadBalancer is set with a temporary IP address. If loadBalancerIP is specified but the cloud provider does not support this feature, the loadBalancerIP field you set is ignored. | `nil` |
| `tidb.service.mysqlNodePort` | MySQL NodePort port exposed by the TiDB service | / |
| `tidb.service.exposeStatus` | Whether the TiDB service exposes the status port | `true` |
| `tidb.service.statusNodePort` | Specifies the `NodePort` where the status port of the TiDB service is exposed | / |
| `tidb.separateSlowLog` | Whether to run a standalone container in sidecar mode to output TiDB's SlowLog | If the TiDB Operator version <= v1.0.0-beta.3, the default value is `false`.<br/>If the TiDB Operator version > v1.0.0-beta.3, the default value is `true`. |
| `tidb.slowLogTailer.image` | `slowLogTailer` image of TiDB. `slowLogTailer` is a sidecar type container used to output TiDB's SlowLog. This configuration only takes effect when `tidb.separateSlowLog` = `true`. | `busybox:1.26.2` |
| `tidb.slowLogTailer.resources.limits.cpu` | CPU resource limit for slowLogTailer of each TiDB Pod | `100m` |
| `tidb.slowLogTailer.resources.limits.memory` | Memory resource limit for slowLogTailer of each TiDB Pod | `50Mi` |
| `tidb.slowLogTailer.resources.requests.cpu` | CPU resource request for slowLogTailer of each TiDB Pod | `20m` |
| `tidb.slowLogTailer.resources.requests.memory` | Memory resource request for slowLogTailer of each TiDB Pod | `5Mi` |
| `tidb.plugin.enable` | Whether to enable the TiDB plugin feature | `false` |
| `tidb.plugin.directory` | Specifies the directory of the TiDB plugin | `/plugins` |
| `tidb.plugin.list` | Specifies the list of plugins loaded by TiDB. Rules of naming Plugin ID: [plugin name]-[version]. For example: 'conn_limit-1' | `[]` |
| `tidb.preparedPlanCacheEnabled` | Whether to enable the prepared plan cache of TiDB.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`[prepared-plan-cache]`<br/>`enabled = false` | `false` |
| `tidb.preparedPlanCacheCapacity` | Number of the prepared plan cache of TiDB.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`[prepared-plan-cache]`<br/>`capacity = 100` | `100` |
| `tidb.txnLocalLatchesEnabled` | Whether to enable the transaction memory lock. It is recommended to enable the transaction memory lock when there are many local transaction conflicts.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`[txn-local-latches]`<br/>`enabled = false` | `false` |
| `tidb.txnLocalLatchesCapacity` | Capacity of the transaction memory lock. The number of slots corresponding to the hash is automatically adjusted up to an exponential multiple of 2. Every slot occupies 32 bytes of memory. When the range of data to be written is wide (such as importing data), setting a small capacity causes performance degradation.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`[txn-local-latches]`<br/>`capacity = 10240000` | `10240000` |
| `tidb.tokenLimit` | Limit for concurrent sessions executed by TiDB<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`token-limit = 1000` | `1000` |
| `tidb.memQuotaQuery` | Memory limit for TiDB query. 32GB by default.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`mem-quota-query = 34359738368` | `34359738368` |
| `tidb.checkMb4ValueInUtf8` | Determines whether to check `mb4` characters when the current character set is `uff8`.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`check-mb4-value-in-utf8 = true` | `true` |
| `tidb.treatOldVersionUtf8AsUtf8mb4` | Used to upgrade compatibility. If this parameter is set to `true`, the `utf8` character set in the old version of tables/columns is treated as the `utf8mb4` character set.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`treat-old-version-utf8-as-utf8mb4 = true` | `true` |
| `tidb.lease` | Term of TiDB Schema lease. It is very dangerous to change this parameter, so do not change it unless you know the possible consequences.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`lease = "45s"` | `45s` |
| `tidb.maxProcs` | The maximum number of CPU cores that are available. 0 represents the total number of CPUs on the machine or Pod.<br/>If the TiDB Operator version > v1.0.0-beta.3, configure in `tidb.config`:<br/>`[performance]`<br/>`max-procs = 0` | `0` |
