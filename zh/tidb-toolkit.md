---
title: Kubernetes 上的 TiDB 工具指南
summary: 详细介绍 Kubernetes 上的 TiDB 相关的工具及其使用方法。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/tidb-toolkit/']
---

# Kubernetes 上的 TiDB 工具指南

Kubernetes 上的 TiDB 运维管理需要使用一些开源工具。同时，在 Kubernetes 上使用 TiDB 生态工具时，也有特殊的操作要求。本文档详细描述 Kubernetes 上的 TiDB 相关的工具及其使用方法。

## 在 Kubernetes 上使用 PD Control

[PD Control](https://pingcap.com/docs-cn/stable/pd-control/) 是 PD 的命令行工具，在使用 PD Control 操作 Kubernetes 上的 TiDB 集群时，需要先使用 `kubectl port-forward` 打开本地到 PD 服务的连接：

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379 &>/tmp/portforward-pd.log &
```

执行上述命令后，就可以通过 `127.0.0.1:2379` 访问到 PD 服务，从而直接使用 `pd-ctl` 命令的默认参数执行操作，如：

{{< copyable "shell-regular" >}}

```shell
pd-ctl -d config show
```

假如你本地的 2379 被占据，则需要选择其它端口：

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-pd ${local_port}:2379 &>/tmp/portforward-pd.log &
```

此时，需要为 `pd-ctl` 命令显式指定 PD 端口：

{{< copyable "shell-regular" >}}

```shell
pd-ctl -u 127.0.0.1:${local_port} -d config show
```

## 在 Kubernetes 上使用 TiKV Control

[TiKV Control](https://pingcap.com/docs-cn/stable/tikv-control/) 是 TiKV 的命令行工具。在使用 TiKV Control 操作 Kubernetes 上的 TiDB 集群时，针对 TiKV Control 的不同操作模式，有不同的操作步骤。

* **远程模式**：此模式下 `tikv-ctl` 命令需要通过网络访问 TiKV 服务或 PD 服务，因此需要先使用 `kubectl port-forward` 打开本地到 PD 服务以及目标 TiKV 节点的连接：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379 &>/tmp/portforward-pd.log &
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl port-forward -n ${namespace} ${pod_name} 20160:20160 &>/tmp/portforward-tikv.log &
    ```

    打开连接后，即可通过本地的对应端口访问 PD 服务和 TiKV 节点：

    {{< copyable "shell-regular" >}}

    ```shell
    $ tikv-ctl --host 127.0.0.1:20160 ${subcommands}
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    tikv-ctl --pd 127.0.0.1:2379 compact-cluster
    ```

* **本地模式**：本地模式需要访问 TiKV 的数据文件，并且需要停止正在运行的 TiKV 实例。需要先使用[诊断模式](tips.md#诊断模式)关闭 TiKV 实例自动重启，关闭 TiKV 进程，再使用 `tkctl debug` 命令在目标 TiKV Pod 中启动一个包含 `tikv-ctl` 可执行文件的新容器来执行操作，步骤如下：

    1. 进入诊断模式：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl annotate pod ${pod_name} -n ${namespace} runmode=debug
        ```

    2. 关闭 TiKV 进程：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl exec ${pod_name} -n ${namespace} -c tikv -- kill -s TERM 1
        ```

    3. 启动 debug 容器：

        {{< copyable "shell-regular" >}}

        ```shell
        tkctl debug ${pod_name} -c tikv
        ```

    4. 开始使用 `tikv-ctl` 的本地模式，需要注意的是 `tikv` 容器的根文件系统在 `/proc/1/root` 下，因此执行命令时也需要调整数据目录的路径：

        {{< copyable "shell-regular" >}}

        ```shell
        tikv-ctl --db /path/to/tikv/db size -r 2
        ```

        Kubernetes 上 TiKV 实例在 debug 容器中的的默认 db 路径是 `/proc/1/root/var/lib/tikv/db size -r 2`

## 在 Kubernetes 上使用 TiDB Control

[TiDB Control](https://pingcap.com/docs-cn/stable/tidb-control/) 是 TiDB 的命令行工具，使用 TiDB Control 时，需要从本地访问 TiDB 节点和 PD 服务，因此建议使用 `kubectl port-forward` 打开到集群中 TiDB 节点和 PD 服务的连接：

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379 &>/tmp/portforward-pd.log &
```

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} ${pod_name} 10080:10080 &>/tmp/portforward-tidb.log &
```

接下来便可开始使用 `tidb-ctl` 命令：

{{< copyable "shell-regular" >}}

```shell
tidb-ctl schema in mysql
```

## 使用 Helm

[Helm](https://helm.sh/) 是一个 Kubernetes 的包管理工具，确保安装的 Helm 版本为 >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797)。安装步骤如下：

### 安装 Helm 客户端

参考[官方文档](https://v2.helm.sh/docs/using_helm/#installing-helm)安装 Helm 客户端。

如果服务器没有外网，需要先将 Helm 客户端在有外网的机器上下载下来，然后再拷贝到服务器上，这里以安装 Helm 客户端 `2.16.7` 为例：

{{< copyable "shell-regular" >}}

```shell
wget https://get.helm.sh/helm-v2.16.7-linux-amd64.tar.gz
tar zxvf helm-v2.16.7-linux-amd64.tar.gz
```

解压之后，有以下文件：

```shell
linux-amd64/
linux-amd64/README.md
linux-amd64/tiller
linux-amd64/helm
linux-amd64/LICENSE
```

请自行将 `linux-amd64/helm` 文件拷贝到服务器上，并将其放到 `/usr/local/bin/` 目录下即可。

然后执行 `helm verison -c`，如果正常输出则表示 Helm 客户端安装成功：

{{< copyable "shell-regular" >}}

```shell
helm version -c
```

```shell
Client: &version.Version{SemVer:"v2.16.7", GitCommit:"5f2584fd3d35552c4af26036f0c464191287986b", GitTreeState:"clean"}
```

### 安装 Helm 服务端

#### 安装 RBAC

如果 Kubernetes 集群没有启用 `RBAC`，请跳过此小节，直接安装 Tiller 即可。

Helm 服务端是一个名字叫 `tiller` 的服务, 请首先安装 `tiller` 所需的 `RBAC` 规则：

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.5/manifests/tiller-rbac.yaml
```

如果服务器没有外网，需要先用有外网的机器下载 `tiller-rbac.yaml` 文件：

{{< copyable "shell-regular" >}}

```shell
wget https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.5/manifests/tiller-rbac.yaml
```

将 `tiller-rbac.yaml` 文件拷贝到服务器上并安装 `RBAC`：

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f tiller-rbac.yaml
```

#### 安装 Tiller

Helm 服务端是一个名字叫 `tiller` 的服务，是作为一个 Pod 运行在 Kubernetes 集群里的。使用下面的命令安装 `tiller`：

{{< copyable "shell-regular" >}}

```shell
helm init --service-account=tiller --upgrade
```

`tiller` 这个Pod 使用的镜像是 `gcr.io/kubernetes-helm/tiller:v2.16.7`，如果服务器无法访问 gcr.io，你可以尝试 mirror 仓库：

{{< copyable "shell-regular" >}}

``` shell
helm init --service-account=tiller --upgrade --tiller-image registry.cn-hangzhou.aliyuncs.com/google_containers/tiller:$(helm version --client --short | grep -Eo 'v[0-9]\.[0-9]+\.[0-9]+')
```

如果服务器没有外网，需要先将 `tiller` 所使用的 Docker 镜像在有外网的机器下载下来：

{{< copyable "shell-regular" >}}

``` shell
docker pull gcr.io/kubernetes-helm/tiller:v2.16.7
docker save -o tiller-v2.16.7.tar gcr.io/kubernetes-helm/tiller:v2.16.7
```

将 `tiller-v2.16.7.tar` 文件拷贝到服务器上，执行 `docker load` 命令将其 load 到服务器上：

{{< copyable "shell-regular" >}}

``` shell
docker load -i tiller-v2.16.7.tar
```

最后通过下面命令安装 `tiller` 并确认 `tiller` Pod 进入 Running 状态：

{{< copyable "shell-regular" >}}

```shell
helm init --service-account=tiller --skip-refresh
kubectl get po -n kube-system -l name=tiller
```

#### 配置 Helm repo

Kubernetes 应用在 Helm 中被打包为 chart。PingCAP 针对 Kubernetes 上的 TiDB 部署运维提供了多个 Helm chart：

* `tidb-operator`：用于部署 TiDB Operator；
* `tidb-cluster`：用于部署 TiDB 集群；
* `tidb-backup`：用于 TiDB 集群备份恢复；
* `tidb-lightning`：用于 TiDB 集群导入数据；
* `tidb-drainer`：用于部署 TiDB Drainer；
* `tikv-importer`：用于部署 TiKV Importer；

这些 chart 都托管在 PingCAP 维护的 helm chart 仓库 `https://charts.pingcap.org/` 中，你可以通过下面的命令添加该仓库：

{{< copyable "shell-regular" >}}

```shell
helm repo add pingcap https://charts.pingcap.org/
```

添加完成后，可以使用 `helm search` 搜索 PingCAP 提供的 chart：

- 如果 Helm 版本 < 2.16.0:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l
    ```

- 如果 Helm 版本 >= 2.16.0:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l --devel
    ```

    ```
    NAME                    CHART VERSION   APP VERSION DESCRIPTION
    pingcap/tidb-backup     v1.0.0                      A Helm chart for TiDB Backup or Restore
    pingcap/tidb-cluster    v1.0.0                      A Helm chart for TiDB Cluster
    pingcap/tidb-operator   v1.0.0                      tidb-operator Helm chart for Kubernetes
    ...
    ```

当新版本的 chart 发布后，你可以使用 `helm repo update` 命令更新本地对于仓库的缓存：

{{< copyable "shell-regular" >}}

```
helm repo update
```

#### Helm 常用操作

Helm 的常用操作有部署（`helm install`）、升级（`helm upgrade`)、销毁（`helm del`)、查询（`helm ls`）。Helm chart 往往都有很多可配置参数，通过命令行进行配置比较繁琐，因此推荐使用 YAML 文件的形式来编写这些配置项。基于 Helm 社区约定俗称的命名方式，在文档中将用于配置 chart 的 YAML 文件称为 `values.yaml` 文件。

