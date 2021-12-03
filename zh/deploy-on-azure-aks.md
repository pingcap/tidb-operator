---
title: 在 Azure AKS 上部署 TiDB 集群
summary: 介绍如何在 Azure AKS (Azure Kubernetes Service) 上部署 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-on-azure-aks/']
---

# 在 Azure AKS 上部署 TiDB 集群

本文介绍了如何在 Azure AKS (Azure Kubernetes Service) 上部署 TiDB 集群。

如果需要部署 TiDB Operator 及 TiDB 集群到自托管 Kubernetes 环境，请参考[部署 TiDB Operator](deploy-tidb-operator.md)及[部署 TiDB 集群](deploy-on-general-kubernetes.md)等文档。

## 前提条件

- 已安装 [Helm 3](https://helm.sh/docs/intro/install/)，用于安装 TiDB Operator。
- 已根据[部署 Azure Kubernetes 服务 (AKS) 群集](https://docs.microsoft.com/zh-cn/azure/aks/tutorial-kubernetes-deploy-cluster) 安装并配置 AKS 的命令行工具 az cli。

    > **注意：**
    >
    > 可运行 `az login` 命令验证 AZ CLI 的配置是否正确。如果登陆账户成功，则 AZ CLI 的配置是正确的。否则，您需要重新配置 AZ CLI。

- 已根据[使用 Azure Kubernetes 服务上的 Azure 超级磁盘（预览）](https://docs.microsoft.com/zh-cn/azure/aks/use-ultra-disks) 创建可以使用超级磁盘的新集群或启用现有集群上的超级磁盘。
- 已获取 [AKS 服务权限](https://docs.microsoft.com/zh-cn/azure/aks/concepts-identity#aks-service-permissions)。
- 在 Kubernetes 版本 < 1.21 的集群中已安装 **aks-preview CLI 扩展**以使用超级磁盘，并在您的订阅中注册过 **EnableAzureDiskFileCSIDriver** 功能。

    执行以下命令，安装 [aks-preview CLI 扩展](https://docs.microsoft.com/zh-cn/azure/aks/custom-node-configuration#install-aks-preview-cli-extension)：

    {{< copyable "shell-regular" >}}

    ```shell
    az extension add --name aks-preview
    ```

    执行以下命令，在[您的 Azure 订阅](https://docs.microsoft.com/zh-cn/cli/azure/feature?view=azure-cli-latest#az_feature_register-optional-parameters)中注册 [EnableAzureDiskFileCSIDriver](https://docs.microsoft.com/zh-cn/azure/aks/csi-storage-drivers#install-csi-storage-drivers-on-a-new-cluster-with-version--121) 功能：

    {{< copyable "shell-regular" >}}

    ```shell
    az feature register --name EnableAzureDiskFileCSIDriver --namespace Microsoft.ContainerService --subscription ${your-subscription-id}
    ```

## 创建 AKS 集群和节点池

TiDB 集群大部分组件使用 Azure 磁盘作为存储，根据 AKS 中的[最佳做法](https://docs.microsoft.com/zh-cn/azure/aks/operator-best-practices-cluster-isolation) ，推荐在创建 AKS 集群的时候确保每个节点池使用一个可用区（至少 3 个可用区）。

### 创建 [启用容器存储接口 (CSI) 驱动程序](https://docs.microsoft.com/zh-cn/azure/aks/csi-storage-drivers) 的 AKS 集群

> **注意：**
>
> 在 Kubernetes 版本 < 1.21 的集群中，需要额外使用 `--aks-custom-headers` 标志来启用 **EnableAzureDiskFileCSIDriver** 特性

{{< copyable "shell-regular" >}}

```shell
az aks create \
    --resource-group ${resourceGroup} \
    --name ${clusterName} \
    --location ${location} \
    --generate-ssh-keys \
    --vm-set-type VirtualMachineScaleSets \
    --load-balancer-sku standard \
    --node-count 3 \
    --zones 1 2 3 \
    --aks-custom-headers EnableAzureDiskFileCSIDriver=true
```

### 创建组件节点池

集群创建成功后，执行如下命令创建组件节点池，每个节点池创建耗时约 2~5 分钟。可以参考[`az aks` 文档](https://docs.microsoft.com/zh-cn/cli/azure/aks?view=azure-cli-latest#az_aks_create) 和 [`az aks nodepool` 文档](https://docs.microsoft.com/zh-cn/cli/azure/aks/nodepool?view=azure-cli-latest) 了解更多集群配置选项。推荐在 TiKV 组件节点池[启用超级磁盘](https://docs.microsoft.com/zh-cn/azure/aks/use-ultra-disks#enable-ultra-disks-on-an-existing-cluster)。

1. 创建 operator & monitor 节点池：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name admin \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=admin
    ```

2. 创建 pd 节点池, nodeType 建议为 Standard_F4s_v2 或更高配置：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name pd \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=pd \
        --node-taints dedicated=pd:NoSchedule
    ```

3. 创建 tidb 节点池, nodeType 建议为 Standard_F8s_v2 或更高配置，默认只需要两个 TiDB 节点，因此可以设置 `--node-count` 为 `2`，支持修改该参数进行扩容：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tidb \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 2 \
        --labels dedicated=tidb \
        --node-taints dedicated=tidb:NoSchedule
    ```

4. 创建 tikv 节点池, nodeType 建议为 Standard_E8s_v4 或更高配置：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tikv \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

### 在可用区部署节点池

Azure AKS 集群使用 "尽量实现区域均衡" 在多个可用区间部署节点，如果您希望使用 "严格执行区域均衡" (AKS 暂时不支持该策略)，可以考虑在每一个可用区部署一个节点池。 例如：

1. 在可用区 1 创建 tikv 节点池 1：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tikv1 \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

2. 在可用区 2 创建 tikv 节点池 2：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tikv2 \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 2 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

3. 在可用区 3 创建 tikv 节点池 3：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tikv3 \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

> **警告：**
>
> 关于节点池扩缩容：
>
> * 如果应用程序需要更改资源，可以手动缩放 AKS 群集以运行不同数量的节点。节点数减少时，节点会被[优雅地清空](https://kubernetes.io/zh/docs/tasks/administer-cluster/safely-drain-node/)，尽量避免对正在运行的应用程序造成中断。参考[在 AKS 中缩放节点数](https://docs.microsoft.com/zh-cn/azure/aks/scale-cluster)。

## 配置 StorageClass

为了提高存储的 IO 写入性能，推荐设置 StorageClass 的 `mountOptions` 字段，来设置存储挂载选项 `nodelalloc` 和 `noatime`。详情可见 [TiDB 环境与系统配置检查](https://docs.pingcap.com/zh/tidb/stable/check-before-deployment#在-tikv-部署目标机器上添加数据盘-ext4-文件系统挂载参数)

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
# ...
mountOptions:
- nodelalloc,noatime
```

## 部署 TiDB Operator

参考快速上手中[部署 TiDB Operator](get-started.md#部署-tidb-operator)，在 AKS 集群中部署 TiDB Operator。

## 部署 TiDB 集群和监控

下面介绍如何在 Azure AKS 上部署 TiDB 集群和监控组件。

### 创建 namespace

执行以下命令，创建 TiDB 集群安装的 namespace：

{{< copyable "shell-regular" >}}

```shell
kubectl create namespace tidb-cluster
```

> **注意：**
>
> 这里创建的 namespace 是指 [Kubernetes 命名空间](https://kubernetes.io/zh/docs/concepts/overview/working-with-objects/namespaces/)。本文档使用 `tidb-cluster` 为例，若使用了其他名字，修改相应的 `-n` 或 `--namespace` 参数为对应的名字即可。

### 部署 TiDB 集群和监控

首先执行以下命令，下载 TidbCluster 和 TidbMonitor CR 的配置文件。

{{< copyable "shell-regular" >}}

```shell
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aks/tidb-cluster.yaml && \
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aks/tidb-monitor.yaml
```

如需了解更详细的配置信息或者进行自定义配置，请参考[配置 TiDB 集群](configure-a-tidb-cluster.md)

> **注意：**
>
> 默认情况下，`tidb-cluster.yaml` 文件中 TiDB 服务的 LoadBalancer 配置为 "internal"。这意味着 LoadBalancer 只能在集群虚拟网络内部访问，而不能在外部访问。要通过 MySQL 协议访问 TiDB，您需要使用一个堡垒机进入集群节点或使用 `kubectl port-forward`。如果您想在互联网上公开访问 TiDB，并且知晓这样做的风险，您可以在 `tidb-cluster.yaml` 文件中将 LoadBalancer 的 "internal" 删除，默认创建的 LoadBalancer 将能够在外部访问。

执行以下命令，在 AKS 集群中部署 TidbCluster 和 TidbMonitor CR。

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```

当上述 yaml 文件被应用到 Kubernetes 集群后，TiDB Operator 会负责根据 yaml 文件描述，创建对应配置的 TiDB 集群及其监控。

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

创建好 TiDB 集群后，您就可以访问数据库，进行测试和开发了。

### 访问方式

- 使用堡垒机访问数据库

我们为 TiDB 集群创建的是内网 LoadBalancer，可以通过创建[堡垒机](https://docs.microsoft.com/zh-cn/azure/bastion/tutorial-create-host-portal)进入集群节点来访问数据库。

> **注意：**
>
> 除使用堡垒机以外，也可以使用 [虚拟网络对等互连](https://docs.microsoft.com/zh-cn/azure/virtual-network/virtual-network-peering-overview) 连接现有机器到集群虚拟网络。若 AKS 创建于已经存在的虚拟网络中，可使用虚拟网络内现有机器。

- 使用 SSH 访问数据库

使用[创建与 Linux 节点的 SSH 连接](https://docs.microsoft.com/zh-cn/azure/aks/ssh#create-the-ssh-connection-to-a-linux-node)从而进入集群节点来访问数据库。

- 使用 node-shell 访问数据库

简单的使用 [node-shell](https://github.com/kvaps/kubectl-node-shell) 等工具进入集群节点，然后访问数据库。

### 安装 MySQL 客户端并连接

登陆集群节点后，我们可以通过 MySQL 客户端来访问 TiDB 集群。

在集群节点上安装 MySQL 客户端：

{{< copyable "shell-regular" >}}

```shell
sudo yum install mysql -y
```

连接到 TiDB 集群：

{{< copyable "shell-regular" >}}

```shell
mysql --comments -h ${tidb-lb-ip} -P 4000 -u root
```

其中 `${tidb-lb-ip}` 为 TiDB Service 的 LoadBalancer 域名，可以通过命令 `kubectl get svc basic-tidb -n tidb-cluster` 输出中的 `EXTERNAL-IP` 字段查看。

以下为一个连接 TiDB 集群的示例：

```shell
mysql --comments -h 20.240.0.7 -P 4000 -u root
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

```shell
kubectl -n tidb-cluster get svc basic-grafana
```

示例输出：

```shell
kubectl get svc basic-grafana
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP                                                             PORT(S)          AGE
basic-grafana   LoadBalancer   10.100.199.42   20.240.0.8    3000:30761/TCP   121m
```

其中 `EXTERNAL-IP` 栏即为 LoadBalancer 域名。

您可以通过浏览器访问 `${grafana-lb}:3000` 地址查看 Grafana 监控指标。其中 `${grafana-lb}` 替换成前面获取的域名。

> **注意：**
>
> Grafana 默认用户名和密码均为 admin。

## 访问 TiDB Dashboard

如果想要安全地访问 TiDB Dashboard，详情可以参见[访问 TiDB Dashboard](access-dashboard.md)。

## 升级 TiDB 集群

要升级 TiDB 集群，可以通过 `kubectl edit tc basic -n tidb-cluster` 命令修改 `spec.version`。

升级过程会持续一段时间，您可以通过 `kubectl get pods -n tidb-cluster --watch` 命令持续观察升级进度。

## 扩容 TiDB 集群

扩容前需要对相应的节点池进行扩容，以便新的实例有足够的资源运行。以下展示扩容 AKS 节点池和 TiDB 集群组件的操作。

### 扩容 AKS 节点池

TiKV 扩容需要保证在各可用区均匀扩容。以下是将集群 `${clusterName}` 的节点池扩容到 6 节点的示例：

{{< copyable "shell-regular" >}}

```shell
az aks nodepool scale \
    --resource-group ${resourceGroup} \
    --cluster-name ${clusterName} \
    --name ${nodePoolName} \
    --node-count 6
```

更多节点池管理可参考 [`az aks nodepool` 文档](https://docs.microsoft.com/zh-cn/cli/azure/aks/nodepool?view=azure-cli-latest)。

### 扩容 TiDB 组件

扩容 AKS 节点池后，可以使用命令 `kubectl edit tc basic -n tidb-cluster` 修改各组件的 `replicas` 为期望的新副本数进行扩容。

## 部署 TiFlash/TiCDC

[TiFlash](https://docs.pingcap.com/zh/tidb/stable/tiflash-overview) 是 TiKV 的列存扩展。

[TiCDC](https://docs.pingcap.com/zh/tidb/stable/ticdc-overview) 是一款通过拉取 TiKV 变更日志实现的 TiDB 增量数据同步工具。

这两个组件*不是必选*安装项，这里提供一个快速安装上手示例。

### 新增节点池

为 TiFlash/TiCDC 各自新增一个节点池。`--node-count` 决定期望的节点数，根据实际需求而定。

- 创建 tiflash 节点池, nodeType 建议为 Standard_E8s_v4 或更高配置：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tiflash \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=tiflash \
        --node-taints dedicated=tiflash:NoSchedule
    ```

- 创建 ticdc 节点池, nodeType 建议为 Standard_E16s_v4 或更高配置：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name ticdc \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=ticdc \
        --node-taints dedicated=ticdc:NoSchedule
    ```

### 配置并部署 TiFlash/TiCDC

如果要部署 TiFlash，可以在 tidb-cluster.yaml 中配置 `spec.tiflash`，例如：

``` yaml
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

其他参数可以参考 [TiDB 集群配置文档](configure-a-tidb-cluster.md)进行配置。

> **警告：**
>
> 由于 TiDB Operator 会按照 `storageClaims` 列表中的配置**按顺序**自动挂载 PV，如果需要为 TiFlash 增加磁盘，请确保只在列表原有配置**末尾添加**，并且**不能**修改列表中原有配置的顺序。

如果要部署 TiCDC，可以在 tidb-cluster.yaml 中配置 `spec.ticdc`，例如：

``` yaml
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

``` yaml
spec:
  ...
  pd:
    baseImage: pingcap/pd-enterprise
  ...
  tikv:
    baseImage: pingcap/tikv-enterprise
```

## 使用其他 Azure 磁盘类型

Azure Disk 支持多种磁盘类型。若需要低延迟、高吞吐，可以选择 `UltraSSD` 类型。首先我们为 `UltraSSD` 新建一个存储类 (Storage Class)：

1. [启用现有群集上的超级磁盘](https://docs.microsoft.com/zh-cn/azure/aks/use-ultra-disks#enable-ultra-disks-on-an-existing-cluster) 并创建存储类 `ultra`:

    ``` yaml
    apiVersion: storage.k8s.io/v1
    kind: StorageClass
    metadata:
      name: ultra
    provisioner: disk.csi.azure.com
    parameters:
      skuname: UltraSSD_LRS  # alias: storageaccounttype, available values: Standard_LRS, Premium_LRS, StandardSSD_LRS, UltraSSD_LRS
      cachingMode: None
    reclaimPolicy: Delete
    allowVolumeExpansion: true
    volumeBindingMode: WaitForFirstConsumer
    mountOptions:
    - nodelalloc,noatime
    ```

    你可以根据实际需要额外配置[驱动参数](https://github.com/kubernetes-sigs/azuredisk-csi-driver/blob/master/docs/driver-parameters.md)。

2. 然后在 tidb cluster 的 YAML 文件中，通过 `storageClassName` 字段指定 `ultra` 存储类申请 `UltraSSD` 类型的 Azure 磁盘。可以参考以下 TiKV 配置示例使用：

    ``` yaml
    spec:
      tikv:
        baseImage: pingcap/tikv
        replicas: 3
        storageClassName: ultra
        requests:
          storage: "100Gi"
    ```

您可以使用任意 Azure 磁盘类型，推荐使用 `Premium_LRS` 或 `UltraSSD_LRS`。

更多关于存储类配置和 Azure 磁盘类型的信息，可以参考 [Storage Class 官方文档](https://github.com/kubernetes-sigs/azuredisk-csi-driver)和 [Azure 磁盘类型官方文档](https://docs.microsoft.com/zh-cn/azure/virtual-machines/disks-types)。

## 使用本地存储

请使用 Azure LRS Disk 作为生产环境的存储类型。如果需要模拟测试裸机部署的性能，可以使用 Azure 部分实例类型提供的 [NVMe SSD 本地磁盘](https://docs.microsoft.com/zh-cn/azure/virtual-machines/sizes-storage)。可以为 TiKV 节点池选择这一类型的实例，以便提供更高的 IOPS 和低延迟。

> **注意：**
>
> 运行中的 TiDB 集群不能动态更换 storage class，可创建一个新的 TiDB 集群测试。
>
> 本地 NVMe 磁盘是临时的，如果停止/解除分配 VM，这些磁盘上的数据都将丢失。由于 AKS 升级或其他原因造成的节点重建，会导致需要迁移 TiKV 数据，如果无法接受这一点，则不建议在生产环境中使用本地磁盘。

了解哪些实例可提供本地磁盘，可以查看 [Lsv2 系列](https://docs.microsoft.com/zh-cn/azure/virtual-machines/lsv2-series)。以下以 `Standard_L8s_v2` 为例：

1. 为 TiKV 创建附带本地磁盘的节点池。

    修改 `az aks nodepool add` 命令中 TiKV 节点池实例类型为 `Standard_L8s_v2`：

    {{< copyable "shell-regular" >}}

    ```shell
    az aks nodepool add --name tikv \
        --cluster-name ${clusterName}  \
        --resource-group ${resourceGroup} \
        --node-vm-size Standard_L8s_v2 \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --enable-ultra-ssd \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule
    ```

    若 tikv 节点池已存在，可先删除再创建，或者修改名字规避冲突。

2. 部署 local volume provisioner。

    本地存储需要使用 [local-volume-provisioner](https://sigs.k8s.io/sig-storage-local-static-provisioner) 程序发现并管理。以下命令会部署并创建一个 `local-storage` 的 Storage Class。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/eks/local-volume-provisioner.yaml
    ```

3. 使用本地存储。

    完成前面步骤后，local-volume-provisioner 即可发现集群内所有本地 NVMe SSD 盘。在 tidb-cluster.yaml 中添加 `tikv.storageClassName` 字段并设置为 `local-storage` 即可，可以参考前文[部署 TiDB 集群和监控](#部署-tidb-集群和监控)部分。
