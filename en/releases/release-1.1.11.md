---
title: TiDB Operator 1.1.11 Release Notes
---

# TiDB Operator 1.1.11 Release Notes

Release date: February 26, 2021

TiDB Operator version: 1.1.11

## New Features

- Support configuring durations for leader election ([#3794](https://github.com/pingcap/tidb-operator/pull/3794), [@july2993](https://github.com/july2993))
- Add the `tidb_cluster` label for the scrape jobs in TidbMonitor to support monitoring multiple clusters ([#3750](https://github.com/pingcap/tidb-operator/pull/3750), [@mikechengwei](https://github.com/mikechengwei))
- Support setting customized store labels according to the node labels ([#3784](https://github.com/pingcap/tidb-operator/pull/3784), [@L3T](https://github.com/L3T))

## Improvements

- Add TiFlash rolling upgrade logic to avoid all TiFlash stores being unavailable at the same time during the upgrade ([#3789](https://github.com/pingcap/tidb-operator/pull/3789), [@handlerww](https://github.com/handlerww))
- Retrieve the region leader count from TiKV Pod directly instead of from PD to get the accurate count ([#3801](https://github.com/pingcap/tidb-operator/pull/3801), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Print RocksDB and Raft logs to stdout to support collecting and querying the logs in Grafana ([#3768](https://github.com/pingcap/tidb-operator/pull/3768), [@baurine](https://github.com/baurine))
