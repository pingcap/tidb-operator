---
title: 使用 BR 恢复持久卷上的备份数据
summary: 介绍如何使用 BR 将存储在持久卷上的备份数据恢复到 TiDB 集群
---

# 使用 BR 恢复持久卷上的备份数据

本文描述了如何将存储在[持久卷](https://kubernetes.io/zh/docs/concepts/storage/persistent-volumes/)上的备份数据恢复到 Kubernetes 环境中的 TiDB 集群。底层通过使用 [`BR`](https://docs.pingcap.com/zh/tidb/dev/backup-and-restore-tool) 来进行集群恢复。

本文描述的持久卷指任何 [Kubernetes 支持的持久卷类型](https://kubernetes.io/zh/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes)。以下示例以 NFS 存储卷为例，介绍如何将存储在持久卷上指定路径的集群备份数据恢复到 TiDB 集群。

本文使用的恢复方式基于 TiDB Operator 新版（v1.1.8 及以上）的 CustomResourceDefinition (CRD) 实现。

## 环境准备

> **注意：**
>
> 如果使用 TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，不需要在 Restore CR 中配置 `spec.to` 字段，并且可以省略以下创建 `restore-demo2-tidb-secret` secret 的步骤和[数据库账户权限](#数据库账户权限)步骤。

1. 下载文件 [`backup-rbac.yaml`](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test2` 这个 namespace 中创建恢复所需的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. 创建 `restore-demo2-tidb-secret` secret，该 secret 存放用来访问 TiDB 服务的账号的密码：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=<password> --namespace=test2
    ```

3. 确认可以从 Kubernetes 集群中访问用于存储备份数据的 NFS 服务器。

## 数据库账户权限

- `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：恢复前后，Restore CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间

## 恢复过程

1. 创建 restore custom resource (CR)，将指定的备份数据恢复至 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ```bash
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
      # # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
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

    ```bash
    kubectl get rt -n test2 -owide
    ```

以上示例将存储在 NFS 上指定路径 `local://${.spec.local.volumeMount.mountPath}/${.spec.local.prefix}/` 文件夹下的备份数据恢复到 namespace `test2` 中的 TiDB 集群 `demo2`。持久卷存储相关配置参考 [Local 存储字段介绍](backup-restore-overview.md#local-存储字段介绍)。

以上示例中，`.spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`timeAgo`、`sendCredToTikv`。更多 `.spec.br` 字段的详细解释参考 [BR 字段介绍](backup-restore-overview.md#br-字段介绍)。

更多 `Restore` CR 字段的详细解释参考 [Restore CR 字段介绍](backup-restore-overview.md#restore-cr-字段介绍)。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
