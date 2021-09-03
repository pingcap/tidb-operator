---
title: Perform a Rolling Update to a TiDB Cluster in Kubernetes
summary: Learn how to perform a rolling update to a TiDB cluster in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/upgrade-a-tidb-cluster/']
---

# Perform a Rolling Update to a TiDB Cluster in Kubernetes

When you perform a rolling update to a TiDB cluster in Kubernetes, the Pod is shut down and recreated with the new image or/and configuration serially in the order of PD, TiKV, TiDB. Under the highly available deployment topology (minimum requirements: PD \* 3, TiKV \* 3, TiDB \* 2), performing a rolling update to PD and TiKV servers does not impact the running clients.

+ For the clients that can retry stale connections, performing a rolling update to TiDB servers neither impacts the running clients.
+ For the clients that **can not** retry stale connections, performing a rolling update to TiDB servers will close the client connections and cause the request to fail. For this situation, it is recommended to add a function for the clients to retry, or to perform a rolling update to TiDB servers in idle time.

## Upgrade using TidbCluster CR

If the TiDB cluster is deployed directly using TidbCluster CR, or deployed using Helm but switched to management by TidbCluster CR, you can upgrade the TiDB cluster by the following steps.

### Upgrade the version of TiDB using TidbCluster CR

1. Modify the image configurations of all components in TidbCluster CR.

    Usually, components in a cluster are in the same version. You can upgrade the TiDB cluster simply by modifying `spec.version`. If you need to use different versions for different components, configure `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`.

    > **Note:**
    >
    > If you want to upgrade to Enterprise Edition, edit the `db.yaml` file to set `spec.<tidb/pd/tikv/tiflash/ticdc/pump>.baseImage` to the enterprise image (`pingcap/<tidb/pd/tikv/tiflash/ticdc/tidb-binlog>-enterprise`).
    >
    > For example, change `spec.pd.baseImage` from `pingcap/pd` to `pingcap/pd-enterprise`.

    The `version` field has following formats:

    - `spec.version`: the format is `imageTag`, such as `v5.2.0`

    - `spec.<pd/tidb/tikv/pump/tiflash/ticdc>.version`: the format is `imageTag`, such as `v3.1.0`

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. Check the upgrade progress:

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

    After all the Pods finish rebuilding and become `Running`, the upgrade is completed.

### Force an upgrade of TiDB cluster using TidbCluster CR

If the PD cluster is unavailable due to factors such as PD configuration error, PD image tag error and NodeAffinity, then [scaling the TiDB cluster](scale-a-tidb-cluster.md), [upgrading the TiDB cluster](#upgrade-the-version-of-tidb-using-tidbcluster-cr) and changing the TiDB cluster configuration cannot be done successfully.

In this case, you can use `force-upgrade` to force an upgrade of the cluster to recover cluster functionality.

First, set `annotation` for the cluster:

{{< copyable "shell-regular" >}}

```shell
kubectl annotate --overwrite tc ${cluster_name} -n ${namespace} tidb.pingcap.com/force-upgrade=true
```

Change the related PD configuration to make sure that PD is in a normal state.

> **Note:**
>
> After the PD cluster recovers, you *must* execute the following command to disable the forced upgrade, or an exception may occur in the next upgrade:
>
> {{< copyable "shell-regular" >}}
>
> ```shell
> kubectl annotate tc ${cluster_name} -n ${namespace} tidb.pingcap.com/force-upgrade-
> ```

### Modify TiDB cluster configuration

> **Note:**
>
> - After the cluster is started for the first time, some PD configuration items are persisted in etcd. The persisted configuration in etcd takes precedence over that in PD. Therefore, after the first start, you cannot modify some PD configuration using parameters. You need to dynamically modify the configuration using SQL statements, pd-ctl, or PD server API. Currently, among all the configuration items listed in [Modify PD configuration online](https://docs.pingcap.com/tidb/stable/dynamic-config#modify-pd-configuration-online), except `log.level`, all the other configuration items cannot be modified using parameters after the first start.
> If you modify configuration items by [dynamically modifying configuration online](https://docs.pingcap.com/tidb/stable/dynamic-config), after rolling upgrade, the configuration might be overwritten by CR.

1. Refer to [Configure TiDB components](configure-a-tidb-cluster.md#configure-tidb-components) to modify the component configuration in the `TidbCluster` CR of the cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. View the updating progress after the configuration is modified:

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

    After all the Pods are recreated and are in the `Running` state, the configuration is successfully modified.

> **Note:**
>
> By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).
