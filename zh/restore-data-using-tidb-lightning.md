---
title: 导入集群数据
summary: 使用 TiDB Lightning 导入集群数据。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/restore-data-using-tidb-lightning/']
---

# 导入集群数据

本文介绍了如何使用 [TiDB Lightning](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-overview) 导入集群数据。

TiDB Lightning 包含两个组件：tidb-lightning 和 tikv-importer。在 Kubernetes 上，tikv-importer 位于单独的 Helm chart 内，被部署为一个副本数为 1 (`replicas=1`) 的 `StatefulSet`；tidb-lightning 位于单独的 Helm chart 内，被部署为一个 `Job`。

目前，TiDB Lightning 支持三种后端：`Importer-backend`、`Local-backend` 、`TiDB-backend`。关于这三种后端的区别和选择，请参阅 [TiDB Lightning 文档](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-backends)。对于 `Importer-backend` 后端，需要分别部署 tikv-importer 与 tidb-lightning；对于 `Local-backend` 或 `TiDB-backend` 后端，仅需要部署 tidb-lightning。

此外，对于 `TiDB-backend` 后端，推荐使用基于 TiDB Operator 新版（v1.1 及以上）的 CustomResourceDefinition (CRD) 实现。具体信息可参考[使用 TiDB Lightning 恢复 GCS 上的备份数据](restore-from-gcs.md)或[使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据](restore-from-s3.md)。

## 部署 TiKV Importer

> **注意：**
>
> 如需要使用 TiDB Lightning 的 `local` 或 `tidb` 后端用于数据恢复，则不需要部署 tikv-importer。

可以通过 `tikv-importer` Helm chart 来部署 tikv-importer，示例如下：

1. 确保 PingCAP Helm 库是最新的：

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm search repo tikv-importer -l
    ```

2. 获取默认的 `values.yaml` 文件以方便自定义：

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tikv-importer --version=${chart_version} > values.yaml
    ```

