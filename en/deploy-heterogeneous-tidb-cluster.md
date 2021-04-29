---
title: Deploy a Heterogeneous TiDB Cluster
summary: Learn how to deploy a heterogeneous cluster for an existing TiDB cluster.
---

# Deploy a Heterogeneous TiDB Cluster

This document describes how to deploy a heterogeneous cluster for an existing TiDB cluster.

## Prerequisites

* You already have a TiDB cluster. If not, refer to [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md).

## Deploy a heterogeneous cluster

A heterogeneous cluster creates differentiated instances for an existing TiDB cluster. You can create a heterogeneous TiKV cluster with different configurations and labels to facilitate hotspot scheduling, or create a heterogeneous TiDB cluster for OLTP and OLAP workloads respectively.

### Create a heterogeneous cluster

Save the following configuration as the `cluster.yaml` file. Replace `${heterogeneous_cluster_name}` with the desired name of your heterogeneous cluster, and replace `${origin_cluster_name}` with the name of the existing cluster.

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: ${heterogeneous_cluster_name}
spec:
  configUpdateStrategy: RollingUpdate
  version: v5.0.1
  timezone: UTC
  pvReclaimPolicy: Delete
  discovery: {}
  cluster:
    name: ${origin_cluster_name}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    # if storageClassName is not set, the default Storage Class of the Kubernetes cluster will be used
    # storageClassName: local-storage
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 1
    replicas: 1
    storageClaims:
      - resources:
          requests:
            storage: 1Gi
        storageClassName: standard
```

Execute the following command to create the heterogeneous cluster:

{{< copyable "shell-regular" >}}

```shell
kubectl create -f cluster.yaml -n ${namespace}
```

The configuration of a heterogeneous cluster is mostly the same as a normal TiDB cluster, except that it uses the `spec.cluster.name` field to join the target cluster.

### Deploy the cluster monitoring component

Save the following configuration as the `tidbmonitor.yaml` file. Replace `${heterogeneous_cluster_name}` with the desired name of your heterogeneous cluster, and replace `${origin_cluster_name}` with the name of the existing cluster.

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: heterogeneous
spec:
  clusters:
    - name: ${origin_cluster_name}
    - name: ${heterogeneous_cluster_name}
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
  grafana:
    baseImage: grafana/grafana
    version: 6.1.6
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.0.1
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

Execute the following command to create the heterogeneous cluster:

{{< copyable "shell-regular" >}}

```shell
kubectl create -f tidbmonitor.yaml -n ${namespace}
```

## Deploy a TLS-enabled heterogeneous cluster

To enable TLS for a heterogeneous cluster, you need to explicitly declare the TLS configuration, issue the certificates using the same certification authority (CA) as the target cluster and create new secrets with the certificates.

If you want to issue the certificate using `cert-manager`, choose the same `Issuer` as that of the target cluster to create your `Certificate`.

For detailed procedures to create certificates for the heterogeneous cluster, refer to the following two documents:

- [Enable TLS between TiDB Components](enable-tls-between-components.md)
- [Enable TLS for the MySQL Client](enable-tls-for-mysql-client.md)

### Create a TLS-enabled heterogeneous cluster

Save the following configuration as the `cluster.yaml` file. Replace `${heterogeneous_cluster_name}` with the desired name of your heterogeneous cluster, and replace `${origin_cluster_name}` with the name of the existing cluster.

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: ${heterogeneous_cluster_name}
spec:
  tlsCluster:
    enabled: true
  configUpdateStrategy: RollingUpdate
  version: v5.0.1
  timezone: UTC
  pvReclaimPolicy: Delete
  discovery: {}
  cluster:
    name: ${origin_cluster_name}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    # if storageClassName is not set, the default Storage Class of the Kubernetes cluster will be used
    # storageClassName: local-storage
    requests:
      storage: "1Gi"
    config:
      storage:
        # In basic examples, we set this to avoid using too much storage.
        reserve-space: "0MB"
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
    tlsClient:
      enabled: true
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 1
    replicas: 1
    storageClaims:
      - resources:
          requests:
            storage: 1Gi
        storageClassName: standard
```

- `spec.tlsCluster.enabled`: Determines whether to enable TLS between the components.
- `spec.tidb.tlsClient.enabled`: Determines whether to enable TLS for MySQL client.

Execute the following command to create the TLS-enabled heterogeneous cluster:

{{< copyable "shell-regular" >}}

```shell
kubectl create -f cluster.yaml -n ${namespace}
```

For the detailed configuration of a TLS-enabled heterogeneous cluster, see ['heterogeneous-tls'](https://github.com/pingcap/tidb-operator/tree/master/examples/heterogeneous-tls) example.
