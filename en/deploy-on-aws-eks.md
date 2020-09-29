---
title: Deploy TiDB on AWS EKS
summary: Learn how to deploy a TiDB cluster on AWS EKS.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-on-aws-eks/']
---

# Deploy TiDB on AWS EKS

This document describes how to deploy a TiDB cluster on AWS Elastic Kubernetes Service (EKS).

## Prerequisites

Before deploying a TiDB cluster on AWS EKS, make sure the following requirements are satisfied:

* Install [Helm](https://helm.sh/docs/intro/install/): used for deploying TiDB Operator.
* Complete all operations in [Getting started with eksctl](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html).

    This guide includes the following contents:

    * Install and configure `awscli`.
    * Install and configure `eksctl` that is used for creating Kubernetes clusters.
    * Install `kubectl`.

> **Note:**
>
> The operations described in this document requires at least the [minumum privileges needed by `eksctl`](https://eksctl.io/usage/minimum-iam-policies/) and the [services privileges needed to create a Linux bastion host](https://docs.aws.amazon.com/quickstart/latest/linux-bastion/architecture.html#aws-services).

## Deploy

This section describes how to deploy EKS, TiDB operator, the TiDB cluster, and the monitoring component.

### Create EKS and the node pool

{{< copyable "shell-regular" >}}

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: <clusterName>
  region: us-west-2

nodeGroups:
  - name: admin
    desiredCapacity: 1
    labels:
      dedicated: admin

  - name: tidb
    desiredCapacity: 2
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule

  - name: pd
    desiredCapacity: 3
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule

  - name: tikv
    desiredCapacity: 3
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
```

Save the configuration above as `cluster.yaml`, and replace `<clusterName>` with your desired cluster name. Execute the following command to create the cluster:

{{< copyable "shell-regular" >}}

```shell
eksctl create cluster -f cluster.yaml
```

> **Note:**
>
> - After executing the command above, you need to wait until the EKS cluster is successfully created and the node group is created and added in the EKS cluster. This process might take 5 to 10 minutes.
> - For more cluster configuration, refer to [`eksctl` documentation](https://eksctl.io/usage/creating-and-managing-clusters/#using-config-files).

### Deploy TiDB Operator

To deploy TiDB Operator in the Kubernetes cluster, refer to the [*Deploy TiDB Operator* section](get-started.md#deploy-tidb-operator) in Getting Started.

### Deploy a TiDB cluster and the monitoring component

1. Prepare the TidbCluster and TidbMonitor CR files:

    {{< copyable "shell-regular" >}}

    ```shell
    curl -LO https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-cluster.yaml &&
    curl -LO https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-monitor.yaml
    ```

2. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create namespace tidb-cluster
    ```

    > **Note:**
    >
    > A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. This document takes `tidb-cluster` as an example. If you want to use other namespace, modify the corresponding arguments of `-n` or `--namespace`.

3. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create -f tidb-cluster.yaml -n tidb-cluster &&
    kubectl create -f tidb-monitor.yaml -n tidb-cluster
    ```

4. View the startup status of the TiDB cluster:

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

## Access the database

After you deploy a TiDB cluster, you can access the TiDB database via MySQL client.

### Prepare a host that can access the cluster

The LoadBalancer created for your TiDB cluster is an intranet LoadBalancer. You can create a [bastion host](https://aws.amazon.com/quickstart/architecture/linux-bastion/) in the cluster VPC to access the database. To create a bastion host on AWS console, refer to [AWS documentation](https://aws.amazon.com/quickstart/architecture/linux-bastion/).

Select the cluster's VPC and Subnet, and verify whether the cluster name is correct in the dropdown box. You can view the cluster's VPC and Subnet by running the following command:

{{< copyable "shell-regular" >}}

```shell
eksctl get cluster -n <clusterName>
```

Allow the bastion host to access the Internet. Select the correct key pair so that you can log in to the host via SSH.

> **Note:**
>
> - In addition to the bastion host, you can also connect the existing machine to the cluster VPC by [VPC Peering](https://docs.aws.amazon.com/vpc/latest/peering/what-is-vpc-peering.html).
> - If the EKS cluster is created in an existing VPC, you can use the host inside the VPC.

### Install the MySQL client and connect

After the bastion host is created, you can connect to the bastion host via SSH and access the TiDB cluster via the MySQL client.

1. Connect to the bastion host via SSH:

    {{< copyable "shell-regular" >}}

    ```shell
    ssh [-i /path/to/your/private-key.pem] ec2-user@<bastion-public-dns-name>
    ```

2. Install the MySQL client:

    {{< copyable "shell-regular" >}}

    ```shell
    sudo yum install mysql -y
    ```

3. Connect the client to the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    mysql -h <tidb-nlb-dnsname> -P 4000 -u root
    ```

    For example:

    ```shell
    $ mysql -h abfc623004ccb4cc3b363f3f37475af1-9774d22c27310bc1.elb.us-west-2.amazonaws.com -P 4000 -u root
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

    `<tidb-nlb-dnsname>` is the LoadBalancer domain name of the TiDB service. You can view the domain name in the `EXTERNAL-IP` field by executing `kubectl get svc basic-tidb -n tidb-cluster`.

> **Note:**
>
> * [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you use MySQL client from MySQL 8.0 to access the TiDB service (TiDB version < v4.0.7), and if the user account has a password, you need to explicitly specify the `--default-auth=mysql_native_password` parameter.
> * By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).

## Monitor

Obtain the LoadBalancer domain name of Grafana:

{{< copyable "shell-regular" >}}

```shell
kubectl -n tidb-cluster get svc basic-grafana
```

For example:

```
$ kubectl get svc basic-grafana
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP                                                             PORT(S)          AGE
basic-grafana   LoadBalancer   10.100.199.42   a806cfe84c12a4831aa3313e792e3eed-1964630135.us-west-2.elb.amazonaws.com 3000:30761/TCP   121m
```

In the output above, the `EXTERNAL-IP` column is the LoadBalancer domain name.

You can access the `<grafana-lb>:3000` address using your web browser to view monitoring metrics. Replace `<grafana-lb>` with the LoadBalancer domain name.

The initial Grafana login credentials are:

- User: admin
- Password: admin

## Upgrade

To upgrade the TiDB cluster, edit the `spec.version` by executing `kubectl edit tc basic -n tidb-cluster`.

The upgrade process does not finish immediately. You can watch the upgrade progress by executing `kubectl get pods -n tidb-cluster --watch`.

## Scale out

Before scaling out the cluster, you need to scale out the corresponding node group so that the new instances have enough resources for operation.

The following example shows how to scale out the `tikv` group of the `<clusterName>` cluster to 4 nodes:

{{< copyable "shell-regular" >}}

```shell
eksctl scale nodegroup --cluster <clusterName> --name tikv --nodes 4 --nodes-min 4 --nodes-max 4
```

After that, execute `kubectl edit tc basic -n tidb-cluster`, and modify each component's `replicas` to the desired number of replicas. The scaling-out process is then completed.

For more information on managing node groups, refer to [`eksctl` documentation](https://eksctl.io/usage/managing-nodegroups/).

## Deploy TiFlash/TiCDC

### Add node groups

In the configuration file of eksctl (`cluster.yaml`), add the following two items to add a node group for TiFlash/TiCDC respectively. `desiredCapacity` is the number of nodes you desire.

```
- name: tiflash
    desiredCapacity: 3
    labels:
      role: tiflash
    taints:
      dedicated: tiflash:NoSchedule
  - name: ticdc
    desiredCapacity: 1
    labels:
      role: ticdc
    taints:
      dedicated: ticdc:NoSchedule
```

- If the cluster is not created, execute `eksctl create cluster -f cluster.yaml` to create the cluster and node groups.
- If the cluster is already created, execute `eksctl create nodegroup -f cluster.yaml` to create the node groups. The existing node groups are ignored and will not be created again.

### Configure and deploy

+ If you want to deploy TiFlash, configure `spec.tiflash` in `tidb-cluster.yaml`:

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
        tolerations:
        - effect: NoSchedule
          key: dedicated
          operator: Equal
          value: tiflash
    ```

    > **Warning:**
    >
    > TiDB Operator automatically mount PVs **in the order of the configuration** in the `storageClaims` list. Therefore, if you need to add disks for TiFlash, make sure that you add the disks **only to the end of the original configuration** in the list. In addition, you must **not** alter the order of the original configuration.

+ If you want to deploy TiCDC, configure `spec.ticdc` in `tidb-cluster.yaml`:

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

    Modify `replicas` according to your needs.

Finally, execute `kubectl -n tidb-cluster apply -f tidb-cluster.yaml` to update the TiDB cluster configuration.

For detailed CR configuration, refer to [API references](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md) and [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

## Deploy TiDB Enterprise Edition

If you need to deploy TiDB/PD/TiKV/TiFlash/TiCDC Enterprise Edition, configure `spec.<tidb/pd/tikv/tiflash/ticdc>.baseImage` in `tidb-cluster.yaml` as the enterprise image. The image format is `pingcap/<tidb/pd/tikv/tiflash/ticdc>-enterprise`.

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

## Use other EBS volume types

AWS EBS supports multiple volume types. If you need low latency and high throughput, you can choose the `io1` type. The steps are as follows:

1. Create a storage class for `io1`:

    ```
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    metadata:
      name: io1
    provisioner: kubernetes.io/aws-ebs
    parameters:
      type: io1
      fsType: ext4
    ```

2. In `tidb-cluster.yaml`, specify the `io1` storage class to apply for the `io1` volume type through the `storageClassName` field.

For more information about the storage class configuration and EBS volume types, refer to [Storage Class documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/) and [EBS Volume Types](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html).

## Use local storage

Some AWS instance types provide additional [NVMe SSD local store volumes](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html). You can choose such instances for the TiKV node pool to achieve higher IOPS and lower latency.

> **Note:**
>
> You cannot dynamically change the storage class of a running TiDB cluster. You can create a new cluster for testing.
> 
> During the EKS upgrade, data in the local storage will be [lost](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-lifetime) due to the node reconstruction. When the node reconstruction occurs, you need to migrate data in TiKV. If you do not want to migrate data, it is recommended not to use the local disk in the production environment.

For instance types that provide local volumes, see [AWS Instance Types](https://aws.amazon.com/ec2/instance-types/). Take `c5d.4xlarge` as an example:

1. Create a node group with local storage for TiKV.

    Modify the instance type of the TiKV node group in the `eksctl` configuration file to `c5d.4xlarge`:

    ```
      - name: tikv
        instanceType: c5d.4xlarge
        labels:
          role: tikv
        taints:
          dedicated: tikv:NoSchedule
        ...
    ```

    Create the node group:

    {{< copyable "shell-regular" >}}

    ```shell
    eksctl create nodegroups -f cluster.yaml
    ```

    If the TiKV node group already exists, you can either delete the old group and then create a new one, or change the group name to avoid conflict.

2. Deploy the local volume provisioner.

    You need to use the [local-volume-provisioner](https://sigs.k8s.io/sig-storage-local-static-provisioner) to discover and manage the local storage. Executing the following command deploys and creates a `local-storage` storage class:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/eks/local-volume-provisioner.yaml
    ```

3. Use the local storage.

    After the steps above, the local volume provisioner can discover all the local NVMe SSD disks in the cluster.

    Modify `tikv.storageClassName` in the `tidb-cluster.yaml` file to `local-storage`.
