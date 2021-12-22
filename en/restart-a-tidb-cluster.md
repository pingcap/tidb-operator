---
title: Restart a TiDB Cluster in Kubernetes
summary: Learn how to restart a TiDB cluster in the Kubernetes cluster.
aliases: ['/docs/tidb-in-kubernetes/dev/restart-a-tidb-cluster/']
---

# Restart a TiDB Cluster in Kubernetes

If you find that the memory leak occurs in a Pod during use, you need to restart the cluster. This document describes how to perform a graceful rolling restart to all Pods in a TiDB component and how to perform a graceful restart to a single TiKV Pod.

> **Warning:**
>
> It is not recommended to manually remove a Pod in the TiDB cluster without graceful restart in a production environment, because this might lead to some request failures of accessing the TiDB cluster though the `StatefulSet` controller pulls the Pod up again.

## Perform a graceful rolling restart to all Pods in a component

After [Deploying TiDB on general Kubernetes](deploy-on-general-kubernetes.md), modify the cluster configuration by running the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl edit tc ${name} -n ${namespace}
```

Add `tidb.pingcap.com/restartedAt` in the annotation of the `spec` of the TiDB component you want to gracefully rolling restart, and set its value to be the current time.

In the following example, annotations of the `pd`, `tikv`, and `tidb` components are set, which means that all the Pods in these three components will be gracefully rolling restarted. You can set the annotation for a specific component according to your needs.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  version: v5.2.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    ...
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
  tikv:
    ...
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
  tidb:
    ...
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
```

## Perform a graceful restart to a single TiKV Pod

Starting from v1.2.5, TiDB Operator supports graceful restart for a single TiKV Pod.

To trigger a graceful restart, add an annotation with the `tidb.pingcap.com/evict-leader` key:

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} annotate pod ${tikv_pod_name} tidb.pingcap.com/evict-leader="delete-pod"
```

When the number of TiKV region leaders drops to zero, according to the value of this annotation, TiDB Operator might have different behaviors:

- `none`: TiDB Operator does nothing.
- `delete-pod`: TiDB Operator deletes the Pod by taking the following steps:

    1. TiDB Operator calls the PD API and adds evict-leader-scheduler for the TiKV store.
    2. When the number of TiKV region leaders drops to zero, TiDB Operator deletes the Pod and recreates it.
    3. When the new Pod becomes ready, remove the evict-leader-scheduler for the TiKV store by calling the PD API.
