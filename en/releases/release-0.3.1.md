---
title: TiDB Operator 0.3.1 Release Notes
---

# TiDB Operator 0.3.1 Release Notes

Release date: October 31, 2018

TiDB Operator version: 0.3.1

## Minor Changes

- Paramertize the serviceAccount ([#116](https://github.com/pingcap/tidb-operator/pull/116) [#111](https://github.com/pingcap/tidb-operator/pull/111)) 
- Bump TiDB to v2.0.7 & allow user specified config files ([#121](https://github.com/pingcap/tidb-operator/pull/))
- Remove binding mode for GKE pd-ssd storageclass ([#130](https://github.com/pingcap/tidb-operator/pull/130))
- Modified placement of tidb_version ([#125](https://github.com/pingcap/tidb-operator/pull/125)) 
- Update google-kubernetes-tutorial.md ([#105](https://github.com/pingcap/tidb-operator/pull/105)) 
- Remove redundant creation statement of namespace tidb-operator-e2e ([#132](https://github.com/pingcap/tidb-operator/pull/132)) 
- Update the label name of app in local dind documentation ([#136](https://github.com/pingcap/tidb-operator/pull/136)) 
- Remove noisy events ([#131](https://github.com/pingcap/tidb-operator/pull/131)) 
- Marketplace ([#123](https://github.com/pingcap/tidb-operator/pull/123) [#135](https://github.com/pingcap/tidb-operator/pull/135)) 
- Change monitor/backup/binlog pvc labels ([#143](https://github.com/pingcap/tidb-operator/pull/143)) 
- TiDB readiness probes ([#147](https://github.com/pingcap/tidb-operator/pull/147)) 
- Add doc on how to provision kubernetes on AWS ([#71](https://github.com/pingcap/tidb-operator/pull/71)) 
- Add imagePullPolicy support ([#152](https://github.com/pingcap/tidb-operator/pull/152)) 
- Separation startup scripts and application config from yaml files ([#149](https://github.com/pingcap/tidb-operator/pull/149)) 
- Update marketplace for our open source offering ([#151](https://github.com/pingcap/tidb-operator/pull/151)) 
- Add validation to crd ([#153](https://github.com/pingcap/tidb-operator/pull/153))
- Marketplace: use the Release.Name ([#157](https://github.com/pingcap/tidb-operator/pull/157)) 

## Bug Fixes

- Fix parallel upgrade bug ([#118](https://github.com/pingcap/tidb-operator/pull/118)) 
- Fix wrong parameter AGRS to ARGS ([#114](https://github.com/pingcap/tidb-operator/pull/114)) 
- Can't recover after a upgrade failed ([#120](https://github.com/pingcap/tidb-operator/pull/120)) 
- Scale in when store id match ([#124](https://github.com/pingcap/tidb-operator/pull/124)) 
- PD can't scale out if not all members are ready ([#142](https://github.com/pingcap/tidb-operator/pull/142)) 
- podLister and pvcLister usages are wrong ([#158](https://github.com/pingcap/tidb-operator/pull/158)) 
