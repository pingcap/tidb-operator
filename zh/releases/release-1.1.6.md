---
title: TiDB Operator 1.1.6 Release Notes
---

# TiDB Operator 1.1.6 Release Notes

发布日期：2020 年 10 月 16 日

TiDB Operator 版本：1.1.6

## 兼容性变化

- 由于 [#3342](https://github.com/pingcap/tidb-operator/pull/3342) 的改动，`spec.pd.config` 会自动从现有的 YAML 格式转换成 TOML 格式，如果 `spec.pd.config` 中配置了如下参数，升级 TiDB Operator 到 v1.1.6 之后，这个转换无法自动完成，需要编辑 TidbCluster CR 把对应参数的值从 string 格式修改为 bool 格式，例如，从 `"true"` 改为 `true`。

    - `replication.strictly-match-label`
    - `replication.enable-placement-rules`
    - `schedule.disable-raft-learner`
    - `schedule.disable-remove-down-replica`
    - `schedule.disable-replace-offline-replica`
    - `schedule.disable-make-up-replica`
    - `schedule.disable-remove-extra-replica`
    - `schedule.disable-location-replacement`
    - `schedule.disable-namespace-relocation`
    - `schedule.enable-one-way-merge`
    - `schedule.enable-cross-table-merge`
    - `pd-server.use-region-storage`

## 滚动升级改动

- 由于 [#3305](https://github.com/pingcap/tidb-operator/pull/3305) 的改动，如果 `spec.tidb.annotations` 或者 `spec.tikv.annotations` 中配置了 `tidb.pingcap.com/sysctl-init: "true"`，升级 TiDB Operator 到 v1.1.6 之后 TiDB 或者 TiKV 集群会滚动升级。
- 由于 [#3345](https://github.com/pingcap/tidb-operator/pull/3345) 的改动，如果集群中部署了 TiFlash，升级 TiDB Operator 到 v1.1.6 之后 TiFlash 集群会滚动升级。  

## 新功能

- 添加 `spec.br.options` 支持 Backup 和 Restore CR 自定义 BR 命令行参数 ([#3360](https://github.com/pingcap/tidb-operator/pull/3360), [@lichunzhu](https://github.com/lichunzhu))
- 添加 `spec.tikv.evictLeaderTimeout` 支持配置 TiKV evict leader 超时时间 ([#3344](https://github.com/pingcap/tidb-operator/pull/3344), [@lichunzhu](https://github.com/lichunzhu))
- 支持使用一个 TidbMonitor 监控多个未开启 TLS 的 TiDB 集群。TidbMonitor CR 添加 `spec.clusterScoped` 配置项，监控多个集群时需要设置为 `true` ([#3308](https://github.com/pingcap/tidb-operator/pull/3308), [@mikechengwei](https://github.com/mikechengwei))
- 所有 initcontainers 支持配置资源 ([#3305](https://github.com/pingcap/tidb-operator/pull/3305), [@shonge](https://github.com/shonge))
- 支持部署异构 TiDB 集群 ([#3003](https://github.com/pingcap/tidb-operator/pull/3003) [#3009](https://github.com/pingcap/tidb-operator/pull/3009) [#3113](https://github.com/pingcap/tidb-operator/pull/3113) [#3155](https://github.com/pingcap/tidb-operator/pull/3155) [#3253](https://github.com/pingcap/tidb-operator/pull/3253), [@mikechengwei](https://github.com/mikechengwei))

## 优化提升

- 支持透传 TiFlash 的 TOML 格式配置 ([#3355](https://github.com/pingcap/tidb-operator/pull/3355), [@july2993](https://github.com/july2993))
- 支持透传 TiKV/PD 的 TOML 格式配置 ([#3342](https://github.com/pingcap/tidb-operator/pull/3342), [@july2993](https://github.com/july2993))
- 支持透传 TiDB 的 TOML 格式配置 ([#3327](https://github.com/pingcap/tidb-operator/pull/3327), [@july2993](https://github.com/july2993))
- 支持透传 Pump 的 TOML 格式配置 ([#3312](https://github.com/pingcap/tidb-operator/pull/3312), [@july2993](https://github.com/july2993))
- TiFlash proxy 的日志输出到 stdout ([#3345](https://github.com/pingcap/tidb-operator/pull/3345), [@lichunzhu](https://github.com/lichunzhu))
- 定时备份到 GCS 时目录名称添加相应备份时间  ([#3340](https://github.com/pingcap/tidb-operator/pull/3340), [@lichunzhu](https://github.com/lichunzhu))
- 删除 apiserver 和相关的 packages ([#3298](https://github.com/pingcap/tidb-operator/pull/3298), [@lonng](https://github.com/lonng))
- 删除 PodRestarter controller 相关的支持 ([#3296](https://github.com/pingcap/tidb-operator/pull/3296), [@lonng](https://github.com/lonng))
- 使用 BR metadata 获取备份大小 ([#3274](https://github.com/pingcap/tidb-operator/pull/3274), [@lichunzhu](https://github.com/lichunzhu))

## Bug 修复

- 修复 Discovery 可能导致启动多个 PD 集群的错误 ([#3365](https://github.com/pingcap/tidb-operator/pull/3365), [@lichunzhu](https://github.com/lichunzhu))
