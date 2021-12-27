---
title: 在 AWS EKS 上部署 TiDB 集群
summary: 介绍如何在 AWS EKS (Elastic Kubernetes Service) 上部署 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-on-aws-eks/']
---

# 在 AWS EKS 上部署 TiDB 集群

本文介绍了如何在 AWS EKS (Elastic Kubernetes Service) 上部署 TiDB 集群。

如果需要部署 TiDB Operator 及 TiDB 集群到自托管 Kubernetes 环境，请参考[部署 TiDB Operator](deploy-tidb-operator.md)及[部署 TiDB 集群](deploy-on-general-kubernetes.md)等文档。

## 环境准备

部署前，请确认已完成以下环境准备：

- 安装 [Helm 3](https://helm.sh/docs/intro/install/)：用于安装 TiDB Operator。

- 完成 AWS [eksctl 入门](https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/getting-started-eksctl.html)中所有操作。

    该教程包含以下内容：

    - 安装并配置 AWS 的命令行工具 awscli
    - 安装并配置创建 Kubernetes 集群的命令行工具 eksctl
    - 安装 Kubernetes 命令行工具 kubectl

要验证 AWS CLI 的配置是否正确，请运行 `aws configure list` 命令。如果此命令的输出显示了 `access_key` 和 `secret_key` 的值，则 AWS CLI 的配置是正确的。否则，你需要重新配置 AWS CLI。

> **注意：**
>
> 本文档的操作需要 AWS Access Key 至少具有 [eksctl 所需最少权限](https://eksctl.io/usage/minimum-iam-policies/)和创建 [Linux 堡垒机所涉及的服务权限](https://docs.aws.amazon.com/quickstart/latest/linux-bastion/architecture.html#aws-services)。

## 推荐机型及存储

- 推荐机型：出于性能考虑，推荐 PD 所在节点使用 c5.xlarge，TiDB 所在节点使用 c5.2xlarge，TiKV 或 TiFlash 所在节点使用 r5b.2xlarge。
- 推荐存储：因为 AWS 目前已经支持 [EBS gp3](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html#gp3-ebs-volume-type) 卷类型，建议使用 EBS gp3 卷类型。对于 gp3 配置，推荐 TiKV 的 gp3 配置达到 400 MiB/s 与 4000 IOPS，推荐 TiFlash 的 gp3 配置达到 625 MiB/s 与 6000 IOPS。

## 创建 EKS 集群和节点池

根据 AWS [官方博客](https://aws.amazon.com/cn/blogs/containers/amazon-eks-cluster-multi-zone-auto-scaling-groups/)推荐和 EKS [最佳实践文档](https://aws.github.io/aws-eks-best-practices/reliability/docs/dataplane/#ensure-capacity-in-each-az-when-using-ebs-volumes)，由于 TiDB 集群大部分组件使用 EBS 卷作为存储，推荐在创建 EKS 的时候针对每个组件在每个可用区（至少 3 个可用区）创建一个节点池。

将以下配置存为 cluster.yaml 文件，并替换 `${clusterName}` 为自己想命名的集群名字。集群和节点组的命名规则需要与正则表达式 `[a-zA-Z][-a-zA-Z0-9]*` 相匹配，避免包含 `_`。

{{< copyable "" >}}

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: ${clusterName}
  region: ap-northeast-1

nodeGroups:
  - name: admin
    desiredCapacity: 1
    privateNetworking: true
    labels:
      dedicated: admin

  - name: tidb-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    instanceType: c5.2xlarge
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule
  - name: tidb-1d
    desiredCapacity: 0
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    instanceType: c5.2xlarge
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule
  - name: tidb-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    instanceType: c5.2xlarge
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule

  - name: pd-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    instanceType: c5.xlarge
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule
  - name: pd-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    instanceType: c5.xlarge
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule
  - name: pd-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    instanceType: c5.xlarge
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule

  - name: tikv-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    instanceType: r5b.2xlarge
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
  - name: tikv-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    instanceType: r5b.2xlarge
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
  - name: tikv-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    instanceType: r5b.2xlarge
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
```

默认只需要两个 TiDB 节点，因此可以设置 `tidb-1d` 节点组的 `desiredCapacity` 为 `0`，后面如果需要可以随时扩容这个节点组。

执行以下命令创建集群：

{{< copyable "shell-regular" >}}

```bash
eksctl create cluster -f cluster.yaml
```

该命令需要等待 EKS 集群创建完成，以及节点组创建完成并加入进去，耗时约 5~20 分钟。可参考 [eksctl 文档](https://eksctl.io/usage/creating-and-managing-clusters/#using-config-files)了解更多集群配置选项。

> **警告：**
>
> 如果使用了 Regional Auto Scaling Group (ASG)：
>
> * 为已经启动的 EC2 [开启实例缩减保护](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-instance-termination.html#instance-protection-instance)，ASG 自身的实例缩减保护不需要打开。
> * [设置 ASG 终止策略](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-instance-termination.html#custom-termination-policy)为 `NewestInstance`。

## 配置 StorageClass

本小节介绍如何为不同的存储类型配置 StorageClass，包括创建 EKS 集群后默认存在的 gp2 存储类型、gp3 存储类型（推荐）或其他 EBS 存储类型、以及用于模拟测试裸机部署性能的本地存储。

### gp2

创建 EKS 集群后默认会存在一个 gp2 存储类型的 StorageClass。为了提高存储的 IO 写入性能，推荐配置 StorageClass 的 `mountOptions` 字段来设置存储挂载选项 `nodelalloc` 和 `noatime`。详情可见 [TiDB 环境与系统配置检查](https://docs.pingcap.com/zh/tidb/stable/check-before-deployment#%E5%9C%A8-tikv-%E9%83%A8%E7%BD%B2%E7%9B%AE%E6%A0%87%E6%9C%BA%E5%99%A8%E4%B8%8A%E6%B7%BB%E5%8A%A0%E6%95%B0%E6%8D%AE%E7%9B%98-ext4-%E6%96%87%E4%BB%B6%E7%B3%BB%E7%BB%9F%E6%8C%82%E8%BD%BD%E5%8F%82%E6%95%B0)。

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
# ...
mountOptions:
- nodelalloc,noatime
```

### gp3 存储类型（推荐）或其他 EBS 存储类型

如果不想使用默认的 gp2 存储类型，可以创建其他存储类型的 StorageClass，例如 gp3 存储类型（推荐）或者 io1 存储类型。

以下步骤以 gp3 存储类型为例说明如何创建并配置 gp3 存储类型的 StorageClass。

1. 对于 gp3 存储类型，请参考 [AWS 文档](https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html)在 EKS 上部署 [Amazon Elastic Block Store (EBS) CSI driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver)。对于其他存储类型，请跳过此步骤。
2. 创建 StorageClass 定义。在 StorageClass 定义中，通过 `parameters.type` 字段指定需要的存储类型。

    ```yaml
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    metadata:
      name: gp3
    provisioner: ebs.csi.aws.com
    allowVolumeExpansion: true
    volumeBindingMode: WaitForFirstConsumer
    parameters:
      type: gp3
      fsType: ext4
      iops: "4000"
      throughput: "400"
    mountOptions:
    - nodelalloc,noatime
    ```

3. 在 TidbCluster 的 YAML 文件中，通过 `storageClassName` 字段指定 gp3 存储类来申请 `gp3` 类型的 EBS 存储。可以参考以下 TiKV 配置示例：

    ```yaml
    spec:
      tikv:
        ...
        storageClassName: gp3
    ```

4. 为了提高存储的 IO 写入性能，推荐配置 StorageClass 的 `mountOptions` 字段来设置存储挂载选项 `nodelalloc` 和 `noatime`。详情可见 [TiDB 环境与系统配置检查](https://docs.pingcap.com/zh/tidb/stable/check-before-deployment#%E5%9C%A8-tikv-%E9%83%A8%E7%BD%B2%E7%9B%AE%E6%A0%87%E6%9C%BA%E5%99%A8%E4%B8%8A%E6%B7%BB%E5%8A%A0%E6%95%B0%E6%8D%AE%E7%9B%98-ext4-%E6%96%87%E4%BB%B6%E7%B3%BB%E7%BB%9F%E6%8C%82%E8%BD%BD%E5%8F%82%E6%95%B0)。

    ```yaml
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    # ...
    mountOptions:
    - nodelalloc,noatime
    ```

如果想了解更多 EBS 存储类型选择和配置信息，请查看 [AWS 官方文档](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html) 和 [Storage Class 官方文档](https://kubernetes.io/docs/concepts/storage/storage-classes/)。

### 本地存储

请使用 AWS EBS 作为生产环境的存储类型。如果需要模拟测试裸机部署的性能，可以为 TiKV 节点池选择 AWS 部分实例类型提供的 [NVMe SSD 本地存储卷](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html)，以提供更高的 IOPS 和更低的延迟。

> **注意：**
>
> - 运行中的 TiDB 集群不能动态更换 StorageClass，可创建一个新的 TiDB 集群测试。
> - 由于 EKS 升级或其他原因造成的节点重建会导致[本地盘数据会丢失](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-lifetime)，在重建前你需要提前备份 TiKV 数据，因此不建议在生产环境中使用本地盘。
> - 为了避免由于节点重建导致本地存储数据丢失，请参考 [AWS 文档](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-suspend-resume-processes.html)停止 TiKV 节点组的 `ReplaceUnhealthy` 功能。

要了解哪些 AWS 实例可提供本地存储卷，可以查看 [AWS 实例类型列表](https://aws.amazon.com/ec2/instance-types/)。

下面以 `c5d.4xlarge` 为例说明如何为本地存储配置 StorageClass。

1. 为 TiKV 创建附带本地存储的节点组。

    1. 修改 `eksctl` 配置文件中 TiKV 节点组实例类型为 `c5d.4xlarge`：

        ```yaml
          - name: tikv-1a
            desiredCapacity: 1
            privateNetworking: true
            availabilityZones: ["ap-northeast-1a"]
            instanceType: c5d.4xlarge
            labels:
              dedicated: tikv
            taints:
              dedicated: tikv:NoSchedule
            ...
        ```

    2. 创建附带本地存储的节点组：

        {{< copyable "shell-regular" >}}

        ```bash
        eksctl create nodegroups -f cluster.yaml
        ```

        若 TiKV 的节点组已存在，为避免名字冲突，可先删除再创建，或者修改节点组的名字。

2. 部署 local volume provisioner。

    1. 为了更方便地发现并管理本地存储，你需要安装 [local-volume-provisioner](https://sigs.k8s.io/sig-storage-local-static-provisioner) 程序。

    2. 通过[普通挂载方式](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/master/docs/operations.md#use-a-whole-disk-as-a-filesystem-pv)将本地存储挂载到 `/mnt/ssd` 目录。

    3. 根据本地存储的挂载情况，修改 [local-volume-provisioner.yaml](https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/eks/local-volume-provisioner.yaml) 文件。

    4. 使用修改后的 `local-volume-provisioner.yaml`，部署并创建一个 `local-storage` 的 Storage Class：

        {{< copyable "shell-regular" >}}

        ```bash
        kubectl apply -f <local-volume-provisioner.yaml>
        ```

3. 使用本地存储。

    完成前面步骤后，local-volume-provisioner 即可发现集群内所有本地 NVMe SSD 盘。

在 local-volume-provisioner 发现本地盘后，当[部署 TiDB 集群和监控](#部署-tidb-集群和监控)时，请在 `tidb-cluster.yaml` 中添加 `tikv.storageClassName` 字段并设置为 `local-storage`。

## 部署 TiDB Operator

参考快速上手中[部署 TiDB Operator](get-started.md#部署-tidb-operator)，在 EKS 集群中部署 TiDB Operator。

## 部署 TiDB 集群和监控

下面介绍如何在 AWS EKS 上部署 TiDB 集群和监控组件。

### 创建 namespace

执行以下命令，创建 TiDB 集群安装的 namespace：

{{< copyable "shell-regular" >}}

```bash
kubectl create namespace tidb-cluster
```

> **注意：**
>
> 这里创建的 namespace 是指 [Kubernetes 命名空间](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)。本文档使用 `tidb-cluster` 为例，若使用了其他名字，修改相应的 `-n` 或 `--namespace` 参数为对应的名字即可。

### 部署 TiDB 集群和监控

首先执行以下命令，下载 TidbCluster 和 TidbMonitor CR 的配置文件。

{{< copyable "shell-regular" >}}

```bash
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-cluster.yaml &&
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-monitor.yaml
```

如需了解更详细的配置信息或者进行自定义配置，请参考[配置 TiDB 集群](configure-a-tidb-cluster.md)

> **注意：**
>
> 默认情况下，`tidb-cluster.yaml` 文件中 TiDB 服务的 LoadBalancer 配置为 "internal"。这意味着 LoadBalancer 只能在 VPC 内部访问，而不能在外部访问。要通过 MySQL 协议访问 TiDB，你需要使用一个堡垒机或使用 `kubectl port-forward`。如果你想在互联网上公开访问 TiDB，并且知晓这样做的风险，你可以在 `tidb-cluster.yaml` 文件中将 LoadBalancer 从 "internal" 改为 "internet-facing"。

执行以下命令，在 EKS 集群中部署 TidbCluster 和 TidbMonitor CR。

{{< copyable "shell-regular" >}}

```bash
kubectl apply -f tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```

当上述 yaml 文件被应用到 Kubernetes 集群后，TiDB Operator 会负责根据 yaml 文件描述，创建对应配置的 TiDB 集群及其监控。

> **注意：**
>
> 如果要将 TiDB 集群部署到 ARM64 机器上，可以参考[在 ARM64 机器上部署 TiDB 集群](deploy-cluster-on-arm64.md)。

### 查看 TiDB 集群启动状态

使用以下命令查看 TiDB 集群启动状态：

{{< copyable "shell-regular" >}}

```bash
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

创建好 TiDB 集群后，我们就可以访问数据库，进行测试和开发了。

### 准备一台堡垒机

我们为 TiDB 集群创建的是内网 LoadBalancer，因此可以在集群 VPC 内创建一台[堡垒机](https://aws.amazon.com/quickstart/architecture/linux-bastion/)来访问数据库。具体参考 [AWS Linux 堡垒机文档](https://aws.amazon.com/quickstart/architecture/linux-bastion/)在 AWS Console 上创建即可。

VPC 和 Subnet 需选择集群的 VPC 和 Subnet，在下拉框通过集群名字确认是否正确。可以通过以下命令查看集群的 VPC 和 Subnet 来验证：

{{< copyable "shell-regular" >}}

```bash
eksctl get cluster -n ${clusterName}
```

同时需允许本机网络访问，并选择正确的 Key Pair 以便能通过 SSH 登录机器。

> **注意：**
>
> 除使用堡垒机以外，也可以使用 [VPC Peering](https://docs.aws.amazon.com/vpc/latest/peering/what-is-vpc-peering.html) 连接现有机器到集群 VPC。若 EKS 创建于已经存在的 VPC 中，可使用 VPC 内现有机器。

### 安装 MySQL 客户端并连接

创建好堡垒机后，我们可以通过 SSH 远程连接到堡垒机，再通过 MySQL 客户端来访问 TiDB 集群。

使用 SSH 登录堡垒机：

{{< copyable "shell-regular" >}}

```bash
ssh [-i /path/to/your/private-key.pem] ec2-user@<bastion-public-dns-name>
```

在堡垒机上安装 MySQL 客户端：

{{< copyable "shell-regular" >}}

```bash
sudo yum install mysql -y
```

连接到 TiDB 集群：

{{< copyable "shell-regular" >}}

```bash
mysql --comments -h ${tidb-nlb-dnsname} -P 4000 -u root
```

其中 `${tidb-nlb-dnsname}` 为 TiDB Service 的 LoadBalancer 域名，可以通过命令 `kubectl get svc basic-tidb -n tidb-cluster` 输出中的 `EXTERNAL-IP` 字段查看。

以下为一个连接 TiDB 集群的示例：

```bash
$ mysql --comments -h abfc623004ccb4cc3b363f3f37475af1-9774d22c27310bc1.elb.us-west-2.amazonaws.com -P 4000 -u root
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
> - [MySQL 8.0 默认认证插件](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin)从 `mysql_native_password` 更新为 `caching_sha2_password`，因此如果使用 MySQL 8.0 客户端访问 TiDB 服务（TiDB 版本 < v4.0.7），并且用户账户有配置密码，需要显示指定 `--default-auth=mysql_native_password` 参数。
> - TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见 [TiDB 遥测功能使用文档](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

## 访问 Grafana 监控

先获取 Grafana 的 LoadBalancer 域名：

{{< copyable "shell-regular" >}}

```bash
kubectl -n tidb-cluster get svc basic-grafana
```

示例输出：

```
$ kubectl get svc basic-grafana
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP                                                             PORT(S)          AGE
basic-grafana   LoadBalancer   10.100.199.42   a806cfe84c12a4831aa3313e792e3eed-1964630135.us-west-2.elb.amazonaws.com 3000:30761/TCP   121m
```

其中 `EXTERNAL-IP` 栏即为 LoadBalancer 域名。

你可以通过浏览器访问 `${grafana-lb}:3000` 地址查看 Grafana 监控指标。其中 `${grafana-lb}` 替换成前面获取的域名。

> **注意：**
>
> Grafana 默认用户名和密码均为 admin。

## 访问 TiDB Dashboard

如果想要安全地访问 TiDB Dashboard，详情可以参见[访问 TiDB Dashboard](access-dashboard.md)。

## 升级 TiDB 集群

要升级 TiDB 集群，可以通过 `kubectl edit tc basic -n tidb-cluster` 命令修改 `spec.version`。

升级过程会持续一段时间，你可以通过 `kubectl get pods -n tidb-cluster --watch` 命令持续观察升级进度。

## 扩容 TiDB 集群

扩容前需要对相应的节点组进行扩容，以便新的实例有足够的资源运行。以下展示扩容 EKS 节点组和 TiDB 集群组件的操作。

### 扩容 EKS 节点组

TiKV 扩容需要保证在各可用区均匀扩容。以下是将集群 `${clusterName}` 的 `tikv-1a`、`tikv-1c`、`tikv-1d` 节点组扩容到 2 节点的示例：

{{< copyable "shell-regular" >}}

```bash
eksctl scale nodegroup --cluster ${clusterName} --name tikv-1a --nodes 2 --nodes-min 2 --nodes-max 2
eksctl scale nodegroup --cluster ${clusterName} --name tikv-1c --nodes 2 --nodes-min 2 --nodes-max 2
eksctl scale nodegroup --cluster ${clusterName} --name tikv-1d --nodes 2 --nodes-min 2 --nodes-max 2
```

更多节点组管理可参考 [eksctl 文档](https://eksctl.io/usage/managing-nodegroups/)。

### 扩容 TiDB 组件

扩容 EKS 节点组后，可以使用命令 `kubectl edit tc basic -n tidb-cluster` 修改各组件的 `replicas` 为期望的新副本数进行扩容。

## 部署 TiFlash/TiCDC

[TiFlash](https://docs.pingcap.com/zh/tidb/stable/tiflash-overview) 是 TiKV 的列存扩展。

[TiCDC](https://docs.pingcap.com/zh/tidb/stable/ticdc-overview) 是一款通过拉取 TiKV 变更日志实现的 TiDB 增量数据同步工具。

这两个组件*不是必选*安装项，这里提供一个快速安装上手示例。

### 新增节点组

在 eksctl 的配置文件 cluster.yaml 中新增以下两项，为 TiFlash/TiCDC 各自新增一个节点组。`desiredCapacity` 决定期望的节点数，根据实际需求而定。

```yaml
  - name: tiflash-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    labels:
      dedicated: tiflash
    taints:
      dedicated: tiflash:NoSchedule
  - name: tiflash-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: tiflash
    taints:
      dedicated: tiflash:NoSchedule
  - name: tiflash-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    labels:
      dedicated: tiflash
    taints:
      dedicated: tiflash:NoSchedule

  - name: ticdc-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    labels:
      dedicated: ticdc
    taints:
      dedicated: ticdc:NoSchedule
  - name: ticdc-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: ticdc
    taints:
      dedicated: ticdc:NoSchedule
  - name: ticdc-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    labels:
      dedicated: ticdc
    taints:
      dedicated: ticdc:NoSchedule
```

具体命令根据 EKS 集群创建情况而定：

- 若集群还未创建，使用 `eksctl create cluster -f cluster.yaml` 命令创建集群和节点组。
- 若集群已经创建，使用 `eksctl create nodegroup -f cluster.yaml` 命令只创建节点组（已经存在的节点组会忽略，不会重复创建）。

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
    tolerations:
    - effect: NoSchedule
      key: dedicated
      operator: Equal
      value: tiflash
```

其他参数可以参考 [TiDB 集群配置文档](configure-a-tidb-cluster.md)进行配置。

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

部署企业版 TiDB/PD/TiKV/TiFlash/TiCDC 时，只需要将 tidb-cluster.yaml 中 `spec.[tidb|pd|tikv|tiflash|ticdc].baseImage` 配置为企业版镜像，格式为 `pingcap/[tidb|pd|tikv|tiflash|ticdc]-enterprise`。

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
