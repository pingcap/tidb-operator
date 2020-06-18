---
title: 在 AWS EKS 上部署 TiDB 
summary: 介绍如何在 AWS EKS (Elastic Kubernetes Service) 上部署 TiDB 集群。
category: how-to
---

# 在 AWS EKS 上部署 TiDB 集群

本文介绍了如何使用个人电脑（Linux 或 macOS 系统）在 AWS EKS (Elastic Kubernetes Service) 上部署 TiDB 集群。

## 环境配置准备

部署前，请确认已安装以下软件并完成配置：

* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73，控制 AWS 资源

    要与 AWS 交互，必须[配置 `awscli`](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html)。最快的方式是使用 `aws configure` 命令:

    {{< copyable "shell-regular" >}}

    ``` shell
    aws configure
    ```

    替换下面的 AWS Access Key ID 和 AWS Secret Access Key：

    ```
    AWS Access Key ID [None]: AKIAIOSFODNN7EXAMPLE
    AWS Secret Access Key [None]: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
    Default region name [None]: us-west-2
    Default output format [None]: json
    ```

    > **注意：**
    >
    > Access key 必须至少具有以下权限：创建 VPC、创建 EBS、创建 EC2 和创建 Role。

* [terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) >= 0.12
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.12
* [helm](https://helm.sh/docs/using_helm/#installing-the-helm-client) >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797)
* [jq](https://stedolan.github.io/jq/download/)
* [aws-iam-authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html)，AWS 权限鉴定工具，确保安装在 `PATH` 路径下。

    最简单的安装方法是下载编译好的二进制文件 `aws-iam-authenticator`，如下所示。

    Linux 用户下载二进制文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator
    ```

    macOS 用户下载二进制文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/darwin/amd64/aws-iam-authenticator
    ```

    二进制文件下载完成后，执行以下操作：

    {{< copyable "shell-regular" >}}

    ``` shell
    chmod +x ./aws-iam-authenticator && \
    sudo mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator
    ```

## 部署集群

### 部署 EKS、TiDB Operator 和 TiDB 集群节点池

使用如下步骤部署 EKS、TiDB Operator 和 TiDB 集群节点池。

从 Github 克隆代码并进入指定路径：

{{< copyable "shell-regular" >}}

``` shell
git clone --depth=1 https://github.com/pingcap/tidb-operator && \
cd tidb-operator/deploy/aws
```

EKS 部署脚本中提供两种部署方式：

* 部署一套快速体验的 Demo 集群。

    Demo 部署会创建一个新的 VPC、一个 t2.micro 实例作为堡垒机，并包含以下 ec2 实例作为工作节点的 EKS 集群：

    * 1 台 c5.large 实例，部署 PD
    * 3 台 c5d.large 实例，部署 TiKV
    * 1 台 c5.large 实例，部署 TiDB
    * 1 台 c5.large 实例，部署监控组件

    执行如下命令使用 Demo 配置：

    {{< copyable "shell-regular" >}}

    ``` shell
    cp demo.tfvars terraform.tfvars
    ```

* 部署一套生产环境集群。

    生产环境集群部署会创建一个新的 VPC、一个 t2.micro 实例作为堡垒机，并包含以下 ec2 实例作为工作节点的 EKS 集群：

    * 3 台 c5.xlarge 实例，部署 PD
    * 3 台 i3.4xlarge 实例，部署 TiKV
    * 2 台 c5.2xlarge 实例，部署 TiDB
    * 1 台 c5.2xlarge 实例，部署监控组件

    执行如下命令使用生产环境配置：

    {{< copyable "shell-regular" >}}

    ``` shell
    cp prod.tfvars terraform.tfvars
    ```

除了上述配置，可以编辑 `terraform.tfvars`，在其中设置变量的值，按需配置集群，可以通过 `variables.tf` 查看有哪些变量可以设置以及各变量的详细描述。

例如，下面示例配置 EKS 集群名称，TiDB 集群名称，TiDB Operator 版本及 PD、TiKV 和 TiDB 节点的数量：

```
default_cluster_pd_count   = 3
default_cluster_tikv_count = 3
default_cluster_tidb_count = 2
default_cluster_name = "tidb"
eks_name = "my-cluster"
operator_version = "v1.1.0"
```

如果需要在集群中部署 TiFlash，需要在 `terraform.tfvars` 中设置 `create_tiflash_node_pool = true`，也可以设置 `cluster_tiflash_count` 和 `cluster_tiflash_instance_type` 来配置 TiFlash 节点池的节点数量和实例类型，`cluster_tiflash_count` 默认为 `2`，`cluster_tiflash_instance_type` 默认为 `i3.4xlarge`。

如果需要在集群中部署 TiCDC，需要在 `terraform.tfvars` 中设置 `create_cdc_node_pool = true`，也可以设置 `cluster_cdc_count` 和 `cluster_cdc_instance_type` 来配置 TiCDC 节点池的节点数量和实例类型，`cluster_cdc_count` 默认为 `3`，`cluster_cdc_instance_type` 默认为 `c5.2xlarge`。

> **注意：**
>
> 请通过 `variables.tf` 文件中的 `operator_version` 确认当前版本脚本中默认的 TiDB Operator 版本，如果默认版本不是想要使用的版本，请在 `terraform.tfvars` 中配置 `operator_version`。

配置完成后，使用 `terraform` 命令初始化并部署集群：

{{< copyable "shell-regular" >}}

``` shell
terraform init
```

{{< copyable "shell-regular" >}}

``` shell
terraform apply
```

> **注意：**
>
> `terraform apply` 过程中必须输入 "yes" 才能继续。

整个过程可能至少需要 10 分钟。`terraform apply` 执行成功后，控制台会输出类似如下的信息：

```
Apply complete! Resources: 67 added，0 changed，0 destroyed.

Outputs:

bastion_ip = [
  "34.219.204.217",
]
default-cluster_monitor-dns = not_created
default-cluster_tidb-dns = not_created
eks_endpoint = https://9A9A5ABB8303DDD35C0C2835A1801723.yl4.us-west-2.eks.amazonaws.com
eks_version = 1.12
kubeconfig_filename = credentials/kubeconfig_my-cluster
region = us-west-21
```

你可以通过 `terraform output` 命令再次获取上面的输出信息。

> **注意：**
>
> 1.14 版本以前的 EKS 不支持自动开启 NLB 跨可用区负载均衡，因此默认配置下 会出现各台 TiDB 实例压力不均衡额状况。生产环境下，强烈建议参考 [AWS 官方文档](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-disable-crosszone-lb.html#enable-cross-zone)手动开启 NLB 的跨可用区负载均衡。

### 部署 TiDB 集群和监控

1. 准备 TidbCluster 和 TidbMonitor CR 文件：

    {{< copyable "shell-regular" >}}

    ```shell
    cp manifests/db.yaml.example db.yaml && cp manifests/db-monitor.yaml.example db-monitor.yaml
    ```

    参考 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)和[集群配置文档](configure-cluster-using-tidbcluster.md)完成 CR 文件配置。

    如果要部署 TiFlash，可以在 db.yaml 中配置 `spec.tiflash`，例如：

    ```yaml
    spec:
      ...
      tiflash:
        baseImage: pingcap/tiflash
        maxFailoverCount: 3
        nodeSelector:
          dedicated: CLUSTER_NAME-tiflash
        replicas: 1
        storageClaims:
        - resources:
            requests:
              storage: 100Gi
          storageClassName: local-storage
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: CLUSTER_NAME-tiflash
    ```

    根据实际情况修改 `replicas`、`storageClaims[].resources.requests.storage`、`storageClassName`。

    如果要部署 TiCDC，可以在 db.yaml 中配置 `spec.ticdc`，例如：

    ```yaml
    spec:
      ...
      tiflash:
        baseImage: pingcap/ticdc
        nodeSelector:
          dedicated: CLUSTER_NAME-cdc
        replicas: 3
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: CLUSTER_NAME-cdc
    ```

    根据实际情况修改 `replicas`。

    > **注意：**
    >
    > * 请使用 EKS 部署过程中配置的 `default_cluster_name` 替换 `db.yaml` 和 `db-monitor.yaml` 文件中所有的 `CLUSTER_NAME`。
    > * 请确保 EKS 部署过程中 PD、TiKV、TiFlash、TiCDC 或者 TiDB 节点的数量的值大于等于 `db.yaml` 中对应组件的 `replicas`。
    > * 请确保 `db-monitor.yaml` 中 `spec.initializer.version` 和 `db.yaml` 中 `spec.version` 一致，以保证监控显示正常。

2. 创建 `Namespace`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} create namespace ${namespace}
    ```

    > **注意：**
    >
    > `namespace` 是[命名空间](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)，可以起一个方便记忆的名字，比如和 `default_cluster_name` 相同的名称。

3. 部署 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} create -f db.yaml -n ${namespace} &&
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} create -f db-monitor.yaml -n ${namespace}
    ```

### 为 TiDB 服务 LoadBalancer 开启 Cross-Zone Load Balancing

由于 AWS Network Load Balancer (NLB) [问题](https://github.com/kubernetes/kubernetes/issues/82595)，为 TiDB 服务创建的 NLB 无法自动开启 Cross-Zone Load Balancing，请参考以下步骤手动开启：

1. 获取 TiDB 服务 NLB 名字：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} get svc ${default_cluster_name}-tidb -n ${namespace}
    ```

    示例：

    ```
    kubectl --kubeconfig credentials/kubeconfig_test get svc test-tidb -n test
    NAME        TYPE           CLUSTER-IP      EXTERNAL-IP                                                                     PORT(S)                          AGE
    tidb-tidb   LoadBalancer   172.20.39.180   a7aa544c49f914930b3b0532022e7d3c-83c0c97d8b659075.elb.us-west-2.amazonaws.com   4000:32387/TCP,10080:31486/TCP   3m46s
    ```

    `EXTERNAL-IP` 字段值中以 `-` 分隔的第一个字段即为 NLB 名字，上述示例中 `a7aa544c49f914930b3b0532022e7d3c` 即为 NLB 名字。

2. 获取 NLB LoadBalancerArn：

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 describe-load-balancers | grep ${LoadBalancerName}
    ```

    `${LoadBalancerName}` 为第一步获取的 NLB 名字。

    示例：

    ```
    aws elbv2 describe-load-balancers | grep a7aa544c49f914930b3b0532022e7d3c
              "LoadBalancerArn": "arn:aws:elasticloadbalancing:us-west-2:687123456789:loadbalancer/net/a7aa544c49f914930b3b0532022e7d3c/83c0c97d8b659075",
              "DNSName": "a7aa544c49f914930b3b0532022e7d3c-83c0c97d8b659075.elb.us-west-2.amazonaws.com",
              "LoadBalancerName": "a7aa544c49f914930b3b0532022e7d3c",
    ```

    `LoadBalancerArn` 字段值即为 NLB LoadBalancerArn。

3. 查看 NLB 属性：

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 describe-load-balancer-attributes --load-balancer-arn ${LoadBalancerArn}
    ```

    `${LoadBalancerArn}` 为第二步获取的 NLB LoadBalancerArn。

    示例：

    ```
    aws elbv2 describe-load-balancer-attributes --load-balancer-arn "arn:aws:elasticloadbalancing:us-west-2:687123456789:loadbalancer/net/a7aa544c49f914930b3b0532022e7d3c/83c0c97d8b659075"
    {
      "Attributes": [
          {
              "Key": "access_logs.s3.enabled",
              "Value": "false"
          },
          {
              "Key": "load_balancing.cross_zone.enabled",
              "Value": "false"
          },
          {
              "Key": "access_logs.s3.prefix",
              "Value": ""
          },
          {
              "Key": "deletion_protection.enabled",
              "Value": "false"
          },
          {
              "Key": "access_logs.s3.bucket",
              "Value": ""
          }
      ]
    }
    ```

    如果 `load_balancing.cross_zone.enabled` 的值为 `false`，继续下一步，为 NLB 开启 Cross-Zone Load Balancing。

4. 为 NLB 开启 Cross-Zone Load Balancing：

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 modify-load-balancer-attributes --load-balancer-arn ${LoadBalancerArn} --attributes Key=load_balancing.cross_zone.enabled,Value=true
    ```

    `${LoadBalancerArn}` 为第二步获取的 NLB LoadBalancerArn。

    示例：

    ```
    aws elbv2 modify-load-balancer-attributes --load-balancer-arn "arn:aws:elasticloadbalancing:us-west-2:687123456789:loadbalancer/net/a7aa544c49f914930b3b0532022e7d3c/83c0c97d8b659075" --attributes Key=load_balancing.cross_zone.enabled,Value=true
    {
      "Attributes": [
          {
              "Key": "load_balancing.cross_zone.enabled",
              "Value": "true"
          },
          {
              "Key": "access_logs.s3.enabled",
              "Value": "false"
          },
          {
              "Key": "access_logs.s3.prefix",
              "Value": ""
          },
          {
              "Key": "deletion_protection.enabled",
              "Value": "false"
          },
          {
              "Key": "access_logs.s3.bucket",
              "Value": ""
          }
      ]
    }
    ```

5. 确认 NLB Cross-Zone Load Balancing 属性已经开启：

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 describe-load-balancer-attributes --load-balancer-arn ${LoadBalancerArn}
    ```

    `${LoadBalancerArn}` 为第二步获取的 NLB LoadBalancerArn。

    示例：

    ```
    aws elbv2 describe-load-balancer-attributes --load-balancer-arn "arn:aws:elasticloadbalancing:us-west-2:687123456789:loadbalancer/net/a7aa544c49f914930b3b0532022e7d3c/83c0c97d8b659075"
    {
      "Attributes": [
          {
              "Key": "access_logs.s3.enabled",
              "Value": "false"
          },
          {
              "Key": "load_balancing.cross_zone.enabled",
              "Value": "true"
          },
          {
              "Key": "access_logs.s3.prefix",
              "Value": ""
          },
          {
              "Key": "deletion_protection.enabled",
              "Value": "false"
          },
          {
              "Key": "access_logs.s3.bucket",
              "Value": ""
          }
      ]
    }
    ```

    确认 `load_balancing.cross_zone.enabled` 的值为 `true`。

## 访问数据库

集群部署完成后，可先通过 `ssh` 远程连接到堡垒机，再通过 MySQL client 来访问 TiDB 集群。

所需命令如下（用上面的输出信息替换 `${}` 部分内容)：

{{< copyable "shell-regular" >}}

```shell
ssh -i credentials/${eks_name}.pem centos@${bastion_ip}
```

{{< copyable "shell-regular" >}}

```shell
mysql -h ${tidb_lb} -P 4000 -u root
```

`eks_name` 默认为 `my-cluster`。如果 DNS 名字无法解析，请耐心等待几分钟。

`tidb_lb` 为 TiDB Service 的 LoadBalancer，可以通过 `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get svc ${default_cluster_name}-tidb -n ${namespace}` 输出中的 `EXTERNAL-IP` 字段查看。

你还可以通过 `kubectl` 和 `helm` 命令使用 kubeconfig 文件 `credentials/kubeconfig_${eks_name}` 和 EKS 集群交互，主要有两种方式，如下所示。

- 指定 --kubeconfig 参数：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} get po -n ${namespace}
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm --kubeconfig credentials/kubeconfig_${eks_name} ls
    ```

- 或者，设置 KUBECONFIG 环境变量：

    {{< copyable "shell-regular" >}}

    ```shell
    export KUBECONFIG=$PWD/credentials/kubeconfig_${eks_name}
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n ${namespace}
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm ls
    ```

## Grafana 监控

你可以通过浏览器访问 `<monitor-lb>:3000` 地址查看 Grafana 监控指标。

`monitor-lb` 是集群 Monitor Service 的 LoadBalancer，可以通过 `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get svc ${default_cluster_name}-grafana -n ${namespace}` 输出中的 `EXTERNAL-IP` 字段查看。

Grafana 默认登录信息：

- 用户名：admin
- 密码：admin

## 升级 TiDB 集群

要升级 TiDB 集群，可以通过 `kubectl --kubeconfig credentials/kubeconfig_${eks_name} edit tc ${default_cluster_name} -n ${namespace}` 修改 `spec.version`。

升级过程会持续一段时间，你可以通过 `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get po -n ${namespace} --watch` 命令持续观察升级进度。

## 扩容 TiDB 集群

若要扩容 TiDB 集群，可以在文件 `terraform.tfvars` 文件中设置 `default_cluster_tikv_count`、`cluster_tiflash_count`、`cluster_cdc_count` 或者 `default_cluster_tidb_count` 变量，然后运行 `terraform apply`，扩容对应组件节点数量，节点扩容完成后，通过 `kubectl --kubeconfig credentials/kubeconfig_${eks_name} edit tc ${default_cluster_name} -n ${namespace}` 修改对应组件的 `replicas`。

例如，可以将 `default_cluster_tidb_count` 从 2 改为 4 以扩容 TiDB 节点：

```hcl
default_cluster_tidb_count = 4
```

节点扩容完成后，修改 `TidbCluster` 中 `spec.tidb.replicas` 扩容 Pod。

> **注意：**
>
> - 由于缩容过程中无法确定会缩掉哪个节点，目前还不支持 TiDB 集群的缩容。
> - 扩容过程会持续几分钟，你可以通过 `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get po -n ${namespace} --watch` 命令持续观察进度。

## 自定义

### 自定义 AWS 相关的资源

由于 TiDB 服务通过 [Internal Elastic Load Balancer](https://aws.amazon.com/blogs/aws/internal-elastic-load-balancers/) 暴露，默认情况下，会创建一个 Amazon EC2 实例作为堡垒机，访问创建的 TiDB 集群。堡垒机上预装了 MySQL 和 Sysbench，所以你可以通过 SSH 方式登陆到堡垒机后通过 ELB 访问 TiDB。如果你的 VPC 中已经有了类似的 EC2 实例，你可以通过设置 `create_bastion` 为 `false` 禁掉堡垒机的创建。

### 自定义 TiDB Operator

你可以在 `terraform.tfvars` 中设置 `operator_values` 参数传入自定义的 `values.yaml` 内容来配置 TiDB Operator。示例如下：

```hcl
operator_values = "./operator_values.yaml"
```

## 管理多个 TiDB 集群

一个 `tidb-cluster` 模块的实例对应一个 TiDB 集群，你可以通过编辑 `clusters.tf` 添加新的 `tidb-cluster` 模块实例来为新的 TiDB 集群创建节点池，示例如下：

```hcl
module example-cluster {
  source = "../modules/aws/tidb-cluster"
  eks = local.eks
  subnets = local.subnets
  region  = var.region
  cluster_name    = "example"
  ssh_key_name                  = module.key-pair.key_name
  pd_count                      = 1
  pd_instance_type              = "c5.large"
  tikv_count                    = 1
  tikv_instance_type            = "c5d.large"
  tidb_count                    = 1
  tidb_instance_type            = "c4.large"
  monitor_instance_type         = "c5.large"
  create_tidb_cluster_release   = false
}
```

> **注意：**
>
> `cluster_name` 必须是唯一的。

修改完成后，执行 `terraform init` 和 `terraform apply` 为集群创建节点池。

最后，参考[部署 TiDB 集群和监控](#部署-TiDB-集群和监控) 部署新集群及其监控。

## 销毁集群

可以参考[销毁 TiDB 集群](destroy-a-tidb-cluster.md#销毁-kubernetes-上的-tidb-集群)删除集群。

然后通过如下命令销毁 EKS 集群：

{{< copyable "shell-regular" >}}

```shell
terraform destroy
```

如果执行 `terraform destroy` 过程中遇到如下错误：

```
Error: Get http://localhost/apis/apps/v1/namespaces/kube-system/deployments/tiller-deploy: dial tcp [::1]:80: connect: connection refused
```

请执行以下命令：

{{< copyable "shell-regular" >}}

```shell
terraform state rm module.tidb-operator.helm_release.tidb-operator
```

然后再次执行 `terraform destroy`。

> **注意：**
>
> * 该操作会销毁 EKS 集群。
> * 如果你不再需要存储卷中的数据，在执行 `terraform destroy` 后，你需要在 AWS 控制台手动删除 EBS 卷。

## 管理多个 Kubernetes 集群

本节详细介绍了如何管理多个 Kubernetes 集群（EKS），并在每个集群上部署一个或更多 TiDB 集群。

上述文档中介绍的 Terraform 脚本组合了多个 Terraform 模块：

- `tidb-operator` 模块，用于创建 EKS 集群并在 EKS 集群上安装配置 [TiDB Operator](deploy-tidb-operator.md)。
- `tidb-cluster` 模块，用于创建 TiDB 集群所需的资源池。
- EKS 上的 TiDB 集群专用的 `vpc` 模块、`key-pair`模块和`bastion` 模块

管理多个 Kubernetes 集群的最佳实践是为每个 Kubernetes 集群创建一个单独的目录，并在新目录中自行组合上述 Terraform 模块。这种方式能够保证多个集群间的 Terraform 状态不会互相影响，也便于自由定制和扩展。下面是一个例子：

{{< copyable "shell-regular" >}}

```shell
mkdir -p deploy/aws-staging
vim deploy/aws-staging/main.tf
```

`deploy/aws-staging/main.tf` 的内容可以是：

```hcl
provider "aws" {
  region = "us-west-1"
}

# 创建一个 ssh key，用于登录堡垒机和 Kubernetes 节点
module "key-pair" {
  source = "../modules/aws/key-pair"

  name = "another-eks-cluster"
  path = "${path.cwd}/credentials/"
}

# 创建一个新的 VPC
module "vpc" {
  source = "../modules/aws/vpc"

  vpc_name = "another-eks-cluster"
}

# 在上面的 VPC 中创建一个 EKS 并部署 tidb-operator
module "tidb-operator" {
  source = "../modules/aws/tidb-operator"

  eks_name           = "another-eks-cluster"
  config_output_path = "credentials/"
  subnets            = module.vpc.private_subnets
  vpc_id             = module.vpc.vpc_id
  ssh_key_name       = module.key-pair.key_name
}

# 特殊处理，确保 helm 操作在 EKS 创建完毕后进行
resource "local_file" "kubeconfig" {
  depends_on        = [module.tidb-operator.eks]
  sensitive_content = module.tidb-operator.eks.kubeconfig
  filename          = module.tidb-operator.eks.kubeconfig_filename
}
provider "helm" {
  alias    = "eks"
  insecure = true
  install_tiller = false
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

# 在上面的 EKS 集群上为 TiDB 集群创建节点池
module "tidb-cluster-a" {
  source = "../modules/aws/tidb-cluster"
  providers = {
    helm = "helm.eks"
  }

  cluster_name = "tidb-cluster-a"
  eks          = module.tidb-operator.eks
  ssh_key_name = module.key-pair.key_name
  subnets      = module.vpc.private_subnets
}

# 在上面的 EKS 集群上为另一个 TiDB 集群创建节点池
module "tidb-cluster-b" {
  source = "../modules/aws/tidb-cluster"
  providers = {
    helm = "helm.eks"
  }
  
  cluster_name = "tidb-cluster-b"
  eks          = module.tidb-operator.eks
  ssh_key_name = module.key-pair.key_name
  subnets      = module.vpc.private_subnets
}

# 创建一台堡垒机
module "bastion" {
  source = "../modules/aws/bastion"

  bastion_name             = "another-eks-cluster-bastion"
  key_name                 = module.key-pair.key_name
  public_subnets           = module.vpc.public_subnets
  vpc_id                   = module.vpc.vpc_id
  target_security_group_id = module.tidb-operator.eks.worker_security_group_id
  enable_ssh_to_workers    = true
}

# 输出堡垒机 IP
output "bastion_ip" {
  description = "Bastion IP address"
  value       = module.bastion.bastion_ip
}
```

上面的例子很容易进行定制，比如，假如你不需要堡垒机，便可以删去对 `bastion` 模块的调用。同时，项目中提供的 Terraform 模块均设置了合理的默认值，因此在调用这些 Terraform 模块时，你可以略去大部分的参数。

你可以参考默认的 Terraform 脚本来定制每个模块的参数，也可以参考每个模块的 `variables.tf` 文件来了解所有可配置的参数。

另外，这些 Terraform 模块可以很容易地集成到你自己的 Terraform 工作流中。假如你对 Terraform 非常熟悉，这也是我们推荐的一种使用方式。

> **注意：**
>
> * 由于 Terraform 本身的限制（[hashicorp/terraform#2430](https://github.com/hashicorp/terraform/issues/2430#issuecomment-370685911)），在你自己的 Terraform 脚本中，也需要保留上述例子中对 `helm provider` 的特殊处理。
> * 创建新目录时，需要注意与 Terraform 模块之间的相对路径，这会影响调用模块时的 `source` 参数。
> * 假如你想在 tidb-operator 项目之外使用这些模块，你需要确保 `modules` 目录中的所有模块的相对路径保持不变。

假如你不想自己写 Terraform 代码，也可以直接拷贝 `deploy/aws` 目录来创建新的 Kubernetes 集群。但要注意不能拷贝已经运行过 `terraform apply` 的目录（已经有 Terraform 的本地状态）。这种情况下，推荐在拷贝前克隆一个新的仓库。
