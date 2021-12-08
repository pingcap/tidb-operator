---
title: Deploy a TiDB Cluster on ARM64 Machines
summary: Learn how to deploy a TiDB cluster on ARM64 machines.
---

# Deploy a TiDB Cluster on ARM64 Machines

This document describes how to deploy a TiDB cluster on ARM64 machines.

## Prerequisites

Before starting the process, make sure that Kubernetes clusters are deployed on your ARM64 machines. If Kubernetes clusters are not deployed, refer to [Deploy the Kubernetes cluster](deploy-tidb-operator.md#deploy-the-kubernetes-cluster).

## Deploy TiDB operator

The process of deploying TiDB operator on ARM64 machines is the same as the process of [Deploy TiDB Operator in Kubernetes](deploy-tidb-operator.md). The only difference is the following configuration in the step [Customize TiDB operator deployment](deploy-tidb-operator.md#customize-tidb-operator-deployment): after getting the `values.yaml` file of the `tidb-operator` chart, you need to modify the `operatorImage` and `tidbBackupManagerImage` fields in that file to the ARM64 image versions. For example:

```yaml
# ...
operatorImage: pingcap/tidb-operator-arm64:v1.2.4
# ...
tidbBackupManagerImage: pingcap/tidb-backup-manager-arm64:v1.2.4
# ...
```

## Deploy a TiDB cluster

The process of deploying a TiDB cluster on ARM64 machines is the same as the process of [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md). The only difference is that, in the TidbCluster definition file, you need to set the images of the related components to the ARM64 versions. For example:

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

## Initialize a TiDB cluster

The process of initializing a TiDB cluster on ARM64 machines is the same as the process of [Initialize a TiDB Cluster in Kubernetes](initialize-a-cluster.md). The only difference is that you need to modify the `spec.image` field in the TidbInitializer definition file to the ARM64 image version. For example:

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

## Deploy monitoring for a TiDB cluster

The process of deploying monitoring for a TiDB cluster on ARM64 machines is the same as the process of [Deploy Monitoring and Alerts for a TiDB Cluster](monitor-a-tidb-cluster.md). The only difference is that you need to modify the `spec.initializer.baseImage` and `spec.reloader.baseImage` fields in the TidbMonitor definition file to the ARM64 image versions.

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