---
title: 在 Kubernetes 上使用 DM
summary: 了解如何在 Kubernetes 上使用 TiDB DM 迁移数据。
---

# 在 Kubernetes 上使用 DM 迁移数据

[TiDB Data Migration](https://docs.pingcap.com/zh/tidb-data-migration/v2.0) (DM) 是一款支持从 MySQL 或 MariaDB 到 TiDB 的全量数据迁移和增量数据复制的一体化数据迁移任务管理平台。本文介绍如何使用 DM 迁移数据到 TiDB 集群。

## 前置条件

* [TiDB Operator 部署](deploy-tidb-operator.md)完成。
* [TiDB DM 部署](deploy-tidb-dm.md)完成。

> **注意：**
>
> 要求 TiDB Operator 版本 >= 1.2.0。

## 启动 DM 同步任务

有两种方式使用 dmctl 访问 DM-master 服务：

1. 通过进入 DM-master 或 DM-worker pod 使用 image 内置 dmctl 进行操作。

2. 通过[访问 Kubernetes 上的 DM 集群](deploy-tidb-dm.md#访问-kubernetes-上的-dm-集群) 暴露 DM-master 服务，在外部使用 dmctl 访问暴露的 DM-master 服务进行操作。

建议使用方式 1 进行迁移。下文将以方式 1 为例介绍如何启动 DM 同步任务，方式 2 与其区别为 `source.yaml` 与 `task.yaml` 文件位置不同以及 dmctl 的 `master-addr` 配置项需要填写暴露出来的 DM-master 服务地址。

### 进入 Pod

通过 `kubectl exec -ti ${dm_cluster_name}-dm-master-0 -n ${namespace} -- /bin/sh` 命令 attach 到 DM-master Pod。

### 创建数据源

1. 参考[创建数据源](https://docs.pingcap.com/zh/tidb-data-migration/v2.0/migrate-data-using-dm#第-3-步创建数据源)将 MySQL 的相关信息写入到 `source1.yaml` 中。

2. 填写 `source1.yaml` 的 `from.host` 为 Kubernetes 集群内部可以访问的 MySQL host 地址。

3. 填写 `source1.yaml` 的 `relay-dir` 为持久卷在 Pod 内挂载目录 `/var/lib/dm-worker` 下的子目录，如 `/var/lib/dm-worker/relay`。

4. 填写好 `source1.yaml` 文件后，运行 `/dmctl --master-addr ${dm_cluster_name}-dm-master:8261 operate-source create source1.yaml` 命令将 MySQL-1 的数据源加载到 DM 集群中。

5. 对 MySQL-2 及其他数据源，采取同样方式填写数据源 `yaml` 文件中的相关信息，并执行 dmctl 命令将对应的数据源加载到 DM 集群中。

### 配置同步任务

1. 参考[配置同步任务](https://docs.pingcap.com/zh/tidb-data-migration/v2.0/migrate-data-using-dm#第-4-步配置任务)编辑任务配置文件 `task.yaml`。

2. 填写 `task.yaml` 中的 `target-database.host` 为 Kubernetes 集群内部可以访问的 TiDB host 地址。如果是 TiDB Operator 部署的集群，填写 `${tidb_cluster_name}-tidb.${namespace}` 即可。

3. 在 `task.yaml` 文件中，添加 `loaders.${customized_name}.dir` 字段作为全量数据的导入导出目录，其中的 `${customized_name}` 是可以由你自定义的名称，然后将此字段的值填写为持久卷在 Pod 内挂载目录 `/var/lib/dm-worker` 下的子目录，如 `/var/lib/dm-worker/dumped_data`；并在实例配置中进行引用，如 `mysql-instances[0].loader-config-name: "{customized_name}"`。

### 启动/查询/停止同步任务

参考[使用 DM 迁移数据](https://docs.pingcap.com/zh/tidb-data-migration/v2.0/migrate-data-using-dm#第-5-步启动任务) 中的对应步骤即可，注意将 master-addr 填写为 `${dm_cluster_name}-dm-master:8261`。
