---
title: Restart a TiDB Cluster in Kubernetes
summary: Learn how to restart a TiDB cluster in the Kubernetes cluster.
category: how-to
---

# Restart a TiDB Cluster in Kubernetes

If you find that the memory leak occurs in a Pod during use, you need to restart the cluster. This document describes how to perform a graceful rolling restart to all the Pods in a component of the TiDB cluster, or gracefully log off a Pod in the TiDB cluster and then restart the Pod using the graceful restart command.

> **Warning:**
>
> It is not recommended to manually remove a Pod in the TiDB cluster without graceful restart in a production environment, because this might lead to some request failures of accessing the TiDB cluster though the `StatefulSet` controller pulls the Pod up again.

## Performing a graceful rolling restart to all Pods in a component

1. Refer to [Deploy TiDB on general Kubernetes](deploy-on-general-kubernetes.md) and modify the `${cluster_name}/tidb-cluster.yaml` file.

    Add `tidb.pingcap.com/restartedAt` in the annotation of the `spec` of the TiDB component you want to gracefully rolling restart, and set its value to be the current time.

    In the following example, annotations of the `pd`, `tikv`, and `tidb` components are set, which means that all the Pods in these three components will be gracefully rolling restarted. You can set the annotation for a specific component according to your needs.

    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      name: basic
    spec:
      version: v3.0.8
      timezone: UTC
      pvReclaimPolicy: Delete
      pd:
        baseImage: pingcap/pd
        replicas: 3
        requests:
          storage: "1Gi"
        config: {}
        annotations:
          tidb.pingcap.com/restartedAt: "202004201200"
      tikv:
        baseImage: pingcap/tikv
        replicas: 3
        requests:
          storage: "1Gi"
        config: {}
        annotations:
          tidb.pingcap.com/restartedAt: "202004201200"
      tidb:
        baseImage: pingcap/tidb
        replicas: 2
        service:
          type: ClusterIP
        config: {}
        annotations:
          tidb.pingcap.com/restartedAt: "202004201200"
    ```

2. Apply the update:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ${cluster_name} -n ${namespace}
    ```

## Gracefully restart a single Pod of the TiDB component

This section describes how to gracefully restart a single Pod of the component in a TiDB cluster.

### Enable the configurations

To activate the graceful logoff feature, you need to enable some related configurations in TiDB Operator. These configurations are disabled by default. Take the following steps to manually turn them on.

1. Edit the `values.yaml` file.

    Enable the `Operator Webhook` feature:

    ```yaml
    admissionWebhook:
      create: true
    ```

    For more information about `Operator Webhook`, see [Enable Admission Controller in TiDB Operator](enable-admission-webhook.md).

2. Install or update TiDB Operator.

    To install or update TiDB Operator, see [Deploy TiDB Operator in Kubernetes](deploy-tidb-operator.md).

### Use annotate to mark the target Pod

You can use `kubectl annotate` to mark the target Pod component of the TiDB cluster. After marking, the TiDB Operator automatically performs graceful logoff of the Pod and restarts the target Pod. To mark the target Pod, run the following command:

{{< copyable "shell-regular" >}}

```sh
kubectl annotate ${pod_name} -n ${namespace} tidb.pingcap.com/pod-defer-deleting=true
```
