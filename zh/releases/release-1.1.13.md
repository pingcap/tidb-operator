---
title: TiDB Operator 1.1.13 Release Notes
---

# TiDB Operator 1.1.13 Release Notes

发布日期：2021 年 7 月 2 日

TiDB Operator 版本：1.1.13

## 优化提升

- 支持为 TiCDC 配置到下游的 TLS 证书 ([#3926](https://github.com/pingcap/tidb-operator/pull/3926), [@handlerww](https://github.com/handlerww))
- 支持在未为 BR `toolImage` 指定 tag 时将 TiKV 版本作为 tag ([#4048](https://github.com/pingcap/tidb-operator/pull/4048), [@KanShiori](https://github.com/KanShiori))
- 支持在扩缩容 TiDB 过程中协调 PVC ([#4033](https://github.com/pingcap/tidb-operator/pull/4033), [@csuzhangxc](https://github.com/csuzhangxc))
- 支持在 backup 日志中隐藏对数据库密码的展示 ([#3979](https://github.com/pingcap/tidb-operator/pull/3979), [@dveeden](https://github.com/dveeden))

## Bug 修复

- 修复部署异构集群时 TiDB Operator 可能 panic 的问题 ([#4054](https://github.com/pingcap/tidb-operator/pull/4054) [#3965](https://github.com/pingcap/tidb-operator/pull/3965), [@KanShiori](https://github.com/KanShiori) [@july2993](https://github.com/july2993))
- 修复 TiDB 实例缩容后在 TiDB Dashboard 中仍然存在的问题 ([#3929](https://github.com/pingcap/tidb-operator/pull/3929), [@july2993](https://github.com/july2993))
