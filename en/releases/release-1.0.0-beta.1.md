---
title: TiDB Operator 1.0 Beta.1 Release Notes
---

# TiDB Operator 1.0 Beta.1 Release Notes

Release date: December 27, 2018

TiDB Operator version: 1.0.0-beta.1

## Bug Fixes

* Fix pd_control bug: avoid relying on PD error response text ([#197](https://github.com/pingcap/tidb-operator/pull/197))
* Add orphan pod cleaner ([#201](https://github.com/pingcap/tidb-operator/pull/201))
* Fix scheduler configuration for Kubernetes 1.12 ([#200](https://github.com/pingcap/tidb-operator/pull/200))
* Fix Grafana configuration ([#206](https://github.com/pingcap/tidb-operator/pull/206))
* Fix pd failover bug: scale out directly when failover occurs ([#217](https://github.com/pingcap/tidb-operator/pull/217))
* Refactor PD failover ([#211](https://github.com/pingcap/tidb-operator/pull/211))
* Refactor tidb_cluster_control logic ([#215](https://github.com/pingcap/tidb-operator/pull/215))
* Fix upgrade logic: avoid updating pd/tikv/tidb simultaneously ([#234](https://github.com/pingcap/tidb-operator/pull/234))
* Fix PD control logic: get member/store before delete member/store and fix member id parse error ([#245](https://github.com/pingcap/tidb-operator/pull/245)) 
* Fix documents errors ([#213](https://github.com/pingcap/tidb-operator/pull/213)) 
* Fix backup and restore script bug ([#251](https://github.com/pingcap/tidb-operator/pull/251) [#254](https://github.com/pingcap/tidb-operator/pull/254) [#255](https://github.com/pingcap/tidb-operator/pull/255))
* Fix GKE multiple availability zones deployment PD disk scheduling bug ([#248](https://github.com/pingcap/tidb-operator/pull/248))

## Minor Improvements

* Add Kubernetes 1.12 local DinD scripts ([#195](https://github.com/pingcap/tidb-operator/pull/195)) 
* Bump default TiDB to v2.1.0 ([#212](https://github.com/pingcap/tidb-operator/pull/212))
* Release tidb-operator/tidb-cluster charts ([#216](https://github.com/pingcap/tidb-operator/pull/216)) 
* Add connection timeout for TiDB password setter job ([#219](https://github.com/pingcap/tidb-operator/pull/219))
* Separate ad-hoc backup and restore to another chart ([#227](https://github.com/pingcap/tidb-operator/pull/227))
* Add compiler version info to tidb-operator binary ([#237](https://github.com/pingcap/tidb-operator/pull/237)) 
* Allow specifying TiDB service LoadBalancer IP ([#246](https://github.com/pingcap/tidb-operator/pull/246))
* Expose TiKV cpu/memory related configuration to values.yaml ([#252](https://github.com/pingcap/tidb-operator/pull/252))
