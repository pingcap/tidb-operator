---
title: Perform a Rolling Update to a TiDB Cluster in Kubernetes
summary: Learn how to perform a rolling update to a TiDB cluster in Kubernetes.
category: how-to
---

# Perform a Rolling Update to a TiDB Cluster in Kubernetes

When you perform a rolling update to a TiDB cluster in Kubernetes, the Pod is shut down and recreated with the new image or/and configuration serially in the order of PD, TiKV, TiDB. Under the highly available deployment topology (minimum requirements: PD \* 3, TiKV \* 3, TiDB \* 2), performing a rolling update to PD and TiKV servers does not impact the running clients.

+ For the clients that can retry stale connections, performing a rolling update to TiDB servers neither impacts the running clients.
+ For the clients that **can not** retry stale connections, performing a rolling update to TiDB servers will close the client connections and cause the request to fail. For this situation, it is recommended to add a function for the clients to retry, or to perform a rolling update to TiDB servers in idle time.

## Upgrade using TidbCluster CR

If the TiDB cluster is deployed directly using TidbCluster CR, or deployed using Helm but switched to management by TidbCluster CR, you can upgrade the TiDB cluster by the following steps.

### Upgrade the version of TiDB using TidbCluster CR

1. Modify the image configurations of all components in TidbCluster CR.

    Note that TidbCluster CR has multiple parameters for the image configuration:

    - `spec.version`: the format is `imageTag`, such as `v3.1.0`

    - `spec.<pd/tidb/tikv/pump>.baseImage`: the format is `imageName`, such as `pingcap/tidb`

    - `spec.<pd/tidb/tikv/pump>.version`: the format is `imageTag`, such as `v3.1.0`

    - `spec.<pd/tidb/tikv/pump>.image`: the format is `imageName:imageTag`, such as `pingcap/tidb:v3.1.0`

    The priority for acquiring image configuration is as follows:

    `spec.<pd/tidb/tikv/pump>.baseImage` + `spec.<pd/tidb/tikv/pump>.version` > `spec.<pd/tidb/tikv/pump>.baseImage` + `spec.version` > `spec.<pd/tidb/tikv/pump>.image`

    Usually, components in a cluster are in the same version. It is recommended to configure `spec.<pd/tidb/tikv/pump>.baseImage` and `spec.version`. Therefore, you can upgrade the TiDB cluster simply by modifying `spec.version`.

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

### Update the configuration of TiDB cluster using TidbCluster CR

The modified configuration is not automatically applied to the TiDB cluster by default. The new configuration file is loaded only when the Pod restarts.

It is recommended that you set `spec.configUpdateStrategy` to `RollingUpdate` to enable automatic update of configurations. This way, every time the configuration is updated, all components are rolling updated automatically, and the modified configuration is applied to the cluster.

1. Set `spec.configUpdateStrategy` to `RollingUpdate`.

2. Modify the configuration items of the cluster, as described in [Configure a TiDB Cluster](configure-a-tidb-cluster.md)

3. View the update progress:

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

    After all the Pods finish rebuilding and become `Running`, the upgrade is completed.

### Force an upgrade of TiDB cluster using TidbCluster CR

If the PD cluster is unavailable due to factors such as PD configuration error, PD image tag error and NodeAffinity, then [scaling the TiDB cluster](scale-a-tidb-cluster.md), [upgrading the TiDB cluster](#upgrade-the-version-of-tidb-cluster) and [changing the TiDB cluster configuration](#change-the-configuration-of-tidb-cluster) cannot be done successfully.

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

## Upgrade using Helm

If you continue to manage your cluster using Helm, refer to the following steps to upgrade the TiDB cluster.

### Upgrade the version of TiDB using Helm

1. Change the `image` of PD, TiKV and TiDB to different image versions in the `values.yaml` file.

2. Run the `helm upgrade` command:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade ${release_name} pingcap/tidb-cluster -f values.yaml --version=${version}
    ```

3. Check the upgrade progress:

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

### Change the configuration of TiDB cluster using Helm

By default, changes to the configuration files are applied to the TiDB cluster automatically through a rolling update. You can disable this feature by setting the `enableConfigMapRollout` variable to `false` in the `values.yaml` file, if so, the change of configuration will be loaded until the server being restarted.

You can change the configuration of TiDB cluster through the following steps:

1. Make sure the `enableConfigMapRollout` feature is not disabled explicitly in the `values.yaml` file.
2. Change the configurations in the `values.yaml` file as needed.
3. Run the `helm upgrade` command:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade ${release_name} pingcap/tidb-cluster -f values.yaml --version=${version}
    ```

4. Check the upgrade process:

    {{< copyable "shell-regular" >}}

    ```shell
    watch kubectl -n ${namespace} get pod -o wide
    ```

> **Note:**
>
> Changing the `enableConfigMapRollout` variable against a running cluster will trigger a rolling update of PD, TiKV, TiDB servers even if there's no change to the configuration.

### Force an upgrade of TiDB cluster using Helm

If the PD cluster is unavailable due to factors such as PD configuration error, PD image tag error and NodeAffinity, then [scaling the TiDB cluster](scale-a-tidb-cluster.md), [upgrading the TiDB cluster](#upgrade-the-version-of-tidb-cluster) and [changing the TiDB cluster configuration](#change-the-configuration-of-tidb-cluster) cannot be done successfully.

In this case, you can use `force-upgrade` (the version of TiDB Operator must be later than v1.0.0-beta.3) to force an upgrade of the cluster to recover cluster functionality.

First, set `annotation` for the cluster:

{{< copyable "shell-regular" >}}

```shell
kubectl annotate --overwrite tc ${release_name} -n ${namespace} tidb.pingcap.com/force-upgrade=true
```

Then execute the `helm upgrade` command to continue your interrupted operation:

{{< copyable "shell-regular" >}}

```shell
helm upgrade ${release_name} pingcap/tidb-cluster -f values.yaml --version=${chart_version}
```

> **Warning:**
>
> After the PD cluster recovers, you *must* execute the following command to disable the forced upgrade, or an exception may occur in the next upgrade:
>
> {{< copyable "shell-regular" >}}
>
> ```shell
> kubectl annotate tc ${release_name} -n ${namespace} tidb.pingcap.com/force-upgrade-
> ```
