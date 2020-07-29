---
title: Common Cluster Exceptions of TiDB in Kubernetes
summary: Learn the common exceptions during the operation of TiDB clusters in Kubernetes and their solutions.
---

# Common Cluster Exceptions of TiDB in Kubernetes

This document describes the common exceptions during the operation of TiDB clusters in Kubernetes and their solutions.

## TiKV Store is in `Tombstone` status abnormally

Normally, when a TiKV Pod is in a healthy state (`Running`), the corresponding TiKV store is also in a healthy state (`UP`). However, concurrent scale-in or scale-out on the TiKV component might cause part of TiKV stores to fall into the `Tombstone` state abnormally. When this happens, try the following steps to fix it:

1. View the state of the TiKV store:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get -n ${namespace} tidbcluster ${cluster_name} -ojson | jq '.status.tikv.stores'
    ```

2. View the state of the TiKV Pod:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get -n ${namespace} po -l app.kubernetes.io/component=tikv
    ```

3. Compare the state of the TiKV store with that of the Pod. If the store corresponding to a TiKV Pod is in the `Offline` state, it means the store is being taken offline abnormally. You can use the following commands to cancel the offline process and perform recovery operations:

    1. Open the connection to the PD service:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl port-forward -n ${namespace} svc/${cluster_name}-pd ${local_port}:2379 &>/tmp/portforward-pd.log &
        ```

    2. Bring online the corresponding store:

        {{< copyable "shell-regular" >}}

        ```shell
        curl -X POST http://127.0.0.1:2379/pd/api/v1/store/${store_id}/state?state=Up
        ```

4. If the TiKV store with the latest `lastHeartbeatTime` that corresponds to a Pod is in a `Tombstone` state, it means that the offline process is completed. At this time, you need to re-create the Pod and bind it with a new PV to perform recovery by taking the following steps:

    1. Set the `reclaimPolicy` value of the PV corresponding to the store to `Delete`:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl patch $(kubectl get pv -l app.kubernetes.io/instance=${release_name},tidb.pingcap.com/store-id=${store_id} -o name) -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}
        ```

    2. Remove the PVC used by the Pod:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl delete -n ${namespace} pvc tikv-${pod_name} --wait=false
        ```

    3. Remove the Pod, and wait for it to be re-created:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl delete -n ${namespace} pod ${pod_name}
        ```

    After the Pod is re-created, a new store is registered in the cluster. Then the recovery is completed.

## Persistent connections are abnormally terminated in TiDB

Load balancers often set the idle connection timeout. If no data is sent via a connection for a specific period of time, the load balancer closes the connection.

- If a persistent connection is terminated when you use TiDB, check the middleware program between the client and the TiDB server.
- If the idle timeout is not long enough for your query, try to set the timeout to a larger value. If you cannot reset it, enable the `tcp-keep-alive` option in TiDB.

In Linux, the keepalive probe packet is sent every 7,200 seconds by default. To shorten the interval, configure `sysctls` via the `podSecurityContext` field.

- If `--allowed-unsafe-sysctls=net.*` can be configured for [kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) in the Kubernetes cluster, configure this parameter for kubelet and configure TiDB in the following way:

    {{< copyable "" >}}

    ```yaml
    tidb:
      ...
      podSecurityContext:
        sysctls:
        - name: net.ipv4.tcp_keepalive_time
          value: "300"
    ```

- If `--allowed-unsafe-sysctls=net.*` cannot be configured for kubelet, configure TiDB in the following way:

    {{< copyable "" >}}

    ```yaml
    tidb:
      annotations:
        tidb.pingcap.com/sysctl-init: "true"
      podSecurityContext:
        sysctls:
        - name: net.ipv4.tcp_keepalive_time
          value: "300"
      ...
    ```

> **Note:**
>
> The configuration above requires TiDB Operator 1.1 or later versions.
