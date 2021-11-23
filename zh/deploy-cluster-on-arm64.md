---
title: 在 ARM64 机器上部署 TiDB 集群
summary: 本文档介绍如何在 ARM64 机器上部署 TiDB 集群
---

# 在 ARM64 机器上部署 TiDB 集群

本文档介绍如何在 ARM64 机器上部署 TiDB 集群。

## 前置条件

* 在 ARM64 机器上已经部署了 Kubernetes。如果尚未部署，请参阅[部署 Kubernetes 集群](deploy-tidb-operator.md#部署-kubernetes-集群)。

## 部署 TiDB Operator

在 ARM64 机器上部署 TiDB Operator 的步骤与[在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md) 的步骤相同。唯一区别是，在[自定义部署 TiDB Operator](deploy-tidb-operator.md#自定义部署-tidb-operator) 这一步，当获取到 `tidb-operator` chart 中的 `value.yaml` 文件后，你需要修改文件中的 `operatorImage` 与 `tidbBackupManagerImage` 字段为 ARM64 版本镜像。例如：

```yaml
# ...
operatorImage: pingcap/tidb-operator-arm64:v1.2.4
# ...
tidbBackupManagerImage: pingcap/tidb-backup-manager-arm64:v1.2.4
# ...
```

## 部署 TiDB 集群

在 ARM64 机器上部署 TiDB 集群的步骤与[在标准 Kubernetes 上部署 TiDB 集群](deploy-on-general-kubernetes.md) 的步骤相同。唯一区别是，你需要将 TidbCluster 定义文件中相关组件的镜像设置为 ARM64 版本。例如：

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: ${cluster_name}
  namespace: ${cluster_namespace}
spec:
  version: "v5.2.1"
  # ...
  helper:
    image: busybox:1.33.0
  # ...
  pd:
    baseImage: pingcap/pd-arm64
    # ...
  tidb:
    baseImage: pingcap/tidb-arm64
    # ...
  tikv:
    baseImage: pingcap/tikv-arm64
    # ...
  pump:
    baseImage: pingcap/tidb-binlog-arm64
    # ...
  ticdc:
    baseImage: pingcap/ticdc-arm64
    # ...
  tiflash:
    baseImage: pingcap/tiflash-arm64
    # ...
```

## 初始化 TiDB 集群

在 ARM64 机器上初始化 TiDB 集群的步骤与[在 Kubernetes 上的初始化 TiDB 集群的步骤](initialize-a-cluster.md) 相同。唯一区别是，你需要将 TidbInitializer 定义文件中的 `spec.image` 字段设置为 ARM64 版本镜像。例如：

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbInitializer
metadata:
  name: ${initializer_name}
  namespace: ${cluster_namespace}
spec:
  image: kanshiori/mysqlclient-arm64
  # ...
```

## 部署 TiDB 集群监控

在 ARM64 机器上部署 TiDB 集群监控的步骤与 [TiDB 集群的监控与告警](monitor-a-tidb-cluster.md) 的步骤相同。唯一区别是，你需要将 TidbMonitor 定义文件中的 `spec.initializer.baseImage` 与 `spec.reloader.baseImage` 字段设置为 ARM64 版本镜像。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: ${monitor_name}
spec:
  # ...
  initializer:
    baseImage: pingcap/tidb-monitor-initializer-arm64
    version: v5.2.1
  reloader:
    baseImage: pingcap/tidb-monitor-reloader-arm64
    version: v1.0.1
  # ...
```