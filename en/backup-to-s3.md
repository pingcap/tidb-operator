---
title: Back up TiDB Cluster Data to S3-Compatible Storage
summary: Learn how to back up the TiDB cluster to the S3-compatible storage.
category: how-to
---

# Back up TiDB Cluster Data to S3-Compatible Storage

This document describes how to back up the data of the TiDB cluster in Kubernetes to the S3-compatible storage. "Backup" in this document refers to full backup (ad-hoc full backup and scheduled full backup). For the underlying implementation, [`mydumper`](https://pingcap.com/docs/v3.0/reference/tools/mydumper) is used to get the logic backup of the TiDB cluster, and then this backup data is sent to the S3-compatible storage.

The backup method described in this document is implemented based on CustomResourceDefinition (CRD) in TiDB Operator v1.1 or later versions. For the backup method implemented based on Helm Charts, refer to [Back up and Restore TiDB Cluster Data Based on Helm Charts](backup-and-restore-using-helm-charts.md).

## Ad-hoc full backup

Ad-hoc full backup describes the backup by creating a `Backup` custom resource (CR) object. TiDB Operator performs the specific backup operation based on this `Backup` object. If an error occurs during the backup process, TiDB Operator does not retry and you need to handle this error manually.

For the current S3-compatible storage types, Ceph and Amazon S3 work normally as tested. Therefore, this document shows examples in which the data of the `demo1` TiDB cluster in the `test1` Kubernetes namespace is backed up to Ceph and Amazon S3 respectively.

### Prerequisites for ad-hoc backup

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml) and execute the following command to create the role-based access control (RBAC) resources in the `test1` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. Create the `s3-secret` secret which stores the credential used to access the S3-compatible storage:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic s3-secret --from-literal=access_key=xxx --from-literal=secret_key=yyy --namespace=test1
    ```

3. Create the `backup-demo1-tidb-secret` secret which stores the root account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=<password> --namespace=test1
    ```

### Ad-hoc backup process

+ Back up data to Amazon S3

    1. In the `backup-s3.yaml` file, edit `host`, `port`, `user`, `projectId` and save your changes.

        {{< copyable "" >}}

        ```yaml
        ---
        apiVersion: pingcap.com/v1alpha1
        kind: Backup
        metadata:
        name: demo1-backup-s3
        namespace: test1
        spec:
        from:
            host: <tidb-host-ip>
            port: <tidb-port>
            user: <tidb-user>
            secretName: backup-demo1-tidb-secret
        s3:
            provider: aws
            secretName: s3-secret
            # region: us-east-1
            # storageClass: STANDARD_IA
            # acl: private
            # endpoint:
        storageClassName: local-storage
        storageSize: 10Gi
        ```

    2. Create the `Backup` CR and back up data to Amazon S3:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f backup-s3.yaml
        ```

+ Back up data to Amazon S3

    1. In the `backup-s3.yaml` file, edit `host`, `port`, `user`, `projectId` and save your changes.

        {{< copyable "" >}}

        ```yaml
        ---
        apiVersion: pingcap.com/v1alpha1
        kind: Backup
        metadata:
        name: demo1-backup-s3
        namespace: test1
        spec:
        from:
            host: <tidb-host-ip>
            port: <tidb-port>
            user: <tidb-user>
            secretName: backup-demo1-tidb-secret
        s3:
            provider: ceph
            secretName: s3-secret
            endpoint: http://10.0.0.1:30074
        storageClassName: local-storage
        storageSize: 10Gi
        ```

    2. Create the `Backup` CR and back up data to Ceph:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-s3.yaml
    ```

In the above two examples, all data of the TiDB cluster is exported and backed up to Amazon S3 and Ceph respectively. You can ignore the `region`, `acl`, `endpoint`, and `storageClass` configuration items in the Amazon S3 configuration. S3-compatible storage types other than Amazon S3 can also use configuration similar to that of Amazon S3. You can also leave the configuration item fields empty if you do not need to configure these items as shown in the above Ceph configuration.

Amazon S3 supports the following access-control list (ACL) polices:

* `private`
* `public-read`
* `public-read-write`
* `authenticated-read`
* `bucket-owner-read`
* `bucket-owner-full-control`

If the ACL policy is not configured, the `private` policy is used by default. For the detailed description of these access control policies, refer to [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html).

Amazon S3 supports the following `storageClass` types:

* `STANDARD`
* `REDUCED_REDUNDANCY`
* `STANDARD_IA`
* `ONEZONE_IA`
* `GLACIER`
* `DEEP_ARCHIVE`

If `storageClass` is not configured, `STANDARD_IA` is used by default. For the detailed description of these storage types, refer to [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html).

After creating the `Backup` CR, you can use the following command to check the backup status:

{{< copyable "shell-regular" >}}

 ```shell
 kubectl get bk -n test1 -owide
 ```

More `Backup` CRs are described as follows:

