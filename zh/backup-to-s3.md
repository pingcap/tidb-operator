---
title: 使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储
summary: 介绍如何使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储。
category: how-to
aliases: ['/docs-cn/tidb-in-kubernetes/dev/backup-to-s3/']
---

# 使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储

本文详细描述了如何将 Kubernetes 上的 TiDB 集群数据备份到兼容 S3 的存储上。本文档中的“备份”，均是指全量备份（Ad-hoc 全量备份和定时全量备份）。底层通过使用 [`Dumpling`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview) 获取集群的逻辑备份，然后在将备份数据上传到兼容 S3 的存储上。

本文使用的备份方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。

## Ad-hoc 全量备份

Ad-hoc 全量备份通过创建一个自定义的 `Backup` custom resource (CR) 对象来描述一次备份。TiDB Operator 根据这个 `Backup` 对象来完成具体的备份过程。如果备份过程中出现错误，程序不会自动重试，此时需要手动处理。

目前兼容 S3 的存储中，Ceph 和 Amazon S3 经测试可正常工作。下文提供了如何将 TiDB 集群的数据备份到 Ceph 和 Amazon S3 这两种存储的示例。示例假设对部署在 Kubernetes `tidb-cluster` 这个 namespace 中的 TiDB 集群 `demo1` 进行数据备份，以下是具体的操作过程。

### Ad-hoc 全量备份环境准备

1. 执行以下命令，根据 [backup-rbac.yaml](https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/backup/backup-rbac.yaml) 在 `tidb-cluster` 命名空间创建基于角色的访问控制 (RBAC) 资源。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/backup/backup-rbac.yaml -n tidb-cluster
    ```

2. 远程存储访问授权。

    如果使用 Amazon S3 来备份集群，可以使用三种权限授予方式授予权限，参考 [AWS 账号授权](grant-permissions-to-remote-storage.md#aws-账号授权)授权访问兼容 S3 的远程存储；使用 Ceph 作为后端存储测试备份时，是通过 AccessKey 和 SecretKey 模式授权，设置方式可参考[通过 AccessKey 和 SecretKey 授权](grant-permissions-to-remote-storage.md#通过-accesskey-和-secretkey-授权)。

3. 创建 `backup-demo1-tidb-secret` secret。该 secret 存放用于访问 TiDB 集群的 root 账号和密钥。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=${password} --namespace=tidb-cluster
    ```

### 数据库账户权限

* `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：备份前后，Backup CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间。
* 全局权限：`SELECT`、`RELOAD`、`LOCK TABLES`、和 `REPLICATION CLIENT`。

以下是如何创建一个备份用户的示例:

```sql
CREATE USER 'backup'@'%' IDENTIFIED BY '...';
GRANT
  SELECT, RELOAD, LOCK TABLES, REPLICATION CLIENT
  ON *.*
  TO 'backup'@'%';
GRANT
  UPDATE, SELECT
  ON mysql.tidb
  TO 'backup'@'%';
```

### 备份数据到兼容 S3 的存储

> **注意：**
>
> 由于 `rclone` 存在[问题](https://rclone.org/s3/#key-management-system-kms)，如果使用 Amazon S3 存储备份，并且 Amazon S3 开启了 `AWS-KMS` 加密，需要在本节示例中的 yaml 文件里添加如下 `spec.s3.options` 配置以保证备份成功：
>
> ```yaml
> spec:
>   ...
>   s3:
>     ...
>     options:
>     - --ignore-checksum
> ```

> **注意：**
>
> 如下所示，本节提供了存储访问的多种方法。只需使用符合你情况的方法即可。
> 
> - 通过导入 AccessKey 和 SecretKey 备份到 Amazon S3 的方法
> - 通过导入 AccessKey 和 SecretKey 备份到 Ceph 的方法
> - 通过绑定 IAM 与 Pod 的方式备份到 Amazon S3 的方法
> - 通过绑定 IAM 与 ServiceAccount 的方式备份到 Amazon S3 的方法

+ 创建 `Backup` CR，通过 AccessKey 和 SecretKey 授权的方式将数据备份到 Amazon S3：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-s3.yaml
    ```

    `backup-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: tidb-cluster
    spec:
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        secretName: s3-secret
        region: ${region}
        bucket: ${bucket}
        # prefix: ${prefix}
        # storageClass: STANDARD_IA
        # acl: private
        # endpoint:
    # dumpling:
    #  options:
    #  - --threads=16
    #  - --rows=10000
    #  tableFilter:
    #  - "test.*"
      # storageClassName: local-storage
      storageSize: 10Gi
    ```

