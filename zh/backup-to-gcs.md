---
title: 使用 Dumpling 备份 TiDB 集群数据到 GCS
summary: 介绍如何使用 Dumpling 将 TiDB 集群数据备份到 Google Cloud Storage (GCS)。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/backup-to-gcs/']
---

# 使用 Dumpling 备份 TiDB 集群数据到 GCS

本文档详细描述了如何将 Kubernetes 上 TiDB 集群的数据备份到 [Google Cloud Storage (GCS)](https://cloud.google.com/storage/docs/) 上。本文档中的“备份”，均是指全量备份（Ad-hoc 全量备份和定时全量备份），底层通过使用 [`Dumpling`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview) 获取集群的逻辑备份，然后再将备份数据上传到远端 GCS。

本文使用的备份方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。基于 Helm Charts 的备份和恢复方式可参考[基于 Helm Charts 实现的 TiDB 集群备份与恢复](backup-and-restore-using-helm-charts.md)。

## Ad-hoc 全量备份

Ad-hoc 全量备份通过创建一个自定义的 `Backup` custom resource (CR) 对象来描述一次备份。TiDB Operator 根据这个 `Backup` 对象来完成具体的备份过程。如果备份过程中出现错误，程序不会自动重试，此时需要手动处理。

为了更好地描述备份的使用方式，本文档提供如下备份示例。示例假设对部署在 Kubernetes `test1` 这个 namespace 中的 TiDB 集群 `demo1` 进行数据备份，下面是具体操作过程。

### Ad-hoc 全量备份环境准备

1. 下载文件 [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test1` 这个 namespace 中创建备份需要的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. 创建 `gcs-secret` secret。该 secret 存放用于访问 GCS 的凭证。`google-credentials.json` 文件存放用户从 GCP console 上下载的 service account key。具体操作参考 [GCP 官方文档](https://cloud.google.com/docs/authentication/getting-started)。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic gcs-secret --from-file=credentials=./google-credentials.json -n test1
    ```

3. 创建 `backup-demo1-tidb-secret` secret。该 secret 存放用于访问 TiDB 集群的 root 账号和密钥。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=${password} --namespace=test1
    ```

### 备份数据到 GCS

创建 `Backup` CR，并将数据备份到 GCS：

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f backup-gcs.yaml
```

`backup-gcs.yaml` 文件内容如下：

```yaml
---
apiVersion: pingcap.com/v1alpha1
kind: Backup
metadata:
  name: demo1-backup-gcs
  namespace: test1
spec:
  from:
    host: ${tidb_host}
    port: ${tidb_port}
    user: ${tidb_user}
    secretName: backup-demo1-tidb-secret
  gcs:
    secretName: gcs-secret
    projectId: ${project_id}
    bucket: ${bucket}
    # prefix: ${prefix}
    # location: us-east1
    # storageClass: STANDARD_IA
    # objectAcl: private
    # bucketAcl: private
# dumpling:
#  options:
#  - --threads=16
#  - --rows=10000
#  tableFilter:
#  - "test.*"
  storageClassName: local-storage
  storageSize: 10Gi
```

以上示例将 TiDB 集群的数据全量导出备份到 GCS。GCS 配置中的 `location`、`objectAcl`、`bucketAcl`、`storageClass` 项均可以省略。

配置中的 `projectId` 代表 GCP 上用户项目的唯一标识。具体获取该标识的方法可参考 [GCP 官方文档](https://cloud.google.com/resource-manager/docs/creating-managing-projects)。

GCS 支持以下几种 `storageClass` 类型：

* `MULTI_REGIONAL`
* `REGIONAL`
* `NEARLINE`
* `COLDLINE`
* `DURABLE_REDUCED_AVAILABILITY`

如果不设置 `storageClass`，则默认使用 `COLDLINE`。这几种存储类型的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/storage-classes)。

GCS 支持以下几种 object access-control list (ACL) 策略：

* `authenticatedRead`
* `bucketOwnerFullControl`
* `bucketOwnerRead`
* `private`
* `projectPrivate`
* `publicRead`

如果不设置 object ACL 策略，则默认使用 `private` 策略。这几种访问控制策略的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/access-control/lists)。

GCS 支持以下几种 bucket ACL 策略：

* `authenticatedRead`
* `private`
* `projectPrivate`
* `publicRead`
* `publicReadWrite`

如果不设置 bucket ACL 策略，则默认策略为 `private`。这几种访问控制策略的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/access-control/lists)。

