# Deploy TiDB Operator and TiDB cluster on GCP GKE

This document describes how to deploy TiDB Operator and a TiDB cluster on GCP GKE with your laptop (Linux or macOS) for development or testing.

## Prerequisites

First of all, make sure the following items are installed:

* [Google Cloud SDK](https://cloud.google.com/sdk/install)
* [terraform](https://www.terraform.io/downloads.html)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0
* [jq](https://stedolan.github.io/jq/download/)

## Configure

Before deploying, you need to configure the following items to guarantee a smooth deployment.

### Configure Cloud SDK

After you have installed Google Cloud SDK, you need to [perform initial setup tasks](https://cloud.google.com/sdk/docs/initializing). 

### Configure Terraform

The terraform script expects three environment variables. You can let Terraform prompt you for them, or `export` them in the `~/.bash_profile` file ahead of time. If you choose to export them, they are:

* `TF_VAR_GCP_CREDENTIALS_PATH`: Path to a valid GCP credentials file. 
    - It is recommended to create a new service account to be used by Terraform. See [this page](https://cloud.google.com/iam/docs/creating-managing-service-accounts) to create a service account and grant `Project Editor` role to it. 
    - See [this page](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) to create service account keys, and choose `JSON` key type during creation. The downloaded `JSON` file that contains the private key is the credentials file you need.
* `TF_VAR_GCP_REGION`: The region to create the resources in, for example: `us-west1`.
* `TF_VAR_GCP_PROJECT`: The name of the GCP project.

> *Note*: The service account must have sufficient permissions to create resources in the project. The `Project Editor` primitive will accomplish this.

To set the three environment variables, you can first run `vi ~/.bash_profile` and insert the following `export` statements in it. Here is an example in `~/.bash_profile`:
 
```bash
# Replace the values with the path to the JSON file you have downloaded, the GCP region and your GCP project name.
export TF_VAR_GCP_CREDENTIALS_PATH="/Path/to/my-project.json"
export TF_VAR_GCP_REGION="us-west1"
export TF_VAR_GCP_PROJECT="my-project"
```

### Configure APIs

If the GCP project is new, make sure the relevant APIs are enabled:

```bash
gcloud services enable cloudresourcemanager.googleapis.com && \
gcloud services enable cloudbilling.googleapis.com && \
gcloud services enable iam.googleapis.com && \
gcloud services enable compute.googleapis.com && \
gcloud services enable container.googleapis.com
```

## Deploy

The default setup will create a new VPC, two subnetworks, and an f1-micro instance as a bastion machine. The GKE cluster is created with the following instance types as worker nodes:

* 3 n1-standard-4 instances for PD
* 3 n1-highmem-8 instances for TiKV
* 3 n1-standard-16 instances for TiDB
* 3 n1-standard-2 instances for monitor

> *NOTE*: The number of nodes created depends on how many availability zones there are in the chosen region. Most have 3 zones, but us-central1 has 4. See [Regions and Zones](https://cloud.google.com/compute/docs/regions-zones/) for more information and see the [Customize](#customize) section on how to customize node pools in a regional cluster.

The default setup, as listed above, will exceed the default CPU quota of a GCP project. To increase your project's quota, please follow the instructions [here](https://cloud.google.com/compute/quotas). The default setup will require at least 91 CPUs, more if you need to scale out.

Now that you have configured everything needed, you can launch the script to deploy the TiDB cluster:

```bash
git clone --depth=1 https://github.com/pingcap/tidb-operator
cd tidb-operator/deploy/gcp
terraform init
terraform apply
```

## Access the database

After `terraform apply` is successful, the TiDB cluster can be accessed by SSHing into the bastion machine and connecting via MySQL:

```bash
gcloud compute ssh bastion --zone <zone>
mysql -h <tidb_ilb_ip> -P 4000 -u root
```

## Interact with the cluster

It is possible to interact with the cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_<cluster_name>`. The default `cluster_name` is `my-cluster`, it can be changed in `variables.tf`:

```bash
# By specifying --kubeconfig argument
kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb
helm --kubeconfig credentials/kubeconfig_<cluster_name> ls

# Or setting KUBECONFIG environment variable
export KUBECONFIG=$PWD/credentials/kubeconfig_<cluster_name>
kubectl get po -n tidb
helm ls
```

## Upgrade

To upgrade TiDB cluster, modify `tidb_version` variable to a higher version in variables.tf and run `terraform apply`.

> *Note*: The upgrading doesn't finish immediately. You can watch the upgrading process by `watch kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb`

## Scale

To scale TiDB cluster, modify `tikv_count`, `tikv_replica_count`, `tidb_count`, and `tidb_replica_count` to your desired count, and then run `terraform apply`.

> *Note*: Currently, scaling in is not supported since we cannot determine which node to remove. Scaling out needs a few minutes to complete, you can watch the scaling out by `watch kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb`

> *Note*: Incrementing the node count will create a node per GCP availability zones.

## Customize

### Customize GCP resources

GCP allows attaching a local SSD to any instance type that is `n1-standard-1` or greater. This allows for good customizability.

### Customize TiDB Parameters

Currently, there are not too many parameters exposed to be customized. However, you can modify `templates/tidb-cluster-values.yaml.tpl` before deploying. If you modify it after the cluster is created and then run `terraform apply`, it will not take effect unless the pod(s) is manually deleted.

### Customize node pools

The cluster is created as a regional, as opposed to a zonal cluster. This means that GKE will replicate node pools to each availability zone. This is desired to maintain high availability, however for the monitoring services, like Grafana, this is potentially unnecessary. It is possible to manually remove nodes if desired via `gcloud`.

> *NOTE*: GKE node pools are managed instance groups, so a node deleted by `gcloud compute instances delete` will be automatically recreated and added back to the cluster.

Suppose we wish to delete a node from the monitor pool, we can do:

```bash
$ gcloud compute instance-groups managed list | grep monitor
```
And the result will be something like this:

```bash
gke-my-cluster-monitor-pool-08578e18-grp  us-west1-b  zone   gke-my-cluster-monitor-pool-08578e18  0     0            gke-my-cluster-monitor-pool-08578e18  no
gke-my-cluster-monitor-pool-7e31100f-grp  us-west1-c  zone   gke-my-cluster-monitor-pool-7e31100f  1     1            gke-my-cluster-monitor-pool-7e31100f  no
gke-my-cluster-monitor-pool-78a961e5-grp  us-west1-a  zone   gke-my-cluster-monitor-pool-78a961e5  1     1            gke-my-cluster-monitor-pool-78a961e5  no
```

The first column is the name of the managed instance group, and the second column is the zone it was created in. We will also need the name of the instance in that group, we can get it as follows:

```bash
$ gcloud compute instance-groups managed list-instances gke-my-cluster-monitor-pool-08578e18-grp --zone us-west1-b
NAME                                       ZONE        STATUS   ACTION  INSTANCE_TEMPLATE                     VERSION_NAME  LAST_ERROR
gke-my-cluster-monitor-pool-08578e18-c7vd  us-west1-b  RUNNING  NONE    gke-my-cluster-monitor-pool-08578e18
```

Now we can delete the instance:

```bash
$ gcloud compute instance-groups managed delete-instances gke-my-cluster-monitor-pool-08578e18-grp --instances=gke-my-cluster-monitor-pool-08578e18-c7vd --zone us-west1-b
```

## Destroy

When you are done, the infrastructure can be torn down by running:

```bash
$ terraform destroy
```
