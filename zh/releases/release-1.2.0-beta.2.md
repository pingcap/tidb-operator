---
title: TiDB Operator 1.2.0-beta.2 Release Notes
---

# TiDB Operator 1.2.0-beta.2 Release Notes

发布日期：2021 年 4 月 29 日

TiDB Operator 版本：1.2.0-beta.2

## 滚动升级改动

- 由于 [#3943](https://github.com/pingcap/tidb-operator/pull/3943) 的改动，升级 TiDB Operator 会导致 TidbMonitor Pod 删除重建。
- 由于 [#3914](https://github.com/pingcap/tidb-operator/pull/3914) 的改动，升级 TiDB Operator 会导致 DM-master Pod 删除重建。

## 新功能

- TidbMonitor 支持监控多个启用了 TLS 的 TidbCluster ([#3867](https://github.com/pingcap/tidb-operator/pull/3867), [@mikechengwei](https://github.com/mikechengwei))
- 支持为所有 TiDB 组件设置 `podSecurityContext` ([#3909](https://github.com/pingcap/tidb-operator/pull/3909), [@liubog2008](https://github.com/liubog2008))
- 支持为所有 TiDB 组件设置 `topologySpreadConstraints` ([#3937](https://github.com/pingcap/tidb-operator/pull/3937), [@liubog2008](https://github.com/liubog2008))
- 支持将 DmCluster 部署在与 TidbCluster 不同的 namespace 内 ([#3914](https://github.com/pingcap/tidb-operator/pull/3914), [@csuzhangxc](https://github.com/csuzhangxc))
- 支持在仅有 namespace 权限时部署 TiDB Operator ([#3896](https://github.com/pingcap/tidb-operator/pull/3896), [@csuzhangxc](https://github.com/csuzhangxc))

## 优化提升

- 为 TidbMonitor Pod 增加 readiness 探测器 ([#3943](https://github.com/pingcap/tidb-operator/pull/3943), [@mikechengwei](https://github.com/mikechengwei))
- 优化 TidbMonitor 对启用了 TLS 的 DmCluster 的监控 ([#3942](https://github.com/pingcap/tidb-operator/pull/3942), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor 支持不生成 Prometheus 的告警规则 ([#3932](https://github.com/pingcap/tidb-operator/pull/3932), [@mikechengwei](https://github.com/mikechengwei))

## Bug 修复

- 修复 TiDB 实例缩容后仍在 TiDB Dashboard 中展示的问题 ([#3929](https://github.com/pingcap/tidb-operator/pull/3929), [@july2993](https://github.com/july2993))
- 修复由于 `status.tikv.stores` 中 `lastHeartbeatTime` 更新而导致 TidbCluster CR 无意义的 sync 的问题 ([#3886](https://github.com/pingcap/tidb-operator/pull/3886), [@songjiansuper](https://github.com/songjiansuper))
