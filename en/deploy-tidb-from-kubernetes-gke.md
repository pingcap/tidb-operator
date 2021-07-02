---
title: Deploy TiDB on Google Cloud
summary: Learn how to quickly deploy a TiDB cluster on Google Cloud using Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-tidb-from-kubernetes-gke/']
---

# Deploy TiDB on Google Cloud

This document is designed to be directly [run in Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/pingcap/docs-tidb-operator&cloudshell_tutorial=en/deploy-tidb-from-kubernetes-gke.md).

<a href="https://console.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/pingcap/docs-tidb-operator&cloudshell_tutorial=en/deploy-tidb-from-kubernetes-gke.md"><img src="https://gstatic.com/cloudssh/images/open-btn.png"/></a>

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
> This document is for testing purposes only. **Do not** follow it in production environments. For production environments, see the instructions in [Deploy TiDB on GCP GKE](deploy-on-gcp-gke.md).

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

```shell
gcloud config set project {{project-id}}
```

```shell
gcloud config set compute/zone us-west1-a
```

## Launch a 3-node Kubernetes cluster

It's now time to launch a 3-node kubernetes cluster! The following command launches a 3-node cluster of `n1-standard-1` machines.

It takes a few minutes to complete:

```shell
gcloud container clusters create tidb
```

Once the cluster has launched, set it to be the default:

```shell
gcloud config set container/cluster tidb
```

The last step is to verify that `kubectl` can connect to the cluster, and all three machines are running:

```shell
kubectl get nodes
```

If you see `Ready` for all nodes, congratulations! You've set up your first Kubernetes cluster.

## Install Helm

[Helm](https://helm.sh/) is a package management tool for Kubernetes.

1. Install the Helm client:

    ```shell
    curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
    ```

2. Add the PingCAP repository:

    ```shell
    helm repo add pingcap https://charts.pingcap.org/
    ```

## Deploy TiDB Operator

TiDB Operator uses [Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) to extend Kubernetes. Therefore, to use TiDB Operator, you must first create the `TidbCluster` CRD.

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

After the `TidbCluster` CRD is created, install TiDB Operator in your Kubernetes cluster.

```shell
kubectl create namespace tidb-admin
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.2.0-rc.2
kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
```

## Deploy the TiDB cluster

To deploy the TiDB cluster, perform the following steps:

1. Create `Namespace`:

    ```shell
    kubectl create namespace demo
    ```

2. Deploy the TiDB cluster:

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml -n demo
    ```

3. Deploy the TiDB cluster monitor:

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n demo
    ```

4. View the Pod status:

    ``` shell
    kubectl get po -n demo
    ```

## Connect to the TiDB cluster

There can be a small delay between the pod being up and running, and the service being available. You can view the service status using the following command:

```shell
kubectl get svc -n demo --watch
```

When you see `basic-tidb` appear, the service is ready to access. You can use <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop the process.

To connect to TiDB within the Kubernetes cluster, you can establish a tunnel between the TiDB service and your Cloud Shell. This is recommended only for debugging purposes, because the tunnel will not automatically be transferred if your Cloud Shell restarts. To establish a tunnel:

```shell
kubectl -n demo port-forward svc/basic-tidb 4000:4000 &>/tmp/pf4000.log &
```

From your Cloud Shell:

```shell
sudo apt-get install -y mysql-client && \
mysql -h 127.0.0.1 -u root -P 4000
```

Try out a MySQL command inside your MySQL terminal:

```sql
select tidb_version();
```

If you did not specify a password during installation, set one now:

```sql
SET PASSWORD FOR 'root'@'%' = '<change-to-your-password>';
```

> **Note:**
>
> This command contains some special characters which cannot be auto-populated in the google cloud shell tutorial, so you might need to copy and paste it into your console manually.

Congratulations, you are now up and running with a distributed TiDB database compatible with MySQL!

> **Note:**
>
> By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).

## Scale out the TiDB cluster

To scale out the TiDB cluster, modify `spec.pd.replicas`, `spec.tidb.replicas`, and `spec.tikv.replicas` in the `TidbCluster` object of the cluster to your desired value using kubectl.

``` shell
kubectl -n demo edit tc basic
```

## Access the Grafana dashboard

To access the Grafana dashboards, you can forward a port from the Cloud Shell to the Grafana service in Kubernetes. (Cloud Shell already uses port 3000 so we use port 8080 in this example instead.)

To do so, use the following command:

```shell
kubectl -n demo port-forward svc/basic-grafana 8080:3000 &>/tmp/pf8080.log &
```

Open this URL to view the Grafana dashboard: <https://ssh.cloud.google.com/devshell/proxy?port=8080> . (Alternatively, in Cloud Shell, click the Web Preview button on the upper right corner and change the port to 8080 if necessary.  If not using Cloud Shell, point a browser to `localhost:8080`.

The default username and password are both "admin".

## Destroy the TiDB cluster

To destroy a TiDB cluster in Kubernetes, run the following command:

```shell
kubectl delete tc basic -n demo
```

To destroy the monitoring component, run the following command:

```shell
kubectl delete tidbmonitor basic -n demo
```

The above commands only delete the running pods, the data is persistent. If you do not need the data anymore, you should run the following commands to clean the data and the dynamically created persistent disks:

```shell
kubectl delete pvc -n demo -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator && \
kubectl get pv -l app.kubernetes.io/namespace=demo,app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=basic -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

## Shut down the Kubernetes cluster

Once you have finished experimenting, you can delete the Kubernetes cluster:

```shell
gcloud container clusters delete tidb
```
