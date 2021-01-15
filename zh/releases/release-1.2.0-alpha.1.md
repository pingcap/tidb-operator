---
title: TiDB Operator 1.2.0-alpha.1 Release Notes
---

# TiDB Operator 1.2.0-alpha.1 Release Notes

发布日期：2021 年 1 月 15 日

TiDB Operator 版本：1.2.0-alpha.1

## 滚动升级改动

- 由于 [#3440](https://github.com/pingcap/tidb-operator/pull/3440) 的改动，升级 TiDB Operator 到 v1.2.0-alpha.1 版本后，TidbMonitor Pod 会被删除重建。

## 新功能

- 跨多个 Kubernetes 集群部署一个 TiDB 集群 ([@L3T](https://github.com/L3T), [@handlerww](https://github.com/handlerww))
- 支持管理 DM 2.0 ([@lichunzhu](https://github.com/lichunzhu), [@BinChenn](https://github.com/BinChenn))
- 通过 PD API 进行弹性伸缩 ([@howardlau1999](https://github.com/howardlau1999))
- 支持灰度升级 TiDB Operator ([#3548](https://github.com/pingcap/tidb-operator/pull/3548), [@shonge](https://github.com/shonge), [#3554](https://github.com/pingcap/tidb-operator/pull/3554), [@cvvz](https://github.com/cvvz))

## 优化提升

- TiDB Lightning chart 支持 local backend ([#3644](https://github.com/pingcap/tidb-operator/pull/3644), [@csuzhangxc](https://github.com/csuzhangxc))
- TiDB Lightning chart 和 TiKV Importer chart 支持 TLS ([#3598](https://github.com/pingcap/tidb-operator/pull/3598), [@csuzhangxc](https://github.com/csuzhangxc))
- TiDB Lightning chart 支持持久化 checkpoint ([#3653](https://github.com/pingcap/tidb-operator/pull/3653), [@csuzhangxc](https://github.com/csuzhangxc))
- TidbMonitor 支持配置 Thanos sidecar ([#3579](https://github.com/pingcap/tidb-operator/pull/3579), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor 管理资源从 Deployment 变为 StatefulSet ([#3440](https://github.com/pingcap/tidb-operator/pull/3440), [@mikechengwei](https://github.com/mikechengwei))

## 其他改进

- 优化队列 rate limiter 间隔 ([#3700](https://github.com/pingcap/tidb-operator/pull/3700), [@dragonly](https://github.com/dragonly))
- 修改 TidbMonitor 自定义告警规则存储目录，从 `tidb:${tidb_image_version}` 变为 `tidb:${initializer_image_version}`。需要注意的是，如果 TidbMonitor 中的 `spec.initializer.version` 和 TidbCluster 中的 TiDB 版本不匹配，升级 TiDB Operator 会导致监控 Pod 重建 ([#3684](https://github.com/pingcap/tidb-operator/pull/3684), [@BinChenn](https://github.com/BinChenn))
