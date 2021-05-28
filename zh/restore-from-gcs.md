---
title: 使用 TiDB Lightning 恢复 GCS 上的备份数据
summary: 介绍如何使用 TiDB Lightning 将存储在 GCS 上的备份数据恢复到 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restore-from-gcs/']
---

# 使用 TiDB Lightning 恢复 GCS 上的备份数据

本文描述了将 Kubernetes 上通过 TiDB Operator 备份的数据恢复到 TiDB 集群的操作过程。

本文使用的恢复方式基于 TiDB Operator v1.1 及以上的 CustomResourceDefinition (CRD) 实现，底层通过使用 [TiDB Lightning TiDB-backend](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-backends#tidb-lightning-tidb-backend) 来恢复数据。

目前，TiDB Lightning 支持三种后端： `Importer-backend`、`Local-backend` 、`TiDB-backend`。关于这三种后端的区别和选择，请参阅 [TiDB Lightning 文档](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-backends)。如果要使用 `Importer-backend` 或者 `Local-backend` 导入数据，请参阅[使用 TiDB Lightning 导入集群数据](restore-data-using-tidb-lightning.md)。

以下示例将存储在 [Google Cloud Storage (GCS)](https://cloud.google.com/storage/docs/) 上指定路径上的集群备份数据恢复到 TiDB 集群。

## 环境准备

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
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=${password} --namespace=test2
    ```

## 数据库账户权限

| 权限 | 作用域 |
|:----|:------|
| SELECT | Tables |
| INSERT | Tables |
| UPDATE | Tables |
| DELETE | Tables |
| CREATE | Databases, tables |
| DROP | Databases, tables |
| ALTER | Tables |

## 将指定备份数据恢复到 TiDB 集群

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
      name: demo2-restore
      namespace: test2
    spec:
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      gcs:
        projectId: ${project_id}
        secretName: gcs-secret
        path: gcs://${backup_path}
      # storageClassName: local-storage
      storageSize: 1Gi
    ```

    以上示例将存储在 GCS 上指定路径 `spec.gcs.path` 的备份数据恢复到 TiDB 集群 `spec.to.host`。关于 GCS 的配置项可以参考 [GCS 字段介绍](backup-restore-overview.md#gcs-存储字段介绍)。

    更多 `Restore` CR 字段的详细解释参考 [Restore CR 字段介绍](backup-restore-overview.md#restore-cr-字段介绍)。

2. 创建好 `Restore` CR 后可通过以下命令查看恢复的状态：

    {{< copyable "shell-regular" >}}

     ```shell
     kubectl get rt -n test2 -owide
     ```

> **注意：**
>
> TiDB Operator 会创建一个 PVC，用于数据恢复，备份数据会先从远端存储下载到 PV，然后再进行恢复。如果恢复完成后想要删掉这个 PVC，可以参考[删除资源](cheat-sheet.md#删除资源)先把恢复 Pod 删掉，然后再把 PVC 删掉。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
