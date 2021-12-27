---
title: Recover the Deleted Cluster
summary: Learn how to recover a TiDB cluster that has been deleted mistakenly.
aliases: ['/docs/tidb-in-kubernetes/dev/recover-deleted-cluster/']
---

# Recover the Deleted Cluster

This document describes how to recover a TiDB cluster that has been deleted mistakenly.

## Recover the cluster managed by TidbCluster

TiDB Operator uses PV (Persistent Volume) and PVC (Persistent Volume Claim) to store persistent data. If you accidentally delete a cluster using `kubectl delete tc`, the PV/PVC objects and data are still retained to ensure data safety.

To recover the deleted cluster, use the `kubectl create` command to create a cluster that has the same name and configuration as the deleted one. In the new cluster, the retained PV/PVC and data are reused.

{{< copyable "shell-regular" >}}

```bash
kubectl -n ${namespace} create -f tidb-cluster.yaml
```

## Recover the cluster managed by Helm

TiDB Operator uses PV (Persistent Volume) and PVC (Persistent Volume Claim) to store persistent data. If you accidentally delete a cluster using `helm uninstall`, the PV/PVC objects and data are still retained to ensure data safety.

To recover the cluster at this time, use the `helm install` command to create a cluster that has the same name and configuration as the deleted one. In the new cluster, the retained PV/PVC and data are reused.

{{< copyable "shell-regular" >}}

```bash
helm install ${release_name} pingcap/tidb-cluster --namespace=${namespace} --version=${chart_version} -f values.yaml
```