创建好 `Backup` CR 后，可通过以下命令查看备份状态：

{{< copyable "shell-regular" >}}

 ```shell
 kubectl get bk -n test1 -owide
 ```

更多 `Backup` CR 字段的详细解释：

* `.spec.metadata.namespace`：`Backup` CR 所在的 namespace。
* `.spec.tikvGCLifeTime`：备份中的临时 `tikv_gc_lifetime` 时间设置，默认为 72h。

    在备份开始之前，若 TiDB 集群的 `tikv_gc_lifetime` 小于用户设置的 `spec.tikvGCLifeTime`，为了保证备份的数据不被 TiKV GC 掉，TiDB Operator 会在备份前[调节 `tikv_gc_lifetime`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview#导出大规模数据时的-tidb-gc-设置) 为 `spec.tikvGCLifeTime`。

    备份结束后不论成功或者失败，只要老的 `tikv_gc_lifetime` 比设置的 `.spec.tikvGCLifeTime` 小，TiDB Operator 都会尝试恢复 `tikv_gc_lifetime` 为备份前的值。在极端情况下，TiDB Operator 访问数据库失败会导致 TiDB Operator 无法自动恢复 `tikv_gc_lifetime` 并认为备份失败。

    此时，可以通过下述语句查看当前 TiDB 集群的 `tikv_gc_lifetime`：

    ```sql
    select VARIABLE_NAME, VARIABLE_VALUE from mysql.tidb where VARIABLE_NAME like "tikv_gc_life_time";
    ```

    如果发现 `tikv_gc_lifetime` 值过大（通常为 10m），则需要按照[调节 `tikv_gc_lifetime`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview#导出大规模数据时的-tidb-gc-设置) 将 `tikv_gc_lifetime` 调回原样。

* `.spec.cleanPolicy`：备份集群后删除备份 CR 时的备份文件清理策略。目前支持三种清理策略：

    * `Retain`：任何情况下，删除备份 CR 时会保留备份出的文件
    * `Delete`：任何情况下，删除备份 CR 时会删除备份出的文件
    * `OnFailure`：如果备份中失败，删除备份 CR 时会删除备份出的文件
    
    如果不配置该字段，或者配置该字段的值为上述三种以外的值，均会保留备份出的文件。值得注意的是，在 v1.1.2 以及之前版本不存在该字段，且默认在删除 CR 的同时删除备份的文件。若 v1.1.3 及之后版本的用户希望保持该行为，需要设置该字段为 `Delete`。

* `.spec.from.host`：待备份 TiDB 集群的访问地址，为需要导出的 TiDB 的 service name，例如 `basic-tidb`。
* `.spec.from.port`：待备份 TiDB 集群的访问端口。
* `.spec.from.user`：待备份 TiDB 集群的访问用户。
* `.spec.from.tidbSecretName`：待备份 TiDB 集群所需凭证的 secret。
* `.spec.gcs.bucket`：存储数据的 bucket 名字。
* `.spec.gcs.prefix`：这个字段可以省略，如果设置了这个字段，则会使用这个字段来拼接在远端存储的存储路径 `s3://${.spec.gcs.bucket}/${.spec.gcs.prefix}/backupName`。
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

* `.spec.storageClassName`：备份时指定所需的 persistent volume (PV) 类型。如果不指定该项，则默认使用 TiDB Operator 启动参数中 `default-backup-storage-class-name` 指定的值，该值默认为 `standard`。
* `.spec.storageSize`：备份时指定所需的 PV 大小，默认为 100 Gi。该值应大于备份 TiDB 集群数据的大小。一个 TiDB 集群的 Backup CR 对应的 PVC 名字是确定的，如果集群命名空间中已存在该 PVC 并且其大小小于 `.spec.storageSize`，这时需要先删除该 PVC 再运行 Backup job。

## 定时全量备份

用户通过设置备份策略来对 TiDB 集群进行定时备份，同时设置备份的保留策略以避免产生过多的备份。定时全量备份通过自定义的 `BackupSchedule` CR 对象来描述。每到备份时间点会触发一次全量备份，定时全量备份底层通过 Ad-hoc 全量备份来实现。下面是创建定时全量备份的具体步骤：

### 定时全量备份环境准备

同 [Ad-hoc 全量备份环境准备](#ad-hoc-全量备份环境准备)。

### 定时全量备份数据到 GCS

创建 `BackupSchedule` CR 开启 TiDB 集群的定时全量备份，将数据备份到 GCS：

{{< copyable "shell-regular" >}}

```shell
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
    from:
      host: ${tidb_host}
      port: ${tidb_port}
      user: ${tidb_user}
      secretName: backup-demo1-tidb-secret
    gcs:
      secretName: gcs-secret
      projectId: ${project_id}
      bucket: ${bucket}
      # prefix: ${prefix}
      # location: us-east1
      # storageClass: STANDARD_IA
      # objectAcl: private
      # bucketAcl: private
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
kubectl get bks -n test1 -owide
```

查看定时全量备份下面所有的备份条目：

{{< copyable "shell-regular" >}}

 ```shell
 kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-gcs -n test1
 ```

从以上示例可知，`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定 GCS 存储相关的配置，该配置与 Ad-hoc 全量备份到 GCS 的配置完全一样，可参考[备份数据到 GCS](#备份数据到-gcs)。下面介绍 `backupSchedule` 独有的配置项：

+ `.spec.maxBackups`：一种备份保留策略，决定定时备份最多可保留的备份个数。超过该数目，就会将过时的备份删除。如果将该项设置为 `0`，则表示保留所有备份。
+ `.spec.maxReservedTime`：一种备份保留策略，按时间保留备份。比如将该参数设置为 `24h`，表示只保留最近 24 小时内的备份条目。超过这个时间的备份都会被清除。时间设置格式参考[`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration)。如果同时设置最大备份保留个数和最长备份保留时间，则以最长备份保留时间为准。
+ `.spec.schedule`：Cron 的时间调度格式。具体格式可参考 [Cron](https://en.wikipedia.org/wiki/Cron)。
+ `.spec.pause`：该值默认为 `false`。如果将该值设置为 `true`，表示暂停定时调度。此时即使到了调度时间点，也不会进行备份。在定时备份暂停期间，备份 [Garbage Collection (GC)](https://pingcap.com/docs-cn/stable/reference/garbage-collection/overview) 仍然正常进行。将 `true` 改为 `false` 则重新开启定时全量备份。

> **注意：**
>
> TiDB Operator 会创建一个 PVC，这个 PVC 同时用于 Ad-hoc 全量备份和定时全量备份，备份数据会先存储到 PV，然后再上传到远端存储。如果备份完成后想要删掉这个 PVC，可以参考[删除资源](cheat-sheet.md#删除资源)先把备份 Pod 删掉，然后再把 PVC 删掉。
>
> 假如备份并上传到远端存储成功，TiDB Operator 会自动删除本地的备份文件。如果上传失败，则本地备份文件将被保留。

## 删除备份的 backup CR

用户可以通过下述语句来删除对应的全量备份 CR 或定时全量备份 CR。

{{< copyable "shell-regular" >}}

```shell
kubectl delete backup ${name} -n ${namespace}
kubectl delete backupschedule ${name} -n ${namespace}
```

如果你使用 v1.1.2 及以前版本，或使用 v1.1.3 及以后版本并将 `spec.cleanPolicy` 设置为 `Delete` 时，TiDB Operator 在删除 CR 时会同时删除备份文件。

在满足上述条件时，如果需要删除 namespace，建议首先删除所有的 Backup/BackupSchedule CR，再删除 namespace。

如果直接删除存在 Backup/BackupSchedule CR 的 namespace，TiDB Operator 会持续尝试创建 Job 清理备份的数据，但因为 namespace 处于 `Terminating` 状态而创建失败，从而导致 namespace 卡在该状态。

同时 v1.1.2 及以前版本如果删除 Backup/BackupSchedule CR 时备份文件已被手动清除，也会出现卡在 `Terminating` 的情况。

这时需要通过下述命令删除 `finalizers`：

{{< copyable "shell-regular" >}}

```shell
kubectl edit backup ${name} -n ${namespace}
```

删除 `metadata.finalizers` 配置，即可正常删除 CR。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
