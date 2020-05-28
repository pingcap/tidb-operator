---
title: Deploy TiDB on General Kubernetes
summary: Learn how to deploy a TiDB cluster on general Kubernetes.
category: how-to
---

# Deploy TiDB on General Kubernetes

This document describes how to deploy a TiDB cluster on general Kubernetes.

## Prerequisites

- Complete [deploying TiDB Operator](deploy-tidb-operator.md).

## Configure TiDB cluster

Refer to the [TidbCluster example](https://github.com/pingcap/tidb-operator/blob/master/examples/basic/tidb-cluster.yaml) and [API documentation](api-references.md) to complete TidbCluster Custom Resource (CR), and save it to the `${cluster_name}/tidb-cluster.yaml` file. Switch the TidbCluster example and API documentation to the currently used version of TiDB Operator.

Note that TidbCluster CR has multiple parameters for the image configuration:

- `spec.version`: the format is `imageTag`, such as `v3.1.0`

- `spec.<pd/tidb/tikv/pump>.baseImage`: the format is `imageName`, such as `pingcap/tidb`

- `spec.<pd/tidb/tikv/pump>.version`: the format is `imageTag`, such as `v3.1.0`

- `spec.<pd/tidb/tikv/pump>.image`: the format is `imageName:imageTag`, such as `pingcap/tidb:v3.1.0`

The priority for acquiring the image configuration is as follows:

`spec.<pd/tidb/tikv/pump>.baseImage` + `spec.<pd/tidb/tikv/pump>.version` > `spec.<pd/tidb/tikv/pump>.baseImage` + `spec.version` > `spec.<pd/tidb/tikv/pump>.image`

Usually, components in a cluster are in the same version. It is recommended to configure `spec.<pd/tidb/tikv/pump>.baseImage` and `spec.version`.

The modified configuration is not automatically applied to the TiDB cluster by default. The new configuration file is loaded only when the Pod restarts.

It is recommended that you set `spec.configUpdateStrategy` to `RollingUpdate` to enable automatic update of configurations. This way, every time the configuration is updated, all components are rolling updated automatically, and the modified configuration is applied to the cluster.

If you want to enable TiFlash in the cluster, configure `spec.pd.config.replication.enable-placement-rules` to `true` and configure `spec.tiflash` in the `${cluster_name}/tidb-cluster.yaml` file as follows:

```yaml
  pd:
    config:
      ...
      replication:
        enable-placement-rules: "true"
        ...
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 3
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

TiFlash supports mounting multiple Persistent Volumes (PVs). If you want to configure multiple PVs for TiFlash, configure multiple `resources` in `tiflash.storageClaims`, each `resources` with a separate `storage request` and `storageClassName`. For example:

```yaml
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 3
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

To enable TiCDC for the cluster, configure `spec.ticdc` in the `${cluster_name}/tidb-cluster.yaml` file:

```yaml
  ticdc:
    baseImage: pingcap/ticdc
    replicas: 3
    config:
      logLevel: info
```

To deploy TiDB cluster monitor, refer to the [TidbMonitor example](https://github.com/pingcap/tidb-operator/blob/master/manifests/monitor/tidb-monitor.yaml) and [API documentation](api-references.md) to complete TidbMonitor CR, and save it to the `${cluster_name}/tidb-monitor.yaml` file. Please switch the TidbMonitor example and API documentation to the currently used version of TiDB Operator.

### Storage class

- For the production environment, local storage is recommended. The actual local storage in Kubernetes clusters might be sorted by disk types, such as `nvme-disks` and `sas-disks`.
- For the demonstration environment or functional verification, you can use network storage, such as `ebs` and `nfs`.

Different components of a TiDB cluster have different disk requirements. Before deploying a TiDB cluster, select the appropriate storage class for each component according to the storage classes supported by the current Kubernetes cluster and usage scenario.

You can set the storage class by modifying `storageClassName` of each component in `${cluster_name}/tidb-cluster.yaml` and `${cluster_name}/tidb-monitor.yaml`. For the [storage classes](configure-storage-class.md) supported by the Kubernetes cluster, check with your system administrator.

> **Note:**
>
> If you set a storage class that does not exist in the TiDB cluster that you are creating, then the cluster creation goes to the Pending state. In this situation, you must [destroy the TiDB cluster in Kubernetes](destroy-a-tidb-cluster.md).

### Cluster topology

The deployed cluster topology by default has 3 PD Pods, 3 TiKV Pods, and 2 TiDB Pods. In this deployment topology, the scheduler extender of TiDB Operator requires at least 3 nodes in the Kubernetes cluster to provide high availability.

If the number of Kubernetes cluster nodes is less than 3, 1 PD Pod goes to the Pending state, and neither TiKV Pods nor TiDB Pods are created.

When the number of nodes in the Kubernetes cluster is less than 3, to start the TiDB cluster, you can reduce both the number of PD Pods and the number of TiKV Pods in the default deployment to `1`.

## Deploy TiDB Cluster

After you deploy and configure TiDB Operator, deploy the TiDB cluster by the following steps:

1. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create namespace ${namespace}
    ```

    > **Note:**
    >
    > A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. You can give it a name that is easy to memorize, such as the same name as `cluster-name`.

2. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ${cluster_name} -n ${namespace}
    ```

3. View the Pod status:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${cluster_name}
    ```

You can use TiDB Operator to deploy and manage multiple TiDB clusters in a single Kubernetes cluster by repeating the above procedure and replacing `cluster-name` with a different name.

Different clusters can be in the same or different `namespace`, which is based on your actual needs.

### Initialize TiDB cluster

If you want to initialize your cluster after deployment, refer to [Initialize a TiDB Cluster in Kubernetes](initialize-a-cluster.md).
