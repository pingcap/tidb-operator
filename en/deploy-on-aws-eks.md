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
> The operations described in this document requires at least the [minimum privileges needed by `eksctl`](https://eksctl.io/usage/minimum-iam-policies/) and the [service privileges needed to create a Linux bastion host](https://docs.aws.amazon.com/quickstart/latest/linux-bastion/architecture.html#aws-services).

## Create a EKS cluster and a node pool

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
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule
  - name: tidb-1d
    desiredCapacity: 0
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule
  - name: tidb-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    labels:
      dedicated: tidb
    taints:
      dedicated: tidb:NoSchedule

  - name: pd-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule
  - name: pd-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule
  - name: pd-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
    labels:
      dedicated: pd
    taints:
      dedicated: pd:NoSchedule

  - name: tikv-1a
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1a"]
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
  - name: tikv-1d
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1d"]
    labels:
      dedicated: tikv
    taints:
      dedicated: tikv:NoSchedule
  - name: tikv-1c
    desiredCapacity: 1
    privateNetworking: true
    availabilityZones: ["ap-northeast-1c"]
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
> A [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is a virtual cluster backed by the same physical cluster. This document takes `tidb-cluster` as an example. If you want to use other namespace, modify the corresponding arguments of `-n` or `--namespace`.

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

After the yaml file above is applied to the Kubernetes cluster, TiDB Operator creates the desired TiDB cluster and its monitoring component according to the yaml file.

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
    mysql -h ${tidb-nlb-dnsname} -P 4000 -u root
    ```

    `${tidb-nlb-dnsname}` is the LoadBalancer domain name of the TiDB service. You can view the domain name in the `EXTERNAL-IP` field by executing `kubectl get svc basic-tidb -n tidb-cluster`.

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

## Use other EBS volume types

AWS EBS supports multiple volume types. If you need low latency and high throughput, you can choose the `io1` type. The steps are as follows:

1. Create a storage class for `io1`:

    ```yaml
    kind: StorageClass
    apiVersion: storage.k8s.io/v1
    metadata:
      name: io1
    provisioner: kubernetes.io/aws-ebs
    parameters:
      type: io1
      fsType: ext4
      iopsPerGB: "10"
      encrypted: "false"
    ```

2. In `tidb-cluster.yaml`, specify the `io1` storage class to apply for the `io1` volume type through the `storageClassName` field.

    The following is a TiKV configuration example you can refer to:

    ```yaml
    spec:
      tikv:
        baseImage: pingcap/tikv
        replicas: 3
        storageClaims:
        - resources:
          requests:
            storage: 100Gi
          storageClassName: io1
    ```

AWS already supports [EBS gp3](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html#gp3-ebs-volume-type), so it is recommended to use EBS gp3 volume type. However, EKS does not support provisioning the EBS gp3 StorageClass by default. For details, refer to the [issue](https://github.com/aws/containers-roadmap/issues/1187). If you use [Amazon Elastic Block Store (EBS) CSI driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver) [v0.8.0](https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/master/CHANGELOG-0.x.md#v080) or later versions, gp3 is already the default volume type.

For more information about the storage class configuration and EBS volume types, refer to [Storage Class documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/) and [EBS Volume Types](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html).

## Use local storage

Use AWS EBS as a primary production configuration. To simulate bare metal performance, some AWS instance types provide additional [NVMe SSD local store volumes](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ssd-instance-store.html). You can choose such instances for the TiKV node pool to achieve higher IOPS and lower latency.

> **Note:**
>
> You cannot dynamically change the storage class of a running TiDB cluster. You can create a new cluster for testing.
>
> During the EKS upgrade, [data in the local storage will be lost](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html#instance-store-lifetime) due to the node reconstruction. When the node reconstruction occurs, you need to migrate data in TiKV. If you do not want to migrate data, it is recommended not to use the local disk in the production environment.
>
> As the node reconstruction will cause the data loss of local storage, refer to [AWS document](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-suspend-resume-processes.html) to suspend the `ReplaceUnhealthy` process for the TiKV node group.

For instance types that provide local volumes, see [AWS Instance Types](https://aws.amazon.com/ec2/instance-types/). Take `c5d.4xlarge` as an example:

1. Create a node group with local storage for TiKV.

    Modify the instance type of the TiKV node group in the `eksctl` configuration file to `c5d.4xlarge`:

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

    For more information, refer to [Deploy TiDB cluster and its monitoring components](#deploy)
