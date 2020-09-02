---
title: TiDB Operator 0.4 Release Notes
---

# TiDB Operator 0.4 Release Notes

Release date: November 9, 2018

TiDB Operator version: 0.4.0

## Notable Changes

- Extend Kubernetes built-in scheduler for TiDB data awareness pod scheduling ([#145](https://github.com/pingcap/tidb-operator/pull/145))
- Restore backup data from GCS bucket ([#160](https://github.com/pingcap/tidb-operator/pull/160))
- Set password for TiDB when a TiDB cluster is first deployed ([#171](https://github.com/pingcap/tidb-operator/pull/171))

## Minor Changes and Bug Fixes

- Update roadmap for the following two months ([#166](https://github.com/pingcap/tidb-operator/pull/166))
- Add more unit tests ([#169](https://github.com/pingcap/tidb-operator/pull/169))
- E2E test with multiple clusters ([#162](https://github.com/pingcap/tidb-operator/pull/162))
- E2E test for meta info synchronization ([#164](https://github.com/pingcap/tidb-operator/pull/164))
- Add TiDB failover limit ([#163](https://github.com/pingcap/tidb-operator/pull/163))
- Synchronize PV reclaim policy early to persist data ([#169](https://github.com/pingcap/tidb-operator/pull/169))
- Use helm release name as instance label ([#168](https://github.com/pingcap/tidb-operator/pull/168)) (breaking change)
- Fix local PV setup script ([#172](https://github.com/pingcap/tidb-operator/pull/172))
