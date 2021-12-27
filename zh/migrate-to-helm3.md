---
title: 从 Helm 2 迁移到 Helm 3
summary: 介绍如何将由 Helm 2 管理的组件迁移到由 Helm 3 管理。
---

# 从 Helm 2 迁移到 Helm 3

本文以将 TiDB Operator 由 Helm 2 管理迁移到由 Helm 3 进行管理为例，介绍如何将由 Helm 2 管理的组件迁移到由 Helm 3 管理。其他如 TiDB Lightning 等由 Helm 2 管理的 release 可使用类似的步骤进行迁移。

更多有关如何将 Helm 2 管理的 release 迁移到 Helm 3 的信息，可参考 [Helm 官方文档](https://helm.sh/docs/topics/v2_v3_migration/)。

## 迁移步骤

假设原来由 Helm 2 管理的 TiDB Operator 安装在 `tidb-admin` namespace 下，名称为 `tidb-operator`。同时在 `tidb-cluster` namespace 下部署了名为 `basic` 的 TidbCluster 及名为 `basic` 的 TidbMonitor。

{{< copyable "shell-regular" >}}

```bash
helm list
```

```
NAME            REVISION        UPDATED                         STATUS          CHART                   APP VERSION     NAMESPACE
tidb-operator   1               Tue Jan  5 15:28:00 2021        DEPLOYED        tidb-operator-v1.1.8    v1.1.8          tidb-admin
```

1. 参考 [Helm 官方文档](https://helm.sh/docs/intro/install/)安装 Helm 3。

    Helm 3 使用与 Helm 2 不同的配置与数据存储方式，因此在安装 Helm 3 的过程中无需担心对原有配置或数据的覆盖。

    > **注意：**
    >
    > 安装过程中需避免 Helm 3 的 CLI binary 覆盖 Helm 2 的 CLI binary。例如，可将 Helm 3 的 CLI binary 命名为 `heml3`（本文后续示例命令中将使用 `helm3`）。

2. 为 Helm 3 安装 [helm-2to3 插件](https://github.com/helm/helm-2to3)。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 plugin install https://github.com/helm/helm-2to3
    ```

    通过如下命令可确认是否已正确安装 helm-2to3 插件。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 plugin list
    ```

    ```
    NAME    VERSION DESCRIPTION
    2to3    0.8.0   migrate and cleanup Helm v2 configuration and releases in-place to Helm v3
    ```

3. 迁移 Helm 2 的仓库、插件等配置到 Helm 3。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 2to3 move config
    ```

    在正式迁移配置前，可使用 `helm3 2to3 move config --dry-run` 了解可能执行的操作及其影响。

    迁移配置完成后，可看到 Helm 3 中已包含 PingCAP 仓库。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 repo list
    ```

    ```
    NAME    URL
    pingcap https://charts.pingcap.org/
    ```

4. 迁移 Helm 2 管理的 release 到 Helm 3。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 2to3 convert tidb-operator
    ```

    在正式迁移 release 前，可使用 `helm3 2to3 convert tidb-operator --dry-run` 了解可能执行的操作及其影响。

    迁移 release 完成后，可通过 Helm 3 看到 TiDB Operator 对应的 release。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 list --namespace=tidb-admin
    ```

    ```
    NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART                   APP VERSION
    tidb-operator   tidb-admin      1               2021-01-05 07:28:00.3545941 +0000 UTC   deployed        tidb-operator-v1.1.8    v1.1.8
    ```

    > **注意：**
    >
    > 如果原 Helm 2 是 Tillerless 的（通过 [helm-tiller](https://github.com/rimusz/helm-tiller) 等插件将 Tiller 安装在本地而不是 Kubernetes 集群中），则可以通过增加 `--tiller-out-cluster` 参数进行迁移，即 `helm3 2to3 convert tidb-operator --tiller-out-cluster`。

5. 确认 TiDB Operator、TidbCluster 及 TidbMonitor 运行正常。

    使用以下命令检查 TiDB Operator 组件是否运行正常：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get pods --namespace=tidb-admin -l app.kubernetes.io/instance=tidb-operator
    ```

    期望输出为所有 Pod 都处于 Running 状态：

    ```
    NAME                                       READY   STATUS    RESTARTS   AGE
    tidb-controller-manager-6d8d5c6d64-b8lv4   1/1     Running   0          2m22s
    tidb-scheduler-644d59b46f-4f6sb            2/2     Running   0          2m22s
    ```

    使用以下命令检查 TidbCluster 和 TidbMonitor 组件是否运行正常：

    {{< copyable "shell-regular" >}}

    ``` shell
    watch kubectl get pods --namespace=tidb-cluster
    ```

    期望输出为所有 Pod 都处于 Running 状态：

    ```
    NAME                              READY   STATUS    RESTARTS   AGE
    basic-discovery-6bb656bfd-xl5pb   1/1     Running   0          9m9s
    basic-monitor-5fc8589c89-gvgjj    3/3     Running   0          8m58s
    basic-pd-0                        1/1     Running   0          9m8s
    basic-tidb-0                      2/2     Running   0          7m14s
    basic-tikv-0                      1/1     Running   0          8m13s
    ```

6. 清理 Helm 2 对应的配置、release 信息等数据。

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 2to3 cleanup --name=tidb-operator
    ```

    在正式清理数据前，可使用 `helm3 2to3 cleanup --name=tidb-operator --dry-run` 了解可能执行的操作及其影响。

    > **注意：**
    >
    > 清理完成后，Helm 2 中将无法再管理对应的 release。
