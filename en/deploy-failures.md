---
title: Common Deployment Failures of TiDB in Kubernetes
summary: Learn the common deployment failures of TiDB in Kubernetes and their solutions.
---

# Common Deployment Failures of TiDB in Kubernetes

This document describes the common deployment failures of TiDB in Kubernetes and their solutions.

## The Pod is not created normally

After creating a cluster, if the Pod is not created, you can diagnose it using the following commands:

{{< copyable "shell-regular" >}}

```shell
kubectl get tidbclusters -n ${namespace} && \
kubectl describe tidbclusters -n ${namespace} ${cluster_name} && \
kubectl get statefulsets -n ${namespace} && \
kubectl describe statefulsets -n ${namespace} ${cluster_name}-pd
```

After creating a backup/restore task, if the Pod is not created, you can perform a diagnostic operation by executing the following commands:

{{< copyable "shell-regular" >}}

```shell
kubectl get backups -n ${namespace}
kubectl get jobs -n ${namespace}
kubectl describe backups -n ${namespace} ${backup_name}
kubectl describe backupschedules -n ${namespace} ${backupschedule_name}
kubectl describe jobs -n ${namespace} ${backupjob_name}
kubectl describe restores -n ${namespace} ${restore_name}
```

## The Pod is in the Pending state

The Pending state of a Pod is usually caused by conditions of insufficient resources, for example:

- The `StorageClass` of the PVC used by PD, TiKV, TiFlash, Pump, Monitor, Backup, and Restore Pods does not exist or the PV is insufficient.
- No nodes in the Kubernetes cluster can satisfy the CPU or memory resources requested by the Pod
- The number of TiKV or PD replicas and the number of nodes in the cluster do not satisfy the high availability scheduling policy of tidb-scheduler

You can check the specific reason for Pending by using the `kubectl describe pod` command:

{{< copyable "shell-regular" >}}

```shell
kubectl describe po -n ${namespace} ${pod_name}
```

### CPU or memory resources are insufficient

If the CPU or memory resources are insufficient, you can lower the CPU or memory resources requested by the corresponding component for scheduling, or add a new Kubernetes node.

### StorageClass of the PVC does not exist

If the `StorageClass` of the PVC cannot be found, take the following steps:

1. Get the available `StorageClass` in the cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get storageclass
    ```

2. Change `storageClassName` to the name of the `StorageClass` available in the cluster.

3. Update the configuration file:

    * If you want to start the TiDB cluster, execute `kubectl edit tc ${cluster_name} -n ${namespace}` to update the cluster.
    * If you want to run a backup/restore task, first execute `kubectl delete bk ${backup_name} -n ${namespace}` to delete the old backup/restore task, and then execute `kubectl apply -f backup.yaml` to create a new backup/restore task.

4. Delete Statefulset and the corresponding PVCs:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete pvc -n ${namespace} ${pvc_name} && \
    kubectl delete sts -n ${namespace} ${statefulset_name}
    ```

### Insufficient available PVs

If a `StorageClass` exists in the cluster but the available PV is insufficient, you need to add PV resources correspondingly. For Local PV, you can expand it by referring to [Local PV Configuration](configure-storage-class.md#local-pv-configuration).

## The high availability scheduling policy of tidb-scheduler is not satisfied

tidb-scheduler has a high availability scheduling policy for PD and TiKV. For the same TiDB cluster, if there are N replicas of TiKV or PD, then the number of PD Pods that can be scheduled to each node is `M=(N-1)/2` (if N<3, then M=1) at most, and the number of TiKV Pods that can be scheduled to each node is `M=ceil(N/3)` (if N<3, then M=1; `ceil` means rounding up) at most.

If the Pod's state becomes `Pending` because the high availability scheduling policy is not satisfied, you need to add more nodes in the cluster.

## The Pod is in the `CrashLoopBackOff` state

A Pod in the `CrashLoopBackOff` state means that the container in the Pod repeatedly aborts (in the loop of abort - restart by `kubelet` - abort). There are many potential causes of `CrashLoopBackOff`. 

### View the log of the current container

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} logs -f ${pod_name}
```

### View the log when the container was last restarted

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} logs -p ${pod_name}
```

After checking the error messages in the log, you can refer to [Cannot start `tidb-server`](https://pingcap.com/docs/stable/how-to/troubleshoot/cluster-setup#cannot-start-tidb-server), [Cannot start `tikv-server`](https://pingcap.com/docs/stable/how-to/troubleshoot/cluster-setup#cannot-start-tikv-server), and [Cannot start `pd-server`](https://pingcap.com/docs/stable/how-to/troubleshoot/cluster-setup#cannot-start-pd-server) for further troubleshooting.

### "cluster id mismatch"

When the "cluster id mismatch" message appears in the TiKV Pod log, the TiKV Pod might have used old data from other or previous TiKV Pod. If the data on the local disk remain uncleared when you configure local storage in the cluster, or the data is not recycled by the local volume provisioner due to a forced deletion of PV, this error might occur.

If you confirm that the TiKV should join the cluster as a new node and that the data on the PV should be deleted, you can delete the TiKV Pod and the corresponding PVC. The TiKV Pod automatically rebuilds and binds the new PV for use. When configuring local storage, delete local storage on the machine to avoid Kubernetes using old data. In cluster operation and maintenance, manage PV using the local volume provisioner and do not delete it forcibly. You can manage the lifecycle of PV by creating, deleting PVCs, and setting `reclaimPolicy` for the PV.

### `ulimit` is not big enough

TiKV might fail to start when `ulimit` is not big enough. In this case, you can modify the `/etc/security/limits.conf` file of the Kubernetes node to increase the `ulimit`:

```
root soft nofile 1000000
root hard nofile 1000000
root soft core unlimited
root soft stack 10240
```

### Other causes 

If you cannot confirm the cause from the log and `ulimit` is also a normal value, troubleshoot the issue by [using the diagnostic mode](tips.md#use-the-diagnostic-mode).
