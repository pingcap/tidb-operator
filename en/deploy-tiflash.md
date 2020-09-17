---
title: Deploy TiFlash in Kubernetes
summary: Learn how to deploy TiFlash in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-tiflash/']
---

# Deploy TiFlash in Kubernetes

This document describes how to deploy TiFlash in Kubernetes.

## Prerequisites

* [Deploy TiDB Operator](deploy-tidb-operator.md).

## Fresh TiFlash deployment

To deploy TiFlash when deploying the TiDB cluster, refer to [Deploy TiDB on General Kubernetes](deploy-on-general-kubernetes.md).

## Add TiFlash to an existing TiDB cluster

Edit the `TidbCluster` Custom Resource:

{{< copyable "shell-regular" >}}

``` shell
kubectl eidt tc ${cluster_name} -n ${namespace}
```

Add the TiFlash configuration as follows:

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

To deploy Enterprise Edition of TiFlash, edit the `dm.yaml` file above to set `spec.tiflash.baseImage` to the enterprise image (`pingcap/tiflash-enterprise`).

For example:

```yaml
spec:
  tiflash:
    baseImage: pingcap/tiflash-enterprise
```

TiFlash supports mounting multiple Persistent Volumes (PVs). If you want to configure multiple PVs for TiFlash, configure multiple `resources` in `tiflash.storageClaims`, each `resources` with a separate `storage request` and `storageClassName`. For example:

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

> **Warning:**
>
> Since TiDB Operator will mount PVs automatically in the **order** of the items in the `storageClaims` list, if you need to add more disks to TiFlash, make sure to append the new item only to the **end** of the original items, and **DO NOT** modify the order of the original items.

To [add TiFlash component to an existing TiDB cluster](https://pingcap.com/docs/stable/tiflash/deploy-tiflash/#add-tiflash-component-to-an-existing-tidb-cluster), `replication.enable-placement-rules` should be set to `true` in PD. After you add the TiFlash configuration in `TidbCluster` by taking the above steps, TiDB Operator will automatically configure `replication.enable-placement-rules: "true"` in PD.

If the server does not have an external network, refer to [deploy the TiDB cluster](deploy-on-general-kubernetes.md#deploy-the-tidb-cluster) to download the required Docker image on the machine with an external network and upload it to the server.

## Configuration notes for different versions

Starting from TiDB Operator v1.1.5, the default configuration of `spec.tiflash.config.config.flash.service_addr` is changed from `${clusterName}-tiflash-POD_NUM.${clusterName}-tiflash-peer.${namespace}.svc:3930` to `0.0.0.0:3930`, and TiFlash needs to configure `spec.tiflash.config.config.flash.service_addr` to `0.0.0.0:3930` since v4.0.5.

Therefore, for different TiFlash and TiDB Operator versions, you need to pay attention to the following configurations:

* If the TiDB Operator version <= v1.1.4
    * If the TiFlash version <= v4.0.4, no need to manually configure `spec.tiflash.config.config.flash.service_addr`.
    * If the TiFlash version >= v4.0.5, you need to set `spec.tiflash.config.config.flash.service_addr` to `0.0.0.0:3930` in the `TidbCluster` CR.
* If the TiDB Operator version >= v1.1.5
    * If the TiFlash version <= v4.0.4, you need to set `spec.tiflash.config.config.flash.service_addr` to `${clusterName}-tiflash-POD_NUM.${clusterName}-tiflash-peer.${namespace}.svc:3930` in the `TidbCluster` CR. `${clusterName}` and `${namespace}` need to be replaced according to the real case.
    * If the TiFlash version >= v4.0.5, no need to manually configure `spec.tiflash.config.config.flash.service_addr`.
    * If you upgrade from TiFlash v4.0.4 or lower versions to TiFlash v4.0.5 or higher versions, you need to delete the configuration of `spec.tiflash.config.config.flash.service_addr` in the `TidbCluster` CR.
