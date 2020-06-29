---
title: Configure a TiDB Cluster in Kubernetes
summary: Learn how to configure a TiDB cluster in Kubernetes.
category: how-to
aliases: ['/docs/tidb-in-kubernetes/dev/configure-a-tidb-cluster/']
---

# Configure a TiDB Cluster in Kubernetes

This document introduces how to configure a TiDB cluster for production deployment. It covers the following content:

- [Configure resources](#configure-resources)

- [Configure TiDB deployment](#configure-tidb-deployment)

- [Configure high availability](#configure-high-availability)

## Configure resources

Before deploying a TiDB cluster, it is necessary to configure the resources for each component of the cluster depending on your needs. PD, TiKV and TiDB are the core service components of a TiDB cluster. In a production environment, you need to configure resources of these components according to their needs. For details, refer to [Hardware Recommendations](https://pingcap.com/docs/stable/hardware-and-software-requirements/).

To ensure the proper scheduling and stable operation of the components of the TiDB cluster in Kubernetes, it is recommended to set Guaranteed-level quality of service (QoS) by making `limits` equal to `requests` when configuring resources. For details, refer to [Configure Quality of Service for Pods](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/).

If you are using a NUMA-based CPU, you need to enable `Static`'s CPU management policy on the node for better performance. In order to allow the TiDB cluster component to monopolize the corresponding CPU resources, the CPU quota must be an integer greater than or equal to `1`, apart from setting Guaranteed-level QoS as mentioned above. For details, refer to [Control CPU Management Policies on the Node](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies).

## Configure TiDB deployment

To configure a TiDB deployment, you need to configure the `TiDBCluster` CR. Refer to the [TidbCluster example](https://github.com/pingcap/tidb-operator/blob/master/examples/tiflash/tidb-cluster.yaml) for an example. For the complete configurations of `TiDBCluster` CR, refer to [API documentation](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md).

> **Note:**
>
> It is recommended to organize configurations for a TiDB cluster under a directory of `cluster_name` and save it as `${cluster_name}/tidb-cluster.yaml`.
The modified configuration is not automatically applied to the TiDB cluster by default. The new configuration file is loaded only when the Pod restarts.

It is recommended that you set `spec.configUpdateStrategy` to `RollingUpdate` to enable automatic update of configurations. This way, every time the configuration is updated, all components are rolling updated automatically, and the modified configuration is applied to the cluster.

### Cluster name

The cluster name can be configured by changing `metadata.name` in the `TiDBCuster` CR.

### Version

Usually, components in a cluster are in the same version. It is recommended to configure `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.baseImage` and `spec.version`, if you need to configure different versions for different components, you can configure `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`.
Here are the formats of the parameters:

- `spec.version`: the format is `imageTag`, such as `v4.0.0`

- `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.baseImage`: the format is `imageName`, such as `pingcap/tidb`

- `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`: the format is `imageTag`, such as `v4.0.0`

### Storage class

You can set the storage class by modifying `storageClassName` of each component in `${cluster_name}/tidb-cluster.yaml` and `${cluster_name}/tidb-monitor.yaml`. For the [storage classes](configure-storage-class.md) supported by the Kubernetes cluster, check with your system administrator.

Different components of a TiDB cluster have different disk requirements. Before deploying a TiDB cluster, select the appropriate storage class for each component according to the storage classes supported by the current Kubernetes cluster and usage scenario.

For the production environment, local storage is recommended for TiKV. The actual local storage in Kubernetes clusters might be sorted by disk types, such as `nvme-disks` and `sas-disks`.

For demonstration environment or functional verification, you can use network storage, such as `ebs` and `nfs`.

> **Note:**
>
> If you set a storage class that does not exist in the TiDB cluster that you are creating, then the cluster creation goes to the Pending state. In this situation, you must [destroy the TiDB cluster in Kubernetes](destroy-a-tidb-cluster.md).

### Cluster topology

#### PD/TiKV/TiDB

The deployed cluster topology by default has 3 PD Pods, 3 TiKV Pods, and 2 TiDB Pods. In this deployment topology, the scheduler extender of TiDB Operator requires at least 3 nodes in the Kubernetes cluster to provide high availability. You can modify the `replicas` configuration to change the number of pods for each component.

> **Note:**
>
> If the number of Kubernetes cluster nodes is less than 3, 1 PD Pod goes to the Pending state, and neither TiKV Pods nor TiDB Pods are created. When the number of nodes in the Kubernetes cluster is less than 3, to start the TiDB cluster, you can reduce both the number of PD Pods and the number of TiKV Pods in the default deployment to `1`.

#### Enable TiFlash

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

> **Warning:**
>
> Since TiDB Operator will mount PVs automatically in the **order** of the items in the `storageClaims` list, if you need to add more disks to TiFlash, make sure to append the new item only to the **end** of the original items, and **DO NOT** modify the order of the original items.

#### Enable TiCDC

If you want to enable TiCDC in the cluster, you can add TiCDC spec to the `TiDBCluster` CR. For example:

```yaml
  spec:
    ticdc:
      baseImage: pingcap/ticdc
      replicas: 3
```

### Configure TiDB components

This section introduces how to configure the parameters of TiDB/TiKV/PD/TiFlash/TiCDC.

The current TiDB Operator v1.1 supports all parameters of TiDB v4.0.

#### Configure TiDB parameters

TiDB parameters can be configured by `spec.tidb.config` in TidbCluster Custom Resource.

For example:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
....
  tidb:
    image: pingcap.com/tidb:v4.0.0
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

For all the configurable parameters of TiDB, refer to [TiDB Configuration File](https://pingcap.com/docs/stable/reference/configuration/tidb-server/configuration-file/).

> **Note:**
>
> If you deploy your TiDB cluster using CR, make sure that `Config: {}` is set, no matter you want to modify `config` or not. Otherwise, TiDB components might not be started successfully. This step is meant to be compatible with `Helm` deployment.

#### Configure TiKV parameters

TiKV parameters can be configured by `spec.tikv.config` in TidbCluster Custom Resource.

For example:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
....
  tikv:
    image: pingcap.com/tikv:v4.0.0
    config:
      log-level: "info"
      slow-log-threshold: "1s"
    replicas: 1
    requests:
      cpu: 2
```

For all the configurable parameters of TiKV, refer to [TiKV Configuration File](https://pingcap.com/docs/stable/reference/configuration/tikv-server/configuration-file/).

> **Note:**
>
> If you deploy your TiDB cluster using CR, make sure that `Config: {}` is set, no matter you want to modify `config` or not. Otherwise, TiKV components might not be started successfully. This step is meant to be compatible with `Helm` deployment.

#### Configure PD parameters

PD parameters can be configured by `spec.pd.config` in TidbCluster Custom Resource.

For example:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
.....
  pd:
    image: pingcap.com/pd:v4.0.0
    config:
      lease: 3
      enable-prevote: true
```

For all the configurable parameters of PD, refer to [PD Configuration File](https://pingcap.com/docs/stable/reference/configuration/pd-server/configuration-file/).

> **Note:**
>
> If you deploy your TiDB cluster using CR, make sure that `Config: {}` is set, no matter you want to modify `config` or not. Otherwise, PD components might not be started successfully. This step is meant to be compatible with `Helm` deployment.

#### Configure TiFlash parameters

TiFlash parameters can be configured by `spec.tiflash.config` in TidbCluster Custom Resource.

For example:

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

For all the configurable parameters of TiFlash, refer to [TiFlash Configuration File](https://pingcap.com/docs/stable/tiflash/tiflash-configuration/).

#### Configure TiCDC start parameters

You can configure TiCDC start parameters through `spec.ticdc.config` in TidbCluster Custom Resource.

For example:

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

For all configurable start parameters of TiCDC, see [TiCDC start parameters](https://pingcap.com/docs/stable/ticdc/deploy-ticdc/#manually-add-ticdc-component-to-an-existing-tidb-cluster).

## Configure high availability

> **Note:**
>
> TiDB Operator provides a custom scheduler that guarantees TiDB service can tolerate host level failures through the specified scheduling algorithm. Currently, the TiDB cluster uses this scheduler as the default scheduler, which is configured through the item `spec.schedulerName`. This section focuses on configuring a TiDB cluster to tolerate failures at other levels such as rack, zone or region. This section is optional.

TiDB is a distributed database and its high availability must ensure that when any physical topology node fails, not only the service is unaffected, but also the data is complete and available. The two configurations of high availability are described separately as follows.

### High avalability of TiDB service

High availability at other levels (such as rack, zone, region) are guaranteed by Affinity's `PodAntiAffinity`. `PodAntiAffinity` can avoid the situation where different instances of the same component are deployed on the same physical topology node. In this way, disaster recovery is achieved. Detailed user guide for Affinity: [Affinity & AntiAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity).

The following is an example of a typical service high availability setup:

{{< copyable "" >}}

```shell
affinity:
 podAntiAffinity:
   preferredDuringSchedulingIgnoredDuringExecution:
   # this term works when the nodes have the label named region
   - weight: 10
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "region"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named zone
   - weight: 20
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "zone"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named rack
   - weight: 40
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "rack"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named kubernetes.io/hostname
   - weight: 80
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "kubernetes.io/hostname"
       namespaces:
       - ${namespace}
```

### High availability of data

Before configuring the high availability of data, read [Information Configuration of the Cluster Typology](https://pingcap.com/docs/stable/location-awareness/) which describes how high availability of TiDB cluster is implemented.

To add the data high availability feature in Kubernetes:

1. Set the label collection of topological location for PD

    Replace the `location-labels` information in the `pd.config` with the label collection that describes the topological location on the nodes in the Kubernetes cluster.

    > **Note:**
    >
    > * For PD versions < v3.0.9, the `/` in the label name is not supported.
    > * If you configure `host` in the `location-labels`, TiDB Operator will get the value from the `kubernetes.io/hostname` in the node label.

2. Set the topological information of the Node where the TiKV node is located.

    TiDB Operator automatically obtains the topological information of the Node for TiKV and calls the PD interface to set this information as the information of TiKV's store labels. Based on this topological information, the TiDB cluster schedules the replicas of the data.

    If the Node of the current Kubernetes cluster does not have a label indicating the topological location, or if the existing label name of topology contains `/`, you can manually add a label to the Node by running the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl label node ${node_name} region=${region_name} zone=${zone_name} rack=${rack_name} kubernetes.io/hostname=${host_name}
    ```

    In the command above, `region`, `zone`, `rack`, and `kubernetes.io/hostname` are just examples. The name and number of the label to be added can be arbitrarily defined, as long as it conforms to the specification and is consistent with the labels set by `location-labels` in `pd.config`.
