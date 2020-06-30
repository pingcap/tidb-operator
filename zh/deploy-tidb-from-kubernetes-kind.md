---
title: 使用 kind 在 Kubernetes 上部署 TiDB 集群
summary: 介绍如何使用 kind 在 Kubernetes 上部署 TiDB 集群。
category: how-to
aliases: ['/docs-cn/devget-started/deploy-tidb-from-kubernetes-dind/']
---

# 使用 kind 在 Kubernetes 上部署 TiDB 集群

本文介绍了如何在个人电脑（Linux 或 MacOS）上采用 [kind](https://kind.sigs.k8s.io/) 方式在 Kubernetes 上部署 [TiDB Operator](https://github.com/pingcap/tidb-operator) 和 TiDB 集群。

kind 通过使用 Docker 容器作为集群节点模拟出一个本地的 Kubernetes 集群。kind 的设计初衷是为了在本地进行 Kubernetes 集群的一致性测试。Kubernetes 集群版本取决于 kind 使用的镜像，你可以指定任一镜像版本用于集群节点，并在 [Docker hub](https://hub.docker.com/r/kindest/node/tags) 中找到想要部署的 Kubernetes 版本。

> **警告：**
>
> 对于生产环境，不要使用此方式进行部署。

## 环境准备

- [docker](https://docs.docker.com/install/)：版本 >= 17.03
- [helm](https://helm.sh/docs/intro/install/)：helm 2 或 helm 3 最新稳定版
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl)：版本 >= 1.12
- [kind](https://kind.sigs.k8s.io/)：版本 >= 0.7.0 建议最新版
- [net.ipv4.ip_forward](https://linuxconfig.org/how-to-turn-on-off-ip-forwarding-in-linux) 需要被设置为 `1`

## 第 1 步：通过 kind 部署 Kubernetes 集群

参考 `kind` [快速使用文档](https://kind.sigs.k8s.io/docs/user/quick-start)安装 `kind` 并创建一个集群。

以下以 0.8.1 版本为例：

```
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.8.1/kind-$(uname)-amd64
chmod +x ./kind
./kind create cluster
```

测试集群是否创建成功：

```
kubectl cluster-info
```

## 第 2 步：部署 TiDB Operator

参考 [Helm 安装文档](https://helm.sh/docs/intro/install/) 安装 Helm 或使用以下命令安装最新 Helm 3 稳定版：

{{< copyable "shell-regular" >}}

```shell
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
```

TiDB Operator 使用 [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) 扩展 Kubernetes，所以要使用 TiDB Operator，必须先创建 `TidbCluster` 等各种自定义资源类型：

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.0/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

安装 TiDB Operator，以下使用 Helm 3 为例：

{{< copyable "shell-regular" >}}

```shell
helm repo add pingcap https://charts.pingcap.org/
kubectl create ns pingcap
helm install --namespace pingcap tidb-operator pingcap/tidb-operator --version v1.1.0
```

若是使用 Helm 2，参考[安装 Helm](tidb-toolkit.md#使用-helm) 初始化 Helm 后，使用以下命令部署 TiDB Operator：

{{< copyable "shell-regular" >}}

```shell
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace pingcap --name tidb-operator pingcap/tidb-operator --version v1.1.0
```

查看 TiDB Operator 运行状态：

{{< copyable "shell-regular" >}}

```shell
kubectl -n pingcap get pods
```

## 第 3 步：部署 TiDB 集群

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

## 访问数据库和监控面板

通过 `kubectl port-forward` 暴露服务到主机，可以访问 TiDB 集群。命令中的端口格式为：`<主机端口>:<k8s 服务端口>`。

- 通过 MySQL 客户端访问 TiDB

    在访问 TiDB 集群之前，请确保已安装 MySQL client。

    1. 使用 kubectl 暴露 TiDB 服务端口：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl port-forward svc/basic-tidb 4000:4000 --namespace=demo
        ```

        > **注意：**
        >
        > 如果代理建立成功，会打印类似输出：`Forwarding from 0.0.0.0:4000 -> 4000`。测试完成后按 `Ctrl + C` 停止代理并退出。

    2. 然后，通过 MySQL 客户端访问 TiDB，打开一个**新**终端标签或窗口，执行下面的命令：

        {{< copyable "shell-regular" >}}

        ``` shell
        mysql -h 127.0.0.1 -P 4000 -u root
        ```

- 查看监控面板

    1. 使用 kubectl 暴露 Grafana 服务端口：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl port-forward svc/basic-grafana 3000:3000 --namespace=demo
        ```

        > **注意：**
        >
        > 如果代理建立成功，会打印类似输出：`Forwarding from 0.0.0.0:3000 -> 3000`。测试完成后按 `Ctrl + C` 停止代理并退出。

    2. 然后，在浏览器中打开 <http://localhost:3000> 访问 Grafana 监控面板：

        - 默认用户名：admin
        - 默认密码：admin

    > **注意：**
    >
    > 如果你不是在本地 PC 而是在远程主机上部署的 kind 环境，可能无法通过 localhost 访问远程主机的服务。
    >
    > 如果使用 kubectl 1.13 或者更高版本，可以在执行 `kubectl port-forward` 命令时添加 `--address 0.0.0.0` 选项，在 `0.0.0.0` 暴露端口而不是默认的 `127.0.0.1`：
    >
    > {{< copyable "shell-regular" >}}
    >
    > ```
    > kubectl port-forward --address 0.0.0.0 -n tidb svc/basic-grafana 3000:3000
    > ```
    >
    > 然后，在浏览器中打开 `http://<远程主机 IP>:3000` 访问 Grafana 监控面板。

> **注意：**
>
> TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见[遥测](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

## 删除 TiDB 集群 与 Kubernetes 集群

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

通过下面命令删除该 Kubernetes 集群：

{{< copyable "shell-regular" >}}

``` shell
kind delete cluster
```
