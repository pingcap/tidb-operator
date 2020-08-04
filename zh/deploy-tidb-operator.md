---
title: 在 Kubernetes 上部署 TiDB Operator
summary: 了解如何在 Kubernetes 上部署 TiDB Operator。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-tidb-operator/']
---

# 在 Kubernetes 上部署 TiDB Operator

本文介绍如何在 Kubernetes 上部署 TiDB Operator。

## 准备环境

TiDB Operator 部署前，请确认以下软件需求：

* Kubernetes v1.12 或者更高版本
* [DNS 插件](https://kubernetes.io/docs/tasks/access-application-cluster/configure-dns-cluster/)
* [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
* [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) 启用（可选）
* [Helm](https://helm.sh) 版本 >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797)

## 部署 Kubernetes 集群

TiDB Operator 运行在 Kubernetes 集群，你可以使用 [Getting started 页面](https://kubernetes.io/docs/setup/)列出的任何一种方法搭建一套 Kubernetes 集群。只要保证 Kubernetes 版本大于等于 v1.12。若想创建一个简单集群测试，可以参考[快速上手教程](get-started.md)。

TiDB Operator 使用[持久化卷](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)持久化存储 TiDB 集群数据（包括数据库，监控和备份数据），所以 Kubernetes 集群必须提供至少一种持久化卷。为提高性能，建议使用本地 SSD 盘作为持久化卷。可以根据[这一步](#配置本地持久化卷)配置本地持久化卷。

Kubernetes 集群建议启用 [RBAC](https://kubernetes.io/docs/admin/authorization/rbac)。

## 安装 Helm

参考 [使用 Helm](tidb-toolkit.md#使用-helm) 安装 Helm 并配置 PingCAP 官方 chart 仓库。

## 配置本地持久化卷

### 准备本地卷

参考[本地 PV 配置](configure-storage-class.md#本地-pv-配置)在你的 Kubernetes 集群中配置本地持久化卷。

## 安装 TiDB Operator

### 创建 CRD

TiDB Operator 使用 [Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) 扩展 Kubernetes，所以要使用 TiDB Operator，必须先创建 `TidbCluster` 自定义资源类型。只需要在你的 Kubernetes 集群上创建一次即可：

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.0/manifests/crd.yaml
```

如果服务器没有外网，需要先用有外网的机器下载 `crd.yaml` 文件，然后再进行安装：

{{< copyable "shell-regular" >}}

```shell
wget https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.0/manifests/crd.yaml
kubectl apply -f ./crd.yaml
```

如果显示如下信息表示 CRD 安装成功：

{{< copyable "shell-regular" >}}

```shell
kubectl get crd
```

```shell
NAME                                 CREATED AT
backups.pingcap.com                  2020-06-11T07:59:40Z
backupschedules.pingcap.com          2020-06-11T07:59:41Z
restores.pingcap.com                 2020-06-11T07:59:40Z
tidbclusterautoscalers.pingcap.com   2020-06-11T07:59:42Z
tidbclusters.pingcap.com             2020-06-11T07:59:38Z
tidbinitializers.pingcap.com         2020-06-11T07:59:42Z
tidbmonitors.pingcap.com             2020-06-11T07:59:41Z
```

### 安装 

创建以上各种自定义资源类型后，接下来在 Kubernetes 集群上安装 TiDB Operator，有两种安装方式：在线和离线安装 TiDB Operator。

#### 在线安装 TiDB Operator

1. 获取你要安装的 `tidb-operator` chart 中的 `values.yaml` 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p ${HOME}/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=${chart_version} > ${HOME}/tidb-operator/values-tidb-operator.yaml
    ```

    > **注意：**
    >
    > `${chart_version}` 在后续文档中代表 chart 版本，例如 `v1.1.0`，可以通过 `helm search -l tidb-operator` 查看当前支持的版本。

2. 配置 TiDB Operator

    TiDB Operator 里面会用到 `k8s.gcr.io/kube-scheduler` 镜像，如果无法下载该镜像，可以修改 `${HOME}/tidb-operator/values-tidb-operator.yaml` 文件中的 `scheduler.kubeSchedulerImageName` 为 `registry.cn-hangzhou.aliyuncs.com/google_containers/kube-scheduler`。

    其他项目例如：`limits`、`requests` 和 `replicas`，请根据需要进行修改。

3. 安装 TiDB Operator

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-operator --name=tidb-operator --namespace=tidb-admin --version=${chart_version} -f ${HOME}/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

4. 升级 TiDB Operator

    如果需要升级 TiDB Operator，请先修改 `${HOME}/tidb-operator/values-tidb-operator.yaml` 文件，然后执行下面的命令进行升级：

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator pingcap/tidb-operator -f  ${HOME}/tidb-operator/values-tidb-operator.yaml
    ```

#### 离线安装 TiDB Operator

如果服务器没有外网，需要按照下面的步骤来离线安装 TiDB Operator：

1. 下载 `tidb-operator` chart

    如果服务器上没有外网，就无法通过配置 Helm repo 来安装 TiDB Operator 组件以及其他应用。这时，需要在有外网的机器上下载集群安装需用到的 chart 文件，再拷贝到服务器上。

    通过以下命令，下载 `tidb-operator` chart 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    wget http://charts.pingcap.org/tidb-operator-v1.1.0.tgz
    ```

    将 `tidb-operator-v1.1.0.tgz` 文件拷贝到服务器上并解压到当前目录：

    {{< copyable "shell-regular" >}}

    ```shell
    tar zxvf tidb-operator.v1.1.0.tgz
    ```

2. 下载 TiDB Operator 运行所需的 Docker 镜像

    如果服务器没有外网，需要在有外网的机器上将 TiDB Operator 用到的所有 Docker 镜像下载下来并上传到服务器上，然后使用 `docker load` 将 Docker 镜像安装到服务器上。

    TiDB Operator 用到的 Docker 镜像有：

    {{< copyable "shell-regular" >}}

    ```shell
    pingcap/tidb-operator:v1.1.0
    pingcap/tidb-backup-manager:v1.1.0
    bitnami/kubectl:latest
    pingcap/advanced-statefulset:v0.3.3
    k8s.gcr.io/kube-scheduler:v1.16.9
    ```

    其中 `k8s.gcr.io/kube-scheduler:v1.16.9` 请跟你的 Kubernetes 集群的版本保持一致即可，不用单独下载。

    接下来通过下面的命令将所有这些镜像下载下来：

    {{< copyable "shell-regular" >}}

    ```shell
    docker pull pingcap/tidb-operator:v1.1.0
    docker pull pingcap/tidb-backup-manager:v1.1.0
    docker pull bitnami/kubectl:latest
    docker pull pingcap/advanced-statefulset:v0.3.3

    docker save -o tidb-operator-v1.1.0.tar pingcap/tidb-operator:v1.1.0
    docker save -o tidb-backup-manager-v1.1.0.tar pingcap/tidb-backup-manager:v1.1.0
    docker save -o bitnami-kubectl.tar bitnami/kubectl:latest
    docker save -o advanced-statefulset-v0.3.3.tar pingcap/advanced-statefulset:v0.3.3
    ```

    接下来将这些 Docker 镜像上传到服务器上，并执行 `docker load` 将这些 Docker 镜像安装到服务器上：

    {{< copyable "shell-regular" >}}

    ```shell
    docker load -i tidb-operator-v1.1.0.tar
    docker load -i tidb-backup-manager-v1.1.0.tar
    docker load -i bitnami-kubectl.tar
    docker load -i advanced-statefulset-v0.3.3.tar
    ```

3. 配置 TiDB Operator

    TiDB Operator 内嵌了一个 `kube-scheduler` 用来实现自定义调度器，请修改 `./tidb-operator/values.yaml` 文件来配置这个内置 `kube-scheduler` 组件的 Docker 镜像名字和版本，例如你的 Kubernetes 集群中的 `kube-scheduler` 使用的镜像为 `k8s.gcr.io/kube-scheduler:v1.16.9`，请这样设置 `./tidb-operator/values.yaml`：

    ```shell
    ...
    scheduler:
      serviceAccount: tidb-scheduler
      logLevel: 2
      replicas: 1
      schedulerName: tidb-scheduler
      resources:
        limits:
          cpu: 250m
          memory: 150Mi
        requests:
          cpu: 80m
          memory: 50Mi
      kubeSchedulerImageName: k8s.gcr.io/kube-scheduler
      kubeSchedulerImageTag: v1.16.9
    ...
    ```

    其他项目例如：`limits`、`requests` 和 `replicas`，请根据需要进行修改。

4. 安装 TiDB Operator

    使用下面的命令安装 TiDB Operator：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ./tidb-operator --name=tidb-operator --namespace=tidb-admin
    ```

5. 升级 TiDB Operator

    如果需要升级 TiDB Operator，请先修改 `./tidb-operator/values.yaml` 文件，然后执行下面的命令进行升级：

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator ./tidb-operator
    ```