+ 创建 `Backup` CR，通过 AccessKey 和 SecretKey 授权的方式将数据备份到 Ceph：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-s3.yaml
    ```

    `backup-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: tidb-cluster
    spec:
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: ceph
        secretName: s3-secret
        endpoint: ${endpoint}
        # prefix: ${prefix}
        bucket: ${bucket}
    # dumpling:
    #  options:
    #  - --threads=16
    #  - --rows=10000
    #  tableFilter:
    #  - "test.*"
      # storageClassName: local-storage
      storageSize: 10Gi
    ```

+ 创建 `Backup` CR，通过 IAM 绑定 Pod 授权的方式将数据备份到 Amazon S3：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-s3.yaml
    ```

    `backup-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: tidb-cluster
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      backupType: full
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        region: ${region}
        bucket: ${bucket}
        # prefix: ${prefix}
        # storageClass: STANDARD_IA
        # acl: private
        # endpoint:
    # dumpling:
    #  options:
    #  - --threads=16
    #  - --rows=10000
    #  tableFilter:
    #  - "test.*"
      # storageClassName: local-storage
      storageSize: 10Gi
    ```

+ 创建 `Backup` CR，通过 IAM 绑定 ServiceAccount 授权的方式将数据备份到 Amazon S3：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-s3.yaml
    ```

    `backup-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: tidb-cluster
    spec:
      backupType: full
      serviceAccount: tidb-backup-manager
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        region: ${region}
        bucket: ${bucket}
        # prefix: ${prefix}
        # storageClass: STANDARD_IA
        # acl: private
        # endpoint:
    # dumpling:
    #  options:
    #  - --threads=16
    #  - --rows=10000
    #  tableFilter:
    #  - "test.*"
      # storageClassName: local-storage
      storageSize: 10Gi
    ```

上述示例将 TiDB 集群的数据全量导出备份到 Amazon S3 和 Ceph 上。Amazon S3 的 `acl`、`endpoint`、`storageClass` 配置项均可以省略。其余非 Amazon S3 的但是兼容 S3 的存储均可使用和 Amazon S3 类似的配置。可参考上面例子中 Ceph 的配置，省略不需要配置的字段。更多兼容 S3 的存储相关配置参考 [S3 存储字段介绍](backup-restore-overview.md#s3-存储字段介绍)。

以上示例中，`.spec.dumpling` 表示 Dumpling 相关的配置，可以在 `options` 字段指定 Dumpling 的运行参数，详情见 [Dumpling 使用文档](https://docs.pingcap.com/zh/tidb/dev/dumpling-overview#dumpling-主要参数表)；默认情况下该字段可以不用配置。当不指定 Dumpling 的配置时，`options` 字段的默认值如下：

```
options:
- --threads=16
- --rows=10000
```

更多 `Backup` CR 字段的详细解释参考 [Backup CR 字段介绍](backup-restore-overview.md#backup-cr-字段介绍)。

创建好 `Backup` CR 后，可通过如下命令查看备份状态：

{{< copyable "shell-regular" >}}

```shell
kubectl get bk -n tidb-cluster -owide
```

要获取一个 Backup job 的详细信息，请使用以下命令。对于此命令中的 `$backup_job_name`，请使用上一条命令输出中的名称。

{{< copyable "shell-regular" >}}

```shell
kubectl describe bk -n tidb-cluster $backup_job_name
```

如果要再次运行 Ad-hoc 备份，你需要[删除备份的 Backup CR](backup-restore-overview.md#删除备份的-backup-cr) 并重新创建。

## 定时全量备份

用户通过设置备份策略来对 TiDB 集群进行定时备份，同时设置备份的保留策略以避免产生过多的备份。定时全量备份通过自定义的 `BackupSchedule` CR 对象来描述。每到备份时间点会触发一次全量备份，定时全量备份底层通过 Ad-hoc 全量备份来实现。下面是创建定时全量备份的具体步骤：

### 定时全量备份环境准备

同 [Ad-hoc 全量备份环境准备](#ad-hoc-全量备份环境准备)。

### 定时全量备份数据到 S3 兼容存储

> **注意：**
>
> 由于 `rclone` 存在[问题](https://rclone.org/s3/#key-management-system-kms)，如果使用 Amazon S3 存储备份，并且 Amazon S3 开启了 `AWS-KMS` 加密，需要在本节示例中的 yaml 文件里添加如下 `spec.backupTemplate.s3.options` 配置以保证备份成功：
>
> ```yaml
> spec:
>   ...
>   backupTemplate:
>     ...
>     s3:
>       ...
>       options:
>       - --ignore-checksum
> ```

+ 创建 `BackupSchedule` CR 开启 TiDB 集群的定时全量备份，通过 AccessKey 和 SecretKey 授权的方式将数据备份到 Amazon S3：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-schedule-s3.yaml
    ```

    `backup-schedule-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: tidb-cluster
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          secretName: s3-secret
          region: ${region}
          bucket: ${bucket}
          # prefix: ${prefix}
          # storageClass: STANDARD_IA
          # acl: private
          # endpoint:
      # dumpling:
      #  options:
      #  - --threads=16
      #  - --rows=10000
      #  tableFilter:
      #  - "test.*"
        # storageClassName: local-storage
        storageSize: 10Gi
    ```