* `.spec.metadata.namespace`: the namespace where the `Backup` CR is located.
* `.spec.from.host`: the address of the TiDB cluster to be backed up.
* `.spec.from.port`: the port of the TiDB cluster to be backed up.
* `.spec.from.user`: the accessing user of the TiDB cluster to be backed up.
* `.spec.from.tidbSecretName`: the secret of the credential needed by the TiDB cluster to be backed up.
* `.spec.storageClassName`: the persistent volume (PV) type specified for the backup operation. If this item is not specified, the value of the `default-backup-storage-class-name` parameter (`standard` by default, specified when TiDB Operator is started) is used by default.
* `.spec.storageSize`: the PV size specified for the backup operation. This value must be greater than the size of the TiDB cluster to be backed up.

More S3-compatible `provider`s are described as follows:

* `alibaba`: Alibaba Cloud Object Storage System (OSS) formerly Aliyun
* `digitalocean`: Digital Ocean Spaces
* `dreamhost`: Dreamhost DreamObjects
* `ibmcos`: IBM COS S3
* `minio`: Minio Object Storage
* `netease`: Netease Object Storage (NOS)
* `wasabi`: Wasabi Object Storage
* `other`: Any other S3 compatible provider

## Scheduled full backup to S3-compatible storage

You can set a backup policy to perform scheduled backups of the TiDB cluster, and set a backup retention policy to avoid excessive backup items. A scheduled full backup is described by a custom `BackupSchedule` CR object. A full backup is triggered at each backup time point. Its underlying implementation is the ad-hoc full backup.

### Prerequisites for scheduled backup

The prerequisites for the scheduled backup is the same as the [prerequisites for ad-hoc backup](#prerequisites-for-ad-hoc-backup).

### Scheduled backup process

+ Scheduled backup to Amazon S3.

    1. In the `backup-gcs.yaml` file, edit `host`, `port`, `user`, `projectId` and save your changes.

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
            from:
            host: <tidb-host-ip>
            port: <tidb-port>
            user: <tidb-user>
            secretName: backup-demo1-tidb-secret
            s3:
            provider: aws
            secretName: s3-secret
            # region: us-east-1
            # storageClass: STANDARD_IA
            # acl: private
            # endpoint:
            storageClassName: local-storage
            storageSize: 10Gi
        ```

    2. Create the `BackupSchedule` CR to enable the scheduled full backup to Amazon S3:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f backup-schedule-s3.yaml
        ```

+ Scheduled backup to Ceph.

    1. In the `backup-gcs.yaml` file, edit `host`, `port`, `user`, `projectId` and save your changes.

        {{< copyable "shell-regular" >}}

        ```yaml
        ---
        apiVersion: pingcap.com/v1alpha1
        kind: BackupSchedule
        metadata:
        name: demo1-backup-schedule-ceph
        namespace: test1
        spec:
        #maxBackups: 5
        #pause: true
        maxReservedTime: "3h"
        schedule: "*/2 * * * *"
        backupTemplate:
            from:
            host: <tidb-host-ip>
            port: <tidb-port>
            user: <tidb-user>
            secretName: backup-demo1-tidb-secret
            s3:
            provider: ceph
            secretName: s3-secret
            endpoint: http://10.0.0.1:30074
            storageClassName: local-storage
            storageSize: 10Gi
        ```

    2. Create the `BackupSchedule` CR to enable the scheduled full backup to Ceph:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f backup-schedule-s3.yaml
        ```

After creating the scheduled full backup, you can use the following command to check the backup status:

{{< copyable "shell-regular" >}}

```shell
kubectl get bks -n test1 -owide
```

You can use the following command to check all the backup items:

{{< copyable "shell-regular" >}}

```shell
kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-s3 -n test1
```

From the above two examples, you can see that the `backupSchedule` configuration consists of two parts. One is the unique configuration of `backupSchedule`, and the other is `backupTemplate`. `backupTemple` specifies the configuration related to the S3-compatible storage, which is the same as the configuration of the ad-hoc full backup to the S3-compatible storage (refer to [Ad-hoc backup process](#ad-hoc-backup-process) for details). The following are the unique configuration items of `backupSchedule`:

+ `.spec.maxBackups`: A backup retention policy, which determines the maximum number of backup items to be retained. When this value is exceeded, the outdated backup items will be deleted. If you set this configuration item to `0`, all backup items are retained.
+ `.spec.maxReservedTime`: A backup retention policy based on time. For example, if you set the value of this configuration to `24h`, only backup items within the recent 24 hours are retained. All backup items out of this time are deleted. For the time format, refer to [`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration). If you have set the maximum number of backup items and the longest retention time of backup items at the same time, the latter setting takes effect.
+ `.spec.schedule`: The time scheduling format of Cron. Refer to [Cron](https://en.wikipedia.org/wiki/Cron) for details.
+ `.spec.pause`: `false` by default. If this parameter is set to `true`, the scheduled scheduling is paused. In this situation, the backup operation will not be performed even if the scheduling time is reached. During this pause, the backup [Garbage Collection](https://pingcap.com/docs/v3.0/reference/garbage-collection/overview) (GC) runs normally. If you change `true` to `false`, the full backup process is restarted.
