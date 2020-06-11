---
title: 通过 TidbCluster 配置 TiDB 集群
summary: 介绍如何使用 TidbCluster 配置 TiDB 集群。
category: how-to
---

# 通过 TidbCluster 配置 TiDB 集群

TidbCluster 文件支持在其上面直接配置 TiDB/TiKV/PD/TiFlash/TiCDC 的配置选项，本篇文章将介绍如何在 TidbCluster 上配置参数。目前 Operator 1.1 版本支持了 TiDB 集群 v3.1 版本参数。针对各组件配置参数，请参考 PingCAP 官方文档。

## 配置 TiDB 配置参数

你可以通过 TidbCluster Custom Resource 的 `spec.tidb.config` 来配置 TiDB 配置参数，以下是一个例子。获取所有可以配置的 TiDB 配置参数，请参考 [TiDB 配置文档](https://pingcap.com/docs-cn/v3.1/reference/configuration/tidb-server/configuration-file/)。

> **注意：**
>
> 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 TiDB 组件无法正常启动。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
....
  tidb:
    image: pingcap.com/tidb:v3.1.0
    imagePullPolicy: IfNotPresent
    replicas: 1
    service:
      type: ClusterIP
    config:
      split-table: true
      oom-action: "log"
    requests:
      cpu: 1
```

## 配置 TiKV 配置参数

你可以通过 TidbCluster Custom Resource 的 `spec.tikv.config` 来配置 TiKV 配置参数，以下是一个例子。获取所有可以配置的 TiKV 配置参数，请参考 [TiKV 配置文档](https://pingcap.com/docs-cn/v3.1/reference/configuration/tikv-server/configuration-file/)

> **注意：**
>
> 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 TiKV 组件无法正常启动。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
....
  tikv:
    image: pingcap.com/tikv:v3.1.0
    config:
      log-level: "info"
      slow-log-threshold: "1s"
    replicas: 1
    requests:
      cpu: 2
```

## 配置 PD 配置参数

你可以通过 TidbCluster Custom Resource 的 `spec.pd.config` 来配置 PD 配置参数，以下是一个例子。获取所有可以配置的 PD 配置参数，请参考 [PD 配置文档](https://pingcap.com/docs-cn/v3.1/reference/configuration/pd-server/configuration-file/)

> **注意：**
>
> 为了兼容 `helm` 部署，如果你是通过 CR 文件部署 TiDB 集群，即使你不设置 Config 配置，也需要保证 `Config: {}` 的设置，从而避免 PD 组件无法正常启动。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
.....
  pd:
    image: pingcap.com/pd:v3.1.0
    config:
      lease: 3
      enable-prevote: true
```

## 配置 TiFlash 配置参数

你可以通过 TidbCluster Custom Resource 的 `spec.tiflash.config` 来配置 TiFlash 配置参数，以下是一个例子。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  ...
  tiflash:
    config:
      config:
        logger:
          count: 5
          level: information
```

## 配置 TiCDC 启动参数

你可以通过 TidbCluster Custom Resource 的 `spec.ticdc.config` 来配置 TiCDC 启动参数，以下是一个例子。获取所有可以配置的 TiCDC 启动参数，请参考 [TiCDC 启动参数文档](https://pingcap.com/docs-cn/stable/ticdc/deploy-ticdc/#手动在原有-tidb-集群上新增-ticdc-组件)。

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
