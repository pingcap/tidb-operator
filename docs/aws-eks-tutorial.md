---
title: Deploy TiDB, a distributed MySQL compatible database, on Kubernetes via AWS EKS
summary: Tutorial for deploying TiDB on Kubernetes via AWS EKS.
category: operations
---

# Deploy TiDB, a distributed MySQL compatible database, on Kubernetes via AWS EKS

## Introduction

This tutorial is designed to be run locally with tools like [AWS Command Line Interface](https://aws.amazon.com/cli/) and [Terraform](https://www.terraform.io/). The Terraform code will ultilize a relatively new AWS service called [Amazon Elastic Container Service for Kubernetes (Amazon EKS)](https://aws.amazon.com/eks).

This guide is for running Tidb cluster in a testing Kubernetes environment. The following steps will use certain unsecure configurations. Do not just follow this guide only to build your production db environment. 

It takes you through these steps:

- Launching a new 3-node Kubernetes cluster via Terraform code provisioning with AWS EKS (optional)
- Installing the Helm package manager for Kubernetes
- Deploying the TiDB Operator
- Deploying your first TiDB cluster
- Connecting to TiDB
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

Then generate a pair of access keys and keep them safe locally. Now we can continue about using Terraform to provision a Kubernetes on AWS.

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

### Setting up your terraform config

1. Before continuing, make sure you have create a new user (other than the
  root user of your AWS account) in IAM, giving it enough permissions.
  For simplicity you can just assign `AdministratorAccess` to the group this user
  belongs to. With more detailed permissions, you will have to be sure you also have
  `AmazonEKSClusterPolicy` and `AmazonEKSServicePolicy` for this user.
1. Use editor to open file `~/.aws/credentials` and edit it with your AWS keys associated with the user mentioned above such as:
    ```ini
    [default]
    aws_access_key_id = XXX
    aws_secret_access_key = XXX
    ```

### (Optional) Verify Terraform setup

You can confirm that Terraform is installed correctly by deploying the most simple infrastructure provision. From the Terraform manual:

Prepare an example file

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

## About starting AWS EKS with Terraform

Steps to provision AWS EKS:

1. Provision an EKS cluster with IAM roles, Security Groups and VPC
2. Deploy worker nodes with Launch Configuration, Autoscaling Group, IAM Roles and Security Groups
3. Connect to EKS

More detailed steps can follow [here](https://github.com/wardviaene/terraform-course/tree/master/eks-demo) and [here](https://github.com/terraform-providers/terraform-provider-aws/tree/master/examples/eks-getting-started), more a perhaps more detailed version [here](https://github.com/liufuyang/terraform-course/tree/master/eks-demo) (which is based on the first one).

For now we will follow the configs from the [last link](https://github.com/liufuyang/terraform-course/tree/master/eks-demo).
As it should be more easier and has more related info.

---

## Launch a 3-node Kubernetes cluster

### Clone a Terraform code repo for AWS EKS

See https://www.terraform.io/docs/providers/aws/guides/eks-getting-started.html for full guide.

This step requires cloning a git repository which contains Terraform templates.

Firstly clone [this repo](https://github.com/liufuyang/terraform-course/tree/master/eks-demo) and go to folder `eks-demo`:

```sh
git clone https://github.com/liufuyang/terraform-course.git
cd terraform-course/eks-demo
```

> Note: the following guide assumes you are at the terraform file folder `eks-demo` from the repo mentioned above

At the end of this guide you will be asked to come back to this `tidb-operator` folder
to continue the Tidb demo.

### Download kubectl

Download kubectl, the command line tool for Kubernetes

```sh
# For macOS
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/darwin/amd64/kubectl

chmod +x kubectl
ln -s $(pwd)/kubectl /usr/local/bin/kubectl
```

```sh
# For linux
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl

chmod +x kubectl
ln -s $(pwd)/kubectl /usr/local/bin/kubectl
```

### Download the aws-iam-authenticator

A tool to use AWS IAM credentials to authenticate to a Kubernetes cluster.

```sh
# For macOS
wget -O heptio-authenticator-aws https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v0.3.0/heptio-authenticator-aws_0.3.0_darwin_amd64

chmod +x heptio-authenticator-aws
ln -s $(pwd)/heptio-authenticator-aws /usr/local/bin/heptio-authenticator-aws
```

```sh
# For linux
wget -O heptio-authenticator-aws https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v0.3.0/heptio-authenticator-aws_0.3.0_linux_amd64

chmod +x heptio-authenticator-aws
ln -s $(pwd)/heptio-authenticator-aws /usr/local/bin/heptio-authenticator-aws
```

### Modify providers.tf

Choose your region. EKS is not available in every region, use the Region Table to check whether your region is supported: https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/

Make changes in providers.tf accordingly (region, optionally profile)

### Terraform apply

Start AWS EKS service and create a fully operational Kubernetes environment on your
AWS cloud.

```sh
terraform init
terraform apply  # This step may take more than 10 minutes to finish. Just wait patiently.
```

### Configure kubectl

Export the configuration of the EKS cluster to `.config`. This will allow the `kubectl` command line client to access our newly deployed cluster.

```sh
mkdir .config

terraform output kubeconfig > .config/ekskubeconfig 
# Save output in ~/.kube/config and then use the following env prarm

export  KUBECONFIG=$KUBECONFIG:$(pwd)/.config/ekskubeconfig
# Remember to run the above command again with the right folders when you open a new shell window later for TiDB related works.
```

### Configure config-map-auth-aws

Setup a ConfigMap on Kubernetes so the node instances can have proper IAM roles.

```sh
terraform output config-map-aws-auth > .config/config-map-aws-auth.yaml
# save output in config-map-aws-auth.yaml

kubectl apply -f .config/config-map-aws-auth.yaml
```

### See service and nodes coming up

Your `kubectl` command line client should now be installed and configured. Check the installed services and kubernetes nodes:

```sh
kubectl get svc

kubectl get nodes
```

More info about getting started with Amazon EKS[here](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)

### Deploy Kubernetes Dashboard

#### Step 1 (Optional): Deploy the Dashboard

To allow an easy way of monitoring what's happening on your Kubernetes
cluster, it is very helpful to have a `kubernetes-dashboard` installed.
And you can then easily monitor every pod or service of your Kubernetes cluster in browser.

```sh
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/master/src/deploy/recommended/kubernetes-dashboard.yaml

kubectl apply -f https://raw.githubusercontent.com/kubernetes/heapster/master/deploy/kube-config/influxdb/heapster.yaml

kubectl apply -f https://raw.githubusercontent.com/kubernetes/heapster/master/deploy/kube-config/influxdb/influxdb.yaml
```

#### Step 2: Create an eks-admin Service Account and Cluster Role Binding

```sh
kubectl apply -f eks-admin-service-account.yaml
kubectl apply -f eks-admin-cluster-role-binding.yaml
```

#### Step 3: Connect to the Dashboard

Now you can monitoring your Kubernetes cluster in browser:

```sh
# output a token for logging in kubernetes-dashboard
kubectl -n kube-system describe secret $(kubectl -n kube-system get secret | grep eks-admin | awk '{print $1}')

kubectl proxy
```

Then open http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/

And use the token output from previous command to login.

More info see [here](https://docs.aws.amazon.com/eks/latest/userguide/dashboard-tutorial.html).

#### Step 4: Create an AWS storage class for your Amazon EKS cluster

Amazon EKS clusters are not created with any storage classes. You must define storage classes for your cluster to use and you should define a default storage class for your persistent volume claims. For the demo purpose,
we simply use the default `gp2` storage class.

`gp2` is a general purpose SSD volume that balances price and performance for a wide variety of workloads.

```sh
kubectl create -f gp2-storage-class.yaml
kubectl get storageclass
```

More info to see:
[here](https://docs.aws.amazon.com/eks/latest/userguide/storage-classes.html) ,
[here](https://kubernetes.io/docs/concepts/storage/storage-classes/#aws)
and 
[here](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSVolumeTypes.html)

---

Congratulations, now you have a Kubernetes cloud running on AWS.

Then from this stage, you should be able to follow the [Tidb-Operator guide](https://github.com/pingcap/tidb-operator/blob/master/docs/google-kubernetes-tutorial.md) from section `Install Helm`, `Deploy TiDB Operator` and so on.

Or just continue with our guide here, simply do:

```sh
curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get > get_helm.sh
chmod 700 get_helm.sh && ./get_helm.sh
```

> Now we will be working under folder `tidb-operator` folder of `pingcap/tidb-operator` repo.

```sh
# clone repo, if you haven't done it
git clone https://github.com/pingcap/tidb-operator.git
cd tidb-operator
```

Please note that if you have opened a new shell or terminal since you have
worked previously worked in the directory `eks-demo`, make sure you have the same `KUBECONFIG` setup again in the current shell,
otherwise `kubectl` command will not be able to
connect onto your newly created cluster.

### Deploy tidb-operator

TiDB Operator manages TiDB clusters on Kubernetes and automates tasks related to operating a TiDB cluster. It makes TiDB a truly cloud-native database.

If you already have helm installed, you can continue here to deploy tidb-operator:

```sh
# within directory tidb-operator of repo https://github.com/pingcap/tidb-operator

kubectl create serviceaccount tiller --namespace kube-system
kubectl apply -f ./manifests/tiller-rbac.yaml
helm init --service-account tiller --upgrade


# verify:
kubectl get pods --namespace kube-system | grep tiller

# Deploy TiDB operator:

kubectl apply -f ./manifests/crd.yaml
kubectl apply -f ./manifests/gp2-storage.yaml
helm install ./charts/tidb-operator -n tidb-admin --namespace=tidb-admin

# verify:
kubectl get pods --namespace tidb-admin -o wide
```

### Adjust max-open-files

See issue:
https://github.com/pingcap/tidb-operator/issues/91

Basically in un-comment all the `max-open-files` settings
in file `charts/tidb-cluster/templates/tikv-configmap.yaml`
and set everyone as:

```yaml
max-open-files = 1024
```

### Deploy your first TiDB cluster

```sh
# Deploy your first TiDB cluster

helm install ./charts/tidb-cluster -n tidb --namespace=tidb --set pd.storageClassName=gp2,tikv.storageClassName=gp2

# Or if something goes wrong later and you want to update the deployment, use command:
# helm upgrade tidb ./charts/tidb-cluster --namespace=tidb --set pd.storageClassName=gp2,tikv.storageClassName=gp2

# verify:
kubectl get pods --namespace tidb -o wide

```

Then keep watching output of:

```sh
watch "kubectl get svc -n tidb"
```

When you see `demo-tidb` appear, you can `Control + C` to stop watching. Then the service is ready to connect to!

```sh
# make sure you have jq installed
# on mac you can do: brew install jq

kubectl run -n tidb mysql-client --rm -i --tty --image mysql -- mysql -P 4000 -u root -h $(kubectl get svc demo-tidb -n tidb --output json | jq -r '.spec.clusterIP')
```

Or just:

```sh
kubectl -n tidb port-forward demo-tidb-0 4000:4000 &>/tmp/port-forward.log &
```

Then open a new terminal:

```sh
mysql -h 127.0.0.1 -u root -P 4000 --default-character-set=utf8

# Then try some sql command:
select tidb_version();
```

It works! :tada:

### Monitoring with Grafana

TiDB cluster by default ships with a Grafana monitoring page which
is very useful for DevOps. You can view it as well now.
Simply do a kubectl port-forward on the demo-monitor pod.

```sh
# Note: you will check your cluster and use the actual demo-monitor pod's name instead of the example blow:
kubectl port-forward demo-monitor-5bc85fdb7f-qwl5h 3000 &
```

Then visit [localhost:3000](localhost:3000) and login with default
username and password as `admin`, `admin`.

(Optional) Or use a LoadBalancer and visit the site with public IP.
However this method is not recommended for demo here as it will create
extra resources in our AWS VPC, making the later `terraform destroy`
command cannot clean up all the resources.

If you know what you are doing, then continue. Otherwise use the
`kubectl port-forward` command mentioned above.

```sh
# setting monitor.server.type = LoadBalancer in charts/tidb-cluster/values.yaml
# then do helm update again to update the tidb release
# then check loadbalancer external ip via command
kubectl get svc -n tidb
```

### Perhaps load some data to test your TiDB

A good way of testing database is to use the `TPC-H` Benchmark.
One can easily generate some data and load into your TiDB via
[PingCap's `tidb-bench` repo](https://github.com/pingcap/tidb-bench/tree/master/tpch)

Please refer to the [README](https://github.com/pingcap/tidb-bench/tree/master/tpch) of the `tidb-bench`
to see how to easily generate some test data and load into TiDB.
And test TiDB with SQL queries in the [`queries` folder](https://github.com/pingcap/tidb-bench/tree/master/tpch/queries)

For example, with our 3 node EKS Kubernetes cluster created above,
with TPC-H data generated with scale factor 1 (local data file around 1GB),
the TiDB (V2.0) should be able to finish the first TPC-H query with 15 seconds.

### Scale out the TiDB cluster

With a single command we can easily scale out the TiDB cluster. To scale out TiKV:

```sh
helm upgrade tidb charts/tidb-cluster --set pd.storageClassName=gp2,tikv.storageClassName=gp2,tikv.replicas=5,tidb.replicas=3
```

Now the number of TiKV pods is increased from the default 3 to 5. You can check it with:

```sh
kubectl get po -n tidb
```

We have noticed with a 5 node EKS Kubernetes cluster, 1G TPC-H data,
the first TPC-H query can be finished within 10 seconds.

---

## Destroy

At the end of the demo, please make sure all the resources created by Kubernetes are removed (LoadBalancers, Security groups), so you get a 
big bill from AWS.

Simply run:

```sh
terraform destroy
```

(Do this command at the end to clean up, you don't have to don't do it now!)