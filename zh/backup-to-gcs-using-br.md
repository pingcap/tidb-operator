---
title: 使用 BR 备份 TiDB 集群到 GCS
summary: 介绍如何使用 BR 备份 TiDB 集群到 Google Cloud Storage (GCS)。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/backup-to-gcs-using-br/']
---

# 使用 BR 备份 TiDB 集群到 GCS

本文档详细描述了如何将 Kubernetes 上 TiDB 集群的数据备份到 [Google Cloud Storage (GCS)](https://cloud.google.com/storage/docs/) 上。[`BR`](https://docs.pingcap.com/zh/tidb/stable/backup-and-restore-tool) 会在底层获取集群的逻辑备份，然后再将备份数据上传到远端 GCS。

本文使用的备份方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。

## Ad-hoc 备份

Ad-hoc 备份支持全量备份与增量备份。Ad-hoc 备份通过创建一个自定义的 `Backup` custom resource (CR) 对象来描述一次备份。TiDB Operator 根据这个 `Backup` 对象来完成具体的备份过程。如果备份过程中出现错误，程序不会自动重试，此时需要手动处理。

为了更好地描述备份的使用方式，本文档提供如下备份示例。示例假设对部署在 Kubernetes `test1` 这个 namespace 中的 TiDB 集群 `demo1` 进行数据备份，下面是具体操作过程。

### Ad-hoc 备份环境准备

> **注意：**
>
> 如果使用 TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，不需要在 Backup CR 中配置 `spec.tikvGCLifeTime` 和 `spec.from` 字段，并且可以省略以下创建 `backup-demo1-tidb-secret` secret 的步骤和[数据库账户权限](#数据库账户权限)步骤。

1. 下载文件 [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test1` 这个 namespace 中创建备份需要的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. 远程存储访问授权。

    参考 [GCS 账号授权](grant-permissions-to-remote-storage.md#gcs-账号授权)授权访问 GCS 远程存储。

3. 创建 `backup-demo1-tidb-secret` secret。该 secret 存放用于访问 TiDB 集群的 root 账号和密钥。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=<password> --namespace=test1
    ```

### 数据库账户权限

* `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：备份前后，Backup CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间

### Ad-hoc 备份过程

1. 创建 `Backup` CR，并将数据备份到 GCS：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-gcs.yaml
    ```

    `backup-gcs.yaml` 文件内容如下：

    {{< copyable "" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-gcs
      namespace: test1
    spec:
      # backupType: full
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      from:
        host: ${tidb-host}
        port: ${tidb-port}
        user: ${tidb-user}
        secretName: backup-demo1-tidb-secret
      br:
        cluster: demo1
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status-addr}
        # concurrency: 4
        # rateLimit: 0
        # checksum: true
        # sendCredToTikv: true
        # options:
        # - --lastbackupts=420134118382108673
      gcs:
        projectId: ${project_id}
        secretName: gcs-secret
        bucket: ${bucket}
        prefix: ${prefix}
        # location: us-east1
        # storageClass: STANDARD_IA
        # objectAcl: private
    ```

    以上示例中，`spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`timeAgo`、`sendCredToTikv`。更多 `.spec.br` 字段的详细解释参考 [BR 字段介绍](backup-restore-overview.md#br-字段介绍)。

    自 v1.1.6 版本起，如果需要增量备份，只需要在 `spec.br.options` 中指定上一次的备份时间戳 `--lastbackupts` 即可。有关增量备份的限制，可参考[使用 BR 进行备份与恢复](https://docs.pingcap.com/zh/tidb/stable/backup-and-restore-tool#增量备份)。

    该示例将 TiDB 集群的数据全量导出备份到 GCS。`spec.gcs` 中的一些参数项均可省略，如 `location`、`objectAcl`、`storageClass`。GCS 存储相关配置参考 [GCS 存储字段介绍](backup-restore-overview.md#gcs-存储字段介绍)。

    更多 `Backup` CR 字段的详细解释参考 [Backup CR 字段介绍](backup-restore-overview.md#backup-cr-字段介绍)。

2. 创建好 `Backup` CR 后，可通过以下命令查看备份状态：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get bk -n test1 -owide
    ```

## 定时全量备份

用户通过设置备份策略来对 TiDB 集群进行定时备份，同时设置备份的保留策略以避免产生过多的备份。定时全量备份通过自定义的 `BackupSchedule` CR 对象来描述。每到备份时间点会触发一次全量备份，定时全量备份底层通过 Ad-hoc 全量备份来实现。下面是创建定时全量备份的具体步骤：

### 定时全量备份环境准备

同 [Ad-hoc 全量备份环境准备](#ad-hoc-备份环境准备)。

### 定时全量备份过程

1. 创建 `BackupSchedule` CR，开启 TiDB 集群的定时全量备份，将数据备份到 GCS：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-schedule-gcs.yaml
    ```

    `backup-schedule-gcs.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-gcs
      namespace: test1
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        br:
          cluster: demo1
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status-addr}
          # concurrency: 4
          # rateLimit: 0
          # checksum: true
          # sendCredToTikv: true
        gcs:
          secretName: gcs-secret
          projectId: ${project_id}
          bucket: ${bucket}
          prefix: ${prefix}
          # location: us-east1
          # storageClass: STANDARD_IA
          # objectAcl: private
    ```

2. 定时全量备份创建完成后，通过以下命令查看备份的状态：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get bks -n test1 -owide
    ```

    查看定时全量备份下面所有的备份条目：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-gcs -n test1
    ```

从以上示例可知，`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定集群及远程存储相关的配置，字段和 Backup CR 中的 `spec` 一样，详细介绍可参考 [Backup CR 字段介绍](backup-restore-overview.md#backup-cr-字段介绍)。`backupSchedule` 独有的配置项具体介绍可参考 [BackupSchedule CR 字段介绍](backup-restore-overview.md#backupschedule-cr-字段介绍)。

## 删除备份的 Backup CR

删除备份的 Backup CR 可参考[删除备份的 Backup CR](backup-restore-overview.md#删除备份的-backup-cr)。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
