---
title: 在 Kubernetes 上部署 TiFlash
summary: 了解如何在 Kubernetes 上部署 TiFlash。
category: how-to
---

# 在 Kubernetes 上部署 TiFlash

本文介绍如何在 Kubernetes 上部署 TiFlash。

## 前置条件

* TiDB Operator [部署](deploy-tidb-operator.md)完成。

## 全新部署 TiDB 集群同时部署 TiFlash

参考 [在标准 Kubernetes 上部署 TiDB 集群](deploy-on-general-kubernetes.md)进行部署。

## 在现有 TiDB 集群上新增 TiFlash 组件

编辑 TidbCluster Custom Resource：

{{< copyable "shell-regular" >}}

``` shell
kubectl edit tc ${cluster_name} -n ${namespace}
```

按照如下示例增加 TiFlash 配置：

```yaml
spec:
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 3
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

TiFlash 支持挂载多个 PV，如果要为 TiFlash 配置多个 PV，可以在 `tiflash.storageClaims` 下面配置多项，每一项可以分别配置 `storage reqeust` 和 `storageClassName`，例如：

```yaml
  tiflash:
    baseImage: pingcap/tiflash
    maxFailoverCount: 3
    replicas: 1
    storageClaims:
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
    - resources:
        requests:
          storage: 100Gi
      storageClassName: local-storage
```

> **警告：**
>
> 由于 TiDB Operator 会按照 `storageClaims` 列表中的配置**按顺序**自动挂载 PV，如果需要为 TiFlash 增加磁盘，请确保只在列表原有配置**最后添加**，并且**不能**修改列表中原有配置的顺序。

[新增部署 TiFlash](https://pingcap.com/docs-cn/stable/reference/tiflash/deploy/#%E5%9C%A8%E5%8E%9F%E6%9C%89-tidb-%E9%9B%86%E7%BE%A4%E4%B8%8A%E6%96%B0%E5%A2%9E-tiflash-%E7%BB%84%E4%BB%B6) 需要 PD 配置 `replication.enable-placement-rules: "true"`，通过上述步骤在 TidbCluster 中增加 TiFlash 配置后，TiDB Operator 会自动为 PD 配置 `replication.enable-placement-rules: "true"`。

如果服务器没有外网，请参考[部署 TiDB 集群](deploy-on-general-kubernetes.md#部署-tidb-集群)在有外网的机器上将用到的 Docker 镜像下载下来并上传到服务器上。
