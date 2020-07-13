---
title: 使用 BR 工具备份 TiDB 集群到 GCS
summary: 介绍如何使用 BR 工具备份 TiDB 集群到 Google Cloud Storage (GCS)。
---

# 使用 BR 工具备份 TiDB 集群到 GCS

本文档详细描述了如何将 Kubernetes 上 TiDB 集群的数据备份到 [Google Cloud Storage (GCS)](https://cloud.google.com/storage/docs/) 上。本文档中的“备份”，均是指全量备份（Ad-hoc 全量备份和定时全量备份），底层通过使用 [`BR`](https://docs.pingcap.com/zh/tidb/dev/backup-and-restore-tool) 获取集群的逻辑备份，然后再将备份数据上传到远端 GCS。

本文使用的备份方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。

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
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=<password> --namespace=test1
    ```

### Ad-hoc 全量备份过程

1. 创建 `Backup` CR，并将数据备份到 GCS：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-gcs.yaml
    ```

    `backup-gcs.yaml` 文件内容如下：

    {{< copyable "shell-regular" >}}

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
    name: demo1-backup-gcs
    namespace: test1
    spec:
      # backupType: full
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
      gcs:
        projectId: ${project-id}
        secretName: gcs-secret
        bucket: ${bucket}
        prefix: ${prefix}
        # location: us-east1
        # storageClass: STANDARD_IA
        # objectAcl: private
    ```

    以上示例中，`spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`sendCredToTikv`。

    部分参数的含义如下：

    - `spec.br.cluster`：代表需要备份的集群名字。
    - `spec.br.clusterNamespace`：代表需要备份的集群所在的 `namespace`。
    - `spec.br.logLevel`：代表日志的级别。默认为 `info`。
    - `spec.br.statusAddr`：为 BR 进程监听一个进程状态的 HTTP 端口，方便用户调试。如果不填，则默认不监听。
    - `spec.br.concurrency`：备份时每一个 TiKV 进程使用的线程数。备份时默认为 4，恢复时默认为 128。
    - `spec.br.rateLimit`：是否对流量进行限制。单位为 MB/s，例如设置为 `4` 代表限速 4 MB/s，默认不限速。
    - `spec.br.checksum`：是否在备份结束之后对文件进行验证。默认为 `true`。
    - `spec.br.sendCredToTikv`：BR 进程是否将自己的 GCP 权限传输给 TiKV 进程。默认为 `true`。

    该示例将 TiDB 集群的数据全量导出备份到 GCS。`spec.gcs` 中的一些参数项均可省略，如 `location`、`objectAcl`、`storageClass`。

    配置中的 `projectId` 代表 GCP 上用户项目的唯一标识。具体获取该标识的方法可参考 [GCP 官方文档](https://cloud.google.com/resource-manager/docs/creating-managing-projects)。

    * 设置 `storageClass`。

        GCS 支持以下几种 `storageClass` 类型：

        * `MULTI_REGIONAL`
        * `REGIONAL`
        * `NEARLINE`
        * `COLDLINE`
        * `DURABLE_REDUCED_AVAILABILITY`

        如果不设置 `storageClass`，则默认使用 `COLDLINE`。这几种存储类型的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/storage-classes)。

    * 设置 object access-control list (ACL) 策略。

        GCS 支持以下几种 ACL 策略：

        * `authenticatedRead`
        * `bucketOwnerFullControl`
        * `bucketOwnerRead`
        * `private`
        * `projectPrivate`
        * `publicRead`

        如果不设置 object ACL 策略，则默认使用 `private` 策略。这几种访问控制策略的详细介绍可参考 [GCS 官方文档](https://cloud.google.com/storage/docs/access-control/lists)。

2. 创建好 `Backup` CR 后，可通过以下命令查看备份状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bk -n test1 -owide
    ```

更多 `Backup` CR 字段的详细解释：

* `.spec.metadata.namespace`：`Backup` CR 所在的 namespace。
* `.spec.from.host`：待备份 TiDB 集群的访问地址，为需要导出的 TiDB 的 service name，例如 `basic-tidb`。
* `.spec.from.port`：待备份 TiDB 集群的访问端口。
* `.spec.from.user`：待备份 TiDB 集群的访问用户。
* `.spec.gcs.bucket`：存储数据的 bucket 名字。
* `.spec.gcs.prefix`：这个字段可以省略，如果设置了这个字段，则会使用这个字段来拼接在远端存储的存储路径 `s3://${.spec.gcs.bucket}/${.spec.gcs.prefix}/backupName`。
* `.spec.from.tidbSecretName`：待备份 TiDB 集群 `.spec.from.user` 用户的密码所对应的 secret。
* `.spec.from.tlsClientSecretName`：指定备份使用的存储证书的 Secret。

    如果 TiDB 集群[已开启 TLS](enable-tls-between-components.md)，但是不想使用[文档](enable-tls-between-components.md)中创建的 `${cluster_name}-cluster-client-secret` 进行备份，可以通过这个参数为备份指定一个 Secret，可以通过如下命令生成：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

## 定时全量备份

用户通过设置备份策略来对 TiDB 集群进行定时备份，同时设置备份的保留策略以避免产生过多的备份。定时全量备份通过自定义的 `BackupSchedule` CR 对象来描述。每到备份时间点会触发一次全量备份，定时全量备份底层通过 Ad-hoc 全量备份来实现。下面是创建定时全量备份的具体步骤：

### 定时全量备份环境准备

同 [Ad-hoc 全量备份环境准备](#ad-hoc-全量备份环境准备)。

### 定时全量备份过程

1. 创建 `BackupSchedule` CR，开启 TiDB 集群的定时全量备份，将数据备份到 GCS：

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
          projectId: ${project-id}
          bucket: ${bucket}
          prefix: ${prefix}
          # location: us-east1
          # storageClass: STANDARD_IA
          # objectAcl: private
    ```

2. 定时全量备份创建完成后，通过以下命令查看备份的状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bks -n test1 -owide
    ```

    查看定时全量备份下面所有的备份条目：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-gcs -n test1
    ```

从以上示例可知，`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定 GCS 存储相关的配置，该配置与 Ad-hoc 全量备份到 GCS 的配置完全一样，可参考[Ad-hoc 全量备份过程](#ad-hoc-全量备份过程)。下面介绍 `backupSchedule` 独有的配置项：

+ `.spec.maxBackups`：一种备份保留策略，决定定时备份最多可保留的备份个数。超过该数目，就会将过时的备份删除。如果将该项设置为 `0`，则表示保留所有备份。
+ `.spec.maxReservedTime`：一种备份保留策略，按时间保留备份。比如将该参数设置为 `24h`，表示只保留最近 24 小时内的备份条目。超过这个时间的备份都会被清除。时间设置格式参考[`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration)。如果同时设置最大备份保留个数和最长备份保留时间，则以最长备份保留时间为准。
+ `.spec.schedule`：Cron 的时间调度格式。具体格式可参考 [Cron](https://en.wikipedia.org/wiki/Cron)。
+ `.spec.pause`：该值默认为 `false`。如果将该值设置为 `true`，表示暂停定时调度。此时即使到了调度时间点，也不会进行备份。在定时备份暂停期间，备份 [Garbage Collection (GC)](https://pingcap.com/docs-cn/stable/reference/garbage-collection/overview/) 仍然正常进行。将 `true` 改为 `false` 则重新开启定时全量备份。
