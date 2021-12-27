---
title: Deploy DM in Kubernetes
summary: Learn how to deploy TiDB DM cluster in Kubernetes.
---

# Deploy DM in Kubernetes

[TiDB Data Migration](https://docs.pingcap.com/tidb-data-migration/v2.0) (DM) is an integrated data migration task management platform that supports the full data migration and the incremental data replication from MySQL/MariaDB into TiDB. This document describes how to deploy DM in Kubernetes using TiDB Operator and how to migrate MySQL data to TiDB cluster using DM.

## Prerequisites

* Complete [deploying TiDB Operator](deploy-tidb-operator.md).

> **Note:**
>
> Make sure that the TiDB Operator version >= 1.2.0.

## Configure DM deployment

To configure the DM deployment, you need to configure the `DMCluster` Custom Resource (CR). For the complete configurations of the `DMCluster` CR, refer to the [DMCluster example](https://github.com/pingcap/tidb-operator/blob/master/examples/dm/dm-cluster.yaml) and [API documentation](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md#dmcluster). Note that you need to choose the example and API of the current TiDB Operator version.

### Cluster name

Configure the cluster name by changing the `metadata.name` in the `DMCluster` CR.

### Version

Usually, components in a cluster are in the same version. It is recommended to configure only `spec.<master/worker>.baseImage` and `spec.version`. If you need to deploy different versions for different components, configure `spec.<master/worker>.version`.

The formats of the related parameters are as follows:

- `spec.version`: the format is `imageTag`, such as `v2.0.7`.
- `spec.<master/worker>.baseImage`: the format is `imageName`, such as `pingcap/dm`.
- `spec.<master/worker>.version`: the format is `imageTag`, such as `v2.0.7`.

TiDB Operator only supports deploying DM 2.0 and later versions.

### Cluster

#### Configure DM-master

DM-master is an indispensable component of the DM cluster. You need to deploy at least three DM-master Pods if you want to achieve high availability.

You can configure DM-master parameters by `spec.master.config` in `DMCluster` CR. For complete DM-master configuration parameters, refer to [DM-master Configuration File](https://docs.pingcap.com/tidb-data-migration/v2.0/dm-master-configuration-file).

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${dm_cluster_name}
  namespace: ${namespace}
spec:
  version: v2.0.7
  pvReclaimPolicy: Retain
  discovery: {}
  master:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    imagePullPolicy: IfNotPresent
    service:
      type: NodePort
      # Configures masterNodePort when you need to expose the DM-master service to a fixed NodePort
      # masterNodePort: 30020
    replicas: 1
    storageSize: "10Gi"
    requests:
      cpu: 1
    config:
      rpc-timeout: 40s

```

#### Configure DM-worker

You can configure DM-worker parameters by `spec.worker.config` in `DMCluster` CR. For complete DM-worker configuration parameters，refer to [DM-worker Configuration File](https://docs.pingcap.com/tidb-data-migration/v2.0/dm-worker-configuration-file).

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${dm_cluster_name}
  namespace: ${namespace}
spec:
  ...
  worker:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    replicas: 1
    storageSize: "100Gi"
    requests:
      cpu: 1
    config:
      keepalive-ttl: 15

```

### Topology Spread Constraint

By configuring `topologySpreadConstraints`, you can make pods evenly spread in different topologies. For instructions about configuring `topologySpreadConstraints`, see [Pod Topology Spread Constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/).

To use `topologySpreadConstraints`, you must meet the following conditions:

- Your Kubernetes cluster uses `default-scheduler` instead of `tidb-scheduler`. For details, refer to [tidb-scheduler and default-scheduler](tidb-scheduler.md#tidb-scheduler-and-default-scheduler).
- Your Kubernetes cluster enables the `EvenPodsSpread` feature gate. If the Kubernetes version in use is earlier than v1.16 or if the `EvenPodsSpread` feature gate is disabled, the configuration of `topologySpreadConstraints` does not take effect.

You can either configure `topologySpreadConstraints` at a cluster level (`spec.topologySpreadConstraints`) for all components or at a component level (such as `spec.tidb.topologySpreadConstraints`) for specific components.

The following is an example configuration:

{{< copyable "" >}}

```yaml
topologySpreadConstrains:
- topologyKey: kubernetes.io/hostname
- topologyKey: topology.kubernetes.io/zone
```

The example configuration can make pods of the same component evenly spread on different zones and nodes.

Currently, `topologySpreadConstraints` only supports the configuration of the `topologyKey` field. In the pod spec, the above example configuration will be automatically expanded as follows:

```yaml
topologySpreadConstrains:
- topologyKey: kubernetes.io/hostname
  maxSkew: 1
  whenUnsatisfiable: DoNotSchedule
  labelSelector: <object>
- topologyKey: topology.kubernetes.io/zone
  maxSkew: 1
  whenUnsatisfiable: DoNotSchedule
  labelSelector: <object>
```

## Deploy the DM cluster

After configuring the yaml file of the DM cluster in the above steps, execute the following command to deploy the DM cluster:

``` shell
kubectl apply -f ${dm_cluster_name}.yaml -n ${namespace}
```

If the server does not have an external network, you need to download the Docker image used by the DM cluster and upload the image to the server, and then execute `docker load` to install the Docker image on the server:

1. Deploy a DM cluster requires the following Docker image (assuming the version of the DM cluster is v2.0.7):

    ```bash
    pingcap/dm:v2.0.7
    ```

2. To download the image, execute the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    docker pull pingcap/dm:v2.0.7
    docker save -o dm-v2.0.7.tar pingcap/dm:v2.0.7
    ```

3. Upload the Docker image to the server, and execute `docker load` to install the image on the server:

    {{< copyable "shell-regular" >}}

    ```bash
    docker load -i dm-v2.0.7.tar
    ```

After deploying the DM cluster, execute the following command to view the Pod status:

```bash
kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${dm_cluster_name}
```

You can use TiDB Operator to deploy and manage multiple DM clusters in a single Kubernetes cluster by repeating the above procedure and replacing `${dm_cluster_name}` with a different name.

Different clusters can be in the same or different `namespace`, which is based on your actual needs.

## Access the DM cluster in Kubernetes

To access DM-master in the pod within a Kubernetes cluster, use the DM-master service domain name `${cluster_name}-dm-master.${namespace}`.

To access the DM cluster outside a Kubernetes cluster, expose the DM-master port by editing the `spec.master.service` field configuration in the `DMCluster` CR.

```yaml
spec:
  ...
  master:
    service:
      type: NodePort
```

You can access the DM-master service via the address of `${kubernetes_node_ip}:${node_port}`.

For more service exposure methods, refer to [Access the TiDB Cluster](access-tidb.md).

## What’s next

- To migrate MySQL data to your TiDB cluster using DM in Kubernetes, see [Migrate MySQL Data to TiDB Cluster Using DM](use-tidb-dm.md).
- To enable TLS between components of the DM cluster in Kubernetes, see [Enable TLS for DM](enable-tls-for-dm.md).