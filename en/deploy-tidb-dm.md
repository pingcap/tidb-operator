---
title: Deploy and Use DM in Kubernetes
summary: Learn how to deploy and use TiDB DM cluster in Kubernetes.
---

# Deploy and Use DM in Kubernetes

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

- `spec.version`: the format is `imageTag`, such as `v2.0.6`.
- `spec.<master/worker>.baseImage`: the format is `imageName`, such as `pingcap/dm`.
- `spec.<master/worker>.version`: the format is `imageTag`, such as `v2.0.6`.

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
  version: v2.0.6
  pvReclaimPolicy: Retain
  discovery: {}
  master:
    baseImage: pingcap/dm
    imagePullPolicy: IfNotPresent
    service:
      type: NodePort
      # Configures masterNodePort when you need to expose the DM-master service to a fixed NodePort
      # masterNodePort: 30020
    replicas: 1
    storageSize: "1Gi"
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
    replicas: 1
    storageSize: "1Gi"
    requests:
      cpu: 1
    config:
      keepalive-ttl: 15

```

### Topology Spread Constraint

By configuring `topologySpreadConstraints`, you can make pods evenly spread in different topologies. For instructions about configuring `topologySpreadConstraints`, see [Pod Topology Spread Constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/).

> **Note:**
>
> To use `topologySpreadConstraints`, you must enable the `EvenPodsSpread` feature gate. If the Kubernetes version in use is earlier than v1.16 or if the `EvenPodsSpread` feature gate is disabled, the configuration of `topologySpreadConstraints` does not take effect.

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

> **Note:**
>
> You can use this feature to replace [TiDB Scheduler](tidb-scheduler.md) for evenly scheduling.

## Deploy the DM cluster

After configuring the yaml file of the DM cluster in the above steps, execute the following command to deploy the DM cluster:

``` shell
kubectl apply -f ${dm_cluster_name}.yaml -n ${namespace}
```

If the server does not have an external network, you need to download the Docker image used by the DM cluster and upload the image to the server, and then execute `docker load` to install the Docker image on the server:

1. Deploy a DM cluster requires the following Docker image (assuming the version of the DM cluster is v2.0.6):

    ```shell
    pingcap/dm:v2.0.6
    ```

2. To download the image, execute the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    docker pull pingcap/dm:v2.0.6
    docker save -o dm-v2.0.6.tar pingcap/dm:v2.0.6
    ```

3. Upload the Docker image to the server, and execute `docker load` to install the image on the server:

    {{< copyable "shell-regular" >}}

    ```shell
    docker load -i dm-v2.0.6.tar
    ```

After deploying the DM cluster, execute the following command to view the Pod status:

```shell
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

## Enable DM data migration tasks

You can access the DM-master service using dmctl in the following two methods:

**Method #1**: Attach to the DM-master or DM-worker Pod to use the built-in `dmctl` in the image.

**Method #2**: Expose the DM-master service by [accessing the DM cluster in Kubernetes](#access-the-dm-cluster-in-kubernetes) and use `dmctl` outside the pods to access the exposed DM-master service.

It is recommended to use **Method #1** for migration. The following steps take **Method #1** as an example to introduce how to start a DM data migration task.

The differences between Method #1 and Method #2 are that the file locations of `source.yaml` and `task.yaml` are different, and that in Method #2 you need to configure the exposed DM-master service address in the `master-addr` configuration item of `dmctl`.

### Get into the Pod

Attach to the DM-master Pod by executing the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl exec -ti ${dm_cluster_name}-dm-master-0 -n ${namespace} - /bin/sh
```

### Create data source

1. Write MySQL-1 related information to `source1.yaml` file, which can refer to [Create data source](https://docs.pingcap.com/tidb-data-migration/v2.0/migrate-data-using-dm#step-3-create-data-source).

2. Configure the `from.host` in the `source1.yaml` file as the MySQL host address that the Kubernetes cluster can access internally.

3. Configure the `relay-dir` in the `source1.yaml` file as a subdirectory of the persistent volume in the Pod mount `/var/lib/dm-worker` directory. For example, `/var/lib/dm-worker/relay`.

4. After you prepare the `source1.yaml` file, load the MySQL-1 data source into the DM cluster by executing the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    /dmctl --master-addr ${dm_cluster_name}-dm-master:8261 operate-source create source1.yaml
    ```

5. For MySQL-2 and other data sources, use the same method to modify the relevant information in the data source `yaml` file and execute the same dmctl command to load the corresponding data source into the DM cluster.

### Configure migration tasks

1. Edit task configuration file `task.yaml`, which can refer to [Configure the data migration task](https://docs.pingcap.com/tidb-data-migration/v2.0/migrate-data-using-dm#step-4-configure-the-data-migration-task).

2. Configure the `target-database.host` in `task.yaml` as the TiDB host address that the Kubernetes cluster can access internally. If the cluster is deployed by TiDB Operator, configure the host as `${tidb_cluster_name}-tidb.${namespace}`.

3. In the `task.yaml` file, take the following steps:

    - Add the `loaders.${customized_name}.dir` field as the import and export directory for the full volume data, where `${customized_name}` is a name that you can customize. 
    - Configure the `loaders.${customized_name}.dir` field as the subdirectory of the persistent volume in the Pod `/var/lib/dm-worker` directory. For example, `/var/lib/dm-worker/dumped_data`.
    - Reference `${customized_name}` in the instance configuration. For example, `mysql-instances[0].loader-config-name: "{customized_name}"`.

### Start/Check/Stop the migration tasks

Refer to the Steps 5-7 in [Migrate Data Using DM](https://docs.pingcap.com/tidb-data-migration/v2.0/migrate-data-using-dm#step-5-start-the-data-migration-task) and fill in the master-addr as `${dm_cluster_name}-dm-master:8261`.
