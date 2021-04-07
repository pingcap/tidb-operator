---
title: TiDB Operator 1.2.0-beta.1 Release Notes
---

# TiDB Operator 1.2.0-beta.1 Release Notes

发布日期：2021 年 4 月 7 日

TiDB Operator 版本：1.2.0-beta.1

## 新功能

- 支持为备份和恢复 Job 设置自定义环境变量 ([#3833](https://github.com/pingcap/tidb-operator/pull/3833)，[@dragonly](https://github.com/dragonly))
- 支持为 TidbMonitor 配置额外的 volume 和 volumeMount ([#3855](https://github.com/pingcap/tidb-operator/pull/3855)，[@mikechengwei](https://github.com/mikechengwei))
- 支持备份恢复 CR 设置 affinity 和 tolerations ([#3835](https://github.com/pingcap/tidb-operator/pull/3835)，[@dragonly](https://github.com/dragonly))
- 设置 `appendReleaseSuffix` 为 `true` 时，支持 tidb-operator chart 使用新的 service account ([#3819](https://github.com/pingcap/tidb-operator/pull/3819)，[@DanielZhangQD](https://github.com/DanielZhangQD))

## 优化提升

- TiDBInitializer 中增加重试机制，解决 DNS 查询异常处理问题 ([#3884](https://github.com/pingcap/tidb-operator/pull/3884)，[@handlerww](https://github.com/handlerww))
- 优化 Thanos 的 example yaml ([#3726](https://github.com/pingcap/tidb-operator/pull/3726)，[@mikechengwei](https://github.com/mikechengwei))
- 滚动更新过程中，等待 TiKV Pod 升级完成之后再删除 evict leader scheduler  ([#3724](https://github.com/pingcap/tidb-operator/pull/3724)，[@handlerww](https://github.com/handlerww))
- 在 PD 的扩缩容和容灾过程中增加多 PVC 支持 ([#3820](https://github.com/pingcap/tidb-operator/pull/3820)，[@dragonly](https://github.com/dragonly))
- 在 TiKV 的扩缩容过程中增加多 PVC 支持([#3816](https://github.com/pingcap/tidb-operator/pull/3816)，[@dragonly](https://github.com/dragonly))
- 支持调整 TiDB PVC 容量 ([#3891](https://github.com/pingcap/tidb-operator/pull/3891)，[@dragonly](https://github.com/dragonly))

## Bug 修复

- 修复 PD/TiKV 挂载多 PVC 时容量设置错误的问题 ([#3858](https://github.com/pingcap/tidb-operator/pull/3858)，[@dragonly](https://github.com/dragonly))
- 修复创建 `.spec.tidb` 为空并开启 TLS 的 TidbCluster 导致 tidb-controller-manager panic 的问题 ([#3852](https://github.com/pingcap/tidb-operator/pull/3852)，[@dragonly](https://github.com/dragonly))
- 修复 TidbMonitor 外部标签包含一些无法识别的环境变量的问题 ([#3785](https://github.com/pingcap/tidb-operator/pull/3785)，[@mikechengwei](https://github.com/mikechengwei))
