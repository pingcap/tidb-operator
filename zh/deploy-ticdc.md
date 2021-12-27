---
title: 在 Kubernetes 上部署 TiCDC
summary: 了解如何在 Kubernetes 上部署 TiCDC。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/deploy-ticdc/']
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

   值得注意的是，如果需要部署企业版的 TiCDC，需要将 db.yaml 中 `spec.ticdc.baseImage` 配置为企业版镜像，格式为 `pingcap/ticdc-enterprise`。

   例如:

   ```yaml
   spec:
     ticdc:
       baseImage: pingcap/ticdc-enterprise
   ```

3. 部署完成后，通过 `kubectl exec` 进入任意一个 TiCDC Pod 进行操作。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl exec -it ${pod_name} -n ${namespace} -- sh
    ```

4. 然后通过 `cdc cli` 进行[管理集群和同步任务](https://pingcap.com/docs-cn/stable/ticdc/manage-ticdc/)。

    {{< copyable "shell-regular" >}}

    ```bash
    /cdc cli capture list --pd=http://${cluster_name}-pd:2379
    ```

    ```bash
    [
      {
        "id": "3ed24f6c-22cf-446f-9fe0-bf4a66d00f5b",
        "is-owner": false,
        "address": "${cluster_name}-ticdc-2.${cluster_name}-ticdc-peer.${namespace}.svc:8301"
      },
      {
        "id": "60e98ed7-cd49-45f4-b5ae-d3b85ba3cd96",
        "is-owner": false,
        "address": "${cluster_name}-ticdc-0.${cluster_name}-ticdc-peer.${namespace}.svc:8301"
      },
      {
        "id": "dc3592c0-dace-42a0-8afc-fb8506e8271c",
        "is-owner": true,
        "address": "${cluster_name}-ticdc-1.${cluster_name}-ticdc-peer.${namespace}.svc:8301"
      }
    ]
    ```

    TiCDC 从 v4.0.3 版本开始支持 TLS，TiDB Operator v1.1.3 版本同步支持 TiCDC 开启 TLS 功能。

    如果在创建 TiDB 集群时开启了 TLS，使用 `cdc cli` 请携带 TLS 证书相关参数：

    {{< copyable "shell-regular" >}}

    ```bash
    /cdc cli capture list --pd=https://${cluster_name}-pd:2379 --ca=/var/lib/cluster-client-tls/ca.crt --cert=/var/lib/cluster-client-tls/tls.crt --key=/var/lib/cluster-client-tls/tls.key
    ```

如果服务器没有外网，请参考 [部署 TiDB 集群](deploy-on-general-kubernetes.md#部署-tidb-集群) 在有外网的机器上将用到的 Docker 镜像下载下来并上传到服务器上。
