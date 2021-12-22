---
title: Perform a Canary Upgrade on TiDB Operator
summary: Learn how to perform a canary upgrade on TiDB Operator in Kubernetes.
---

# Perform a Canary Upgrade on TiDB Operator

This document describes how to perform a canary upgrade on TiDB Operator. Using canary upgrades, you can prevent normal TiDB Operator upgrade from causing an unexpected impact on all the TiDB clusters in Kubernetes. After you confirm the impact of TiDB Operator upgrade or that the upgraded TiDB Operator works stably, you can normally upgrade TiDB Operator.

When you use TiDB Operator, `tidb-scheduler` is not mandatory. Refer to [tidb-scheduler and default-scheduler](tidb-scheduler.md#tidb-scheduler-and-default-scheduler) to confirm whether you need to deploy `tidb-scheduler`.

> **Note:**
>
> - You can perform a canary upgrade only on `tidb-controller-manager` and `tidb-scheduler`. AdvancedStatefulSet controller and `tidb-admission-webhook` do not support the canary upgrade.
> - Canary upgrade is supported since v1.1.10. The version of your current TiDB Operator should be >= v1.1.10.

## Related parameters

To support canary upgrade, some parameters are added to the `values.yaml` file in the `tidb-operator` chart. See [Related parameters](deploy-multiple-tidb-operator.md#related-parameters) for details.

## Canary upgrade process

1. Configure selector for the current TiDB Operator:

    Refer to [Upgrade TiDB Operator](upgrade-tidb-operator.md). Add the following configuration in the `values.yaml` file, and upgrade TiDB Operator:

    ```yaml
    controllerManager:
      selector:
      - version!=canary
    ```

    If you have already performed the step above, skip to Step 2.

2. Deploy the canary TiDB Operator:

    Refer to [Deploy TiDB Operator](deploy-tidb-operator.md). Add the following configuration in the `values.yaml` file, and deploy the canary TiDB Operator in **a different namespace** (such as `tidb-admin-canary`) with a **different [Helm Release Name](https://helm.sh/docs/intro/using_helm/#three-big-concepts)** (such as `helm install tidb-operator-canary ...`):

    ```yaml
    controllerManager:
      selector:
      - version=canary
    appendReleaseSuffix: true
    #scheduler:
    # If you do not need tidb-scheduler, set this value to false.
    #  create: false
    advancedStatefulset:
      create: false
    admissionWebhook:
      create: false
    ```

    > **Note:**
    >
    > * It is recommended to deploy the new TiDB Operator in a separate namespace.
    > * Set `appendReleaseSuffix` to `true`.
    > * If you do not need to perform a canary upgrade on `tidb-scheduler`, configure `scheduler.create: false`.
    > * If you configure `scheduler.create: true`, a scheduler named `{{ .scheduler.schedulerName }}-{{.Release.Name}}` will be created. To use this scheduler, configure `spec.schedulerName` in the `TidbCluster` CR to the name of this scheduler.
    > * You need to set `advancedStatefulset.create: false` and `admissionWebhook.create: false`, because AdvancedStatefulSet controller and `tidb-admission-webhook` do not support the canary upgrade.

3. To test the canary upgrade of `tidb-controller-manager`, set labels for a TiDB cluster by running the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} label tc ${cluster_name} version=canary
    ```

    Check the logs of the two deployed `tidb-controller-manager`s, and you can see this TiDB cluster is now managed by the canary TiDB Operator:

    1. View the log of `tidb-controller-manager` of the current TiDB Operator:

        ```shell
        kubectl -n tidb-admin logs tidb-controller-manager-55b887bdc9-lzdwv
        ```

        ```
        I0305 07:52:04.558973       1 tidb_cluster_controller.go:148] TidbCluster has been deleted tidb-cluster-1/basic1
        ```

    2. View the log of `tidb-controller-manager` of the canary TiDB Operator:

        ```shell
        kubectl -n tidb-admin-canary logs tidb-controller-manager-canary-6dcb9bdd95-qf4qr
        ```

        ```
        I0113 03:38:43.859387       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-1/basic1] updated successfully
        ```

4. To test the canary upgrade of `tidb-scheduler`, modify `spec.schedulerName` of some TiDB cluster to `tidb-scheduler-canary` by running the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} edit tc ${cluster_name}
    ```

    After the modification, all components in the cluster will be rolling updated.

    Check the logs of `tidb-scheduler` of the canary TiDB Operator, and you can see this TiDB cluster is now using the canary `tidb-scheduler`:

    ```shell
    kubectl -n tidb-admin-canary logs tidb-scheduler-canary-7f7b6c7c6-j5p2j -c tidb-scheduler
    ```

5. After the tests, you can revert the changes in Step 3 and Step 4 so that the TiDB cluster is again managed by the current TiDB Operator.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} label tc ${cluster_name} version-
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} edit tc ${cluster_name}
    ```

6. Delete the canary TiDB Operator:

    ```shell
    helm -n tidb-admin-canary uninstall ${release_name}
    ```

7. Refer to [Upgrade TiDB Operator](upgrade-tidb-operator.md) and upgrade the current TiDB Operator normally.
