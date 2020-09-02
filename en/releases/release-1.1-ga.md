---
title: TiDB Operator 1.1 GA Release Notes
---

# TiDB Operator 1.1 GA Release Notes

Release date: May 28, 2020

TiDB Operator version: 1.1.0

## Upgrade from v1.0.x

For v1.0.x users, refer to [Upgrade TiDB Operator](../upgrade-tidb-operator.md) to upgrade TiDB Operator in your cluster. Note that you should read the release notes (especially breaking changes and action required items) before the upgrade.

## Breaking changes since v1.0.0

- Change TiDB pod `readiness` probe from `HTTPGet` to `TCPSocket` 4000 port. This will trigger rolling-upgrade for the `tidb-server` component. You can set `spec.paused` to `true` before upgrading tidb-operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your TiDB server ([#2139](https://github.com/pingcap/tidb-operator/pull/2139), [@weekface](https://github.com/weekface))
- `--advertise-address` is configured for `tidb-server`, which would trigger rolling-upgrade for the TiDB server. You can set `spec.paused` to `true` before upgrading TiDB Operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your TiDB server ([#2076](https://github.com/pingcap/tidb-operator/pull/2076), [@cofyc](https://github.com/cofyc))
- `--default-storage-class-name` and `--default-backup-storage-class-name` flags are abandoned, and the storage class defaults to Kubernetes default storage class right now. If you have set default storage class different than Kubernetes default storage class, set them explicitly in your TiDB cluster Helm or YAML files. ([#1581](https://github.com/pingcap/tidb-operator/pull/1581), [@cofyc](https://github.com/cofyc))
- Add the `timezone` support for [all charts](https://github.com/pingcap/tidb-operator/tree/master/charts) ([#1122](https://github.com/pingcap/tidb-operator/pull/1122), [@weekface](https://github.com/weekface)).

    For the `tidb-cluster` chart, we already have the `timezone` option (`UTC` by default). If the user does not change it to a different value (for example, `Asia/Shanghai`), none of the Pods will be recreated.

    If the user changes it to another value (for example, `Aisa/Shanghai`), all the related Pods (add a `TZ` env) will be recreated, namely rolling updated.

    The related Pods include `pump`, `drainer`, `discovery`, `monitor`, `scheduled backup`, `tidb-initializer`, and `tikv-importer`.

    All images' time zone maintained by TiDB Operator is `UTC`. If you use your own images, you need to make sure that the time zone inside your images is `UTC`.

## Other Notable changes

- Fix `TidbCluster` upgrade bug when `PodWebhook` and `Advancend StatefulSet` are both enabled ([#2507](https://github.com/pingcap/tidb-operator/pull/2507), [@Yisaer](https://github.com/Yisaer))
- Support preemption in `tidb-scheduler` ([#2510](https://github.com/pingcap/tidb-operator/pull/2510), [@cofyc](https://github.com/cofyc))
- Update BR to v4.0.0-rc.2 to include the `auto_random` fix ([#2508](https://github.com/pingcap/tidb-operator/pull/2508), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Supports advanced statefulset for TiFlash ([#2469](https://github.com/pingcap/tidb-operator/pull/2469), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Sync Pump before TiDB ([#2515](https://github.com/pingcap/tidb-operator/pull/2515), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Improve performance by removing `TidbControl` lock ([#2489](https://github.com/pingcap/tidb-operator/pull/2489), [@weekface](https://github.com/weekface))
- Support TiCDC in `TidbCluster` ([#2362](https://github.com/pingcap/tidb-operator/pull/2362), [@weekface](https://github.com/weekface))
- Update TiDB/TiKV/PD configuration to 4.0.0 GA version ([#2571](https://github.com/pingcap/tidb-operator/pull/2571), [@Yisaer](https://github.com/Yisaer))
- TiDB Operator will not do failover for PD pods which are not desired ([#2570](https://github.com/pingcap/tidb-operator/pull/2570), [@Yisaer](https://github.com/Yisaer))
