---
title: 备份与恢复简介
summary: 介绍如何使用 BR、Dumpling、TiDB Lightning 工具对 Kubernetes 上的 TiDB 集群进行数据备份和数据恢复。
---

# 备份与恢复简介

本文档介绍如何使用 [BR](https://docs.pingcap.com/zh/tidb/stable/backup-and-restore-tool)、[Dumpling](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview)、[TiDB Lightning](https://docs.pingcap.com/zh/tidb/stable/get-started-with-tidb-lightning) 对 Kubernetes 上的 TiDB 集群进行数据备份和数据恢复。

TiDB Operator 1.1 及以上版本推荐使用基于 CustomResourceDefinition (CRD) 实现的备份恢复方式实现：

+ 如果 TiDB 集群版本 >= v3.1，可以参考以下文档：

    - [使用 BR 备份 TiDB 集群到兼容 S3 的存储](backup-to-aws-s3-using-br.md)
    - [使用 BR 备份 TiDB 集群到 GCS](backup-to-gcs-using-br.md)
    - [使用 BR 备份 TiDB 集群到持久卷](backup-to-pv-using-br.md)
    - [使用 BR 恢复兼容 S3 的存储上的备份数据](restore-from-aws-s3-using-br.md)
    - [使用 BR 恢复 GCS 上的备份数据](restore-from-gcs-using-br.md)
    - [使用 BR 恢复持久卷上的备份数据](restore-from-pv-using-br.md)

+ 如果 TiDB 集群版本 < v3.1，可以参考以下文档：

    - [使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储](backup-to-s3.md)
    - [使用 Dumpling 备份 TiDB 集群数据到 GCS](backup-to-gcs.md)
    - [使用 TiDB Lightning 恢复兼容 S3 的存储上的备份数据](restore-from-s3.md)
    - [使用 TiDB Lightning 恢复 GCS 上的备份数据](restore-from-gcs.md)

## 使用场景

[Dumpling](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview) 是一个数据导出工具，该工具可以把存储在 TiDB/MySQL 中的数据导出为 SQL 或者 CSV 格式，可以用于完成逻辑上的全量备份或者导出。如果需要直接备份 SST 文件（键值对）或者对延迟不敏感的增量备份，请参阅 [BR](https://docs.pingcap.com/zh/tidb/stable/backup-and-restore-tool)。如果需要实时的增量备份，请参阅 [TiCDC](https://docs.pingcap.com/zh/tidb/stable/ticdc-overview)。

[TiDB Lightning](https://docs.pingcap.com/zh/tidb/stable/get-started-with-tidb-lightning) 是一个将全量数据高速导入到 TiDB 集群的工具，TiDB Lightning 有以下两个主要的使用场景：一是大量新数据的快速导入；二是全量备份数据的恢复。目前，TiDB Lightning 支持 Dumpling 或 CSV 输出格式的数据源。你可以在以下两种场景下使用 TiDB Lightning：

- 迅速导入大量新数据。
- 恢复所有备份数据。

[BR](https://docs.pingcap.com/zh/tidb/stable/backup-and-restore-tool) 是 TiDB 分布式备份恢复的命令行工具，用于对 TiDB 集群进行数据备份和恢复。相比 Dumpling 和 Mydumper，BR 更适合大数据量的场景，BR 只支持 TiDB v3.1 及以上版本。

## Backup CR 字段介绍

为了对 Kubernetes 上的 TiDB 集群进行数据备份，用户可以通过创建一个自定义的 `Backup` Custom Resource (CR) 对象来描述一次备份，具体备份过程可参考 [备份与恢复简介](#备份与恢复简介)中列出的文档。以下介绍 Backup CR 各个字段的具体含义。

### 通用字段介绍

* `.spec.metadata.namespace`：`Backup` CR 所在的 namespace。
* `.spec.toolImage`：用于指定 `Backup` 使用的工具镜像。

    - 使用 BR 备份时，可以用该字段指定 BR 的版本:

        - 如果未指定或者为空，默认使用镜像 `pingcap/br:${tikv_version}` 进行备份。
        - 如果指定了 BR 的版本，例如 `.spec.toolImage: pingcap/br:v5.2.1`，那么使用指定的版本镜像进行备份。
        - 如果指定了镜像但未指定版本，例如 `.spec.toolImage: private/registry/br`，那么使用镜像 `private/registry/br:${tikv_version}` 进行备份。
    - 使用 Dumpling 备份时，可以用该字段指定 Dumpling 的版本，例如， `spec.toolImage: pingcap/dumpling:v5.2.1`。如果不指定，默认使用 [Backup Manager Dockerfile](https://github.com/pingcap/tidb-operator/blob/master/images/tidb-backup-manager/Dockerfile) 文件中 `TOOLKIT_VERSION` 指定的 Dumpling 版本进行备份。           
    - TiDB Operator 从 v1.1.9 版本起支持这项配置。
* `.spec.tikvGCLifeTime`：备份中的临时 `tikv_gc_life_time` 时间设置，默认为 72h。

    在备份开始之前，若 TiDB 集群的 `tikv_gc_life_time` 小于用户设置的 `spec.tikvGCLifeTime`，为了保证备份的数据不被 TiKV GC 掉，TiDB Operator 会在备份前[调节 `tikv_gc_life_time`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview#导出大规模数据时的-tidb-gc-设置) 为 `spec.tikvGCLifeTime`。

    备份结束后不论成功或者失败，只要老的 `tikv_gc_life_time` 比设置的 `.spec.tikvGCLifeTime` 小，TiDB Operator 都会尝试恢复 `tikv_gc_life_time` 为备份前的值。在极端情况下，TiDB Operator 访问数据库失败会导致 TiDB Operator 无法自动恢复 `tikv_gc_life_time` 并认为备份失败。

    此时，可以通过下述语句查看当前 TiDB 集群的 `tikv_gc_life_time`：

    ```sql
    select VARIABLE_NAME, VARIABLE_VALUE from mysql.tidb where VARIABLE_NAME like "tikv_gc_life_time";
    ```

    如果发现 `tikv_gc_life_time` 值过大（通常为 10m），则需要按照[调节 tikv_gc_life_time](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview#导出大规模数据时的-tidb-gc-设置) 将 `tikv_gc_life_time` 调回原样。

* `.spec.cleanPolicy`：备份集群后删除备份 CR 时的备份文件清理策略。目前支持三种清理策略：

    * `Retain`：任何情况下，删除备份 CR 时会保留备份出的文件
    * `Delete`：任何情况下，删除备份 CR 时会删除备份出的文件
    * `OnFailure`：如果备份中失败，删除备份 CR 时会删除备份出的文件

    如果不配置该字段，或者配置该字段的值为上述三种以外的值，均会保留备份出的文件。值得注意的是，在 v1.1.2 以及之前版本不存在该字段，且默认在删除 CR 的同时删除备份的文件。若 v1.1.3 及之后版本的用户希望保持该行为，需要设置该字段为 `Delete`。

* `.spec.from.host`：待备份 TiDB 集群的访问地址，为需要导出的 TiDB 的 service name，例如 `basic-tidb`。
* `.spec.from.port`：待备份 TiDB 集群的访问端口。
* `.spec.from.user`：待备份 TiDB 集群的访问用户。
* `.spec.from.secretName`：存储 `.spec.from.user` 用户的密码的 Secret。
* `.spec.from.tlsClientSecretName`：指定备份使用的存储证书的 Secret。

    如果 TiDB 集群开启了 [TLS](enable-tls-between-components.md)，但是不想使用[文档](enable-tls-between-components.md)中创建的 `${cluster_name}-cluster-client-secret` 进行备份，可以通过这个参数为备份指定一个 Secret，可以通过如下命令生成：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

* `.spec.storageClassName`：备份时所需的 persistent volume (PV) 类型。
* `.spec.storageSize`：备份时指定所需的 PV 大小，默认为 100 Gi。该值应大于备份 TiDB 集群数据的大小。一个 TiDB 集群的 Backup CR 对应的 PVC 名字是确定的，如果集群命名空间中已存在该 PVC 并且其大小小于 `.spec.storageSize`，这时需要先删除该 PVC 再运行 Backup job。
* `.spec.tableFilter`：备份时指定让 Dumpling 或者 BR 备份符合 [table-filter 规则](https://docs.pingcap.com/zh/tidb/stable/table-filter/)的表。默认情况下该字段可以不用配置。

    当不配置时，如果使用 Dumpling 备份，`tableFilter` 字段的默认值如下：

    ```bash
    tableFilter:
    - "*.*"
    - "!/^(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA)$/.*"
    ```

    如果使用 BR 备份，BR 会备份除系统库以外的所有数据库。

    > **注意：**
    >
    > 如果要使用排除规则 `"!db.table"` 导出除 `db.table` 的所有表，那么在 `"!db.table"` 前必须先添加 `*.*` 规则。如下面例子所示：
    >
    > ```
    > tableFilter:
    > - "*.*"
    > - "!db.table"
    > ```

### BR 字段介绍

* `.spec.br.cluster`：代表需要备份的集群名字。
* `.spec.br.clusterNamespace`：代表需要备份的集群所在的 `namespace`。
* `.spec.br.logLevel`：代表日志的级别。默认为 `info`。
* `.spec.br.statusAddr`：为 BR 进程监听一个进程状态的 HTTP 端口，方便用户调试。如果不填，则默认不监听。
* `.spec.br.concurrency`：备份时每一个 TiKV 进程使用的线程数。备份时默认为 4，恢复时默认为 128。
* `.spec.br.rateLimit`：是否对流量进行限制。单位为 MB/s，例如设置为 `4` 代表限速 4 MB/s，默认不限速。
* `.spec.br.checksum`：是否在备份结束之后对文件进行验证。默认为 `true`。
* `.spec.br.timeAgo`：备份 timeAgo 以前的数据，默认为空（备份当前数据），[支持](https://golang.org/pkg/time/#ParseDuration) "1.5h"，"2h45m" 等数据。
* `.spec.br.sendCredToTikv`：BR 进程是否将自己的 AWS 权限 或者 GCP 权限传输给 TiKV 进程。默认为 `true`。
* `.spec.br.options`：BR 工具支持的额外参数，需要以字符串数组的形式传入。自 v1.1.6 版本起支持该参数。可用于指定 `lastbackupts` 以进行增量备份。

### S3 存储字段介绍

* `.spec.s3.provider`：支持的兼容 S3 的 `provider`。

    更多支持的兼容 S3 的 `provider` 如下：

    * `alibaba`：Alibaba Cloud Object Storage System (OSS)，formerly Aliyun
    * `digitalocean`：Digital Ocean Spaces
    * `dreamhost`：Dreamhost DreamObjects
    * `ibmcos`：IBM COS S3
    * `minio`：Minio Object Storage
    * `netease`：Netease Object Storage (NOS)
    * `wasabi`：Wasabi Object Storage
    * `other`：Any other S3 compatible provider

* `.spec.s3.region`：使用 Amazon S3 存储备份，需要配置 Amazon S3 所在的 region。
* `.spec.s3.bucket`：兼容 S3 存储的 bucket 名字。
* `.spec.s3.prefix`：如果设置了这个字段，则会使用这个字段来拼接在远端存储的存储路径 `s3://${.spec.s3.bucket}/${.spec.s3.prefix}/backupName`。
* `.spec.s3.acl`：支持的 access-control list (ACL) 策略。

    Amazon S3 支持以下几种 access-control list (ACL) 策略：

    * `private`
    * `public-read`
    * `public-read-write`
    * `authenticated-read`
    * `bucket-owner-read`
    * `bucket-owner-full-control`

    如果不设置 ACL 策略，则默认使用 `private` 策略。这几种访问控制策略的详细介绍参考 [AWS 官方文档](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html)。

* `.spec.s3.storageClass`：支持的 `storageClass` 类型。

    Amazon S3 支持以下几种 `storageClass` 类型：

    * `STANDARD`
    * `REDUCED_REDUNDANCY`
    * `STANDARD_IA`
    * `ONEZONE_IA`
    * `GLACIER`
    * `DEEP_ARCHIVE`

    如果不设置 `storageClass`，则默认使用 `STANDARD_IA`。这几种存储类型的详细介绍参考 [AWS 官方文档](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html)。

### GCS 存储字段介绍

* `.spec.gcs.projectId`：代表 GCP 上用户项目的唯一标识。具体获取该标识的方法可参考 [GCP 官方文档](https://cloud.google.com/resource-manager/docs/creating-managing-projects)。
* `.spec.gcs.bucket`：存储数据的 bucket 名字。
* `.spec.gcs.prefix`：如果设置了这个字段，则会使用这个字段来拼接在远端存储的存储路径 `gcs://${.spec.gcs.bucket}/${.spec.gcs.prefix}/backupName`。
* `spec.gcs.storageClass`：GCS 支持以下几种 `storageClass` 类型：

    * `MULTI_REGIONAL`
    * `REGIONAL`
    * `NEARLINE`
    * `COLDLINE`
    * `DURABLE_REDUCED_AVAILABILITY`

    如果不设置 `storageClass`，则默认使用 `COLDLINE`。这几种存储类型的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/storage-classes)。

* `spec.gcs.objectAcl`：设置 object access-control list (ACL) 策略。

    GCS 支持以下几种 ACL 策略：

    * `authenticatedRead`
    * `bucketOwnerFullControl`
    * `bucketOwnerRead`
    * `private`
    * `projectPrivate`
    * `publicRead`

    如果不设置 object ACL 策略，则默认使用 `private` 策略。这几种访问控制策略的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/access-control/lists)。

* `spec.gcs.bucketAcl`：设置 bucket access-control list (ACL) 策略。

    GCS 支持以下几种 bucket ACL 策略：

    * `authenticatedRead`
    * `private`
    * `projectPrivate`
    * `publicRead`
    * `publicReadWrite`

    如果不设置 bucket ACL 策略，则默认策略为 `private`。这几种访问控制策略的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/access-control/lists)。

### Local 存储字段介绍

* `.spec.local.prefix`：持久卷存储目录。如果设置了这个字段，则会使用这个字段来拼接在持久卷的存储路径 `local://${.spec.local.volumeMount.mountPath}/${.spec.local.prefix}/`。
* `.spec.local.volume`：持久卷配置。
* `.spec.local.volumeMount`：持久卷挂载配置。

## Restore CR 字段介绍

为了对 Kubernetes 上的 TiDB 集群进行数据恢复，用户可以通过创建一个自定义的 `Restore` Custom Resource (CR) 对象来描述一次恢复，具体恢复过程可参考 [备份与恢复简介](#备份与恢复简介)中列出的文档。以下介绍 Restore CR 各个字段的具体含义。

* `.spec.metadata.namespace`：`Restore` CR 所在的 namespace。
* `.spec.toolImage`：用于指定 `Restore` 使用的工具镜像。
    - 使用 BR 恢复时，可以用该字段指定 BR 的版本。例如，`spec.toolImage: pingcap/br:v5.2.1`。如果不指定，默认使用 `pingcap/br:${tikv_version}` 进行恢复。
    - 使用 Lightning 恢复时，可以用该字段指定 Lightning 的版本，例如`spec.toolImage: pingcap/lightning:v5.2.1`。如果不指定，默认使用 [Backup Manager Dockerfile](https://github.com/pingcap/tidb-operator/blob/master/images/tidb-backup-manager/Dockerfile) 文件中 `TOOLKIT_VERSION` 指定的 Lightning 版本进行恢复。
    - TiDB Operator 从 v1.1.9 版本起支持这项配置。
* `.spec.to.host`：待恢复 TiDB 集群的访问地址。
* `.spec.to.port`：待恢复 TiDB 集群的访问端口。
* `.spec.to.user`：待恢复 TiDB 集群的访问用户。
* `.spec.to.secretName`：存储 `.spec.to.user` 用户的密码的 secret。
* `.spec.to.tlsClientSecretName`：指定恢复备份使用的存储证书的 Secret。

    如果 TiDB 集群开启了 [TLS](enable-tls-between-components.md)，但是不想使用[文档](enable-tls-between-components.md)中创建的 `${cluster_name}-cluster-client-secret` 恢复备份，可以通过这个参数为恢复备份指定一个 Secret，可以通过如下命令生成：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

* `.spec.storageClassName`：指定恢复时所需的 PV 类型。
* `.spec.storageSize`：指定恢复集群时所需的 PV 大小。该值应大于 TiDB 集群备份的数据大小。
* `.spec.tableFilter`：恢复时指定让 BR 恢复符合 [table-filter 规则](https://docs.pingcap.com/zh/tidb/stable/table-filter/) 的表。默认情况下该字段可以不用配置。

    当不配置时，如果使用 TiDB Lightning 恢复，`tableFilter` 字段的默认值如下：

    ```bash
    tableFilter:
    - "*.*"
    - "!/^(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA)$/.*"
    ```

    如果使用 BR 恢复，BR 会恢复备份文件中的所有数据库：

    > **注意：**
    >
    > 如果要使用排除规则 `"!db.table"` 导出除 `db.table` 的所有表，那么在 `"!db.table"` 前必须先添加 `*.*` 规则。如下面例子所示：
    >
    > ```
    > tableFilter:
    > - "*.*"
    > - "!db.table"
    > ```

* `.spec.br`：BR 相关配置，具体介绍参考 [BR 字段介绍](#br-字段介绍)。
* `.spec.s3`：S3 兼容存储相关配置，具体介绍参考 [S3 字段介绍](#s3-存储字段介绍)。
* `.spec.gcs`：GCS 存储相关配置，具体介绍参考 [GCS 字段介绍](#gcs-存储字段介绍)。
* `.spec.local`：持久卷存储相关配置，具体介绍参考 [Local 字段介绍](#local-存储字段介绍)。

## BackupSchedule CR 字段介绍

`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定集群及远程存储相关的配置，字段和 Backup CR 中的 `spec` 一样，详细介绍可参考 [Backup CR 字段介绍](#backup-cr-字段介绍)。下面介绍 `backupSchedule` 独有的配置项：

+ `.spec.maxBackups`：一种备份保留策略，决定定时备份最多可保留的备份个数。超过该数目，就会将过时的备份删除。如果将该项设置为 `0`，则表示保留所有备份。
+ `.spec.maxReservedTime`：一种备份保留策略，按时间保留备份。例如将该参数设置为 `24h`，表示只保留最近 24 小时内的备份条目。超过这个时间的备份都会被清除。时间设置格式参考 [`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration)。如果同时设置最大备份保留个数和最长备份保留时间，则以最长备份保留时间为准。
+ `.spec.schedule`：Cron 的时间调度格式。具体格式可参考 [Cron](https://en.wikipedia.org/wiki/Cron)。
+ `.spec.pause`：该值默认为 `false`。如果将该值设置为 `true`，表示暂停定时调度。此时即使到了调度时间点，也不会进行备份。在定时备份暂停期间，备份 Garbage Collection (GC) 仍然正常进行。将 `true` 改为 `false` 则重新开启定时全量备份。

## 删除备份的 Backup CR

用户可以通过下述语句来删除对应的备份 CR 或定时全量备份 CR。

{{< copyable "shell-regular" >}}

```shell
kubectl delete backup ${name} -n ${namespace}
kubectl delete backupschedule ${name} -n ${namespace}
```

如果你使用 v1.1.2 及以前版本，或使用 v1.1.3 及以后版本并将 `spec.cleanPolicy` 设置为 `Delete` 时，TiDB Operator 在删除 CR 时会同时清理备份文件。

在满足上述条件时，如果需要删除 namespace，建议首先删除所有的 Backup/BackupSchedule CR，再删除 namespace。

如果直接删除存在 Backup/BackupSchedule CR 的 namespace，TiDB Operator 会持续尝试创建 Job 清理备份的数据，但因为 namespace 处于 `Terminating` 状态而创建失败，从而导致 namespace 卡在该状态。

这时需要通过下述命令删除 `finalizers`：

{{< copyable "shell-regular" >}}

```shell
kubectl edit backup ${name} -n ${namespace}
```

删除 `metadata.finalizers` 配置，即可正常删除 CR。

### 清理备份文件

TiDB Operator v1.2.3 及之前的版本，清理备份文件的方式为：循环删除备份文件，一次删除一个文件。

TiDB Operator v1.2.4 及以后的版本，清理备份文件的方式为：循环删除备份文件，一次批量删除多个文件。对于每次批量删除多个文件的操作，根据备份使用的后端存储类型的不同，删除方式不同。

* S3 兼容的后端存储采用并发批量删除方式。TiDB Operator 启动多个 Go 协程，每个 Go 协程每次调用批量删除接口 ["DeleteObjects"](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html) 来删除多个文件。
* 其他类型的后端存储采用并发删除方式。TiDB Operator 启动多个 Go 协程，每个 Go 协程每次删除一个文件。

对于 TiDB Operator v1.2.4 及以后的版本，你可以使用 Backup CR 中的以下字段控制清理行为：

* `.spec.cleanOption.pageSize`：指定每次批量删除的文件数量。默认值为 10000。
* `.spec.cleanOption.disableBatchConcurrency`：当设置为 true 时，TiDB Operator 会禁用并发批量删除方式，使用并发删除方式。
  
    如果 S3 兼容的后端存储不支持 `DeleteObjects` 接口，默认的并发批量删除会失败，需要配置该字段为 `true` 来使用并发删除方式。

* `.spec.cleanOption.batchConcurrency`: 指定并发批量删除方式下启动的 Go 协程数量。默认值为 10。
* `.spec.cleanOption.routineConcurrency`: 指定并发删除方式下启动的 Go 协程数量。默认值为 100。
