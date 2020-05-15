---
Title: Scale TiDB in Kubernetes
summary: Learn how to horizontally and vertically scale up and down a TiDB cluster in Kubernetes.
Category: how-to
---

# Scale TiDB in Kubernetes

This document introduces how to horizontally and vertically scale a TiDB cluster in Kubernetes.

## Horizontal scaling

Horizontally scaling TiDB means that you scale TiDB out or in by adding or remove nodes in your pool of resources. When you scale a TiDB cluster, PD, TiKV, and TiDB are scaled out or in sequentially according to the values of their replicas. Scaling out operations add nodes based on the node ID in ascending order, while scaling in operations remove nodes based on the node ID in descending order.

Currently, the TiDB cluster supports management by Helm or by TidbCluster Custom Resource (CR). You can choose the scaling method based on the management method of your TiDB cluster.

### Horizontal scaling operations (CR)

Modify `spec.pd.replicas`, `spec.tidb.replicas`, and `spec.tikv.replicas` in the `TidbCluster` object of the cluster to a desired value using kubectl.

If TiFlash is deployed in the cluster, you can scale in and out TiFlash by modifying `spec.tiflash.replicas`.

### Horizontal scaling operations (Helm)

To perform a horizontal scaling operation, take the following steps:

1. Modify `pd.replicas`, `tidb.replicas`, `tikv.replicas` in the `value.yaml` file of the cluster to a desired value.

2. Run the `helm upgrade` command to scale out or in:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade ${release_name} pingcap/tidb-cluster -f values.yaml --version=${version}
    ```

### View the scaling status

To view the scaling status of the cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
watch kubectl -n ${namespace} get pod -o wide
```

When the number of Pods for all components reaches the preset value and all components go to the `Running` state, the horizontal scaling is completed.

> **Note:**
>
> - The PD, TiKV and TiFlash components do not trigger scaling in and out operations during the rolling update.
> - When the TiKV component scales in, TiDB Operator calls the PD interface to mark the corresponding TiKV instance as offline, and then migrates the data on it to other TiKV nodes. During the data migration, the TiKV Pod is still in the `Running` state, and the corresponding Pod is deleted only after the data migration is completed. The time consumed by scaling in depends on the amount of data on the TiKV instance to be scaled in. You can check whether TiKV is in the `Offline` state by running `kubectl get tidbcluster -n ${namespace} ${release_name} -o json | jq '.status.tikv.stores'`.
> - The TiKV component does not support scale out while a scale-in operation is in progress. Forcing a scale-out operation might cause anomalies in the cluster. If an anomaly already happens, refer to [TiKV Store is in Tombstone status abnormally](troubleshoot.md#tikv-store-is-in-tombstone-status-abnormally) to fix it.
> - The TiFlash component has the same scale-in logic as TiKV.
> - When the PD, TiKV, and TiFlash components scale in, the PVC of the deleted node is retained during the scaling in process. Because the PV's reclaim policy is changed to `Retain`, the data can still be retrieved even if the PVC is deleted.

## Vertical scaling

Vertically scaling TiDB means that you scale TiDB up or down by increasing or decreasing the limit of resources on the node. Vertically scaling is essentially the rolling update of the nodes.

Currently, the TiDB cluster supports management by Helm or by TidbCluster Custom Resource (CR). You can choose the scaling method based on the management method of your TiDB cluster.

### Vertical scaling operations (CR)

Modify `spec.pd.resources`, `spec.tikv.resources`, and `spec.tidb.resources` in the `TidbCluster` object that corresponds to the cluster to the desired values using kubectl.

If TiFlash is deployed in the cluster, you can scale up and down TiFlash by modifying `spec.tiflash.resources`.

### Vertical scaling operations (Helm)

To perform a vertical scaling operation:

1. Modify `tidb.resources`, `tikv.resources`, `pd.resources` in the `values.yaml` file to a desired value.

2. Run the `helm upgrade` command to upgrade:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade ${release_name} pingcap/tidb-cluster -f values.yaml --version=${version}
    ```

### View the upgrade progress

To view the upgrade progress of the cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
watch kubectl -n ${namespace} get pod -o wide
```

When all Pods are rebuilt and in the `Running` state, the vertical scaling is completed.

> **Note:**
>
> - If the resource's `requests` field is modified during the vertical scaling process, and if PD, TiKV, and TiFlash use `Local PV`, they will be scheduled back to the original node after the upgrade. At this time, if the original node does not have enough resources, the Pod ends up staying in the `Pending` status and thus impacts the service.
> - TiDB is a horizontally scalable database, so it is recommended to take advantage of it simply by adding more nodes rather than upgrading hardware resources like you do with a traditional database.