执行部署、升级、销毁等操作前，可以使用 `helm ls` 查看集群中已部署的应用：

{{< copyable "shell-regular" >}}

```shell
helm ls
```

在执行部署和升级操作时，必须指定使用的 chart 名字（`chart-name`）和部署后的应用名（`release-name`），还可以指定一个或多个 `values.yaml` 文件来配置 chart。此外，假如对 chart 有特定的版本需求，则需要通过 `--version` 参数指定 `chart-version` (默认为最新的 GA 版本）。命令形式如下：

* 执行安装：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${chart_name} --name=${release_name} --namespace=${namespace} --version=${chart_version} -f ${values_file}
    ```

* 执行升级（升级可以是修改 `chart-version` 升级到新版本的 chart，也可以是修改 `values.yaml` 文件更新应用配置）：

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade ${release_name} ${chart_name} --version=${chart_version} -f ${values_file}
    ```

最后，假如要删除 helm 部署的应用，可以执行：

{{< copyable "shell-regular" >}}

```shell
helm del --purge ${release_name}
```

更多 helm 的相关文档，请参考 [Helm 官方文档](https://helm.sh/docs/)。

#### 离线情况下使用 Helm chart

如果服务器上没有外网，就无法通过配置 Helm repo 来安装 TiDB Operator 组件以及其他应用。这时，需要在有外网的机器上下载集群安装需用到的 chart 文件，再拷贝到服务器上。

通过以下命令，下载集群安装时需要的 chart 文件：

{{< copyable "shell-regular" >}}

```shell
wget http://charts.pingcap.org/tidb-operator-v1.1.5.tgz
wget http://charts.pingcap.org/tidb-drainer-v1.1.5.tgz
wget http://charts.pingcap.org/tidb-lightning-v1.1.5.tgz
```

将这些 chart 文件拷贝到服务器上并解压，可以通过 `helm install` 命令使用这些 chart 来安装相应组件，以 `tidb-operator` 为例：

{{< copyable "shell-regular" >}}

```shell
tar zxvf tidb-operator.v1.1.5.tgz
helm install ./tidb-operator --name=${release_name} --namespace=${namespace}
```

## 使用 Terraform

[Terraform](https://www.terraform.io/) 是一个基础设施即代码（Infrastructure as Code）管理工具。它允许用户使用声明式的风格描述自己的基础设施，并针对描述生成执行计划来创建或调整真实世界的计算资源。Kubernetes 上的 TiDB 使用 Terraform 来在公有云上创建和管理 TiDB 集群。

你可以参考 [Terraform 官方文档](https://www.terraform.io/downloads.html) 来安装 Terraform。
