---
title: TidbMonitor 分片功能
summary: 如何使用 TidbMonitor 分片功能
---

# 开启 TidbMonitor 分片功能

本文档介绍如何使用 TidbMonitor 分片功能。

## 功能介绍

TidbMonitor 负责单个或者多个 TiDB 集群的监控数据采集。当监控数据量很大的时候，单点计算能力会达到瓶颈。此时，你可以采用 Prometheus [Modulus](https://prometheus.io/docs/prometheus/latest/configuration/configuration/) 分片功能，对 `__address__` 做 `hashmod`，将多个目标（关键字为 `Targets`）的监控打散到多个 TidbMonitor Pod 上。

TidbMonitor 分片功能需要采用数据聚合方案，推荐使用 [Thanos](https://thanos.io/tip/thanos/design.md/) 方案。

## 如何开启分片功能

开启分片功能，需要指定 `shards` 字段，示例如下:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: monitor
spec:
  replicas: 1
  shards: 2
  clusters:
    - name: basic
  prometheus:
    baseImage: prom/prometheus
    version: v2.27.1
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.2.1
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

> **注意：**
>
> - TidbMonitor 对应的 Pod 实例数量取决于 `replicas` 和 `shards` 的乘积。例如，当 `replicas` 为 1 个副本，`shards` 为 2 个分片时，TiDB Operator 将产生 2 个 TidbMonitor Pod 实例。
> - `shards` 变更后，`Targets` 会重新分配，但是原本在节点上的监控数据不会重新分配。

可以参考 [分片示例](https://github.com/pingcap/tidb-operator/tree/master/examples/monitor-shards)。
