---
title: 使用 BR 工具备份 TiDB 集群到持久卷
summary: 介绍如何使用 BR 工具备份 TiDB 集群到持久卷。
---

# 使用 BR 工具备份 TiDB 集群到持久卷

本文档详细描述了如何将 Kubernetes 上 TiDB 集群的数据备份到[持久卷](https://kubernetes.io/zh/docs/concepts/storage/persistent-volumes/)上。

本文使用的备份方式基于 TiDB Operator 新版（v1.1.8 及以上）的 CustomResourceDefinition (CRD) 实现。

本文描述的持久卷指任何 [Kubernetes 支持的持久卷类型](https://kubernetes.io/zh/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes)。本文将以 NFS 为例，介绍如何使用 BR 工具备份 TiDB 集群的数据到持久卷。

## Ad-hoc 备份

Ad-hoc 备份支持全量备份与增量备份。Ad-hoc 备份通过创建一个自定义的 `Backup` custom resource (CR) 对象来描述一次备份。TiDB Operator 根据这个 `Backup` 对象来完成具体的备份过程。如果备份过程中出现错误，程序不会自动重试，此时需要手动处理。

为了更好地描述备份的使用方式，本文档提供如下备份示例。示例假设对部署在 Kubernetes `test1` 这个 namespace 中的 TiDB 集群 `demo1` 进行数据备份，下面是具体操作过程。

### Ad-hoc 备份环境准备

1. 下载文件 [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test1` 这个 namespace 中创建备份需要的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. 创建 `backup-demo1-tidb-secret` secret。该 secret 存放用于访问 TiDB 集群的账号的密码。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=<password> --namespace=test1
    ```

    > **注意：**
    >
    > 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，该步骤可以省略。

3. 确认可以从 Kubernetes 集群中访问用于存储备份数据的 NFS 服务器。

### 数据库账户权限

* `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：备份前后，backup CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间

> **注意：**
>
> 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，该步骤可以省略。

### Ad-hoc 备份过程

1. 创建 `Backup` CR，并将数据备份到 NFS：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-nfs.yaml
    ```

    `backup-nfs.yaml` 文件内容如下：

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

    以上示例中，`spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`timeAgo`。

    自 v1.1.6 版本起，如果需要增量备份，只需要在 `spec.br.options` 中指定上一次的备份时间戳 `--lastbackupts` 即可。有关增量备份的限制，可参考 [使用 BR 进行备份与恢复](https://docs.pingcap.com/zh/tidb/stable/backup-and-restore-tool#增量备份)。

    部分参数的含义如下：

    - `spec.br.cluster`：代表需要备份的集群名字。
    - `spec.br.clusterNamespace`：代表需要备份的集群所在的 `namespace`。
    - `spec.br.logLevel`：代表日志的级别。默认为 `info`。
    - `spec.br.statusAddr`：为 BR 进程监听一个进程状态的 HTTP 端口，方便用户调试。如果不填，则默认不监听。
    - `spec.br.concurrency`：备份时每一个 TiKV 进程使用的线程数。备份时默认为 4，恢复时默认为 128。
    - `spec.br.rateLimit`：是否对流量进行限制。单位为 MB/s，例如设置为 `4` 代表限速 4 MB/s，默认不限速。
    - `spec.br.checksum`：是否在备份结束之后对文件进行验证。默认为 `true`。
    - `spec.br.timeAgo`：备份 timeAgo 以前的数据，默认为空（备份当前数据），[支持](https://golang.org/pkg/time/#ParseDuration) "1.5h", "2h45m" 等数据。
    - `spec.br.options`：BR 工具支持的额外参数，需要以字符串数组的形式传入。自 v1.1.6 版本起支持该参数。可用于指定 `lastbackupts` 以进行增量备份。

    该示例将 TiDB 集群的数据全量导出备份到 NFS。

2. 创建好 `Backup` CR 后，可通过以下命令查看备份状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bk -n test1 -owide
    ```

更多 `Backup` CR 字段的详细解释：

- `.spec.metadata.namespace`：`Backup` CR 所在的 namespace。
- `.spec.tikvGCLifeTime`：备份中的临时 `tikv_gc_life_time` 时间设置，默认为 72h。

    在备份开始之前，若 TiDB 集群的 `tikv_gc_life_time` 小于用户设置的 `spec.tikvGCLifeTime`，为了保证备份的数据不被 TiKV GC 掉，TiDB Operator 会在备份前[调节 `tikv_gc_life_time`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview#导出大规模数据时的-tidb-gc-设置) 为 `spec.tikvGCLifeTime`。

    备份结束后不论成功或者失败，只要老的 `tikv_gc_life_time` 比设置的 `.spec.tikvGCLifeTime` 小，TiDB Operator 都会尝试恢复 `tikv_gc_life_time` 为备份前的值。在极端情况下，TiDB Operator 访问数据库失败会导致 TiDB Operator 无法自动恢复 `tikv_gc_life_time` 并认为备份失败。

    此时，可以通过下述语句查看当前 TiDB 集群的 `tikv_gc_life_time`：

    {{< copyable "sql" >}}

    ```sql
    SELECT VARIABLE_NAME, VARIABLE_VALUE FROM mysql.tidb WHERE VARIABLE_NAME LIKE "tikv_gc_life_time";
    ```

    如果发现 `tikv_gc_life_time` 值过大（通常为 10m），则需要按照[调节 `tikv_gc_life_time`](https://docs.pingcap.com/zh/tidb/stable/dumpling-overview#导出大规模数据时的-tidb-gc-设置) 将 `tikv_gc_life_time` 调回原样：

    {{< copyable "sql" >}}

    ```sql
    UPDATE mysql.tidb SET VARIABLE_VALUE = '10m' WHERE VARIABLE_NAME = 'tikv_gc_life_time';
    ```

    > **Note:**
    >
    > 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，无需配置 `spec.tikvGCLifeTime`.

- `.spec.cleanPolicy`：备份集群后删除备份 CR 时的备份文件清理策略。

    目前支持三种清理策略：

    - `Retain`：任何情况下，删除备份 CR 时会保留备份出的文件
    - `Delete`：任何情况下，删除备份 CR 时会删除备份出的文件
    - `OnFailure`：如果备份中失败，删除备份 CR 时会删除备份出的文件

    如果不配置该字段，或者配置该字段的值为上述三种以外的值，均会保留备份出的文件。
    
    值得注意的是，在 v1.1.2 以及之前版本不存在该字段，且默认在删除 CR 的同时删除备份的文件。若 v1.1.3 及之后版本的用户希望保持该行为，需要设置该字段为 `Delete`。

- `.spec.from.host`：待备份 TiDB 集群的访问地址，为需要导出的 TiDB 的 service name，例如 `demo1-tidb`。
- `.spec.from.port`：待备份 TiDB 集群的访问端口。
- `.spec.from.user`：待备份 TiDB 集群的访问用户。
- `.spec.from.tidbSecretName`：待备份 TiDB 集群 `.spec.from.user` 用户的密码所对应的 secret。
- `.spec.from.tlsClientSecretName`：指定备份使用的存储证书的 Secret。

    如果 TiDB 集群[已开启 TLS](enable-tls-between-components.md)，但是不想使用[文档](enable-tls-between-components.md)中创建的 `${cluster_name}-cluster-client-secret` 进行备份，可以通过 `.spec.from.tlsClientSecretName` 参数为备份指定一个 secret。使用如下命令生成 secret：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

    > **Note:**
    >
    > 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，无需配置 `spec.from`.

- `.spec.local.prefix`：建议配置这个字段，如果设置了这个字段，则会使用这个字段来拼接在持久卷的存储路径 `local://${.spec.local.volumeMount.mountPath}/${.spec.local.prefix}/`。
- `.spec.tableFilter`：备份时指定让 BR 备份符合 [table-filter 规则](https://docs.pingcap.com/zh/tidb/stable/table-filter/) 的表。默认情况下该字段可以不用配置。当不配置时，BR 会备份除系统库以外的所有数据库：

    > **注意：**
    >
    > tableFilter 如果要写排除规则导出除 db.table 的所有表 "!db.table" 必须先添加 `*.*` 规则来导出所有表，如下面例子所示：

    ```
    tableFilter:
    - "*.*"
    - "!db.table"
    ```

## 定时全量备份

用户通过设置备份策略来对 TiDB 集群进行定时备份，同时设置备份的保留策略以避免产生过多的备份。定时全量备份通过自定义的 `BackupSchedule` CR 对象来描述。每到备份时间点会触发一次全量备份，定时全量备份底层通过 Ad-hoc 全量备份来实现。下面是创建定时全量备份的具体步骤：

### 定时全量备份环境准备

同 [Ad-hoc 全量备份环境准备](#ad-hoc-备份环境准备)。

### 定时全量备份过程

1. 创建 `BackupSchedule` CR，开启 TiDB 集群的定时全量备份，将数据备份到 NFS：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-schedule-nfs.yaml
    ```

    `backup-schedule-nfs.yaml` 文件内容如下：

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

2. 定时全量备份创建完成后，通过以下命令查看备份的状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bks -n test1 -owide
    ```

    查看定时全量备份下面所有的备份条目：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-nfs -n test1
    ```

从以上示例可知，`backupSchedule` 的配置由两部分组成。一部分是 `backupSchedule` 独有的配置，另一部分是 `backupTemplate`。`backupTemplate` 指定 NFS 存储相关的配置，该配置与 Ad-hoc 全量备份到 NFS 的配置完全一样，可参考[Ad-hoc 全量备份过程](#ad-hoc-备份过程)。

<details>
<summary><code>backupSchedule</code> 独有的配置项</summary>

- `.spec.maxBackups`：一种备份保留策略，决定定时备份最多可保留的备份个数。超过该数目，就会将过时的备份删除。如果将该项设置为 `0`，则表示保留所有备份。

- `.spec.maxReservedTime`：一种备份保留策略，按时间保留备份。比如将该参数设置为 `24h`，表示只保留最近 24 小时内的备份条目。超过这个时间的备份都会被清除。时间设置格式参考[`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration)。如果同时设置最大备份保留个数和最长备份保留时间，则以最长备份保留时间为准。

- `.spec.schedule`：Cron 的时间调度格式。具体格式可参考 [Cron](https://en.wikipedia.org/wiki/Cron)。

- `.spec.pause`：该值默认为 `false`。如果将该值设置为 `true`，表示暂停定时调度。此时即使到了调度时间点，也不会进行备份。在定时备份暂停期间，备份 [Garbage Collection (GC)](https://pingcap.com/docs-cn/stable/reference/garbage-collection/overview/) 仍然正常进行。将 `true` 改为 `false` 则重新开启定时全量备份。

</details>

## 删除备份的 backup CR

用户可以通过下述语句来删除对应的备份 CR 或定时全量备份 CR。

{{< copyable "shell-regular" >}}

```shell
kubectl delete backup ${name} -n ${namespace}
kubectl delete backupschedule ${name} -n ${namespace}
```

如果你使用 v1.1.2 及以前版本，或使用 v1.1.3 及以后版本并将 `spec.cleanPolicy` 设置为 `Delete` 时，TiDB Operator 在删除 CR 时会同时删除备份文件。在满足上述条件时，如果需要删除 namespace，建议首先删除所有的 Backup/BackupSchedule CR，再删除 namespace。

如果直接删除存在 Backup/BackupSchedule CR 的 namespace，TiDB Operator 会持续尝试创建 Job 清理备份的数据，但因为 namespace 处于 `Terminating` 状态而创建失败，从而导致 namespace 卡在该状态。

这时需要通过下述命令删除 `finalizers`：

{{< copyable "shell-regular" >}}

```shell
kubectl edit backup ${name} -n ${namespace}
```

删除 `metadata.finalizers` 配置，即可正常删除 CR。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
