---
title: TiDB Operator 灰度升级
summary: 介绍如何灰度升级 TiDB Operator。
---

# TiDB Operator 灰度升级

本文介绍如何灰度升级 TiDB Operator。灰度升级可以控制 TiDB Operator 升级的影响范围，避免由于 TiDB Operator 升级导致对整个 Kubernetes 集群中的所有 TiDB 集群产生不可预知的影响，在确认 TiDB Operator 升级的影响或者确认 TiDB Operator 新版本能正常稳定工作后再正常升级 TiDB Operator。

> **注意：**
>
> - 目前仅支持灰度升级 tidb-controller-manager 和 tidb-scheduler，不支持灰度升级 AdvancedStatefulSet controller 和 AdmissionWebhook。
> - v1.1.10 开始支持此项功能，所以当前 TiDB Operator 版本需 >= v1.1.10。

## 相关参数

为了支持灰度升级 TiDB Operator，`tidb-operator` chart 中 `values.yaml` 文件里面添加了一些参数，可以参考[文档](deploy-multiple-tidb-operator.md#相关参数)。

## 灰度升级 TiDB Operator

1. 为当前 TiDB Operator 配置 selector。

    参考[升级 TiDB Operator 文档](upgrade-tidb-operator.md)，在 values.yaml 中添加如下配置，升级当前 TiDB Operator：

    ```yaml
    controllerManager:
      selector:
      - version!=canary
    ```

    如果之前已经执行过上述步骤，可以直接进入下一步。

2. 部署灰度 TiDB Operator。

    参考[部署 TiDB Operator 文档](deploy-tidb-operator.md)，在 `values.yaml` 中添加如下配置，在**不同的 namespace** 中（例如 `tidb-admin-canary`）使用**不同的 [Helm Release Name](https://helm.sh/docs/intro/using_helm/#three-big-concepts)**（例如 `helm install tidb-operator-canary ...`）部署灰度 TiDB Operator：

    ```yaml
    controllerManager:
      selector:
      - version=canary
    appendReleaseSuffix: true
    #scheduler:
    #  create: false
    advancedStatefulset:
      create: false
    admissionWebhook:
      create: false
    ```

    > **注意：**
    >
    > * 建议在单独的 namespace 部署新的 TiDB Operator。
    > * `appendReleaseSuffix` 需要设置为 `true`。
    > * 如果不需要灰度升级 tidb-scheduler，可以设置 `scheduler.create: false`。
    > * 如果配置 `scheduler.create: true`，会创建一个名字为 `{{ .scheduler.schedulerName }}-{{.Release.Name}}` 的 scheduler，要使用这个 scheduler，需要配置 TidbCluster CR 中的 `spec.schedulerName` 为这个 scheduler。
    > * 由于不支持灰度升级 AdvancedStatefulSet controller 和 AdmissionWebhook，需要配置 `advancedStatefulset.create: false` 和 `admissionWebhook.create: false`。

3. 如果需要测试 tidb-controller-manager 灰度升级，通过如下命令为某个 TiDB 集群设置 label：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} label tc ${cluster_name} version=canary
    ```

    通过查看已经部署的两个 tidb-controller-manager 的日志可以确认，这个 TiDB 集群已经归灰度 TiDB Operator 管理。

    1. 查看当前 TiDB Operator `tidb-controller-manager` 的日志:

        ```shell
        kubectl -n tidb-admin logs tidb-controller-manager-55b887bdc9-lzdwv
        ```

        ```
        I0305 07:52:04.558973       1 tidb_cluster_controller.go:148] TidbCluster has been deleted tidb-cluster-1/basic1
        ```

    2. 查看灰度 TiDB Operator `tidb-controller-manager` 的日志:

        ```shell
        kubectl -n tidb-admin-canary logs tidb-controller-manager-canary-6dcb9bdd95-qf4qr
        ```

        ```
        I0113 03:38:43.859387       1 tidbcluster_control.go:69] TidbCluster: [tidb-cluster-1/basic1] updated successfully
        ```

4. 如果需要测试 tidb-scheduler 灰度升级，通过如下命令为某个 TiDB 集群修改 `spec.schedulerName` 为 `tidb-scheduler-canary`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} edit tc ${cluster_name}
    ```

    修改后，集群内各组件会滚动升级，可以通过查看灰度 TiDB Operator `tidb-scheduler` 的日志确认集群已经使用灰度 `tidb-scheduler`：

    ```shell
    kubectl -n tidb-admin-canary logs tidb-scheduler-canary-7f7b6c7c6-j5p2j -c tidb-scheduler
    ```

5. 灰度测试完成后，可以将 3，4 步骤中的修改改回去，重新使用当前 TiDB Operator 管理。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} label tc ${cluster_name} version-
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} edit tc ${cluster_name}
    ```

6. 删除灰度 TiDB Operator。

    ```shell
    helm -n tidb-admin-canary uninstall ${release_name}
    ```

7. 参考[升级 TiDB Operator 文档](upgrade-tidb-operator.md)正常升级当前 TiDB Operator。
