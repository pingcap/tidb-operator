# Deploy TiDB Operator and TiDB Cluster on Alibaba Cloud Kubernetes

[中文](README-CN.md)

## Requirements

- [aliyun-cli](https://github.com/aliyun/aliyun-cli) >= 3.0.15 and [configure aliyun-cli](https://www.alibabacloud.com/help/doc-detail/90766.htm?spm=a2c63.l28256.a3.4.7b52a893EFVglq)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.12
- [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.1
- [jq](https://stedolan.github.io/jq/download/) >= 1.6
- [terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) 0.11.*

> You can use the Alibaba [Cloud Shell](https://shell.aliyun.com) service, which has all the tools pre-installed and properly configured.

## Overview 

The default setup will create:
 
- A new VPC 
- An ECS instance as bastion machine
- A managed ACK(Alibaba Cloud Kubernetes) cluster with the following ECS instance worker nodes:
  - An auto-scaling group of 2 * instances(1c1g) as ACK mandatory workers for system service like CoreDNS
  - An auto-scaling group of 3 * `ecs.i2.xlarge` instances for PD
  - An auto-scaling group of 3 * `ecs.i2.2xlarge` instances for TiKV
  - An auto-scaling group of 2 * instances(16c32g) for TiDB
  - An auto-scaling group of 1 * instance(4c8g) for monitoring components

In addition, the monitoring node will mount a 500GB cloud disk as data volume. All the instances except ACK mandatory workers span in multiple available zones to provide cross-AZ high availability.

The auto-scaling group will ensure the desired number of healthy instances, so the cluster can auto recover from node failure or even available zone failure.

## Setup

Configure target region and credential (you can also set these variables in `terraform` command prompt):
```shell
export TF_VAR_ALICLOUD_REGION=<YOUR_REGION>
export TF_VAR_ALICLOUD_ACCESS_KEY=<YOUR_ACCESS_KEY>
export TF_VAR_ALICLOUD_SECRET_KEY=<YOUR_SECRET_KEY>
```

Apply the stack:

```shell
$ git clone https://github.com/pingcap/tidb-operator
$ cd tidb-operator/deploy/alicloud
$ terraform init
$ terraform apply
```

`terraform apply` will take 5 to 10 minutes to create the whole stack, once complete, you can interact with the ACK cluster using `kubectl` and `helm`: 

```shell
$ export KUBECONFIG=$PWD/credentials/kubeconfig_<cluster_name>
$ kubectl version
$ helm ls
```

Then you can connect the TiDB cluster via the bastion instance:

```shell
$ ssh -i credentials/bastion-key.pem root@<bastion_ip>
$ mysql -h <tidb_slb_ip> -P <tidb_port> -u root
```

## Monitoring 

Visit `<monitor_endpoint>` to view the grafana dashboards.

> It is strongly recommended to set `monitor_slb_network_type` to `intranet` for security if you already have a VPN connecting to your VPC or plan to setup one.

## Upgrade TiDB cluster

To upgrade TiDB cluster, modify `tidb_version` variable to a higher version in variables.tf and run `terraform apply`.

## Scale TiDB cluster

To scale TiDB cluster, modify `tikv_count` or `tidb_count` to your desired count, and then run `terraform apply`.

## Destroy

```shell
$ terraform destroy
```

> Note: You have to manually delete the cloud disk used by monitoring node after destroying if you don't need it anymore.

## Customize

By default, the terraform script will create a new VPC. You can use an existing VPC by setting `vpc_id` to use an existing VPC. Note that kubernetes node will only be created in available zones that has vswitch existed when using existing VPC. 

An ecs instance is also created by default as bastion machine to connect to the created TiDB cluster, because the TiDB service is only exposed to intranet. The bastion instance has mysql-cli and sysbench installed that helps you use and test TiDB.

If you don't have to access TiDB from internet, you could disable the creation of bastion instance by setting `create_bastion` to false in `variables.tf`

The worker node instance types are also configurable, there are two ways to configure that:

1. by specifying instance type id
2. by specifying capacity like instance cpu count and memory size

Because the Alibaba Cloud offers different instance types in different region, it is recommended to specify the capacity instead of certain type. You can configure these in the `variables.tf`, note that instance type will override capacity configurations.

There's a exception for PD and TiKV instances, because PD and TiKV required local SSD, so you cannot specify instance type for them. Instead, you can choose the type family among `ecs.i1`,`ecs.i2` and `ecs.i2g`, which has one or more local NVMe SSD, and select a certain type in the type family by specifying `instance_memory_size`.

For more customization options, please refer to `variables.tf`

## Limitations

You cannot change pod cidr, service cidr and worker instance types once the cluster created.

