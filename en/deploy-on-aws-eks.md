---
title: Deploy TiDB on AWS EKS
summary: Learn how to deploy a TiDB cluster on AWS Elastic Kubernetes Service (EKS).
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-on-aws-eks/']
---

# Deploy TiDB on AWS EKS

This document describes how to deploy a TiDB cluster on AWS Elastic Kubernetes Service (EKS).

To deploy TiDB Operator and the TiDB cluster in a self-managed Kubernetes environment, refer to [Deploy TiDB Operator](deploy-tidb-operator.md) and [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md).

## Prerequisites

Before deploying a TiDB cluster on AWS EKS, make sure the following requirements are satisfied:

* Install [Helm 3](https://helm.sh/docs/intro/install/): used for deploying TiDB Operator.
* Complete all operations in [Getting started with eksctl](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html).

    This guide includes the following contents:

    * Install and configure `awscli`.
    * Install and configure `eksctl` used for creating Kubernetes clusters.
    * Install `kubectl`.

To verify whether AWS CLI is configured correctly, run the `aws configure list` command. If the output shows the values for `access_key` and `secret_key`, AWS CLI is configured correctly. Otherwise, you need to re-configure AWS CLI.

> **Note:**
>
> The operations described in this document require at least the [minimum privileges needed by `eksctl`](https://eksctl.io/usage/minimum-iam-policies/) and the [service privileges needed to create a Linux bastion host](https://docs.aws.amazon.com/quickstart/latest/linux-bastion/architecture.html#aws-services).

## Recommended instance types and storage

- Instance types: to gain better performance, the following is recommended:
    - PD nodes: `c5.xlarge`
    - TiDB nodes: `c5.2xlarge`
    - TiKV or TiFlash nodes: `r5b.2xlarge`
- Storage: Because AWS supports the [EBS `gp3`](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html#gp3-ebs-volume-type) volume type, it is recommended to use EBS `gp3`. For `gp3` provisioning, the following is recommended:
    - TiKV: 400 MiB/s, 4000 IOPS
    - TiFlash: 625 MiB/s, 6000 IOPS

## Create an EKS cluster and a node pool

According to AWS [Official Blog](https://aws.amazon.com/blogs/containers/amazon-eks-cluster-multi-zone-auto-scaling-groups/) recommendation and EKS [Best Practice Document](https://aws.github.io/aws-eks-best-practices/reliability/docs/dataplane/#ensure-capacity-in-each-az-when-using-ebs-volumes), since most of the TiDB cluster components use EBS volumes as storage, it is recommended to create a node pool in each availability zone (at least 3 in total) for each component when creating an EKS.

Save the following configuration as the `cluster.yaml` file. Replace `${clusterName}` with your desired cluster name. The cluster and node group names should match the regular expression `[a-zA-Z][-a-zA-Z0-9]*`, so avoid names that contain `_`.

{{< copyable "" >}}

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: ${clusterName}
  region: ap-northeast-1

nodeGroups:
  - name: admin
    desiredCapacity: 1
    privateNetworking: true
    labels:
      dedicated: admin

  - name: tidb-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    instanceType: c5.2xlarge
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule
  - name: tidb-1d
    desiredCapacity: 0
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    instanceType: c5.2xlarge
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule
  - name: tidb-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    instanceType: c5.2xlarge
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule

  - name: pd-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    instanceType: c5.xlarge
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule
  - name: pd-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    instanceType: c5.xlarge
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule
  - name: pd-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    instanceType: c5.xlarge
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule

  - name: tikv-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    instanceType: r5b.2xlarge
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
  - name: tikv-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    instanceType: r5b.2xlarge
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
  - name: tikv-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    instanceType: r5b.2xlarge
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
```

By default, only two TiDB nodes are required, so you can set the `desiredCapacity` of the `tidb-1d` node group to `0`. You can scale out this node group any time if necessary.

Execute the following command to create the cluster:

{{< copyable "shell-regular" >}}

```shell
eksctl create cluster -f cluster.yaml
```

After executing the command above, you need to wait until the EKS cluster is successfully created and the node group is created and added in the EKS cluster. This process might take 5 to 20 minutes. For more cluster configuration, refer to [`eksctl` documentation](https://eksctl.io/usage/creating-and-managing-clusters/#using-config-files).

> **Warning:**
>
> If the Regional Auto Scaling Group (ASG) is used:
>
> * [Enable the instance scale-in protection](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-instance-termination.html#instance-protection-instance) for all the EC2s that have been started. The instance scale-in protection for the ASG is not required.
> * [Set termination policy](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-instance-termination.html#custom-termination-policy) to `NewestInstance` for the ASG.

## Configure StorageClass

This section describes how to configure the storage class for different storage types. These storage types are:

- The default `gp2` storage type after creating the EKS cluster.
- The `gp3` storage type (recommended) or other EBS storage types.
- The local storage used for testing bare-metal performance.

### Configure `gp2`

After you create an EKS cluster, the default StorageClass is `gp2`. To improve I/O write performance, it is recommended to configure `nodelalloc` and `noatime` in the `mountOptions` field of the `StorageClass` resource.

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
# ...
mountOptions:
- nodelalloc,noatime
```

For more information on the mount options, see [TiDB Environment and System Configuration Check](https://docs.pingcap.com/tidb/stable/check-before-deployment#mount-the-data-disk-ext4-filesystem-with-options-on-the-target-machines-that-deploy-tikv).

### Configure `gp3` (recommended) or other EBS storage types

If you do not want to use the default `gp2` storage type, you can create StorageClass for other storage types. For example, you can use the `gp3` (recommended) or `io1` storage type.

The following example shows how to create and configure a StorageClass for the `gp3` storage type:

1. Deploy the [AWS EBS Container Storage Interface (CSI) driver](https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html) on the EKS cluster. If you are using a storage type other than `gp3`, skip this step.

2. Create a `StorageClass` resource. In the resource definition, specify your desired storage type in the `parameters.type` field.

    ```yaml
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    metadata:
      name: gp3
    provisioner: kubernetes.io/aws-ebs
    parameters:
      type: gp3
      fsType: ext4
      iopsPerGB: "10"
      encrypted: "false"
    mountOptions:
    - nodelalloc,noatime
    ```

3. In the TidbCluster YAML file, configure `gp3` in the `storageClassName` field. For example:

    ```yaml
    spec:
      tikv:
        baseImage: pingcap/tikv
        replicas: 3
        requests:
          storage: 100Gi
        storageClassName: gp3
    ```

4. To improve I/O write performance, it is recommended to configure `nodelalloc` and `noatime` in the `mountOptions` field of the `StorageClass` resource.

    ```yaml
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    # ...
    mountOptions:
    - nodelalloc,noatime
    ```

    For more information on the mount options, see [TiDB Environment and System Configuration Check](https://docs.pingcap.com/tidb/stable/check-before-deployment#mount-the-data-disk-ext4-filesystem-with-options-on-the-target-machines-that-deploy-tikv).

For more information on the EBS storage types and configuration, refer to [Amazon EBS volume types](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html) and [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/).

### Configure local storage

Local storage is used for testing bare-metal performance. For higher IOPS and lower latency, you can choose [NVMe SSD volumes](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html) offered by some AWS instances for the TiKV node pool. However, for the production environment, use AWS EBS as your storage type.

> **Note:**
>
> - You cannot dynamically change StorageClass for a running TiDB cluster. For testing purposes, create a new TiDB cluster with the desired StorageClass.
> - EKS upgrade or other reasons might cause node reconstruction. In such cases, [data in the local storage might be lost](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-lifetime). To avoid data loss, you need to back up TiKV data before node reconstruction.
> - To avoid data loss from node reconstruction, you can refer to [AWS documentation](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-suspend-resume-processes.html) and disable the `ReplaceUnhealthy` feature of the TiKV node group.

For instance types that provide NVMe SSD volumes, check out [Amazon EC2 Instance Types](https://aws.amazon.com/ec2/instance-types/).

The following `c5d.4xlarge` example shows how to configure StorageClass for the local storage:

1. Create a node group with local storage for TiKV.

    1. In the `eksctl` configuration file, modify the instance type of the TiKV node group to `c5d.4xlarge`:

        ```yaml
          - name: tikv-1a
            desiredCapacity: 1
            privateNetworking: true
            availabilityZones: ["ap-northeast-1a"]
            instanceType: c5d.4xlarge
            labels:
              dedicated: tikv
            taints:
              dedicated: tikv:NoSchedule
            ...
        ```

    2. Create a node group with local storage:

        {{< copyable "shell-regular" >}}

        ```shell
        eksctl create nodegroups -f cluster.yaml
        ```

    If the TiKV node group already exists, to avoid name conflict, you can take either of the following actions:

    - Delete the old group and create a new one.
    - Change the group name.

2. Deploy local volume provisioner.

    1. To conveniently discover and manage local storage volumes, install [local-volume-provisioner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner).

    2. [Mount the local storage](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/master/docs/operations.md#use-a-whole-disk-as-a-filesystem-pv) to the `/mnt/ssd` directory.

    3. According to the mounting configuration, modify the [local-volume-provisioner.yaml](https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/eks/local-volume-provisioner.yaml) file.

    4. Deploy and create a `local-storage` storage class using the modified `local-volume-provisioner.yaml` file.

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f <local-volume-provisioner.yaml>
        ```

3. Use the local storage.

    After you complete the previous step, local-volume-provisioner can discover all the local NVMe SSD volumes in the cluster.

After local-volume-provisioner discovers the local volumes, when you [Deploy a TiDB cluster and the monitoring component](#deploy-a-tidb-cluster-and-the-monitoring-component), you need to add the `tikv.storageClassName` field to `tidb-cluster.yaml` and set the field value to `local-storage`.

## Deploy TiDB Operator

To deploy TiDB Operator in the EKS cluster, refer to the [*Deploy TiDB Operator* section](get-started.md#deploy-tidb-operator) in Getting Started.

## Deploy a TiDB cluster and the monitoring component

This section describes how to deploy a TiDB cluster and its monitoring component in AWS EKS.

### Create namespace

To create a namespace to deploy the TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl create namespace tidb-cluster
```

> **Note:**
>
> A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. This document takes `tidb-cluster` as an example. If you want to use another namespace, modify the corresponding arguments of `-n` or `--namespace`.

### Deploy

First, download the sample `TidbCluster` and `TidbMonitor` configuration files:

{{< copyable "shell-regular" >}}

```shell
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-cluster.yaml && \
curl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/aws/tidb-monitor.yaml
```

Refer to [configure the TiDB cluster](configure-a-tidb-cluster.md) to further customize and configure the CR before applying.

> **Note:**
>
> By default, the configuration in `tidb-cluster.yaml` sets up the LoadBalancer for TiDB with the "internal" scheme. This means that the LoadBalancer is only accessible within the VPC, not externally. To access TiDB over the MySQL protocol, you need to use a bastion host or use `kubectl port-forward`. If you want to expose TiDB over the internet and if you are aware of the risks of doing this, you can change the scheme for the LoadBalancer from "internal" to "internet-facing" in the `tidb-cluster.yaml` file.

To deploy the `TidbCluster` and `TidbMonitor` CR in the EKS cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```

After the YAML file above is applied to the Kubernetes cluster, TiDB Operator creates the desired TiDB cluster and its monitoring component according to the YAML file.

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

## Access the database

After you have deployed a TiDB cluster, you can access the TiDB database to test or develop your application.

### Prepare a bastion host

The LoadBalancer created for your TiDB cluster is an intranet LoadBalancer. You can create a [bastion host](https://aws.amazon.com/quickstart/architecture/linux-bastion/) in the cluster VPC to access the database. To create a bastion host on AWS console, refer to [AWS documentation](https://aws.amazon.com/quickstart/architecture/linux-bastion/).

Select the cluster's VPC and Subnet, and verify whether the cluster name is correct in the dropdown box. You can view the cluster's VPC and Subnet by running the following command:

{{< copyable "shell-regular" >}}

```shell
eksctl get cluster -n ${clusterName}
```

Allow the bastion host to access the Internet. Select the correct key pair so that you can log in to the host via SSH.

> **Note:**
>
> In addition to the bastion host, you can also connect an existing host to the cluster VPC by [VPC Peering](https://docs.aws.amazon.com/vpc/latest/peering/what-is-vpc-peering.html). If the EKS cluster is created in an existing VPC, you can use the host in the VPC.

### Install the MySQL client and connect

After the bastion host is created, you can connect to the bastion host via SSH and access the TiDB cluster via the MySQL client.

1. Log in to the bastion host via SSH:

    {{< copyable "shell-regular" >}}

    ```shell
    ssh [-i /path/to/your/private-key.pem] ec2-user@<bastion-public-dns-name>
    ```

2. Install the MySQL client on the bastion host:

    {{< copyable "shell-regular" >}}

    ```shell
    sudo yum install mysql -y
    ```

3. Connect the client to the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    mysql --comments -h ${tidb-nlb-dnsname} -P 4000 -u root
    ```

    `${tidb-nlb-dnsname}` is the LoadBalancer domain name of the TiDB service. You can view the domain name in the `EXTERNAL-IP` field by executing `kubectl get svc basic-tidb -n tidb-cluster`.

    For example:

    ```shell
    $ mysql --comments -h abfc623004ccb4cc3b363f3f37475af1-9774d22c27310bc1.elb.us-west-2.amazonaws.com -P 4000 -u root
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
> * [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you use MySQL client from MySQL 8.0 to access the TiDB service (cluster version < v4.0.7), and if the user account has a password, you need to explicitly specify the `--default-auth=mysql_native_password` parameter.
> * By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).

## Access the Grafana monitoring dashboard

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

You can access the `${grafana-lb}:3000` address using your web browser to view monitoring metrics. Replace `${grafana-lb}` with the LoadBalancer domain name.

> **Note:**
>
> The default Grafana username and password are both `admin`.

## Access the TiDB Dashboard

See [Access TiDB Dashboard](access-dashboard.md) for instructions about how to securely allow access to the TiDB Dashboard.

## Upgrade

To upgrade the TiDB cluster, edit the `spec.version` by executing `kubectl edit tc basic -n tidb-cluster`.

The upgrade process does not finish immediately. You can watch the upgrade progress by executing `kubectl get pods -n tidb-cluster --watch`.

## Scale out

Before scaling out the cluster, you need to scale out the corresponding node group so that the new instances have enough resources for operation.

This section describes how to scale out the EKS node group and TiDB components.

### Scale out EKS node group

When scaling out TiKV, the node groups must be scaled out evenly among the different availability zones. The following example shows how to scale out the `tikv-1a`, `tikv-1c`, and `tikv-1d` groups of the `${clusterName}` cluster to 2 nodes:

{{< copyable "shell-regular" >}}

```shell
eksctl scale nodegroup --cluster ${clusterName} --name tikv-1a --nodes 2 --nodes-min 2 --nodes-max 2
eksctl scale nodegroup --cluster ${clusterName} --name tikv-1c --nodes 2 --nodes-min 2 --nodes-max 2
eksctl scale nodegroup --cluster ${clusterName} --name tikv-1d --nodes 2 --nodes-min 2 --nodes-max 2
```

For more information on managing node groups, refer to [`eksctl` documentation](https://eksctl.io/usage/managing-nodegroups/).

### Scale out TiDB components

After scaling out the EKS node group, execute `kubectl edit tc basic -n tidb-cluster`, and modify each component's `replicas` to the desired number of replicas. The scaling-out process is then completed.

## Deploy TiFlash/TiCDC

[TiFlash](https://docs.pingcap.com/tidb/stable/tiflash-overview) is the columnar storage extension of TiKV.

[TiCDC](https://docs.pingcap.com/tidb/stable/ticdc-overview) is a tool for replicating the incremental data of TiDB by pulling TiKV change logs.

The two components are *not required* in the deployment. This section shows a quick start example.

### Add node groups

In the configuration file of eksctl (`cluster.yaml`), add the following two items to add a node group for TiFlash/TiCDC respectively. `desiredCapacity` is the number of nodes you desire.

```yaml
  - name: tiflash-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    labels:
      dedicated: tiflash
    taints:
      dedicated: tiflash:NoSchedule
  - name: tiflash-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: tiflash
    taints:
      dedicated: tiflash:NoSchedule
  - name: tiflash-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    labels:
      dedicated: tiflash
    taints:
      dedicated: tiflash:NoSchedule

  - name: ticdc-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    labels:
      dedicated: ticdc
    taints:
      dedicated: ticdc:NoSchedule
  - name: ticdc-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: ticdc
    taints:
      dedicated: ticdc:NoSchedule
  - name: ticdc-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    labels:
      dedicated: ticdc
    taints:
      dedicated: ticdc:NoSchedule
```

Depending on the EKS cluster status, use different commands:

- If the cluster is not created, execute `eksctl create cluster -f cluster.yaml` to create the cluster and node groups.
- If the cluster is already created, execute `eksctl create nodegroup -f cluster.yaml` to create the node groups. The existing node groups are ignored and will not be created again.

### Configure and deploy

+ To deploy TiFlash, configure `spec.tiflash` in `tidb-cluster.yaml`:

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

    For other parameters, refer to [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

    > **Warning:**
    >
    > TiDB Operator automatically mount PVs **in the order of the configuration** in the `storageClaims` list. Therefore, if you need to add disks for TiFlash, make sure that you add the disks **only to the end of the original configuration** in the list. In addition, you must **not** alter the order of the original configuration.

+ To deploy TiCDC, configure `spec.ticdc` in `tidb-cluster.yaml`:

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
