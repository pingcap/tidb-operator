---
title: 升级 TiDB Operator
summary: 介绍如何升级 TiDB Operator。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/upgrade-tidb-operator/']
---

# 升级 TiDB Operator

本文介绍如何升级 TiDB Operator。

 > **注意：**
 >
 > 你可以通过 `helm search repo -l tidb-operator` 命令查看当前支持的 TiDB Operator 版本。
 > 如果此命令的输出中未包含最新版本，可以使用 `helm repo update` 命令更新 repo。详情请参考[配置 Helm repo](tidb-toolkit.md#配置-helm-repo) 。

## 在线升级步骤

1. 更新 Kubernetes 的 CustomResourceDefinition (CRD)。关于 CRD 的更多信息，请参阅 [CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/)。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.2.1/manifests/crd.yaml && \
    kubectl get crd tidbclusters.pingcap.com
    ```

2. 获取你要升级的 `tidb-operator` chart 中的 `values.yaml` 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p ${HOME}/tidb-operator/v1.2.1 && \
    helm inspect values pingcap/tidb-operator --version=v1.2.1 > ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
    ```

3. 修改 `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml` 中 `operatorImage` 镜像版本为要升级到的版本，并将旧版本 `values.yaml` 中的自定义配置合并到 `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml`，然后执行 `helm upgrade`：

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator pingcap/tidb-operator --version=v1.2.1 -f ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
    ```
    
    Pod 全部正常启动之后，运行以下命令确认 TiDB Operator 镜像版本：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n tidb-admin -l app.kubernetes.io/instance=tidb-operator -o yaml | grep 'image:.*operator:'
    ```

    输出类似下方结果则表示升级成功，`v1.2.1`表示要升级到的版本号。

    ```
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    ```

    > **注意：**
    >
    > TiDB Operator 升级之后，所有 TiDB 集群中的 `discovery` deployment 都会自动升级到对应的 TiDB Operator 版本。

## 离线升级步骤

如果服务器没有连接外网，你可以按照以下步骤离线升级 TiDB Operator：

1. 使用有外网的机器下载升级所需的文件和镜像。

    1. 下载 TiDB Operator 需要的 `crd.yaml` 文件。关于 CRD 的更多信息，请参阅 [CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/)。
   
        {{< copyable "shell-regular" >}}

        ```shell
        wget https://raw.githubusercontent.com/pingcap/tidb-operator/v1.2.1/manifests/crd.yaml
        ```

    2. 下载 `tidb-operator` chart 包文件：

        {{< copyable "shell-regular" >}}

        ```shell
        wget http://charts.pingcap.org/tidb-operator-v1.2.1.tgz
        ```
   
    3. 下载 TiDB Operator 升级所需的 Docker 镜像:

        {{< copyable "shell-regular" >}}

        ```shell
        docker pull pingcap/tidb-operator:v1.2.1
        docker pull pingcap/tidb-backup-manager:v1.2.1

        docker save -o tidb-operator-v1.2.1.tar pingcap/tidb-operator:v1.2.1
        docker save -o tidb-backup-manager-v1.2.1.tar pingcap/tidb-backup-manager:v1.2.1
        ```
   
2. 将下载的文件和镜像上传到需要升级的服务器上，然后按照以下步骤进行安装：

    1. 安装 TiDB Operator 需要的 `crd.yaml` 文件：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f ./crd.yaml
        ```

    2. 解压 `tidb-operator` chart 包文件，并拷贝 `values.yaml` 文件：

        {{< copyable "shell-regular" >}}

        ```shell
        tar zxvf tidb-operator-v1.2.1.tgz && \
        mkdir -p ${HOME}/tidb-operator/v1.2.1 && \
        cp tidb-operator/values.yaml ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
        ```

    3. 安装 Docker 镜像到服务器上：

        {{< copyable "shell-regular" >}}

        ```shell
        docker load -i tidb-operator-v1.2.1.tar
        docker load -i tidb-backup-manager-v1.2.1.tar
        ```

3. 修改 `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml` 中 `operatorImage` 镜像版本为要升级到的版本，并将旧版本 `values.yaml` 中的自定义配置合并到 `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml`，然后执行 `helm upgrade`：

   {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator ./tidb-operator --version=v1.2.1 -f ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
    ```

   Pod 全部正常启动之后，运行以下命令确认 TiDB Operator 镜像版本：

   {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n tidb-admin -l app.kubernetes.io/instance=tidb-operator -o yaml | grep 'image:.*operator:'
    ```

   输出类似下方结果则表示升级成功，`v1.2.1`表示要升级到的版本号。

    ```
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    ```

   > **注意：**
   >
   > TiDB Operator 升级之后，所有 TiDB 集群中的 `discovery` deployment 都会自动升级到对应的 TiDB Operator 版本。

## 从 TiDB Operator v1.0 版本升级到 v1.1 及之后版本

从 TiDB Operator v1.1.0 开始，PingCAP 不再继续更新维护 tidb-cluster chart，原来由 tidb-cluster chart 负责管理的组件或者功能在 v1.1 中切换到 CR (Custom Resource) 或者单独的 chart 进行管理，详细信息请参考 [TiDB Operator v1.1 重要注意事项](notes-tidb-operator-v1.1.md)。
