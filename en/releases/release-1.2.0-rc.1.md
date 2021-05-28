---
title: TiDB Operator 1.2.0-rc.1 Release Notes
---

# TiDB Operator 1.2.0-rc.1 Release Notes

Release date: May 28, 2021

TiDB Operator version: 1.2.0-rc.1

## Rolling update changes

- Upgrading TiDB Operator will cause the recreation of the Pump Pod due to [#3973](https://github.com/pingcap/tidb-operator/pull/3973)

## New features

- Support customized labels for TidbCluster Pods and services ([#3892](https://github.com/pingcap/tidb-operator/pull/3892), [@SabaPing](https://github.com/SabaPing) [@july2993](https://github.com/july2993))
- Support full lifecycle management for Pump ([#3973](https://github.com/pingcap/tidb-operator/pull/3973), [@july2993](https://github.com/july2993))

## Improvements

- Mask the backup password in logging ([#3979](https://github.com/pingcap/tidb-operator/pull/3979), [@dveeden](https://github.com/dveeden))
- Add an additional volumeMounts field for Grafana ([#3960](https://github.com/pingcap/tidb-operator/pull/3960), [@mikechengwei](https://github.com/mikechengwei))
- Add several useful additional printout columns for TidbMonitor ([#3958](https://github.com/pingcap/tidb-operator/pull/3958), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor supports writing monitor configuration to PD etcd directly ([#3924](https://github.com/pingcap/tidb-operator/pull/3924), [@mikechengwei](https://github.com/mikechengwei))

## Bug fixes

- Fix the issue that TidbMonitor may not work for DmCluster with TLS enabled ([#3991](https://github.com/pingcap/tidb-operator/pull/3991), [@csuzhangxc](https://github.com/csuzhangxc))
- Fix the wrong count of PD members when scaling out PD ([#3940](https://github.com/pingcap/tidb-operator/pull/3940), [@cvvz](https://github.com/cvvz))
- Fix the issue that DM-master might fail to restart ([#3972](https://github.com/pingcap/tidb-operator/pull/3972), [@csuzhangxc](https://github.com/csuzhangxc))
- Fix the issue that rolling update might happen after changing `configUpdateStrategy` from `InPlace` to `RollingUpdate` ([#3970](https://github.com/pingcap/tidb-operator/pull/3970), [@cvvz](https://github.com/cvvz))
- Fix the issue that backup using Dumpling might fail ([#3986](https://github.com/pingcap/tidb-operator/pull/3986), [@liubog2008](https://github.com/liubog2008))
