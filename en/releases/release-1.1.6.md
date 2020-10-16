---
title: TiDB Operator 1.1.6 Release Notes
---

# TiDB Operator 1.1.6 Release Notes

Release date: October 16, 2020

TiDB Operator version: 1.1.6

## New Features

- Add `spec.br.options` to the Backup and Restore CR to support customizing arguments for BR ([#3360](https://github.com/pingcap/tidb-operator/pull/3360), [@lichunzhu](https://github.com/lichunzhu))
- Add `spec.tikv.evictLeaderTimeout` to TidbCluster CR to make TiKV evict leader timeout configurable ([#3344](https://github.com/pingcap/tidb-operator/pull/3344), [@lichunzhu](https://github.com/lichunzhu))
- Support monitoring multiple TiDB clusters with one TidbMonitor CR when TLS is disabled. `spec.clusterScoped` is added to the TidbMonitor CR and needs to be set to `true` to monitor multiple clusters ([#3308](https://github.com/pingcap/tidb-operator/pull/3308), [@mikechengwei](https://github.com/mikechengwei))
- Support specifying resources for all initcontainers ([#3305](https://github.com/pingcap/tidb-operator/pull/3305), [@shonge](https://github.com/shonge))

## Improvements

- Support passing raw TOML config for TiFlash ([#3355](https://github.com/pingcap/tidb-operator/pull/3355), [@july2993](https://github.com/july2993))
- Support passing raw TOML config for TiKV/PD ([#3342](https://github.com/pingcap/tidb-operator/pull/3342), [@july2993](https://github.com/july2993))
- Support passing raw TOML config for TiDB ([#3327](https://github.com/pingcap/tidb-operator/pull/3327), [@july2993](https://github.com/july2993))
- Support passing raw TOML config for Pump ([#3312](https://github.com/pingcap/tidb-operator/pull/3312), [@july2993](https://github.com/july2993))
- Print proxy log of TiFlash to stdout ([#3345](https://github.com/pingcap/tidb-operator/pull/3345), [@lichunzhu](https://github.com/lichunzhu))
- Add timestamp to the prefix of scheduled backup on GCS ([#3340](https://github.com/pingcap/tidb-operator/pull/3340), [@lichunzhu](https://github.com/lichunzhu))
- Remove the apiserver and related packages ([#3298](https://github.com/pingcap/tidb-operator/pull/3298), [@lonng](https://github.com/lonng))
- Remove the PodRestarter controller and `tidb.pingcap.com/pod-defer-deleting` annotation ([#3296](https://github.com/pingcap/tidb-operator/pull/3296), [@lonng](https://github.com/lonng))
- Use BR metadata to get the total backup size ([#3274](https://github.com/pingcap/tidb-operator/pull/3274), [@lichunzhu](https://github.com/lichunzhu))

## Bug Fixes

- Fix the problem that may bootstrap multiple PD clusters ([#3365](https://github.com/pingcap/tidb-operator/pull/3365), [@lichunzhu](https://github.com/lichunzhu))
