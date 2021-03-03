---
title: 聚合多个 TiDB 集群的监控数据
summary: 通过 Thanos 框架聚合多个 TiDB 集群的监控数据
---

# 聚合多个 TiDB 集群的监控数据

本文档介绍如何通过 Thanos 聚合多个 TiDB 集群的监控数据，解决多集群下监控数据的中心化问题。

## Thanos 介绍

Thanos 是 Prometheus 高可用的解决方案，用于简化 Prometheus 的可用性保证。详细内容请参考 [Thanos 官方文档](https://thanos.io/design.md/)。

Thanos 提供了跨 Prometheus 的统一查询方案 [Thanos Query](https://thanos.io/tip/components/query.md/) 组件，可以利用这个功能解决 TiDB 多集群监控数据聚合的问题。

## 配置 Thanos Query

首先，需要为每个 TidbMonitor 配置一个 Thanos Sidecar 容器。示例如下：

{{< copyable "shell-regular" >}}

```
kubectl -n ${namespace} apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/tidb-monitor.yaml"
```

然后需要部署 Thanos Query 组件：

{{< copyable "shell-regular" >}}

```
kubectl -n ${namespace} apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/thanos-query.yaml"
```

在 Thanos Query 中，一个 Prometheus 对应一个 Store，也就对应一个 TidbMonitor。部署完 Thanos Query，就可以通过 Thanos Query 的 API 提供监控数据的统一查询接口。

## 配置 Grafana

部署完 Thanos Query，Grafana 只需要将 DataSource 更改成 Thanos 源，就可以查询到多个 TidbMonitor 的监控数据。

## 增加或者减少 TidbMonitor

当需要从 Thanos Query 增加、更新或者下线 Monitor Store 时，需要更新 Thanos Query Store 的启动配置 `--store`，滚动更新 Thanos Query 组件。

```shell
   thanos query \
            --http-address     "0.0.0.0:9090" \
            --store            "<store-api>:<grpc-port>" \
            --store            "<store-api2>:<grpc-port>"      
```

## 配置 Thanos Sidecar 归档存储

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
