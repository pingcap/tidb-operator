---
title: TiDB Operator 1.1.5 Release Notes
---

# TiDB Operator 1.1.5 Release Notes

发布日期：2020 年 9 月 18 日

TiDB Operator 版本：1.1.5

## 兼容性变化

- 如果 TiFlash 版本低于 `v4.0.5`，请在将 TiDB Operator 升级到 v1.1.5 和更高版本之前，在 TidbCluster CR 中设置 `spec.tiflash.config.config.flash.service_addr` 为 `{clusterName}-tiflash-POD_NUM.{clusterName}-tiflash-peer.{namespace}.svc:3930` (`{clusterName}` 和 `{namespace}` 需要改为集群实际值)。如果这时需要将 TiFlash 升级到 `v4.0.5` 或更高版本，请同时在 TidbCluster CR 中删除 `spec.tiflash.config.config.flash.service_addr` 项 ([#3191](https://github.com/pingcap/tidb-operator/pull/3191), [@DanielZhangQD](https://github.com/DanielZhangQD))

## 新功能

- 支持为 TiDB/Pump/PD 配置 `serviceAccount` ([#3246](https://github.com/pingcap/tidb-operator/pull/3246), [@july2993](https://github.com/july2993))
- 支持配置 `spec.tikv.config.log-format` 和 `spec.tikv.config.server.max-grpc-send-msg-len` ([#3199](https://github.com/pingcap/tidb-operator/pull/3199), [@kolbe](https://github.com/kolbe))
- 支持配置 TiDB 的 labels 参数 ([#3188](https://github.com/pingcap/tidb-operator/pull/3188), [@howardlau1999](https://github.com/howardlau1999))
- 支持从 TiFlash/TiKV 的 failover 中恢复 ([#3189](https://github.com/pingcap/tidb-operator/pull/3189), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 为 PD/TiKV 添加 `mountClusterClientSecret` 配置，该值为 true 时 Operator 会将 `${cluster_name}-cluster-client-secret` 挂载到 PD/TiKV 容器 ([#3282](https://github.com/pingcap/tidb-operator/pull/3282), [@DanielZhangQD](https://github.com/DanielZhangQD))

## 优化提升

- 支持 TiDB/PD/TiKV 的 v4.0.6 配置 ([#3180](https://github.com/pingcap/tidb-operator/pull/3180), [@lichunzhu](https://github.com/lichunzhu))
- 挂载集群客户端证书到 PD Pod ([#3248](https://github.com/pingcap/tidb-operator/pull/3248), [@weekface](https://github.com/weekface))
- 对于 TiFlash/PD/TiDB，使伸缩实例优先于升级，避免升级失败时无法扩缩容 Pod ([#3187](https://github.com/pingcap/tidb-operator/pull/3187), [@lichunzhu](https://github.com/lichunzhu))
- Pump 支持 `imagePullSecrets` 配置 ([#3214](https://github.com/pingcap/tidb-operator/pull/3214), [@weekface](https://github.com/weekface))
- 更新 TiFlash 的默认配置项 ([#3191](https://github.com/pingcap/tidb-operator/pull/3191), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 移除 TidbMonitor CR 的 ClusterRole 资源 ([#3190](https://github.com/pingcap/tidb-operator/pull/3190), [@weekface](https://github.com/weekface))
- 不再重启 Helm 部署的正常退出的 Drainer ([#3151](https://github.com/pingcap/tidb-operator/pull/3151), [@lichunzhu](https://github.com/lichunzhu))
- tidb-scheduler 高可用策略将 failover pod 纳入考虑 ([#3171](https://github.com/pingcap/tidb-operator/pull/3171), [@cofyc](https://github.com/cofyc))

## Bug 修复

- 修复 TidbMonitor CR 中的 Grafana container 忽略 `Env` 配置的问题 ([#3237](https://github.com/pingcap/tidb-operator/pull/3237), [@tirsen](https://github.com/tirsen))
