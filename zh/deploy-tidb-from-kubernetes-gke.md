---
title: 在 GCP 上通过 Kubernetes 部署 TiDB 集群
summary: 在 GCP 上通过 Kubernetes 部署 TiDB 集群教程。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-tidb-from-kubernetes-gke/']
---

# 在 GCP 上通过 Kubernetes 部署 TiDB 集群

本文介绍如何使用 [TiDB Operator](https://github.com/pingcap/tidb-operator) 在 GCP 上部署 TiDB 集群。本教程需要在 [Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/docs-tidb-operator&tutorial=zh/deploy-tidb-from-kubernetes-gke.md) 上运行。

<a href="https://console.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/pingcap/docs-tidb-operator&cloudshell_tutorial=zh/deploy-tidb-from-kubernetes-gke.md"><img src="https://gstatic.com/cloudssh/images/open-btn.png"/></a>

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
> 本文中的部署说明仅用于测试目的，**不要**直接用于生产环境。如果要在生产环境部署，请参阅[在 GCP 上通过 Kubernetes 部署 TiDB 集群](deploy-on-gcp-gke.md)。

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

``` shell
gcloud config set project {{project-id}} && \
gcloud config set compute/zone us-west1-a
```

## 启动 3 个节点的 Kubernetes 集群

下面命令启动一个包含 3 个 `n1-standard-1` 类型节点的 Kubernetes 集群。

命令执行需要几分钟时间：

``` shell
gcloud container clusters create tidb
```

集群启动完成后，将其设置为默认集群：

``` shell
gcloud config set container/cluster tidb
```

最后验证 `kubectl` 可以访问集群并且 3 个节点正常运行：

``` shell
kubectl get nodes
```

如果所有节点状态为 `Ready`，恭喜你，你已经成功搭建你的第一个 Kubernetes 集群。

## 安装 Helm

[Helm](https://helm.sh/) 是一个 Kubernetes 的包管理工具。

1. 安装 Helm 服务端

    ```shell
    curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
    ```

2. 通过下面的命令添加仓库：

    ```shell
    helm repo add pingcap https://charts.pingcap.org/
    ```

## 部署 TiDB Operator

TiDB Operator 使用 [Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) 扩展 Kubernetes，所以要使用 TiDB Operator，必须先创建 `TidbCluster` 等各种自定义资源类型：

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

创建 `TidbCluster` 自定义资源类型后，接下来在 Kubernetes 集群上安装 TiDB Operator。

```shell
kubectl create namespace tidb-admin
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.2.4
kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
```

## 部署 TiDB 集群

通过下面命令部署 TiDB 集群：

1. 创建 `Namespace`：

    ```shell
    kubectl create namespace demo
    ```

2. 部署 TiDB 集群：

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml -n demo
    ```

3. 部署 TiDB 集群监控：

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n demo
    ```

4. 通过下面命令查看 Pod 状态：

    ``` shell
    kubectl get po -n demo
    ```

## 访问 TiDB 集群

从 pod 启动、运行到服务可以访问有一些延时，可以通过下面命令查看服务：

``` shell
kubectl get svc -n demo --watch
```

如果看到 `basic-tidb` 出现，说明服务已经可以访问，可以 <kbd>Ctrl</kbd>+<kbd>C</kbd> 停止。

要访问 Kubernetes 集群中的 TiDB 服务，可以在 TiDB 服务和 Google Cloud Shell 之间建立一条隧道。建议这种方式只用于调试，因为如果 Google Cloud Shell 重启，隧道不会自动重新建立。要建立隧道：

``` shell
kubectl -n demo port-forward svc/basic-tidb 4000:4000 &>/tmp/pf4000.log &
```

在 Cloud Shell 上运行：

``` shell
sudo apt-get install -y mysql-client && \
mysql --comments -h 127.0.0.1 -u root -P 4000
```

在 MySQL 终端中输入一条 MySQL 命令：

``` sql
select tidb_version();
```

如果安装的过程中没有指定密码，现在可以设置：

``` sql
SET PASSWORD FOR 'root'@'%' = '<change-to-your-password>';
```

> **注意：**
>
> 这条命令中包含一些特殊字符，Google Cloud Shell 无法自动填充，你需要手动复制、粘贴到控制台中。

恭喜，你已经启动并运行一个兼容 MySQL 的分布式 TiDB 数据库！

> **注意：**
>
> TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见[遥测](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

## 扩容 TiDB 集群

使用 kubectl 修改集群所对应的 `TidbCluster` 对象中的 `spec.pd.replicas`、`spec.tidb.replicas`、`spec.tikv.replicas` 至期望值进行水平扩容。

``` shell
kubectl -n demo edit tc basic
```

## 访问 Grafana 面板

要访问 Grafana 面板，可以在 Grafana 服务和 shell 之间建立一条隧道，可以使用如下命令（Cloud Shell 占用了 3000 端口，我们选择 8080 做映射）：

``` shell
kubectl -n demo port-forward svc/basic-grafana 8080:3000 &>/tmp/pf8080.log &
```

在 Cloud Shell 中，点击右上方的 Web Preview 按钮并修改端口为 8080 后点击预览，将打开一个新的浏览器标签页访问 Grafana 面板。或者也可以在新浏览器标签或者窗口中直接访问 URL：<https://ssh.cloud.google.com/devshell/proxy?port=8080>。

如果没有使用 Cloud Shell，可以在浏览器中访问 `localhost:8080`。

默认用户名和密码为：admin / admin 。

> **注意：**
>
> 默认会提醒修改账户密码，可点击 Skip 跳过。生产环境中建议配置安全密码。

## 销毁 TiDB 集群

要删除 TiDB 集群，执行以下命令：

```shell
kubectl delete tc basic -n demo
```

要删除监控组件，执行以下命令：

```shell
kubectl delete tidbmonitor basic -n demo
```

上面的命令只会删除运行的 Pod，但是数据还会保留。如果你不再需要那些数据，可以执行下面的命令清除数据和动态创建的持久化磁盘：

``` shell
kubectl delete pvc -n demo -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator && \
kubectl get pv -l app.kubernetes.io/namespace=demo,app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=basic -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

## 删除 Kubernetes 集群

实验结束后，可以使用如下命令删除 Kubernetes 集群：

``` shell
gcloud container clusters delete tidb
```
