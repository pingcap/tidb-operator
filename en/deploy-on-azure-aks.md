---
title: Deploy TiDB on Azure AKS
summary: Learn how to deploy a TiDB cluster on Azure Kubernetes Service (AKS).
---

# Deploy TiDB on Azure AKS

This document describes how to deploy a TiDB cluster on Azure Kubernetes Service (AKS).

To deploy TiDB Operator and the TiDB cluster in a self-managed Kubernetes environment, refer to [Deploy TiDB Operator](deploy-tidb-operator.md) and [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md).

## Prerequisites

Before deploying a TiDB cluster on Azure AKS, perform the following operations:

* Install [Helm 3](https://helm.sh/docs/intro/install/) for deploying TiDB Operator.
* [Deploy a Kubernetes (AKS) cluster](https://docs.microsoft.com/en-us/azure/aks/tutorial-kubernetes-deploy-cluster) and install and configure `az cli`.

    > **Note：**
    >
    > To verify whether AZ CLI is configured correctly, run the `az login` command. If login with account credentials succeeds, AZ CLI is configured correctly. Otherwise, you need to re-configure AZ CLI.

* Refer to [use Ultra disks](https://docs.microsoft.com/en-us/azure/aks/use-ultra-disks) to create a new cluster that can use Ultra disks or enable Ultra disks in an exist cluster.
* Acquire [AKS service permissions](https://docs.microsoft.com/en-us/azure/aks/concepts-identity#aks-service-permissions).
* If the Kubernetes version of the cluster is earlier than 1.21, install [aks-preview CLI extension](https://docs.microsoft.com/en-us/azure/aks/custom-node-configuration#install-aks-preview-cli-extension) for using Ultra Disks and register [EnableAzureDiskFileCSIDriver](https://docs.microsoft.com/en-us/azure/aks/csi-storage-drivers#install-csi-storage-drivers-on-a-new-cluster-with-version--121) in [your subscription](https://docs.microsoft.com/en-us/cli/azure/feature?view=azure-cli-latest#az_feature_register-optional-parameters).

    - Install the aks-preview CLI extension:

      {{< copyable "shell-regular" >}}

      ```bash
      az extension add --name aks-preview
      ```

    - Register `EnableAzureDiskFileCSIDriver`:

      {{< copyable "shell-regular" >}}

      ```bash
      az feature register --name EnableAzureDiskFileCSIDriver --namespace Microsoft.ContainerService --subscription ${your-subscription-id}
      ```

## Create an AKS cluster and a node pool

Most of the TiDB cluster components use Azure disk as storage. According to [AKS Best Practices](https://docs.microsoft.com/en-us/azure/aks/operator-best-practices-cluster-isolation), when creating an AKS cluster, it is recommended to ensure that each node pool uses one availability zone (at least 3 in total).

### Create an AKS cluster with CSI enabled

To create an AKS cluster with [CSI enabled](https://docs.microsoft.com/en-us/azure/aks/csi-storage-drivers), run the following command:

> **Note:**
>
> If the Kubernetes version of the cluster is earlier than 1.21, you need to append an `--aks-custom-headers` flag to enable the **EnableAzureDiskFileCSIDriver** feature by running the following command:

{{< copyable "shell-regular" >}}

```bash
# create AKS cluster
az aks create \
    --resource-group ${resourceGroup} \
    --name ${clusterName} \
    --location ${location} \
    --generate-ssh-keys \
    --vm-set-type VirtualMachineScaleSets \
    --load-balancer-sku standard \
    --node-count 3 \
    --zones 1 2 3 \
    --aks-custom-headers EnableAzureDiskFileCSIDriver=true
```

### Create component node pools

After creating an AKS cluster, run the following commands to create component node pools. Each node pool may take two to five minutes to create. It is recommended to enable [Ultra disks](https://docs.microsoft.com/en-us/azure/aks/use-ultra-disks#enable-ultra-disks-on-an-existing-cluster) in the TiKV node pool. For more details about cluster configuration, refer to [`az aks` documentation](https://docs.microsoft.com/en-us/cli/azure/aks?view=azure-cli-latest#az_aks_create) and [`az aks nodepool` documentation](https://docs.microsoft.com/en-us/cli/azure/aks/nodepool?view=azure-cli-latest).

1. To create a TiDB Operator and monitor pool:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name admin \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=admin
    ```

2. Create a PD node pool with `nodeType` being `Standard_F4s_v2` or higher:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name pd \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=pd \
        --node-taints dedicated=pd:NoSchedule
    ```

3. Create a TiDB node pool with `nodeType` being `Standard_F8s_v2` or higher. You can set `--node-count` to `2` because only two TiDB nodes are required by default. You can also scale out this node pool by modifying this parameter at any time if necessary.

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tidb \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 2 \
        --labels dedicated=tidb \
        --node-taints dedicated=tidb:NoSchedule
    ```

4. Create a TiKV node pool with `nodeType` being `Standard_E8s_v4` or higher:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tikv \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

### Deploy component node pools in availability zones

The Azure AKS cluster deploys nodes across multiple zones using "best effort zone balance". If you want to apply "strict zone balance" (not supported in AKS now), you can deploy one node pool in one zone. For example:

1. Create TiKV node pool 1 in zone 1:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tikv1 \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

2. Create TiKV node pool 2 in zone 2:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tikv2 \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 2 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

3. Create TiKV node pool 3 in zone 3:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tikv3 \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 1 \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule \
        --enable-ultra-ssd
    ```

> **Warning:**
>
> About node pool scale-in:
>
> * You can manually scale in or out an AKS cluster to run a different number of nodes. When you scale in, nodes are carefully [cordoned and drained](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/) to minimize disruption to running applications. Refer to [Scale the node count in an Azure Kubernetes Service (AKS) cluster](https://docs.microsoft.com/en-us/azure/aks/scale-cluster).

## Configure StorageClass

To improve disk IO performance, it is recommended to add `mountOptions` in `StorageClass` to configure `nodelalloc` and `noatime`. Refer to [Mount the data disk ext4 filesystem with options on the target machines that deploy TiKV](https://docs.pingcap.com/tidb/stable/check-before-deployment#mount-the-data-disk-ext4-filesystem-with-options-on-the-target-machines-that-deploy-tikv).

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
# ...
mountOptions:
- nodelalloc,noatime
```

## Deploy TiDB Operator

Deploy TiDB Operator in the AKS cluster by referring to [*Deploy TiDB Operator* section](get-started.md#deploy-tidb-operator).

## Deploy a TiDB cluster and the monitoring component

This section describes how to deploy a TiDB cluster and its monitoring component in Azure AKS.

### Create namespace

To create a namespace to deploy the TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl create namespace tidb-cluster
```

> **Note:**
>
> A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. This document takes `tidb-cluster` as an example. If you want to use other namespaces, modify the corresponding arguments of `-n` or `--namespace`.

### Deploy

First, download the sample `TidbCluster` and `TidbMonitor` configuration files:

{{< copyable "shell-regular" >}}

```bash
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aks/tidb-cluster.yaml && \
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aks/tidb-monitor.yaml
```

Refer to [configure the TiDB cluster](configure-a-tidb-cluster.md) to further customize and configure the CR before applying.

> **Note:**
>
> By default, TiDB LoadBalancer in `tidb-cluster.yaml` is set to "internal", indicating that the LoadBalancer is only accessible within the cluster virtual network, not externally. To access TiDB over the MySQL protocol, you need to use a bastion to access the internal host of the cluster or use `kubectl port-forward`. You can delete the "internal" schema in the `tidb-cluster.yaml` file to expose the LoadBalancer publicly by default. However, notice that this practice may expose TiDB to risks.

To deploy the `TidbCluster` and `TidbMonitor` CR in the AKS cluster, run the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl apply -f tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```

After the yaml file above is applied to the Kubernetes cluster, TiDB Operator creates the desired TiDB cluster and its monitoring component according to the yaml file.

### View the cluster status

To view the status of the TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl get pods -n tidb-cluster
```

When all the pods are in the `Running` or `Ready` state, the TiDB cluster is successfully started. For example:

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

## Access the database

After deploying a TiDB cluster, you can access the TiDB database to test or develop applications.

### Access method

- Access via Bastion

The LoadBalancer created for your TiDB cluster resides in an intranet. You can create a [Bastion](https://docs.microsoft.com/en-us/azure/bastion/tutorial-create-host-portal) in the cluster virtual network to connect to an internal host and then access the database.

> **Note:**
>
> In addition to the bastion host, you can also connect an existing host to the cluster virtual network by [Peering](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-peering-overview). If the AKS cluster is created in an existing virtual network, you can use hosts in this virtual network to access the database.

- Access via SSH

You can [create the SSH connection to a Linux node](https://docs.microsoft.com/en-us/azure/aks/ssh#create-the-ssh-connection-to-a-linux-node) to access the database.

- Access via node-shell

You can simply use tools like [node-shell](https://github.com/kvaps/kubectl-node-shell) to connect to nodes in the cluster, then access the database.

### Access via the MySQL client

After access to the internal host via SSH, you can access the TiDB cluster through the MySQL client.

1. Install the MySQL client on the host:

    {{< copyable "shell-regular" >}}

    ```bash
    sudo yum install mysql -y
    ```

2. Connect the client to the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```bash
    mysql --comments -h ${tidb-lb-ip} -P 4000 -u root
    ```

    `${tidb-lb-ip}` is the LoadBalancer IP address of the TiDB service. To obtain it, run the `kubectl get svc basic-tidb -n tidb-cluster` command. The `EXTERNAL-IP` field returned is the IP address.

    For example:

    ```bash
    $ mysql --comments -h 20.240.0.7 -P 4000 -u root
    Welcome to the MariaDB monitor.  Commands end with ; or \g.
    Your MySQL connection id is 1189
    Server version: 5.7.25-TiDB-v4.0.2 TiDB Server (Apache License 2.0) Community Edition, MySQL 5.7 compatible

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
    | server_id          | ed4ba88b-436a-424d-9087-977e897cf5ec |
    +--------------------+--------------------------------------+
    6 rows in set (0.00 sec)
    ```

> **Note:**
>
> * [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you access the TiDB service (earlier than v4.0.7) by using MySQL 8.0 client via password authentication, you need to specify the `--default-auth=mysql_native_password` parameter.
> * By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).

## Access the Grafana monitoring dashboard

Obtain the LoadBalancer IP address of Grafana:

{{< copyable "shell-regular" >}}

```bash
kubectl -n tidb-cluster get svc basic-grafana
```

For example:

```bash
kubectl get svc basic-grafana
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
basic-grafana   LoadBalancer   10.100.199.42   20.240.0.8    3000:30761/TCP   121m
```

In the output above, the `EXTERNAL-IP` column is the LoadBalancer IP address.

You can access the `${grafana-lb}:3000` address using your web browser to view monitoring metrics. Replace `${grafana-lb}` with the LoadBalancer IP address.

> **Note:**
>
> The default Grafana username and password are both `admin`.

## Access TiDB Dashboard

See [Access TiDB Dashboard](access-dashboard.md) for instructions about how to securely allow access to TiDB Dashboard.

## Upgrade

To upgrade the TiDB cluster, edit `spec.version` by running the `kubectl edit tc basic -n tidb-cluster` command.

The upgrade process does not finish immediately. You can view the upgrade progress by running the `kubectl get pods -n tidb-cluster --watch` command.

## Scale out

Before scaling out the cluster, you need to scale out the corresponding node pool so that the new instances have enough resources for operation.

This section describes how to scale out the AKS node pool and TiDB components.

### Scale out AKS node pool

When scaling out TiKV, the node pools must be scaled out evenly among availability zones. The following example shows how to scale out the TiKV node pool of the `${clusterName}` cluster to 6 nodes:

{{< copyable "shell-regular" >}}

```bash
az aks nodepool scale \
    --resource-group ${resourceGroup} \
    --cluster-name ${clusterName} \
    --name ${nodePoolName} \
    --node-count 6
```

For more information on node pool management, refer to [`az aks nodepool`](https://docs.microsoft.com/zh-cn/cli/azure/aks/nodepool?view=azure-cli-latest).

### Scale out TiDB components

After scaling out the AKS node pool, run the `kubectl edit tc basic -n tidb-cluster` command with `replicas` of each component set to desired value. The scaling-out process is then completed.

## Deploy TiFlash/TiCDC

[TiFlash](https://docs.pingcap.com/tidb/stable/tiflash-overview) is the columnar storage extension of TiKV.

[TiCDC](https://docs.pingcap.com/tidb/stable/ticdc-overview) is a tool for replicating the incremental data of TiDB by pulling TiKV change logs.

The two components are *not required* in the deployment. This section shows a quick start example.

### Add node pools

Add a node pool for TiFlash/TiCDC respectively. You can set `--node-count` as required.

1. Create a TiFlash node pool with `nodeType` being `Standard_E8s_v4` or higher:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tiflash \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=tiflash \
        --node-taints dedicated=tiflash:NoSchedule
    ```

2. Create a TiCDC node pool with `nodeType` being `Standard_E16s_v4` or higher:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name ticdc \
        --cluster-name ${clusterName} \
        --resource-group ${resourceGroup} \
        --node-vm-size ${nodeType} \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --labels dedicated=ticdc \
        --node-taints dedicated=ticdc:NoSchedule
    ```

### Configure and deploy

+ To deploy TiFlash, configure `spec.tiflash` in `tidb-cluster.yaml`. The following is an example:

    ```yaml
    spec:
      ...
      tiflash:
        baseImage: pingcap/tiflash
        maxFailoverCount: 0
        replicas: 1
        storageClaims:
        - resources:
            requests:
              storage: 100Gi
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: tiflash
    ```

    For other parameters, refer to [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

    > **Warning:**
    >
    > TiDB Operator automatically mounts PVs **in the order of the configuration** in the `storageClaims` list. Therefore, if you need to add disks for TiFlash, make sure that you add the disks **only to the end of the original configuration** in the list. In addition, you must **not** alter the order of the original configuration.

+ To deploy TiCDC, configure `spec.ticdc` in `tidb-cluster.yaml`. The following is an example:

    ```yaml
    spec:
      ...
      ticdc:
        baseImage: pingcap/ticdc
        replicas: 1
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: ticdc
    ```

    Modify `replicas` as required.

Finally, run the `kubectl -n tidb-cluster apply -f tidb-cluster.yaml` command to update the TiDB cluster configuration.

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

## Use other Disk volume types

Azure disks support multiple volume types. Among them, `UltraSSD` delivers low latency and high throughput and can be enabled by performing the following steps:

1. [Enable Ultra disks on an existing cluster](https://docs.microsoft.com/en-us/azure/aks/use-ultra-disks#enable-ultra-disks-on-an-existing-cluster) and create a storage class for `UltraSSD`:

    ```yaml
    apiVersion: storage.k8s.io/v1
    kind: StorageClass
    metadata:
      name: ultra
    provisioner: disk.csi.azure.com
    parameters:
      skuname: UltraSSD_LRS  # alias: storageaccounttype, available values: Standard_LRS, Premium_LRS, StandardSSD_LRS, UltraSSD_LRS
      cachingMode: None
    reclaimPolicy: Delete
    allowVolumeExpansion: true
    volumeBindingMode: WaitForFirstConsumer
    mountOptions:
    - nodelalloc,noatime
    ```

    You can add more [Driver Parameters](https://github.com/kubernetes-sigs/azuredisk-csi-driver/blob/master/docs/driver-parameters.md) as required.

2. In `tidb-cluster.yaml`, specify the `ultra` storage class to apply for the `UltraSSD` volume type through the `storageClassName` field.

    The following is a TiKV configuration example you can refer to:

    ```yaml
    spec:
      tikv:
        ...
        storageClassName: ultra
    ```

You can use any supported Azure disk type. It is recommended to use `Premium_LRS` or `UltraSSD_LRS`.

For more information about the storage class configuration and Azure disk types, refer to [Storage Class documentation](https://github.com/kubernetes-sigs/azuredisk-csi-driver) and [Azure Disk Types](https://docs.microsoft.com/en-us/azure/virtual-machines/disks-types).

## Use local storage

Use Azure LRS disks for storage in production environment. To simulate bare-metal performance, use additional [NVMe SSD local store volumes](https://docs.microsoft.com/en-us/azure/virtual-machines/sizes-storage) provided by some Azure instances. You can choose such instances for the TiKV node pool to achieve higher IOPS and lower latency.

> **Note:**
>
> * You cannot dynamically change the storage class of a running TiDB cluster. In this case, create a new cluster for testing.
> * Local NVMe Disks are ephemeral. Data will be lost on these disks if you stop/deallocate your node. When the node is reconstructed, you need to migrate data in TiKV. If you do not want to migrate data, it is recommended not to use the local disk in a production environment.

For instance types that provide local disks, refer to [Lsv2-series](https://docs.microsoft.com/en-us/azure/virtual-machines/lsv2-series). The following takes `Standard_L8s_v2` as an example:

1. Create a node pool with local storage for TiKV.

    Modify the instance type of the TiKV node pool in the `az aks nodepool add` command to `Standard_L8s_v2`:

    {{< copyable "shell-regular" >}}

    ```bash
    az aks nodepool add --name tikv \
        --cluster-name ${clusterName}  \
        --resource-group ${resourceGroup} \
        --node-vm-size Standard_L8s_v2 \
        --zones 1 2 3 \
        --aks-custom-headers EnableAzureDiskFileCSIDriver=true \
        --node-count 3 \
        --enable-ultra-ssd \
        --labels dedicated=tikv \
        --node-taints dedicated=tikv:NoSchedule
    ```

    If the TiKV node pool already exists, you can either delete the old group and then create a new one, or change the group name to avoid conflict.

2. Deploy the local volume provisioner.

    You need to use the [local-volume-provisioner](https://sigs.k8s.io/sig-storage-local-static-provisioner) to discover and manage the local storage. Run the following command to deploy and create a `local-storage` storage class:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/eks/local-volume-provisioner.yaml
    ```

3. Use local storage.

    After the steps above, the local volume provisioner can discover all the local NVMe SSD disks in the cluster.

    Add the `tikv.storageClassName` field to the `tidb-cluster.yaml` file and set the value of the field to `local-storage`.

    For more information, refer to [Deploy TiDB cluster and its monitoring components](#deploy)
