---
title: TiDB Operator 1.1 RC.2 Release Notes
---

# TiDB Operator 1.1 RC.2 Release Notes

Release date: April 15, 2020

TiDB Operator version: 1.1.0-rc.2

## Action Required

- Change TiDB pod `readiness` probe from `HTTPGet` to `TCPSocket` 4000 port. This will trigger rolling-upgrade for the `tidb-server` component. You can set `spec.paused` to `true` before upgrading tidb-operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your TiDB server ([#2139](https://github.com/pingcap/tidb-operator/pull/2139), [@weekface](https://github.com/weekface))

## Notable Changes

- Add `status` field for `TidbAutoScaler` CR ([#2182](https://github.com/pingcap/tidb-operator/pull/2182), [@Yisaer](https://github.com/Yisaer))
- Add `spec.pd.maxFailoverCount` field to limit max failover replicas for PD ([#2184](https://github.com/pingcap/tidb-operator/pull/2184), [@cofyc](https://github.com/cofyc))
- Emit more events for `TidbCluster` and `TidbClusterAutoScaler` to help users know TiDB running status ([#2150](https://github.com/pingcap/tidb-operator/pull/2150), [@Yisaer](https://github.com/Yisaer))
- Add the `AGE` column to show creation timestamp for all CRDs ([#2168](https://github.com/pingcap/tidb-operator/pull/2168), [@cofyc](https://github.com/cofyc))
- Add a switch to skip PD Dashboard TLS configuration ([#2143](https://github.com/pingcap/tidb-operator/pull/2143), [@weekface](https://github.com/weekface))
- Support deploying TiFlash with TidbCluster CR ([#2157](https://github.com/pingcap/tidb-operator/pull/2157), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add TLS support for TiKV metrics API ([#2137](https://github.com/pingcap/tidb-operator/pull/2137), [@weekface](https://github.com/weekface))
- Set PD DashboardConfig when TLS between the MySQL client and TiDB server is enabled ([#2085](https://github.com/pingcap/tidb-operator/pull/2085), [@weekface](https://github.com/weekface))
- Remove unnecessary informer caches to reduce the memory footprint of tidb-controller-manager ([#1504](https://github.com/pingcap/tidb-operator/pull/1504), [@aylei](https://github.com/aylei))
- Fix the failure that Helm cannot load the kubeconfig file when deleting the tidb-operator release during `terraform destroy` ([#2148](https://github.com/pingcap/tidb-operator/pull/2148), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support configuring the Webhook TLS setting by loading a secret ([#2135](https://github.com/pingcap/tidb-operator/pull/2135), [@Yisaer](https://github.com/Yisaer))
- Support TiFlash in TidbCluster CR ([#2122](https://github.com/pingcap/tidb-operator/pull/2122), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix the error that alertmanager couldn't be set in `TidbMonitor` ([#2108](https://github.com/pingcap/tidb-operator/pull/2108), [@Yisaer](https://github.com/Yisaer))
