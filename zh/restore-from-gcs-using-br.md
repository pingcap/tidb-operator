---
title: 使用 BR 恢复 GCS 上的备份数据
summary: 介绍如何使用 BR 将存储在 GCS 上的备份数据恢复到 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restore-from-gcs-using-br/']
---

# 使用 BR 恢复 GCS 上的备份数据

本文描述了如何将存储在 [Google Cloud Storage (GCS)](https://cloud.google.com/storage/docs/) 上的备份数据恢复到 Kubernetes 环境中的 TiDB 集群。底层通过使用 [`BR`](https://docs.pingcap.com/zh/tidb/dev/backup-and-restore-tool) 来进行集群恢复。

本文使用的恢复方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。

以下示例将存储在 GCS 上指定路径的集群备份数据恢复到 TiDB 集群。

## 环境准备

> **注意：**
>
> 如果使用 TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，不需要在 Restore CR 中配置 `spec.to` 字段，并且可以省略以下创建 `restore-demo2-tidb-secret` secret 的步骤和[数据库账户权限](#数据库账户权限)步骤。

1. 下载文件 [`backup-rbac.yaml`](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test2` 这个 namespace 中创建恢复所需的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. 远程存储访问授权。

    参考 [GCS 账号授权](grant-permissions-to-remote-storage.md#gcs-账号授权)授权访问 GCS 远程存储。

3. 创建 `restore-demo2-tidb-secret` secret，该 secret 存放用来访问 TiDB 集群的 root 账号和密钥：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=<password> --namespace=test2
    ```

## 数据库账户权限

* `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：恢复前后，Restore CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间

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
      name: demo2-restore-gcs
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
        # sendCredToTikv: true
      # # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      # to:
      #   host: ${tidb_host}
      #   port: ${tidb_port}
      #   user: ${tidb_user}
      #   secretName: restore-demo2-tidb-secret
      gcs:
        projectId: ${project_id}
        secretName: gcs-secret
        bucket: ${bucket}
        prefix: ${prefix}
        # location: us-east1
        # storageClass: STANDARD_IA
        # objectAcl: private
    ```

2. 创建好 `Restore` CR 后，通过以下命令查看恢复的状态：

    {{< copyable "shell-regular" >}}

     ```shell
     kubectl get rt -n test2 -owide
     ```

以上示例将存储在 GCS 上指定路径 `spec.gcs.bucket` 存储桶中 `spec.gcs.prefix` 文件夹下的备份数据恢复到 namespace `test2` 中的 TiDB 集群 `demo2`。GCS 存储相关配置参考 [GCS 存储字段介绍](backup-restore-overview.md#gcs-存储字段介绍)。

以上示例中，`.spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`timeAgo`、`sendCredToTikv`。更多 `.spec.br` 字段的详细解释参考 [BR 字段介绍](backup-restore-overview.md#br-字段介绍)。

更多 `Restore` CR 字段的详细解释参考 [Restore CR 字段介绍](backup-restore-overview.md#restore-cr-字段介绍)。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
