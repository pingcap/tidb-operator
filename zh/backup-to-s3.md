---
title: 使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储
summary: 介绍如何使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储。
category: how-to
---

# 使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储

本文详细描述了如何将 Kubernetes 上的 TiDB 集群数据备份到兼容 S3 的存储上。本文档中的“备份”，均是指全量备份（Ad-hoc 全量备份和定时全量备份）。底层通过使用 [`Dumpling`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview) 获取集群的逻辑备份，然后在将备份数据上传到兼容 S3 的存储上。

本文使用的备份方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。基于 Helm Charts 实现的备份和恢复方式可参考[基于 Helm Charts 实现的 TiDB 集群备份与恢复](backup-and-restore-using-helm-charts.md)。

## Ad-hoc 全量备份

Ad-hoc 全量备份通过创建一个自定义的 `Backup` custom resource (CR) 对象来描述一次备份。TiDB Operator 根据这个 `Backup` 对象来完成具体的备份过程。如果备份过程中出现错误，程序不会自动重试，此时需要手动处理。

目前兼容 S3 的存储中，Ceph 和 Amazon S3 经测试可正常工作。下文对 Ceph 和 Amazon S3 这两种存储的使用进行描述。本文档提供如下备份示例。示例假设对部署在 Kubernetes `test1` 这个 namespace 中的 TiDB 集群 `demo1` 进行数据备份，下面是具体操作过程。

### AWS 账号的三种权限授予方式

