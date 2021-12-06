---
title: 聚合多个 TiDB 集群的监控数据
summary: 通过 Thanos 框架聚合多个 TiDB 集群的监控数据
---

# 聚合多个 TiDB 集群的监控数据

本文档介绍如何通过 Thanos 聚合多个 TiDB 集群的监控数据，解决多集群下监控数据的中心化问题。

## Thanos 介绍

Thanos 是 Prometheus 高可用的解决方案，用于简化 Prometheus 的可用性保证。详细内容请参考 [Thanos 官方文档](https://thanos.io/tip/thanos/design.md/)。

Thanos 提供了跨 Prometheus 的统一查询方案 [Thanos Query](https://thanos.io/tip/components/query.md/) 组件，可以利用这个功能解决 TiDB 多集群监控数据聚合的问题。

## 通过 Thanos Query 聚合监控数据

### 配置 Thanos Query

1. 为每个 TidbMonitor 配置一个 Thanos Sidecar 容器。

    示例如下：

    {{< copyable "shell-regular" >}}

    ```
    kubectl -n ${namespace} apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/tidb-monitor.yaml
    ```

2. 部署 Thanos Query 组件。

    1. 下载 Thanos Query 的部署文件 `thanos-query.yaml`：

        {{< copyable "shell-regular" >}}
        
        ```
        curl -sl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/thanos-query.yaml
        ```

    2. 手动修改 `thanos-query.yaml` 文件中的 `--store` 参数，将 `basic-prometheus:10901` 改为 `basic-prometheus.${namespace}:10901`。
    
        其中，`${namespace}` 表示 TidbMonitor 部署的命名空间。

    3. 执行 `kubectl apply` 命令部署：

        {{< copyable "shell-regular" >}}

        ```
        kubectl -n ${thanos_namespace} apply -f thanos-query.yaml
        ```

        其中，`${thanos_namespace}` 表示 Thanos Query 组件部署的命名空间。

在 Thanos Query 中，一个 Prometheus 对应一个 Store，也就对应一个 TidbMonitor。部署完 Thanos Query，就可以通过 Thanos Query 的 API 提供监控数据的统一查询接口。

### 访问 Thanos Query 面板

要访问 Thanos Query 面板，请执行以下命令，然后通过浏览器访问 <http://127.0.0.1:9090>

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${thanos_namespace} svc/thanos-query 9090
```

如果你想通过 NodePort 或 LoadBalancer 访问，请参考：

- [NodePort 方式](access-tidb.md#nodeport)
- [LoadBalancer 方式](access-tidb.md#loadbalancer)

### 配置 Grafana

部署 Thanos Query 之后，要查询多个 TidbMonitor 的监控数据，请进行以下操作：

1. 登陆 Grafana。
2. 在左侧导航栏中，选择 `Configuration` > `Data Sources`。
3. 添加或修改一个 Prometheus 类型的 DataSource。
4. 将 HTTP 下面的 URL 设置为 `http://thanos-query.${thanos_namespace}:9090`

### 增加或者减少 TidbMonitor

在 Thanos Query 中，一个 Prometheus 对应一个 Monitor Store，也就对应一个 TidbMonitor。当需要从 Thanos Query 增加、更新或者下线 Monitor Store 时，需要更新 Thanos Query 组件的命令参数 `--store`，滚动更新 Thanos Query 组件。

```yaml
spec:
 containers:
   - args:
       - query
       - --grpc-address=0.0.0.0:10901
       - --http-address=0.0.0.0:9090
       - --log.level=debug
       - --log.format=logfmt
       - --query.replica-label=prometheus_replica
       - --query.replica-label=rule_replica
       - --store=<TidbMonitorName1>-prometheus.<TidbMonitorNs1>:10901
       - --store=<TidbMonitorName2>-prometheus.<TidbMonitorNs2>:10901
```

### 配置 Thanos Sidecar 归档存储

> **注意：**
>
> 为确保配置成功，必须先创建 S3 bucket。如果你选择 AWS S3，请参考 [AWS S3 创建 bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html) 和 [AWS S3 endpoint 列表](https://docs.aws.amazon.com/general/latest/gr/s3.html)。

Thanos Sidecar 支持将监控数据同步到 S3 远端存储，配置如下:

TidbMonitor CR 配置如下：

```yaml
spec:
  thanos:
    baseImage: thanosio/thanos
    version: v0.17.2
    objectStorageConfig:
      key: objectstorage.yaml
      name: thanos-objectstorage
```

同时需要创建一个 Secret，示例如下:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: thanos-objectstorage
type: Opaque
stringData:
  objectstorage.yaml: |
    type: S3
    config:
      bucket: "xxxxxx"
      endpoint: "xxxx"
      region: ""
      access_key: "xxxx"
      insecure: true
      signature_version2: true
      secret_key: "xxxx"
      put_user_metadata: {}
      http_config:
        idle_conn_timeout: 90s
        response_header_timeout: 2m
      trace:
        enable: true
      part_size: 41943040
```

## RemoteWrite 模式

除了 Thanos Query 监控聚合模式，也可以通过 Prometheus RemoteWrite 推送监控数据到 Thanos。

在启动 TiDBMonitor 时可以指定 Prometheus RemoteWrite 配置，示例如下:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
  - name: basic
  prometheus:
    baseImage: prom/prometheus
    version: v2.27.1
    remoteWrite:
      - url: "http://thanos-receiver:19291/api/v1/receive"
  grafana:
    baseImage: grafana/grafana
    version: 7.5.11
  initializer:
    baseImage: registry.cn-beijing.aliyuncs.com/tidb/tidb-monitor-initializer
    version: v5.2.1
  reloader:
    baseImage: registry.cn-beijing.aliyuncs.com/tidb/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

Prometheus 将会把数据推送到 [Thanos Receiver](https://thanos.io/tip/components/receive.md/) 服务，详情可以参考 [Receiver 架构设计](https://thanos.io/v0.8/proposals/201812_thanos-remote-receive/)。

部署方案可以参考 [Example](https://github.com/pingcap/tidb-operator/tree/master/examples/monitor-prom-remotewrite)。
