# Deploy TiDB Operator and TiDB cluster on AWS EKS

## Requirements:
* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0
* [aws-iam-authenticator](https://github.com/kubernetes-sigs/aws-iam-authenticator#4-set-up-kubectl-to-use-authentication-tokens-provided-by-aws-iam-authenticator-for-kubernetes)

## Configure awscli

https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html

## Setup

``` shell
$ git clone https://github.com/pingcap/tidb-operator
$ cd tidb-operator/cloud/aws
$ terraform init
$ terraform apply
```

After `terraform apply` is executed successfully, you can access the `monitor_endpoint` using your web browser.

To access TiDB cluster, use the following command to first ssh into the bastion machine, and then connect it via MySQL client:

``` shell
ssh -i credentials/k8s-prod-my-cluster.pem ec2-user@<bastion_ip>
mysql -h <tidb_dns> -P <tidb_port> -u root
```

If the DNS name is not resolvable, be patient and wait a few minutes.

You can interact with the EKS cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_<cluster_name>`. The default `cluster_name` is `my-cluster`, you can change it in the variables.tf.

``` shell
# By specifying --kubeconfig argument
kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb
helm --kubeconfig credentials/kubeconfig_<cluster_name> ls

# Or setting KUBECONFIG environment variable
export KUBECONFIG=$PWD/credentials/kubeconfig_<cluster_name>
kubectl get po -n tidb
helm ls
```

> **NOTE:** You have to manually delete the EBS volumes after running `terraform destroy` if you don't need the data on the volumes any more.

## Upgrade TiDB cluster

To upgrade TiDB cluster, modify `tidb_version` variable to a higher version in variables.tf and run `terraform apply`.

## Scale TiDB cluster

To scale TiDB cluster, modify `tikv_count` or `tidb_count` to your desired count, and then run `terraform apply`.

## Customize

By default, the terraform script will create a new VPC. You can use an existing VPC by setting `create_vpc` to `false` and specify your existing VPC id and subnet ids to `vpc_id` and `subnets` variables.

An ec2 instance is also created by default as bastion machine to connect to the created TiDB cluster, because the TiDB service is exposed as an [Internal Elastic Load Balancer](https://aws.amazon.com/blogs/aws/internal-elastic-load-balancers/). The ec2 instance has MySQL and Sysbench pre-installed, so you can SSH into the ec2 instance and connect to TiDB using the ELB endpoint. You can disable the bastion instance creation by setting `create_bastion` to `false` if you already have an ec2 instance in the VPC.

The TiDB version and component count are also configurable in variables.tf, you can customize these variables to suit your need.

Currently, the instance type of TiDB cluster component is not configurable because PD and TiKV relies on [NVMe SSD instance store](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html), different instance types have different disks.

## TODO
- [ ] auto-scaling group policy
- [ ] Allow create a minimal TiDB cluster
