---
title: Automatic Failover
summary: Learn the automatic failover policies of TiDB cluster components on Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/use-auto-failover/']
---

# Automatic failover

TiDB Operator manages the deployment and scaling of Pods based on [`StatefulSet`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/). When some Pods or nodes fail, `StatefulSet` does not support automatically creating new Pods to replace the failed ones. To solve this issue, TiDB Operator supports the automatic failover feature by scaling Pods automatically.

## Configure automatic failover

The automatic failover feature is enabled by default in TiDB Operator. 

When deploying TiDB Operator, you can configure the waiting timeout for failover of the PD, TiKV, TiDB, and TiFlash components in a TiDB cluster in the `charts/tidb-operator/values.yaml` file. An example is as follows:

```yaml
controllerManager:
 ...
 # autoFailover is whether tidb-operator should auto failover when failure occurs
 autoFailover: true
 # pd failover period default(5m)
 pdFailoverPeriod: 5m
 # tikv failover period default(5m)
 tikvFailoverPeriod: 5m
 # tidb failover period default(5m)
 tidbFailoverPeriod: 5m
 # tiflash failover period default(5m)
 tiflashFailoverPeriod: 5m
```

In the example, `pdFailoverPeriod`, `tikvFailoverPeriod`, `tiflashFailoverPeriod` and `tidbFailoverPeriod` indicate the waiting timeout (5 minutes by default) after an instance failure is identified. After the timeout, TiDB Operator starts the automatic failover process.

