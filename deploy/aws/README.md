# Deploy TiDB Operator and TiDB cluster on AWS EKS

## Requirements:
* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73, to control AWS resources

  The `awscli` must be [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) before it can interact with AWS. The fastest way to set up is using the `aws configure` command:

  ``` shell
  # Replace AWS Access Key ID and AWS Secret Access Key to your own keys
  $ aws configure
  AWS Access Key ID [None]: AKIAIOSFODNN7EXAMPLE
  AWS Secret Access Key [None]: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  Default region name [None]: us-west-2
  Default output format [None]: json
  ```
  > **Note:** The access key must have at least permissions to: create VPC, create EBS, create EC2 and create role
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0
* [jq](https://stedolan.github.io/jq/download/)
* [aws-iam-authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html) installed in `PATH`, to authenticate with AWS

  The easist way to install `aws-iam-authenticator` is to download the prebuilt binary:

  ``` shell
  curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator
  chmod +x ./aws-iam-authenticator
  sudo mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator
  ```

## Setup

The default setup will create a new VPC and a t2.micro instance as bastion machine. And EKS cluster with the following ec2 instance worker nodes:

* 3 m5d.xlarge instances for PD
* 3 i3.2xlarge instances for TiKV
* 2 c4.4xlarge instances for TiDB
* 1 c5.xlarge instance for monitor

``` shell
# Get the code
$ git clone --depth=1 https://github.com/pingcap/tidb-operator
$ cd tidb-operator/deploy/aws

# Apply the configs, note that you must answer "yes" to `terraform apply` to continue
$ terraform init
$ terraform apply
```

It might take 10 minutes or more for the process to finish. After `terraform apply` is executed successfully, some useful information is printed to the console. You can access the `monitor_endpoint` address (printed in output) using your web browser to view monitoring metrics.

A successful deploy will print output like:

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
tidb_version = v3.0.0-rc.1
```

> **Note:** You can use the `terraform output` command to get the output again.

To access TiDB cluster, use the following command to first ssh into the bastion machine, and then connect it via MySQL client (replace the `<>` parts with values from the output):

``` shell
ssh -i credentials/k8s-prod-<cluster_name>.pem ec2-user@<bastion_ip>
mysql -h <tidb_dns> -P <tidb_port> -u root
```

The default value of `cluster_name` is `my-cluster`. If the DNS name is not resolvable, be patient and wait a few minutes.

You can interact with the EKS cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_<cluster_name>`.

``` shell
# By specifying --kubeconfig argument
kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb
helm --kubeconfig credentials/kubeconfig_<cluster_name> ls

# Or setting KUBECONFIG environment variable
export KUBECONFIG=$PWD/credentials/kubeconfig_<cluster_name>
kubectl get po -n tidb
helm ls
```

# Destroy

It may take some while to finish destroying the cluster.

``` shell
$ terraform destroy
```

> **Note:** You have to manually delete the EBS volumes in AWS console after running `terraform destroy` if you do not need the data on the volumes anymore.

## Upgrade TiDB cluster

To upgrade TiDB cluster, modify `tidb_version` variable to a higher version in variables.tf and run `terraform apply`.

> *Note*: The upgrading doesn't finish immediately. You can watch the upgrading process by `kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb --watch`.

## Scale TiDB cluster

To scale TiDB cluster, modify `tikv_count` or `tidb_count` to your desired count, and then run `terraform apply`.

> *Note*: Currently, scaling in is not supported since we cannot determine which node to scale. Scaling out needs a few minutes to complete, you can watch the scaling out by `kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb --watch`.

## Customize

You can change default values in `variables.tf` (like the cluster name and versions) as needed.

### Customize AWS related resources

By default, the terraform script will create a new VPC. You can use an existing VPC by setting `create_vpc` to `false` and specify your existing VPC id and subnet ids to `vpc_id` and `subnets` variables.

An ec2 instance is also created by default as bastion machine to connect to the created TiDB cluster, because the TiDB service is exposed as an [Internal Elastic Load Balancer](https://aws.amazon.com/blogs/aws/internal-elastic-load-balancers/). The ec2 instance has MySQL and Sysbench pre-installed, so you can SSH into the ec2 instance and connect to TiDB using the ELB endpoint. You can disable the bastion instance creation by setting `create_bastion` to `false` if you already have an ec2 instance in the VPC.

The TiDB version and component count are also configurable in variables.tf, you can customize these variables to suit your need.

Currently, the instance type of TiDB cluster component is not configurable because PD and TiKV relies on [NVMe SSD instance store](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html), different instance types have different disks.

### Customize TiDB parameters

Currently, there are not much parameters exposed to be customizable. If you need to customize these, you should modify the `templates/tidb-cluster-values.yaml.tpl` files before deploying. Or if you modify it and run `terraform apply` again after the cluster is running, it will not take effect unless you manually delete the pod via `kubectl delete po -n tidb --all`. This will be resolved when issue [#255](https://github.com/pingcap/tidb-operator/issues/225) is fixed.

## TODO

- [ ] Use [cluster autoscaler](https://github.com/kubernetes/autoscaler)
- [ ] Allow create a minimal TiDB cluster for testing
- [ ] Make the resource creation synchronously to follow Terraform convention
- [ ] Make more parameters customizable
