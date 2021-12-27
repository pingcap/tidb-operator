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
* [Helm 3](https://helm.sh)

## 部署 Kubernetes 集群

TiDB Operator 运行在 Kubernetes 集群，你可以使用 [Getting started 页面](https://kubernetes.io/docs/setup/)列出的任何一种方法搭建一套 Kubernetes 集群。只要保证 Kubernetes 版本大于等于 v1.12。若想创建一个简单集群测试，可以参考[快速上手教程](get-started.md)。

对于部分公有云环境，可以参考如下文档部署 TiDB Operator 及 TiDB 集群：

- [部署到 AWS EKS](deploy-on-aws-eks.md)
- [部署到 GCP GKE](deploy-on-gcp-gke.md)
- [部署到阿里云 ACK](deploy-on-alibaba-cloud.md)

TiDB Operator 使用[持久化卷](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)持久化存储 TiDB 集群数据（包括数据库，监控和备份数据），所以 Kubernetes 集群必须提供至少一种持久化卷。

Kubernetes 集群建议启用 [RBAC](https://kubernetes.io/docs/admin/authorization/rbac)。

## 安装 Helm

参考 [使用 Helm](tidb-toolkit.md#使用-helm) 安装 Helm 并配置 PingCAP 官方 chart 仓库。

## 部署 TiDB Operator

### 创建 CRD

TiDB Operator 使用 [Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) 扩展 Kubernetes，所以要使用 TiDB Operator，必须先创建 `TidbCluster` 自定义资源类型。只需要在你的 Kubernetes 集群上创建一次即可：

{{< copyable "shell-regular" >}}

```bash
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml
```

如果服务器没有外网，需要先用有外网的机器下载 `crd.yaml` 文件，然后再进行安装：

{{< copyable "shell-regular" >}}

```bash
wget https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml
kubectl create -f ./crd.yaml
```

> **注意：**
> 
> 对于 Kubernetes 1.16 之前的版本，Kubernetes 仅支持 v1beta1 版本的 CRD，你需要将上述命令中的 `crd.yaml` 修改为 `crd_v1beta1.yaml`。

如果显示如下信息表示 CRD 安装成功：

{{< copyable "shell-regular" >}}

```bash
kubectl get crd
```

```bash
NAME                                 CREATED AT
backups.pingcap.com                  2020-06-11T07:59:40Z
backupschedules.pingcap.com          2020-06-11T07:59:41Z
restores.pingcap.com                 2020-06-11T07:59:40Z
tidbclusterautoscalers.pingcap.com   2020-06-11T07:59:42Z
tidbclusters.pingcap.com             2020-06-11T07:59:38Z
tidbinitializers.pingcap.com         2020-06-11T07:59:42Z
tidbmonitors.pingcap.com             2020-06-11T07:59:41Z
```

### 自定义部署 TiDB Operator

若需要快速部署 TiDB Operator，可参考快速上手中[部署 TiDB Operator文档](get-started.md#部署-tidb-operator)。本节介绍自定义部署 TiDB Operator 的配置方式。

创建 CRDs 之后，在 Kubernetes 集群上部署 TiDB Operator有两种方式：在线和离线部署。

在使用 TiDB Operator 时，`tidb-scheduler` 并不是必须使用。你可以参考 [tidb-scheduler 与 default-scheduler](tidb-scheduler.md#tidb-scheduler-与-default-scheduler)，确认是否需要部署 `tidb-scheduler`。如果不需要 `tidb-scheduler`，在部署 TiDB Operator 过程中，可以通过在 `values.yaml` 文件中配置 `scheduler.create: false` 不部署 `tidb-scheduler`。

#### 在线部署 TiDB Operator

1. 获取你要部署的 `tidb-operator` chart 中的 `values.yaml` 文件：

    {{< copyable "shell-regular" >}}

    ```bash
    mkdir -p ${HOME}/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=${chart_version} > ${HOME}/tidb-operator/values-tidb-operator.yaml
    ```

    > **注意：**
    >
    > `${chart_version}` 在后续文档中代表 chart 版本，例如 `v1.2.4`，可以通过 `helm search repo -l tidb-operator` 查看当前支持的版本。

2. 配置 TiDB Operator

    如果要部署 `tidb-scheduler`，会用到 `k8s.gcr.io/kube-scheduler` 镜像，如果无法下载该镜像，可以修改 `${HOME}/tidb-operator/values-tidb-operator.yaml` 文件中的 `scheduler.kubeSchedulerImageName` 为 `registry.cn-hangzhou.aliyuncs.com/google_containers/kube-scheduler`。

    TiDB Operator 默认会管理 Kubernetes 集群中的所有 TiDB 集群，如仅需其管理特定 namespace 下的集群，则可在 `values.yaml` 中设置 `clusterScoped: false`。

    > **注意：**
    >
    > 在设置 `clusterScoped: false` 后，TiDB Operator 默认仍会操作 Kubernetes 集群中的 Nodes、Persistent Volumes 与 Storage Classes。若部署 TiDB Operator 的角色不具备这些资源的操作权限，则可以将 `controllerManager.clusterPermissions` 下的相应权限请求设置为 `false` 以禁用 TiDB Operator 对这些资源的操作。

    其他项目例如：`limits`、`requests` 和 `replicas`，请根据需要进行修改。

3. 部署 TiDB Operator

    {{< copyable "shell-regular" >}}

    ```bash
    helm install tidb-operator pingcap/tidb-operator --namespace=tidb-admin --version=${chart_version} -f ${HOME}/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

    > **注意：**
    >
    > 如果对应 `tidb-admin` namespace 不存在，则可先使用 `kubectl create namespace tidb-admin` 创建该 namespace。

4. 升级 TiDB Operator

    如果需要升级 TiDB Operator，请先修改 `${HOME}/tidb-operator/values-tidb-operator.yaml` 文件，然后执行下面的命令进行升级：

    {{< copyable "shell-regular" >}}

    ```bash
    helm upgrade tidb-operator pingcap/tidb-operator --namespace=tidb-admin -f ${HOME}/tidb-operator/values-tidb-operator.yaml
    ```

#### 离线安装 TiDB Operator

如果服务器没有外网，需要按照下面的步骤来离线安装 TiDB Operator：

1. 下载 `tidb-operator` chart

    如果服务器上没有外网，就无法通过配置 Helm repo 来安装 TiDB Operator 组件以及其他应用。这时，需要在有外网的机器上下载集群安装需用到的 chart 文件，再拷贝到服务器上。

    通过以下命令，下载 `tidb-operator` chart 文件：

    {{< copyable "shell-regular" >}}

    ```bash
    wget http://charts.pingcap.org/tidb-operator-v1.2.4.tgz
    ```

    将 `tidb-operator-v1.2.4.tgz` 文件拷贝到服务器上并解压到当前目录：

    {{< copyable "shell-regular" >}}

    ```bash
    tar zxvf tidb-operator.v1.2.4.tgz
    ```

2. 下载 TiDB Operator 运行所需的 Docker 镜像

    如果服务器没有外网，需要在有外网的机器上将 TiDB Operator 用到的所有 Docker 镜像下载下来并上传到服务器上，然后使用 `docker load` 将 Docker 镜像安装到服务器上。

    TiDB Operator 用到的 Docker 镜像有：

    ```bash
    pingcap/tidb-operator:v1.2.4
    pingcap/tidb-backup-manager:v1.2.4
    bitnami/kubectl:latest
    pingcap/advanced-statefulset:v0.3.3
    k8s.gcr.io/kube-scheduler:v1.16.9
    ```

    其中 `k8s.gcr.io/kube-scheduler:v1.16.9` 请跟你的 Kubernetes 集群的版本保持一致即可，不用单独下载。

    接下来通过下面的命令将所有这些镜像下载下来：

    {{< copyable "shell-regular" >}}

    ```bash
    docker pull pingcap/tidb-operator:v1.2.4
    docker pull pingcap/tidb-backup-manager:v1.2.4
    docker pull bitnami/kubectl:latest
    docker pull pingcap/advanced-statefulset:v0.3.3

    docker save -o tidb-operator-v1.2.4.tar pingcap/tidb-operator:v1.2.4
    docker save -o tidb-backup-manager-v1.2.4.tar pingcap/tidb-backup-manager:v1.2.4
    docker save -o bitnami-kubectl.tar bitnami/kubectl:latest
    docker save -o advanced-statefulset-v0.3.3.tar pingcap/advanced-statefulset:v0.3.3
    ```

    接下来将这些 Docker 镜像上传到服务器上，并执行 `docker load` 将这些 Docker 镜像安装到服务器上：

    {{< copyable "shell-regular" >}}

    ```bash
    docker load -i tidb-operator-v1.2.4.tar
    docker load -i tidb-backup-manager-v1.2.4.tar
    docker load -i bitnami-kubectl.tar
    docker load -i advanced-statefulset-v0.3.3.tar
    ```

3. 配置 TiDB Operator

    如果需要部署 `tidb-scheduler`，请修改 `./tidb-operator/values.yaml` 文件来配置内置 `kube-scheduler` 组件的 Docker 镜像名字和版本，例如你的 Kubernetes 集群中的 `kube-scheduler` 使用的镜像为 `k8s.gcr.io/kube-scheduler:v1.16.9`，请这样设置 `./tidb-operator/values.yaml`：

    ```bash
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

    ```bash
    helm install tidb-operator ./tidb-operator --namespace=tidb-admin
    ```

    > **注意：**
    >
    > 如果对应 `tidb-admin` namespace 不存在，则可先使用 `kubectl create namespace tidb-admin` 创建该 namespace。

5. 升级 TiDB Operator

    如果需要升级 TiDB Operator，请先修改 `./tidb-operator/values.yaml` 文件，然后执行下面的命令进行升级：

    {{< copyable "shell-regular" >}}

    ```bash
    helm upgrade tidb-operator ./tidb-operator --namespace=tidb-admin
    ```

## 自定义配置 TiDB Operator

可以通过修改 `${HOME}/tidb-operator/values-tidb-operator.yaml` 来配置 TiDB Operator。本节后续使用 `values.yaml` 来代表 `${HOME}/tidb-operator/values-tidb-operator.yaml`。

TiDB Operator 包含两个组件：

* tidb-controller-manager
* tidb-scheduler

这两个组件都是无状态的，由 `Deployment` 部署。在 `values.yaml` 文件中，你可以配置其中的 `limit`、`request` 与 `replicas` 参数。

修改了 `values.yaml` 文件后，请运行以下命令使更改生效：

{{< copyable "shell-regular" >}}

```bash
helm upgrade tidb-operator pingcap/tidb-operator --version=${chart_version} --namespace=tidb-admin -f ${HOME}/tidb-operator/values-tidb-operator.yaml
```
