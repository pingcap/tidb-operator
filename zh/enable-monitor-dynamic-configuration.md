---
title: TidbMonitor 开启动态配置功能
summary: 动态更新 TidbMonitor 配置
---

# 开启动态配置功能

本文档介绍如何开启 TidbMonitor 动态配置功能。

## 功能介绍

TidbMonitor 支持多集群、分片等功能，当 Prometheus 的配置、Rule、Targets 变更时，如果不开启动态配置，这些变更只能在重启后才能生效。如果监控数据量很大，重启后恢复 Prometheus 快照数据耗时会比较长。

开启动态配置功能后，TidbMonitor 的配置更改即可动态更新。

## 如何开启动态配置功能

在 TidbMonitor 的 spec 中，你可以通过指定 `prometheusReloader` 开启动态配置功能。示例如下:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: monitor
spec:
  clusterScoped: true
  clusters:
    - name: ns1
      namespace: ns1
    - name: ns2
      namespace: ns2
  prometheusReloader:
    baseImage: quay.io/prometheus-operator/prometheus-config-reloader
    version: v0.49.0
  imagePullPolicy: IfNotPresent
```

`prometheusReloader` 配置变更后，TidbMonitor 会自动重启。重启后，所有针对 Prometheus 的配置变更都会动态更新。

可以参考 [monitor-dynamic-configmap 配置示例](https://github.com/pingcap/tidb-operator/tree/master/examples/monitor-dynamic-configmap)。

## 关闭动态配置功能

去除 `prometheusReloader` 字段并变更。
