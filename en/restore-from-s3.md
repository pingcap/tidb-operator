---
title: Restore Data from S3-Compatible Storage Using TiDB Lightning
summary: Learn how to restore data from the S3-compatible storage.
category: how-to
aliases: ['/docs/tidb-in-kubernetes/dev/restore-from-s3/']
---

# Restore Data from S3-Compatible Storage Using TiDB Lightning

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. For the underlying implementation, [`Lightning`](https://pingcap.com/docs/stable/how-to/get-started/tidb-lightning/#tidb-lightning-tutorial) is used to perform the restoration.

The restoration method described in this document is implemented based on CustomResourceDefinition (CRD) in TiDB Operator v1.1 or later versions. For the restoration method implemented based on Helm Charts, refer to [Back up and Restore TiDB Cluster Data Based on Helm Charts](backup-and-restore-using-helm-charts.md).

This document shows an example in which the backup data stored in the specified path on the S3-compatible storage is restored to the TiDB cluster.

## Three methods to grant AWS account permissions

- If you use Amazon S3 to back up and restore the cluster, you have three methods to grant permissions. For details, refer to [Back up TiDB Cluster Data to AWS Using BR](backup-to-aws-s3-using-br.md#three-methods-to-grant-aws-account-permissions).
- If Ceph is used as backend storage in backup and restore test, the permission is granted by importing AccessKey and SecretKey.

## Prerequisites

Refer to [Prerequisites](restore-from-aws-s3-using-br.md#prerequisites).

## Restoration process

> **Note:**
>
> Because of the `rclone` [issue](https://rclone.org/s3/#key-management-system-kms), if the backup data is stored in Amazon S3 and the `AWS-KMS` encryption is enabled, you need to add the following `spec.s3.options` configuration to the YAML file in the examples of this section:
>
> ```yaml
> spec:
>   ...
>   s3:
>     ...
>     options:
>     - --ignore-checksum
> ```

**Examples:**

+ Create the `Restore` CR, and restore the cluster data from Ceph by importing AccessKey and SecretKey to grant permissions:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore.yaml
    ```

    The content of `restore.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore
      namespace: test2
    spec:
      backupType: full
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: ceph
        endpoint: ${endpoint}
        secretName: s3-secret
        path: s3://${backup_path}
      storageClassName: local-storage
      storageSize: 1Gi
    ```

+ Create the `Restore` CR, and restore the cluster data from Amazon S3 by importing AccessKey and SecretKey to grant permissions:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore.yaml
    ```

    The `restore.yaml` file has the following content:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore
      namespace: test2
    spec:
      backupType: full
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: ${region}
        secretName: s3-secret
        path: s3://${backup_path}
      storageClassName: local-storage
      storageSize: 1Gi
    ```

+ Create the `Restore` CR, and restore the cluster data from Amazon S3 by binding IAM with Pod to grant permissions:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore.yaml
    ```

    The content of `restore.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore
      namespace: test2
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
      spec:
        backupType: full
        to:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: restore-demo2-tidb-secret
        s3:
          provider: aws
          region: ${region}
          path: s3://${backup_path}
        storageClassName: local-storage
        storageSize: 1Gi
    ```

+ Create the `Restore` CR, and restore the cluster data from Amazon S3 by binding IAM with ServiceAccount to grant permissions:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore.yaml
    ```

    The content of `restore.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore
      namespace: test2
      spec:
        backupType: full
        serviceAccount: tidb-backup-manager
        to:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: restore-demo2-tidb-secret
        s3:
          provider: aws
          region: ${region}
          path: s3://${backup_path}
        storageClassName: local-storage
        storageSize: 1Gi
    ```

After creating the `Restore` CR, execute the following command to check the restoration status:

{{< copyable "shell-regular" >}}

```shell
kubectl get rt -n test2 -owide
```

In the examples above, the backup data stored in the `spec.s3.path` path on the S3-compatible storage is restored to the `spec.to.host` TiDB cluster. For the configuration of the S3-compatible storage, refer to [backup-s3.yaml](backup-to-s3.md#ad-hoc-backup-process).

More `Restore` CRs are described as follows:

* `.spec.metadata.namespace`: the namespace where the `Restore` CR is located.
* `.spec.to.host`: the address of the TiDB cluster to be restored.
* `.spec.to.port`: the port of the TiDB cluster to be restored.
* `.spec.to.user`: the accessing user of the TiDB cluster to be restored.
* `.spec.to.secretName`: the secret contains the password of the `.spec.to.user`.
* `.spec.storageClassName`: the persistent volume (PV) type specified for the restoration.
* `.spec.storageSize`: the PV size specified for the restoration. This value must be greater than the backup data size of the TiDB cluster.

> **Note:**
>
> TiDB Operator creates a PVC for data recovery. The backup data is downloaded from the remote storage to the PV first, and then restored. If you want to delete this PVC after the recovery is completed, you can refer to [Delete Resource](cheat-sheet.md#delete-resources) to delete the recovery Pod first, and then delete the PVC.
