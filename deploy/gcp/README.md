# Deploy TiDB Operator and TiDB cluster on GCP GKE

## Requirements:
* [gcloud](https://cloud.google.com/sdk/install)
* [terraform](https://www.terraform.io/downloads.html)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) >= 1.11
* [helm](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client) >= 2.9.0
* [jq](https://stedolan.github.io/jq/download/)

## Configure gcloud

https://cloud.google.com/sdk/docs/initializing

## Setup

The default setup will create a new VPC, two subnetworks, and an f1-micro instance as a bastion machine. The GKE cluster is created with the following instance types as worker nodes:

* 3 n1-standard-4 instances for PD
* 3 n1-highmem-8 instances for TiKV
* 3 n1-standard-16 instances for TiDB
* 3 n1-standard-2 instances for monitor

> *NOTE*: The number of nodes created depends on how many availability zones there are in the chosen region. Most have 3 zones, but us-central1 has 4. See https://cloud.google.com/compute/docs/regions-zones/ for more information.

The terraform script expects three environment variables. You can let Terraform prompt you for them, or `export` them ahead of time. If you choose to export them, they are:

* `TF_VAR_GCP_CREDENTIALS_PATH`: Path to a valid GCP credentials file
* `TF_VAR_GCP_REGION`: The region to create the resources in, for example: `us-west1`
* `TF_VAR_GCP_PROJECT`: The name of the GCP project

It is generally considered a good idea to create a service account to be used by Terraform. See https://cloud.google.com/iam/docs/creating-managing-service-accounts for more information on how to manage them.

The service account should have sufficient permissions to create resources in the project. The `Project Editor` primitive will accomplish this.

If the GCP project is new, make sure the relevant APIs are enabled:

```bash
gcloud services enable cloudresourcemanager.googleapis.com && \
gcloud services enable cloudbilling.googleapis.com && \
gcloud services enable iam.googleapis.com && \
gcloud services enable compute.googleapis.com && \
gcloud services enable container.googleapis.com
```

Now we can launch the script:

```bash
git clone https://github.com/pingcap/tidb-operator
cd tidb-operator/deploy/gcp
terraform init
terraform apply
```

After `terraform apply` is successful, the TiDB cluster can be accessed by SSHing into the bastion machine and connecting via MySQL:
```bash
gcloud compute ssh bastion --zone <zone>
mysql -h <tidb_ilb_ip> -P 4000 -u root
```

It is possible to interact with the cluster using `kubectl` and `helm` with the kubeconfig file `credentials/kubeconfig_<cluster_name>`. The default `cluster_name` is `my-cluster`, it can be changed in `variables.tf`
```bash
# By specifying --kubeconfig argument
kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb
helm --kubeconfig credentials/kubeconfig_<cluster_name> ls

# Or setting KUBECONFIG environment variable
export KUBECONFIG=$PWD/credentials/kubeconfig_<cluster_name>
kubectl get po -n tidb
helm ls
```

When done, the infrastructure can be torn down by running `terraform destroy`


> *NOTE*: Any provisioned disks will have to be manually deleted after `terraform destroy`, assuming you do not need the data on the volumes anymore.

## Upgrade TiDB cluster

To upgrade TiDB cluster, modify `tidb_version` variable to a higher version in variables.tf and run `terraform apply`.

> *Note*: The upgrading doesn't finish immediately. You can watch the upgrading process by `watch kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb`

## Scale TiDB cluster

To scale TiDB cluster, modify `tikv_count` or `tidb_count` to your desired count, and then run `terraform apply`.

> *Note*: Currently, scaling in is not supported since we cannot determine which node to scale. Scaling out needs a few minutes to complete, you can watch the scaling out by `watch kubectl --kubeconfig credentials/kubeconfig_<cluster_name> get po -n tidb`

> *Note*: Incrementing the node count will create a node per GCP availability zones.

## Customize

### Customize GCP resources

GCP allows attaching a local SSD to any instance type that is `n1-standard-1` or greater. This allows for good customizability.

### Customize TiDB Parameters

Currently, there are not too many parameters exposed to be customized. However, you can modify `templates/tidb-cluster-values.yaml.tpl` before deploying. If you modify it after the cluster is created and then run `terraform apply`, it will not take effect unless the pod(s) is manually deleted.