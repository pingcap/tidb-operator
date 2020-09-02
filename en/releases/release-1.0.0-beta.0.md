---
title: TiDB Operator 1.0 Beta.0 Release Notes
---

# TiDB Operator 1.0 Beta.0 Release Notes

Release date: November 26, 2018

TiDB Operator version: 1.0.0-beta.0

## Notable Changes

- Introduce basic chaos testing
- Improve unit test coverage ([#179](https://github.com/pingcap/tidb-operator/pull/179) [#181](https://github.com/pingcap/tidb-operator/pull/181) [#182](https://github.com/pingcap/tidb-operator/pull/182) [#184](https://github.com/pingcap/tidb-operator/pull/184) [#190](https://github.com/pingcap/tidb-operator/pull/190) [#192](https://github.com/pingcap/tidb-operator/pull/192) [#194](https://github.com/pingcap/tidb-operator/pull/194))
- Add default value for log-level of PD/TiKV/TiDB ([#185](https://github.com/pingcap/tidb-operator/pull/185))
- Fix PD connection timeout issue for DinD environment ([#186](https://github.com/pingcap/tidb-operator/pull/186))
- Fix monitor configuration ([#193](https://github.com/pingcap/tidb-operator/pull/193))
- Fix document Helm client version requirement ([#175](https://github.com/pingcap/tidb-operator/pull/175))
- Keep scheduler name consistent in chart ([#188](https://github.com/pingcap/tidb-operator/pull/188))
- Remove unnecessary warning message when volumeName is empty ([#177](https://github.com/pingcap/tidb-operator/pull/177))
- Migrate to Go 1.11 module ([#178](https://github.com/pingcap/tidb-operator/pull/178))
- Add user guide ([#187](https://github.com/pingcap/tidb-operator/pull/187))