如果使用 Amazon S3 来备份恢复集群，可以使用三种权限授予方式授予权限，参考[使用 BR 工具备份 AWS 上的 TiDB 集群](backup-to-aws-s3-using-br.md#aws-账号权限授予的三种方式)，使用 Ceph 作为后端存储测试备份恢复时，是通过 AccessKey 和 SecretKey 模式授权。

### Ad-hoc 全量备份环境准备

参考 [Ad-hoc 全量备份环境准备](backup-to-aws-s3-using-br.md#ad-hoc-全量备份环境准备)

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
      namespace: test1
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
      storageClassName: local-storage
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
      namespace: test1
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
      storageClassName: local-storage
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
      namespace: test1
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
      storageClassName: local-storage
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
      namespace: test1
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
      storageClassName: local-storage
      storageSize: 10Gi
    ```

上述示例将 TiDB 集群的数据全量导出备份到 Amazon S3 和 Ceph 上。Amazon S3 的 `acl`、`endpoint`、`storageClass` 配置项均可以省略。其余非 Amazon S3 的但是兼容 S3 的存储均可使用和 Amazon S3 类似的配置。可参考上面例子中 Ceph 的配置，省略不需要配置的字段。

Amazon S3 支持以下几种 access-control list (ACL) 策略：

* `private`
* `public-read`
* `public-read-write`
* `authenticated-read`
* `bucket-owner-read`
* `bucket-owner-full-control`

如果不设置 ACL 策略，则默认使用 `private` 策略。这几种访问控制策略的详细介绍参考 [AWS 官方文档](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html)。

Amazon S3 支持以下几种 `storageClass` 类型：

* `STANDARD`
* `REDUCED_REDUNDANCY`
* `STANDARD_IA`
* `ONEZONE_IA`
* `GLACIER`
* `DEEP_ARCHIVE`

如果不设置 `storageClass`，则默认使用 `STANDARD_IA`。这几种存储类型的详细介绍参考 [AWS 官方文档](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html)。

创建好 `Backup` CR 后，可通过如下命令查看备份状态：

{{< copyable "shell-regular" >}}

 ```shell
 kubectl get bk -n test1 -owide
 ```

更多 `Backup` CR 字段的详细解释:

* `.spec.metadata.namespace`：`Backup` CR 所在的 namespace。
* `.spec.cleanData`：设置为 true 时删除该 Backup CR 时会同时清除该 CR 备份出的数据，默认为 false。值得注意的是，在 v1.1.2 以及之前版本不存在该字段，且默认在删除 CR 的同时删除备份的文件。若 v1.1.3 及之后版本的用户希望保持该行为，需要设置该字段为 true。
* `.spec.from.host`：待备份 TiDB 集群的访问地址，为需要导出的 TiDB 的 service name，例如 `basic-tidb`。
* `.spec.from.port`：待备份 TiDB 集群的访问端口。
* `.spec.from.user`：待备份 TiDB 集群的访问用户。
* `.spec.from.secretName`：存储 `.spec.from.user` 用户的密码的 secret。
* `.spec.s3.region`：使用 Amazon S3 存储备份，需要配置 Amazon S3 所在的 region。
* `.spec.s3.bucket`：兼容 S3 存储的 bucket 名字。
* `.spec.s3.prefix`：这个字段可以省略，如果设置了这个字段，则会使用这个字段来拼接在远端存储的存储路径 `s3://${.spec.s3.bucket}/${.spec.s3.prefix}/backupName`。
* `.spec.dumpling`：Dumpling 相关的配置，主要有两个字段：一个是 `options` 字段，里面可以指定 Dumpling 的运行参数，详情见 [Dumpling 使用文档](https://docs.pingcap.com/zh/tidb/dev/dumpling-overview#dumpling-主要参数表)；一个是 `tableFilter` 字段，可以指定让 Dumpling 备份符合 [table-filter 规则](https://docs.pingcap.com/zh/tidb/stable/table-filter/) 的表。默认情况下 dumpling 这个字段可以不用配置。当不指定 dumpling 的配置时，`options` 和 `tableFilter` 字段的默认值如下：

    ```
    options:
    - --threads=16
    - --rows=10000
    tableFilter:
    - "*.*"
    - "!/^(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA)$/.*"
    ```

    > **注意：**
    >
    > tableFilter 如果要写排除规则导出除 db.table 的所有表 "!db.table" 必须先添加 `*.*` 规则来导出所有表，如下面例子所示：

    ```
    tableFilter:
    - "*.*"
    - "!db.table"
    ```

* `.spec.storageClassName`：备份时所需的 persistent volume (PV) 类型。
* `.spec.storageSize`：备份时指定所需的 PV 大小，默认为 100 Gi。该值应大于备份 TiDB 集群数据的大小。一个 TiDB 集群的 Backup CR 对应的 PVC 名字是确定的，如果集群命名空间中已存在该 PVC 并且其大小小于 `.spec.storageSize`，这时需要先删除该 PVC 再运行 Backup job。

更多支持的兼容 S3 的 `provider` 如下：

* `alibaba`：Alibaba Cloud Object Storage System (OSS) formerly Aliyun
* `digitalocean`：Digital Ocean Spaces
* `dreamhost`：Dreamhost DreamObjects
* `ibmcos`：IBM COS S3
* `minio`：Minio Object Storage
* `netease`：Netease Object Storage (NOS)
* `wasabi`：Wasabi Object Storage
* `other`：Any other S3 compatible provider

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
      namespace: test1
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
        storageClassName: local-storage
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
      namespace: test1
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
        storageClassName: local-storage
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
      namespace: test1
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
        storageClassName: local-storage
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
      namespace: test1
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
        storageClassName: local-storage
        storageSize: 10Gi

定时全量备份创建完成后，可以通过以下命令查看定时全量备份的状态：

{{< copyable "shell-regular" >}}

```shell
kubectl get bks -n test1 -owide
```

查看定时全量备份下面所有的备份条目：

{{< copyable "shell-regular" >}}

```shell
kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-s3 -n test1
```

从以上示例可知，`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定 S3 兼容存储相关的配置，该配置与 Ad-hoc 全量备份到兼容 S3 的存储配置完全一样，可参考[备份数据到兼容 S3 的存储](#备份数据到兼容-s3-的存储)。下面介绍 `backupSchedule` 独有的配置项：

+ `.spec.maxBackups`：一种备份保留策略，决定定时备份最多可保留的备份个数。超过该数目，就会将过时的备份删除。如果将该项设置为 `0`，则表示保留所有备份。
+ `.spec.maxReservedTime`：一种备份保留策略，按时间保留备份。例如将该参数设置为 `24h`，表示只保留最近 24 小时内的备份条目。超过这个时间的备份都会被清除。时间设置格式参考 [`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration)。如果同时设置最大备份保留个数和最长备份保留时间，则以最长备份保留时间为准。
+ `.spec.schedule`：Cron 的时间调度格式。具体格式可参考 [Cron](https://en.wikipedia.org/wiki/Cron)。
+ `.spec.pause`：该值默认为 `false`。如果将该值设置为 `true`，表示暂停定时调度。此时即使到了调度时间点，也不会进行备份。在定时备份暂停期间，备份 Garbage Collection (GC) 仍然正常进行。将 `true` 改为 `false` 则重新开启定时全量备份。

> **注意：**
>
> TiDB Operator 会创建一个 PVC，这个 PVC 同时用于 Ad-hoc 全量备份和定时全量备份，备份数据会先存储到 PV，然后再上传到远端存储。如果备份完成后想要删掉这个 PVC，可以参考[删除资源](cheat-sheet.md#删除资源)先把备份 Pod 删掉，然后再把 PVC 删掉。
