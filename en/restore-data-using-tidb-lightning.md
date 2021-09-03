---
title: Import Data
summary: Learn how to quickly import data with TiDB Lightning.
aliases: ['/docs/tidb-in-kubernetes/dev/restore-data-using-tidb-lightning/']
---

# Import Data

This document describes how to import data into a TiDB cluster in Kubernetes using [TiDB Lightning](https://docs.pingcap.com/tidb/stable/tidb-lightning-overview).

TiDB Lightning contains two components: tidb-lightning and tikv-importer. In Kubernetes, the tikv-importer is inside the separate Helm chart of the TiDB cluster. And tikv-importer is deployed as a `StatefulSet` with `replicas=1` while tidb-lightning is in a separate Helm chart and deployed as a `Job`.

TiDB Lightning supports three backends: `Importer-backend`, `Local-backend`, and `TiDB-backend`. For the differences of these backends and how to choose backends, see [TiDB Lightning Backends](https://docs.pingcap.com/tidb/stable/tidb-lightning-backends).

- For `Importer-backend`, both tikv-importer and tidb-lightning need to be deployed.
- For `Local-backend`, only tidb-lightning needs to be deployed.
- For `TiDB-backend`, only tidb-lightning needs to be deployed, and it is recommended to import data using CustomResourceDefinition (CRD) in TiDB Operator v1.1 and later versions. For details, refer to [Restore Data from GCS Using TiDB Lightning](restore-from-gcs.md) or [Restore Data from S3-Compatible Storage Using TiDB Lightning](restore-from-s3.md)

## Deploy TiKV Importer

> **Note:**
>
> If you use the `local` or `tidb` backend for data restore, you can skip deploying tikv-importer and [deploy tidb-lightning](#deploy-tidb-lightning) directly.

You can deploy tikv-importer using the Helm chart. See the following example:

1. Make sure that the PingCAP Helm repository is up to date:

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm search repo tikv-importer -l
    ```

2. Get the default `values.yaml` file for easier customization:

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tikv-importer --version=${chart_version} > values.yaml
    ```

3. Modify the `values.yaml` file to specify the target TiDB cluster. See the following example:

    {{< copyable "" >}}

    ```yaml
    clusterName: demo
    image: pingcap/tidb-lightning:v5.2.0
    imagePullPolicy: IfNotPresent
    storageClassName: local-storage
    storage: 20Gi
    pushgatewayImage: prom/pushgateway:v0.3.1
    pushgatewayImagePullPolicy: IfNotPresent
    config: |
      log-level = "info"
      [metric]
      job = "tikv-importer"
      interval = "15s"
      address = "localhost:9091"
    ```

    `clusterName` must match the target TiDB cluster.

    If the target TiDB cluster has enabled TLS between components (`spec.tlsCluster.enabled: true`), refer to [Generate certificates for components of the TiDB cluster](enable-tls-between-components.md#generate-certificates-for-components-of-the-tidb-cluster) to genereate a server-side certificate for TiKV Importer, and configure `tlsCluster.enabled: true` in `values.yaml` to enable TLS.

4. Deploy tikv-importer:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${cluster_name} pingcap/tikv-importer --namespace=${namespace} --version=${chart_version} -f values.yaml
    ```

    > **Note:**
    >
    > You must deploy tikv-importer in the same namespace where the target TiDB cluster is deployed.

## Deploy TiDB Lightning

### Configure TiDB Lightning

Use the following command to get the default configuration of TiDB Lightning:

{{< copyable "shell-regular" >}}

```shell
helm inspect values pingcap/tidb-lightning --version=${chart_version} > tidb-lightning-values.yaml
```

Configure a `backend` used by TiDB Lightning depending on your needs. To do that, you can set the `backend` value in `values.yaml` to an option in `importer`, `local`, or `tidb`.

> **Note:**
>
> If you use the [`local` backend](https://docs.pingcap.com/tidb/stable/tidb-lightning-backends#tidb-lightning-local-backend), you must set `sortedKV` in `values.yaml` to create the corresponding PVC. The PVC is used for local KV sorting.

Starting from v1.1.10, the tidb-lightning Helm chart saves the [TiDB Lightning checkpoint information](https://docs.pingcap.com/tidb/stable/tidb-lightning-checkpoints) in the directory of the source data. When the a new tidb-lightning job is running, it can resume the data import according to the checkpoint information.

For versions earlier than v1.1.10, you can modify `config` in `values.yaml` to save the checkpoint information in the target TiDB cluster, other MySQL-compatible databases or a shared storage directory. For more information, refer to [TiDB Lightning checkpoint](https://docs.pingcap.com/tidb/stable/tidb-lightning-checkpoints).

If TLS between components has been enabled on the target TiDB cluster (`spec.tlsCluster.enabled: true`), refer to [Generate certificates for components of the TiDB cluster](enable-tls-between-components.md#generate-certificates-for-components-of-the-tidb-cluster) to genereate a server-side certificate for TiDB Lightning, and configure `tlsCluster.enabled: true` in `values.yaml` to enable TLS between components.

If the target TiDB cluster has enabled TLS for the MySQL client (`spec.tidb.tlsClient.enabled: true`), and the corresponding client-side certificate is configured (the Kubernetes Secret object is `${cluster_name}-tidb-client-secret`), you can configure `tlsClient.enabled: true` in `values.yaml` to enable TiDB Lightning to connect to the TiDB server using TLS.

To use different client certificates to connect to the TiDB server, refer to [Issue two sets of certificates for the TiDB cluster](enable-tls-for-mysql-client.md#issue-two-sets-of-certificates-for-the-tidb-cluster) to generate the client-side certificate for TiDB Lightning, and configure the corresponding Kubernetes secret object in `tlsCluster.tlsClientSecretName` in `values.yaml`.

> **Note:**
>
> If TLS is enabled between components via `tlsCluster.enabled: true` but not enabled between TiDB Lightning and the TiDB server via `tlsClient.enabled: true`, you need to explicitly disable TLS between TiDB Lightning and the TiDB server in `config` in `values.yaml`:
>
> ```toml
> [tidb]
> tls="false"
> ```

TiDB Lightning Helm chart supports both local and remote data sources.

#### Local

In the local mode, the backup data must be on one of the Kubernetes node. To enable this mode, set `dataSource.local.nodeName` to the node name and `dataSource.local.hostPath` to the path of the backup data. The path should contain a file named `metadata`.

#### Remote

Unlike the local mode, the remote mode needs to use [rclone](https://rclone.org) to download the backup tarball file from a network storage to a PV. Any cloud storage supported by rclone should work, but currently only the following have been tested: [Google Cloud Storage (GCS)](https://cloud.google.com/storage/), [Amazon S3](https://aws.amazon.com/s3/), [Ceph Object Storage](https://ceph.com/ceph-storage/object-storage/).

To restore backup data from the remote source, take the following steps:

1. Make sure that `dataSource.local.nodeName` and `dataSource.local.hostPath` in `values.yaml` are commented out.

2. Grant permissions to the remote storage

    If you use Amazon S3 as the storage, refer to [AWS account Permissions](grant-permissions-to-remote-storage.md#aws-account-permissions). The configuration varies with different methods.

    If you use Ceph as the storage, you can only grant permissions by importing AccessKey and SecretKey. See [Grant permissions by AccessKey and SecretKey](grant-permissions-to-remote-storage.md#grant-permissions-by-accesskey-and-secretkey).

    If you use GCS as the storage, refer to [GCS account permissions](grant-permissions-to-remote-storage.md#gcs-account-permissions).

    * Grant permissions by importing AccessKey and SecretKey

        1. Create a `Secret` configuration file `secret.yaml` containing the rclone configuration. A sample configuration is listed below. Only one cloud storage configuration is required.

            {{< copyable "" >}}

            ```yaml
            apiVersion: v1
            kind: Secret
            metadata:
              name: cloud-storage-secret
            type: Opaque
            stringData:
              rclone.conf: |
                [s3]
                type = s3
                provider = AWS
                env_auth = false
                access_key_id = ${access_key}
                secret_access_key = ${secret_key}
                region = us-east-1

                [ceph]
                type = s3
                provider = Ceph
                env_auth = false
                access_key_id = ${access_key}
                secret_access_key = ${secret_key}
                endpoint = ${endpoint}
                region = :default-placement

                [gcs]
                type = google cloud storage
                # The service account must include Storage Object Viewer role
                # The content can be retrieved by `cat ${service-account-file} | jq -c .`
                service_account_credentials = ${service_account_json_file_content}
            ```

        2. Execute the following command to create `Secret`:

            {{< copyable "shell-regular" >}}

            ```shell
            kubectl apply -f secret.yaml -n ${namespace}
            ```

    * Grant permissions by associating IAM with Pod or with ServiceAccount

       If you use Amazon S3 as the storage, you can grant permissions by associating IAM with Pod or with ServiceAccount, in which `s3.access_key_id` and `s3.secret_access_key` can be ignored.

        1. Save the following configurations as `secret.yaml`.

            {{< copyable "" >}}

            ```yaml
            apiVersion: v1
            kind: Secret
            metadata:
              name: cloud-storage-secret
            type: Opaque
            stringData:
              rclone.conf: |
                [s3]
                type = s3
                provider = AWS
                env_auth = true
                access_key_id =
                secret_access_key =
                region = us-east-1
            ```

        2. Execute the following command to create `Secret`:

            {{< copyable "shell-regular" >}}

            ```shell
            kubectl apply -f secret.yaml -n ${namespace}
            ```

3. Configure the `dataSource.remote.storageClassName` to an existing storage class in the Kubernetes cluster.

#### Ad hoc

When restoring data from remote storage, sometimes the restore process is interrupted due to the exception. In such cases, if you do not want to download backup data from the network storage repeatedly, you can use the ad hoc mode to directly recover the data that has been downloaded and decompressed into PV in the remote mode. The steps are as follows:

1. Ensure `dataSource.local` and `dataSource.remote` in the config file `values.yaml` are empty。

2. Configure `dataSource.adhoc.pvcName` in `values.yaml` to the PVC name used in restoring data from remote storage.

3. Configure `dataSource.adhoc.backupName` in `values.yaml` to the name of the original backup data, such as: `backup-2020-12-17T10:12:51Z` (Do not contain the '. tgz' suffix of the compressed file name on network storage).

### Deploy TiDB Lightning

The method of deploying TiDB Lightning varies with different methods of granting permissions and with different storages.

* For [Local Mode](#local), [Ad hoc Mode](#ad-hoc), and [Remote Mode](#remote) (only for remote modes that meet one of the three requirements: using Amazon S3 AccessKey and SecretKey permission granting methods, using Ceph as the storage backend, or using GCS as the storage backend), run the following command to deploy TiDB Lightning.

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${release_name} pingcap/tidb-lightning --namespace=${namespace} --set failFast=true -f tidb-lightning-values.yaml --version=${chart_version}
    ```

* For [Remote Mode](#remote), if you grant permissions by associating Amazon S3 IAM with Pod, take the following steps:

    1. Create the IAM role:

        [Create an IAM role](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_create.html) for the account, and [grant the required permission](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_manage-attach-detach.html) to the role. The IAM role requires the `AmazonS3FullAccess` permission because TiDB Lightning needs to access Amazon S3 storage.

    2. Modify `tidb-lightning-values.yaml`, and add the `iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user` annotation in the `annotations` field.

    3. Deploy TiDB Lightning:

        {{< copyable "shell-regular" >}}

        ```shell
        helm install ${release_name} pingcap/tidb-lightning --namespace=${namespace} --set failFast=true -f tidb-lightning-values.yaml --version=${chart_version}
        ```

        > **Note:**
        >
        > `arn:aws:iam::123456789012:role/user` is the IAM role created in Step 1.

* For [Remote Mode](#remote), if you grant permissions by associating Amazon S3 with ServiceAccount, take the following steps:

    1. Enable the IAM role for the service account on the cluster:

        To enable the IAM role permission on the EKS cluster, see [AWS Documentation](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html).

    2. Create the IAM role:

        [Create an IAM role](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html). Grant the `AmazonS3FullAccess` permission to the role, and edit `Trust relationships` of the role.

    3. Associate IAM with the ServiceAccount resources:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl annotate sa ${servieaccount} -n eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/user
        ```

    4. Deploy TiDB Lightning:

        {{< copyable "shell-regular" >}}

        ```shell
        helm install ${release_name} pingcap/tidb-lightning --namespace=${namespace} --set-string failFast=true,serviceAccount=${servieaccount} -f tidb-lightning-values.yaml --version=${chart_version}
        ```

        > **Note:**
        >
        > `arn:aws:iam::123456789012:role/user` is the IAM role created in Step 1.
        > `${service-account}` is the ServiceAccount used by TiDB Lightning. The default value is `default`.

## Destroy TiKV Importer and TiDB Lightning

Currently, TiDB Lightning only supports restoring data offline. After the restore, if the TiDB cluster needs to provide service for external applications, you can destroy TiDB Lightning to save cost.

To destroy tikv-importer, execute the following command:

{{< copyable "shell-regular" >}}

```shell
helm uninstall ${release_name} -n ${namespace}
```

To destroy tidb-lightning, execute the following command:

{{< copyable "shell-regular" >}}

```shell
helm uninstall ${release_name} -n ${namespace}
```

## Troubleshoot TiDB Lightning

When TiDB Lightning fails to restore data, you cannot simply restart it. **Manual intervention** is required. Therefore, the TiDB Lightning's `Job` restart policy is set to `Never`.

> **Note:**
>
> If you have not configured to persist the checkpoint information in the target TiDB cluster, other MySQL-compatible databases or a shared storage directory, after the restore failure, you need to first delete the part of data already restored to the target cluster. After that, deploy tidb-lightning again and retry the data restore.

If TiDB Lightning fails to restore data, and if you have configured to persist the checkpoint information in the target TiDB cluster, other MySQL-compatible databases or a shared storage directory, follow the steps below to do manual intervention:

1. View the log by executing the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl logs -n ${namespace} ${pod_name}
    ```

    * If you restore data using the remote data source, and the error occurs when TiDB Lightning downloads data from remote storage:

        1. Address the problem according to the log.
        2. Deploy tidb-lightning again and retry the data restore.

    * For other cases, refer to the following steps.

2. Refer to [TiDB Lightning Troubleshooting](https://docs.pingcap.com/tidb/stable/troubleshoot-tidb-lightning) and learn the solutions to different issues.

3. Address the issues accordingly:

    - If `tidb-lightning-ctl` is required:

        1. Configure `dataSource` in `values.yaml`. Make sure the new `Job` uses the data source and checkpoint information of the failed `Job`:

            - In the local or ad hoc mode, you do not need to modify `dataSource`.
            - In the remote mode, modify `dataSource` to the ad hoc mode. `dataSource.adhoc.pvcName` is the PVC name created by the original Helm chart. `dataSource.adhoc.backupName` is the backup name of the data to be restored.

        2. Modify `failFast` in `values.yaml` to `false`, and create a `Job` used for `tidb-lightning-ctl`.

            - Based on the checkpoint information, TiDB Lightning checks whether the last data restore encountered an error. If yes, TiDB Lightning pauses the restore automatically.
            - TiDB Lightning uses the checkpoint information to avoid repeatedly restoring the same data. Therefore, creating the `Job` does not affect data correctness.

        3. After the Pod corresponding to the new `Job` is running, view the log by running `kubectl logs -n ${namespace} ${pod_name}` and confirm tidb-lightning in the new `Job` already stops data restore. If the log has the following message, the data restore is stopped:

            - `tidb lightning encountered error`
            - `tidb lightning exit`

        4. Enter the container by running `kubectl exec -it -n ${namespace} ${pod_name} -it -- sh`.

        5. Obtain the starting script by running `cat /proc/1/cmdline`.

        6. Get the command-line parameters from the starting script. Refer to [TiDB Lightning Troubleshooting](https://docs.pingcap.com/tidb/stable/troubleshoot-tidb-lightning) and troubleshoot using `tidb-lightning-ctl`.

        7. After the troubleshooting, modify `failFast` in `values.yaml` to `true` and create a new `Job` to resume data restore.

    - If `tidb-lightning-ctl` is not required:

        1. [Troubleshoot TiDB Lightning](https://docs.pingcap.com/tidb/stable/troubleshoot-tidb-lightning).

        2. Configure `dataSource` in `values.yaml`. Make sure the new `Job` uses the data source and checkpoint information of the failed `Job`:

            - In the local or ad hoc mode, you do not need to modify `dataSource`.
            - In the remote mode, modify `dataSource` to the ad hoc mode. `dataSource.adhoc.pvcName` is the PVC name created by the original Helm chart. `dataSource.adhoc.backupName` is the backup name of the data to be restored.

        3. Create a new `Job` using the modified `values.yaml` file and resume data restore.

4. After the troubleshooting and data restore is completed, [delete the `Job`s](#destroy-tikv-importer-and-tidb-lightning) for data restore and troubleshooting.
