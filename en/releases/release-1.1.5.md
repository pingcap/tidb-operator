---
title: TiDB Operator 1.1.5 Release Notes
---

# TiDB Operator 1.1.5 Release Notes

Release date: September 18, 2020

TiDB Operator version: 1.1.5

## Compatibility Changes

- If the TiFlash version is earlier than `v4.0.5`, please set `spec.tiflash.config.config.flash.service_addr: {clusterName}-tiflash-POD_NUM.{clusterName}-tiflash-peer.{namespace}.svc:3930` in the TidbCluster CR (`{clusterName}` and `{namespace}` need to be replaced with the real value) before upgrading TiDB Operator to v1.1.5 or later versions. When TiFlash is going to be upgraded to `v4.0.5` or later versions, please remove `spec.tiflash.config.config.flash.service_addr` in the TidbCluster CR at the same time ([#3191](https://github.com/pingcap/tidb-operator/pull/3191), [@DanielZhangQD](https://github.com/DanielZhangQD))

## New Features

- Support configuring `serviceAccount` for TiDB/Pump/PD ([#3246](https://github.com/pingcap/tidb-operator/pull/3246), [@july2993](https://github.com/july2993))
- Support configuring `spec.tikv.config.log-format` and `spec.tikv.config.server.max-grpc-send-msg-len` ([#3199](https://github.com/pingcap/tidb-operator/pull/3199), [@kolbe](https://github.com/kolbe))
- Support labels configuration for TiDB ([#3188](https://github.com/pingcap/tidb-operator/pull/3188), [@howardlau1999](https://github.com/howardlau1999))
- Support recovery from failover for TiFlash and TiKV ([#3189](https://github.com/pingcap/tidb-operator/pull/3189), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add `mountClusterClientSecret` configuration for PD and TiKV. If the config is set to `true`, TiDB Operator will mount the `${cluster_name}-cluster-client-secret` to the PD or TiKV containers ([#3282](https://github.com/pingcap/tidb-operator/pull/3282), [@DanielZhangQD](https://github.com/DanielZhangQD))

## Improvements

- Adapt TiDB/PD/TiKV configurations to v4.0.6 ([#3180](https://github.com/pingcap/tidb-operator/pull/3180), [@lichunzhu](https://github.com/lichunzhu))
- Support mounting the cluster client certificate to PD pod ([#3248](https://github.com/pingcap/tidb-operator/pull/3248), [@weekface](https://github.com/weekface))
- Scaling takes precedence over upgrading for TiFlash/PD/TiDB. This is to avoid that Pods cannot be scaled in case of upgrade failure ([#3187](https://github.com/pingcap/tidb-operator/pull/3187), [@lichunzhu](https://github.com/lichunzhu))
- Support the `imagePullSecrets` configuration for Pump ([#3214](https://github.com/pingcap/tidb-operator/pull/3214), [@weekface](https://github.com/weekface))
- Update the default configuration for TiFlash ([#3191](https://github.com/pingcap/tidb-operator/pull/3191), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Remove `ClusterRole` from TidbMonitor CR ([#3190](https://github.com/pingcap/tidb-operator/pull/3190), [@weekface](https://github.com/weekface))
- Drainer that are deployed by Helm and exits normally will no longer be restarted ([#3151](https://github.com/pingcap/tidb-operator/pull/3151), [@lichunzhu](https://github.com/lichunzhu))
- The tidb-scheduler HA strategy takes failover pods into consideration ([#3171](https://github.com/pingcap/tidb-operator/pull/3171), [@cofyc](https://github.com/cofyc))

## Bug Fixes

- Fix the problem that the `Env` settings are ignored for the Grafana container in the TidbMonitor CR ([#3237](https://github.com/pingcap/tidb-operator/pull/3237), [@tirsen](https://github.com/tirsen))
