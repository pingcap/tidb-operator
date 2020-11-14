---
title: TiDB Operator 1.1.7 Release Notes
---

# TiDB Operator 1.1.7 Release Notes

发布日期：2020 年 11 月 13 日

TiDB Operator 版本：1.1.7

## 兼容性变化

- 配置项 `prometheus.spec.config.commandOptions` 的行为发生了变化。该列表中不允许存在重复 flags，否则 Prometheus 无法启动。([#3390](https://github.com/pingcap/tidb-operator/pull/3390), [@mightyguava](https://github.com/mightyguava))

    不能设置的 flags 有：

    - `--web.enable-admin-api`
    - `--web.enable-lifecycle`
    - `--config.file`
    - `--storage.tsdb.path`
    - `--storage.tsdb.retention`

## 新功能

- 新增 `Backup` 和 `Restore` CR 的配置项 `spec.toolImage` 来指定 BR 工具使用的二进制镜像，默认使用 `pingcap/br:${tikv_version}` ([#3471](https://github.com/pingcap/tidb-operator/pull/3471), [@namco1992](https://github.com/namco1992))
- 新增配置项 `spec.pd.storageVolumes`、`spec.tidb.storageVolumes` 和 `spec.tikv.storageVolumes`，支持为 PD、TiDB 和 TiKV 挂载多个 PV ([#3444](https://github.com/pingcap/tidb-operator/pull/3444) [#3425](https://github.com/pingcap/tidb-operator/pull/3425), [@mikechengwei](https://github.com/mikechengwei))
- 新增配置项 `spec.tidb.readinessProbe`，支持使用 `http://127.0.0.0:10080/status` 作为 TiDB 的就绪探针，需要 TiDB 版本 >= v4.0.9 ([#3438](https://github.com/pingcap/tidb-operator/pull/3438), [@july2993](https://github.com/july2993))
- PD leader transfer 支持 advanced StatefulSet 启用的情况 ([#3395](https://github.com/pingcap/tidb-operator/pull/3395), [@tangwz](https://github.com/tangwz))
- 支持在 TidbCluster CR 中设置 `spec.statefulSetUpdateStrategy` 为 `OnDelete`，来控制 StatefulSets 的更新策略 ([#3408](https://github.com/pingcap/tidb-operator/pull/3408), [@cvvz](https://github.com/cvvz))
- 支持故障转移发生时的高可用 (HA) 调度 ([#3419](https://github.com/pingcap/tidb-operator/pull/3419), [@cvvz](https://github.com/cvvz))
- 支持将使用 TiDB Ansible 或 TiUP 部署的 TiDB 集群，以及在同一个 Kubernetes 上部署的 TiDB 集群，在线迁移到新的 TiDB 集群 ([#3226](https://github.com/pingcap/tidb-operator/pull/3226), [@cvvz](https://github.com/cvvz))
- 在 tidb-scheduler 中支持 advanced StatefulSet ([#3388](https://github.com/pingcap/tidb-operator/pull/3388), [@cvvz](https://github.com/cvvz))

## 优化提升

- 当 UP 状态的 stores 数量小于等于 3 时，禁止缩容 TiKV 实例 ([#3367](https://github.com/pingcap/tidb-operator/pull/3367), [@cvvz](https://github.com/cvvz))
- 在 `BackupStatus` 和 `RestoreStatus` 中新增 `phase` 状态，用于在 `kubectl get` 返回结果中显示当前最新状态 ([#3397](https://github.com/pingcap/tidb-operator/pull/3397), [@namco1992](https://github.com/namco1992))
- BR 在备份和恢复前，跳过设置 `tikv_gc_life_time`，由 BR 自动设置，需要 BR & TiKV 版本 >= v4.0.8 ([#3443](https://github.com/pingcap/tidb-operator/pull/3443), [@namco1992](https://github.com/namco1992))

## Bug 修复

- 修复在当前 `TidbCluster` 之外存在 PD member 的情况下，当前 TiDB Cluster 无法把 PD scale 到 0 的 bug ([#3456](https://github.com/pingcap/tidb-operator/pull/3456), [@dragonly](https://github.com/dragonly))
