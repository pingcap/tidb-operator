---
title: TiDB Operator 1.1 RC.4 Release Notes
---

# TiDB Operator 1.1 RC.4 Release Notes

发布日期：2020 年 5 月 15 日

TiDB Operator 版本：1.1.0-rc.4

## 需要采取的行动

- 每个组件可以使用单独的 TiDB 客户端证书。用户应该将 `Backup` 和 `Restore` CR 中的旧的 TLS 配置迁移到新的配置。更多详细信息，请参考[#2403](https://github.com/pingcap/tidb-operator/pull/2403)([#2403](https://github.com/pingcap/tidb-operator/pull/2403)，[@weekface](https://github.com/weekface))

## 其他值得注意的变化

- 修复了 `pingcap.com/last-applied-configuration` 的标注被同步到 `tc.spec.tidb.service.annotations` 的问题（[#2471](https://github.com/pingcap/tidb-operator/pull/2471)，[@Yisaer](https:// github.com/Yisaer)）
- 修复了 Kubernetes `healthCheckNodePort`  在 TiDB Service 调和过程中一直被修改的问题（[#2438](https://github.com/pingcap/tidb-operator/pull/2438)，[@aylei](https://github.com/aylei)）
- 在 `TidbCluster` 状态中添加了 `TidbMonitorRef`（[#2424](https://github.com/pingcap/tidb-operator/pull/2424)，[@Yisaer](https://github.com/Yisaer)）
- 支持设置远程存储的备份路径前缀（[#2435](https://github.com/pingcap/tidb-operator/pull/2435)，[@onlymellb](https://github.com/onlymellb)）
- 在 CRD 备份中支持自定义的 mydumper 选项（[#2407](https://github.com/pingcap/tidb-operator/pull/2407)，[@onlymellb](https://github.com/onlymellb)）
- 在 `TidbCluster` CR 中支持 TiCDC（[#2338](https://github.com/pingcap/tidb-operator/pull/2338)，[@weekface](https://github.com/weekface)）
- 将 `tidb-backup-manager` 镜像中的 BR 的版本更新为 v3.1.1（[#2425](https://github.com/pingcap/tidb-operator/pull/2425)，[@DanielZhangQD](https://github.com/DanielZhangQD)）
- 支持为 ACK 上的 `TiFlash` 和 `CDC` 创建节点池（[#2420](https://github.com/pingcap/tidb-operator/pull/2420)，[@DanielZhangQD](https://github.com/DanielZhangQD)）
- 支持在 EKS 上为 `TiFlash` 和 `CDC` 创建节点池（[#2413](https://github.com/pingcap/tidb-operator/pull/2413)，[@DanielZhangQD](https://github.com/DanielZhangQD )）
- 启用存储时，为 `TidbMonitor` 暴露 `PVReclaimPolicy`（[#2379](https://github.com/pingcap/tidb-operator/pull/2379)，[@Yisaer](https://github.com/Yisaer)）
- 在 tidb-scheduler 中支持任意基于拓扑的 HA（例如节点区域）（[#2366](https://github.com/pingcap/tidb-operator/pull/2366)，[@PengJi](https://github.com/PengJi)）
- 如果 TiDB 版本低于 4.0.0，跳过 PD dashboard 的 TLS 设置（[#2389](https://github.com/pingcap/tidb-operator/pull/2389)，[@weekface](https://github.com/weekface)）
- 支持使用 BR 对 GCS 进行备份和还原（[#2267](https://github.com/pingcap/tidb-operator/pull/2267)，[@shuijing198799](https://github.com/shuijing198799)）
- 更新 `TiDBConfig` 和 `TiKVConfig` 以支持 `4.0.0-rc` 版本（[#2322](https://github.com/pingcap/tidb-operator/pull/2322)，[@Yisaer](https://github.com/Yisaer)）
- 修复了 `TidbCluster` 中服务类型为 `NodePort` 时，`NodePort` 的值会频繁更改的问题（[#2284](https://github.com/pingcap/tidb-operator/pull/2284)，[@Yisaer](https://github.com/Yisaer)）
- 为 `TidbClusterAutoScaler` 添加了外部策略功能（[#2279](https://github.com/pingcap/tidb-operator/pull/2279)，[@Yisaer](https://github.com/Yisaer)）
- 删除 `TidbMonitor` 时，不会删除 `PVC`（[#2374](https://github.com/pingcap/tidb-operator/pull/2374)，[@Yisaer](https://github.com/Yisaer)）
- 支持为 `TiFlash` 扩缩容（[#2237](https://github.com/pingcap/tidb-operator/pull/2237)，[@DanielZhangQD](https://github.com/DanielZhangQD)）
