---
title: 使用 BR 恢复 S3 兼容存储上的备份数据
summary: 介绍如何使用 BR 恢复 Amazon S3 兼容存储上的备份数据。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restore-from-aws-s3-using-br/']
---

# 使用 BR 恢复 S3 兼容存储上的备份数据

本文详细描述了如何将存储在 Amazon S3 存储的备份数据恢复到 AWS Kubernetes 环境中的 TiDB 集群，底层通过使用 [BR](https://pingcap.com/docs-cn/stable/br/backup-and-restore-tool/) 进行数据恢复。

本文使用的恢复方式基于 TiDB Operator 新版（v1.1 及以上）的 Custom Resource Definition (CRD) 实现。

以下示例将 Amazon S3 的存储（指定路径）上的备份数据恢复到 AWS Kubernetes 环境中的 TiDB 集群。

## 环境准备

> **注意：**
>
> 如果使用 TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR 会自动调整 `tikv_gc_life_time` 参数，不需要在 Restore CR 中配置 `spec.to` 字段，并且可以省略以下创建 `restore-demo2-tidb-secret` secret 的步骤和[数据库账户权限](#数据库账户权限)步骤。

1. 下载文件 [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml)，并执行以下命令在 `test2` 这个 namespace 中创建备份需要的 RBAC 相关资源：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. 远程存储访问授权。

    如果从 Amazon S3 恢复集群数据，可以使用三种权限授予方式授予权限，参考 [AWS 账号授权](grant-permissions-to-remote-storage.md#aws-账号授权)授权访问兼容 S3 的远程存储；使用 Ceph 作为后端存储测试恢复时，是通过 AccessKey 和 SecretKey 模式授权，设置方式可参考[通过 AccessKey 和 SecretKey 授权](grant-permissions-to-remote-storage.md#通过-accesskey-和-secretkey-授权)。

3. 创建 `restore-demo2-tidb-secret` secret。该 secret 存放用于访问 TiDB 集群的 root 账号和密钥。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=password=${password} --namespace=test2
    ```

## 数据库账户权限

* `mysql.tidb` 表的 `SELECT` 和 `UPDATE` 权限：恢复前后，Restore CR 需要一个拥有该权限的数据库账户，用于调整 GC 时间

## 将指定备份数据恢复到 TiDB 集群

+ 创建 `Restore` CR，通过 accessKey 和 secretKey 授权的方式恢复集群：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f resotre-aws-s3.yaml
    ```

    `restore-aws-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
    spec:
      br:
        cluster: demo2
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # sendCredToTikv: true
      # # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      # to:
      #   host: ${tidb_host}
      #   port: ${tidb_port}
      #   user: ${tidb_user}
      #   secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        secretName: s3-secret
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

+ 创建 `Restore` CR，通过 IAM 绑定 Pod 授权的方式备份集群：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f restore-aws-s3.yaml
    ```

    `restore-aws-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      br:
        cluster: demo2
        sendCredToTikv: false
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

+ 创建 `Restore` CR，通过 IAM 绑定 ServiceAccount 授权的方式备份集群：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f restore-aws-s3.yaml
    ```

    `restore-aws-s3.yaml` 文件内容如下：

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
    spec:
      serviceAccount: tidb-backup-manager
      br:
        cluster: demo2
        sendCredToTikv: false
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

创建好 `Restore` CR 后，可通过以下命令查看恢复的状态：

{{< copyable "shell-regular" >}}

```bash
kubectl get rt -n test2 -o wide
```

以上示例将存储在 Amazon S3 上指定路径 `spec.s3.bucket` 存储桶中 `spec.s3.prefix` 文件夹下的备份数据恢复到 namespace `test2` 中的 TiDB 集群 `demo2`。兼容 S3 的存储相关配置参考 [S3 存储字段介绍](backup-restore-overview.md#s3-存储字段介绍)。

以上示例中，`.spec.br` 中的一些参数项均可省略，如 `logLevel`、`statusAddr`、`concurrency`、`rateLimit`、`checksum`、`timeAgo`、`sendCredToTikv`。更多 `.spec.br` 字段的详细解释参考 [BR 字段介绍](backup-restore-overview.md#br-字段介绍)。

更多 `restore` CR 字段的详细解释参考 [Restore CR 字段介绍](backup-restore-overview.md#restore-cr-字段介绍)。

## 故障诊断

在使用过程中如果遇到问题，可以参考[故障诊断](deploy-failures.md)。
