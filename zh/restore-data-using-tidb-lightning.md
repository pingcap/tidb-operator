---
title: 使用 TiDB Lightning 恢复 Kubernetes 上的集群数据
summary: 使用 TiDB Lightning 快速恢复 Kubernetes 上的 TiDB 集群数据。
category: how-to
---

# 使用 TiDB Lightning 恢复 Kubernetes 上的集群数据

本文介绍了如何使用 [TiDB Lightning](https://github.com/pingcap/tidb-lightning) 快速恢复 Kubernetes 上的 TiDB 集群数据。

TiDB Lightning 包含两个组件：tidb-lightning 和 tikv-importer。在 Kubernetes 上，tikv-importer 位于单独的 Helm chart 内，被部署为一个副本数为 1 (`replicas=1`) 的 `StatefulSet`；tidb-lightning 位于单独的 Helm chart 内，被部署为一个 `Job`。

为了使用 TiDB Lightning 恢复数据，tikv-importer 和 tidb-lightning 都必须分别部署。

## 部署 tikv-importer

可以通过 `tikv-importer` Helm chart 来部署 tikv-importer，示例如下：

1. 确保 PingCAP Helm 库是最新的：

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm search tikv-importer -l
    ```

2. 获取默认的 `values.yaml` 文件以方便自定义：

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tikv-importer --version=${chart_version} > values.yaml
    ```

3. 修改 `values.yaml` 文件以指定目标 TiDB 集群。示例如下：

    ```yaml
    clusterName: demo
    image: pingcap/tidb-lightning:v3.0.8
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

4. 部署 tikv-importer：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tikv-importer --name=${cluster_name} --namespace=${namespace} --version=${chart_version} -f values.yaml
    ```

    > **注意：**
    >
    > tikv-importer 必须与目标 TiDB 集群安装在相同的命名空间中。

## 部署 tidb-lightning

### 配置 TiDB Lightning

使用如下命令获得 TiDB Lightning 的默认配置：

{{< copyable "shell-regular" >}}

```shell
helm inspect values pingcap/tidb-lightning --version=${chart_version} > tidb-lightning-values.yaml
```

tidb-lightning Helm chart 支持恢复本地或远程的备份数据。

* 本地模式：

    本地模式要求 Mydumper 备份数据位于其中一个 Kubernetes 节点上。要启用该模式，你需要将 `dataSource.local.nodeName` 设置为该节点名称，将 `dataSource.local.hostPath` 设置为 Mydumper 备份数据目录路径，该路径中需要包含名为 `metadata` 的文件。

* 远程模式：

    与本地模式不同，远程模式需要使用 [rclone](https://rclone.org) 将 Mydumper 备份 tarball 文件从网络存储中下载到 PV 中。远程模式能在 rclone 支持的任何云存储下工作，目前已经有以下存储进行了相关测试：[Google Cloud Storage (GCS)](https://cloud.google.com/storage/)、[Amazon S3](https://aws.amazon.com/s3/) 和 [Ceph Object Storage](https://ceph.com/ceph-storage/object-storage/)。

    使用远程模式恢复备份数据的步骤如下：

    1. 确保 `values.yaml` 中的 `dataSource.local.nodeName` 和 `dataSource.local.hostPath` 被注释掉。

    2. 新建一个包含 rclone 配置的 `Secret`。rclone 配置示例如下。一般只需要配置一种云存储。有关其他的云存储，请参考 [rclone 官方文档](https://rclone.org/)。和使用 BR 和 Mydumper 进行数据恢复时一样，使用 Amazon S3 作为后端存储时，同样存在三种权限授予方式，参考[使用 BR 工具备份 AWS 上的 TiDB 集群](backup-to-aws-s3-using-br.md#aws-账号权限授予的三种方式)。在使用不同的权限授予方式时，需要使用不用的配置。

       + 使用 Amazon S3 AccessKey 和 SecretKey 权限授予方式，或者使用 Ceph、GCS 作为存储后端时:
    
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
    
        + 使用 Amazon S3 IAM 绑定 Pod 的授权方式或者 Amazon S3 IAM 绑定 ServiceAccount 授权方式时，可以省略 `s3.access_key_id` 以及 `s3.secret_access_key：
    
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

            使用你的实际配置替换上述配置中的占位符，并将该文件存储为 `secret.yaml`。然后通过 `kubectl apply -f secret.yaml -n ${namespace}` 命令创建该 `Secret`。

    3. 将 `dataSource.remote.storageClassName` 设置为 Kubernetes 集群中现有的一个存储类型。

### 部署 TiDB Lightning

部署 TiDB Lightning 的方式根据不同的权限授予方式及存储方式，有不同的情况。

+ 使用 Amazon S3 AccessKey 和 SecretKey 权限授予方式，或者使用 Ceph，GCS 作为存储后端时，运行以下命令部署 TiDB Lightning：

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-lightning --name=${release_name} --namespace=${namespace} --set failFast=true -f tidb-lightning-values.yaml --version=${chart_version}
    ```

+ 使用 Amazon S3 IAM 绑定 Pod 的授权方式时，需要做以下步骤：

    1. 创建 IAM 角色：

        可以参考 [AWS 官方文档](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_create.html)来为账号创建一个 IAM 角色，并且通过 [AWS 官方文档](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_manage-attach-detach.html)为 IAM 角色赋予需要的权限。由于 `Lightning` 需要访问 AWS 的 S3 存储，所以这里给 IAM 赋予了 `AmazonS3FullAccess` 的权限。

    2. 修改 tidb-lightning-values.yaml, 找到字段 `annotations`，增加 annotation `iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user`。

    3. 部署 Tidb-Lightning：

        {{< copyable "shell-regular" >}}

        ```shell
        helm install pingcap/tidb-lightning --name=${release_name} --namespace=${namespace} --set failFast=true -f tidb-lightning-values.yaml --version=${chart_version}
        ```

        > **注意：**
        >
        > `arn:aws:iam::123456789012:role/user` 为步骤 1 中创建的 IAM 角色。

+ 使用 Amazon S3 IAM 绑定 ServiceAccount 授权方式时：

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
        helm install pingcap/tidb-lightning --name=${release_name} --namespace=${namespace} --set-string failFast=true,serviceAccount=${servieaccount} -f tidb-lightning-values.yaml --version=${chart_version}
        ```

        > **注意：**
        >
        > `arn:aws:iam::123456789012:role/user` 为步骤 1 中创建的 IAM 角色。
        > ${service-account} 为 tidb-lightning 使用的 ServiceAccount，默认为 default。

当 TiDB Lightning 未能成功恢复数据时，不能简单地直接重启进程，必须进行**手动干预**，否则将很容易出现错误。因此，tidb-lightning 的 `Job` 重启策略被设置为 `Never`。

如果 TiDB Lightning 未能成功恢复数据，需要采用以下步骤进行手动干预：

1. 运行 `kubectl delete job -n ${namespace} ${release_name}-tidb-lightning`，删除 lightning `Job`。

2. 运行 `helm template pingcap/tidb-lightning --name ${release_name} --set failFast=false -f tidb-lightning-values.yaml | kubectl apply -n ${namespace} -f -`，重新创建禁用 `failFast` 的 lightning `Job`。

3. 当 lightning pod 重新运行时，在 lightning 容器中执行 `kubectl exec -it -n ${namespace} ${pod_name} sh` 命令。

4. 运行 `cat /proc/1/cmdline`，获得启动脚本。

5. 参考[故障排除指南](https://pingcap.com/docs-cn/v3.0/how-to/troubleshoot/tidb-lightning/#tidb-lightning-troubleshooting)，对 lightning 进行诊断。

## 销毁 TiDB Lightning

目前，TiDB Lightning 只能在线下恢复数据。当恢复过程结束、TiDB 集群需要向外部应用提供服务时，可以销毁 TiDB Lightning 以节省开支。

删除 tikv-importer 的步骤：

* 运行 `helm delete ${release_name} --purge`。

删除 tidb-lightning 的方法：

* 运行 `helm delete ${release_name} --purge`。
