---
title: TiDB Operator 1.0.5 Release Notes
---

# TiDB Operator 1.0.5 Release Notes

Release date: December 11, 2019

TiDB Operator version: 1.0.5

## v1.0.5 What's New

There is no action required if you are upgrading from [v1.0.4](release-1.0.4.md).

### Scheduled Backup

- Fix the issue that backup failed when `clusterName` is too long ([#1229](https://github.com/pingcap/tidb-operator/pull/1229))

### TiDB Binlog

- It is recommended that TiDB and Pump be deployed on the same node through the `affinity` feature and Pump be dispersed on different nodes through the `anti-affinity` feature. At most only one Pump instance is allowed on each node. We added a guide to the chart. ([#1251](https://github.com/pingcap/tidb-operator/pull/1251))

### Compatibility

- Fix `tidb-scheduler` RBAC permission in Kubernetes v1.16 ([#1282](https://github.com/pingcap/tidb-operator/pull/1282))
- Do not set `DNSPolicy` if `hostNetwork` is disabled to keep backward compatibility ([#1287](https://github.com/pingcap/tidb-operator/pull/1287))

### E2E

- Fix e2e nil point dereference ([#1221](https://github.com/pingcap/tidb-operator/pull/1221))
