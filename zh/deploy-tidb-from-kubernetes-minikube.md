---
title: 在 Minikube 集群上部署 TiDB 集群
summary: 介绍如何在 Minikube 集群上部署 TiDB 集群。
category: how-to
---

# 在 Minikube 集群上部署 TiDB 集群

[Minikube](https://kubernetes.io/docs/setup/minikube/) 可以让你在个人电脑上的虚拟机中创建一个 Kubernetes 集群，支持 macOS、Linux 和 Windows 系统。本文介绍如何在 Minikube 集群上部署 TiDB 集群。

> **警告：**
>
> - 对于生产环境，不要使用此方式进行部署。
>
> - 尽管 Minikube 支持通过 `--vm-driver=none` 选项使用主机 Docker 而不使用虚拟机，但是目前尚没有针对 TiDB Operator 做过全面的测试，可能会无法正常工作。如果你想在不支持虚拟化的系统（例如，VPS）上试用 TiDB Operator，可以考虑使用 [kind](deploy-tidb-from-kubernetes-kind.md)。

## 安装 Minikube 并启动 Kubernetes 集群

参考[安装 Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)，在你的机器上安装 Minikube 1.0.0+。

安装完 Minikube 后，可以执行下面命令启动一个 Kubernetes 集群：

{{< copyable "shell-regular" >}}

``` shell
minikube start
```

对于中国大陆用户，可以使用国内 gcr.io mirror 仓库，例如 `registry.cn-hangzhou.aliyuncs.com/google_containers`。

{{< copyable "shell-regular" >}}

``` shell
minikube start --image-repository registry.cn-hangzhou.aliyuncs.com/google_containers
```

或者给 Docker 配置 HTTP/HTTPS 代理。

将下面命令中的 `127.0.0.1:1086` 替换为你自己的 HTTP/HTTPS 代理地址：

{{< copyable "shell-regular" >}}

``` shell
minikube start --docker-env https_proxy=http://127.0.0.1:1086 \
  --docker-env http_proxy=http://127.0.0.1:1086
```

> **注意：**
>
> 由于 Minikube 通过虚拟机（默认）运行，`127.0.0.1` 是虚拟机本身，有些情况下你可能想要使用你的主机的实际 IP。

参考 [Minikube setup](https://kubernetes.io/docs/setup/minikube/) 查看配置虚拟机和 Kubernetes 集群的更多选项。

## 安装 kubectl 并访问集群

Kubernetes 命令行工具 [kubectl](https://kubernetes.io/docs/user-guide/kubectl/)，可以让你执行命令访问 Kubernetes 集群。

参考 [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 安装和配置 kubectl。

kubectl 安装完成后，测试 Minikube Kubernetes 集群：

{{< copyable "shell-regular" >}}

``` shell
kubectl cluster-info
```

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

## 测试 TiDB 集群

测试 TiDB 集群之前，请确保已经安装 MySQL 客户端。从 Pod 启动、运行到服务可以访问有一些延时，可以通过下面命令查看服务：

{{< copyable "shell-regular" >}}

``` shell
kubectl get svc -n demo --watch
```

如果看到 `basic-tidb` 出现，说明服务已经可以访问，可以 <kbd>Ctrl</kbd>+<kbd>C</kbd> 停止。

按照以下步骤访问 TiDB 集群：

1. 转发本地端口到 TiDB 端口。

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl -n demo port-forward svc/basic-tidb 4000:4000
    ```

2. 在另一个终端窗口中，通过 MySQL 客户端访问 TiDB：

    {{< copyable "shell-regular" >}}

    ``` shell
    mysql -h 127.0.0.1 -P 4000 -uroot
    ```

    或者可以直接执行 SQL 命令：

    {{< copyable "shell-regular" >}}

    ``` shell
    mysql -h 127.0.0.1 -P 4000 -uroot -e 'select tidb_version();'
    ```

> **注意：**
>
> TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见[遥测](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

## 监控 TiDB 集群

按照以下步骤监控 TiDB 集群状态：

1. 转发本地端口到 Grafana 端口。

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl -n demo port-forward svc/basic-grafana 3000:3000
    ```

2. 打开浏览器，通过 `http://localhost:3000` 访问 Grafana。

    或者，Minikube 提供了 `minikube service` 的方式暴露 Grafana 服务，可以更方便的接入。

    {{< copyable "shell-regular" >}}

    ``` shell
    minikube service basic-grafana -n demo
    ```

    上述命令会自动搭建代理并在浏览器中打开 Grafana。

### 删除 TiDB 集群

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

更新 demo 集群使用的 PV 的 reclaim 策略为 Delete：

{{< copyable "shell-regular" >}}

``` shell
kubectl get pv -l app.kubernetes.io/instance=basic -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

删除 PVC：

{{< copyable "shell-regular" >}}

``` shell
kubectl delete pvc -l app.kubernetes.io/managed-by=tidb-operator
```

## FAQ

### Minikube 上的 TiDB 集群不响应或者响应非常慢

Minikube 虚拟机默认配置为 2048 MB 内存和 2 个 CPU。可以在 `minikube start` 时通过`--memory` 和 `--cpus` 选项为其分配更多资源。注意，为了使配置修改生效，你需要重新创建 Minikube 虚拟机。

{{< copyable "shell-regular" >}}

``` shell
minikube delete && \
minikube start --cpus 4 --memory 4096 ...
```
