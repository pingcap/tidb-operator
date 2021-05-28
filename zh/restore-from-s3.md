---
title: 使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据
summary: 了解如何使用 TiDB Lightning 将兼容 S3 存储上的备份数据恢复到 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restore-from-s3/']
---

# 使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据

本文描述了将 Kubernetes 上通过 TiDB Operator 备份的数据恢复到 TiDB 集群的操作过程。

本文使用的恢复方式基于 TiDB Operator v1.1 及以上的 CustomResourceDefinition (CRD) 实现，底层通过使用 [TiDB Lightning TiDB-backend](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-backends#tidb-lightning-tidb-backend) 来恢复数据。

目前，TiDB Lightning 支持三种后端： `Importer-backend`、`Local-backend` 、`TiDB-backend`。关于这三种后端的区别和选择，请参阅 [TiDB Lightning 文档](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-backends)。如果要使用 `Importer-backend` 或者 `Local-backend` 导入数据，请参考[使用 TiDB Lightning 导入集群数据](restore-data-using-tidb-lightning.md)。

以下示例将兼容 S3 的存储（指定路径）上的备份数据恢复到 TiDB 集群。

## 环境准备

1. 下载文件 [`backup-rbac.yaml`](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test2` 这个 namespace 中创建恢复所需的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. 远程存储访问授权。

    如果从 Amazon S3 恢复集群数据，可以使用三种权限授予方式授予权限，参考 [AWS 账号授权](grant-permissions-to-remote-storage.md#aws-账号授权)授权访问兼容 S3 的远程存储；使用 Ceph 作为后端存储测试恢复时，是通过 AccessKey 和 SecretKey 模式授权，设置方式可参考[通过 AccessKey 和 SecretKey 授权](grant-permissions-to-remote-storage.md#通过-accesskey-和-secretkey-授权)。

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

> **注意：**
>
> 由于 `rclone` 存在[问题](https://rclone.org/s3/#key-management-system-kms)，如果使用 Amazon S3 存储备份，并且 Amazon S3 开启了 `AWS-KMS` 加密，需要在本节示例中的 yaml 文件里添加如下 `spec.s3.options` 配置以保证备份恢复成功：
>
> ```yaml
> spec:
>   ...
>   s3:
>     ...
>     options:
>     - --ignore-checksum
> ```

1. 创建 Restore customer resource (CR)，将制定备份数据恢复至 TiDB 集群

    + 创建 Restore custom resource (CR)，通过 AccessKey 和 SecretKey 授权的方式将指定的备份数据由 Ceph 恢复至 TiDB 集群：

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
          backupType: full
          to:
            host: ${tidb_host}
            port: ${tidb_port}
            user: ${tidb_user}
            secretName: restore-demo2-tidb-secret
          s3:
            provider: ceph
            endpoint: ${endpoint}
            secretName: s3-secret
            path: s3://${backup_path}
          # storageClassName: local-storage
          storageSize: 1Gi
        ```

    + 创建 Restore custom resource (CR)，通过 AccessKey 和 SecretKey 授权的方式将指定的备份数据由 Amazon S3 恢复至 TiDB 集群：

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
          backupType: full
          to:
            host: ${tidb_host}
            port: ${tidb_port}
            user: ${tidb_user}
            secretName: restore-demo2-tidb-secret
          s3:
            provider: aws
            region: ${region}
            secretName: s3-secret
            path: s3://${backup_path}
          # storageClassName: local-storage
          storageSize: 1Gi
        ```

    + 创建 Restore custom resource (CR)，通过 IAM 绑定 Pod 授权的方式将指定的备份数据恢复至 TiDB 集群：

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
          annotations:
            iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
        spec:
          backupType: full
          to:
            host: ${tidb_host}
            port: ${tidb_port}
            user: ${tidb_user}
            secretName: restore-demo2-tidb-secret
          s3:
            provider: aws
            region: ${region}
            path: s3://${backup_path}
          # storageClassName: local-storage
          storageSize: 1Gi
        ```

    + 创建 Restore custom resource (CR)，通过 IAM 绑定 ServiceAccount 授权的方式将指定的备份数据恢复至 TiDB 集群：

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
          backupType: full
          serviceAccount: tidb-backup-manager
          to:
            host: ${tidb_host}
            port: ${tidb_port}
            user: ${tidb_user}
            secretName: restore-demo2-tidb-secret
          s3:
            provider: aws
            region: ${region}
            path: s3://${backup_path}
          # storageClassName: local-storage
          storageSize: 1Gi
        ```

2. 创建好 `Restore` CR 后，可通过以下命令查看恢复的状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get rt -n test2 -owide
    ```

以上示例将兼容 S3 的存储（`spec.s3.path` 路径下）中的备份数据恢复到 TiDB 集群 `spec.to.host`。有关兼容 S3 的存储的配置项，可以参考 [S3 字段介绍](backup-restore-overview.md#s3-存储字段介绍)。

更多 `Restore` CR 字段的详细解释参考[Restore CR 字段介绍](backup-restore-overview.md#restore-cr-字段介绍)。

> **注意：**
>
> TiDB Operator 会创建一个 PVC，用于数据恢复，备份数据会先从远端存储下载到 PV，然后再进行恢复。如果恢复完成后想要删掉这个 PVC，可以参考[删除资源](cheat-sheet.md#删除资源)先把恢复 Pod 删掉，然后再把 PVC 删掉。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
