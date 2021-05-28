---
title: TiDB Operator 1.1.12 Release Notes
---

# TiDB Operator 1.1.12 Release Notes

发布日期：2021 年 4 月 15 日

TiDB Operator 版本：1.1.12

## 新功能

- 支持为备份和恢复 Job 设置自定义环境变量 ([#3833](https://github.com/pingcap/tidb-operator/pull/3833)，[@dragonly](https://github.com/dragonly))
- 支持备份恢复 CR 设置 affinity 和 tolerations ([#3835](https://github.com/pingcap/tidb-operator/pull/3835)，[@dragonly](https://github.com/dragonly))
- 设置 `appendReleaseSuffix` 为 `true` 时，支持 tidb-operator chart 使用新的 service account ([#3819](https://github.com/pingcap/tidb-operator/pull/3819)，[@DanielZhangQD](https://github.com/DanielZhangQD))

## 优化提升

- TiDBInitializer 中增加重试机制，解决 DNS 查询异常处理问题 ([#3884](https://github.com/pingcap/tidb-operator/pull/3884)，[@handlerww](https://github.com/handlerww))
- 在 PD 的扩缩容和容灾过程中增加多 PVC 支持 ([#3820](https://github.com/pingcap/tidb-operator/pull/3820)，[@dragonly](https://github.com/dragonly))
- 优化 `PodsAreChanged` 函数 ([#3901](https://github.com/pingcap/tidb-operator/pull/3901), [@shonge](https://github.com/shonge))

## Bug 修复

- 修复 PD/TiKV 挂载多 PVC 时容量设置错误的问题 ([#3858](https://github.com/pingcap/tidb-operator/pull/3858)，[@dragonly](https://github.com/dragonly))
- 修复创建 `.spec.tidb` 为空并开启 TLS 的 TidbCluster 导致 tidb-controller-manager panic 的问题 ([#3852](https://github.com/pingcap/tidb-operator/pull/3852)，[@dragonly](https://github.com/dragonly))
- 修复 PD 与 DM 的 `UnjoinedMembers` 中 PVC 状态异常的问题 ([#3836](https://github.com/pingcap/tidb-operator/pull/3836), [@dragonly](https://github.com/dragonly))
