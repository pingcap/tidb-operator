---
title: 在 AWS EKS 上部署 TiDB
summary: 介绍如何在 AWS EKS (Elastic Kubernetes Service) 上部署 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-on-aws-eks/']
---

# 在 AWS EKS 上部署 TiDB 集群

本文介绍了如何在 AWS EKS (Elastic Kubernetes Service) 上部署 TiDB 集群。

## 环境配置准备

部署前，请确认已完成以下环境准备：

* 安装 [Helm](https://helm.sh/docs/intro/install/)：用于安装 TiDB Operator。

* 完成 AWS [eksctl 入门](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/getting-started-eksctl.html) 中所有操作。

该教程包含以下内容：

* 安装并配置 AWS 的命令行工具 awscli
* 安装并配置创建 Kubernetes 集群的命令行工具 eksctl
* 安装 Kubernetes 命令行工具 kubectl

> **注意：**
>
> 本文档的操作需要 AWS Access Key 至少具有 [eksctl 所需最少权限](https://eksctl.io/usage/minimum-iam-policies/) 和创建 [Linux 堡垒机所涉及的服务权限](https://docs.aws.amazon.com/quickstart/latest/linux-bastion/architecture.html#aws-services)。

## 部署

### 创建 EKS 和节点池

{{< copyable "shell-regular" >}}

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: <clusterName>
  region: us-west-2

nodeGroups:
  - name: admin
    desiredCapacity: 1
    labels:
      dedicated: admin

  - name: tidb
    desiredCapacity: 2
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule

  - name: pd
    desiredCapacity: 3
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule

  - name: tikv
    desiredCapacity: 3
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
```

将以上配置存为 cluster.yaml 文件，并替换 `<clusterName>` 为自己想命名的集群名字后，执行以下命令创建集群：

{{< copyable "shell-regular" >}}

```shell
eksctl create cluster -f cluster.yaml
```

> **注意：**
>
> - 该命令需要等待 EKS 集群创建完成，以及节点组创建完成并加入进去，耗时 5 到 10 分钟不等。
> - 可参考 [eksctl 文档](https://eksctl.io/usage/creating-and-managing-clusters/#using-config-files)了解更多集群配置选项。

### 部署 TiDB Operator

参考快速上手中[部署 TiDB Operator](get-started.md#部署-tidb-operator)，将 TiDB Operator 部署进 Kubernetes 集群。

### 部署 TiDB 集群和监控

1. 准备 TidbCluster 和 TidbMonitor CR 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    curl -LO https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-cluster.yaml &&
    curl -LO https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-monitor.yaml
    ```

2. 创建 `Namespace`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create namespace tidb-cluster
    ```

    > **注意：**
    >
    > `namespace` 是[命名空间](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)。本文档使用 `tidb-cluster` 为例，若使用了其他名字，修改相应的 `-n` 或 `--namespace` 参数为对应的名字即可。

3. 部署 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create -f tidb-cluster.yaml -n tidb-cluster &&
    kubectl create -f tidb-monitor.yaml -n tidb-cluster
    ```

4. 查看 TiDB 集群启动状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pods -n tidb-cluster
    ```

    当所有 pods 都处于 Running & Ready 状态时，则可以认为 TiDB 集群已经成功启动。一个正常运行的 TiDB 集群的案例：

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

### 准备一台可以访问集群的机器

我们为 TiDB 集群创建的是内网 LoadBalancer。我们可在集群 VPC 内创建一台[堡垒机](https://aws.amazon.com/quickstart/architecture/linux-bastion/)访问数据库，参考 [AWS Linux 堡垒机文档](https://aws.amazon.com/quickstart/architecture/linux-bastion/)在 AWS Console 上创建即可。

VPC 和 Subnet 需选择集群的 VPC 和 Subnet，在下拉框通过集群名字确认是否正确。可以通过以下命令查看集群的 VPC 和 Subnet 来验证：

{{< copyable "shell-regular" >}}

```shell
eksctl get cluster -n <clusterName>
```

同时需允许本机网络访问，并选择正确的 Key Pair 以便能通过 SSH 登录机器。

> **注意：**
>
> - 除使用堡垒机以外，也可以使用 [VPC Peering](https://docs.aws.amazon.com/vpc/latest/peering/what-is-vpc-peering.html) 连接现有机器到集群 VPC。
> - 若 EKS 创建于已经存在的 VPC 中，可使用 VPC 内现有机器。

### 安装 MySQL 客户端并连接

待创建好堡垒机后，我们可以通过 SSH 远程连接到堡垒机，再通过 MySQL 客户端 来访问 TiDB 集群。

用 SSH 连接到堡垒机：

{{< copyable "shell-regular" >}}

```shell
ssh [-i /path/to/your/private-key.pem] ec2-user@<bastion-public-dns-name>
```

安装 MySQL 客户端：

{{< copyable "shell-regular" >}}

```shell
sudo yum install mysql -y
```

连接到 TiDB 集群：

{{< copyable "shell-regular" >}}

```shell
mysql -h <tidb-nlb-dnsname> -P 4000 -u root
```

`<tidb-nlb-dnsname>` 为 TiDB Service 的 LoadBalancer 域名，可以通过 `kubectl get svc basic-tidb -n tidb-cluster` 输出中的 `EXTERNAL-IP` 字段查看。

示例：

```shell
$ mysql -h abfc623004ccb4cc3b363f3f37475af1-9774d22c27310bc1.elb.us-west-2.amazonaws.com -P 4000 -u root
Welcome to the MariaDB monitor.  Commands end with ; or \g.
Your MySQL connection id is 1189
Server version: 5.7.25-TiDB-v4.0.2 TiDB Server (Apache License 2.0) Community Edition, MySQL 5.7 compatible

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
| server_id          | ed4ba88b-436a-424d-9087-977e897cf5ec |
+--------------------+--------------------------------------+
6 rows in set (0.00 sec)
```

> **注意：**
>
> TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见[遥测](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

## Grafana 监控

先获取 Grafana 的 LoadBalancer 域名：

{{< copyable "shell-regular" >}}

```shell
kubectl -n tidb-cluster get svc basic-grafana
```

示例：

```
$ kubectl get svc basic-grafana
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP                                                             PORT(S)          AGE
basic-grafana   LoadBalancer   10.100.199.42   a806cfe84c12a4831aa3313e792e3eed-1964630135.us-west-2.elb.amazonaws.com 3000:30761/TCP   121m
```

其中 `EXTERNAL-IP` 栏即为 LoadBalancer 域名。

你可以通过浏览器访问 `<grafana-lb>:3000` 地址查看 Grafana 监控指标。其中 `<grafana-lb>` 替换成前面获取的域名。

Grafana 默认登录信息：

- 用户名：admin
- 密码：admin

## 升级 TiDB 集群

要升级 TiDB 集群，可以通过 `kubectl edit tc basic -n tidb-cluster` 修改 `spec.version`。

升级过程会持续一段时间，你可以通过 `kubectl get pods -n tidb-cluster --watch` 命令持续观察升级进度。

## 扩容 TiDB 集群

注意扩容前需要对相应的节点组进行扩容，以便新的实例有足够的资源运行。

下面是将集群 `<clusterName>` 的 `tikv` 组扩容到 4 节点的示例：

{{< copyable "shell-regular" >}}

```shell
eksctl scale nodegroup --cluster <clusterName> --name tikv --nodes 4 --nodes-min 4 --nodes-max 4
```

然后通过 `kubectl edit tc basic -n tidb-cluster` 修改各组件的 `replicas` 为期望的新副本数进行扩容。

更多节点组管理可参考 [eksctl 文档](https://eksctl.io/usage/managing-nodegroups/)。

## 部署 TiFlash/TiCDC

### 新增节点组

在 eksctl 的配置文件 cluster.yaml 中新增以下两项，为 TiFlash/TiCDC 各自新增一个节点组。`desiredCapacity` 决定期望的节点数，根据实际需求而定。

```
  - name: tiflash
    desiredCapacity: 3
    labels:
      role: tiflash
    taints:
      dedicated: tiflash:NoSchedule
  - name: ticdc
    desiredCapacity: 1
    labels:
      role: ticdc
    taints:
      dedicated: ticdc:NoSchedule
```

- 若集群还未创建，使用 `eksctl create cluster -f cluster.yaml` 命令创建集群和节点组。
- 若集群已经创建，使用 `eksctl create nodegroup -f cluster.yaml` 命令只创建节点组（已经存在的节点组会忽略，不会重复创建）。

### 配置并部署

如果要部署 TiFlash，可以在 tidb-cluster.yaml 中配置 `spec.tiflash`，例如：

```yaml
spec:
  ...
  tiflash:
    baseImage: pingcap/tiflash
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
    tolerations:
    - effect: NoSchedule
      key: dedicated
      operator: Equal
      value: tiflash
```

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

值得注意的是，如果需要部署企业版的 TiDB/PD/TiKV/TiFlash/TiCDC，需要将 tidb-cluster.yaml 中 `spec.<tidb/pd/tikv/tiflash/ticdc>.baseImage` 配置为企业版镜像，格式为 `pingcap/<tidb/pd/tikv/tiflash/ticdc>-enterprise`。

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

## 使用其他 EBS 存储类型

AWS EBS 支持多种存储类型。若需要低延迟、高吞吐，可以选择 `io1` 类型。首先我们为 `io1` 新建一个存储类 (Storage Class)：

```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: io1
provisioner: kubernetes.io/aws-ebs
parameters:
  type: io1
  fsType: ext4
```

然后在 tidb cluster 的 YAML 文件中，通过 `storageClassName` 字段指定 `io1` 存储类申请 `io1` 类型的 EBS 存储。

更多存储类配置以及 EBS 存储类型选择，可以查看 [Storage Class 官方文档](https://kubernetes.io/docs/concepts/storage/storage-classes/) 和 [EBS 存储类型文档](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html)。

## 使用本地存储

AWS 部分实例类型提供额外的 [NVMe SSD 本地存储卷](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html)。可以为 TiKV 节点池选择这一类型的实例，以便提供更高的 IOPS 和低延迟。

> **注意：**
>
> 运行中的 TiDB 集群不能动态更换 storage class，可创建一个新的 TiDB 集群测试。
>
> 由于 EKS 升级过程中节点重建，本地盘数据会[丢失](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-lifetime)。由于 EKS 升级或其他原因造成的节点重建，会导致需要迁移 TiKV 数据，如果无法接受这一点，则不建议在生产环境中使用本地盘。

了解哪些实例可提供本地存储卷，可以查看 [AWS 实例列表](https://aws.amazon.com/ec2/instance-types/)。以下以 `c5d.4xlarge` 为例：

1. 为 TiKV 创建附带本地存储的节点组。

    修改 `eksctl` 配置文件中 TiKV 节点组实例类型为 `c5d.4xlarge`：

    ```
      - name: tikv
        instanceType: c5d.4xlarge
        labels:
          role: tikv
        taints:
          dedicated: tikv:NoSchedule
        ...
    ```

    创建节点组：

    {{< copyable "shell-regular" >}}

    ```shell
    eksctl create nodegroups -f cluster.yaml
    ```

    若 tikv 组已存在，可先删除再创建，或者修改名字规避名字冲突。

2. 部署 local volume provisioner。

    本地存储需要使用 [local-volume-provisioner](https://sigs.k8s.io/sig-storage-local-static-provisioner) 程序发现并管理。以下命令会部署并创建一个 `local-storage` 的 Storage Class。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/eks/local-volume-provisioner.yaml
    ```

3. 使用本地存储。

    完成前面步骤后，local-volume-provisioner 即可发现集群内所有本地 NVMe SSD 盘。在 tidb-cluster.yaml 中添加 `tikv.storageClassName` 字段并设置为 `local-storage` 即可。