In addition, when configuring a TiDB cluster, you can specify `spec.${component}.maxFailoverCount` for each component, which is the threshold of the maximum number of Pods that the TiDB Operator can create during automatic failover. For more information, see the [TiDB component configuration documentation](configure-a-tidb-cluster.md#configure-automatic-failover-thresholds-of-pd-tidb-tikv-and-tiflash).

> **Note:**
> 
> If there are not enough resources in the cluster for TiDB Operator to create new Pods, the newly scaled Pods will be in the pending status.

## Automatic failover policies

There are six components in a TiDB cluster: PD, TiKV, TiDB, TiFlash, TiCDC, and Pump. Currently, TiCDC and Pump do not support the automatic failover feature. PD, TiKV, TiDB, and TiFlash have different failover policies. This section gives a detailed introduction to these policies.

### Failover with PD

TiDB Operator collects the health status of PD members via the `pd/health` PD API and records the status in the `.status.pd.members` field of the TidbCluster CR.

Take a PD cluster with 3 Pods as an example. If a Pod fails for more than 5 minutes (`pdFailoverPeriod` is configurable), TiDB Operator automatically does the following operations:

1. TiDB Operator records the Pod information in the `.status.pd.failureMembers` field of TidbCluster CR.
2. TiDB Operator takes the Pod offline: TiDB Operator calls PD API to remove the Pod from the member list, and then deletes the Pod and its PVC. 
3. The StatefulSet controller recreates the Pod, and the recreated Pod joins the cluster as a new member.
4. When calculating the replicas of PD StatefulSet, TiDB Operator takes the deleted `.status.pd.failureMembers` into account, so it will create a new Pod. Then, 4 Pods will exist at the same time.

When all the failed Pods in the cluster recover, TiDB Operator will automatically remove the newly created Pods, and the number of Pods gets back to the original.

> **Note:**
>
> - For each PD cluster, the maximum number of Pods that TiDB Operator can create is `spec.pd.maxFailoverCount` (the default value is `3`). After the threshold is reached, TiDB Operator will not perform failover. 
> - If most members in a PD cluster fail, which makes the PD cluster unavailable, TiDB Operator will not perform failover for the PD cluster.

### Failover with TiDB

TiDB Operator collects the Pod health status by accessing the `/status` interface of each TiDB Pod and records the status in the `.status.tidb.members` field of the TidbCluster CR.

Take a TiDB cluster with 3 Pods as an example. If a Pod fails for more than 5 minutes (`tidbFailoverPeriod` is configurable), TiDB Operator automatically does the following operations:

1. TiDB Operator records the Pod information in the `.status.tidb.failureMembers` field of TidbCluster CR. 
2. When calculating the replicas of TiDB StatefulSet, TiDB Operator takes the `.status.tidb.failureMembers` into account, so it will create a new Pod. Then, 4 Pods will exist at the same time.

When the failed Pod in the cluster recovers, TiDB Operator will automatically remove the newly created Pod, and the number of Pods gets back to 3.

> **Note:**
>
> For each TiDB cluster, the maximum number of Pods that TiDB Operator can create is `spec.tidb.maxFailoverCount` (the default value is `3`). After the threshold is reached, TiDB Operator will not perform failover.

### Failover with TiKV

TiDB Operator collects the TiKV store health status by accessing the PD API and records the status in the `.status.tikv.stores` field in TidbCluster CR.

Take a TiKV cluster with 3 Pods as an example. When a TiKV Pod fails, the store status of the Pod changes to `Disconnected`. By default, after 30 minutes (configurable by changing `max-store-down-time = "30m"` in the `[schedule]` section of `pd.config`), the status changes to `Down`. Then, TiDB Operator automatically does the following operations:

1. Wait for another 5 minutes (configurable by modifying `tikvFailoverPeriod`), if this TiKV Pod is still not recovered, TiDB Operator records the Pod information in the `.status.tikv.failureStores` field of TidbCluster CR.
2. When calculating the replicas of TiKV StatefulSet, TiDB Operator takes the `.status.tikv.failureStores` into account, so it will create a new Pod. Then, 4 Pods will exist at the same time.

When the failed Pod in the cluster recovers, TiDB Operator **DOES NOT** remove the newly created Pod, but continues to keep 4 Pods. This is because scaling in TiKV Pods will trigger data migration, which might affect the cluster performance.

> **Note:**
>
> For each TiKV cluster, the maximum number of Pods that TiDB Operator can create is `spec.tikv.maxFailoverCount` (the default value is `3`). After the threshold is reached, TiDB Operator will not perform failover.

If **all** failed Pods have recovered, and you want to remove the newly created Pods, you can follow the procedure below:

Configure `spec.tikv.recoverFailover: true` (Supported since TiDB Operator v1.1.5):

{{< copyable "shell-regular" >}}

```bash
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator will remove the newly created Pods automatically. When the removal is finished, configure `spec.tikv.recoverFailover: false` to avoid the auto-scaling operation when the next failover occurs and recovers.

### Failover with TiFlash

TiDB Operator collects the TiFlash store health status by accessing the PD API and records the status in the `.status.tiflash.stores` field in TidbCluster CR.

Take a TiFlash cluster with 3 Pods as an example. When a TiFlash Pod fails, the store status of the Pod changes to `Disconnected`. By default, after 30 minutes (configurable by changing `max-store-down-time = "30m"` in the `[schedule]` section of `pd.config`), the status changes to `Down`. Then, TiDB Operator automatically does the following operations:

1. Wait for another 5 minutes (configurable by modifying `tiflashFailoverPeriod`), if the TiFlash Pod is still not recovered, TiDB Operator records the Pod information in the `.status.tiflash.failureStores` field of TidbCluster CR.
2. When calculating the replicas of TiFlash StatefulSet, TiDB Operator takes the `.status.tiflash.failureStores` into account, so it will create a new Pod. Then, 4 Pods will exist at the same time.

When the failed Pod in the cluster recovers, TiDB Operator **DOES NOT** remove the newly created Pod, but continues to keep 4 Pods. This is because scaling in TiFlash Pods will trigger data migration, which might affect the cluster performance.

> **Note:**
>
> For each TiFlash cluster, the maximum number of Pods that TiDB Operator can create is `spec.tiflash.maxFailoverCount` (the default value is `3`). After the threshold is reached, TiDB Operator will not perform failover.

If **all** of the failed Pods have recovered, and you want to remove the newly created Pods, you can follow the procedure below:

Configure `spec.tiflash.recoverFailover: true` (Supported since TiDB Operator v1.1.5):

{{< copyable "shell-regular" >}}

```bash
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator will remove the newly created Pods automatically. When the removal is finished, configure `spec.tiflash.recoverFailover: false` to avoid the auto-scaling operation when the next failover occurs and recovers.

### Disable automatic failover

You can disable the automatic failover feature at the cluster or component level:

- To disable the automatic failover feature at the cluster level, set `controllerManager.autoFailover` to `false` in the `charts/tidb-operator/values.yaml` file when deploying TiDB Operator. An example is as follows:

    ```yaml
    controllerManager:
    ...
    # autoFailover is whether tidb-operator should auto failover when failure occurs
    autoFailover: false
    ```

- To disable the automatic failover feature at the component level, set `spec.${component}.maxFailoverCount` of the target component to `0` in the TidbCluster CR when creating the TiDB cluster.
