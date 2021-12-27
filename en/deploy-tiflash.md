---
title: Deploy TiFlash in Kubernetes
summary: Learn how to deploy TiFlash in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-tiflash/']
---

# Deploy TiFlash in Kubernetes

This document describes how to deploy TiFlash in Kubernetes.

## Prerequisites

* [Deploy TiDB Operator](deploy-tidb-operator.md).

## Fresh TiFlash deployment

To deploy TiFlash when deploying the TiDB cluster, refer to [Deploy TiDB on General Kubernetes](deploy-on-general-kubernetes.md).

## Add TiFlash to an existing TiDB cluster

Edit the `TidbCluster` Custom Resource:

{{< copyable "shell-regular" >}}

``` shell
kubectl eidt tc ${cluster_name} -n ${namespace}
```

Add the TiFlash configuration as follows:

{{< copyable "" >}}

```yaml
spec:
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 0
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

To configure other parameters, refer to [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

To deploy Enterprise Edition of TiFlash, edit the `dm.yaml` file above to set `spec.tiflash.baseImage` to the enterprise image (`pingcap/tiflash-enterprise`).

For example:

```yaml
spec:
  tiflash:
    baseImage: pingcap/tiflash-enterprise
```

TiFlash supports mounting multiple Persistent Volumes (PVs). If you want to configure multiple PVs for TiFlash, configure multiple `resources` in `tiflash.storageClaims`, each `resources` with a separate `storage request` and `storageClassName`. For example:

```yaml
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 0
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

TiDB Operator manages TiFlash by creating [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/). Since `StatefulSet` does not support modifying `volumeClaimTemplates` after creation, updating `storageClaims` to add the disk cannot mount the additional PV to the Pod. There are two solutions:

* When deploying the TiFlash cluster for the first time, determine how many PVs are required and configure `storageClaims`.
* If you really want to add a PV, after configuring `storageClaims`, you need to manually delete the TiFlash StatefulSet (`kubectl delete sts -n ${namespace} ${cluster_name}-tiflash`) and wait for the TiDB Operator to recreate it.

To add TiFlash component to an existing TiDB cluster, you need to set `replication.enable-placement-rules: true` in PD. After you add the TiFlash configuration in `TidbCluster` by taking the above steps, TiDB Operator automatically configures `replication.enable-placement-rules: true` in PD.