+ 创建 `BackupSchedule` CR 开启 TiDB 集群的定时全量备份，通过 AccessKey 和 SecretKey 授权的方式将数据备份到 Ceph：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-schedule-s3.yaml
    ```

    `backup-schedule-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-ceph
      namespace: tidb-cluster
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: ceph
          secretName: s3-secret
          endpoint: ${endpoint}
          bucket: ${bucket}
          # prefix: ${prefix}
      # dumpling:
      #  options:
      #  - --threads=16
      #  - --rows=10000
      #  tableFilter:
      #  - "test.*"
        # storageClassName: local-storage
        storageSize: 10Gi
    ```

+ 创建 `BackupSchedule` CR 开启 TiDB 集群的定时全量备份，通过 IAM 绑定 Pod 授权的方式将数据备份到 Amazon S3：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-schedule-s3.yaml
    ```

    `backup-schedule-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: tidb-cluster
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          region: ${region}
          bucket: ${bucket}
          # prefix: ${prefix}
          # storageClass: STANDARD_IA
          # acl: private
          # endpoint:
      # dumpling:
      #  options:
      #  - --threads=16
      #  - --rows=10000
      #  tableFilter:
      #  - "test.*"
        # storageClassName: local-storage
        storageSize: 10Gi
    ```

+ 创建 `BackupSchedule` CR 开启 TiDB 集群的定时全量备份，通过 IAM 绑定 ServiceAccount 授权的方式将数据备份到 Amazon S3：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-schedule-s3.yaml
    ```

    `backup-schedule-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: tidb-cluster
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      serviceAccount: tidb-backup-manager
      backupTemplate:
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          region: ${region}
          bucket: ${bucket}
          # prefix: ${prefix}
          # storageClass: STANDARD_IA
          # acl: private
          # endpoint:
      # dumpling:
      #  options:
      #  - --threads=16
      #  - --rows=10000
      #  tableFilter:
      #  - "test.*"
        # storageClassName: local-storage
        storageSize: 10Gi
    ```

定时全量备份创建完成后，可以通过以下命令查看定时全量备份的状态：

{{< copyable "shell-regular" >}}

```shell
kubectl get bks -n tidb-cluster -owide
```

查看定时全量备份下面所有的备份条目：

{{< copyable "shell-regular" >}}

```shell
kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-s3 -n tidb-cluster
```

从以上示例可知，`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定集群及远程存储相关的配置，字段和 Backup CR 中的 `spec` 一样，详细介绍可参考 [Backup CR 字段介绍](backup-restore-overview.md#backup-cr-字段介绍)。`backupSchedule` 独有配置项介绍可参考 [BackupSchedule CR 字段介绍](backup-restore-overview.md#backupschedule-cr-字段介绍)。

> **注意：**
>
> TiDB Operator 会创建一个 PVC，这个 PVC 同时用于 Ad-hoc 全量备份和定时全量备份，备份数据会先存储到 PV，然后再上传到远端存储。如果备份完成后想要删掉这个 PVC，可以参考[删除资源](cheat-sheet.md#删除资源)先把备份 Pod 删掉，然后再把 PVC 删掉。
>
> 假如备份并上传到远端存储成功，TiDB Operator 会自动删除本地的备份文件。如果上传失败，则本地备份文件将被保留。

## 删除备份的 Backup CR

删除备份的 Backup CR 可参考[删除备份的 Backup CR](backup-restore-overview.md#删除备份的-backup-cr)。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
