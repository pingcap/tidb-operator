---
title: Deploy TiDB on Alibaba Cloud Kubernetes
summary: Learn how to deploy a TiDB cluster on Alibaba Cloud Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-on-alibaba-cloud/']
---

# Deploy TiDB on Alibaba Cloud Kubernetes

This document describes how to deploy a TiDB cluster on Alibaba Cloud Kubernetes with your laptop (Linux or macOS) for development or testing.

To deploy TiDB Operator and the TiDB cluster in a self-managed Kubernetes environment, refer to [Deploy TiDB Operator](deploy-tidb-operator.md) and [Deploy TiDB in General Kubernetes](deploy-on-general-kubernetes.md).

## Prerequisites

- [aliyun-cli](https://github.com/aliyun/aliyun-cli) >= 3.0.15 and [configure aliyun-cli](https://www.alibabacloud.com/help/doc-detail/90766.htm?spm=a2c63.l28256.a3.4.7b52a893EFVglq)

    > **Note:**
    >
    > The access key must be granted permissions to control the corresponding resources.

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) >= 1.12
- [Helm 3](https://helm.sh)
- [jq](https://stedolan.github.io/jq/download/) >= 1.6
- [terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) 0.12.*

You can use [Cloud Shell](https://shell.aliyun.com) of Alibaba Cloud to perform operations. All the tools have been pre-installed and configured in the Cloud Shell of Alibaba Cloud.

### Required privileges

To deploy a TiDB cluster, make sure you have the following privileges:

- AliyunECSFullAccess
- AliyunESSFullAccess
- AliyunVPCFullAccess
- AliyunSLBFullAccess
- AliyunCSFullAccess
- AliyunEIPFullAccess
- AliyunECIFullAccess
- AliyunVPNGatewayFullAccess
- AliyunNATGatewayFullAccess

## Overview of things to create

In the default configuration, you will create:

- A new VPC
- An ECS instance as the bastion machine
- A managed ACK (Alibaba Cloud Kubernetes) cluster with the following ECS instance worker nodes:

    - An auto-scaling group of 2 * instances (2 cores, 2 GB RAM). The default auto-scaling group of managed Kubernetes must have at least two instances to host the whole system service, like CoreDNS
    - An auto-scaling group of 3 * `ecs.g5.large` instances for deploying the PD cluster
    - An auto-scaling group of 3 * `ecs.i2.2xlarge` instances for deploying the TiKV cluster
    - An auto-scaling group of 2 * `ecs.c5.4xlarge` instances for deploying the TiDB cluster
    - An auto-scaling group of 1 * `ecs.c5.xlarge` instance for deploying monitoring components
    - A 100 GB cloud disk used to store monitoring data

All the instances except ACK mandatory workers are deployed across availability zones (AZs) to provide cross-AZ high availability. The auto-scaling group ensures the desired number of healthy instances, so the cluster can auto-recover from node failure or even AZ failure.

## Deploy

### Deploy ACK, TiDB Operator and the node pool for TiDB cluster

1. Configure the target region and Alibaba Cloud key (you can also set these variables in the `terraform` command prompt):

    {{< copyable "shell-regular" >}}

    ```shell
    export TF_VAR_ALICLOUD_REGION=${REGION} && \
    export TF_VAR_ALICLOUD_ACCESS_KEY=${ACCESS_KEY} && \
    export TF_VAR_ALICLOUD_SECRET_KEY=${SECRET_KEY}
    ```

    The `variables.tf` file contains default settings of variables used for deploying the cluster. You can change it or use the `-var` option to override a specific variable to fit your need.

2. Use Terraform to set up the cluster.

    {{< copyable "shell-regular" >}}

    ```shell
    git clone --depth=1 https://github.com/pingcap/tidb-operator && \
    cd tidb-operator/deploy/aliyun
    ```

    You can create or modify `terraform.tfvars` to set the values of the variables, and configure the cluster to fit your needs. You can view the configurable variables and their descriptions in `variables.tf`. The following is an example of how to configure the ACK cluster name, the TiDB cluster name, the TiDB Operator version, and the number of PD, TiKV, and TiDB nodes.

    ```
    cluster_name = "testack"
    tidb_cluster_name = "testdb"
    tikv_count = 3
    tidb_count = 2
    pd_count = 3
    operator_version = "v1.2.4"
    ```

    * To deploy TiFlash in the cluster, set `create_tiflash_node_pool = true` in `terraform.tfvars`. You can also configure the node count and instance type of the TiFlash node pool by modifying `tiflash_count` and `tiflash_instance_type`. By default, the value of `tiflash_count` is `2`, and the value of `tiflash_instance_type` is `ecs.i2.2xlarge`.

    * To deploy TiCDC in the cluster, set `create_cdc_node_pool = true` in `terraform.tfvars`. You can also configure the node count and instance type of the TiCDC node pool by modifying `cdc_count` and `cdc_instance_type`. By default, the value of `cdc_count` is `3`, and the value of `cdc_instance_type` is `ecs.c5.2xlarge`.

    > **Note:**
    >
    > Check the `operator_version` in the `variables.tf` file for the default TiDB Operator version of the current scripts. If the default version is not your desired one, configure `operator_version` in `terraform.tfvars`.

    After the configuration, execute the following commands to initialize and deploy the cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    terraform init
    ```

    Input "yes" to confirm execution when you run the following `apply` command:

    {{< copyable "shell-regular" >}}

    ```shell
    terraform apply
    ```

    If you get an error while running `terraform apply`, fix the error (for example, lack of permission) according to the error description and run `terraform apply` again.

    It takes 5 to 10 minutes to create the whole stack using `terraform apply`. Once the installation is complete, the basic cluster information is printed:

    ```
    Apply complete! Resources: 3 added, 0 changed, 1 destroyed.

    Outputs:

    bastion_ip = 47.96.174.214
    cluster_id = c2d9b20854a194f158ef2bc8ea946f20e

    kubeconfig_file = /tidb-operator/deploy/aliyun/credentials/kubeconfig
    monitor_endpoint = not_created
    region = cn-hangzhou
    ssh_key_file = /tidb-operator/deploy/aliyun/credentials/my-cluster-keyZ.pem
    tidb_endpoint = not_created
    tidb_version = v3.0.0
    vpc_id = vpc-bp1v8i5rwsc7yh8dwyep5
    ```

    > **Note:**
    >
    > You can use the `terraform output` command to get the output again.

3. You can then interact with the ACK cluster using `kubectl` or `helm`:

    {{< copyable "shell-regular" >}}

    ```shell
    export KUBECONFIG=$PWD/credentials/kubeconfig
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl version
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm ls
    ```

### Deploy the TiDB cluster and monitor

1. Prepare the `TidbCluster` and `TidbMonitor` CR files:

    {{< copyable "shell-regular" >}}

    ```shell
    cp manifests/db.yaml.example db.yaml && cp manifests/db-monitor.yaml.example db-monitor.yaml
    ```

    To complete the CR file configuration, refer to [TiDB Operator API documentation](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md) and [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

    * To deploy TiFlash, configure `spec.tiflash` in `db.yaml` as follows:

        ```yaml
        spec
          ...
          tiflash:
            baseImage: pingcap/tiflash
            maxFailoverCount: 3
            nodeSelector:
              dedicated: TIDB_CLUSTER_NAME-tiflash
            replicas: 1
            storageClaims:
            - resources:
                requests:
                  storage: 100Gi
              storageClassName: local-volume
            tolerations:
            - effect: NoSchedule
              key: dedicated
              operator: Equal
              value: TIDB_CLUSTER_NAME-tiflash
        ```

        To configure other parameters, refer to [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

        Modify `replicas`, `storageClaims[].resources.requests.storage`, and `storageClassName` according to your needs.

        > **Warning:**
        >
        > Since TiDB Operator will mount PVs automatically in the **order** of the items in the `storageClaims` list, if you need to add more disks to TiFlash, make sure to append the new item only to the **end** of the original items, and **DO NOT** modify the order of the original items.

    * To deploy TiCDC, configure `spec.ticdc` in `db.yaml` as follows:

        ```yaml
        spec
          ...
          ticdc:
            baseImage: pingcap/ticdc
            nodeSelector:
              dedicated: TIDB_CLUSTER_NAME-cdc
            replicas: 3
            tolerations:
            - effect: NoSchedule
              key: dedicated
              operator: Equal
              value: TIDB_CLUSTER_NAME-cdc
        ```

        Modify `replicas` according to your needs.

    To deploy Enterprise Edition of TiDB/PD/TiKV/TiFlash/TiCDC, edit the `db.yaml` file to set `spec.<tidb/pd/tikv/tiflash/ticdc>.baseImage` to the enterprise image (`pingcap/<tidb/pd/tikv/tiflash/ticdc>-enterprise`).

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

    > **Note:**
    >
    > * Replace all the `TIDB_CLUSTER_NAME` in the `db.yaml` and `db-monitor.yaml` files with `tidb_cluster_name` configured in the deployment of ACK.
    > * Make sure the number of PD, TiKV, TiFlash, TiCDC, or TiDB nodes is >= the `replicas` value of the corresponding component in `db.yaml`.
    > * Make sure `spec.initializer.version` in `db-monitor.yaml` is the same as `spec.version` in `db.yaml`. Otherwise, the monitor might not display correctly.

2. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig create namespace ${namespace}
    ```

    > **Note:**
    >
    > You can give the [`namespace`](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) a name that is easy to memorize, such as the same name as `tidb_cluster_name`.

3. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl --kubeconfig credentials/kubeconfig create -f db.yaml -n ${namespace} &&
    kubectl --kubeconfig credentials/kubeconfig create -f db-monitor.yaml -n ${namespace}
    ```

> **Note:**
>
> If you need to deploy a TiDB cluster on ARM64 machines, refer to [Deploy a TiDB Cluster on ARM64 Machines](deploy-cluster-on-arm64.md).

## Access the database

You can connect the TiDB cluster via the bastion instance. All necessary information is in the output printed after installation is finished (replace the `${}` parts with values from the output):

{{< copyable "shell-regular" >}}

```shell
ssh -i credentials/${cluster_name}-key.pem root@${bastion_ip}
```

{{< copyable "shell-regular" >}}

```shell
mysql -h ${tidb_lb_ip} -P 4000 -u root
```

`tidb_lb_ip` is the LoadBalancer IP of the TiDB service.

> **Note:**
>
> * [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you use MySQL client from MySQL 8.0 to access the TiDB service (TiDB version < v4.0.7), and if the user account has a password, you need to explicitly specify the `--default-auth=mysql_native_password` parameter.
> * By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).

## Monitor

Visit `<monitor-lb>:3000` to view the Grafana dashboards. `monitor-lb` is the LoadBalancer IP of the Monitor service.

The initial login user account and password:

- User: admin
- Password: admin

> **Warning:**
>
> If you already have a VPN connecting to your VPC or plan to set up one, it is strongly recommended that you go to the `spec.grafana.service.annotations` section in the `db-monitor.yaml` file and set `service.beta.kubernetes.io/alicloud-loadbalancer-address-type` to `intranet` for security.

## Upgrade

To upgrade the TiDB cluster, modify the `spec.version` variable by executing `kubectl --kubeconfig credentials/kubeconfig edit tc ${tidb_cluster_name} -n ${namespace}`.

This may take a while to complete. You can watch the process using the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get pods --namespace ${namespace} -o wide --watch
```

## Scale out the TiDB cluster

To scale out the TiDB cluster, modify `tikv_count`, `tiflash_count`, `cdc_count`, or `tidb_count` in the `terraform.tfvars` file, and then run `terraform apply` to scale out the number of nodes for the corresponding components.

After the nodes scale out, modify the `replicas` of the corresponding components by running `kubectl --kubeconfig credentials/kubeconfig edit tc ${tidb_cluster_name} -n ${namespace}`.

> **Note:**
>
> - Because it is impossible to determine which node will be taken offline during the scale-in process, the scale-in of TiDB clusters is currently not supported.
> - The scale-out process takes a few minutes. You can watch the status by running `kubectl --kubeconfig credentials/kubeconfig get po -n ${namespace} --watch`.

## Configure

### Configure TiDB Operator

You can set the variables in `terraform.tfvars` to configure TiDB Operator. Most configuration items can be modified after you understand the semantics based on the comments of the `variable`. Note that the `operator_helm_values` configuration item can provide a customized `values.yaml` configuration file for TiDB Operator. For example:

- Set `operator_helm_values` in `terraform.tfvars`:

    ```hcl
    operator_helm_values = "./my-operator-values.yaml"
    ```

- Set `operator_helm_values` in `main.tf`:

    ```hcl
    operator_helm_values = file("./my-operator-values.yaml")
    ```

In the default configuration, the Terraform script creates a new VPC. To use the existing VPC, set `vpc_id` in `variable.tf`. In this case, Kubernetes nodes are not deployed in AZs with vswitch not configured.

### Configure the TiDB cluster

See [TiDB Operator API Documentation](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md) and [Configure a TiDB Cluster](configure-a-tidb-cluster.md).

## Manage multiple TiDB clusters

To manage multiple TiDB clusters in a single Kubernetes cluster, you need to edit `./main.tf` and add the `tidb-cluster` declaration based on your needs. For example:

```hcl
module "tidb-cluster-dev" {
  source = "../modules/aliyun/tidb-cluster"
  providers = {
    helm = helm.default
  }

  cluster_name = "dev-cluster"
  ack          = module.tidb-operator

  pd_count                   = 1
  tikv_count                 = 1
  tidb_count                 = 1
}

module "tidb-cluster-staging" {
  source = "../modules/aliyun/tidb-cluster"
  providers = {
    helm = helm.default
  }

  cluster_name = "staging-cluster"
  ack          = module.tidb-operator

  pd_count                   = 3
  tikv_count                 = 3
  tidb_count                 = 2
}
```

> **Note:**
>
> You need to set a unique `cluster_name` for each TiDB cluster.

All the configurable parameters in `tidb-cluster` are as follows:

| Parameter | Description | Default value |
| :----- | :---- | :----- |
| `ack` | The structure that enwraps the target Kubernetes cluster information (required) | `nil` |
| `cluster_name` | The TiDB cluster name (required and unique) | `nil` |
| `tidb_version` | The TiDB cluster version | `v3.0.1` |
| `tidb_cluster_chart_version` | `tidb-cluster` helm chart version | `v1.0.1` |
| `pd_count` | The number of PD nodes | 3 |
| `pd_instance_type` | The PD instance type | `ecs.g5.large` |
| `tikv_count` | The number of TiKV nodes | 3 |
| `tikv_instance_type` | The TiKV instance type | `ecs.i2.2xlarge` |
| `tiflash_count` | The count of TiFlash nodes | 2 |
| `tiflash_instance_type` | The TiFlash instance type | `ecs.i2.2xlarge` |
| `cdc_count` | The count of TiCDC nodes | 3 |
| `cdc_instance_type` | The TiCDC instance type | `ecs.c5.2xlarge` |
| `tidb_count` | The number of TiDB nodes | 2 |
| `tidb_instance_type` | The TiDB instance type | `ecs.c5.4xlarge` |
| `monitor_instance_type` | The instance type of monitoring components | `ecs.c5.xlarge` |
| `override_values` | The `values.yaml` configuration file of the TiDB cluster. You can read it using the `file()` function | `nil` |
| `local_exec_interpreter` | The interpreter that executes the command line instruction | `["/bin/sh", "-c"]` |
| `create_tidb_cluster_release` | Whether to create the TiDB cluster using Helm | `false` |

## Manage multiple Kubernetes clusters

It is recommended to use a separate Terraform module to manage a specific Kubernetes cluster. (A Terraform module is a directory that contains the `.tf` script.)

`deploy/aliyun` combines multiple reusable Terraform scripts in `deploy/modules`. To manage multiple clusters, perform the following operations in the root directory of the `tidb-operator` project:

1. Create a directory for each cluster. For example:

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p deploy/aliyun-staging
    ```

2. Refer to `main.tf` in `deploy/aliyun` and write your own script. For example:

    ```hcl
    provider "alicloud" {
        region     = ${REGION}
        access_key = ${ACCESS_KEY}
        secret_key = ${SECRET_KEY}
    }

    module "tidb-operator" {
        source     = "../modules/aliyun/tidb-operator"

        region          = ${REGION}
        access_key      = ${ACCESS_KEY}
        secret_key      = ${SECRET_KEY}
        cluster_name    = "example-cluster"
        key_file        = "ssh-key.pem"
        kubeconfig_file = "kubeconfig"
    }

    provider "helm" {
        alias    = "default"
        insecure = true
        install_tiller = false
        kubernetes {
            config_path = module.tidb-operator.kubeconfig_filename
        }
    }

    module "tidb-cluster" {
        source = "../modules/aliyun/tidb-cluster"
        providers = {
            helm = helm.default
        }

        cluster_name = "example-cluster"
        ack          = module.tidb-operator
    }

    module "bastion" {
        source = "../modules/aliyun/bastion"

        bastion_name             = "example-bastion"
        key_name                 = module.tidb-operator.key_name
        vpc_id                   = module.tidb-operator.vpc_id
        vswitch_id               = module.tidb-operator.vswitch_ids[0]
        enable_ssh_to_worker     = true
        worker_security_group_id = module.tidb-operator.security_group_id
    }
    ```

You can customize this script. For example, you can remove the `module "bastion"` declaration if you do not need the bastion machine.

> **Note:**
>
> You can copy the `deploy/aliyun` directory. But you cannot copy a directory on which the `terraform apply` operation is currently performed. In this case, it is recommended to clone the repository again and then copy it.

## Destroy

1. Refer to [Destroy a TiDB cluster](destroy-a-tidb-cluster.md) to delete the cluster.

2. Destroy the ACK cluster by running the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    terraform destroy
    ```

If the Kubernetes cluster is not successfully created, the `destroy` operation might return an error and fail. In such cases, manually remove the Kubernetes resources from the local state:

{{< copyable "shell-regular" >}}

```shell
terraform state list
```

{{< copyable "shell-regular" >}}

```shell
terraform state rm module.ack.alicloud_cs_managed_kubernetes.k8s
```

It may take a long time to finish destroying the cluster.

> **Note:**
>
> You have to manually delete the cloud disk used by the components in the Alibaba Cloud console.

## Limitation

You cannot change `pod cidr`, `service cidr`, and worker instance types once the cluster is created.
