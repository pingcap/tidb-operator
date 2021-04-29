---
title: TiDB Operator 1.2.0-beta.2 Release Notes
---

# TiDB Operator 1.2.0-beta.2 Release Notes

Release date: April 29, 2021

TiDB Operator version: 1.2.0-beta.2

## Rolling update changes

- Upgrading TiDB Operator will cause the recreation of the TidbMonitor Pod due to [#3943](https://github.com/pingcap/tidb-operator/pull/3943)
- Upgrading TiDB Operator will cause the recreation of the DM-master Pod due to [#3914](https://github.com/pingcap/tidb-operator/pull/3914)

## New features

- TidbMonitor supports monitoring multiple TidbClusters with TLS enabled ([#3867](https://github.com/pingcap/tidb-operator/pull/3867), [@mikechengwei](https://github.com/mikechengwei))
- Support configuring `podSecurityContext` for all TiDB components ([#3909](https://github.com/pingcap/tidb-operator/pull/3909), [@liubog2008](https://github.com/liubog2008))
- Support configuring `topologySpreadConstraints` for all TiDB components ([#3937](https://github.com/pingcap/tidb-operator/pull/3937), [@liubog2008](https://github.com/liubog2008))
- Support deploying a DmCluster in a different namespace than a TidbCluster ([#3914](https://github.com/pingcap/tidb-operator/pull/3914), [@csuzhangxc](https://github.com/csuzhangxc))
- Support installing TiDB Operator with only namespace-scoped permissions ([#3896](https://github.com/pingcap/tidb-operator/pull/3896), [@csuzhangxc](https://github.com/csuzhangxc))

## Improvements

- Add the readiness probe for the TidbMonitor Pod ([#3943](https://github.com/pingcap/tidb-operator/pull/3943), [@mikechengwei](https://github.com/mikechengwei))
- Optimize TidbMonitor for DmCluster with TLS enabled ([#3942](https://github.com/pingcap/tidb-operator/pull/3942), [@mikechengwei](https://github.com/mikechengwei))
- TidbMonitor supports not generating Prometheus alert rules ([#3932](https://github.com/pingcap/tidb-operator/pull/3932), [@mikechengwei](https://github.com/mikechengwei))

## Bug fixes

- Fix the issue that TiDB instances are kept in TiDB Dashboard after being scaled in ([#3929](https://github.com/pingcap/tidb-operator/pull/3929), [@july2993](https://github.com/july2993))
- Fix the useless sync of TidbCluster CR caused by the update of `lastHeartbeatTime` in `status.tikv.stores` ([#3886](https://github.com/pingcap/tidb-operator/pull/3886), [@songjiansuper](https://github.com/songjiansuper))
