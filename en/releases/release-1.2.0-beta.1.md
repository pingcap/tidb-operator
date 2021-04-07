---
title: TiDB Operator 1.2.0-beta.1 Release Notes
---

# TiDB Operator 1.2.0-beta.1 Release Notes

Release date: April 7, 2021

TiDB Operator version: 1.2.0-beta.1

## New Features

- Support setting customized environment variables for backup and restore job containers ([#3833](https://github.com/pingcap/tidb-operator/pull/3833), [@dragonly](https://github.com/dragonly))
- Add additional volume and volumeMount configurations to TidbMonitor([#3855](https://github.com/pingcap/tidb-operator/pull/3855), [@mikechengwei](https://github.com/mikechengwei))
- Support affinity and tolerations in backup/restore CR ([#3835](https://github.com/pingcap/tidb-operator/pull/3835), [@dragonly](https://github.com/dragonly))
- The resources in the tidb-operator chart use the new service account when `appendReleaseSuffix` is set to `true` ([#3819](https://github.com/pingcap/tidb-operator/pull/3819), [@DanielZhangQD](https://github.com/DanielZhangQD))

## Improvements

- Add retry for DNS lookup failure exception in TiDBInitializer ([#3884](https://github.com/pingcap/tidb-operator/pull/3884), [@handlerww](https://github.com/handlerww))
- Optimize thanos example yaml files ([#3726](https://github.com/pingcap/tidb-operator/pull/3726), [@mikechengwei](https://github.com/mikechengwei))
- Delete the evict leader scheduler after TiKV Pod is recreated during the rolling update ([#3724](https://github.com/pingcap/tidb-operator/pull/3724), [@handlerww](https://github.com/handlerww))
- Support multiple PVCs for PD during scaling and failover ([#3820](https://github.com/pingcap/tidb-operator/pull/3820), [@dragonly](https://github.com/dragonly))
- Support multiple PVCs for TiKV during scaling ([#3816](https://github.com/pingcap/tidb-operator/pull/3816), [@dragonly](https://github.com/dragonly))
- Support PVC resizing for TiDB ([#3891](https://github.com/pingcap/tidb-operator/pull/3891), [@dragonly](https://github.com/dragonly))

## Bug Fixes

- Fix the issue that PVCs will be set to incorrect size if multiple PVCs are configured for PD/TiKV ([#3858](https://github.com/pingcap/tidb-operator/pull/3858), [@dragonly](https://github.com/dragonly))
- Fix the panic issue when `.spec.tidb` is not set in the TidbCluster CR with TLS enabled ([#3852](https://github.com/pingcap/tidb-operator/pull/3852), [@dragonly](https://github.com/dragonly))
- Fix the issue that some unrecognized environment variables are included in the external labels of the TidbMonitor ([#3785](https://github.com/pingcap/tidb-operator/pull/3785), [@mikechengwei](https://github.com/mikechengwei))
