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

## Install Helm

[Helm](https://helm.sh/) is a package management tool for Kubernetes. Make sure your Helm version >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797). The installation steps are as follows:

1. Refer to [Helm official documentation](https://v2.helm.sh/docs/using_helm/#installing-helm) to install the Helm client.

2. Install the Helm server.

    Apply the `RBAC` rule required by the `tiller` component in the cluster and install `tiller`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/tiller-rbac.yaml && \
    helm init --service-account=tiller --upgrade
    ```

    To confirm that the `tiller` Pod is in the `running` state, run the following command:

    ```shell
    kubectl get po -n kube-system -l name=tiller
    ```

3. Add the repository:

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo add pingcap https://charts.pingcap.org/
    ```

    Use `helm search` to search the chart provided by PingCAP:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l
    ```

## Deploy TiDB Operator

TiDB Operator uses [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to extend Kubernetes. Therefore, to use TiDB Operator, you must first create the `TidbCluster` CRD.

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

After `TidbCluster` CRD is created, install TiDB Operator in your Kubernetes cluster.

1. Get the `values.yaml` file of the `tidb-operator` chart you want to install:

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p /home/tidb/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=v1.1.0-rc.1 > /home/tidb/tidb-operator/values-tidb-operator.yaml
    ```

    Modify the configuration in `values.yaml` according to your needs.

2. Install TiDB Operator:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-operator --name=tidb-operator --namespace=tidb-admin --version=v1.1.0-rc.1 -f /home/tidb/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

3. Create the `pd-ssd` StorageClass:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/gke/persistent-disk.yaml
    ```

## Deploy the TiDB cluster

To deploy the TiDB cluster, perform the following steps:

1. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create namespace demo
    ```

2. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml -n demo
    ```

3. Deploy the TiDB cluster monitor:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n demo
    ```

4. View the Pod status:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n demo
    ```

## Connect to the TiDB cluster

There can be a small delay between the pod being up and running, and the service being available. You can view the service status using the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get svc -n demo --watch
```

When you see `basic-tidb` appear, the service is ready to access. You can use <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop the process.

To connect to TiDB within the Kubernetes cluster, you can establish a tunnel between the TiDB service and your Cloud Shell. This is recommended only for debugging purposes, because the tunnel will not automatically be transferred if your Cloud Shell restarts. To establish a tunnel:

{{< copyable "shell-regular" >}}

```shell
kubectl -n demo port-forward svc/basic-tidb 4000:4000 &>/tmp/port-forward.log &
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

If you did not specify a password during installation, set one now:

{{< copyable "sql" >}}

```sql
SET PASSWORD FOR 'root'@'%' = '<change-to-your-password>';
```

> **Note:**
>
> This command contains some special characters which cannot be auto-populated in the google cloud shell tutorial, so you might need to copy and paste it into your console manually.

Congratulations, you are now up and running with a distributed TiDB database compatible with MySQL!

## Scale out the TiDB cluster

To scale out the TiDB cluster, modify `spec.pd.replicas`, `spec.tidb.replicas`, and `spec.tikv.replicas` in the `TidbCluster` object of the cluster to your desired value using kubectl.

{{< copyable "shell-regular" >}}

``` shell
kubectl -n demo edit tc basic
```

## Access the Grafana dashboard

To access the Grafana dashboards, you can create a tunnel between the Grafana service and your shell.
To do so, use the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl -n demo port-forward svc/basic-grafana 3000:3000 &>/dev/null &
```

In Cloud Shell, click on the Web Preview button and enter 3000 for the port. This opens a new browser tab pointing to the Grafana dashboards. Alternatively, use the following URL <https://ssh.cloud.google.com/devshell/proxy?port=3000> in a new tab or window.

If not using Cloud Shell, point a browser to `localhost:3000`.

## Destroy the TiDB cluster

To destroy a TiDB cluster in Kubernetes, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl delete tc basic -n demo
```

To destroy the monitoring component, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n demo
```

The above commands only delete the running pods, the data is persistent. If you do not need the data anymore, you should run the following commands to clean the data and the dynamically created persistent disks:

{{< copyable "shell-regular" >}}

```shell
kubectl delete pvc -n demo -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator && \
kubectl get pv -l app.kubernetes.io/namespace=demo,app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=basic -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

## Shut down the Kubernetes cluster

Once you have finished experimenting, you can delete the Kubernetes cluster with:

{{< copyable "shell-regular" >}}

```shell
gcloud container clusters delete tidb
```

## More Information

A simple [deployment based on Terraform] is also provided.