3. 修改 `values.yaml` 文件以指定目标 TiDB 集群。示例如下：

    {{< copyable "" >}}

    ```yaml
    clusterName: demo
    image: pingcap/tidb-lightning:v5.2.0
    imagePullPolicy: IfNotPresent
    storageClassName: local-storage
    storage: 20Gi
    pushgatewayImage: prom/pushgateway:v0.3.1
    pushgatewayImagePullPolicy: IfNotPresent
    config: |
      log-level = "info"
      [metric]
      job = "tikv-importer"
      interval = "15s"
      address = "localhost:9091"
    ```

    `clusterName` 必须匹配目标 TiDB 集群。

    如果目标 TiDB 集群组件间开启了 TLS (`spec.tlsCluster.enabled: true`)，则可以参考[为 TiDB 集群各个组件生成证书](enable-tls-between-components.md#第一步为-tidb-集群各个组件生成证书)为 TiKV importer 组件生成 Server 端证书，并在 `values.yaml` 中通过配置 `tlsCluster.enabled: true` 开启 TLS 支持。

4. 部署 tikv-importer：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${cluster_name} pingcap/tikv-importer --namespace=${namespace} --version=${chart_version} -f values.yaml
    ```

    > **注意：**
    >
    > tikv-importer 必须与目标 TiDB 集群安装在相同的命名空间中。

## 部署 TiDB Lightning

### 配置 TiDB Lightning

使用如下命令获得 TiDB Lightning 的默认配置：

{{< copyable "shell-regular" >}}

```shell
helm inspect values pingcap/tidb-lightning --version=${chart_version} > tidb-lightning-values.yaml
```

根据需要配置 TiDB Lightning 所使用的后端 `backend`，即将 `values.yaml` 中的 `backend` 设置为 `importer`、`local` 、`tidb` 中的一个。

> **注意：**
>
> 如果使用 [`local` 后端](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-backends#tidb-lightning-local-backend)，则还需要在 `values.yaml` 中设置 `sortedKV` 来创建相应的 PVC 以用于本地 KV 排序。

自 v1.1.10 版本起，tidb-lightning Helm chart 默认会将 [TiDB Lightning 的 checkpoint 信息](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-checkpoints)存储在源数据所在目录内。这样在运行新的 lightning job 时，可以根据 checkpoint 信息进行断点续传。

对于 v1.1.10 之前的版本，可参考 [TiDB Lightning 断点续传](https://docs.pingcap.com/zh/tidb/stable/tidb-lightning-checkpoints)，在 `values.yaml` 中的 `config` 配置下，设置将 checkpoint 信息保存到目标 TiDB 集群、其他 MySQL 协议兼容的数据库或共享存储目录中。

如果目标 TiDB 集群组件间开启了 TLS (`spec.tlsCluster.enabled: true`)，则可以参考[为 TiDB 集群各个组件生成证书](enable-tls-between-components.md#第一步为-tidb-集群各个组件生成证书)为 TiDB Lightning 组件生成 Server 端证书，并在 `values.yaml` 中通过配置 `tlsCluster.enabled: true` 开启集群内部的 TLS 支持。

如果目标 TiDB 集群为 MySQL 客户端开启了 TLS (`spec.tidb.tlsClient.enabled: true`) 并配置了相应的 Client 端证书（对应的 Kubernetes Secret 对象为 `${cluster_name}-tidb-client-secret`），则可以通过在 `values.yaml` 中配置 `tlsClient.enabled: true` 以使 TiDB Lightning 通过 TLS 方式连接 TiDB Server。

如果需要 TiDB Lightning 使用不同的 Client 证书来连接 TiDB Server，则可以参考[为 TiDB 集群颁发两套证书](enable-tls-for-mysql-client.md#第一步为-tidb-集群颁发两套证书)为 TiDB Lightning 组件生成 Client 端证书，并在 `values.yaml` 中通过 `tlsCluster.tlsClientSecretName` 指定对应的 Kubernetes Sceret 对象。

> **注意：**
>
> 如果通过 `tlsCluster.enabled: true` 开启了集群内部的 TLS 支持，但未通过 `tlsClient.enabled: true` 开启 TiDB Lightning 到 TiDB Server 的 TLS 支持，则需要在 `values.yaml` 中的 `config` 内通过如下配置显式地禁用 TiDB Lightning 到 TiDB Server 的 TLS 连接支持。
>
> ```toml
> [tidb]
> tls="false"
> ```

tidb-lightning Helm chart 支持恢复本地或远程的备份数据。

#### 本地模式

本地模式要求备份工具导出的备份数据位于其中一个 Kubernetes 节点上。要启用该模式，你需要将 `dataSource.local.nodeName` 设置为该节点名称，将 `dataSource.local.hostPath` 设置为备份数据目录路径，该路径中需要包含名为 `metadata` 的文件。

#### 远程模式

与本地模式不同，远程模式需要使用 [rclone](https://rclone.org) 将备份工具备份的 tarball 文件从网络存储中下载到 PV 中。远程模式能在 rclone 支持的任何云存储下工作，目前已经有以下存储进行了相关测试：[Google Cloud Storage (GCS)](https://cloud.google.com/storage/)、[Amazon S3](https://aws.amazon.com/s3/) 和 [Ceph Object Storage](https://ceph.com/ceph-storage/object-storage/)。

使用远程模式恢复备份数据的步骤如下：

1. 确保 `values.yaml` 中的 `dataSource.local.nodeName` 和 `dataSource.local.hostPath` 被注释掉。

2. 存储访问授权

    使用 Amazon S3 作为后端存储时，参考 [AWS 账号授权](grant-permissions-to-remote-storage.md#aws-账号授权)。在使用不同的权限授予方式时，需要使用不用的配置。

    使用 Ceph 作为存储后端时，参考[通过 AccessKey 和 SecretKey 授权](grant-permissions-to-remote-storage.md#通过-accesskey-和-secretkey-授权)。

    使用 GCS 作为存储后端时，参考 [GCS 账号授权](grant-permissions-to-remote-storage.md#gcs-账号授权)。

    * 通过 AccessKey 和 SecretKey 授权

        1. 新建一个包含 rclone 配置的 `Secret` 配置文件 `secret.yaml`。rclone 配置示例如下。一般只需要配置一种云存储。

            {{< copyable "" >}}

            ```yaml
            apiVersion: v1
            kind: Secret
            metadata:
              name: cloud-storage-secret
            type: Opaque
            stringData:
              rclone.conf: |
                [s3]
                type = s3
                provider = AWS
                env_auth = false
                access_key_id = ${access_key}
                secret_access_key = ${secret_key}
                region = us-east-1
                [ceph]
                type = s3
                provider = Ceph
                env_auth = false
                access_key_id = ${access_key}
                secret_access_key = ${secret_key}
                endpoint = ${endpoint}
                region = :default-placement
                [gcs]
                type = google cloud storage
                # 该服务账号必须被授予 Storage Object Viewer 角色。
                # 该内容可以通过 `cat ${service-account-file} | jq -c .` 命令获取。
                service_account_credentials = ${service_account_json_file_content}
            ```

        2. 运行以下命令创建 `Secret`：

            {{< copyable "shell-regular" >}}

            ```shell
            kubectl apply -f secret.yaml -n ${namespace}
            ```

    * 通过 IAM 绑定 Pod 授权或者通过 IAM 绑定 ServiceAccount 授权

        使用 Amazon S3 作为存储后端时支持通过 IAM 绑定 Pod 授权或者通过 IAM 绑定 ServiceAccount 授权，此时可省略 `s3.access_key_id` 以及 `s3.secret_access_key`。

        1. 将下面文件存储为 `secret.yaml`。

            {{< copyable "" >}}

            ```yaml
            apiVersion: v1
            kind: Secret
            metadata:
              name: cloud-storage-secret
            type: Opaque
            stringData:
              rclone.conf: |
                [s3]
                type = s3
                provider = AWS
                env_auth = true
                access_key_id =
                secret_access_key =
                region = us-east-1
            ```

        2. 运行以下命令创建 `Secret`：

            {{< copyable "shell-regular" >}}

            ```shell
            kubectl apply -f secret.yaml -n ${namespace}
            ```

3. 将 `dataSource.remote.storageClassName` 设置为 Kubernetes 集群中现有的一个存储类型。

#### Ad hoc 模式

当使用远程模式进行恢复时，如果在恢复过程中由于异常而造成中断、但又不希望重复从网络存储中下载备份数据，则可以使用 Ad hoc 模式直接恢复已通过远程模式下载并解压到 PV 中的数据。步骤如下：

1. 确保 `values.yaml` 中的 `dataSource.local` 和 `dataSource.remote` 均为空配置。

2. 配置 `values.yaml` 中的 `dataSource.adhoc.pvcName` 为使用远程模式时创建的 PVC 名称。

3. 配置 `values.yaml` 中的 `dataSource.adhoc.backupName` 为原备份数据对应的名称，如 `backup-2020-12-17T10:12:51Z` (不包含在网络存储上压缩文件名的 `.tgz` 后缀)。

### 部署 TiDB Lightning

部署 TiDB Lightning 的方式根据不同的权限授予方式及存储方式，有不同的情况。

+ 对于[本地模式](#本地模式)、[Ad hoc 模式](#ad-hoc-模式)、[远程模式](#远程模式)（需要是符合以下三个条件之一的远程模式：使用 Amazon S3 AccessKey 和 SecretKey 权限授予方式、使用 Ceph 作为存储后端、使用 GCS 作为存储后端），运行以下命令部署 TiDB Lightning：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${release_name} pingcap/tidb-lightning --namespace=${namespace} --set failFast=true -f tidb-lightning-values.yaml --version=${chart_version}
    ```

+ 使用 Amazon S3 IAM 绑定 Pod 的授权方式的[远程模式](#远程模式)时，需要完成以下步骤：

    1. 创建 IAM 角色：

        可以参考 [AWS 官方文档](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_create.html)来为账号创建一个 IAM 角色，并且通过 [AWS 官方文档](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_manage-attach-detach.html)为 IAM 角色赋予需要的权限。由于 `Lightning` 需要访问 AWS 的 S3 存储，所以这里给 IAM 赋予了 `AmazonS3FullAccess` 的权限。

    2. 修改 `tidb-lightning-values.yaml`，找到 `annotations` 字段，增加 annotation `iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user`。

    3. 部署 Tidb-Lightning：

        {{< copyable "shell-regular" >}}

        ```shell
        helm install ${release_name} pingcap/tidb-lightning --namespace=${namespace} --set failFast=true -f tidb-lightning-values.yaml --version=${chart_version}
        ```

        > **注意：**
        >
        > `arn:aws:iam::123456789012:role/user` 为步骤 1 中创建的 IAM 角色。

+ 使用 Amazon S3 IAM 绑定 ServiceAccount 授权方式的[远程模式](#远程模式)时：

    1. 在集群上为服务帐户启用 IAM 角色：

        可以参考 [AWS 官方文档](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html) 开启所在的 EKS 集群的 IAM 角色授权。

    2. 创建 IAM 角色：

        可以参考 [AWS 官方文档](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html)创建一个 IAM 角色，为角色赋予 `AmazonS3FullAccess` 的权限，并且编辑角色的 `Trust relationships`。

    3. 绑定 IAM 到 ServiceAccount 资源上：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl annotate sa ${servieaccount} -n eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/user
        ```

    4. 部署 Tidb-Lightning：

        {{< copyable "shell-regular" >}}

        ```shell
        helm install ${release_name} pingcap/tidb-lightning --namespace=${namespace} --set-string failFast=true,serviceAccount=${servieaccount} -f tidb-lightning-values.yaml --version=${chart_version}
        ```

        > **注意：**
        >
        > `arn:aws:iam::123456789012:role/user` 为步骤 1 中创建的 IAM 角色。
        > ${service-account} 为 tidb-lightning 使用的 ServiceAccount，默认为 default。

## 销毁 TiKV Importer 和 TiDB Lightning

目前，TiDB Lightning 只能在线下恢复数据。当恢复过程结束、TiDB 集群需要向外部应用提供服务时，可以销毁 TiDB Lightning 以节省开支。

删除 tikv-importer 的步骤：

* 运行 `helm uninstall ${release_name} -n ${namespace}`。

删除 tidb-lightning 的方法：

* 运行 `helm uninstall ${release_name} -n ${namespace}`。

## 故障诊断

当 TiDB Lightning 未能成功恢复数据时，通常不能简单地直接重启进程，必须进行**手动干预**，否则将很容易出现错误。因此，tidb-lightning 的 `Job` 重启策略被设置为 `Never`。

> **注意：**
>
> 如未设置将 checkpoint 信息持久化保存到目标 TiDB 集群、其他 MySQL 协议兼容的数据库或共享存储目录中，发生故障后，需要清理目标集群中已恢复的部分数据，并重新部署 tidb-lightning 进行数据恢复。

如果 TiDB Lightning 未能成功恢复数据，且已配置将 checkpoint 信息存储在源数据所在目录、其他用户配置的数据库或存储目录中，可采用以下步骤进行手动干预：

1. 运行 `kubectl logs -n ${namespace} ${pod_name}` 查看 log。

    如果使用远程模式进行数据恢复，且异常发生在从网络存储下载数据的过程中，则依据 log 信息进行处理后，直接重新部署 tidb-lightning 进行数据恢复。否则，继续按下述步骤进行处理。

2. 依据 log 并参考 [TiDB Lightning 故障排除指南](https://pingcap.com/docs-cn/stable/troubleshoot-tidb-lightning/)，了解各故障类型的处理方法。

3. 对于不同的故障类型，分别进行处理：

    - 如果需要使用 tidb-lightning-ctl 进行处理：

        1. 设置 `values.yaml` 的 `dataSource` 以确保新 `Job` 将使用发生故障的 `Job` 已有的数据源及 checkpoint 信息：

            - 如果使用本地模式或 Ad hoc 模式，则 `dataSource` 无需修改。

            - 如果使用远程模式，则修改 `dataSource` 为 Ad hoc 模式。其中 `dataSource.adhoc.pvcName` 为原 Helm chart 创建的 PVC 名称，`dataSource.adhoc.backupName` 为待恢复数据的 backup 名称。

        2. 修改 `values.yaml` 中的 `failFast` 为 `false` 并创建用于使用 tidb-lightning-ctl 的 `Job`。

            - TiDB Lightning 会依据 checkpoint 信息检测前一次数据恢复是否发生错误，当检测到错误时会自动中止运行。

            - TiDB Lightning 会依据 checkpoint 信息来避免对已恢复数据的重复恢复，因此创建该 `Job` 不会影响数据正确性。

        3. 当新 `Job` 对应的 pod 运行后，使用 `kubectl logs -n ${namespace} ${pod_name}` 查看 log 并确认新 `Job` 中的 tidb-lightning 已停止进行数据恢复，即 log 中包含类似以下的任意信息：

            - `tidb lightning encountered error`

            - `tidb lightning exit`

        4. 执行 `kubectl exec -it -n ${namespace} ${pod_name} -it -- sh` 命令进入容器。

        5. 运行 `cat /proc/1/cmdline`，获得启动脚本。

        6. 根据启动脚本中的命令行参数，参考 [TiDB Lightning 故障排除指南](https://pingcap.com/docs-cn/stable/troubleshoot-tidb-lightning/)并使用 tidb-lightning-ctl 进行故障处理。

        7. 故障处理完成后，将 `values.yaml` 中的 `failFast` 设置为 `true` 并再次创建新的 `Job` 用于继续数据恢复。

    - 如果不需要使用 tidb-lightning-ctl 进行处理：

        1. 参考 [TiDB Lightning 故障排除指南](https://pingcap.com/docs-cn/stable/troubleshoot-tidb-lightning/)进行故障处理。

        2. 设置 `values.yaml` 的 `dataSource` 以确保新 `Job` 将使用发生故障的 `Job` 已有的数据源及 checkpoint 信息：

            - 如果使用本地模式或 Ad hoc 模式，则 `dataSource` 无需修改。

            - 如果使用远程模式，则修改 `dataSource` 为 Ad hoc 模式。其中 `dataSource.adhoc.pvcName` 为原 Helm chart 创建的 PVC 名称，`dataSource.adhoc.backupName` 为待恢复数据的 backup 名称。

        3. 根据新的 `values.yaml` 创建新的 `Job` 用于继续数据恢复。

4. 故障处理及数据恢复完成后，参考[销毁 TiKV Importer 和 TiDB Lightning](#销毁-tikv-importer-和-tidb-lightning) 删除用于数据恢复的 `Job` 及用于故障处理的 `Job`。
