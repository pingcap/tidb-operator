---
title: 访问 Kubernetes 上的 TiDB 集群
summary: 介绍如何访问 Kubernetes 上的 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/access-tidb/']
---

# 访问 Kubernetes 上的 TiDB 集群

在 Kubernetes 集群内访问 TiDB 时，使用 TiDB service 域名 `${cluster_name}-tidb.${namespace}` 即可。

若需要在集群外访问，则需将 TiDB 服务端口暴露出去。在 `TidbCluster` CR 中，通过 `spec.tidb.service` 字段进行配置：

{{< copyable "" >}}

```yaml
spec:
  ...
  tidb:
    service:
      type: NodePort
      # externalTrafficPolicy: Cluster
      # annotations:
      #   cloud.google.com/load-balancer-type: Internal
```

## NodePort

在没有 LoadBalancer 时，可选择通过 NodePort 暴露。NodePort 有两种模式：

- `externalTrafficPolicy=Cluster`：集群所有的机器都会给 TiDB 分配 TCP 端口，此为默认值

    使用 `Cluster` 模式时，可以通过任意一台机器的 IP 加同一个端口访问 TiDB 服务，如果该机器上没有 TiDB Pod，则相应请求会转发到有 TiDB Pod 的机器上。

    > **注意：**
    >
    > 该模式下 TiDB 服务获取到的请求源 IP 是主机 IP，并不是真正的客户端源 IP，所以基于客户端源 IP 的访问权限控制在该模式下不可用。

- `externalTrafficPolicy=Local`：只有运行 TiDB 的机器会分配 TCP 端口，用于访问本地的 TiDB 实例

    使用 `Local` 模式时，建议打开 tidb-scheduler 的 `StableScheduling` 特性。tidb-scheduler 会尽可能在升级过程中将新 TiDB 实例调度到原机器，这样集群外的客户端便不需要在 TiDB 重启后更新配置。

### 查看 NodePort 模式下对外暴露的 IP/PORT

查看 Service 分配的 Node Port，可通过获取 TiDB 的 Service 对象来获知：

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} get svc ${cluster_name}-tidb -ojsonpath="{.spec.ports[?(@.name=='mysql-client')].nodePort}{'\n'}"
```

查看可通过哪些节点的 IP 访问 TiDB 服务，有两种情况：

- `externalTrafficPolicy` 为 `Cluster` 时，所有节点 IP 均可
- `externalTrafficPolicy` 为 `Local` 时，可通过以下命令获取指定集群的 TiDB 实例所在的节点

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pods -l "app.kubernetes.io/component=tidb,app.kubernetes.io/instance=${cluster_name}" -ojsonpath="{range .items[*]}{.spec.nodeName}{'\n'}{end}"
    ```

## LoadBalancer

若运行在有 LoadBalancer 的环境，比如 GCP/AWS 平台，建议使用云平台的 LoadBalancer 特性。

访问 [Kubernetes Service 文档](https://kubernetes.io/docs/concepts/services-networking/service/)，了解更多 Service 特性以及云平台 Load Balancer 支持。

## 平滑升级 TiDB 集群

滚动更新 TiDB 集群的过程中，在停止 TiDB Pod 之前，Kubernetes 会向 TiDB server 进程发送一个 [`TERM`](https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods) 信号。在收到 `TERM` 信号后，TiDB server 会尝试等待所有的连接关闭，不过 15 秒后会强制关闭所有连接并退出进程。

从 v1.1.2 版本开始，TiDB Operator 已经支持平滑升级 TiDB 集群。通过配置下面两个属性来实现平滑升级 TiDB 集群：

- `spec.tidb.terminationGracePeriodSeconds`：滚动更新的时候，删除旧的 TiDB Pod 最多容忍的时间，即过了这个时间，TiDB Pod 会被强制删除；
- `spec.tidb.lifecycle`：设置 TiDB Pod 的 `preStop` Hook，在 TiDB server 停止之前执行的操作。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  version: v4.0.4
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
