---
title: TiDB Operator 1.2.0-rc.1 Release Notes
---

# TiDB Operator 1.2.0-rc.1 Release Notes

发布日期：2021 年 5 月 28 日

TiDB Operator 版本：1.2.0-rc.1

## 滚动升级改动

- 由于 [#3973](https://github.com/pingcap/tidb-operator/pull/3973) 的改动，升级 TiDB Operator 会导致 Pump Pod 删除重建

## 新功能

- 支持为 TidbCluster 中的 Pod 与 service 设置自定义的 label ([#3892](https://github.com/pingcap/tidb-operator/pull/3892), [@SabaPing](https://github.com/SabaPing) [@july2993](https://github.com/july2993))
- 支持对 Pump 的完整生命周期管理 ([#3973](https://github.com/pingcap/tidb-operator/pull/3973), [@july2993](https://github.com/july2993))

## 优化提升

- 在 backup 日志中隐藏对数据库密码的展示 ([#3979](https://github.com/pingcap/tidb-operator/pull/3979), [@dveeden](https://github.com/dveeden))
- 支持为 Grafana 配置额外的 volumeMount ([#3960](https://github.com/pingcap/tidb-operator/pull/3960), [@mikechengwei](https://github.com/mikechengwei))
- 为 TidbMonitor 增加额外的信息展示列 ([#3958](https://github.com/pingcap/tidb-operator/pull/3958), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor 支持将配置信息直接写入到 PD 的 etcd 中 ([#3924](https://github.com/pingcap/tidb-operator/pull/3924), [@mikechengwei](https://github.com/mikechengwei))

## Bug 修复

- 修复 TidbMonitor 可能无法对启用了 TLS 的 DmCluster 进行监控的问题 ([#3991](https://github.com/pingcap/tidb-operator/pull/3991), [@csuzhangxc](https://github.com/csuzhangxc))
- 修复 PD 在扩容过程中 member 数量统计不正确的问题 ([#3940](https://github.com/pingcap/tidb-operator/pull/3940), [@cvvz](https://github.com/cvvz))
- 修复 DM-master 可能无法成功重启的问题 ([#3972](https://github.com/pingcap/tidb-operator/pull/3972), [@csuzhangxc](https://github.com/csuzhangxc))
- 修复将 `configUpdateStrategy` 从 `InPlace` 修改为 `RollingUpdate` 后可能造成的 TidbCluster 组件滚动更新的问题 ([#3970](https://github.com/pingcap/tidb-operator/pull/3970), [@cvvz](https://github.com/cvvz))
- 修复使用 Dumpling 备份数据时可能失败的问题 ([#3986](https://github.com/pingcap/tidb-operator/pull/3986), [@liubog2008](https://github.com/liubog2008))
