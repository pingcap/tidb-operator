---
title: 使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据
summary: 了解如何使用 TiDB Lightning 将兼容 S3 存储上的备份数据恢复到 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restore-from-s3/']
---

# 使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据

本文描述了将 Kubernetes 上通过 TiDB Operator 备份的数据恢复到 TiDB 集群的操作过程。底层通过使用 [TiDB Lightning](https://pingcap.com/docs/stable/how-to/get-started/tidb-lightning/#tidb-lightning-tutorial) 来恢复数据。

本文使用的恢复方式基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。基于 Helm Charts 实现的备份和恢复方式可参考[基于 Helm Charts 实现的 TiDB 集群备份恢复](backup-and-restore-using-helm-charts.md)。

## AWS 账号的三种权限授予方式

如果使用 Amazon S3 来备份恢复集群，可以使用三种权限授予方式授予权限，参考[使用 BR 工具备份 AWS 上的 TiDB 集群](backup-to-aws-s3-using-br.md#aws-账号权限授予的三种方式)，使用 Ceph 作为后端存储测试备份恢复时，是通过 AccessKey 和 SecretKey 模式授权。

以下示例将兼容 S3 的存储（指定路径）上的备份数据恢复到 TiDB 集群。

## 环境准备

参考[环境准备](restore-from-aws-s3-using-br.md#环境准备)

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
          storageClassName: local-storage
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
          storageClassName: local-storage
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
            storageClassName: local-storage
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
            storageClassName: local-storage
            storageSize: 1Gi
        ```

2. 创建好 `Restore` CR 后，可通过以下命令查看恢复的状态：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get rt -n test2 -owide
    ```

以上示例将兼容 S3 的存储（`spec.s3.path` 路径下）中的备份数据恢复到 TiDB 集群 (`spec.to.host`)。有关兼容 S3 的存储的配置项，可以参考 [backup-s3.yaml](backup-to-s3.md#备份数据到兼容-s3-的存储)。

更多 `Restore` CR 字段的详细解释：

* `.spec.metadata.namespace`：`Restore` CR 所在的 namespace。
* `.spec.to.host`：待恢复 TiDB 集群的访问地址。
* `.spec.to.port`：待恢复 TiDB 集群的访问端口。
* `.spec.to.user`：待恢复 TiDB 集群的访问用户。
* `.spec.to.secretName`：存储 `.spec.to.user` 用户的密码的 secret。
* `.spec.storageClassName`：指定恢复时所需的 PV 类型。
* `.spec.storageSize`：指定恢复集群时所需的 PV 大小。该值应大于 TiDB 集群备份的数据大小。

> **注意：**
>
> TiDB Operator 会创建一个 PVC，用于数据恢复，备份数据会先从远端存储下载到 PV，然后再进行恢复。如果恢复完成后想要删掉这个 PVC，可以参考[删除资源](cheat-sheet.md#删除资源)先把恢复 Pod 删掉，然后再把 PVC 删掉。
