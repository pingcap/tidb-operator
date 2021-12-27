---
title: Use DM in Kubernetes
summary: Learn how to migrate MySQL data to TiDB cluster using DM in Kubernetes.
---

# Use DM in Kubernetes

[TiDB Data Migration](https://docs.pingcap.com/tidb-data-migration/v2.0) (DM) is an integrated data migration task management platform that supports the full data migration and the incremental data replication from MySQL/MariaDB into TiDB. This document describes how to migrate MySQL data to TiDB cluster using DM in Kubernetes.

## Prerequisites

* Complete [deploying TiDB Operator](deploy-tidb-operator.md).
* Complete [deploying DM in Kubernetes](deploy-tidb-dm.md).

> **Note:**
>
> Make sure that the TiDB Operator version >= 1.2.0.

## Enable DM data migration tasks

You can access the DM-master service using dmctl in the following two methods:

**Method #1**: Attach to the DM-master or DM-worker Pod to use the built-in `dmctl` in the image.

**Method #2**: Expose the DM-master service by [accessing the DM cluster in Kubernetes](deploy-tidb-dm.md#access-the-dm-cluster-in-kubernetes) and use `dmctl` outside the pods to access the exposed DM-master service.

It is recommended to use **Method #1** for migration. The following steps take **Method #1** as an example to introduce how to start a DM data migration task.

The differences between Method #1 and Method #2 are that the file locations of `source.yaml` and `task.yaml` are different, and that in Method #2 you need to configure the exposed DM-master service address in the `master-addr` configuration item of `dmctl`.

### Get into the Pod

Attach to the DM-master Pod by executing the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl exec -ti ${dm_cluster_name}-dm-master-0 -n ${namespace} - /bin/sh
```

### Create data source

1. Write MySQL-1 related information to `source1.yaml` file, which can refer to [Create data source](https://docs.pingcap.com/tidb-data-migration/v2.0/migrate-data-using-dm#step-3-create-data-source).

2. Configure the `from.host` in the `source1.yaml` file as the MySQL host address that the Kubernetes cluster can access internally.

3. Configure the `relay-dir` in the `source1.yaml` file as a subdirectory of the persistent volume in the Pod mount `/var/lib/dm-worker` directory. For example, `/var/lib/dm-worker/relay`.

4. After you prepare the `source1.yaml` file, load the MySQL-1 data source into the DM cluster by executing the following command:

    {{< copyable "shell-regular" >}}

    ```bash
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

Refer to the corresponding steps in [Migrate Data Using DM](https://docs.pingcap.com/tidb-data-migration/v2.0/migrate-data-using-dm#step-5-start-the-data-migration-task) and fill in the master-addr as `${dm_cluster_name}-dm-master:8261`.
