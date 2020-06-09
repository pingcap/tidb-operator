---
title: Deploy TiDB on General Kubernetes
summary: Learn how to deploy a TiDB cluster on general Kubernetes.
category: how-to
---

# Deploy TiDB on General Kubernetes

This document describes how to deploy a TiDB cluster on general Kubernetes.

## Prerequisites

- Complete [deploying TiDB Operator](deploy-tidb-operator.md).

## Deploy TiDB cluster

Before you deploy TiDB cluster, you should configure the TiDB cluster. Refer to [Configure a TiDB Cluster in Kubernetes](configure-a-tidb-cluster.md)

After you configure TiDB cluster, deploy the TiDB cluster by the following steps:

1. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create namespace ${namespace}
    ```

    > **Note:**
    >
    > A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. You can give it a name that is easy to memorize, such as the same name as `cluster_name`.

2. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ${cluster_name} -n ${namespace}
    ```

    > **Note:**
    >
    > It is recommended to organize configurations for a TiDB cluster under a directory of `cluster_name` and save it as `${cluster_name}/tidb-cluster.yaml`.

3. View the Pod status:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${cluster_name}
    ```

You can use TiDB Operator to deploy and manage multiple TiDB clusters in a single Kubernetes cluster by repeating the above procedure and replacing `cluster_name` with a different name.

Different clusters can be in the same or different `namespace`, which is based on your actual needs.

## Initialize TiDB cluster

If you want to initialize your cluster after deployment, refer to [Initialize a TiDB Cluster in Kubernetes](initialize-a-cluster.md).
