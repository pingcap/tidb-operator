---
title: Backup and Restore Overview
summary: Learn how to perform backup and restore on the TiDB cluster in Kubernetes using BR, Dumpling, and TiDB Lightning.
---

# Backup and Restore Overview

This document describes how to perform backup and restore on the TiDB cluster in Kubernetes. The backup and restore tools used are [BR](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool), [Dumpling](https://docs.pingcap.com/tidb/stable/dumpling-overview), and [TiDB Lightning](https://docs.pingcap.com/tidb/stable/get-started-with-tidb-lightning).

TiDB Operator 1.1 and later versions implement the backup and restore methods using Custom Resource Definition (CRD):

+ If your TiDB cluster version is v3.1 or later, refer to the following documents:

    - [Back up Data to S3-Compatible Storage Using BR](backup-to-aws-s3-using-br.md)
    - [Back up Data to GCS Using BR](backup-to-gcs-using-br.md)
    - [Back up Data to PV Using BR](backup-to-pv-using-br.md)
    - [Restore Data from S3-Compatible Storage Using BR](restore-from-aws-s3-using-br.md)
    - [Restore Data from GCS Using BR](restore-from-gcs-using-br.md)
    - [Restore Data from PV Using BR](restore-from-pv-using-br.md)

+ If your TiDB cluster version is earlier than v3.1, refer to the following documents:

    - [Back up Data to S3-Compatible Storage Using Dumpling](backup-to-s3.md)
    - [Back up Data to GCS Using Dumpling](backup-to-gcs.md)
    - [Restore Data from S3-Compatible Storage Using TiDB Lightning](restore-from-s3.md)
    - [Restore Data from GCS Using TiDB Lightning](restore-from-gcs.md)

## User scenarios

[Dumpling](https://docs.pingcap.com/tidb/stable/dumpling-overview) is a data export tool that exports data stored in TiDB/MySQL as SQL or CSV data files to get the logic full backup or export. If you need to back up SST files (Key-Value pairs) directly or perform latency-insensitive incremental backup, refer to [BR](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool). For real-time incremental backup, refer to [TiCDC](https://docs.pingcap.com/tidb/stable/ticdc-overview).

[TiDB Lightning](https://docs.pingcap.com/tidb/stable/get-started-with-tidb-lightning) is a tool used for fast full data import into a TiDB cluster. TiDB Lightning supports Dumpling or CSV format data source. You can use TiDB Lightning for the following two purposes:

- Quickly import large amounts of data
- Restore all backup data

[BR](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool) is a command-line tool for distributed backup and restore of the TiDB cluster data. Compared with Dumpling and mydumper, BR is more suitable for scenarios of huge data volume. BR only supports TiDB v3.1 and later versions.

## Backup CR fields

To back up data for a TiDB cluster in Kubernetes, you can create a `Backup` Custom Resource (CR) object. For detailed backup process, refer to documents listed in [Backup and Restore Overview](#backup-and-restore-overview).

This section introduces the fields in the `Backup` CR.

### General fields

* `.spec.metadata.namespace`: The namespace where the `Backup` CR is located.
* `.spec.toolImage`: The tool image used by `Backup`.

    - When using BR for backup, you can specify the BR version in this field.
        - If the field is not specified or the value is empty, the `pingcap/br:${tikv_version}` image is used for backup by default.
        - If the BR version is specified in this field, such as `.spec.toolImage: pingcap/br:v5.2.1`, the image of the specified version is used for backup.
        - If the BR version is not specified in the field, such as `.spec.toolImage: private/registry/br`, the `private/registry/br:${tikv_version}` image is used for backup.
    - When using Dumpling for backup, you can specify the Dumpling version in this field. For example, `spec.toolImage: pingcap/dumpling:v5.2.1`. If not specified, the Dumpling version specified in `TOOLKIT_VERSION` of the [Backup Manager Dockerfile](https://github.com/pingcap/tidb-operator/blob/master/images/tidb-backup-manager/Dockerfile) is used for backup by default.
    - TiDB Operator supports this configuration starting from v1.1.9.
* `.spec.tikvGCLifeTime`: The temporary `tikv_gc_life_time` time setting during the backup, which defaults to 72h.

    Before the backup begins, if the `tikv_gc_life_time` setting in the TiDB cluster is smaller than `spec.tikvGCLifeTime` set by the user, TiDB Operator [adjusts the value of `tikv_gc_life_time`](https://docs.pingcap.com/tidb/stable/dumpling-overview#tidb-gc-settings-when-exporting-a-large-volume-of-data) to the value of `spec.tikvGCLifeTime`. This operation makes sure that the backup data is not garbage-collected by TiKV.

    After the backup, no matter the backup is successful or not, as long as the previous `tikv_gc_life_time` value is smaller than `.spec.tikvGCLifeTime`, TiDB Operator will try to set `tikv_gc_life_time` to the previous value.

    In extreme cases, if TiDB Operator fails to access the database, TiDB Operator cannot automatically recover the value of `tikv_gc_life_time` and treats the backup as failed. At this time, you can view `tikv_gc_life_time` of the current TiDB cluster using the following statement:

    {{< copyable "sql" >}}

    ```sql
    select VARIABLE_NAME, VARIABLE_VALUE from mysql.tidb where VARIABLE_NAME like "tikv_gc_life_time";
    ```

    In the output of the command above, if the value of `tikv_gc_life_time` is still larger than expected (usually 10m), you need to [set `tikv_gc_life_time` back](https://docs.pingcap.com/tidb/stable/dumpling-overview#tidb-gc-settings-when-exporting-a-large-volume-of-data) to the previous value manually:

* `.spec.cleanPolicy`: The cleaning policy for the backup data when the backup CR is deleted.

    Three clean policies are supported:

    * `Retain`: Under any circumstances, retain the backup data when deleting the backup CR.
    * `Delete`: Under any circumstances, delete the backup data when deleting the backup CR.
    * `OnFailure`: If the backup fails, delete the backup data when deleting the backup CR.

    If this field is not configured, or if you configure a value other than the three policies above, the backup data is retained.

    Note that in v1.1.2 and earlier versions, this field does not exist. The backup data is deleted along with the CR by default. For v1.1.3 or later versions, if you want to keep this earlier behavior, set this field to `Delete`.

* `.spec.from.host`: The address of the TiDB cluster to be backed up, which is the service name of the TiDB cluster to be exported, such as `basic-tidb`.
* `.spec.from.port`: The port of the TiDB cluster to be backed up.
* `.spec.from.user`: The accessing user of the TiDB cluster to be backed up.
* `.spec.from.secretName`: The secret that contains the password of the `.spec.from.user`.
* `.spec.from.tlsClientSecretName`: The secret of the certificate used during the backup.

    If [TLS](enable-tls-between-components.md) is enabled for the TiDB cluster, but you do not want to back up data using the `${cluster_name}-cluster-client-secret` as described in [Enable TLS between TiDB Components](enable-tls-between-components.md), you can use the `.spec.from.tlsClient.tlsSecret` parameter to specify a secret for the backup. To generate the secret, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

* `.spec.storageClassName`: The persistent volume (PV) type specified for the backup operation.
* `.spec.storageSize`: The PV size specified for the backup operation (`100 Gi` by default). This value must be greater than the size of the TiDB cluster to be backed up.

    The PVC name corresponding to the `Backup` CR of a TiDB cluster is fixed. If the PVC already exists in the cluster namespace and the size is smaller than `spec.storageSize`, you need to delete this PVC and then run the Backup job.

* `.spec.tableFilter`: Specifies tables that match the [table filter rules](https://docs.pingcap.com/tidb/stable/table-filter/) for BR or Dumpling. This field can be ignored by default.

    If the field is not configured, the default value of `tableFilter` is as follows:

    ```bash
    tableFilter:
    - "*.*"
    - "!/^(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA)$/.*"
    ```

    If you use BR to perform backup, BR backs up all schemas except the system schema.

    > **Note:**
    >
    > To use the table filter to exclude `db.table`, you need to first add the `*.*` rule to include all tables. For example:
    >
    > ```
    > tableFilter:
    > - "*.*"
    > - "!db.table"
    > ```

### BR fields

* `.spec.br.cluster`: The name of the cluster to be backed up.
* `.spec.br.clusterNamespace`: The `namespace` of the cluster to be backed up.
* `.spec.br.logLevel`: The log level (`info` by default).
* `.spec.br.statusAddr`: The listening address through which BR provides statistics. If not specified, BR does not listen on any status address by default.
* `.spec.br.concurrency`: The number of threads used by each TiKV process during backup. Defaults to `4` for backup and `128` for restore.
* `.spec.br.rateLimit`: The speed limit, in MB/s. If set to `4`, the speed limit is 4 MB/s. The speed limit is not set by default.
* `.spec.br.checksum`: Whether to verify the files after the backup is completed. Defaults to `true`.
* `.spec.br.timeAgo`: Backs up the data before `timeAgo`. If the parameter value is not specified (empty by default), it means backing up the current data. It supports data formats such as "1.5h" and "2h45m". See [ParseDuration](https://golang.org/pkg/time/#ParseDuration) for more information.
* `.spec.br.sendCredToTikv`: Whether the BR process passes its AWS or GCP privileges to the TiKV process. Defaults to `true`.
* `.spec.br.options`: The extra arguments that BR supports. This field is supported since TiDB Operator v1.1.6. It accepts an array of strings and can be used to specify the last backup timestamp `--lastbackupts` for incremental backup.

### S3 storage fields

* `.spec.s3.provider`: The supported S3-compatible storage provider. Options are as follows:

    - `alibaba`: Alibaba Cloud Object Storage System (OSS), formerly Aliyun
    - `digitalocean`: Digital Ocean Spaces
    - `dreamhost`: Dreamhost DreamObjects
    - `ibmcos`: IBM COS S3
    - `minio`: Minio Object Storage
    - `netease`: Netease Object Storage (NOS)
    - `wasabi`: Wasabi Object Storage
    - `other`: Any other S3 compatible provider

* `spec.s3.region`: If you want to use Amazon S3 for backup storage, configure this field as the region where Amazon S3 is located.
* `.spec.s3.bucket`: The name of the bucket compatible with S3 storage.
* `.spec.s3.prefix`: If you set this field, the value is used to make up the remote storage path `s3://${.spec.s3.bucket}/${.spec.s3.prefix}/backupName`.
* `.spec.s3.acl`: The supported access-control list (ACL) policies.

    Amazon S3 supports the following ACL options:

    * `private`
    * `public-read`
    * `public-read-write`
    * `authenticated-read`
    * `bucket-owner-read`
    * `bucket-owner-full-control`

    If the field is not configured, the policy defaults to `private`. For more information on the ACL policies, refer to [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html).

* `.spec.s3.storageClass`: The supported storage class.

    Amazon S3 supports the following storage class options:

    * `STANDARD`
    * `REDUCED_REDUNDANCY`
    * `STANDARD_IA`
    * `ONEZONE_IA`
    * `GLACIER`
    * `DEEP_ARCHIVE`

    If the field is not configured, the storage class defaults to `STANDARD_IA`. For more information on storage classes, refer to [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html).

### GCS fields

* `.spec.gcs.projectId`: The unique identifier of the user project on GCP. To obtain the project ID, refer to [GCP documentation](https://cloud.google.com/resource-manager/docs/creating-managing-projects).
* `.spec.gcs.bucket`: The name of the bucket which stores data.
* `.spec.gcs.prefix`: If you set this field, the value is used to make up the path of the remote storage: `gcs://${.spec.gcs.bucket}/${.spec.gcs.prefix}/backupName`. This field can be ignored.
* `spec.gcs.storageClass`: The supported storage class.

    GCS supports the following storage class options:

    * `MULTI_REGIONAL`
    * `REGIONAL`
    * `NEARLINE`
    * `COLDLINE`
    * `DURABLE_REDUCED_AVAILABILITY`

    If the field is not configured, the storage class defaults to `COLDLINE`. For more information on storage classes, refer to [GCS documentation](https://cloud.google.com/storage/docs/storage-classes).

* `.spec.gcs.objectAcl`: The supported object access-control list (ACL) policies.

    GCS supports the following object ACL options:

    * `authenticatedRead`
    * `bucketOwnerFullControl`
    * `bucketOwnerRead`
    * `private`
    * `projectPrivate`
    * `publicRead`

    If the field is not configured, the policy defaults to `private`. For more information on the ACL policies, refer to [GCS documentation](https://cloud.google.com/storage/docs/access-control/lists).

* `.spec.gcs.bucketAcl`: The supported bucket access-control list (ACL) policies.

    GCS supports the following bucket ACL options:

    * `authenticatedRead`
    * `private`
    * `projectPrivate`
    * `publicRead`
    * `publicReadWrite`

    If the field is not configured, the policy defaults to `private`. For more information on the ACL policies, refer to [GCS documentation](https://cloud.google.com/storage/docs/access-control/lists).

### Local storage fields

* `.spec.local.prefix`: The storage directory of the persistent volumes. If you set this field, the value is used to make up the storage path of the persistent volume: `local://${.spec.local.volumeMount.mountPath}/${.spec.local.prefix}/`.
* `.spec.local.volume`: The persistent volume configuration.
* `.spec.local.volumeMount`: The persistent volume mount configuration.

## Restore CR fields

To restore data to a TiDB cluster in Kubernetes, you can create a `Restore` CR object. For detailed restore process, refer to documents listed in [Backup and Restore Overview](#backup-and-restore-overview).

This section introduces the fields in the `Restore` CR.

* `.spec.metadata.namespace`: The namespace where the `Restore` CR is located.
* `.spec.toolImage`：The tools image used by `Restore`.
    - When using BR for restoring, you can specify the BR version in this field. For example,`spec.toolImage: pingcap/br:v5.2.1`. If not specified, `pingcap/br:${tikv_version}` is used for restoring by default.
    - When using Lightning for restoring, you can specify the Lightning version in this field. For example, `spec.toolImage: pingcap/lightning:v5.2.1`. If not specified, the Lightning version specified in `TOOLKIT_VERSION` of the [Backup Manager Dockerfile](https://github.com/pingcap/tidb-operator/blob/master/images/tidb-backup-manager/Dockerfile) is used for restoring by default.
    - TiDB Operator supports this configuration starting from v1.1.9.
* `.spec.to.host`: The address of the TiDB cluster to be restored.
* `.spec.to.port`: The port of the TiDB cluster to be restored.
* `.spec.to.user`: The accessing user of the TiDB cluster to be restored.
* `.spec.to.secretName`: The secret that contains the password of the `.spec.to.user`.
* `.spec.to.tlsClientSecretName`: The secret of the certificate used during the restore.

    If [TLS](enable-tls-between-components.md) is enabled for the TiDB cluster, but you do not want to restore data using the `${cluster_name}-cluster-client-secret` as described in [Enable TLS between TiDB Components](enable-tls-between-components.md), you can use the `.spec.to.tlsClient.tlsSecret` parameter to specify a secret for the restore. To generate the secret, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

* `.spec.storageClassName`: The persistent volume (PV) type specified for the restore operation.
* `.spec.storageSize`: The PV size specified for the restore operation. This value must be greater than the size of the backup data.
* `.spec.tableFilter`: Specifies tables that match the [table filter rules](https://docs.pingcap.com/tidb/stable/table-filter/) for BR. This field can be ignored by default.

    If the field is not configured, the default `tableFilter` value for TiDB Lightning is as follows:

    ```bash
    tableFilter:
    - "*.*"
    - "!/^(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA)$/.*"
    ```

    If this field is not configured, BR restores all the schemas in the backup file.

    > **Note:**
    >
    > To use the table filter to exclude `db.table`, you need to first add the `*.*` rule to include all tables. For example:
    >
    > ```
    > tableFilter:
    > - "*.*"
    > - "!db.table"
    > ```

* `.spec.br`: BR-related configuration. Refer to [BR fields](#br-fields).
* `.spec.s3`: S3-related configuration. Refer to [S3 storage fields](#s3-storage-fields).
* `.spec.gcs`: GCS-related configuration. Refer to [GCS fields](#gcs-fields).
* `.spec.local`: Persistent volume-related configuration. Refer to [Local storage fields](#local-storage-fields).

## BackupSchedule CR fields

The `backupSchedule` configuration consists of two parts. One is the unique configuration of `backupSchedule`, and the other is `backupTemplate`. `backupTemplate` specifies the configuration related to the cluster and remote storage, which is the same as the `spec` configuration of [the `Backup` CR](#backup-cr-fields).

The unique configuration items of `backupSchedule` are as follows:

* `.spec.maxBackups`: A backup retention policy, which determines the maximum number of backup files to be retained. When the number of backup files exceeds this value, the outdated backup file will be deleted. If you set this field to `0`, all backup items are retained.
* `.spec.maxReservedTime`: A backup retention policy based on time. For example, if you set the value of this field to `24h`, only backup files within the recent 24 hours are retained. All backup files older than this value are deleted. For the time format, refer to [`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration). If you have set `.spec.maxBackups` and `.spec.maxReservedTime` at the same time, the latter takes effect.
* `.spec.schedule`: The time scheduling format of Cron. Refer to [Cron](https://en.wikipedia.org/wiki/Cron) for details.
* `.spec.pause`: `false` by default. If this field is set to `true`, the scheduled scheduling is paused. In this situation, the backup operation will not be performed even if the scheduling time is reached. During this pause, the backup Garbage Collection runs normally. If you change `true` to `false`, the scheduled full backup process is restarted.

## Delete the Backup CR

You can delete the `Backup` CR or `BackupSchedule` CR by running the following commands:

{{< copyable "shell-regular" >}}

```shell
kubectl delete backup ${name} -n ${namespace}
kubectl delete backupschedule ${name} -n ${namespace}
```

If you use TiDB Operator v1.1.2 or an earlier version, or if you use TiDB Operator v1.1.3 or a later version and set the value of `spec.cleanPolicy` to `Delete`, TiDB Operator cleans the backup data when it deletes the CR.

In such cases, if you need to delete the namespace, it is recommended that you first delete all the `Backup`/`BackupSchedule` CRs and then delete the namespace.

If you delete the namespace before you delete the `Backup`/`BackupSchedule` CR, TiDB Operator will keep creating jobs to clean the backup data. However, because the namespace is in `Terminating` state, TiDB Operator fails to create such a job, which causes the namespace to be stuck in this state.

To address this issue, delete `finalizers` by running the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl edit backup ${name} -n ${namespace}
```

After deleting the `metadata.finalizers` configuration, you can delete the CR normally.

### Clean backup data

For TiDB Operator v1.2.3 and earlier versions, TiDB Operator cleans the backup data by deleting the backup files one by one.

For TiDB Operator v1.2.4 and later versions, TiDB Operator cleans the backup data by deleting the backup files in batches. For the batch deletion, the deletion methods are different depending on the type of backend storage used for backups.

* For the S3-compatible backend storage, TiDB Operator uses the concurrent batch deletion method, which deletes files in batch concurrently. TiDB Operator starts multiple goroutines concurrently, and each goroutine uses the batch delete API ["DeleteObjects"](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html) to delete multiple files.
* For other types of backend storage, TiDB Operator uses the concurrent deletion method, which deletes files concurrently. TiDB Operator starts multiple goroutines, and each goroutine deletes one file at a time.

For TiDB Operator v1.2.4 and later versions, you can configure the following fields in the Backup CR to control the clean behavior:

* `.spec.cleanOption.pageSize`: Specifies the number of files to be deleted in each batch at a time. The default value is 10000.
* `.spec.cleanOption.disableBatchConcurrency`: If the value of this field is `true`, TiDB Operator disables the concurrent batch deletion method and uses the concurrent deletion method.

    If your S3-compatible backend storage does not support the `DeleteObjects` API, the default concurrent batch deletion method fails. You need to configure this field to `true` to use the concurrent deletion method.

* `.spec.cleanOption.batchConcurrency`: Specifies the number of goroutines to start for the concurrent batch deletion method. The default value is `10`.
* `.spec.cleanOption.routineConcurrency`: Specifies the number of goroutines to start  for the concurrent deletion method. The default value is `100`.
