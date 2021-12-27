---
title: TiCDC 组件同步数据到开启 TLS 的下游服务
summary: 了解如何让 TiCDC 组件同步数据到开启 TLS 的下游服务
---

# 使用 TiCDC 组件同步数据到开启 TLS 的下游服务

本文介绍在 Kubernetes 上如何让 TiCDC 组件同步数据到开启 TLS 的下游服务。

## 准备条件

在开始之前，请进行以下准备工作：

- 部署一个下游服务，并开启客户端 TLS 认证。
- 生成客户端访问下游服务所需要的密钥文件。

## TiCDC 同步数据到开启 TLS 的下游服务

1. 创建一个 Kubernetes Secret 对象，此对象需要包含用于访问下游服务的客户端 TLS 证书。证书来自于你为客户端生成的密钥文件。
    
    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic ${secret_name} --namespace=${cluster_namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

2. 挂载证书文件到 TiCDC Pod。

    * 如果你还未部署 TiDB 集群，在 TidbCluster CR 定义中添加 `spec.ticdc.tlsClientSecretNames` 字段，然后部署 TiDB 集群。

    * 如果你已经部署了 TiDB 集群，执行 `kubectl edit tc ${cluster_name} -n ${cluster_namespace}`，并添加 `spec.ticdc.tlsClientSecretNames` 字段，然后等待 TiCDC 的 Pod 自动滚动更新。

    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      name: ${cluster_name}
      namespace: ${cluster_namespace}
    spec:
      # ...
      ticdc:
        baseImage: pingcap/ticdc
        version: "v5.0.1"
        # ...
        tlsClientSecretNames:
        - ${secret_name}
    ```

    TiCDC Pod 运行后，创建的 Kubernetes Secret 对象会被挂载到 TiCDC 的 Pod。你可以在 Pod 内的 `/var/lib/sink-tls/${secret_name}` 目录找到被挂载的密钥文件。

3. 通过 `cdc cli` 工具创建同步任务。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl exec ${cluster_name}-ticdc-0 -- /cdc cli changefeed create --pd=https://${cluster_name}-pd:2379 --sink-uri="mysql://${user}:{$password}@${downstream_service}/?ssl-ca=/var/lib/sink-tls/${secret_name}/ca.crt&ssl-cert=/var/lib/sink-tls/${secret_name}/tls.crt&ssl-key=/var/lib/sink-tls/${secret_name}/tls.key"
    ```
