---
title: Deploy TiDB on GCP GKE
summary: Learn how to deploy a TiDB cluster on GCP GKE.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-on-gcp-gke/']
---

# Deploy TiDB on GCP GKE

<!-- markdownlint-disable MD029 -->
<!-- markdownlint-disable MD037 -->

This document describes how to deploy a GCP Google Kubernetes Engine (GKE) cluster and deploy a TiDB cluster on GCP GKE.

To deploy TiDB Operator and the TiDB cluster in a self-managed Kubernetes environment, refer to [Deploy TiDB Operator](deploy-tidb-operator.md) and [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md).

## Prerequisites

Before deploying a TiDB cluster on GCP GKE, make sure the following requirements are satisfied:

* Install [Helm 3](https://helm.sh/docs/intro/install/): used for deploying TiDB Operator.
* Install [gcloud](https://cloud.google.com/sdk/gcloud): a command-line tool used for creating and managing GCP services.
* Complete the operations in the **Before you begin** section of [GKE Quickstart](https://cloud.google.com/kubernetes-engine/docs/quickstart#before-you-begin).

    This guide includes the following contents:

    * Enable Kubernetes APIs
    * Configure enough quota

## Configure the GCP service

Configure your GCP project and default region:

{{< copyable "shell-regular" >}}

```bash
gcloud config set core/project <gcp-project>
gcloud config set compute/region <gcp-region>
```

## Create a GKE cluster and node pool

1. Create a GKE cluster and a default node pool:

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container clusters create tidb --region us-east1 --machine-type n1-standard-4 --num-nodes=1
    ```

    * The command above creates a regional cluster.
    * The `--num-nodes=1` option indicates that one node is created in each zone. So if there are three zones in the region, there are three nodes in total, which ensures high availability.
    * It is recommended to use regional clusters in production environments. For other types of clusters, refer to [Types of GKE clusters](https://cloud.google.com/kubernetes-engine/docs/concepts/types-of-clusters).
    * The command above creates a cluster in the default network. If you want to specify a network, use the `--network/subnet` option. For more information, refer to [Creating a regional cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-regional-cluster).

2. Create separate node pools for PD, TiKV, and TiDB:

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container node-pools create pd --cluster tidb --machine-type n1-standard-4 --num-nodes=1 \
        --node-labels=dedicated=pd --node-taints=dedicated=pd:NoSchedule
    gcloud container node-pools create tikv --cluster tidb --machine-type n1-highmem-8 --num-nodes=1 \
        --node-labels=dedicated=tikv --node-taints=dedicated=tikv:NoSchedule
    gcloud container node-pools create tidb --cluster tidb --machine-type n1-standard-8 --num-nodes=1 \
        --node-labels=dedicated=tidb --node-taints=dedicated=tidb:NoSchedule
    ```

    The process might take a few minutes.

## Deploy TiDB Operator

To deploy TiDB Operator on GKE, refer to [deploy TiDB Operator](get-started.md#deploy-tidb-operator).

## Deploy a TiDB cluster and the monitoring component

This section describes how to deploy a TiDB cluster and its monitoring component on GCP GKE.

### Create namespace

To create a namespace to deploy the TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl create namespace tidb-cluster
```

> **Note:**
>
> A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. This document takes `tidb-cluster` as an example. If you want to use other namespace, modify the corresponding arguments of `-n` or `--namespace`.

### Deploy

First, download the sample `TidbCluster` and `TidbMonitor` configuration files:

{{< copyable "shell-regular" >}}

```shell
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/gcp/tidb-cluster.yaml && \
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/gcp/tidb-monitor.yaml
```

Refer to [configure the TiDB cluster](configure-a-tidb-cluster.md) to further customize and configure the CR before applying.

To deploy the `TidbCluster` and `TidbMonitor` CR in the GKE cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl create -f tidb-cluster.yaml -n tidb-cluster && \
kubectl create -f tidb-monitor.yaml -n tidb-cluster
```

After the yaml file above is applied to the Kubernetes cluster, TiDB Operator creates the desired TiDB cluster and its monitoring component according to the yaml file.

> **Note:**
>
> If you need to deploy a TiDB cluster on ARM64 machines, refer to [Deploy a TiDB Cluster on ARM64 Machines](deploy-cluster-on-arm64.md).

### View the cluster status

To view the status of the starting TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get pods -n tidb-cluster
```

When all the Pods are in the `Running` or `Ready` state, the TiDB cluster is successfully started. For example:

```
NAME                              READY   STATUS    RESTARTS   AGE
tidb-discovery-5cb8474d89-n8cxk   1/1     Running   0          47h
tidb-monitor-6fbcc68669-dsjlc     3/3     Running   0          47h
tidb-pd-0                         1/1     Running   0          47h
tidb-pd-1                         1/1     Running   0          46h
tidb-pd-2                         1/1     Running   0          46h
tidb-tidb-0                       2/2     Running   0          47h
tidb-tidb-1                       2/2     Running   0          46h
tidb-tikv-0                       1/1     Running   0          47h
tidb-tikv-1                       1/1     Running   0          47h
tidb-tikv-2                       1/1     Running   0          47h
```

## Access the TiDB database

After you deploy a TiDB cluster, you can access the TiDB database via MySQL client.

### Prepare a bastion host

The LoadBalancer created for your TiDB cluster is an intranet LoadBalancer. You can create a [bastion host](https://cloud.google.com/solutions/connecting-securely#bastion) in the cluster VPC to access the database.

{{< copyable "shell-regular" >}}

```shell
gcloud compute instances create bastion \
    --machine-type=n1-standard-4 \
    --image-project=centos-cloud \
    --image-family=centos-7 \
    --zone=${your-region}-a
```

> **Note:**
>
> `${your-region}-a` is the `a` zone in the region of the cluster, such as `us-central1-a`. You can also create the bastion host in other zones in the same region.

### Install the MySQL client and connect

After the bastion host is created, you can connect to the bastion host via SSH and access the TiDB cluster via the MySQL client.

1. Connect to the bastion host via SSH:

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud compute ssh tidb@bastion
    ```

2. Install the MySQL client:

    {{< copyable "shell-regular" >}}

    ```shell
    sudo yum install mysql -y
    ```

3. Connect the client to the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    mysql -h ${tidb-nlb-dnsname} -P 4000 -u root
    ```

    `${tidb-nlb-dnsname}` is the LoadBalancer IP of the TiDB service. You can view the IP in the `EXTERNAL-IP` field of the `kubectl get svc basic-tidb -n tidb-cluster` execution result.

    For example:

    ```shell
    $ mysql -h 10.128.15.243 -P 4000 -u root
    Welcome to the MariaDB monitor.  Commands end with ; or \g.
    Your MySQL connection id is 7823
    Server version: 5.7.25-TiDB-v4.0.4 TiDB Server (Apache License 2.0) Community Edition, MySQL 5.7 compatible

    Copyright (c) 2000, 2018, Oracle, MariaDB Corporation Ab and others.

    Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

    MySQL [(none)]> show status;
    +--------------------+--------------------------------------+
    | Variable_name      | Value                                |
    +--------------------+--------------------------------------+
    | Ssl_cipher         |                                      |
    | Ssl_cipher_list    |                                      |
    | Ssl_verify_mode    | 0                                    |
    | Ssl_version        |                                      |
    | ddl_schema_version | 22                                   |
    | server_id          | 717420dc-0eeb-4d4a-951d-0d393aff295a |
    +--------------------+--------------------------------------+
    6 rows in set (0.01 sec)
    ```

> **Note:**
>
> * [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you use MySQL client from MySQL 8.0 to access the TiDB service (TiDB version < v4.0.7), and if the user account has a password, you need to explicitly specify the `--default-auth=mysql_native_password` parameter.
> * By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).

### Access the Grafana monitoring dashboard

Obtain the LoadBalancer IP of Grafana:

{{< copyable "shell-regular" >}}

```shell
kubectl -n tidb-cluster get svc basic-grafana
```

For example:

```
$ kubectl -n tidb-cluster get svc basic-grafana
NAME                     TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)               AGE
basic-grafana            LoadBalancer   10.15.255.169   34.123.168.114   3000:30657/TCP        35m
```

In the output above, the `EXTERNAL-IP` column is the LoadBalancer IP.

You can access the `${grafana-lb}:3000` address using your web browser to view monitoring metrics. Replace `${grafana-lb}` with the LoadBalancer IP.

> **Note:**
>
> The default Grafana username and password are both `admin`.

## Upgrade

To upgrade the TiDB cluster, edit the `spec.version` by executing `kubectl edit tc basic -n tidb-cluster`.

The upgrade process does not finish immediately. You can watch the upgrade progress by executing `kubectl get pods -n tidb-cluster --watch`.

## Scale out

Before scaling out the cluster, you need to scale out the corresponding node pool so that the new instances have enough resources for operation.

This section describes how to scale out the EKS node group and TiDB components.

### Scale out GKE node group

The following example shows how to scale out the `tikv` node pool of the `tidb` cluster to 6 nodes:

{{< copyable "shell-regular" >}}

```shell
gcloud container clusters resize tidb --node-pool tikv --num-nodes 2
```

> **Note:**
>
> In the regional cluster, the nodes are created in 3 zones. Therefore, after scaling out, the number of nodes is `2 * 3 = 6`.

### Scale out TiDB components

After that, execute `kubectl edit tc basic -n tidb-cluster` and modify each component's `replicas` to the desired number of replicas. The scaling-out process is then completed.

For more information on managing node pools, refer to [GKE Node pools](https://cloud.google.com/kubernetes-engine/docs/concepts/node-pools).

## Deploy TiFlash and TiCDC

[TiFlash](https://docs.pingcap.com/tidb/stable/tiflash-overview) is the columnar storage extension of TiKV.

[TiCDC](https://docs.pingcap.com/tidb/stable/ticdc-overview) is a tool for replicating the incremental data of TiDB by pulling TiKV change logs.

The two components are *not required* in the deployment. This section shows a quick start example.

### Create new node pools

* Create a node pool for TiFlash:

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container node-pools create tiflash --cluster tidb --machine-type n1-highmem-8 --num-nodes=1 \
        --node-labels dedicated=tiflash --node-taints dedicated=tiflash:NoSchedule
    ```

* Create a node pool for TiCDC:

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container node-pools create ticdc --cluster tidb --machine-type n1-standard-4 --num-nodes=1 \
        --node-labels dedicated=ticdc --node-taints dedicated=ticdc:NoSchedule
    ```

### Configure and deploy

* To deploy TiFlash, configure `spec.tiflash` in `tidb-cluster.yaml`. For example:

    ```yaml
    spec:
      ...
      tiflash:
        baseImage: pingcap/tiflash
        replicas: 1
        storageClaims:
        - resources:
            requests:
            storage: 100Gi
        nodeSelector:
          dedicated: tiflash
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: tiflash
    ```

    To configure other parameters, refer to [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

    > **Warning:**
    >
    > TiDB Operator automatically mounts PVs **in the order of the configuration** in the `storageClaims` list. Therefore, if you need to add disks for TiFlash, make sure that you add the disks **only to the end of the original configuration** in the list. In addition, you must **not** alter the order of the original configuration.

* To deploy TiCDC, configure `spec.ticdc` in `tidb-cluster.yaml`. For example:

    ```yaml
    spec:
      ...
      ticdc:
        baseImage: pingcap/ticdc
        replicas: 1
        nodeSelector:
          dedicated: ticdc
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: ticdc
    ```

    Modify `replicas` according to your needs.

Finally, execute `kubectl -n tidb-cluster apply -f tidb-cluster.yaml` to update the TiDB cluster configuration.

For detailed CR configuration, refer to [API references](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md) and [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

## Deploy TiDB Enterprise Edition

To deploy TiDB/PD/TiKV/TiFlash/TiCDC Enterprise Edition, configure `spec.[tidb|pd|tikv|tiflash|ticdc].baseImage` in `tidb-cluster.yaml` as the enterprise image. The enterprise image format is `pingcap/[tidb|pd|tikv|tiflash|ticdc]-enterprise`.

For example:

```yaml
spec:
  ...
  pd:
    baseImage: pingcap/pd-enterprise
  ...
  tikv:
    baseImage: pingcap/tikv-enterprise
```

## Use local storage

Use [Zonal Persistent disks](https://cloud.google.com/compute/docs/disks#pdspecs) as a primary production configuration. To simulate bare metal performance, some GCP instance types provide additional [local store volumes](https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/local-ssd). You can choose such instances for the TiKV node pool to achieve higher IOPS and lower latency.

> **Note:**
>
> You cannot dynamically change the storage class of a running TiDB cluster. You can create a new cluster for testing.
>
> During the GKE upgrade, [data in the local storage will be lost](https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/local-ssd) due to the node reconstruction. When the node reconstruction occurs, you need to migrate data in TiKV. If you do not want to migrate data, it is recommended not to use the local disk in the production environment.

1. Create a node pool with local storage for TiKV:

    {{< copyable "shell-regular" >}}

    ```shell
    gcloud container node-pools create tikv --cluster tidb --machine-type n1-standard-4 --num-nodes=1 --local-ssd-count 1 \
      --node-labels dedicated=tikv --node-taints dedicated=tikv:NoSchedule
    ```

    If the TiKV node pool already exists, you can either delete the old pool and then create a new one, or change the pool name to avoid conflict.

2. Deploy the local volume provisioner.

    You need to use the [local-volume-provisioner](https://sigs.k8s.io/sig-storage-local-static-provisioner) to discover and manage the local storage. Executing the following command deploys and creates a `local-storage` storage class:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/gke/local-ssd-provision/local-ssd-provision.yaml
    ```

3. Use the local storage.

    After the steps above, the local volume provisioner can discover all the local NVMe SSD disks in the cluster.

    Modify `tikv.storageClassName` in the `tidb-cluster.yaml` file to `local-storage`.
