---
title: Back up Data to S3-Compatible Storage Using BR
summary: Learn how to back up data to Amazon S3-compatible storage using BR.
aliases: ['/docs/tidb-in-kubernetes/dev/backup-to-aws-s3-using-br/']
---

<!-- markdownlint-disable MD007 -->

# Back up Data to S3-Compatible Storage Using BR

This document describes how to back up the data of a TiDB cluster in AWS Kubernetes to the AWS storage using Helm charts. [BR](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool) is used to get the logic backup of the TiDB cluster, and then this backup data is sent to the AWS storage.

The backup method described in this document is implemented using Custom Resource Definition (CRD) in TiDB Operator v1.1 or later versions.

## Ad-hoc backup

Ad-hoc backup supports both full backup and incremental backup. It describes the backup by creating a `Backup` Custom Resource (CR) object. TiDB Operator performs the specific backup operation based on this `Backup` object. If an error occurs during the backup process, TiDB Operator does not retry, and you need to handle this error manually.

To better describe the backup process, this document provides examples in which the data of the `demo1` TiDB cluster in the `test1` Kubernetes namespace is backed up to AWS storage.

### Prerequisites for ad-hoc backup

Before you perform ad-hoc backup, AWS account permissions need to be granted. This section describes three methods to grant AWS account permissions.

> **Note:**
>
> If TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR will automatically adjust `tikv_gc_life_time`. You do not need to configure `spec.tikvGCLifeTime` and `spec.from` fields in the `Backup` CR. In addition, you can skip the steps of creating the `backup-demo1-tidb-secret` secret and [configuring database account privileges](#required-database-account-privileges).

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test1` namespace:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. Grant permissions to the remote storage.

    To grant permissions to access the S3-compatible remote storage, refer to [AWS account permissions](grant-permissions-to-remote-storage.md#aws-account-permissions).

    If you use Ceph as the backend storage for testing, you can grant permissions by [using AccessKey and SecretKey](grant-permissions-to-remote-storage.md#grant-permissions-by-accesskey-and-secretkey).

3. Create the `backup-demo1-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=${password} --namespace=test1
    ```

### Required database account privileges

* The `SELECT` and `UPDATE` privileges of the `mysql.tidb` table: Before and after the backup, the `Backup` CR needs a database account with these privileges to adjust the GC time.

### Process of ad-hoc backup

- If you grant permissions by importing AccessKey and SecretKey, create the `Backup` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-aws-s3.yaml
    ```

    The content of `backup-aws-s3.yaml` is as follows:

    {{< copyable "" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: test1
    spec:
      backupType: full
      br:
        cluster: demo1
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # sendCredToTikv: true
        # options:
        # - --lastbackupts=420134118382108673
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        secretName: s3-secret
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

- If you grant permissions by associating IAM with Pod, create the `Backup` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-aws-s3.yaml
    ```

    {{< copyable "" >}}

    The content of `backup-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: test1
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      backupType: full
      br:
        cluster: demo1
        sendCredToTikv: false
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # options:
        # - --lastbackupts=420134118382108673
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

- If you grant permissions by associating IAM with ServiceAccount, create the `Backup` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-aws-s3.yaml
    ```

    The content of `backup-aws-s3.yaml` is as follows:

    {{< copyable "" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: test1
    spec:
      backupType: full
      serviceAccount: tidb-backup-manager
      br:
        cluster: demo1
        sendCredToTikv: false
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # options:
        # - --lastbackupts=420134118382108673
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

The three examples above use three methods to grant permissions to back up data to Amazon S3 storage. The `acl`, `endpoint`, `storageClass` configuration items of Amazon S3 can be ignored. For more information about S3-compatible storage configuration, refer to [S3 storage fields](backup-restore-overview.md#s3-storage-fields).

In the examples above, some parameters in `.spec.br` can be ignored, such as `logLevel`, `statusAddr`, `concurrency`, `rateLimit`, `checksum`, `timeAgo`, and `sendCredToTikv`. For more information about BR configuration, refer to [BR fields](backup-restore-overview.md#br-fields).

Since TiDB Operator v1.1.6, if you want to back up data incrementally, you only need to specify the last backup timestamp `--lastbackupts` in `spec.br.options`. For the limitations of incremental backup, refer to [Use BR to Back up and Restore Data](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool#back-up-incremental-data).

For more information about the `Backup` CR fields, refer to [Backup CR fields](backup-restore-overview.md#backup-cr-fields).

After you create the `Backup` CR, view the backup status by running the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl get bk -n test1 -o wide
```

## Scheduled full backup

You can set a backup policy to perform scheduled backups of the TiDB cluster, and set a backup retention policy to avoid excessive backup items. A scheduled full backup is described by a custom `BackupSchedule` CR object. A full backup is triggered at each backup time point. Its underlying implementation is the ad-hoc full backup.

### Prerequisites for scheduled full backup

The prerequisites for the scheduled full backup is the same as the [prerequisites for ad-hoc backup](#prerequisites-for-ad-hoc-backup).

### Process of scheduled full backup

+ If you grant permissions by importing AccessKey and SecretKey, create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-scheduler-aws-s3.yaml
    ```

    The content of `backup-scheduler-aws-s3.yaml` is as follows:

    {{< copyable "" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: test1
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        backupType: full
        br:
          cluster: demo1
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status_addr}
          # concurrency: 4
          # rateLimit: 0
          # timeAgo: ${time}
          # checksum: true
          # sendCredToTikv: true
        # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          secretName: s3-secret
          region: us-west-1
          bucket: my-bucket
          prefix: my-folder
    ```

+ If you grant permissions by associating IAM with the Pod, create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-scheduler-aws-s3.yaml
    ```

    The content of `backup-scheduler-aws-s3.yaml` is as follows:

    {{< copyable "" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: test1
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        backupType: full
        br:
          cluster: demo1
          sendCredToTikv: false
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status_addr}
          # concurrency: 4
          # rateLimit: 0
          # timeAgo: ${time}
          # checksum: true
        # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          region: us-west-1
          bucket: my-bucket
          prefix: my-folder
    ```

+ If you grant permissions by associating IAM with ServiceAccount, create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-scheduler-aws-s3.yaml
    ```

    The content of `backup-scheduler-aws-s3.yaml` is as follows:

    {{< copyable "" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: test1
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      serviceAccount: tidb-backup-manager
      backupTemplate:
        backupType: full
        br:
          cluster: demo1
          sendCredToTikv: false
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status_addr}
          # concurrency: 4
          # rateLimit: 0
          # timeAgo: ${time}
          # checksum: true
        # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          region: us-west-1
          bucket: my-bucket
          prefix: my-folder
    ```

After creating the scheduled full backup, use the following command to check the backup status:

{{< copyable "shell-regular" >}}

```bash
kubectl get bks -n test1 -o wide
```

You can use the following command to check all the backup items:

{{< copyable "shell-regular" >}}

```bash
kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-s3 -n test1
```

From the example above, you can see that the `backupSchedule` configuration consists of two parts. One is the unique configuration of `backupSchedule`, and the other is `backupTemplate`.

`backupTemplate` specifies the configuration related to the cluster and remote storage, which is the same as the `spec` configuration of [the `Backup` CR](backup-restore-overview.md#backup-cr-fields). For the unique configuration of `backupSchedule`, refer to [BackupSchedule CR fields](backup-restore-overview.md#backupschedule-cr-fields).

## Delete the backup CR

Refer to [Delete the Backup CR](backup-restore-overview.md#delete-the-backup-cr).

## Troubleshooting

If you encounter any problem during the backup process, refer to [Common Deployment Failures](deploy-failures.md).
