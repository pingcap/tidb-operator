---
title: Restore Data from S3-Compatible Storage Using BR
summary: Learn how to restore data from Amazon S3-compatible storage using BR.
aliases: ['/docs/tidb-in-kubernetes/dev/restore-from-aws-s3-using-br/']
---

# Restore Data from S3-Compatible Storage Using BR

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. [BR](https://pingcap.com/docs/stable/br/backup-and-restore-tool/) is used to perform the restore.

The restore method described in this document is implemented based on Custom Resource Definition (CRD) in TiDB Operator v1.1 or later versions.

This document shows an example in which the backup data stored in the specified path on the Amazon S3 storage is restored to the TiDB cluster.

## Prerequisites

> **Note:**
>
> If TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR will automatically adjust `tikv_gc_life_time`. You do not need to configure `spec.to` fields in the `Restore` CR. In addition, you can skip the steps of creating the `restore-demo2-tidb-secret` secret and [configuring database account privileges](#required-database-account-privileges).

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Grant permissions to the remote storage.

    To grant permissions to access S3-compatible remote storage, refer to [AWS account permissions](grant-permissions-to-remote-storage.md#aws-account-permissions).

    If you use Ceph as the backend storage for testing, you can grant permissions by [using AccessKey and SecretKey](grant-permissions-to-remote-storage.md#grant-permissions-by-accesskey-and-secretkey).

3. Create the `restore-demo2-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=password=${password} --namespace=test2
    ```

## Required database account privileges

* The `SELECT` and `UPDATE` privileges of the `mysql.tidb` table: Before and after the restore, the `Restore` CR needs a database account with these privileges to adjust the GC time.

## Restore process

+ If you grant permissions by importing AccessKey and SecretKey, create the `Restore` CR, and restore cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f resotre-aws-s3.yaml
    ```

    The content of `restore-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
    spec:
      br:
        cluster: demo2
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # sendCredToTikv: true
      # # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      # to:
      #   host: ${tidb_host}
      #   port: ${tidb_port}
      #   user: ${tidb_user}
      #   secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        secretName: s3-secret
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

+ If you grant permissions by associating IAM with Pod, create the `Restore` CR, and restore cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f restore-aws-s3.yaml
    ```

    The content of `restore-aws-s3.yaml` is as follows:

     ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      br:
        cluster: demo2
        sendCredToTikv: false
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

+ If you grant permissions by associating IAM with ServiceAccount, create the `Restore` CR, and restore cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f restore-aws-s3.yaml
    ```

    The content of `restore-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
    spec:
      serviceAccount: tidb-backup-manager
      br:
        cluster: demo2
        sendCredToTikv: false
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

After creating the `Restore` CR, execute the following command to check the restore status:

{{< copyable "shell-regular" >}}

```bash
kubectl get rt -n test2 -o wide
```

The examples above restore data from the `spec.s3.prefix` folder of the `spec.s3.bucket` bucket on Amazon S3 storage to the `demo2` TiDB cluster in the `test2` namespace. For more information about S3-compatible storage configuration, refer to [S3 storage fields](backup-restore-overview.md#s3-storage-fields).

In the examples above, some parameters in `.spec.br` can be ignored, such as `logLevel`, `statusAddr`, `concurrency`, `rateLimit`, `checksum`, `timeAgo`, and `sendCredToTikv`. For more information about BR configuration, refer to [BR fields](backup-restore-overview.md#br-fields).

For more information about the `Restore` CR fields, refer to [Restore CR fields](backup-restore-overview.md#restore-cr-fields).

## Troubleshooting

If you encounter any problem during the restore process, refer to [Common Deployment Failures](deploy-failures.md).
