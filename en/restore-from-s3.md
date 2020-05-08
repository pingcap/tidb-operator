---
title: Restore Data from S3-Compatible Storage Using Loader
summary: Learn how to restore data from the S3-compatible storage.
category: how-to
---

# Restore Data from S3-Compatible Storage Using Loader

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. For the underlying implementation, [`loader`](https://pingcap.com/docs/v3.0/reference/tools/loader) is used to perform the restoration.

The restoration method described in this document is implemented based on CustomResourceDefinition (CRD) in TiDB Operator v1.1 or later versions. For the restoration method implemented based on Helm Charts, refer to [Back up and Restore TiDB Cluster Data Based on Helm Charts](backup-and-restore-using-helm-charts.md).

This document shows an example in which the backup data stored in the specified path on the S3-compatible storage is restored to the TiDB cluster.

## Three methods to grant AWS account permissions

- If you use Amazon S3 to back up and restore the cluster, you have three methods to grant permissions. For details, refer to [Back up TiDB Cluster Data to AWS Using BR](backup-to-aws-s3-using-br.md#three-methods-to-grant-aws-account-permissions).
- If Ceph is used as backend storage in backup and restore test, the permission is granted by importing AccessKey and SecretKey.

## Prerequisites

Refer to [Prerequisites](restore-from-aws-s3-using-br.md#prerequisites-for-ad-hoc-full-backup).

## Restoration process

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
        endpoint: http://10.233.2.161
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
        region: us-west-1
        secretName: s3-secret
        path: s3://${backup_path}
      storageClassName: local-storage
      storageSize: 1Gi
    ```

+ Create the `Restore` CR, and restore the cluster data by binding IAM with Pod to grant permissions:

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
          region: us-west-1
          path: s3://${backup_path}
        storageClassName: local-storage
        storageSize: 1Gi
    ```

+ Create the `Restore` CR, and restore the cluster data by binding IAM with ServiceAccount to grant permissions:

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
          region: us-west-1
          path: s3://${backup_path}
        storageClassName: local-storage
        storageSize: 1Gi
    ```

After creating the `Restore` CR, execute the following command to check the restoration status:

{{< copyable "shell-regular" >}}

```shell
kubectl get rt -n test2 -owide
```

In the above example, the backup data stored in the `spec.s3.path` path on the S3-compatible storage is restored to the `spec.to.host` TiDB cluster. For the configuration of the S3-compatible storage, refer to [backup-s3.yaml](backup-to-s3.md#ad-hoc-backup-process).

More `Restore` CRs are described as follows:

* `.spec.metadata.namespace`: the namespace where the `Restore` CR is located.
* `.spec.to.host`: the address of the TiDB cluster to be restored.
* `.spec.to.port`: the port of the TiDB cluster to be restored.
* `.spec.to.user`: the accessing user of the TiDB cluster to be restored.
* `.spec.to.tidbSecretName`: the secret of the credential needed by the TiDB cluster to be restored.
* `.spec.storageClassName`: the persistent volume (PV) type specified for the restoration. If this item is not specified, the value of the `default-backup-storage-class-name` parameter (`standard` by default, specified when TiDB Operator is started) is used by default.
* `.spec.storageSize`: the PV size specified for the restoration. This value must be greater than the size of the backed up TiDB cluster.
