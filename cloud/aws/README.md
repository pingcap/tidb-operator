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

$ # customize variables.tf according to your needs
$ # especially you need to adjust the vpc_id and subnets id list
$ terraform init
$ terraform apply
```

After `terraform apply` is executed successfully, you can access the `monitor_endpoint` using your web browser and access the TiDB cluster via `tidb_endpoint` using MySQL clinet in an ec2 instance in the VPC. If the endpoint DNS name is not resolvable, be patient and wait a few minutes.

You can interact with the EKS cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_<cluster_name>`.

You have to manually delete the EBS volumes after running `terraform destroy` if you don't need the data on the volumes any more.

## TODO
- [ ] auto-scaling group policy
- [ ] Allow both creating VPC or using existing VPC
- [ ] Delete load balancers automatically when running `terraform destroy`
