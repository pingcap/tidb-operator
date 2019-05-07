---
title: Deploy TiDB, a distributed MySQL compatible database, on Kubernetes via AWS EKS
summary: Tutorial for deploying TiDB on Kubernetes via AWS EKS.
category: operations
---

# Deploy TiDB, a distributed MySQL compatible database, on Kubernetes via AWS EKS

## Requirements:
* [awscli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) >= 1.16.73
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0
* [jq](https://stedolan.github.io/jq/download/)
* [aws-iam-authenticator](https://github.com/kubernetes-sigs/aws-iam-authenticator#4-set-up-kubectl-to-use-authentication-tokens-provided-by-aws-iam-authenticator-for-kubernetes)
* [terraform](https://www.terraform.io/downloads.html)

## Introduction

This tutorial is designed to be a self-contained deployment of a Kubernetes cluster on AWS EKS with a running TiDB installattion managed by the TiDB Kubernetes operator.

It takes you through these steps:

- Standing up a Kubernetes cluster with TiDB running inside
- Connecting to TiDB
- Scale out the cluster
- Shutting down down the Kubernetes cluster

> Warning: Following this guide will create objects in your AWS account that will cost you money against your AWS bill.

## More about EKS

AWS EKS provides managed Kubernetes master nodes

- There's no master nodes to manage
- The master nodes are multi-AZ to provide redundancy
- The master nodes will scale automatically when necessary

## Configure AWS user

Before continuing, make sure you have create a new user (other than the
root user of your AWS account) in IAM, giving it enough permissions.
For simplicity you can just assign `AdministratorAccess` to the group this user
belongs to. With more detailed permissions, you will have to be sure you also have
`AmazonEKSClusterPolicy` and `AmazonEKSServicePolicy` for this user.

Then generate a pair of access keys and keep them safe locally. 

## A bit more about Terraform

Information about using Terraform with EKS can be found [here](https://www.terraform.io/docs/providers/aws/guides/eks-getting-started.html).
However if this is the first time using Terraform, please go take a glance
at [their tutorial here](https://www.terraform.io/intro/getting-started/install.html).
Firstly making sure your AWS access key pairs and your local Terraform works with
the most simplest infrastructure provision (e.g [this example](https://www.terraform.io/intro/getting-started/build.html#configuration)) before you
continue.


We will use Terraform templates to deploy EKS. Please install terraform using the steps [described in the terraform manual](https://www.terraform.io/intro/getting-started/install.html). For example, on MacOS or Linux:

```sh
# For mac
wget https://releases.hashicorp.com/terraform/0.11.10/terraform_0.11.10_darwin_amd64.zip

unzip terraform*
sudo mv terraform /usr/local/bin/
```

```sh
# For linux
wget https://releases.hashicorp.com/terraform/0.11.10/terraform_0.11.10_linux_amd64.zip

unzip terraform*
sudo mv terraform /usr/local/bin/
```

At this point, let us make sure that `awscli` and `terraform` are properly configured. Here is a simple example that will provision a `t2.micro` for us

```tf
# example.tf

provider "aws" {
  region     = "us-east-1"
}

resource "aws_instance" "example" {
  ami           = "ami-2757f631"
  instance_type = "t2.micro"
}
```

Then run the following command to be sure the Terraform command is working.

```sh
terraform init
terraform apply
# then verify instance creation on AWS console
terraform destroy
```

Note that Terraform will automatically search for saved API credentials (for example, in ~/.aws/credentials)

In our next step, we will be deploying infrastructure based on the Terraform EKS tutorial.

## Running the Terraform script

The Terraform script is located in `./deploy/aws`. After changing directory, we can run the same commands as with our test example
```sh
terraform init
terraform apply -var-file=aws-tutorial.tfvars
```
Here, we are passing in a file that specifies values for certain variables to the `terraform apply` command.

The provisioning of all the infrastructure will take several minutes. We are creating:

* 1 VPC
* 1 t2.micro as a bastion machine
* 1 internal ELB
* 1 c5d.large for PD pods
* 1 c5d.large for TiKV pods
* 1 c5d.large for TiDB pods
* 1 c5d.large for monitoring related pods

When everything has been successfully created, you will see something like this:

```sh
Apply complete! Resources: 67 added, 0 changed, 0 destroyed.

Outputs:

bastion_ip = [
    3.14.255.194
]
eks_endpoint = https://8B49E8619835B8C79BD383B542161819.sk1.us-east-2.eks.amazonaws.com
eks_version = 1.12
monitor_endpoint = http://a37987df9710211e9b48c0ae40bc8d7b-1847612729.us-east-2.elb.amazonaws.com:3000
region = us-east-2
tidb_dns = internal-a37a17c22710211e9b48c0ae40bc8d7b-1891023212.us-east-2.elb.amazonaws.com
tidb_port = 4000
tidb_version = v2.1.8
```
### Connecting to TiDB

Access to TiDB is gated behind the bastion machine. First ssh into it and then use the mysql client
```sh
ssh -i credentials/k8s-prod-aws_tutorial.pem ec2-user@<bastion_ip>
mysql -h <tidb_dns> -P <tidb_port> -u root
```

### Interacting with the Kubernetes cluster

It is possible to interact with the cluster via `kubectl` and `helm` with the kubeconfig file that is created `credentials/kubeconfig_aws_tutorial`.

```sh 
# By specifying --kubeconfig
kubectl --kubeconfig credentials/kubeconfig_aws_tutorial get po -n tidb
helm --kubeconfig credentials/kubeconfig_aws_tutorial ls

# or setting KUBECONFIG environment variable
export KUBECONFIG=$PWD/credentials/kubeconfig_aws_tutorial
kubectl get po -n tidb
helm ls
```

### Viewing the Grafana dashboards

Now that we know how to interact with the cluster, we can port-forward the Grafana service locally

```bash
kubectl port-forward svc/tidb-cluster-grafana 3000:3000 -n tidb &>/dev/null &
```

We can now point a browser to `localhost:3000` and view the dashboards.


### Scale TiDB cluster

To scale out TiDB cluster, modify `tikv_count` or `tidb_count` in `aws-tutorial.tfvars` to your desired count, and then run `terraform apply -var-file=aws-tutorial.tfvars`.

> *Note*: Currently, scaling in is not supported since we cannot determine which node to scale. Scaling out needs a few minutes to complete, you can watch the scaling out by `watch kubectl --kubeconfig credentials/kubeconfig_aws_tutorial get po -n tidb`

> *Note*: There are taints and tolerations in place such that only a single pod will be scheduled per node. The count is also passed onto helm via terraform. For this reason attempting to scale out pods via helm or `kubectl scale` will not work as expected.
---

## Destroy

At the end of the tutorial, please make sure all the resources created by Kubernetes are removed (LoadBalancers, Security groups), so you get a
big bill from AWS.

Simply run:

```sh
terraform destroy
```

(Do this command at the end to clean up, you don't have to do it now!)

> **NOTE:** You have to manually delete the EBS volumes after running `terraform destroy` if you don't need the data on the volumes any more.
