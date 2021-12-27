---
title: 部署多套 TiDB Operator 分别管理不同的 TiDB 集群
summary: 介绍如何部署多套 TiDB Operator 分别管理不同的 TiDB 集群。
aliases: ['/zh/tidb-in-kubernetes/dev/canary-deployment-tidb-operator/']
---

# 部署多套 TiDB Operator 分别管理不同的 TiDB 集群

本文介绍如何部署多套 TiDB Operator，分别管理不同的 TiDB 集群。

在使用 TiDB Operator 时，`tidb-scheduler` 并不是必须使用。你可以参考 [tidb-scheduler 与 default-scheduler](tidb-scheduler.md#tidb-scheduler-与-default-scheduler)，确认是否需要部署 `tidb-scheduler`。

> **注意：**
>
> - 目前仅支持部署多套 tidb-controller-manager 和 tidb-scheduler，不支持部署多套 AdvancedStatefulSet controller 和 AdmissionWebhook。
> - 如果部署了多套 TiDB Operator，有的开启了 [`Advanced StatefulSet`](advanced-statefulset.md)，有的没有开启，那么同一个 TidbCluster Custom Resource (CR) 不能在这些 TiDB Operator 之间切换。
> - v1.1.10 开始支持此项功能。

## 相关参数

为了支持部署多套 TiDB Operator，`tidb-operator` chart 中 `values.yaml` 文件里面添加了以下参数。

- `appendReleaseSuffix`

    如果配置为 `true`，部署时会自动为 `tidb-controller-manager` 和 `tidb-scheduler` 相关的资源名称添加后缀 `-{{ .Release.Name }}`，例如，通过 `helm install canary pingcap/tidb-operator ...` 命令部署的 `tidb-controller-manager` deployment 名称为：`tidb-controller-manager-canary`，如果要部署多套 TiDB Operator 需要开启此参数。

    默认值：`false`。

- `controllerManager.create`

    控制是否创建 `tidb-controller-manager`。

    默认值：`true`。

- `controllerManager.selector`

    配置 `tidb-controller-manager` 的 `-selector` 参数，用于根据 CR 的 label 筛选 `tidb-controller-manager` 控制的 CR，多个 selector 之间为 `and` 关系。

    默认值：`[]`，控制所有 CR。

    示例：

    ```yaml
    selector:
    - canary-release=v1
    - k1==v1
    - k2!=v2
    ```

- `scheduler.create`

    控制是否创建 `tidb-scheduler`。

    默认值：`true`。

## 部署多套 TiDB Operator 分别控制不同 TiDB 集群

1. 部署第一套 TiDB Operator。

    参考[部署 TiDB Operator 文档](deploy-tidb-operator.md)，在 values.yaml 中添加如下配置，部署第一套 TiDB Operator：

    ```yaml
    controllerManager:
      selector:
      - user=dev
    ```

2. 部署 TiDB 集群。

    1. 参考[在 Kubernetes 中配置 TiDB 集群](configure-a-tidb-cluster.md)配置 TidbCluster CR，并配置 `labels` 匹配上一步中为 `tidb-controller-manager` 配置的 `selector`，例如：

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

        如果创建 TiDB 集群时没有设置 label，也可以通过如下命令设置：

        {{< copyable "shell-regular" >}}

        ```bash
        kubectl -n ${namespace} label tidbcluster ${cluster_name} user=dev
        ```

    2. 参考[在 Kubernetes 中部署 TiDB 集群](deploy-on-general-kubernetes.md)部署 TiDB 集群，并确认集群各组件正常启动。

3. 部署第二套 TiDB Operator。

    参考[部署 TiDB Operator 文档](deploy-tidb-operator.md)，在 `values.yaml` 中添加如下配置，在**不同的 namespace** 中（例如 `tidb-admin-qa`）使用**不同的 [Helm Release Name](https://helm.sh/docs/intro/using_helm/#three-big-concepts)**（例如 `helm install tidb-operator-qa ...`）部署第二套 TiDB Operator (没有部署 `tidb-scheduler`)：

    ```yaml
    controllerManager:
      selector:
      - user=qa
    appendReleaseSuffix: true
    scheduler:
      # 如果你不需要 `tidb-scheduler`，将这个值设置为 false
      create: false
    advancedStatefulset:
      create: false
    admissionWebhook:
      create: false
    ```

    > **注意：**
    >
    > * 建议在单独的 namespace 部署新的 TiDB Operator。
    > * `appendReleaseSuffix` 需要设置为 `true`。
    > * 如果配置 `scheduler.create: true`，会创建一个名字为 `{{ .scheduler.schedulerName }}-{{.Release.Name}}` 的 scheduler，要使用这个 scheduler，需要配置 TidbCluster CR 中的 `spec.schedulerName` 为这个 scheduler。
    > * 由于不支持部署多套 AdvancedStatefulSet controller 和 AdmissionWebhook，需要配置 `advancedStatefulset.create: false` 和 `admissionWebhook.create: false`。

4. 部署 TiDB 集群。

    1. 参考[在 Kubernetes 中配置 TiDB 集群](configure-a-tidb-cluster.md)配置 TidbCluster CR，并配置 `labels` 匹配上一步中为 `tidb-controller-manager` 配置的 `selector`，例如：

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

        如果创建 TiDB 集群时没有设置 label，也可以通过如下命令设置：

        {{< copyable "shell-regular" >}}

        ```bash
        kubectl -n ${namespace} label tidbcluster ${cluster_name} user=qa
        ```

    2. 参考[在 Kubernetes 中部署 TiDB 集群](deploy-on-general-kubernetes.md)部署 TiDB 集群，并确认集群各组件正常启动。

5. 查看两套 TiDB Operator 的日志，确认两套 TiDB Operator 分别管理各自匹配 selector 的 TiDB 集群。

    示例：

    查看第一套 TiDB Operator `tidb-controller-manager` 的日志:

    ```bash
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

    查看第二套 TiDB Operator `tidb-controller-manager` 的日志:

    ```bash
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

    通过对比两套 TiDB Operator tidb-controller-manager 日志，第一套 TiDB Operator 仅管理 `tidb-cluster-1/basic1` 集群，第二套 TiDB Operator 仅管理 `tidb-cluster-2/basic2` 集群。
