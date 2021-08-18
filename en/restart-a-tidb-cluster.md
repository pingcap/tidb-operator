---
title: Restart a TiDB Cluster in Kubernetes
summary: Learn how to restart a TiDB cluster in the Kubernetes cluster.
aliases: ['/docs/tidb-in-kubernetes/dev/restart-a-tidb-cluster/']
---

# Restart a TiDB Cluster in Kubernetes

If you find that the memory leak occurs in a Pod during use, you need to restart the cluster. This document describes how to perform a graceful rolling restart to all the Pods in a component of the TiDB cluster, or gracefully log off a Pod in the TiDB cluster and then restart the Pod using the graceful restart command.

> **Warning:**
>
> It is not recommended to manually remove a Pod in the TiDB cluster without graceful restart in a production environment, because this might lead to some request failures of accessing the TiDB cluster though the `StatefulSet` controller pulls the Pod up again.

## Performing a graceful rolling restart to all Pods in a component

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
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
  tikv:
    baseImage: pingcap/tikv
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
    annotations:
      tidb.pingcap.com/restartedAt: 2020-04-20T12:00
```