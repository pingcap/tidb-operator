---
title: 在 GCP 上通过 Kubernetes 部署 TiDB 集群
summary: 在 GCP 上通过 Kubernetes 部署 TiDB 集群教程。
category: how-to
---

# 在 GCP 上通过 Kubernetes 部署 TiDB 集群

本文介绍如何使用 [TiDB Operator](https://github.com/pingcap/tidb-operator) 在 GCP 上部署 TiDB 集群。本教程需要在 [Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator&tutorial=docs/google-kubernetes-tutorial.md) 上运行。

所包含的步骤如下：

- 启动一个包含 3 个节点的 Kubernetes 集群（可选）
- 部署 TiDB Operator 和 TiDB 集群
- 访问 TiDB 集群
- 扩容 TiDB 集群
- 访问 Grafana 面板
- 销毁 TiDB 集群
- 删除 Kubernetes 集群（可选）

> **警告：**
>
> 对于生产环境，不要使用此方式进行部署。

## 选择一个项目

本教程会启动一个包含 3 个 `n1-standard-1` 类型节点的 Kubernetes 集群。价格信息可以参考 [All pricing](https://cloud.google.com/compute/pricing)。

继续之前请选择一个项目：

<walkthrough-project-billing-setup key="project-id">
</walkthrough-project-billing-setup>

## 启用 API

本教程需要使用计算和容器 API。继续之前请启用：

<walkthrough-enable-apis apis="container.googleapis.com,compute.googleapis.com">
</walkthrough-enable-apis>

## 配置 gcloud

这一步配置 glcoud 默认访问你要用的项目和[可用区](https://cloud.google.com/compute/docs/regions-zones/)，可以简化后面用到的命令：

{{< copyable "shell-regular" >}}

``` shell
gcloud config set project {{project-id}} && \
gcloud config set compute/zone us-west1-a
```

## 启动 3 个节点的 Kubernetes 集群

下面命令启动一个包含 3 个 `n1-standard-1` 类型节点的 Kubernetes 集群。

命令执行需要几分钟时间：

{{< copyable "shell-regular" >}}

``` shell
gcloud container clusters create tidb
```

集群启动完成后，将其设置为默认集群：

{{< copyable "shell-regular" >}}

``` shell
gcloud config set container/cluster tidb
```

最后验证 `kubectl` 可以访问集群并且 3 个节点正常运行：

{{< copyable "shell-regular" >}}

``` shell
kubectl get nodes
```

如果所有节点状态为 `Ready`，恭喜你，你已经成功搭建你的第一个 Kubernetes 集群。

## 安装 Helm

[Helm](https://helm.sh/) 是一个 Kubernetes 的包管理工具，确保安装的 Helm 版本为 >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797)。安装步骤如下：

1. 参考[官方文档](https://v2.helm.sh/docs/using_helm/#installing-helm)安装 Helm 客户端
2. 安装 Helm 服务端

    在集群中应用 Helm 服务端组件 `tiller` 所需的 `RBAC` 规则，并安装 `tiller`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/tiller-rbac.yaml && \
    helm init --service-account=tiller --upgrade
    ```

    通过下面命令确认 `tiller` Pod 进入 running 状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n kube-system -l name=tiller
    ```

3. 通过下面的命令添加仓库：

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo add pingcap https://charts.pingcap.org/
    ```

    添加完成后，可以使用 `helm search` 搜索 PingCAP 提供的 chart：

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l
    ```

## 部署 TiDB Operator

TiDB Operator 使用 [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) 扩展 Kubernetes，所以要使用 TiDB Operator，必须先创建 `TidbCluster` 等各种自定义资源类型：

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

创建 `TidbCluster` 自定义资源类型后，接下来在 Kubernetes 集群上安装 TiDB Operator。

1. 获取你要安装的 `tidb-operator` chart 中的 `values.yaml` 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p /home/tidb/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=v1.1.0-rc.1 > /home/tidb/tidb-operator/values-tidb-operator.yaml
    ```

    按需修改 `values.yaml` 文件中的配置。

2. 安装 TiDB Operator

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-operator --name=tidb-operator --namespace=tidb-admin --version=v1.1.0-rc.1 -f /home/tidb/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

3. 创建 `pd-ssd` StorageClass：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/gke/persistent-disk.yaml
    ```

## 部署 TiDB 集群

通过下面命令部署 TiDB 集群：

1. 创建 `Namespace`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create namespace demo
    ```

2. 部署 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml -n demo
    ```

3. 部署 TiDB 集群监控：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n demo
    ```

4. 通过下面命令查看 Pod 状态：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n demo
    ```

## 访问 TiDB 集群

从 pod 启动、运行到服务可以访问有一些延时，可以通过下面命令查看服务：

{{< copyable "shell-regular" >}}

``` shell
kubectl get svc -n demo --watch
```

如果看到 `basic-tidb` 出现，说明服务已经可以访问，可以 <kbd>Ctrl</kbd>+<kbd>C</kbd> 停止。

要访问 Kubernetes 集群中的 TiDB 服务，可以在 TiDB 服务和 Google Cloud Shell 之间建立一条隧道。建议这种方式只用于调试，因为如果 Google Cloud Shell 重启，隧道不会自动重新建立。要建立隧道：

{{< copyable "shell-regular" >}}

``` shell
kubectl -n demo port-forward svc/basic-tidb 4000:4000 &>/tmp/port-forward.log &
```

在 Cloud Shell 上运行：

{{< copyable "shell-regular" >}}

``` shell
sudo apt-get install -y mysql-client && \
mysql -h 127.0.0.1 -u root -P 4000
```

在 MySQL 终端中输入一条 MySQL 命令：

{{< copyable "sql" >}}

``` sql
select tidb_version();
```

如果安装的过程中没有指定密码，现在可以设置：

{{< copyable "sql" >}}

``` sql
SET PASSWORD FOR 'root'@'%' = '<change-to-your-password>';
```

> **注意：**
>
> 这条命令中包含一些特殊字符，Google Cloud Shell 无法自动填充，你需要手动复制、粘贴到控制台中。

恭喜，你已经启动并运行一个兼容 MySQL 的分布式 TiDB 数据库！

## 扩容 TiDB 集群

使用 kubectl 修改集群所对应的 `TidbCluster` 对象中的 `spec.pd.replicas`、`spec.tidb.replicas`、`spec.tikv.replicas` 至期望值进行水平扩容。

{{< copyable "shell-regular" >}}

``` shell
kubectl -n demo edit tc basic
```

## 访问 Grafana 面板

要访问 Grafana 面板，可以在 Grafana 服务和 shell 之间建立一条隧道，可以使用如下命令：

{{< copyable "shell-regular" >}}

``` shell
kubectl -n demo port-forward svc/basic-grafana 3000:3000 &>/dev/null &
```

在 Cloud Shell 中，点击 Web Preview 按钮并输入端口 3000，将打开一个新的浏览器标签页访问 Grafana 面板。或者也可以在新浏览器标签或者窗口中直接访问 URL：<https://ssh.cloud.google.com/devshell/proxy?port=3000>。

如果没有使用 Cloud Shell，可以在浏览器中访问 `localhost:3000`。

## 销毁 TiDB 集群

要删除 TiDB 集群，执行以下命令：

{{< copyable "shell-regular" >}}

```shell
kubectl delete tc basic -n demo
```

要删除监控组件，执行以下命令：

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n demo
```

上面的命令只会删除运行的 Pod，但是数据还会保留。如果你不再需要那些数据，可以执行下面的命令清除数据和动态创建的持久化磁盘：

{{< copyable "shell-regular" >}}

``` shell
kubectl delete pvc -n demo -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator && \
kubectl get pv -l app.kubernetes.io/namespace=demo,app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=basic -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

## 删除 Kubernetes 集群

实验结束后，可以使用如下命令删除 Kubernetes 集群：

{{< copyable "shell-regular" >}}

``` shell
gcloud container clusters delete tidb
```

## 更多信息

我们还提供[基于 Terraform 的部署方案](deploy-on-gcp-gke.md)。