If the server does not have an external network, refer to [deploy the TiDB cluster](deploy-on-general-kubernetes.md#deploy-the-tidb-cluster) to download the required Docker image on the machine with an external network and upload it to the server.

## Remove TiFlash

1. Adjust the number of replicas of the tables replicated to the TiFlash cluster.

    To completely remove TiFlash, you need to set the number of replicas of all tables replicated to the TiFlash to `0`.

    1. To connect to the TiDB service, refer to the steps in [Access the TiDB Cluster in Kubernetes](access-tidb.md).

    2. To adjust the number of replicas of the tables replicated to the TiFlash cluster, run the following command:

       {{< copyable "sql" >}}

        ```sql
        alter table <db_name>.<table_name> set tiflash replica 0;
        ```

2. Wait for the TiFlash replicas of the related tables to be deleted.

    Connect to the TiDB service and run the following command. If you can not find the replication information of the related tables, it means that the replicas are deleted:

    {{< copyable "sql" >}}

    ```sql
    SELECT * FROM information_schema.tiflash_replica WHERE TABLE_SCHEMA = '<db_name>' and TABLE_NAME = '<table_name>';
    ```

3. To remove TiFlash Pods, run the following command to modify `spec.tiflash.replicas` to `0`:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl edit tidbcluster ${cluster_name} -n ${namespace}
    ```

4. Check the state of TiFlash Pods and TiFlash stores.

   First, run the following command to check whether you delete the TiFlash Pod successfully:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get pod -n ${namespace} -l app.kubernetes.io/component=tiflash,app.kubernetes.io/instance=${cluster_name}
    ```

   If the output is empty, it means that you delete the Pod of the TiFlash cluster successfully.

   To check whether the stores of the TiFlash are in the `Tombstone` state, run the following command:

    ```bash
    kubectl get tidbcluster ${cluster_name} -n ${namespace} -o yaml
    ```

   The value of the `status.tiflash` field in the output result is similar to the example below.

    ```
    tiflash:
        ...
        tombstoneStores:
        "88":
            id: "88"
            ip: basic-tiflash-0.basic-tiflash-peer.default.svc
            lastHeartbeatTime: "2020-12-31T04:42:12Z"
            lastTransitionTime: null
            leaderCount: 0
            podName: basic-tiflash-0
            state: Tombstone
        "89":
            id: "89"
            ip: basic-tiflash-1.basic-tiflash-peer.default.svc
            lastHeartbeatTime: "2020-12-31T04:41:50Z"
            lastTransitionTime: null
            leaderCount: 0
            podName: basic-tiflash-1
            state: Tombstone
    ```

    Only after you delete all Pods of the TiFlash cluster successfully and all the TiFlash stores have changed to the `Tombstone` state, can you perform the next operation.

5. Delete the TiFlash StatefulSet.

   To modify the TidbCluster CR and delete the `spec.tiflash` field, run the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl edit tidbcluster ${cluster_name} -n ${namespace}
    ```

   To delete the TiFlash StatefulSet, run the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl delete statefulsets -n ${namespace} -l app.kubernetes.io/component=tiflash,app.kubernetes.io/instance=${cluster_name}
    ```

   To check whether you delete the StatefulSet of the TiFlash cluster successfully, run the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get sts -n ${namespace} -l app.kubernetes.io/component=tiflash,app.kubernetes.io/instance=${cluster_name}
    ```

   If the output is empty, it means that you delete the StatefulSet of the TiFlash cluster successfully.

6. (Optional) Delete PVC and PV.

   If you confirm that you do not use the data in TiFlash, and you want to delete the data, you need to strictly follow the steps below to delete the data in TiFlash:

   1. Delete the PVC object corresponding to the PV

      {{< copyable "shell-regular" >}}

      ```bash
      kubectl delete pvc -n ${namespace} -l app.kubernetes.io/component=tiflash,app.kubernetes.io/instance=${cluster_name}
      ```

    2. If the PV reclaim policy is `Retain`, the corresponding PV is still retained after you delete the PVC object. If you want to delete the PV, you can set the reclaim policy of the PV to `Delete`, and the PV can be deleted and recycled automatically.

      {{< copyable "shell-regular" >}}

      ```bash
      kubectl patch pv ${pv_name} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
      ```

      In the above command, `${pv_name}` represents the PV name of the TiFlash cluster. You can check the PV name by running the following command:

      {{< copyable "shell-regular" >}}

      ```bash
      kubectl get pv -l app.kubernetes.io/component=tiflash,app.kubernetes.io/instance=${cluster_name}
      ```

## Configuration notes for different versions

Starting from TiDB Operator v1.1.5, the default configuration of `spec.tiflash.config.config.flash.service_addr` is changed from `${clusterName}-tiflash-POD_NUM.${clusterName}-tiflash-peer.${namespace}.svc:3930` to `0.0.0.0:3930`, and TiFlash needs to configure `spec.tiflash.config.config.flash.service_addr` to `0.0.0.0:3930` since v4.0.5.

Therefore, for different TiFlash and TiDB Operator versions, you need to pay attention to the following configurations:

* If the TiDB Operator version <= v1.1.4
    * If the TiFlash version <= v4.0.4, no need to manually configure `spec.tiflash.config.config.flash.service_addr`.
    * If the TiFlash version >= v4.0.5, you need to set `spec.tiflash.config.config.flash.service_addr` to `0.0.0.0:3930` in the `TidbCluster` CR.
* If the TiDB Operator version >= v1.1.5
    * If the TiFlash version <= v4.0.4, you need to set `spec.tiflash.config.config.flash.service_addr` to `${clusterName}-tiflash-POD_NUM.${clusterName}-tiflash-peer.${namespace}.svc:3930` in the `TidbCluster` CR. `${clusterName}` and `${namespace}` need to be replaced according to the real case.
    * If the TiFlash version >= v4.0.5, no need to manually configure `spec.tiflash.config.config.flash.service_addr`.
    * If you upgrade from TiFlash v4.0.4 or lower versions to TiFlash v4.0.5 or higher versions, you need to delete the configuration of `spec.tiflash.config.config.flash.service_addr` in the `TidbCluster` CR.
