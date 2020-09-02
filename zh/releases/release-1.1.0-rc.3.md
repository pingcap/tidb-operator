---
title: TiDB Operator 1.1 RC.3 Release Notes
---

# TiDB Operator 1.1 RC.3 Release Notes

Release date: April 30, 2020

TiDB Operator version: 1.1.0-rc.3

## Notable Changes

- Skip auto-failover when pods are not scheduled and perform recovery operation no matter what state failover pods are in ([#2263](https://github.com/pingcap/tidb-operator/pull/2263), [@cofyc](https://github.com/cofyc))
- Support `TiFlash` metrics in `TidbMonitor` ([#2341](https://github.com/pingcap/tidb-operator/pull/2341), [@Yisaer](https://github.com/Yisaer))
- Do not print `rclone` config in the Pod logs ([#2343](https://github.com/pingcap/tidb-operator/pull/2343), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Using `Patch` in `periodicity` controller to avoid updating `StatefulSet` to the wrong state ([#2332](https://github.com/pingcap/tidb-operator/pull/2332), [@Yisaer](https://github.com/Yisaer))
- Set `enable-placement-rules` to `true` for PD if TiFlash is enabled in the cluster ([#2328](https://github.com/pingcap/tidb-operator/pull/2328), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support `rclone` options in the Backup and Restore CR ([#2318](https://github.com/pingcap/tidb-operator/pull/2318), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix the issue that statefulsets are updated during each sync even if no changes are made to the config ([#2308](https://github.com/pingcap/tidb-operator/pull/2308), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support configuring `Ingress` in `TidbMonitor` ([#2314](https://github.com/pingcap/tidb-operator/pull/2314), [@Yisaer](https://github.com/Yisaer))
- Fix a bug that auto-created failover pods can't be deleted when they are in the failed state ([#2300](https://github.com/pingcap/tidb-operator/pull/2300), [@cofyc](https://github.com/cofyc))
- Add useful `Event` in `TidbCluster` during upgrading and scaling when `admissionWebhook.validation.pods` in operator configuration is enabled ([#2305](https://github.com/pingcap/tidb-operator/pull/2305), [@Yisaer](https://github.com/Yisaer))
- Fix the issue that services are updated during each sync even if no changes are made to the service configuration ([#2299](https://github.com/pingcap/tidb-operator/pull/2299), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix a bug that would cause panic in statefulset webhook when the update strategy of `StatefulSet` is not `RollingUpdate` ([#2291](https://github.com/pingcap/tidb-operator/pull/2291), [@Yisaer](https://github.com/Yisaer))
- Fix a panic in syncing `TidbClusterAutoScaler` status when the target `TidbCluster` does not exist ([#2289](https://github.com/pingcap/tidb-operator/pull/2289), [@Yisaer](https://github.com/Yisaer))
- Fix the `pdapi` cache issue while the cluster TLS is enabled ([#2275](https://github.com/pingcap/tidb-operator/pull/2275), [@weekface](https://github.com/weekface))
- Fix the config error in restore ([#2250](https://github.com/pingcap/tidb-operator/pull/2250), [@Yisaer](https://github.com/Yisaer))
- Support failover for TiFlash ([#2249](https://github.com/pingcap/tidb-operator/pull/2249), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Update the default `eks` version in terraform scripts to 1.15 ([#2238](https://github.com/pingcap/tidb-operator/pull/2238), [@Yisaer](https://github.com/Yisaer))
- Support upgrading for TiFlash ([#2246](https://github.com/pingcap/tidb-operator/pull/2246), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add `stderr` logs from BR to the backup-manager logs ([#2213](https://github.com/pingcap/tidb-operator/pull/2213), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add field `TiKVEncryptionConfig` in `TiKVConfig`, which defines how to encrypt data key and raw data in TiKV, and how to back up and restore the master key. See the description for details in `tikv_config.go` ([#2151](https://github.com/pingcap/tidb-operator/pull/2151), [@shuijing198799](https://github.com/shuijing198799))
