---
title: Deploy Multiple Sets of TiDB Operator
summary: Learn how to deploy multiple sets of TiDB Operator to manage different TiDB clusters.
---

# Deploy Multiple Sets of TiDB Operator

This document describes how to deploy multiple sets of TiDB Operator to manage different TiDB clusters.

When you use TiDB Operator, `tidb-scheduler` is not mandatory. Refer to [tidb-scheduler and default-scheduler](tidb-scheduler.md#tidb-scheduler-and-default-scheduler) to confirm whether you need to deploy `tidb-scheduler`.

> **Note:**
>
> - Currently, you can only deploy multiple sets of `tidb-controller-manager` and `tidb-scheduler`. Deploying multiple sets of AdvancedStatefulSet controller and `tidb-admission-webhook` is not supported.
> - If you have deployed multiple sets of TiDB Operator and only some of them enable [Advanced StatefulSet](advanced-statefulset.md), the same TidbCluster Custom Resource (CR) cannot be switched among these TiDB Operator.
> - This feature is supported since v1.1.10.

## Related parameters

To support deploying multiple sets of TiDB Operator, the following parameters are added to the `values.yaml` file in the `tidb-operator` chart:

- `appendReleaseSuffix`

    If this parameter is set to `true`, when you deploy TiDB Operator, the Helm chart automatically adds a suffix (`-{{ .Release.Name }}`) to the name of resources related to `tidb-controller-manager` and `tidb-scheduler`.

    For example, if you execute `helm install canary pingcap/tidb-operator ...`, the name of the `tidb-controller-manager` deployment is `tidb-controller-manager-canary`.

    If you need to deploy multiple sets of TiDB Operator, set this parameter to `true`.

    Default value: `false`.

- `controllerManager.create`

    Controls whether to create `tidb-controller-manager`.

    Default value: `true`.

- `controllerManager.selector`

    Sets the `-selector` parameter for `tidb-controller-manager`. The parameter is used to filter the CRs controlled by `tidb-controller-manager` according to the CR labels. If multiple selectors exist, the selectors are in `and` relationship.

    Default value: `[]` (`tidb-controller-manager` controls all CRs).

    Example:

    ```yaml
    selector:
    - canary-release=v1
    - k1==v1
    - k2!=v2
    ```

- `scheduler.create`

    Controls whether to create `tidb-scheduler`.

    Default value: `true`.

## Deploy

1. Deploy the first TiDB Operator.

    Refer to [Deploy TiDB Operator](deploy-tidb-operator.md) to deploy the first TiDB Operator. Add the following configuration in the `values.yaml`:

    ```yaml
    controllerManager:
      selector:
      - user=dev
    ```

2. Deploy the TiDB cluster.

    1. Refer to [Configure the TiDB Cluster](configure-a-tidb-cluster.md) to configure the TidbCluster CR, and configure `labels` to match the `selector` set in the last step. For example:

        ```yaml
        apiVersion: pingcap.com/v1alpha1
        kind: TidbCluster
        metadata:
          name: basic1
          labels:
            user: dev
        spec:
          ...
        ```

        If `labels` is not set when you deploy the TiDB cluster, you can configure `labels` by running the following command:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl -n ${namespace} label tidbcluster ${cluster_name} user=dev
        ```

    2. Refer to [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md) to deploy the TiDB cluster. Confirm that each component in the cluster is started normally.

3. Deploy the second TiDB Operator.

    Refer to [Deploy TiDB Operator](deploy-tidb-operator.md) to deploy the second TiDB Operator without `tidb-scheduler`. Add the following configuration in the `values.yaml` file, and deploy the second TiDB Operator (without `tidb-scheduler`) in **a different namespace** (such as `tidb-admin-qa`) with a **different [Helm Release Name](https://helm.sh/docs/intro/using_helm/#three-big-concepts)** (such as `helm install tidb-operator-qa ...`):

    ```yaml
    controllerManager:
      selector:
      - user=qa
    appendReleaseSuffix: true
    scheduler:
      # If you do not need tidb-scheduler, set this value to false.
      create: false
    advancedStatefulset:
      create: false
    admissionWebhook:
      create: false
    ```

    > **Note:**
    >
    > * It is recommended to deploy the new TiDB Operator in a separate namespace.
    > * Set `appendReleaseSuffix` to `true`.
    > * If you configure `scheduler.create: true`, a `tidb-scheduler` named `{{ .scheduler.schedulerName }}-{{.Release.Name}}` is created. To use this `tidb-scheduler`, you need to configure `spec.schedulerName` in the `TidbCluster` CR to the name of this scheduler.
    > * You need to set `advancedStatefulset.create: false` and `admissionWebhook.create: false`, because deploying multiple sets of AdvancedStatefulSet controller and `tidb-admission-webhook` is not supported.

4. Deploy the TiDB cluster.

    1. Refer to [Configure the TiDB Cluster](configure-a-tidb-cluster.md) to configure the TidbCluster CR, and configure `labels` to match the `selector` set in the last step. For example:

        ```yaml
        apiVersion: pingcap.com/v1alpha1
        kind: TidbCluster
        metadata:
          name: basic2
          labels:
            user: qa
        spec:
          ...
        ```

        If `labels` is not set when you deploy the TiDB cluster, you can configure `labels` by running the following command:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl -n ${namespace} label tidbcluster ${cluster_name} user=qa
        ```

    2. Refer to [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md) to deploy the TiDB cluster. Confirm that each component in the cluster is started normally.

5. View the logs of the two sets of TiDB Operator, and confirm that each TiDB Operator manages the TiDB cluster that matches the corresponding selectors.

    For example:

    View the log of `tidb-controller-manager` of the first TiDB Operator:

    ```shell
    kubectl -n tidb-admin logs tidb-controller-manager-55b887bdc9-lzdwv
    ```

    <details>
    <summary>Output</summary>
    <pre><code>
    ...
    I0113 02:50:13.195779       1 main.go:69] FLAG: --selector="user=dev"
    ...
    I0113 02:50:32.409378       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-1/basic1] updated successfully
    I0113 02:50:32.773635       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-1/basic1] updated successfully
    I0113 02:51:00.294241       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-1/basic1] updated successfully
    </code></pre>
    </details>

    View the log of `tidb-controller-manager` of the second TiDB Operator:

    ```shell
    kubectl -n tidb-admin-qa logs tidb-controller-manager-qa-5dfcd7f9-vll4c
    ```

    <details>
    <summary>Output</summary>
    <pre><code>
    ...
    I0113 02:50:13.195779       1 main.go:69] FLAG: --selector="user=qa"
    ...
    I0113 03:38:43.859387       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-2/basic2] updated successfully
    I0113 03:38:45.060028       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-2/basic2] updated successfully
    I0113 03:38:46.261045       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-2/basic2] updated successfully
    </code></pre>
    </details>

    By comparing the logs of the two sets of TiDB Operator, you can confirm that the first TiDB Operator only manages the `tidb-cluster-1/basic1` cluster, and the second TiDB Operator only manages the `tidb-cluster-2/basic2` cluster.
