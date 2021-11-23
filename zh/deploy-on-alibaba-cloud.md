---
title: 在阿里云上部署 TiDB 集群
summary: 介绍如何在阿里云上部署 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-on-alibaba-cloud/']
---

# 在阿里云上部署 TiDB 集群

本文介绍了如何使用个人电脑（Linux 或 macOS 系统）在阿里云上部署 TiDB 集群。

如果需要部署 TiDB Operator 及 TiDB 集群到自托管 Kubernetes 环境，请参考[部署 TiDB Operator](deploy-tidb-operator.md)及[部署 TiDB 集群](deploy-on-general-kubernetes.md)等文档。

## 环境需求

- [aliyun-cli](https://github.com/aliyun/aliyun-cli) >= 3.0.15 并且[配置 aliyun-cli](https://www.alibabacloud.com/help/doc-detail/90766.htm?spm=a2c63.l28256.a3.4.7b52a893EFVglq)

    > **注意：**
    >
    > Access Key 需要具有操作相应资源的权限。

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.12
- [Helm 3](https://helm.sh)
- [jq](https://stedolan.github.io/jq/download/) >= 1.6
- [terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) 0.12.*

你可以使用阿里云的[云命令行](https://shell.aliyun.com)服务来进行操作，云命令行中已经预装并配置好了所有工具。

### 权限

完整部署集群需要具备以下权限：

- AliyunECSFullAccess
- AliyunESSFullAccess
- AliyunVPCFullAccess
- AliyunSLBFullAccess
- AliyunCSFullAccess
- AliyunEIPFullAccess
- AliyunECIFullAccess
- AliyunVPNGatewayFullAccess
- AliyunNATGatewayFullAccess

## 概览

默认配置下，会创建：

- 一个新的 VPC
- 一台 ECS 实例作为堡垒机
- 一个托管版 ACK（阿里云 Kubernetes）集群以及一系列 worker 节点：
    - 属于一个伸缩组的 2 台 ECS 实例（2 核 2 GB），托管版 Kubernetes 的默认伸缩组中必须至少有两台实例，用于承载整个的系统服务，例如 CoreDNS
    - 属于一个伸缩组的 3 台 `ecs.g5.large` 实例，用于部署 PD
    - 属于一个伸缩组的 3 台 `ecs.i2.2xlarge` 实例，用于部署 TiKV
    - 属于一个伸缩组的 2 台 `ecs.c5.4xlarge` 实例用于部署 TiDB
    - 属于一个伸缩组的 1 台 `ecs.c5.xlarge` 实例用于部署监控组件

除了默认伸缩组之外的其它所有实例都是跨可用区部署的。而伸缩组 (Auto-scaling Group) 能够保证集群的健康实例数等于期望数值。因此，当发生节点故障甚至可用区故障时，伸缩组能够自动为我们创建新实例来确保服务可用性。

## 安装部署

### 部署 ACK，TiDB Operator 和 TiDB 集群节点池

使用如下步骤部署 ACK，TiDB Operator 和 TiDB 集群节点池。

1. 设置目标 region 和阿里云密钥（也可以在运行 `terraform` 命令时根据命令提示输入）：

    {{< copyable "shell-regular" >}}

    ```shell
    export TF_VAR_ALICLOUD_REGION=${REGION} && \
    export TF_VAR_ALICLOUD_ACCESS_KEY=${ACCESS_KEY} && \
    export TF_VAR_ALICLOUD_SECRET_KEY=${SECRET_KEY}
    ```

    用于部署集群的各变量的默认值存储在 `variables.tf` 文件中，如需定制可以修改此文件或在安装时通过 `-var` 参数覆盖。

2. 使用 Terraform 进行安装：

    {{< copyable "shell-regular" >}}

    ```shell
    git clone --depth=1 https://github.com/pingcap/tidb-operator && \
    cd tidb-operator/deploy/aliyun
    ```

    可以新建或者编辑 `terraform.tfvars`，在其中设置变量的值，按需配置集群，可以通过 `variables.tf` 查看有哪些变量可以设置以及各变量的详细描述。例如，下面示例配置 ACK 集群名称、TiDB 集群名称，TiDB Operator 版本及 PD、TiKV 和 TiDB 节点的数量：

    ```
    cluster_name = "testack"
    tidb_cluster_name = "testdb"
    tikv_count = 3
    tidb_count = 2
    pd_count = 3
    operator_version = "v1.2.4"
    ```

    如果需要在集群中部署 TiFlash，需要在 `terraform.tfvars` 中设置 `create_tiflash_node_pool = true`，也可以设置 `tiflash_count` 和 `tiflash_instance_type` 来配置 TiFlash 节点池的节点数量和实例类型，`tiflash_count` 默认为 `2`，`tiflash_instance_type` 默认为 `ecs.i2.2xlarge`。

    如果需要在集群中部署 TiCDC，需要在 `terraform.tfvars` 中设置 `create_cdc_node_pool = true`，也可以设置 `cdc_count` 和 `cdc_instance_type` 来配置 TiCDC 节点池的节点数量和实例类型，`cdc_count` 默认为 `3`，`cdc_instance_type` 默认为 `ecs.c5.2xlarge`。

    > **注意：**
    >
    > 请通过 `variables.tf` 文件中的 `operator_version` 确认当前版本脚本中默认的 TiDB Operator 版本，如果默认版本不是想要使用的版本，请在 `terraform.tfvars` 中配置 `operator_version`。

    配置完成后，使用 `terraform` 命令初始化并部署集群：

    {{< copyable "shell-regular" >}}

    ```shell
    terraform init
    ```

    `apply` 过程中需要输入 `yes` 来确认执行：

    {{< copyable "shell-regular" >}}

    ```shell
    terraform apply
    ```

    假如在运行 `terraform apply` 时出现报错，可根据报错信息（例如缺少权限）进行修复后再次运行 `terraform apply`。

    整个安装过程大约需要 5 至 10 分钟，安装完成后会输出集群的关键信息（想要重新查看这些信息，可以运行 `terraform output`）：

    ```
    Apply complete! Resources: 3 added, 0 changed, 1 destroyed.

    Outputs:

    bastion_ip = 47.96.174.214
    cluster_id = c2d9b20854a194f158ef2bc8ea946f20e
    kubeconfig_file = /tidb-operator/deploy/aliyun/credentials/kubeconfig
    monitor_endpoint = not_created
    region = cn-hangzhou
    ssh_key_file = /tidb-operator/deploy/aliyun/credentials/my-cluster-keyZ.pem
    tidb_endpoint = not_created
    tidb_version = v3.0.0
    vpc_id = vpc-bp1v8i5rwsc7yh8dwyep5
    ```

3. 用 `kubectl` 或 `helm` 对集群进行操作：

    {{< copyable "shell-regular" >}}

    ```shell
    export KUBECONFIG=$PWD/credentials/kubeconfig
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl version
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm ls
    ```

### 部署 TiDB 集群和监控

1. 准备 TidbCluster 和 TidbMonitor CR 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    cp manifests/db.yaml.example db.yaml && cp manifests/db-monitor.yaml.example db-monitor.yaml
    ```

    参考 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)和[集群配置文档](configure-a-tidb-cluster.md)完成 CR 文件配置。

    如果要部署 TiFlash，可以在 db.yaml 中配置 `spec.tiflash`，例如：

    ```yaml
    spec
      ...
      tiflash:
        baseImage: pingcap/tiflash
        maxFailoverCount: 3
        nodeSelector:
          dedicated: TIDB_CLUSTER_NAME-tiflash
        replicas: 1
        storageClaims:
        - resources:
            requests:
              storage: 100Gi
          storageClassName: local-volume
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: TIDB_CLUSTER_NAME-tiflash
    ```

    其他参数可以参考[集群配置文档](configure-a-tidb-cluster.md)进行配置。

    根据实际情况修改 `replicas`、`storageClaims[].resources.requests.storage`、`storageClassName`。

    > **警告：**
    >
    > 由于 TiDB Operator 会按照 `storageClaims` 列表中的配置**按顺序**自动挂载 PV，如果需要为 TiFlash 增加磁盘，请确保只在列表原有配置**最后添加**，并且**不能**修改列表中原有配置的顺序。

    如果要部署 TiCDC，可以在 db.yaml 中配置 `spec.ticdc`，例如：

    ```yaml
    spec
      ...
      ticdc:
        baseImage: pingcap/ticdc
        nodeSelector:
          dedicated: TIDB_CLUSTER_NAME-cdc
        replicas: 3
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: TIDB_CLUSTER_NAME-cdc
    ```

    根据实际情况修改 `replicas`。

    值得注意的是，如果需要部署企业版的 TiDB/PD/TiKV/TiFlash/TiCDC，需要将 db.yaml 中 `spec.<tidb/pd/tikv/tiflash/ticdc>.baseImage` 配置为企业版镜像，格式为 `pingcap/<tidb/pd/tikv/tiflash/ticdc>-enterprise`。

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

    > **注意：**
    >
    > * 请使用 ACK 部署过程中配置的 `tidb_cluster_name` 替换 `db.yaml` 和 `db-monitor.yaml` 文件中所有的 `TIDB_CLUSTER_NAME`。
    > * 请确保 ACK 部署过程中 PD、TiKV、TiFlash、TiCDC 或者 TiDB 节点的数量的值大于等于 `db.yaml` 中对应组件的 `replicas`。
    > * 请确保 `db-monitor.yaml` 中 `spec.initializer.version` 和 `db.yaml` 中 `spec.version` 一致，以保证监控显示正常。

2. 创建 `Namespace`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig create namespace ${namespace}
    ```

    > **注意：**
    >
    > `namespace` 是[命名空间](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)，可以起一个方便记忆的名字，比如和 `tidb_cluster_name` 相同的名称。

3. 部署 TiDB 集群：

  {{< copyable "shell-regular" >}}

  ```shell
  kubectl --kubeconfig credentials/kubeconfig create -f db.yaml -n ${namespace} &&
  kubectl --kubeconfig credentials/kubeconfig create -f db-monitor.yaml -n ${namespace}
  ```

> **注意：**
>
> 如果要将 TiDB 集群部署到 ARM64 机器上，可以参考[在 ARM64 机器上部署 TiDB 集群](deploy-cluster-on-arm64.md)。

## 连接数据库

通过堡垒机可连接 TiDB 集群进行测试，相关信息在安装完成后的输出中均可找到：

{{< copyable "shell-regular" >}}

```shell
ssh -i credentials/${cluster_name}-key.pem root@${bastion_ip}
```

{{< copyable "shell-regular" >}}

```shell
mysql -h ${tidb_lb_ip} -P 4000 -u root
```

`tidb_lb_ip` 为 TiDB Service 的 LoadBalancer IP。

> **注意：**
>
> * [MySQL 8.0 默认认证插件](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin)从 `mysql_native_password` 更新为 `caching_sha2_password`，因此如果使用 MySQL 8.0 客户端访问 TiDB 服务（TiDB 版本 < v4.0.7），并且用户账户有配置密码，需要显示指定 `--default-auth=mysql_native_password` 参数。
> * TiDB（v4.0.2 起）默认会定期收集使用情况信息，并将这些信息分享给 PingCAP 用于改善产品。若要了解所收集的信息详情及如何禁用该行为，请参见[遥测](https://docs.pingcap.com/zh/tidb/stable/telemetry)。

## 监控

你可以通过浏览器访问 `<monitor-lb>:3000` 地址查看 Grafana 监控指标。

`monitor-lb` 是集群 Monitor Service 的 LoadBalancer IP。

默认帐号密码为：

- 用户名：admin
- 密码：admin

> **警告：**
>
> 出于安全考虑，假如你已经或将要配置 VPN 用于访问 VPC，强烈建议将 `db-monitor.yaml` 文件里 `spec.grafana.service.annotations` 中的 `service.beta.kubernetes.io/alicloud-loadbalancer-address-type` 设置为 `intranet` 以禁止监控服务的公网访问。

## 升级 TiDB 集群

要升级 TiDB 集群，可以通过 `kubectl --kubeconfig credentials/kubeconfig edit tc ${tidb_cluster_name} -n ${namespace}` 修改 `spec.version`。

升级操作可能会执行较长时间，可以通过以下命令来持续观察进度：

{{< copyable "shell-regular" >}}

```
kubectl get pods --namespace ${namespace} -o wide --watch
```

## TiDB 集群扩容

若要扩容 TiDB 集群，可以在文件 `terraform.tfvars` 文件中设置 `tikv_count`、`tiflash_count`、`cdc_count` 或者 `tidb_count` 变量，然后运行 `terraform apply`，扩容对应组件节点数量，节点扩容完成后，通过 `kubectl --kubeconfig credentials/kubeconfig edit tc ${tidb_cluster_name} -n ${namespace}` 修改对应组件的 `replicas`。

> **注意：**
>
> - 由于缩容过程中无法确定会缩掉哪个节点，目前还不支持 TiDB 集群的缩容。
> - 扩容过程会持续几分钟，你可以通过 `kubectl --kubeconfig credentials/kubeconfig get po -n ${namespace} --watch` 命令持续观察进度。

## 销毁集群

可以参考[销毁 TiDB 集群](destroy-a-tidb-cluster.md#销毁-kubernetes-上的-tidb-集群)删除集群。

然后通过如下命令销毁 ACK 集群：

{{< copyable "shell-regular" >}}

```shell
terraform destroy
```

假如 Kubernetes 集群没有创建成功，那么在 destroy 时会出现报错，无法进行正常清理。此时需要手动将 Kubernetes 资源从本地状态中移除：

{{< copyable "shell-regular" >}}

```shell
terraform state list
```

{{< copyable "shell-regular" >}}

```shell
terraform state rm module.ack.alicloud_cs_managed_kubernetes.k8s
```

销毁集群操作需要执行较长时间。

> **注意：**
>
> 组件挂载的云盘需要在阿里云管理控制台中手动删除。

## 配置

### 配置 TiDB Operator

通过在 `terraform.tfvars` 中设置变量的值来配置 TiDB Operator，大多数配置项均能按照 `variable` 的注释理解语义后进行修改。需要注意的是，`operator_helm_values` 配置项允许为 TiDB Operator 提供一个自定义的 `values.yaml` 配置文件，示例如下：

- 在 `terraform.tfvars` 中设置 `operator_helm_values`：

    ```hcl
    operator_helm_values = "./my-operator-values.yaml"
    ```

- 在 `main.tf` 中设置 `operator_helm_values`：

    ```hcl
    operator_helm_values = file("./my-operator-values.yaml")
    ```

同时，在默认配置下 Terraform 脚本会创建一个新的 VPC，假如要使用现有的 VPC，可以在 `variable.tf` 中设置 `vpc_id`。注意，当使用现有 VPC 时，没有设置 vswitch 的可用区将不会部署 Kubernetes 节点。

### 配置 TiDB 集群

参考 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)和[集群配置文档](configure-a-tidb-cluster.md)修改 TiDB 集群配置。

## 管理多个 TiDB 集群

需要在一个 Kubernetes 集群下管理多个 TiDB 集群时，需要编辑 `./main.tf`，按实际需要新增 `tidb-cluster` 声明，示例如下：

```hcl
module "tidb-cluster-dev" {
  source = "../modules/aliyun/tidb-cluster"
  providers = {
    helm = helm.default
  }

  cluster_name = "dev-cluster"
  ack          = module.tidb-operator

  pd_count                   = 1
  tikv_count                 = 1
  tidb_count                 = 1
}

module "tidb-cluster-staging" {
  source = "../modules/aliyun/tidb-cluster"
  providers = {
    helm = helm.default
  }

  cluster_name = "staging-cluster"
  ack          = module.tidb-operator

  pd_count                   = 3
  tikv_count                 = 3
  tidb_count                 = 2
}
```

注意，多个 TiDB 集群之间 `cluster_name` 必须保持唯一。下面是 `tidb-cluster` 模块的所有可配置参数：

| 参数名 | 说明 | 默认值 |
| :----- | :---- | :----- |
| `ack` | 封装目标 Kubernetes 集群信息的结构体，必填 | `nil` |
| `cluster_name` | TiDB 集群名，必填且必须唯一 | `nil` |
| `tidb_version` | TiDB 集群版本 | `v3.0.1` |
| `tidb_cluster_chart_version` | `tidb-cluster` helm chart 的版本 | `v1.0.1` |
| `pd_count` | PD 节点数 | 3 |
| `pd_instance_type` | PD 实例类型 | `ecs.g5.large` |
| `tikv_count` | TiKV 节点数 | 3 |
| `tikv_instance_type` | TiKV 实例类型 | `ecs.i2.2xlarge` |
| `tiflash_count` | TiFlash 节点数 | 2 |
| `tiflash_instance_type` | TiFlash 实例类型 | `ecs.i2.2xlarge` |
| `cdc_count` | TiCDC 节点数 | 3 |
| `cdc_instance_type` | TiCDC 实例类型 | `ecs.c5.2xlarge` |
| `tidb_count` | TiDB 节点数 | 2 |
| `tidb_instance_type` | TiDB 实例类型 | `ecs.c5.4xlarge` |
| `monitor_instance_type` | 监控组件的实例类型 | `ecs.c5.xlarge` |
| `override_values` | TiDB 集群的 `values.yaml` 配置文件，通常通过 `file()` 函数从文件中读取 | `nil` |
| `local_exec_interpreter` | 执行命令行指令的解释器  | `["/bin/sh", "-c"]` |
| `create_tidb_cluster_release` | 是否通过 Helm 创建 TiDB 集群 | `false` |

## 管理多个 Kubernetes 集群

推荐针对每个 Kubernetes 集群都使用单独的 Terraform 模块进行管理（一个 Terraform Module 即一个包含 `.tf` 脚本的目录）。

`deploy/aliyun` 实际上是将 `deploy/modules` 中的数个可复用的 Terraform 脚本组合在了一起。当管理多个集群时（下面的操作在 `tidb-operator` 项目根目录下进行）：

1. 首先针对每个集群创建一个目录，如：

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p deploy/aliyun-staging
    ```

2. 参考 `deploy/aliyun` 的 `main.tf`，编写自己的脚本，下面是一个简单的例子：

    ```hcl
    provider "alicloud" {
        region     = ${REGION}
        access_key = ${ACCESS_KEY}
        secret_key = ${SECRET_KEY}
    }

    module "tidb-operator" {
        source     = "../modules/aliyun/tidb-operator"

        region          = ${REGION}
        access_key      = ${ACCESS_KEY}
        secret_key      = ${SECRET_KEY}
        cluster_name    = "example-cluster"
        key_file        = "ssh-key.pem"
        kubeconfig_file = "kubeconfig"
    }

    provider "helm" {
        alias    = "default"
        insecure = true
        install_tiller = false
        kubernetes {
            config_path = module.tidb-operator.kubeconfig_filename
        }
    }

    module "tidb-cluster" {
        source = "../modules/aliyun/tidb-cluster"
        providers = {
            helm = helm.default
        }

        cluster_name = "example-cluster"
        ack          = module.tidb-operator
    }

    module "bastion" {
        source = "../modules/aliyun/bastion"

        bastion_name             = "example-bastion"
        key_name                 = module.tidb-operator.key_name
        vpc_id                   = module.tidb-operator.vpc_id
        vswitch_id               = module.tidb-operator.vswitch_ids[0]
        enable_ssh_to_worker     = true
        worker_security_group_id = module.tidb-operator.security_group_id
    }
    ```

上面的脚本可以自由定制，比如，假如不需要堡垒机则可以移除 `module "bastion"` 相关声明。

你也可以直接拷贝 `deploy/aliyun` 目录，但要注意不能拷贝已经运行了 `terraform apply` 的目录，建议重新 clone 仓库再进行拷贝。

## 使用限制

目前，`pod cidr`，`service cidr` 和节点型号等配置在集群创建后均无法修改。
