---
title: 在 GCP GKE 上部署 TiDB 集群
summary: 了解如何在 GCP GKE 上部署 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-on-gcp-gke/']
---

# 在 GCP GKE 上部署 TiDB 集群

<!-- markdownlint-disable MD029 -->

本文介绍了如何部署 GCP Google Kubernetes Engine (GKE) 集群，并在其中部署 TiDB 集群。

如果需要部署 TiDB Operator 及 TiDB 集群到自托管 Kubernetes 环境，请参考[部署 TiDB Operator](deploy-tidb-operator.md)及[部署 TiDB 集群](deploy-on-general-kubernetes.md)等文档。

## 环境准备

部署前，请确认已完成以下环境准备：

* [Helm 3](https://helm.sh/docs/intro/install/)：用于安装 TiDB Operator
* [gcloud](https://cloud.google.com/sdk/gcloud)：用于创建和管理 GCP 服务的命令行工具
* 完成 [GKE 快速入门](https://cloud.google.com/kubernetes-engine/docs/quickstart#before-you-begin) 中的**准备工作** (Before you begin)

    该教程包含以下内容：

    * 启用 Kubernetes API
    * 配置足够的配额等

## 配置 GCP 服务

{{< copyable "shell-regular" >}}

```shell
gcloud config set core/project <gcp-project>
gcloud config set compute/region <gcp-region>
```

使用以上命令，设置好你的 GCP 项目和默认的区域。

## 创建 GKE 集群和节点池

1. 创建 GKE 集群和一个默认节点池：

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container clusters create tidb --region us-east1 --machine-type n1-standard-4 --num-nodes=1
    ```

    该命令创建一个区域 (regional) 集群，在该集群模式下，节点会在该区域中分别创建三个可用区 (zone)，以保障高可用。`--num-nodes=1` 参数，表示在各分区各自创建一个节点，总节点数为 3 个。生产环境推荐该集群模式。其他集群类型，可以参考 [GKE 集群的类型](https://cloud.google.com/kubernetes-engine/docs/concepts/types-of-clusters)。

    以上命令集群创建在默认网络中，若希望创建在指定的网络中，通过 `--network/subnet` 参数指定。更多可查询 [GKE 集群创建文档](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-regional-cluster)。

2. 分别为 PD、TiKV 和 TiDB 创建独立的节点池：

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container node-pools create pd --cluster tidb --machine-type n1-standard-4 --num-nodes=1 \
        --node-labels=dedicated=pd --node-taints=dedicated=pd:NoSchedule
    gcloud container node-pools create tikv --cluster tidb --machine-type n1-highmem-8 --num-nodes=1 \
        --node-labels=dedicated=tikv --node-taints=dedicated=tikv:NoSchedule
    gcloud container node-pools create tidb --cluster tidb --machine-type n1-standard-8 --num-nodes=1 \
        --node-labels=dedicated=tidb --node-taints=dedicated=tidb:NoSchedule
    ```

此过程可能需要几分钟。

## 部署 TiDB Operator

参考快速上手中[部署 TiDB Operator](get-started.md#部署-tidb-operator)，在 GKE 集群中部署 TiDB Operator。

## 部署 TiDB 集群和监控

下面介绍如何在 GCP GKE 上部署 TiDB 集群和监控组件。

### 创建 namespace

执行以下命令，创建 TiDB 集群安装的 namespace：

{{< copyable "shell-regular" >}}

```shell
kubectl create namespace tidb-cluster
```

> **注意：**
>
> 这里创建的 namespace 是指 [Kubernetes 命名空间](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)。本文档使用 `tidb-cluster` 为例，若使用了其他名字，修改相应的 `-n` 或 `--namespace` 参数为对应的名字即可。

### 部署 TiDB 集群

首先执行以下命令，下载 TidbCluster 和 TidbMonitor CR 的配置文件。

{{< copyable "shell-regular" >}}

```shell
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/gcp/tidb-cluster.yaml &&
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/gcp/tidb-monitor.yaml
```

如需了解更详细的配置信息或者进行自定义配置，请参考[配置 TiDB 集群](configure-a-tidb-cluster.md)

执行以下命令，在 GKE 集群中部署 TidbCluster 和 TidbMonitor CR。

{{< copyable "shell-regular" >}}

```shell
kubectl create -f tidb-cluster.yaml -n tidb-cluster && \
kubectl create -f tidb-monitor.yaml -n tidb-cluster
```

当上述 yaml 文件被应用到 Kubernetes 集群后，TiDB Operator 会负责根据 yaml 文件描述，创建对应配置的 TiDB 集群。

> **注意：**
>
> 如果要将 TiDB 集群部署到 ARM64 机器上，可以参考[在 ARM64 机器上部署 TiDB 集群](deploy-cluster-on-arm64.md)。

### 查看 TiDB 集群启动状态

使用以下命令查看 TiDB 集群启动状态：

{{< copyable "shell-regular" >}}

```shell
kubectl get pods -n tidb-cluster
```

当所有 pods 都处于 Running & Ready 状态时，则可以认为 TiDB 集群已经成功启动。如下是一个正常运行的 TiDB 集群的示例输出：

```
NAME                              READY   STATUS    RESTARTS   AGE
tidb-discovery-5cb8474d89-n8cxk   1/1     Running   0          47h
tidb-monitor-6fbcc68669-dsjlc     3/3     Running   0          47h
tidb-pd-0                         1/1     Running   0          47h
tidb-pd-1                         1/1     Running   0          46h
tidb-pd-2                         1/1     Running   0          46h
tidb-tidb-0                       2/2     Running   0          47h
tidb-tidb-1                       2/2     Running   0          46h
tidb-tikv-0                       1/1     Running   0          47h
tidb-tikv-1                       1/1     Running   0          47h
tidb-tikv-2                       1/1     Running   0          47h
```

## 访问数据库

### 准备一台堡垒机

我们为 TiDB 集群创建的是内网 LoadBalancer。我们可在集群 VPC 内创建一台[堡垒机](https://cloud.google.com/solutions/connecting-securely#bastion)来访问数据库。

{{< copyable "shell-regular" >}}

```shell
gcloud compute instances create bastion \
    --machine-type=n1-standard-4 \
    --image-project=centos-cloud \
    --image-family=centos-7 \
    --zone=${your-region}-a
```

> **注意：**
>
> `${your-region}-a` 为集群所在的区域的 a 可用区，比如 us-central1-a。也可在同区域下的其他可用区创建堡垒机。

### 安装 MySQL 客户端并连接

待创建好堡垒机后，我们可以通过 SSH 远程连接到堡垒机，再通过 MySQL 客户端来访问 TiDB 集群。

1. 用 SSH 连接到堡垒机：

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud compute ssh tidb@bastion
    ```

2. 安装 MySQL 客户端：

    {{< copyable "shell-regular" >}}

    ```shell
    sudo yum install mysql -y
    ```

3. 连接到 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ```shell
    mysql --comments -h ${tidb-nlb-dnsname} -P 4000 -u root
    ```

    `${tidb-nlb-dnsname}` 为 TiDB Service 的 LoadBalancer IP，可以通过 `kubectl get svc basic-tidb -n tidb-cluster` 输出中的 `EXTERNAL-IP` 字段查看。

    示例：

    ```shell
    $ mysql --comments -h 10.128.15.243 -P 4000 -u root
    Welcome to the MariaDB monitor.  Commands end with ; or \g.
    Your MySQL connection id is 7823
    Server version: 5.7.25-TiDB-v5.2.1 TiDB Server (Apache License 2.0) Community Edition, MySQL 5.7 compatible

    Copyright (c) 2000, 2018, Oracle, MariaDB Corporation Ab and others.

    Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

    MySQL [(none)]> show status;
    +--------------------+--------------------------------------+
    | Variable_name      | Value                                |
    +--------------------+--------------------------------------+
    | Ssl_cipher         |                                      |
    | Ssl_cipher_list    |                                      |
    | Ssl_verify_mode    | 0                                    |
    | Ssl_version        |                                      |
    | ddl_schema_version | 22                                   |
    | server_id          | 717420dc-0eeb-4d4a-951d-0d393aff295a |
    +--------------------+--------------------------------------+
    6 rows in set (0.01 sec)
    ```

    > **注意：**
    >
    > * [MySQL 8.0 默认认证插件](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin)从 `mysql_native_password` 更新为 `caching_sha2_password`，因此如果使用 MySQL 8.0 客户端访问 TiDB 服务（TiDB 版本 < v4.0.7），并且用户账户有配置密码，需要显式指定 `--default-auth=mysql_native_password` 参数。
    > * TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见[遥测](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

### 访问 Grafana 监控

先获取 Grafana 的 LoadBalancer 域名：

{{< copyable "shell-regular" >}}

```shell
kubectl -n tidb-cluster get svc basic-grafana
```

示例：

```
$ kubectl -n tidb-cluster get svc basic-grafana
NAME                     TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)               AGE
basic-grafana            LoadBalancer   10.15.255.169   34.123.168.114   3000:30657/TCP        35m
```

其中 `EXTERNAL-IP` 栏即为 LoadBalancer IP。

你可以通过浏览器访问 `${grafana-lb}:3000` 地址查看 Grafana 监控指标。其中 `${grafana-lb}` 替换成前面获取的 IP。

> **注意：**
>
> Grafana 默认用户名和密码均为 admin。

## 升级 TiDB 集群

要升级 TiDB 集群，可以通过 `kubectl edit tc basic -n tidb-cluster` 命令修改 `spec.version`。

升级过程会持续一段时间，你可以通过 `kubectl get pods -n tidb-cluster --watch` 命令持续观察升级进度。

## 扩容 TiDB 集群

扩容前需要对相应的节点组进行扩容，以便新的实例有足够的资源运行。以下展示扩容 GKE 节点组和 TiDB 集群组件的操作。

### 扩容 GKE 节点组

下面是将 GKE 集群 `tidb` 的 `tikv` 节点池扩容到 6 节点的示例：

{{< copyable "shell-regular" >}}

```shell
gcloud container clusters resize tidb --node-pool tikv --num-nodes 2
```

> **注意：**
>
> 在区域集群下，节点分别创建在 3 个可用区下。这里扩容后，节点数为 2 * 3 = 6 个。

### 扩容 TiDB 组件

然后通过 `kubectl edit tc basic -n tidb-cluster` 修改各组件的 `replicas` 为期望的新副本数进行扩容。

更多节点池管理可参考 [Node Pools 文档](https://cloud.google.com/kubernetes-engine/docs/concepts/node-pools)。

## 部署 TiFlash/TiCDC

[TiFlash](https://docs.pingcap.com/zh/tidb/stable/tiflash-overview) 是 TiKV 的列存扩展，[TiCDC](https://docs.pingcap.com/zh/tidb/stable/ticdc-overview) 是一款通过拉取 TiKV 变更日志实现的 TiDB 增量数据同步工具。这两个组件不是必选安装项，这里提供一个快速安装上手示例。

### 新增节点组

为 TiFlash 新增节点组：

{{< copyable "shell-regular" >}}

```shell
gcloud container node-pools create tiflash --cluster tidb --machine-type n1-highmem-8 --num-nodes=1 \
    --node-labels dedicated=tiflash --node-taints dedicated=tiflash:NoSchedule
```

为 TiCDC 新增节点组：

{{< copyable "shell-regular" >}}

```shell
gcloud container node-pools create ticdc --cluster tidb --machine-type n1-standard-4 --num-nodes=1 \
    --node-labels dedicated=ticdc --node-taints dedicated=ticdc:NoSchedule
```

### 配置并部署 TiFlash/TiCDC

如果要部署 TiFlash，可以在 tidb-cluster.yaml 中配置 `spec.tiflash`，例如：

```yaml
spec:
  ...
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 0
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
    nodeSelector:
      dedicated: tiflash
    tolerations:
    - effect: NoSchedule
      key: dedicated
      operator: Equal
      value: tiflash
```

其他参数可以参考[TiDB 集群配置文档](configure-a-tidb-cluster.md)进行配置。

> **警告：**
>
> 由于 TiDB Operator 会按照 `storageClaims` 列表中的配置**按顺序**自动挂载 PV，如果需要为 TiFlash 增加磁盘，请确保只在列表原有配置**末尾添加**，并且**不能**修改列表中原有配置的顺序。

如果要部署 TiCDC，可以在 tidb-cluster.yaml 中配置 `spec.ticdc`，例如：

```yaml
spec:
  ...
  ticdc:
    baseImage: pingcap/ticdc
    replicas: 1
    nodeSelector:
      dedicated: ticdc
    tolerations:
    - effect: NoSchedule
      key: dedicated
      operator: Equal
      value: ticdc
```

根据实际情况修改 `replicas` 等参数。

最后使用 `kubectl -n tidb-cluster apply -f tidb-cluster.yaml` 更新 TiDB 集群配置。

更多可参考 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)和[集群配置文档](configure-a-tidb-cluster.md)完成 CR 文件配置。

## 使用企业版

部署企业版 TiDB、PD、TiKV、TiFlash、TiCDC 时，只需要将 tidb-cluster.yaml 中 `spec.[tidb|pd|tikv|tiflash|ticdc].baseImage` 配置为企业版镜像，格式为 `pingcap/[tidb|pd|tikv|tiflash|ticdc]-enterprise`。

例如:

```yaml
spec:
  ...
  pd:
    baseImage: pingcap/pd-enterprise
  ...
  tikv:
    baseImage: pingcap/tikv-enterprise
```

## 使用本地存储

请使用[区域永久性磁盘](https://cloud.google.com/compute/docs/disks#pdspecs)作为生产环境的存储类型。如果需要模拟测试裸机部署的性能，可以使用 GCP 部分实例类型提供的[本地存储卷](https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/local-ssd)。可以为 TiKV 节点池选择这一类型的实例，以便提供更高的 IOPS 和低延迟。

> **警告：**
>
> 运行中的 TiDB 集群不能动态更换 storage class，可创建一个新的 TiDB 集群测试。
>
> 由于 GKE 升级过程中节点重建，[本地盘数据会丢失](https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/local-ssd)。由于 GKE 升级或其他原因造成的节点重建，会导致需要迁移 TiKV 数据，如果无法接受这一点，则不建议在生产环境中使用本地盘。

1. 为 TiKV 创建附带本地存储的节点组。

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container node-pools create tikv --cluster tidb --machine-type n1-standard-4 --num-nodes=1 --local-ssd-count 1 \
      --node-labels dedicated=tikv --node-taints dedicated=tikv:NoSchedule
    ```

    若 tikv 组已存在，可先删除再创建，或者修改名字规避名字冲突。

2. 部署 local volume provisioner。

    本地存储需要使用 [local-volume-provisioner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner) 程序发现并管理。以下命令会部署并创建一个 `local-storage` 的 Storage Class。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/gke/local-ssd-provision/local-ssd-provision.yaml
    ```

3. 使用本地存储。

    完成前面步骤后，local-volume-provisioner 即可发现集群内所有本地 SSD 盘。在 tidb-cluster.yaml 中添加 `tikv.storageClassName` 字段并设置为 `local-storage` 即可。
