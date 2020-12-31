---
title: 使用 BR 工具恢复持久卷上的备份数据
summary: 介绍如何使用 BR 工具将存储在持久卷上的备份数据恢复到 TiDB 集群
---

# 使用 BR 工具恢复持久卷上的备份数据

本文描述了如何将存储在[持久卷](https://kubernetes.io/zh/docs/concepts/storage/persistent-volumes/)上的备份数据恢复到 Kubernetes 环境中的 TiDB 集群。底层通过使用 [`BR`](https://docs.pingcap.com/zh/tidb/dev/backup-and-restore-tool) 来进行集群恢复。

本文描述的持久卷指任何 [Kubernetes 支持的持久卷类型](https://kubernetes.io/zh/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes)。以下示例以 NFS 存储卷为例，介绍如何将存储在持久卷上指定路径的集群备份数据恢复到 TiDB 集群。

本文使用的恢复方式基于 TiDB Operator 新版（v1.1.8 及以上）的 CustomResourceDefinition (CRD) 实现。

## 数据库账户权限

- `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：恢复前后，restore CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间

> **注意：**
>
> 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，该步骤可以省略。

## 环境准备

1. 下载文件 [`backup-rbac.yaml`](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test2` 这个 namespace 中创建恢复所需的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. 创建 `restore-demo2-tidb-secret` secret，该 secret 存放用来访问 TiDB 服务的账号的密码：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=<password> --namespace=test2
    ```

    > **注意：**
    >
    > 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，该步骤可以省略。

3. 确认可以从 Kubernetes 集群中访问用于存储备份数据的 NFS 服务器。

## 恢复过程

1. 创建 restore custom resource (CR)，将指定的备份数据恢复至 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore.yaml
    ```

    `restore.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-nfs
      namespace: test2
    spec:
      # backupType: full
      br:
        cluster: demo2
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status-addr}
        # concurrency: 4
        # rateLimit: 0
        # checksum: true
      # # Only needed for TiDB Operator < v1.1.7 or TiDB < v4.0.8
      # to:
      #   host: ${tidb_host}
      #   port: ${tidb_port}
      #   user: ${tidb_user}
      #   secretName: restore-demo2-tidb-secret
      local:
        prefix: backup-nfs
        volume:
          name: nfs
          nfs:
            server: ${nfs_server_if}
            path: /nfs
        volumeMount:
          name: nfs
          mountPath: /nfs
    ```

2. 创建好 `Restore` CR 后，通过以下命令查看恢复的状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get rt -n test2 -owide
    ```

关于 BR 和持久卷的配置项可以参考 [backup-pv.yaml](backup-to-pv-using-br.md#ad-hoc-备份过程) 中的配置。

更多 `Restore` CR 字段的详细解释如下：

- `.spec.metadata.namespace`： `Restore` CR 所在的 namespace。
- `.spec.to.host`：待恢复 TiDB 集群的访问地址。
- `.spec.to.port`：待恢复 TiDB 集群访问的端口。
- `.spec.to.user`：待恢复 TiDB 集群的访问用户。
- `.spec.to.tidbSecretName`：待备份 TiDB 集群 `.spec.to.user` 用户的密码所对应的 secret。
- `.spec.to.tlsClientSecretName`：指定备份使用的存储证书的 Secret。

    如果 TiDB 集群[已开启 TLS](enable-tls-between-components.md)，但是不想使用[文档](enable-tls-between-components.md)中创建的 `${cluster_name}-cluster-client-secret` 恢复备份，可以通过这个参数为恢复备份指定一个 Secret，可以通过如下命令生成：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

    > **注意：**
    >
    > 如果使用 TiDB Operator >= v1.1.7 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，无需配置 `spec.to`.

- `.spec.tableFilter`：恢复时指定让 BR 恢复符合 [Table Filter 规则](https://docs.pingcap.com/zh/tidb/stable/table-filter/)的表。默认情况下该字段可以不用配置。当不配置时，BR 会恢复备份文件中的所有数据库：

    > **注意：**
    >
    > `tableFilter` 如果要写排除规则导出除 `db.table` 的所有表，`"!db.table"` 前必须先添加 `*.*` 规则来导出所有表，如下面例子所示：

    ```
    tableFilter:
    - "*.*"
    - "!db.table"
    ```

以上示例中，`.spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`timeAgo`、`sendCredToTikv`。

- `.spec.br.cluster`：代表需要备份的集群名字。
- `.spec.br.clusterNamespace`：代表需要备份的集群所在的 `namespace`。
- `.spec.br.logLevel`：代表日志的级别。默认为 `info`。
- `.spec.br.statusAddr`：为 BR 进程监听一个进程状态的 HTTP 端口，方便用户调试。如果不填，则默认不监听。
- `.spec.br.concurrency`：备份时每一个 TiKV 进程使用的线程数。备份时默认为 4，恢复时默认为 128。
- `.spec.br.rateLimit`：是否对流量进行限制。单位为 MB/s，例如设置为 `4` 代表限速 4 MB/s，默认不限速。
- `.spec.br.checksum`：是否在备份结束之后对文件进行验证。默认为 `true`。
- `.spec.br.timeAgo`：备份 timeAgo 以前的数据，默认为空（备份当前数据），[支持](https://golang.org/pkg/time/#ParseDuration) "1.5h", "2h45m" 等数据。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
