---
title: Deploy TiDB on AWS EKS
summary: Learn how to deploy a TiDB cluster on AWS EKS.
category: how-to
---

# Deploy TiDB on AWS EKS

This document describes how to deploy a TiDB cluster on AWS EKS with your laptop (Linux or macOS) for development or testing.

## Prerequisites

Before deploying a TiDB cluster on AWS EKS, make sure the following requirements are satisfied:

* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73, to control AWS resources

    You must [configure](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) `awscli` before it can interact with AWS. The fastest way is using the `aws configure` command:

    {{< copyable "shell-regular" >}}

    ```shell
    aws configure
    ```

    Replace AWS Access Key ID and AWS Secret Access Key with your own keys:

    ```
    AWS Access Key ID [None]: AKIAIOSFODNN7EXAMPLE
    AWS Secret Access Key [None]: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
    Default region name [None]: us-west-2
    Default output format [None]: json
    ```

    > **Note:**
    >
    > The access key must have at least permissions to: create VPC, create EBS, create EC2 and create role.

* [terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) >= 0.12
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) >= 1.12
* [helm](https://helm.sh/docs/using_helm/#installing-the-helm-client) >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797)
* [jq](https://stedolan.github.io/jq/download/)
* [aws-iam-authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html) installed in `PATH`, to authenticate with AWS

    The easiest way to install `aws-iam-authenticator` is to download the prebuilt binary as shown below:

    Download the binary for Linux:

    {{< copyable "shell-regular" >}}

    ```shell
    curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator
    ```

    Or, download binary for macOS:

    {{< copyable "shell-regular" >}}

    ```shell
    curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/darwin/amd64/aws-iam-authenticator
    ```

    Then execute the following commands:

    {{< copyable "shell-regular" >}}

    ```shell
    chmod +x ./aws-iam-authenticator && \
    sudo mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator
    ```

## Deploy

This section describes how to deploy EKS, TiDB operator, TiDB cluster and monitor.

### Deploy EKS, TiDB Operator, and TiDB cluster node pool

Use the following commands to deploy EKS, TiDB Operator, and the TiDB cluster node pool.

Get the code from Github:

{{< copyable "shell-regular" >}}

```shell
git clone --depth=1 https://github.com/pingcap/tidb-operator && \
cd tidb-operator/deploy/aws
```

The default setup creates a new VPC and a `t2.micro` instance as the bastion machine, and an EKS cluster with following Amazon EC2 instances as worker nodes:

* 3 m5.xlarge instances for PD
* 3 c5d.4xlarge instances for TiKV
* 2 c5.4xlarge instances for TiDB
* 1 c5.2xlarge instance for monitor

You can create or modify `terraform.tfvars` to set the value of variables and configure the cluster as needed. See the variables that can be set and their descriptions in `variables.tf`.

The following is an example of how to configure the EKS cluster name, the TiDB cluster name, the TiDB Operator version, and the number of PD, TiKV and TiDB nodes:

```
default_cluster_pd_count   = 3
default_cluster_tikv_count = 3
default_cluster_tidb_count = 2
default_cluster_name = "tidb"
eks_name = "my-cluster"
operator_version = "v1.1.0-rc.1"
```

If you need to deploy TiFlash in the cluster, set `create_tiflash_node_pool = true` in `terraform.tfvars`. You can also configure the node count and instance type of the TiFlash node pool by modifying `cluster_tiflash_count` and `cluster_tiflash_instance_type`. By default, the value of `cluster_tiflash_count` is `2`, and the value of `cluster_tiflash_instance_type` is `i3.4xlarge`.

> **Note:**
>
> Check the `operator_version` in the `variables.tf` file for the default TiDB Operator version of the current scripts. If the default version is not your desired one, configure `operator_version` in `terraform.tfvars`.

After configuration, execute the `terraform` command to initialize and deploy the cluster:

{{< copyable "shell-regular" >}}

```shell
terraform init
```

{{< copyable "shell-regular" >}}

```shell
terraform apply
```

It might take 10 minutes or more to finish the process. After `terraform apply` is executed successfully, some useful information is printed to the console.

A successful deployment will give the output like:

```
Apply complete! Resources: 67 added, 0 changed, 0 destroyed.

Outputs:

bastion_ip = [
  "34.219.204.217",
]
default-cluster_monitor-dns = not_created
default-cluster_tidb-dns = not_created
eks_endpoint = https://9A9A5ABB8303DDD35C0C2835A1801723.yl4.us-west-2.eks.amazonaws.com
eks_version = 1.12
kubeconfig_filename = credentials/kubeconfig_my-cluster
region = us-west-2
```

You can use the `terraform output` command to get the output again.

> **Note:**
>
> EKS versions earlier than 1.14 do not support auto enabling cross-zone load balancing via Network Load Balancer (NLB). Therefore, unbalanced pressure distributed among TiDB instances can be expected in default settings. It is strongly recommended that you refer to [AWS Documentation](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-disable-crosszone-lb.html#enable-cross-zone) to manually enable cross-zone load balancing for a production environment.

### Deploy TiDB cluster and monitor

1. Prepare the TidbCluster and TidbMonitor CR files:

    {{< copyable "shell-regular" >}}

    ```shell
    cp manifests/db.yaml.example db.yaml && cp manifests/db-monitor.yaml.example db-monitor.yaml
    ```

    To complete the CR file configuration, refer to [API documentation](api-references.md) and [Configure a TiDB Cluster Using TidbCluster](configure-cluster-using-tidbcluster.md).

    To deploy TiFlash, configure `spec.tiflash` in `db.yaml` as follows:

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

    Modify `replicas`, `storageClaims[].resources.requests.storage`, and `storageClassName` according to your needs.

    > **Note:**
    >
    > * Replace all `CLUSTER_NAME` in `db.yaml` and `db-monitor.yaml` files with `default_cluster_name` configured during EKS deployment.
    > * Make sure that during EKS deployment, the number of PD, TiKV, TiFlash, or TiDB nodes is >= the value of the `replicas` field of the corresponding component in `db.yaml`.
    > * Make sure that `spec.initializer.version` in `db-monitor.yaml` and `spec.version` in `db.yaml` are the same to ensure normal monitor display.

2. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} create namespace ${namespace}
    ```

    > **Note:**
    >
    > A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. You can give it a name that is easy to memorize, such as the same name as `default_cluster_name`.

3. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} create -f db.yaml -n ${namespace} &&
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} create -f db-monitor.yaml -n ${namespace}
    ```

### Enable cross-zone load balancing for the LoadBalancer of the TiDB service

Due to an [issue](https://github.com/kubernetes/kubernetes/issues/82595) of AWS Network Load Balancer (NLB), the NLB created for the TiDB service cannot automatically enable cross-zone load balancing. You can manually enable it by taking the following steps:

1. Get the name of the NLB created for the TiDB service:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} get svc ${default_cluster_name}-tidb -n ${namespace}
    ```

    This is an example of the output:

    ```
    kubectl --kubeconfig credentials/kubeconfig_test get svc test-tidb -n test
    NAME        TYPE           CLUSTER-IP      EXTERNAL-IP                                                                     PORT(S)                          AGE
    tidb-tidb   LoadBalancer   172.20.39.180   a7aa544c49f914930b3b0532022e7d3c-83c0c97d8b659075.elb.us-west-2.amazonaws.com   4000:32387/TCP,10080:31486/TCP   3m46s
    ```

    In the value of the `EXTERNAL-IP` field, the first field that is separated by `-` is the name of NLB. In the example above, the NLB name is `a7aa544c49f914930b3b0532022e7d3c`.

2. Get the LoadBalancerArn for the NLB:

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 describe-load-balancers | grep ${LoadBalancerName}
    ```

    `${LoadBalancerName}` is the NLB name obtained in Step 1.

    This is an example of the output:

    ```
    aws elbv2 describe-load-balancers | grep a7aa544c49f914930b3b0532022e7d3c
              "LoadBalancerArn": "arn:aws:elasticloadbalancing:us-west-2:687123456789:loadbalancer/net/a7aa544c49f914930b3b0532022e7d3c/83c0c97d8b659075",
              "DNSName": "a7aa544c49f914930b3b0532022e7d3c-83c0c97d8b659075.elb.us-west-2.amazonaws.com",
              "LoadBalancerName": "a7aa544c49f914930b3b0532022e7d3c",
    ```

    The value of the `LoadBalancerArn` field is the NLB LoadBalancerArn.

3. View the NLB attributes:

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 describe-load-balancer-attributes --load-balancer-arn ${LoadBalancerArn}
    ```

    `${LoadBalancerArn}` is the NLB LoadBalancerArn obtained in Step 2.

    This is an example of the output:

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

    If the value of `load_balancing.cross_zone.enabled` is `false`, continue the next step to enable cross-zone load balancing.

4. Enable cross-zone load balancing for NLB:

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 modify-load-balancer-attributes --load-balancer-arn ${LoadBalancerArn} --attributes Key=load_balancing.cross_zone.enabled,Value=true
    ```

    `${LoadBalancerArn}` is the NLB LoadBalancerArn obtained in Step 2.

    This is an example of the output:

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

5. Confirm that the NLB cross-zone load balancing attribute is enabled:

    {{< copyable "shell-regular" >}}

    ```shell
    aws elbv2 describe-load-balancer-attributes --load-balancer-arn ${LoadBalancerArn}
    ```

    `${LoadBalancerArn}` is the NLB LoadBalancerArn obtained in Step 2.

    This is an example of the output:

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

    Confirm that the value of `load_balancing.cross_zone.enabled` is `true`.

## Access the database

After deploying the cluster, to access the deployed TiDB cluster, use the following commands to first `ssh` into the bastion machine, and then connect it via MySQL client (replace the `${}` parts with values from the output):

{{< copyable "shell-regular" >}}

```shell
ssh -i credentials/${eks_name}.pem centos@${bastion_ip}
```

{{< copyable "shell-regular" >}}

```shell
mysql -h ${tidb_lb} -P 4000 -u root
```

The default value of `eks_name` is `my-cluster`. If the DNS name is not resolvable, be patient and wait a few minutes.

`tidb_lb` is the LoadBalancer of TiDB Service. To check this value, run `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get svc ${default_cluster_name}-tidb -n ${namespace}` and view the `EXTERNAL-IP` field in the output information.

You can interact with the EKS cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_${eks_name}` in the following two ways.

- By specifying `--kubeconfig` argument:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig_${eks_name} get po -n ${namespace}
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm --kubeconfig credentials/kubeconfig_${eks_name} ls
    ```

- Or by setting the `KUBECONFIG` environment variable:

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

## Monitor

You can access the `<monitor-lb>:3000` address (printed in outputs) using your web browser to view monitoring metrics.

`monitor-lb` is the LoadBalancer of the cluster Monitor Service. To check this value, run `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get svc ${default_cluster_name}-grafana -n ${namespace}` and view the `EXTERNAL-IP` field in the output information.

The initial Grafana login credentials are:

- User: admin
- Password: admin

## Upgrade

To upgrade the TiDB cluster, edit the `spec.version` by `kubectl --kubeconfig credentials/kubeconfig_${eks_name} edit tc ${default_cluster_name} -n ${namespace}`.

The upgrading doesn't finish immediately. You can watch the upgrading progress by `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get po -n ${namespace} --watch`.

## Scale

To scale out the TiDB cluster, modify the `default_cluster_tikv_count`, `cluster_tiflash_count`, or `default_cluster_tidb_count` variable in the `terraform.tfvars` file to your desired count, and then run `terraform apply` to scale out the number of the corresponding component nodes.

After the scaling, modify the `replicas` of the corresponding component by the following command:

{{< copyable "shell-regular" >}}

```
kubectl --kubeconfig credentials/kubeconfig_${eks_name} edit tc ${default_cluster_name} -n ${namespace}
```

For example, to scale out the TiDB nodes, you can modify the number of TiDB instances from 2 to 4:

```hcl
default_cluster_tidb_count = 4
```

After the nodes scale out, modify the `spec.tidb.replicas` in `TidbCluster` to scale out the Pod.

> **Note:**
>
> Currently, scaling in is NOT supported because we cannot determine which node to scale in.
> Scaling out needs a few minutes to complete, you can watch the scaling out progress by `kubectl --kubeconfig credentials/kubeconfig_${eks_name} get po -n ${namespace} --watch`.

## Customize

This section describes how to customize the AWS related resources and TiDB Operator.

### Customize AWS related resources

An Amazon EC2 instance is also created by default as the bastion machine to connect to the created TiDB cluster. This is because the TiDB service is exposed as an [Internal Elastic Load Balancer](https://aws.amazon.com/blogs/aws/internal-elastic-load-balancers/). The EC2 instance has MySQL and Sysbench pre-installed, so you can use SSH to log into the EC2 instance and connect to TiDB using the ELB endpoint. You can disable the bastion instance creation by setting `create_bastion` to `false` if you already have an EC2 instance in the VPC.

### Customize TiDB Operator

To customize the TiDB Operator, modify the `operator_values` parameter in `terraform.tfvars` to pass the customized contents of `values.yaml`. For example:

```hcl
operator_values = "./operator_values.yaml"
```

## Manage multiple TiDB clusters

An instance of `tidb-cluster` module corresponds to a TiDB cluster in the EKS cluster. If you want to create a node pool for a new TiDB cluster, edit `./cluster.tf` and add a new instance of `tidb-cluster` module:

```hcl
module example-cluster {
  source = "../modules/aws/tidb-cluster"
  eks = local.eks
  subnets = local.subnets
  region = var.region
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

> **Note:**
>
> The `cluster_name` of each cluster must be unique.

When you finish the modification, execute `terraform init` and `terraform apply` to create the nodes pool for the TiDB cluster.

Finally, [deploy TiDB cluster and monitor](#deploy-tidb-cluster-and-monitor).

## Destroy clusters

1. [Destroy the TiDB clusters](destroy-a-tidb-cluster.md).

2. Destroy the EKS cluster by the following command:

    {{< copyable "shell-regular" >}}

    ``` shell
    terraform destroy
    ```

    If the following error occurs during `terraform destroy`:

    ```
    Error: Get http://localhost/apis/apps/v1/namespaces/kube-system/deployments/tiller-deploy: dial tcp [::1]:80: connect: connection refused
    ```

    Run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    terraform state rm module.tidb-operator.helm_release.tidb-operator
    ```

    And then run `terraform destroy` again.

> **Note:**
>
> * This will destroy your EKS cluster.
> * If you do not need the data on the volumes anymore, you have to manually delete the EBS volumes in AWS console after running `terraform destroy`.

## Manage multiple Kubernetes clusters

This section describes the best practice to manage multiple Kubernetes clusters, each with one or more TiDB clusters installed.

The Terraform module in our case typically combines several sub-modules:

- A `tidb-operator` module, which creates the EKS cluster and [deploy TiDB Operator](deploy-tidb-operator.md) on the EKS cluster
- A `tidb-cluster` module, which creates the resource pool required by the TiDB cluster
- A `VPC` module, a `bastion` module and a `key-pair` module that are dedicated to TiDB on AWS

The best practice for managing multiple Kubernetes clusters is creating a new directory for each of your Kubernetes clusters, and combine the above modules according to your needs via Terraform scripts, so that the Terraform states among clusters do not interfere with each other, and it is convenient to expand. Here's an example:

{{< copyable "shell-regular" >}}

```shell
# assume we are in the project root
mkdir -p deploy/aws-staging
vim deploy/aws-staging/main.tf
```

The content of `deploy/aws-staging/main.tf` could be:

```hcl
provider "aws" {
  region = "us-west-1"
}

# Creates an SSH key to log in the bastion and the Kubernetes node
module "key-pair" {
  source = "../modules/aws/key-pair"

  name = "another-eks-cluster"
  path = "${path.cwd}/credentials/"
}

# Provisions a VPC
module "vpc" {
  source = "../modules/aws/vpc"

  vpc_name = "another-eks-cluster"
}

# Provisions an EKS control plane with TiDB Operator installed
module "tidb-operator" {
  source = "../modules/aws/tidb-operator"

  eks_name           = "another-eks-cluster"
  config_output_path = "credentials/"
  subnets            = module.vpc.private_subnets
  vpc_id             = module.vpc.vpc_id
  ssh_key_name       = module.key-pair.key_name
}

# HACK: enforces Helm to depend on the EKS
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

# Provisions a node pool for the TiDB cluster in the EKS cluster
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

# Provisions a node pool for another TiDB cluster in the EKS cluster
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

# Provisions a bastion machine to access the TiDB service and worker nodes
module "bastion" {
  source = "../modules/aws/bastion"

  bastion_name             = "another-eks-cluster-bastion"
  key_name                 = module.key-pair.key_name
  public_subnets           = module.vpc.public_subnets
  vpc_id                   = module.vpc.vpc_id
  target_security_group_id = module.tidb-operator.eks.worker_security_group_id
  enable_ssh_to_workers    = true
}

output "bastion_ip" {
  description = "Bastion IP address"
  value       = module.bastion.bastion_ip
}
```

As shown in the code above, you can omit most of the parameters in each of the module calls because there are reasonable defaults, and it is easy to customize the configuration. For example, just delete the bastion module call if you do not need it.

To customize each field, you can refer to the default Terraform module. Also, you can always refer to the `variables.tf` file of each module to learn about all the available parameters.

In addition, you can easily integrate these modules into your own Terraform workflow. If you are familiar with Terraform, this is our recommended way of use.

> **Note:**
>
> * When creating a new directory, please pay attention to its relative path to Terraform modules, which affects the `source` parameter during module calls.
> * If you want to use these modules outside the tidb-operator project, make sure you copy the whole `modules` directory and keep the relative path of each module inside the directory unchanged.
> * Due to limitation [hashicorp/terraform#2430](https://github.com/hashicorp/terraform/issues/2430#issuecomment-370685911) of Terraform, the hack processing of Helm provider is necessary in the above example. It is recommended that you keep it in your own Terraform scripts.

If you are unwilling to write Terraform code, you can also copy the `deploy/aws` directory to create new Kubernetes clusters. But note that you cannot copy a directory that you have already run `terraform apply` against, when the Terraform state already exists in local. In this case, it is recommended to clone a new repository before copying the directory.
