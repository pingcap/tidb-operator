---
title: Back up Data to PV Using BR
summary: Learn how to back up data to Persistent Volume (PV) using BR.
---

# Back up Data to PV Using BR

This document describes how to back up the data of a TiDB cluster in Kubernetes to [Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (PVs). [BR](https://pingcap.com/docs/stable/br/backup-and-restore-tool/) is used to get the backup of the TiDB cluster, and then the backup data is sent to PVs.

PVs in this documentation can be any [Kubernetes supported Persistent Volume types](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes). This document uses NFS as an example PV type.

The backup method described in this document is implemented using Custom Resource Definition (CRD) in TiDB Operator v1.1 or later versions.

## Ad-hoc backup

Ad-hoc backup supports both full backup and incremental backup. It describes the backup by creating a `Backup` Custom Resource (CR) object. TiDB Operator performs the specific backup operation based on this `Backup` object. If an error occurs during the backup process, TiDB Operator does not retry, and you need to handle this error manually.

This document provides examples in which the data of the `demo1` TiDB cluster in the `test1` Kubernetes namespace is backed up to NFS.

### Prerequisites for ad-hoc backup

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test1` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. Create the `backup-demo1-tidb-secret` secret which stores the root account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=<password> --namespace=test1
    ```

    > **Note:**
    >
    > If TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, `tikv_gc_life_time` will be adjusted by BR automatically, so you can omit this step.

3. Ensure that the NFS server is accessible from your Kubernetes cluster.

### Required database account privileges

* The `SELECT` and `UPDATE` privileges of the `mysql.tidb` table: Before and after the backup, the `Backup` CR needs a database account with these privileges to adjust the GC time.

> **Note:**
>
> If TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, `tikv_gc_life_time` will be adjusted by BR automatically, so you can omit this step.

### Process of ad-hoc backup

1. Create the `Backup` CR, and back up cluster data to NFS as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-nfs.yaml
    ```

    The content of `backup-nfs.yaml` is as follows:

    {{< copyable "shell-regular" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-nfs
      namespace: test1
    spec:
      # # backupType: full
      # # Only needed for TiDB Operator < v1.1.7 or TiDB < v4.0.8
      # from:
      #   host: ${tidb-host}
      #   port: ${tidb-port}
      #   user: ${tidb-user}
      #   secretName: backup-demo1-tidb-secret
      br:
        cluster: demo1
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status-addr}
        # concurrency: 4
        # rateLimit: 0
        # checksum: true
        # options:
        # - --lastbackupts=420134118382108673
      local:
        prefix: backup-nfs
        volume:
          name: nfs
          nfs:
            server: ${nfs_server_ip}
            path: /nfs
        volumeMount:
          name: nfs
          mountPath: /nfs
    ```

    In the example above, some parameters in `spec.br` can be ignored, such as `logLevel`, `statusAddr`, `concurrency`, `rateLimit`, `checksum`, and `timeAgo`.

    Since TiDB Operator v1.1.6, if you want to back up incrementally, you only need to specify the last backup timestamp `--lastbackupts` in `spec.br.options`. For the limitations of incremental backup, refer to [Use BR to Back up and Restore Data](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool#back-up-incremental-data).

    More `br` parameter descriptions:

    * `spec.br.cluster`: The name of the cluster to be backed up.
    * `spec.br.clusterNamespace`: The `namespace` of the cluster to be backed up.
    * `spec.br.logLevel`: The log level (`info` by default).
    * `spec.br.statusAddr`: The listening address through which BR provides statistics. If not specified, BR does not listen on any status address by default.
    * `spec.br.concurrency`: The number of threads used by each TiKV process during backup. Defaults to `4` for backup and `128` for restore.
    * `spec.br.rateLimit`: The speed limit, in MB/s. If set to `4`, the speed limit is 4 MB/s. The speed limit is not set by default.
    * `spec.br.checksum`: Whether to verify the files after the backup is completed. Defaults to `true`.
    * `spec.br.timeAgo`: Backs up the data before `timeAgo`. If the parameter value is not specified (empty by default), it means backing up the current data. It supports data formats such as "1.5h" and "2h45m". See [ParseDuration](https://golang.org/pkg/time/#ParseDuration) for more information.
    * `spec.br.options`: The extra BR argument supported since TiDB Operator v1.1.6. It accepts an array of strings. This could be used to specify the last backup timestamp `--lastbackupts` for incremental backup.

    This example backs up all data in the TiDB cluster to NFS.

2. After creating the `Backup` CR, use the following command to check the backup status:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bk -n test1 -owide
    ```

More descriptions of fields in the `Backup` CR:

* `.spec.metadata.namespace`: The namespace where the `Backup` CR is located.
* `.spec.tikvGCLifeTime`: The temporary `tikv_gc_life_time` time setting during the backup. Defaults to 72h.

    Before the backup begins, if the `tikv_gc_life_time` setting in the TiDB cluster is smaller than `spec.tikvGCLifeTime` set by the user, TiDB Operator [adjusts the value of `tikv_gc_life_time`](https://docs.pingcap.com/tidb/stable/dumpling-overview#tidb-gc-settings-when-exporting-a-large-volume-of-data) to the value of `spec.tikvGCLifeTime`. This operation makes sure that the backup data is not garbage-collected by TiKV.

    After the backup, no matter whether the backup is successful or not, as long as the previous `tikv_gc_life_time` is smaller than `.spec.tikvGCLifeTime`, TiDB Operator will try to set `tikv_gc_life_time` to the previous value.

    In extreme cases, if TiDB Operator fails to access the database, TiDB Operator cannot automatically recover the value of `tikv_gc_life_time` and treats the backup as failed. At this time, you can view `tikv_gc_life_time` of the current TiDB cluster using the following statement:

    {{< copyable "sql" >}}

    ```sql
    SELECT VARIABLE_NAME, VARIABLE_VALUE FROM mysql.tidb WHERE VARIABLE_NAME LIKE "tikv_gc_life_time";
    ```

    In the output of the command above, if the value of `tikv_gc_life_time` is still larger than expected (10m by default), it means TiDB Operator failed to automatically recover the value. Therefore, you need to [set `tikv_gc_life_time`](https://docs.pingcap.com/tidb/stable/dumpling-overview#tidb-gc-settings-when-exporting-a-large-volume-of-data) back to the previous value manually:

    {{< copyable "sql" >}}

    ```sql
    UPDATE mysql.tidb SET VARIABLE_VALUE = '10m' WHERE VARIABLE_NAME = 'tikv_gc_life_time';
    ```

    > **Note:**
    >
    > If TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, `tikv_gc_life_time` will be adjusted by BR automatically, so you can omit `spec.tikvGCLifeTime`.

- `.spec.cleanPolicy`: The clean policy of the backup data when the backup CR is deleted after the backup is completed.

    Three clean policies are supported:

    - `Retain`: On any circumstances, retain the backup data when deleting the backup CR.
    - `Delete`: On any circumstances, delete the backup data when deleting the backup CR.
    - `OnFailure`: If the backup fails, delete the backup data when deleting the backup CR.

    If this field is not configured, or if you configure a value other than the three policies above, the backup data is retained.

    Note that in v1.1.2 and earlier versions, this field does not exist. The backup data is deleted along with the CR by default. For v1.1.3 or later versions, if you want to keep this behavior, set this field to `Delete`.

- `.spec.from.host`: The address of the TiDB cluster to be backed up, which is the service name of the TiDB cluster to be exported, such as `demo1-tidb`.
- `.spec.from.port`: The port of the TiDB cluster to be backed up.
- `.spec.from.user`: The accessing user of the TiDB cluster to be backed up.
- `.spec.from.tidbSecretName`: The secret containing the password of the `.spec.from.user` in the TiDB cluster.
- `.spec.from.tlsClientSecretName`: The secret of the certificate used during the backup.

    If [TLS](enable-tls-between-components.md) is enabled for the TiDB cluster, but you do not want to back up data using the `${cluster_name}-cluster-client-secret` as described in [Enable TLS between TiDB Components](enable-tls-between-components.md), you can use the `.spec.from.tlsClientSecretName` parameter to specify a secret for the backup. To generate the secret, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

    > **Note:**
    >
    > If TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, `tikv_gc_life_time` will be adjusted by BR automatically, so you can omit `spec.from`.

- `.spec.local.prefix`: This field is used to make up the path of the remote storage: `local://${.spec.local.volumeMount.mountPath}/${.spec.local.prefix}/`. It is recommended to configure this field.
- `.spec.tableFilter`: BR only backs up tables that match the [table filter rules](https://docs.pingcap.com/tidb/stable/table-filter/). This field can be ignored by default. If the field is not configured, BR backs up all schemas except the system schemas.

    > **Note:**
    >
    > To use the table filter to exclude `db.table`, you need to add the `*.*` rule to include all tables first. For example:

    ```
    tableFilter:
    - "*.*"
    - "!db.table"
    ```

## Scheduled full backup

You can set a backup policy to perform scheduled backups of the TiDB cluster, and set a backup retention policy to avoid excessive backup items. A scheduled full backup is described by a custom `BackupSchedule` CR object. A full backup is triggered at each backup time point. Its underlying implementation is the ad-hoc full backup.

### Prerequisites for scheduled full backup

The prerequisites for the scheduled full backup is the same with the [prerequisites for ad-hoc backup](#prerequisites-for-ad-hoc-backup).

### Process of scheduled full backup

1. Create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-schedule-nfs.yaml
    ```

    The content of `backup-schedule-nfs.yaml` is as follows:

   ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-nfs
      namespace: test1
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        # Only needed for TiDB Operator < v1.1.7 or TiDB < v4.0.8
        # from:
        #   host: ${tidb_host}
        #   port: ${tidb_port}
        #   user: ${tidb_user}
        #   secretName: backup-demo1-tidb-secret
        br:
          cluster: demo1
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status-addr}
          # concurrency: 4
          # rateLimit: 0
          # checksum: true
        local:
          prefix: backup-nfs
          volume:
            name: nfs
            nfs:
              server: ${nfs_server_ip}
              path: /nfs
          volumeMount:
            name: nfs
            mountPath: /nfs
    ```

2. After creating the scheduled full backup, use the following command to check the backup status:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bks -n test1 -owide
    ```

    Use the following command to check all the backup items:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-nfs -n test1
    ```

From the example above, you can see that the `backupSchedule` configuration consists of two parts. One is the unique configuration of `backupSchedule`, and the other is `backupTemplate`. `backupTemplate` specifies the configuration related to NFS, which is the same as the configuration of the ad-hoc backup to NFS (refer to [Ad-hoc backup process](#process-of-ad-hoc-backup) for details).

<details>
<summary>The unique configuration items of <code>backupSchedule</code></summary>

- `.spec.maxBackups`: A backup retention policy, which determines the maximum number of backup items to be retained. When this value is exceeded, the outdated backup items will be deleted. If you set this configuration item to `0`, all backup items are retained.

- `.spec.maxReservedTime`: A backup retention policy based on time. For example, if you set the value of this configuration to `24h`, only backup items within the recent 24 hours are retained. All backup items out of this time are deleted. For the time format, refer to [`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration). If you have set the maximum number of backup items and the longest retention time of backup items at the same time, the latter setting takes effect.

- `.spec.schedule`: The time scheduling format of Cron. Refer to [Cron](https://en.wikipedia.org/wiki/Cron) for details.

- `.spec.pause`: `false` by default. If this parameter is set to `true`, the scheduled scheduling is paused. In this situation, the backup operation will not be performed even if the scheduling time is reached. During this pause, the backup [Garbage Collection](https://docs.pingcap.com/tidb/stable/garbage-collection-overview) (GC) runs normally. If you change `true` to `false`, the full backup process is restarted.

</details>

## Delete the backup CR

You can delete the backup CR (`Backup`) and the scheduled backup CR (`BackupSchedule`) by the following commands:

{{< copyable "shell-regular" >}}

```shell
kubectl delete backup ${name} -n ${namespace}
kubectl delete backupschedule ${name} -n ${namespace}
```

If you use TiDB Operator v1.1.2 or earlier versions, or if you use TiDB Operator v1.1.3 or later versions and set the value of `spec.cleanPolicy` to `Delete`, TiDB Operator deletes the backup data when it deletes the CR. In such cases, if you need to delete the namespace as well, it is recommended that you first delete all the `Backup`/`BackupSchedule` CR and then delete the namespace.

If you delete the namespace before you delete the `Backup`/`BackupSchedule` CR, TiDB Operator continues to create jobs to clean the backup data. However, since the namespace is in `Terminating` state, TiDB Operator fails to create a job, which causes the namespace to be stuck in the state.

To address this issue, delete `finalizers` using the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl edit backup ${name} -n ${namespace}
```

After deleting the `metadata.finalizers` configuration, you can delete CR normally.

## Troubleshooting

If you encounter any problem during the backup process, refer to [Common Deployment Failures](deploy-failures.md).
