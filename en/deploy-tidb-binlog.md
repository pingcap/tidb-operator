---
title: Deploy TiDB Binlog
summary: Learn how to deploy TiDB Binlog for a TiDB cluster in Kubernetes.
category: how-to
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-tidb-binlog/']
---

# Deploy TiDB Binlog

This document describes how to maintain [TiDB Binlog](https://pingcap.com/docs/stable/tidb-binlog/tidb-binlog-overview/) of a TiDB cluster in Kubernetes.

## Prerequisites

- [Deploy TiDB Operator](deploy-tidb-operator.md);
- [Install Helm](tidb-toolkit.md#use-helm) and configure it with the official PingCAP chart.

## Deploy TiDB Binlog of a TiDB cluster

TiDB Binlog is disabled in the TiDB cluster by default. To create a TiDB cluster with TiDB Binlog enabled, or enable TiDB Binlog in an existing TiDB cluster, take the following steps.

### Deploy Pump

1. Modify the TidbCluster CR file to add the Pump configuration.

    For example:

    ```yaml
    spec:
      ...
      pump:
        baseImage: pingcap/tidb-binlog
        version: v3.0.11
        replicas: 1
        storageClassName: local-storage
        requests:
          storage: 30Gi
        schedulerName: default-scheduler
        config:
          addr: 0.0.0.0:8250
          gc: 7
          heartbeat-interval: 2
    ```

    Edit `version`, `replicas`, `storageClassName`, and `requests.storage` according to your cluster.

2. Set affinity and anti-affinity for TiDB and Pump.
    
    If you enable TiDB Binlog in the production environment, it is recommended to set affinity and anti-affinity for TiDB and the Pump component; if you enable TiDB Binlog in a test environment on the internal network, you can skip this step.

    By default, the affinity of TiDB and Pump is set to `{}`. Currently, each TiDB instance does not have a corresponding Pump instance by default. When TiDB Binlog is enabled, if Pump and TiDB are separately deployed and network isolation occurs, and `ignore-error` is enabled in TiDB components, TiDB loses binlogs.

    In this situation, it is recommended to deploy a TiDB instance and a Pump instance on the same node using the affinity feature, and to split Pump instances on different nodes using the anti-affinity feature. For each node, only one Pump instance is required. The steps are as follows:

    * Configure `spec.tidb.affinity` as follows:

        ```yaml
        spec:
          tidb:
            affinity:
              podAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - weight: 100
                  podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: "app.kubernetes.io/component"
                        operator: In
                        values:
                        - "pump"
                      - key: "app.kubernetes.io/managed-by"
                        operator: In
                        values:
                        - "tidb-operator"
                      - key: "app.kubernetes.io/name"
                        operator: In
                        values:
                        - "tidb-cluster"
                      - key: "app.kubernetes.io/instance"
                        operator: In
                        values:
                        - ${cluster_name}
                    topologyKey: kubernetes.io/hostname
        ```

    * Configure `spec.pump.affinity` as follows:

        ```yaml
        spec:
          pump:
            affinity:
              podAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - weight: 100
                  podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: "app.kubernetes.io/component"
                        operator: In
                        values:
                        - "tidb"
                      - key: "app.kubernetes.io/managed-by"
                        operator: In
                        values:
                        - "tidb-operator"
                      - key: "app.kubernetes.io/name"
                        operator: In
                        values:
                        - "tidb-cluster"
                      - key: "app.kubernetes.io/instance"
                        operator: In
                        values:
                        - ${cluster_name}
                    topologyKey: kubernetes.io/hostname
              podAntiAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - weight: 100
                  podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: "app.kubernetes.io/component"
                        operator: In
                        values:
                        - "pump"
                      - key: "app.kubernetes.io/managed-by"
                        operator: In
                        values:
                        - "tidb-operator"
                      - key: "app.kubernetes.io/name"
                        operator: In
                        values:
                        - "tidb-cluster"
                      - key: "app.kubernetes.io/instance"
                        operator: In
                        values:
                        - ${cluster_name}
                    topologyKey: kubernetes.io/hostname
        ```

    > **Note:**
    >
    > If you update the affinity configuration of the TiDB components, it will cause rolling updates of the TiDB components in the cluster.

## Deploy drainer

To deploy multiple drainers using the `tidb-drainer` Helm chart for a TiDB cluster, take the following steps:

1. Make sure that the PingCAP Helm repository is up to date:

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm search tidb-drainer -l
    ```

2. Get the default `values.yaml` file to facilitate customization:

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tidb-drainer --version=${chart_version} > values.yaml
    ```

3. Modify the `values.yaml` file to specify the source TiDB cluster and the downstream database of the drainer. Here is an example:

    ```yaml
    clusterName: example-tidb
    clusterVersion: v3.0.0
    storageClassName: local-storage
    storage: 10Gi
    config: |
      detect-interval = 10
      [syncer]
      worker-count = 16
      txn-batch = 20
      disable-dispatch = false
      ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql"
      safe-mode = false
      db-type = "tidb"
      [syncer.to]
      host = "slave-tidb"
      user = "root"
      password = ""
      port = 4000
    ```

    The `clusterName` and `clusterVersion` must match the desired source TiDB cluster.

    For complete configuration details, refer to [TiDB Binlog Drainer Configurations in Kubernetes](configure-tidb-binlog-drainer.md).

4. Deploy the drainer:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-drainer --name=${cluster_name} --namespace=${namespace} --version=${chart_version} -f values.yaml
    ```

    > **Note:**
    >
    > This chart must be installed to the same namespace as the source TiDB cluster.

## Enable TLS

If you want to enable TLS for the TiDB cluster and TiDB Binlog, refer to [Enable TLS between Components](enable-tls-between-components.md).
