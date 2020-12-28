---
title: TiDB Operator 1.1.9 Release Notes
---

# TiDB Operator 1.1.9 Release Notes

Release date: December 28, 2020

TiDB Operator version: 1.1.9

## Improvements

- Support `spec.toolImage` for `Backup` & `Restore` to define the image used to provide the Dumpling/TiDB Lightning binary executables ([#3641](https://github.com/pingcap/tidb-operator/pull/3641), [@BinChenn](https://github.com/BinChenn))

## Bug Fixes

- Fix the issue that Prometheus can't pull metrics for TiKV Importer ([#3631](https://github.com/pingcap/tidb-operator/pull/3631), [@csuzhangxc](https://github.com/csuzhangxc))

- Fix the compatibility issue for using BR to back up/restore from/to GCS ([#3654](https://github.com/pingcap/tidb-operator/pull/3654), [@dragonly](https://github.com/dragonly))
