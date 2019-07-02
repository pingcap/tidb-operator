# Deploy TiDB Operator and TiDB cluster on AWS EKS

This document describes how to deploy TiDB Operator and a TiDB cluster on AWS EKS with your laptop (Linux or macOS) for development or testing.

## Prerequisites

Before deploying a TiDB cluster on AWS EKS, make sure the following requirements are satisfied:
* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73, to control AWS resources

  The `awscli` must be [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) before it can interact with AWS. The fastest way is using the `aws configure` command:

  ``` shell
  # Replace AWS Access Key ID and AWS Secret Access Key with your own keys
  $ aws configure
  AWS Access Key ID [None]: AKIAIOSFODNN7EXAMPLE
  AWS Secret Access Key [None]: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  Default region name [None]: us-west-2
  Default output format [None]: json
  ```
  > **Note:** The access key must have at least permissions to: create VPC, create EBS, create EC2 and create role
* [terraform](https://learn.hashicorp.com/terraform/getting-started/install.html)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0 and < 3.0.0
* [jq](https://stedolan.github.io/jq/download/)
* [aws-iam-authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html) installed in `PATH`, to authenticate with AWS

  The easiest way to install `aws-iam-authenticator` is to download the prebuilt binary:

  ``` shell
  # Download binary for Linux
  curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator

  # Or, download binary for macOS
  curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/darwin/amd64/aws-iam-authenticator

  chmod +x ./aws-iam-authenticator
  sudo mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator
  ```

## Deploy

The default setup will create a new VPC and a t2.micro instance as bastion machine, and an EKS cluster with the following ec2 instances as worker nodes:

* 3 m5.large instances for PD
* 3 c5d.4xlarge instances for TiKV
* 2 c5.4xlarge instances for TiDB
* 1 c5.2xlarge instance for monitor

Use the following commands to set up the cluster:

``` shell
# Get the code
$ git clone --depth=1 https://github.com/pingcap/tidb-operator
$ cd tidb-operator/deploy/aws

# Apply the configs, note that you must answer "yes" to `terraform apply` to continue
$ terraform init
$ terraform apply
```

It might take 10 minutes or more to finish the process. After `terraform apply` is executed successfully, some useful information is printed to the console.

A successful deployment will give the output like:

```
Apply complete! Resources: 67 added, 0 changed, 0 destroyed.

Outputs:

bastion_ip = [
    52.14.50.145
]
eks_endpoint = https://E10A1D0368FFD6E1E32E11573E5CE619.sk1.us-east-2.eks.amazonaws.com
eks_version = 1.12
monitor_endpoint = http://abd299cc47af411e98aae02938da0762-1989524000.us-east-2.elb.amazonaws.com:3000
region = us-east-2
tidb_dns = abd2e3f7c7af411e98aae02938da0762-17499b76b312be02.elb.us-east-2.amazonaws.com
tidb_port = 4000
tidb_version = v3.0.0
```

> **Note:** You can use the `terraform output` command to get the output again.

## Access the database

To access the deployed TiDB cluster, use the following commands to first `ssh` into the bastion machine, and then connect it via MySQL client (replace the `<>` parts with values from the output):

``` shell
ssh -i credentials/<cluster_name>.pem ec2-user@<bastion_ip>
mysql -h <tidb_dns> -P <tidb_port> -u root
```

The default value of `cluster_name` is `my-cluster`. If the DNS name is not resolvable, be patient and wait a few minutes.

You can interact with the EKS cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_<cluster_name>`:

``` shell
# By specifying --kubeconfig argument
kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb
helm --kubeconfig credentials/kubeconfig_<cluster_name> ls

# Or setting KUBECONFIG environment variable
export KUBECONFIG=$PWD/credentials/kubeconfig_<cluster_name>
kubectl get po -n tidb
helm ls
```

## Monitor

You can access the `monitor_endpoint` address (printed in outputs) using your web browser to view monitoring metrics.

The initial Grafana login credentials are:

- User: admin
- Password: admin

## Upgrade

To upgrade the TiDB cluster, edit the `variables.tf` file with your preferred text editor and modify the `tidb_version` variable to a higher version, and then run `terraform apply`.

For example, to upgrade the cluster to version 3.0.0-rc.1, modify the `tidb_version` to `v3.0.0`:

```
 variable "tidb_version" {
   description = "tidb cluster version"
   default = "v3.0.0"
 }
```

> **Note**: The upgrading doesn't finish immediately. You can watch the upgrading process by `kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb --watch`.

## Scale

To scale the TiDB cluster, edit the `variables.tf` file with your preferred text editor and modify the `default_cluster_tikv_count` or `default_cluster_tidb_count` variable to your desired count, and then run `terraform apply`.

For example, to scale out the cluster, you can modify the number of TiDB instances from 2 to 3:

```
 variable "default_cluster_tidb_count" {
   default = 4
 }
```

> **Note**: Currently, scaling in is NOT supported since we cannot determine which node to scale. Scaling out needs a few minutes to complete, you can watch the scaling out by `kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb --watch`.

## Customize

You can change default values in `variables.tf` (such as the default cluster name and image versions) as needed.

### Customize AWS related resources

By default, the terraform script will create a new VPC. You can use an existing VPC by setting `create_vpc` to `false` and specify your existing VPC id and subnet ids to `vpc_id`, `private_subnet_ids` and `public_subnet_ids` variables.

> **Note:** Reusing VPC and subnets of an existing EKS cluster is not supported yet due to limitations of AWS and Terraform, so only change this option if you have to use a manually created VPC.

An ec2 instance is also created by default as bastion machine to connect to the created TiDB cluster, because the TiDB service is exposed as an [Internal Elastic Load Balancer](https://aws.amazon.com/blogs/aws/internal-elastic-load-balancers/). The ec2 instance has MySQL and Sysbench pre-installed, so you can SSH into the ec2 instance and connect to TiDB using the ELB endpoint. You can disable the bastion instance creation by setting `create_bastion` to `false` if you already have an ec2 instance in the VPC.

The TiDB version and component count are also configurable in variables.tf, you can customize these variables to suit your need.

Currently, the instance type of TiDB cluster component is not configurable because PD and TiKV relies on [NVMe SSD instance store](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html), different instance types have different disks.

### Customize TiDB parameters

By default, the terraform script will pass `./values/default.yaml` to the tidb-cluster helm chart. You can change the `overrides_values` of the tidb cluster module to specify a customized values file.

## Multiple Cluster Management

An instance of `./tidb-cluster` module corresponds to a TiDB cluster in the EKS cluster. If you want to add a new TiDB cluster, you can edit `./cluster.tf` and add a new instance of `./tidb-cluster` module:

```hcl
module example-cluster {
  source = "./tidb-cluster"
  
  # The target EKS, required
  eks_info = local.default_eks
  # The subnets of node pools of this TiDB cluster, required
  subnets = local.default_subnets
  
  # TiDB cluster name, required
  cluster_name    = "example-cluster"
  # Helm values file, required
  override_values = "values/example-cluster.yaml"
  
  # TiDB cluster version
  cluster_version               = "v3.0.0"
  # SSH key of cluster nodes
  ssh_key_name                  = module.key-pair.key_name
  # PD replica number
  pd_count                      = 3
  # TiKV instance type
  pd_instance_type              = "t2.xlarge"
  # The storage class used by PD
  pd_storage_class              = "ebs-gp2"
  # TiKV replica number
  tikv_count                    = 3
  # TiKV instance type
  tikv_instance_type            = "t2.xlarge"
  # The storage class used by TiKV, if the TiKV instance type do not have local SSD, you should change it to storage class 
  # of cloud disks like 'ebs-gp2'. Note that TiKV without local storage is strongly not recommended in production env.
  tikv_storage_class            = "local-storage"
  # TiDB replica number
  tidb_count                    = 2
  # TiDB instance type
  tidb_instance_type            = "t2.xlarge"
  # Monitor instance type
  monitor_instance_type         = "t2.xlarge"
  # The version of tidb-cluster helm chart
  tidb_cluster_chart_version    = "v1.0.0-beta.3"
}

module other-cluster {
  source   = "./tidb-cluster"
  
  cluster_name    = "other-cluster"
  override_values = "values/other-cluster.yaml"
  #......
}
```

> **Note:**
> 
> The `cluster_name` of each cluster must be unique.

You can refer to [./tidb-cluster/variables.tf](./tidb-cluster/variables.tf) for the complete configuration reference of `./tidb-cluster` module.

It is recommended to provide a dedicated values file for each TiDB cluster in favor of the ease of management. You can copy the `values/default.yaml` to get a reasonable default.

You can get the DNS name of TiDB service and grafana service via kubectl. If you want terraform to print these information like the `default-cluster`, you can add `output` sections in `outputs.tf`:

```hcl
output "example-cluster_tidb-dns" {
  value       = module.example-cluster.tidb_dns
}

output "example-cluster_monitor-dns" {
  value       = module.example-cluster.monitor_dns
}
```

## Destroy

It may take some while to finish destroying the cluster.

``` shell
$ terraform destroy
```

> **Note:**
>
> This will destroy your EKS cluster along with all the TiDB clusters you deployed on it.

> **Note:**
>
> If you specify service type `LoadBalancer` for the services like the default configuration do, you have to delete these services before destroy, otherwise they will block the subnets from being destroyed.

> **Note:**
>
> You have to manually delete the EBS volumes in AWS console after running terraform destroy if you do not need the data on the volumes anymore.
