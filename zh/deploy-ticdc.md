---
title: 在 Kubernetes 上部署 TiCDC
summary: 了解如何在 Kubernetes 上部署 TiCDC。
category: how-to
---

# 在 Kubernetes 上部署 TiCDC

[TiCDC](https://pingcap.com/docs-cn/stable/ticdc/ticdc-overview/) 是一款 TiDB 增量数据同步工具，本文介绍如何使用 TiDB Operator 在 Kubernetes 上部署 TiCDC。

## 前置条件

* TiDB Operator [部署](deploy-tidb-operator.md)完成。

## 全新部署 TiDB 集群同时部署 TiCDC

参考 [在标准 Kubernetes 上部署 TiDB 集群](deploy-on-general-kubernetes.md)进行部署。

## 在现有 TiDB 集群上新增 TiCDC 组件

1. 编辑 TidbCluster Custom Resource：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. 按照如下示例增加 TiCDC 配置：

    ```yaml
    spec:
      ticdc:
        baseImage: pingcap/ticdc
        replicas: 3
    ```

3. 部署完成后，通过 `kubectl exec` 进入任意一个 TiCDC Pod 进行操作。

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec -it ${pod_name} -n ${namespace} sh
    ```

4. 然后通过 `cdc cli` 进行[管理集群和同步任务](https://pingcap.com/docs-cn/stable/ticdc/manage-ticdc/)。

    {{< copyable "shell-regular" >}}

    ```shell
    /cdc cli capture list --pd=${pd_address}:2379
    ```

    ```shell
    [
            {
                    "id": "6d92386a-73fc-43f3-89de-4e337a42b766",
                    "is-owner": true
            },
            {
                    "id": "b293999a-4168-4988-a4f4-35d9589b226b",
                    "is-owner": false
            }
    ]
    ```

如果服务器没有外网，请参考 [部署 TiDB 集群](deploy-on-general-kubernetes.md#部署-tidb-集群) 在有外网的机器上将用到的 Docker 镜像下载下来并上传到服务器上。
