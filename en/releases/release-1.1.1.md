---
title: TiDB Operator 1.1.1 Release Notes
---

# TiDB Operator 1.1.1 Release Notes

Release date: June 19, 2020

TiDB Operator version: 1.1.1

## Notable changes

- Add the `additionalContainers` and `additionalVolumes` fields so that TiDB Operator can support adding sidecars to `TiDB`, `TiKV`, `PD`, etc. ([#2229](https://github.com/pingcap/tidb-operator/pull/2229), [@yeya24](https://github.com/yeya24))
- Add cross check to ensure TiKV is not scaled or upgraded at the same time ([#2705](https://github.com/pingcap/tidb-operator/pull/2705), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix the bug that TidbMonitor will scrape multi TidbCluster with the same name in different namespaces when then namespace in `ClusterRef` is not set ([#2746](https://github.com/pingcap/tidb-operator/pull/2746), [@Yisaer](https://github.com/Yisaer))
- Update TiDB Operator examples to deploy TiDB Cluster 4.0.0 images ([#2600](https://github.com/pingcap/tidb-operator/pull/2600), [@kolbe](https://github.com/kolbe))
- Add the `alertMangerAlertVersion` option to TidbMonitor ([#2744](https://github.com/pingcap/tidb-operator/pull/2744), [@weekface](https://github.com/weekface))
- Fix alert rules lost after rolling upgrade ([#2715](https://github.com/pingcap/tidb-operator/pull/2715), [@weekface](https://github.com/weekface))
- Fix an issue that pods may be stuck in pending for a long time in scale-out after a scale-in ([#2709](https://github.com/pingcap/tidb-operator/pull/2709), [@cofyc](https://github.com/cofyc))
- Add `EnableDashboardInternalProxy` in `PDSpec` to let user directly visit PD Dashboard ([#2713](https://github.com/pingcap/tidb-operator/pull/2713), [@Yisaer](https://github.com/Yisaer))
- Fix the PV syncing error when `TidbMonitor` and `TidbCluster` have different values in `reclaimPolicy` ([#2707](https://github.com/pingcap/tidb-operator/pull/2707), [@Yisaer](https://github.com/Yisaer))
- Update Configuration to v4.0.1 ([#2702](https://github.com/pingcap/tidb-operator/pull/2702), [@Yisaer](https://github.com/Yisaer))
- Change tidb-discovery strategy type to `Recreate` to fix the bug that more than one discovery pod may exist ([#2701](https://github.com/pingcap/tidb-operator/pull/2701), [@weekface](https://github.com/weekface))
- Expose the `Dashboard` service with `HTTP` endpoint whether `tlsCluster` is enabled ([#2684](https://github.com/pingcap/tidb-operator/pull/2684), [@Yisaer](https://github.com/Yisaer))
- Add the `.tikv.dataSubDir` field to specify subdirectory within the data volume to store TiKV data ([#2682](https://github.com/pingcap/tidb-operator/pull/2682), [@cofyc](https://github.com/cofyc))
- Add the `imagePullSecrets` attribute to all components ([#2679](https://github.com/pingcap/tidb-operator/pull/2679), [@weekface](https://github.com/weekface))
- Enable StatefulSet and Pod validation webhook to work at the same time ([#2664](https://github.com/pingcap/tidb-operator/pull/2664), [@Yisaer](https://github.com/Yisaer))
- Emit an event if it fails to sync labels to TiKV stores ([#2587](https://github.com/pingcap/tidb-operator/pull/2587), [@PengJi](https://github.com/PengJi))
- Make `datasource` information hidden in log for `Backup` and `Restore` jobs ([#2652](https://github.com/pingcap/tidb-operator/pull/2652), [@Yisaer](https://github.com/Yisaer))
- Support the `DynamicConfiguration` switch in TidbCluster Spec ([#2539](https://github.com/pingcap/tidb-operator/pull/2539), [@Yisaer](https://github.com/Yisaer))
- Support `LoadBalancerSourceRanges` in the `ServiceSpec` for the `TidbCluster` and `TidbMonitor` ([#2610](https://github.com/pingcap/tidb-operator/pull/2610), [@shonge](https://github.com/shonge))
- Support `Dashboard` metrics ability for `TidbCluster` when `TidbMonitor` deployed ([#2483](https://github.com/pingcap/tidb-operator/pull/2483), [@Yisaer](https://github.com/Yisaer))
- Bump the DM version to v2.0.0-beta.1 ([#2615](https://github.com/pingcap/tidb-operator/pull/2615), [@tennix](https://github.com/tennix))
- support setting discovery resources ([#2434](https://github.com/pingcap/tidb-operator/pull/2434), [@shonge](https://github.com/shonge))
- Support the Denoising for the `TidbCluster` Auto-scaling ([#2307](https://github.com/pingcap/tidb-operator/pull/2307), [@vincent178](https://github.com/vincent178))
- Support scraping `Pump` and `Drainer` metrics in TidbMonitor ([#2750](https://github.com/pingcap/tidb-operator/pull/2750), [@Yisaer](https://github.com/Yisaer))
