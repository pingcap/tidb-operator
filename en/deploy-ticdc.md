---
title: Deploy TiCDC in Kubernetes
summary: Learn how to deploy TiCDC in Kubernetes.
category: how-to
---

# Deploy TiCDC in Kubernetes

[TiCDC](https://pingcap.com/docs/stable/ticdc/ticdc-overview/) is a tool for replicating the incremental data of TiDB. This document describes how to deploy TiCDC in Kubernetes using TiDB Operator.

You can deploy TiCDC when deploying a new TiDB cluster, or add the TiCDC component to an existing TiDB cluster.

## Prerequisites

TiDB Operator is [deployed](deploy-tidb-operator.md).

## Fresh TiCDC deployment

To deploy TiCDC when deploying the TiDB cluster, refer to [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md).

## Add TiCDC component to an existing TiDB cluster

1. Edit TidbCluster Custom Resource:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. Add the TiCDC configuration as follows:

    ```yaml
    spec:
      ticdc:
        baseImage: pingcap/ticdc
        replicas: 3
    ```

3. After the deployment, enter a TiCDC Pod by running `kubectl exec`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec -it ${pod_name} -n ${namespace} sh
    ```

4. [Manage the cluster and data replication tasks](https://pingcap.com/docs/stable/ticdc/manage-ticdc/#use-cdc-cli-to-manage-cluster-status-and-data-replication-task) by using `cdc cli`.

    {{< copyable "shell-regular" >}}

    ```shell
    /cdc cli capture list --pd=${pd_address}:2379
    ```

    ```shell
    [
            {
                    "id": "6d92386a-73fc-43f3-89de-4e337a42b766",
                    "is-owner": true
            },
            {
                    "id": "b293999a-4168-4988-a4f4-35d9589b226b",
                    "is-owner": false
            }
    ]
    ```
