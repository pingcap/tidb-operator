---
title: TiDB Operator 1.1.9 Release Notes
---

# TiDB Operator 1.1.9 Release Notes

发布日期：2020 年 12 月 28 日

TiDB Operator 版本：1.1.9

## 优化提升

- 支持使用 `spec.toolImage` 来为 `Backup` 和 `Restore` 指定 Dumpling/TiDB Lightning 的二进制可执行文件 ([#3641](https://github.com/pingcap/tidb-operator/pull/3641), [@BinChenn](https://github.com/BinChenn))

## Bug 修复

- 修复 Prometheus 不能拉取 TiKV Importer 的 metrics ([#3631](https://github.com/pingcap/tidb-operator/pull/3631), [@csuzhangxc](https://github.com/csuzhangxc))
- 修复用 BR 和 GCS 进行备份与恢复时的兼容性问题 ([#3654](https://github.com/pingcap/tidb-operator/pull/3654), [@dragonly](https://github.com/dragonly))
