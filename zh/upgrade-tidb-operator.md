---
title: 升级 TiDB Operator
summary: 介绍如何升级 TiDB Operator。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/upgrade-tidb-operator/']
---

# 升级 TiDB Operator

本文介绍如何升级 TiDB Operator。

## 升级步骤

1. 更新 [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/)：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${version}/manifests/crd.yaml && \
    kubectl get crd tidbclusters.pingcap.com
    ```

    > **注意：**
    >
    > `${version}` 在后续文档中代表 TiDB Operator 版本，例如 `v1.1.5`，可以通过 `helm search -l tidb-operator` 查看当前支持的版本。

2. 获取你要安装的 `tidb-operator` chart 中的 `values.yaml` 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p ${HOME}/tidb-operator/${version} && \
    helm inspect values pingcap/tidb-operator --version=${version} > ${HOME}/tidb-operator/${version}/values-tidb-operator.yaml
    ```
    
3. 修改 `${HOME}/tidb-operator/${version}/values-tidb-operator.yaml` 中 `operatorImage` 镜像版本，并将旧版本 `values.yaml` 中的自定义配置合并到 `${HOME}/tidb-operator/${version}/values-tidb-operator.yaml`，然后执行 `helm upgrade`：

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator pingcap/tidb-operator --version=${version} -f ${HOME}/tidb-operator/${version}/values-tidb-operator.yaml
    ```

    > **注意：**
    >
    > TiDB Operator 升级之后，所有 TiDB 集群中的 `discovery` deployment 都会自动升级到指定的 TiDB Operator 版本。

## 从 TiDB Operator v1.0 版本升级到 v1.1 及之后版本

从 TiDB Operator v1.1.0 开始，PingCAP 不再继续更新维护 tidb-cluster chart，原来由 tidb-cluster chart 负责管理的组件或者功能在 v1.1 中切换到 CR (Custom Resource) 或者单独的 chart 进行管理，详细信息请参考 [TiDB Operator v1.1 重要注意事项](notes-tidb-operator-v1.1.md)。
