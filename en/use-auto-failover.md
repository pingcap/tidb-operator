---
title: Automatic Failover
summary: Learn the automatic failover policies of TiDB cluster components on Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/use-auto-failover/']
---

# Automatic Failover

Automatic failover means that when a node in the TiDB cluster fails, TiDB Operator automatically adds a new one to ensure the high availability of the cluster. It works similarly with the `Deployment` behavior in Kubernetes.

TiDB Operator manages Pods based on `StatefulSet`, which does not automatically create a new node to replace the original node when a Pod goes down. For this reason, the automatic failover feature is added to TiDB Operator, which expands the behavior of `StatefulSet`.

## Configure automatic failover

The automatic failover feature is enabled by default in TiDB Operator. 

> **Note:**
>
> If there are not enough resources in the cluster for TiDB Operator to create new nodes, the automatic failover feature will not take effect.

### Disable Automatic Failover

You can disable it by setting `controllerManager.autoFailover` to `false` in the `charts/tidb-operator/values.yaml` file when deploying TiDB Operator. For example:

```yaml
controllerManager:
 serviceAccount: tidb-controller-manager
 logLevel: 2
 replicas: 1
 resources:
   limits:
     cpu: 250m
     memory: 150Mi
   requests:
     cpu: 80m
     memory: 50Mi
 # autoFailover is whether tidb-operator should auto failover when failure occurs
 autoFailover: false
 # pd failover period default(5m)
 pdFailoverPeriod: 5m
 # tikv failover period default(5m)
 tikvFailoverPeriod: 5m
 # tidb failover period default(5m)
 tidbFailoverPeriod: 5m
 # tiflash failover period default(5m)
 tiflashFailoverPeriod: 5m
```

By default, `pdFailoverPeriod`, `tikvFailoverPeriod`, `tiflashFailoverPeriod` and `tidbFailoverPeriod` are set to be 5 minutes, which is the waiting timeout after an instance failure is identified. After this time, TiDB Operator begins the automatic failover process.

## Automatic failover policies

There are three components in a TiDB cluster - PD, TiKV, and TiDB, each of which has its own automatic failover policy. This section gives an in-depth introduction to these policies.

### Failover with PD

Assume that there are 3 nodes in a PD cluster. If a PD node is down for over 5 minutes (configurable by modifying `tidbFailoverPeriod`), TiDB Operator takes this node offline first, and creates a new PD node. At this time, there are 4 nodes in the cluster. If the failed PD node gets back online, TiDB Operator deletes the newly created node and the number of nodes gets back to 3.

### Failover with TiKV

When a TiKV Pod fails, its status turns to `Disconnected`. After 30 minutes (configurable by setting [`max-store-down-time`](https://pingcap.com/docs/stable/pd-configuration-file/#max-store-down-time) to `"30m"` in PD's configuration file), the status becomes `Down`. After waiting for another 5 minutes (configurable by modifying `tikvFailoverPeriod`), if this TiKV Pod is still down, TiDB Operator creates a new TiKV Pod.

If the failed TiKV Pod gets back online, TiDB Operator does not automatically delete the newly created Pod. This is because scaling in the TiKV Pods will trigger data transfer.

If **all** of the failed Pods have recovered, and you want to scale in the newly created Pods, you can follow the procedure below:

Configure `spec.tikv.recoverFailover: true` (Supported since TiDB Operator v1.1.5):

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator will scale in the newly created Pods automatically. When the scaling in is finished, configure `spec.tikv.recoverFailover: false` to avoid the auto-scaling operation when the next failover occurs and recovers.

### Failover with TiDB

The TiDB automatic failover policy works the same way as `Deployment` does in Kubernetes. Assume that there are 3 nodes in a TiDB cluster. If a TiDB node is down for over 5 minutes (configurable by modifying `tidbFailoverPeriod`), TiDB Operator creates a new TiDB node. At this time, there are 4 nodes in the cluster. When the failed TiDB node gets back online, TiDB Operator deletes the newly created node and the number of nodes gets back to 3.

### Failover with TiFlash

When a TiFlash Pod fails, its status turns to `Disconnected`. After 30 minutes (configurable by setting [`max-store-down-time`](https://pingcap.com/docs/stable/pd-configuration-file/#max-store-down-time) to `"30m"` in PD's configuration file), the status becomes `Down`. After waiting for another 5 minutes (configurable by modifying `tiflashFailoverPeriod`), if this TiFlash Pod is still down, TiDB Operator creates a new TiFlash Pod.

If the failed TiFlash Pod gets back online, TiDB Operator does not automatically delete the newly created Pod. This is because scaling in the TiFlash Pods will trigger data transfer.

If **all** of the failed Pods have recovered, and you want to scale in the newly created Pods, you can follow the procedure below:

Configure `spec.tiflash.recoverFailover: true` (Supported since TiDB Operator v1.1.5):

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc -n ${namespace} ${cluster_name}
```

TiDB Operator will scale in the newly created Pods automatically. When the scaling in is finished, configure `spec.tiflash.recoverFailover: false` to avoid the auto-scaling operation when the next failover occurs and recovers.
