---
title: Scale TiDB in Kubernetes
summary: Learn how to horizontally and vertically scale up and down a TiDB cluster in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/scale-a-tidb-cluster/']
---

# Scale TiDB in Kubernetes

This document introduces how to horizontally and vertically scale a TiDB cluster in Kubernetes.

## Horizontal scaling

Horizontally scaling TiDB means that you scale TiDB out or in by adding or remove nodes in your pool of resources. When you scale a TiDB cluster, PD, TiKV, and TiDB are scaled out or in sequentially according to the values of their replicas. Scaling out operations add nodes based on the node ID in ascending order, while scaling in operations remove nodes based on the node ID in descending order.

Currently, the TiDB cluster supports management by TidbCluster Custom Resource (CR).

### Scale PD, TiDB, and TiKV

Modify `spec.pd.replicas`, `spec.tidb.replicas`, and `spec.tikv.replicas` in the `TidbCluster` object of the cluster to a desired value using kubectl. You can modify the values in the local file or using online command.

- You can also online modify the `TidbCluster` definition in the Kubernetes cluster by running the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl edit tidbcluster ${cluster_name} -n ${namespace}
    ```

Check whether the TiDB cluster in Kubernetes has updated to your desired definition by running the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl get tidbcluster ${cluster_name} -n ${namespace} -oyaml
```

In the `TidbCluster` file output by the command above, if the values of `spec.pd.replicas`, `spec.tidb.replicas`, and `spec.tikv.replicas` are consistent with the values you have modified, check whether the number of `TidbCluster` Pods has increased or decreased by running the following command:

{{< copyable "shell-regular" >}}

```bash
watch kubectl -n ${namespace} get pod -o wide
```

For the PD and TiDB components, it might take 10-30 seconds to scale in or out.

For the TiKV component, it might take 3-5 minutes to scale in or out because the process involves data migration.

#### Scale out TiFlash

If TiFlash is deployed in the cluster, you can scale out TiFlash by modifying `spec.tiflash.replicas`.

#### Scale TiCDC

If TiCDC is deployed in the cluster, you can scale out TiCDC by modifying `spec.ticdc.replicas`.

#### Scale in TiFlash

1. Expose the PD service by using `port-forward`:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379
    ```

2. Open a **new** terminal tab or window. Check the maximum number (`N`) of replicas of all data tables with which TiFlash is enabled by running the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    curl 127.0.0.1:2379/pd/api/v1/config/rules/group/tiflash | grep count
    ```

    In the printed result, the largest value of `count` is the maximum number (`N`) of replicas of all data tables.

3. Go back to the terminal window in Step 1, where `port-forward` is running. Press <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop `port-forward`.

4. After the scale-in operation, if the number of remaining Pods in TiFlash >= `N`, skip to Step 6. Otherwise, take the following steps:

    1. Refer to [Access TiDB](access-tidb.md) and connect to the TiDB service.

    2. For all the tables that have more replicas than the remaining Pods in TiFlash, run the following command:

        {{< copyable "sql" >}}

        ```sql
        alter table <db_name>.<table_name> set tiflash replica 0;
        ```

5. Wait for TiFlash replicas in the related tables to be deleted.

    Connect to the TiDB service, and run the following command:

    {{< copyable "sql" >}}

    ```sql
    SELECT * FROM information_schema.tiflash_replica WHERE TABLE_SCHEMA = '<db_name>' and TABLE_NAME = '<table_name>';
    ```

    If you cannot view the replication information of related tables, the TiFlash replicas are successfully deleted.

6. Modify `spec.tiflash.replicas` to scale in TiFlash.

    Check whether TiFlash in the TiDB cluster in Kubernetes has updated to your desired definition. Run the following command and see whether the value of `spec.tiflash.replicas` returned is expected:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get tidbcluster ${cluster-name} -n ${namespace} -oyaml
    ```

### View the horizontal scaling status

To view the scaling status of the cluster, run the following command:

{{< copyable "shell-regular" >}}

```bash
watch kubectl -n ${namespace} get pod -o wide
```

When the number of Pods for all components reaches the preset value and all components go to the `Running` state, the horizontal scaling is completed.

> **Note:**
>
> - The PD, TiKV and TiFlash components do not trigger the rolling update operations during scaling in and out.
> - When the TiKV component scales in, TiDB Operator calls the PD interface to mark the corresponding TiKV instance as offline, and then migrates the data on it to other TiKV nodes. During the data migration, the TiKV Pod is still in the `Running` state, and the corresponding Pod is deleted only after the data migration is completed. The time consumed by scaling in depends on the amount of data on the TiKV instance to be scaled in. You can check whether TiKV is in the `Offline` state by running `kubectl get -n ${namespace} tidbcluster ${cluster_name} -o json | jq '.status.tikv.stores'`.
> - When the number of `UP` stores is equal to or less than the parameter value of `MaxReplicas` in the PD configuration, the TiKV components can not be scaled in.
> - The TiKV component does not support scale out while a scale-in operation is in progress. Forcing a scale-out operation might cause anomalies in the cluster. If an anomaly already happens, refer to [TiKV Store is in Tombstone status abnormally](exceptions.md#tikv-store-is-in-tombstone-status-abnormally) to fix it.
> - The TiFlash component has the same scale-in logic as TiKV.
> - When the PD, TiKV, and TiFlash components scale in, the PVC of the deleted node is retained during the scaling in process. Because the PV's reclaim policy is changed to `Retain`, the data can still be retrieved even if the PVC is deleted.

### Horizontal scaling failure

During the horizontal scaling operation, Pods might go to the Pending state because of insufficient resources. See [Troubleshoot the Pod in Pending state](deploy-failures.md#the-pod-is-in-the-pending-state).

## Vertical scaling

Vertically scaling TiDB means that you scale TiDB up or down by increasing or decreasing the limit of resources on the node. Vertically scaling is essentially the rolling update of the nodes.

Currently, the TiDB cluster supports management by TidbCluster Custom Resource (CR).

### Vertical scaling operations

Modify `spec.pd.resources`, `spec.tikv.resources`, and `spec.tidb.resources` in the `TidbCluster` object that corresponds to the cluster to the desired values using kubectl.

If TiFlash is deployed in the cluster, you can scale up and down TiFlash by modifying `spec.tiflash.resources`.

If TiCDC is deployed in the cluster, you can scale up and down TiCDC by modifying `spec.ticdc.resources`.

### View the vertical scaling progress

To view the upgrade progress of the cluster, run the following command:

{{< copyable "shell-regular" >}}

```bash
watch kubectl -n ${namespace} get pod -o wide
```

When all Pods are rebuilt and in the `Running` state, the vertical scaling is completed.

> **Note:**
>
> - If the resource's `requests` field is modified during the vertical scaling process, and if PD, TiKV, and TiFlash use `Local PV`, they will be scheduled back to the original node after the upgrade. At this time, if the original node does not have enough resources, the Pod ends up staying in the `Pending` status and thus impacts the service.
> - TiDB is a horizontally scalable database, so it is recommended to take advantage of it simply by adding more nodes rather than upgrading hardware resources like you do with a traditional database.

### Vertical scaling failure

During the vertical scaling operation, Pods might go to the Pending state because of insufficient resources. See [Troubleshoot the Pod in Pending state](deploy-failures.md#the-pod-is-in-the-pending-state) for details.
