---
title: Deploy TiDB in General Kubernetes
summary: Learn how to deploy a TiDB cluster on general Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-on-general-kubernetes/']
---

# Deploy TiDB in General Kubernetes

This document describes how to deploy a TiDB cluster in general Kubernetes.

## Prerequisites

- Meet [prerequisites](prerequisites.md).
- Complete [deploying TiDB Operator](deploy-tidb-operator.md).
- [Configure the TiDB cluster](configure-a-tidb-cluster.md)

## Deploy the TiDB cluster

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

    If the server does not have an external network, you need to download the Docker image used by the TiDB cluster on a machine with Internet access and upload it to the server, and then use `docker load` to install the Docker image on the server.

    To deploy a TiDB cluster, you need the following Docker images (assuming the version of the TiDB cluster is v5.2.0):

    ```shell
    pingcap/pd:v5.2.0
    pingcap/tikv:v5.2.0
    pingcap/tidb:v5.2.0
    pingcap/tidb-binlog:v5.2.0
    pingcap/ticdc:v5.2.0
    pingcap/tiflash:v5.2.0
    pingcap/tidb-monitor-reloader:v1.0.1
    pingcap/tidb-monitor-initializer:v5.2.0
    grafana/grafana:6.0.1
    prom/prometheus:v2.18.1
    busybox:1.26.2
    ```

    Next, download all these images with the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    docker pull pingcap/pd:v5.2.0
    docker pull pingcap/tikv:v5.2.0
    docker pull pingcap/tidb:v5.2.0
    docker pull pingcap/tidb-binlog:v5.2.0
    docker pull pingcap/ticdc:v5.2.0
    docker pull pingcap/tiflash:v5.2.0
    docker pull pingcap/tidb-monitor-reloader:v1.0.1
    docker pull pingcap/tidb-monitor-initializer:v5.2.0
    docker pull grafana/grafana:6.0.1
    docker pull prom/prometheus:v2.18.1
    docker pull busybox:1.26.2

    docker save -o pd-v5.2.0.tar pingcap/pd:v5.2.0
    docker save -o tikv-v5.2.0.tar pingcap/tikv:v5.2.0
    docker save -o tidb-v5.2.0.tar pingcap/tidb:v5.2.0
    docker save -o tidb-binlog-v5.2.0.tar pingcap/tidb-binlog:v5.2.0
    docker save -o ticdc-v5.2.0.tar pingcap/ticdc:v5.2.0
    docker save -o tiflash-v5.2.0.tar pingcap/tiflash:v5.2.0
    docker save -o tidb-monitor-reloader-v1.0.1.tar pingcap/tidb-monitor-reloader:v1.0.1
    docker save -o tidb-monitor-initializer-v5.2.0.tar pingcap/tidb-monitor-initializer:v5.2.0
    docker save -o grafana-6.0.1.tar grafana/grafana:6.0.1
    docker save -o prometheus-v2.18.1.tar prom/prometheus:v2.18.1
    docker save -o busybox-1.26.2.tar busybox:1.26.2
    ```

    Next, upload these Docker images to the server, and execute `docker load` to install these Docker images on the server:

    {{< copyable "shell-regular" >}}

    ```shell
    docker load -i pd-v5.2.0.tar
    docker load -i tikv-v5.2.0.tar
    docker load -i tidb-v5.2.0.tar
    docker load -i tidb-binlog-v5.2.0.tar
    docker load -i ticdc-v5.2.0.tar
    docker load -i tiflash-v5.2.0.tar
    docker load -i tidb-monitor-reloader-v1.0.1.tar
    docker load -i tidb-monitor-initializer-v5.2.0.tar
    docker load -i grafana-6.0.1.tar
    docker load -i prometheus-v2.18.1.tar
    docker load -i busybox-1.26.2.tar
    ```

3. View the Pod status:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${cluster_name}
    ```

You can use TiDB Operator to deploy and manage multiple TiDB clusters in a single Kubernetes cluster by repeating the above procedure and replacing `cluster_name` with a different name.

Different clusters can be in the same or different `namespace`, which is based on your actual needs.

## Initialize the TiDB cluster

If you want to initialize your cluster after deployment, refer to [Initialize a TiDB Cluster in Kubernetes](initialize-a-cluster.md).

> **Note:**
>
> By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).
