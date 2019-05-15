# Deploy TiDB Operator and TiDB cluster on AWS EKS

## Requirements:
* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0
* [jq](https://stedolan.github.io/jq/download/)
* [aws-iam-authenticator](https://github.com/kubernetes-sigs/aws-iam-authenticator) installed in `PATH`

## Configure awscli

https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html

## Setup

The default setup will create a new VPC and a t2.micro instance as bastion machine. And EKS cluster with the following ec2 instance worker nodes:

* 3 m5d.xlarge instances for PD
* 3 i3.2xlarge instances for TiKV
* 2 c4.4xlarge instances for TiDB
* 1 c5.xlarge instance for monitor

You can change default values in `variables.tf` (like the cluster name and versions) as needed. The default value of `cluster_name` is `my-cluster`.

``` shell
$ git clone --depth=1 https://github.com/pingcap/tidb-operator
$ cd tidb-operator/deploy/aws
$ terraform init
$ terraform apply
```

It might take 10 minutes or more for the process to finish. After `terraform apply` is executed successfully, some basic information is printed to the console. You can access the `monitor_endpoint` using your web browser.

> **Note:** You can use the `terraform output` command to get that information again.

To access TiDB cluster, use the following command to first ssh into the bastion machine, and then connect it via MySQL client:

``` shell
ssh -i credentials/k8s-prod-<cluster_name>.pem ec2-user@<bastion_ip>
mysql -h <tidb_dns> -P <tidb_port> -u root
```

If the DNS name is not resolvable, be patient and wait a few minutes.

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

# Destory

It may take some while to finish destroying the cluster.

```shell
$ terraform destroy
```

> **Note:** You have to manually delete the EBS volumes in AWS console after running `terraform destroy` if you do not need the data on the volumes anymore.

## Upgrade TiDB cluster

To upgrade TiDB cluster, modify `tidb_version` variable to a higher version in variables.tf and run `terraform apply`.

> *Note*: The upgrading doesn't finish immediately. You can watch the upgrading process by `watch kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb`

## Scale TiDB cluster

To scale TiDB cluster, modify `tikv_count` or `tidb_count` to your desired count, and then run `terraform apply`.

> *Note*: Currently, scaling in is not supported since we cannot determine which node to scale. Scaling out needs a few minutes to complete, you can watch the scaling out by `watch kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb`

## Customize

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
