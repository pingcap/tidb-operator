---
title: Deploy TiDB on Google Cloud
summary: Learn how to quickly deploy a TiDB cluster on Google Cloud using Kubernetes.
category: how-to
---

# Deploy TiDB on Google Cloud

This tutorial is designed to be directly [run in Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/docs&tutorial=deploy-tidb-from-kubernetes-gke.md).

<a href="https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/docs&tutorial=deploy-tidb-from-kubernetes-gke.md"><img src="https://gstatic.com/cloudssh/images/open-btn.png"/></a>

It takes you through the following steps:

- Launch a new 3-node Kubernetes cluster (optional)
- Deploy TiDB Operator and your first TiDB cluster
- Connect to the TiDB cluster
- Scale out the TiDB cluster
- Access the Grafana dashboard
- Destroy the TiDB cluster
- Shut down the Kubernetes cluster (optional)

> **Warning:**
>
> This is for testing only. DO NOT USE in production!

## Select a project

This tutorial launches a 3-node Kubernetes cluster of `n1-standard-1` machines. Pricing information can be [found here](https://cloud.google.com/compute/pricing).

Please select a project before proceeding:

<walkthrough-project-billing-setup key="project-id">
</walkthrough-project-billing-setup>

## Enable API access

This tutorial requires use of the Compute and Container APIs. Please enable them before proceeding:

<walkthrough-enable-apis apis="container.googleapis.com,compute.googleapis.com">
</walkthrough-enable-apis>

## Configure gcloud defaults

This step defaults gcloud to your preferred project and [zone](https://cloud.google.com/compute/docs/regions-zones/), which simplifies the commands used for the rest of this tutorial:

{{< copyable "shell-regular" >}}

```shell
gcloud config set project {{project-id}}
```

{{< copyable "shell-regular" >}}

```shell
gcloud config set compute/zone us-west1-a
```

## Launch a 3-node Kubernetes cluster

It's now time to launch a 3-node kubernetes cluster! The following command launches a 3-node cluster of `n1-standard-1` machines.

It takes a few minutes to complete:

{{< copyable "shell-regular" >}}

```shell
gcloud container clusters create tidb
```

Once the cluster has launched, set it to be the default:

{{< copyable "shell-regular" >}}

```shell
gcloud config set container/cluster tidb
```

The last step is to verify that `kubectl` can connect to the cluster, and all three machines are running:

{{< copyable "shell-regular" >}}

```shell
kubectl get nodes
```

If you see `Ready` for all nodes, congratulations! You've setup your first Kubernetes cluster.

## Deploy TiDB Operator and TiDB cluster

1. Install Helm and add the Helm chart repository maintained by PingCAP. For details, refer to [Use Helm](tidb-toolkit.md#use-helm)

2. Deploy TiDB Operator. For details, refer to [Install TiDB Operator](deploy-tidb-operator.md#install-tidb-operator).

3. Create the `pd-ssd` StorageClass:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/gke/persistent-disk.yaml
    ```

4. Deploy the TiDB cluster, as in [Deploy TiDB on General Kubernetes](deploy-on-general-kubernetes.md#deploy-tidb-cluster). Set the `storageClassName` of all components to `pd-ssd`.

## Connect to the TiDB cluster

There can be a small delay between the pod being up and running, and the service being available. You can watch list services available with:

{{< copyable "shell-regular" >}}

```shell
watch "kubectl get svc -n tidb"
```

When you see `demo-tidb` appear, you can <kbd>Ctrl</kbd>+<kbd>C</kbd>. The service is ready to connect to!

To connect to TiDB within the Kubernetes cluster, you can establish a tunnel between the TiDB service and your Cloud Shell. This is recommended only for debugging purposes, because the tunnel will not automatically be transferred if your Cloud Shell restarts. To establish a tunnel:

{{< copyable "shell-regular" >}}

```shell
kubectl -n tidb port-forward svc/demo-tidb 4000:4000 &>/tmp/port-forward.log &
```

From your Cloud Shell:

{{< copyable "shell-regular" >}}

```shell
sudo apt-get install -y mysql-client && \
mysql -h 127.0.0.1 -u root -P 4000
```

Try out a MySQL command inside your MySQL terminal:

{{< copyable "sql" >}}

```sql
select tidb_version();
```

If you did not specify a password in the process of installation, set one now:

{{< copyable "sql" >}}

```sql
SET PASSWORD FOR 'root'@'%' = '<change-to-your-password>';
```

> **Note:**
>
> This command contains some special characters which cannot be auto-populated in the google cloud shell tutorial, so you might need to copy and paste it into your console manually.

Congratulations, you are now up and running with a distributed TiDB database compatible with MySQL!

## Scale out the TiDB cluster

To scale out the TiDB cluster, refer to [Scale TiDB in Kubernetes](scale-a-tidb-cluster.md).

## Accessing the Grafana dashboard

To access the Grafana dashboards, you can create a tunnel between the Grafana service and your shell.
To do so, use the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl -n tidb port-forward svc/demo-grafana 3000:3000 &>/dev/null &
```

In Cloud Shell, click on the Web Preview button and enter 3000 for the port. This opens a new browser tab pointing to the Grafana dashboards. Alternatively, use the following URL <https://ssh.cloud.google.com/devshell/proxy?port=3000> in a new tab or window.

If not using Cloud Shell, point a browser to `localhost:3000`.

## Destroy the TiDB cluster

When the TiDB cluster is no longer needed, you can delete it as in [Destroy TiDB Clusters in Kubernetes](destroy-a-tidb-cluster.md).

The above commands only delete the running pods, the data is persistent. If you do not need the data anymore, you should run the following commands to clean the data and the dynamically created persistent disks:

{{< copyable "shell-regular" >}}

```shell
kubectl delete pvc -n tidb -l app.kubernetes.io/instance=demo,app.kubernetes.io/managed-by=tidb-operator && \
kubectl get pv -l app.kubernetes.io/namespace=tidb,app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=demo -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

## Shut down the Kubernetes cluster

Once you have finished experimenting, you can delete the Kubernetes cluster with:

{{< copyable "shell-regular" >}}

```shell
gcloud container clusters delete tidb
```
