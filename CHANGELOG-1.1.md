# TiDB Operator v1.1.1 Release Notes

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


# TiDB Operator v1.1.0 Release Notes

This is the GA release of TiDB Operator 1.1, which focuses on the usability, extensibility and security.

See our official [documentation site](https://pingcap.com/docs/tidb-in-kubernetes/stable/) for new features, guides, and instructions in production, etc.

## Upgrade from v1.0.x

For v1.0.x users, refer to [Upgrade TiDB Operator](https://pingcap.com/docs/tidb-in-kubernetes/stable/upgrade-tidb-operator/) to upgrade TiDB Operator in your cluster. Note that you should read the release notes (especially breaking changes and action required items) before the upgrade.

## Breaking changes since v1.0.0

- Change TiDB pod `readiness` probe from `HTTPGet` to `TCPSocket` 4000 port. This will trigger rolling-upgrade for the `tidb-server` component. You can set `spec.paused` to `true` before upgrading tidb-operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your TiDB server ([#2139](https://github.com/pingcap/tidb-operator/pull/2139), [@weekface](https://github.com/weekface))
- `--advertise-address` is configured for `tidb-server`, which would trigger rolling-upgrade for the TiDB server. You can set `spec.paused` to `true` before upgrading TiDB Operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your TiDB server ([#2076](https://github.com/pingcap/tidb-operator/pull/2076), [@cofyc](https://github.com/cofyc))
- `--default-storage-class-name` and `--default-backup-storage-class-name` flags are abandoned, and the storage class defaults to Kubernetes default storage class right now. If you have set default storage class different than Kubernetes default storage class, set them explicitly in your TiDB cluster Helm or YAML files. ([#1581](https://github.com/pingcap/tidb-operator/pull/1581), [@cofyc](https://github.com/cofyc))
- Add the `timezone` support for [all charts](https://github.com/pingcap/tidb-operator/tree/master/charts) ([#1122](https://github.com/pingcap/tidb-operator/pull/1122), [@weekface](https://github.com/weekface)).

  For the `tidb-cluster` chart, we already have the `timezone` option (`UTC` by default). If the user does not change it to a different value (for example, `Asia/Shanghai`), none of the Pods will be recreated.
  If the user changes it to another value (for example, `Aisa/Shanghai`), all the related Pods (add a `TZ` env) will be recreated, namely rolling updated.

  The related Pods include `pump`, `drainer`, `dicovery`, `monitor`, `scheduled backup`, `tidb-initializer`, and `tikv-importer`.

  All images' time zone maintained by TiDB Operator is `UTC`. If you use your own images, you need to make sure that the time zone inside your images is `UTC`.

## Other Notable changes

- Fix `TidbCluster` upgrade bug when `PodWebhook` and `Advancend StatefulSet` are both enabled ([#2507](https://github.com/pingcap/tidb-operator/pull/2507), [@Yisaer](https://github.com/Yisaer))
- Support preemption in `tidb-scheduler` ([#2510](https://github.com/pingcap/tidb-operator/pull/2510), [@cofyc](https://github.com/cofyc))
- Update BR to v4.0.0-rc.2 to include the `auto_random` fix ([#2508](https://github.com/pingcap/tidb-operator/pull/2508), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Supports advanced statefulset for TiFlash ([#2469](https://github.com/pingcap/tidb-operator/pull/2469), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Sync Pump before TiDB ([#2515](https://github.com/pingcap/tidb-operator/pull/2515), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Improve performance by removing `TidbControl` lock ([#2489](https://github.com/pingcap/tidb-operator/pull/2489), [@weekface](https://github.com/weekface))
- Support TiCDC in `TidbCluster` ([#2362](https://github.com/pingcap/tidb-operator/pull/2362), [@weekface](https://github.com/weekface))
- Update TiDB/TiKV/PD configuration to 4.0.0 GA version ([#2571](https://github.com/pingcap/tidb-operator/pull/2571), [@Yisaer](https://github.com/Yisaer))
- TiDB Operator will not do failover for PD pods which are not desired ([#2570](https://github.com/pingcap/tidb-operator/pull/2570), [@Yisaer](https://github.com/Yisaer))

## Previous releases

- [v1.1.0-rc.4](#tidb-operator-v110-rc4-release-notes)
- [v1.1.0-rc.3](#tidb-operator-v110-rc3-release-notes)
- [v1.1.0-rc.2](#tidb-operator-v110-rc2-release-notes)
- [v1.1.0-rc.1](#tidb-operator-v110-rc1-release-notes)
- [v1.1.0-beta.2](#tidb-operator-v110-beta2-release-notes)
- [v1.1.0-beta.1](#tidb-operator-v110-beta1-release-notes)

# TiDB Operator v1.1.0-rc.4 Release Notes

This is the fourth release candidate of `v1.1.0`, which focuses on the usability, extensibility and security of TiDB Operator. While we encourage usage in non-critical environments, it is **NOT** recommended to use this version in critical environments.

## Action Required

- Separate TiDB client certificates can be used for each component. Users should migrate the old TLS configs of Backup and Restore to the new configs. Refer to [#2403](https://github.com/pingcap/tidb-operator/pull/2403) for more details ([#2403](https://github.com/pingcap/tidb-operator/pull/2403), [@weekface](https://github.com/weekface))

## Other Notable Changes

- Fix the bug that the service annotations would be exposed in `TidbCluster` specification ([#2471](https://github.com/pingcap/tidb-operator/pull/2471), [@Yisaer](https://github.com/Yisaer))
- Fix a bug when reconciling TiDB service while the `healthCheckNodePort` is already generated by Kubernetes ([#2438](https://github.com/pingcap/tidb-operator/pull/2438), [@aylei](https://github.com/aylei))
- Support `TidbMonitorRef` in `TidbCluster` Status ([#2424](https://github.com/pingcap/tidb-operator/pull/2424), [@Yisaer](https://github.com/Yisaer))
- Support setting the backup path prefix for remote storage ([#2435](https://github.com/pingcap/tidb-operator/pull/2435), [@onlymellb](https://github.com/onlymellb))
- Support customizing `mydumper` options in Backup CR ([#2407](https://github.com/pingcap/tidb-operator/pull/2407), [@onlymellb](https://github.com/onlymellb))
- Support TiCDC in TidbCluster CR. ([#2338](https://github.com/pingcap/tidb-operator/pull/2338), [@weekface](https://github.com/weekface))
- Update BR to `v3.1.1` in the `tidb-backup-manager` image ([#2425](https://github.com/pingcap/tidb-operator/pull/2425), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support creating node pools for TiFlash and CDC on ACK ([#2420](https://github.com/pingcap/tidb-operator/pull/2420), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support creating node pools for TiFlash and CDC on EKS ([#2413](https://github.com/pingcap/tidb-operator/pull/2413), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Expose `PVReclaimPolicy` for `TidbMonitor` when storage is enabled ([#2379](https://github.com/pingcap/tidb-operator/pull/2379), [@Yisaer](https://github.com/Yisaer))
- Support arbitrary topology-based HA in tidb-scheduler (e.g. node zones) ([#2366](https://github.com/pingcap/tidb-operator/pull/2366), [@PengJi](https://github.com/PengJi))
- Skip setting the TLS for PD dashboard when the TiDB version is earlier than 4.0.0 ([#2389](https://github.com/pingcap/tidb-operator/pull/2389), [@weekface](https://github.com/weekface))
- Support backup and restore with GCS using BR ([#2267](https://github.com/pingcap/tidb-operator/pull/2267), [@shuijing198799](https://github.com/shuijing198799))
- Update `TiDBConfig` and `TiKVConfig` to support the `4.0.0-rc` version ([#2322](https://github.com/pingcap/tidb-operator/pull/2322), [@Yisaer](https://github.com/Yisaer))
- Fix the bug when `TidbCluster` service type is `NodePort`, the value of `NodePort` would change frequently ([#2284](https://github.com/pingcap/tidb-operator/pull/2284), [@Yisaer](https://github.com/Yisaer))
- Add external strategy ability for `TidbClusterAutoScaler` ([#2279](https://github.com/pingcap/tidb-operator/pull/2279), [@Yisaer](https://github.com/Yisaer))
- PVC will not be deleted when `TidbMonitor` gets deleted ([#2374](https://github.com/pingcap/tidb-operator/pull/2374), [@Yisaer](https://github.com/Yisaer))
- Support scaling for TiFlash ([#2237](https://github.com/pingcap/tidb-operator/pull/2237), [@DanielZhangQD](https://github.com/DanielZhangQD))


# TiDB Operator v1.1.0-rc.3 Release Notes

This is the third release candidate of `v1.1.0`, which focuses on the usability, extensibility and security of TiDB Operator. While we encourage usage in non-critical environments, it is **NOT** recommended to use this version in critical environments.

## Notable Changes

- Skip auto-failover when pods are not scheduled and perform recovery operation no matter what state failover pods are in ([#2263](https://github.com/pingcap/tidb-operator/pull/2263), [@cofyc](https://github.com/cofyc))
- Support `TiFlash` metrics in `TidbMonitor` ([#2341](https://github.com/pingcap/tidb-operator/pull/2341), [@Yisaer](https://github.com/Yisaer))
- Do not print `rclone` config in the Pod logs ([#2343](https://github.com/pingcap/tidb-operator/pull/2343), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Using `Patch` in `periodicity` controller to avoid updating `StatefulSet` to the wrong state ([#2332](https://github.com/pingcap/tidb-operator/pull/2332), [@Yisaer](https://github.com/Yisaer))
- Set `enable-placement-rules` to `true` for PD if TiFlash is enabled in the cluster ([#2328](https://github.com/pingcap/tidb-operator/pull/2328), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support `rclone` options in the Backup and Restore CR ([#2318](https://github.com/pingcap/tidb-operator/pull/2318), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix the issue that statefulsets are updated during each sync even if no changes are made to the config ([#2308](https://github.com/pingcap/tidb-operator/pull/2308), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support configuring `Ingress` in `TidbMonitor` ([#2314](https://github.com/pingcap/tidb-operator/pull/2314), [@Yisaer](https://github.com/Yisaer))
- Fix a bug that auto-created failover pods can't be deleted when they are in the failed state ([#2300](https://github.com/pingcap/tidb-operator/pull/2300), [@cofyc](https://github.com/cofyc))
- Add useful `Event` in `TidbCluster` during upgrading and scaling when `admissionWebhook.validation.pods` in operator configuration is enabled ([#2305](https://github.com/pingcap/tidb-operator/pull/2305), [@Yisaer](https://github.com/Yisaer))
- Fix the issue that services are updated during each sync even if no changes are made to the service configuration ([#2299](https://github.com/pingcap/tidb-operator/pull/2299), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix a bug that would cause panic in statefulset webhook when the update strategy of `StatefulSet` is not `RollingUpdate` ([#2291](https://github.com/pingcap/tidb-operator/pull/2291), [@Yisaer](https://github.com/Yisaer))
- Fix a panic in syncing `TidbClusterAutoScaler` status when the target `TidbCluster` does not exist ([#2289](https://github.com/pingcap/tidb-operator/pull/2289), [@Yisaer](https://github.com/Yisaer))
- Fix the `pdapi` cache issue while the cluster TLS is enabled ([#2275](https://github.com/pingcap/tidb-operator/pull/2275), [@weekface](https://github.com/weekface))
- Fix the config error in restore ([#2250](https://github.com/pingcap/tidb-operator/pull/2250), [@Yisaer](https://github.com/Yisaer))
- Support failover for TiFlash ([#2249](https://github.com/pingcap/tidb-operator/pull/2249), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Update the default `eks` version in terraform scripts to 1.15 ([#2238](https://github.com/pingcap/tidb-operator/pull/2238), [@Yisaer](https://github.com/Yisaer))
- Support upgrading for TiFlash ([#2246](https://github.com/pingcap/tidb-operator/pull/2246), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add `stderr` logs from BR to the backup-manager logs ([#2213](https://github.com/pingcap/tidb-operator/pull/2213), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add field `TiKVEncryptionConfig` in `TiKVConfig`, which defines how to encrypt data key and raw data in TiKV, and how to back up and restore the master key. See the description for details in `tikv_config.go` ([#2151](https://github.com/pingcap/tidb-operator/pull/2151), [@shuijing198799](https://github.com/shuijing198799))


# TiDB Operator v1.1.0-rc.2 Release Notes

This is the second release candidate of `v1.1.0`, which focuses on the usability, extensibility and security of TiDB Operator. While we encourage usage in non-critical environments, it is **NOT** recommended to use this version in critical environments.

## Action Required

- Change TiDB pod `readiness` probe from `HTTPGet` to `TCPSocket` 4000 port. This will trigger rolling-upgrade for the `tidb-server` component. You can set `spec.paused` to `true` before upgrading tidb-operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your tidb server ([#2139](https://github.com/pingcap/tidb-operator/pull/2139), [@weekface](https://github.com/weekface))

## Notable Changes

- Add `status` field for `TidbAutoScaler` CR ([#2182](https://github.com/pingcap/tidb-operator/pull/2182), [@Yisaer](https://github.com/Yisaer))
- Add `spec.pd.maxFailoverCount` field to limit max failover replicas for PD ([#2184](https://github.com/pingcap/tidb-operator/pull/2184), [@cofyc](https://github.com/cofyc))
- Emit more events for `TidbCluster` and `TidbClusterAutoScaler` to help users know TiDB running status ([#2150](https://github.com/pingcap/tidb-operator/pull/2150), [@Yisaer](https://github.com/Yisaer))
- Add the `AGE` column to show creation timestamp for all CRDs ([#2168](https://github.com/pingcap/tidb-operator/pull/2168), [@cofyc](https://github.com/cofyc))
- Add a switch to skip PD Dashboard TLS configuration ([#2143](https://github.com/pingcap/tidb-operator/pull/2143), [@weekface](https://github.com/weekface))
- Support deploying TiFlash with TidbCluster CR ([#2157](https://github.com/pingcap/tidb-operator/pull/2157), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add TLS support for TiKV metrics API ([#2137](https://github.com/pingcap/tidb-operator/pull/2137), [@weekface](https://github.com/weekface))
- Set PD DashboardConfig when TLS between the MySQL client and TiDB server is enabled ([#2085](https://github.com/pingcap/tidb-operator/pull/2085), [@weekface](https://github.com/weekface))
- Remove unnecessary informer caches to reduce the memory footprint of tidb-controller-manager ([#1504](https://github.com/pingcap/tidb-operator/pull/1504), [@aylei](https://github.com/aylei))
- Fix the failure that Helm cannot load the kubeconfig file when deleting the tidb-operator release during `terraform destroy` ([#2148](https://github.com/pingcap/tidb-operator/pull/2148), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support configuring the Webhook TLS setting by loading a secret ([#2135](https://github.com/pingcap/tidb-operator/pull/2135), [@Yisaer](https://github.com/Yisaer))
- Support TiFlash in TidbCluster CR ([#2122](https://github.com/pingcap/tidb-operator/pull/2122), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix the error that alertmanager couldn't be set in `TidbMonitor` ([#2108](https://github.com/pingcap/tidb-operator/pull/2108), [@Yisaer](https://github.com/Yisaer))


# TiDB Operator v1.1.0-rc.1 Release Notes

This is a release candidate of `v1.1.0`, which focuses on the usability, extensibility and security of TiDB Operator. While we encourage usage in non-critical environments, it is **NOT** recommended to use this version in critical environments.

## Action Required

- `--advertise-address` will be configured for `tidb-server`, which would trigger rolling-upgrade for the `tidb-server` component. You can set `spec.paused` to `true` before upgrading tidb-operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your tidb server ([#2076](https://github.com/pingcap/tidb-operator/pull/2076), [@cofyc](https://github.com/cofyc))
- Add the `tlsClient.tlsSecret` field in the backup and restore spec, which supports specifying a secret name that includes the cert ([#2003](https://github.com/pingcap/tidb-operator/pull/2003), [@shuijing198799](https://github.com/shuijing198799))
- Remove `spec.br.pd`, `spec.br.ca`, `spec.br.cert`, `spec.br.key` and add `spec.br.cluster`, `spec.br.clusterNamespace` for the `Backup`, `Restore` and `BackupSchedule` custom resources, which makes the BR configuration more reasonable ([#1836](https://github.com/pingcap/tidb-operator/pull/1836), [@shuijing198799](https://github.com/shuijing198799))


## Other Notable Changes

- Use `tidb-lightning` in `Restore` instead of `loader` ([#2068](https://github.com/pingcap/tidb-operator/pull/2068), [@Yisaer](https://github.com/Yisaer))
- Add `cert-allowed-cn` support to TiDB components ([#2061](https://github.com/pingcap/tidb-operator/pull/2061), [@weekface](https://github.com/weekface))
- Fix the PD `location-labels` configuration ([#1941](https://github.com/pingcap/tidb-operator/pull/1941), [@aylei](https://github.com/aylei))
- Able to pause and unpause tidb cluster deployment via `spec.paused` ([#2013](https://github.com/pingcap/tidb-operator/pull/2013), [@cofyc](https://github.com/cofyc))
- Default the `max-backups` for TiDB server configuration to `3` if the TiDB cluster is deployed by CR ([#2045](https://github.com/pingcap/tidb-operator/pull/2045), [@Yisaer](https://github.com/Yisaer))
- Able to configure custom environments for components ([#2052](https://github.com/pingcap/tidb-operator/pull/2052), [@cofyc](https://github.com/cofyc))
- Fix the error that `kubectl get tc` cannot show correct images ([#2031](https://github.com/pingcap/tidb-operator/pull/2031), [@Yisaer](https://github.com/Yisaer))
- 1. Default the `spec.tikv.maxFailoverCount` and `spec.tidb.maxFailoverCount` to `3` when they are not defined
  2. Disable auto-failover when `maxFailoverCount` is set to `0` ([#2015](https://github.com/pingcap/tidb-operator/pull/2015), [@Yisaer](https://github.com/Yisaer))
- Support deploying TiDB clusters with TidbCluster and TidbMonitor CRs via Terraform on ACK ([#2012](https://github.com/pingcap/tidb-operator/pull/2012), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Update PDConfig for TidbCluster to PD v3.1.0 ([#1928](https://github.com/pingcap/tidb-operator/pull/1928), [@Yisaer](https://github.com/Yisaer))
- Support deploying TiDB clusters with TidbCluster and TidbMonitor CRs via Terraform on AWS ([#2004](https://github.com/pingcap/tidb-operator/pull/2004), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Update TidbConfig for TidbCluster to TiDB v3.1.0 ([#1906](https://github.com/pingcap/tidb-operator/pull/1906), [@Yisaer](https://github.com/Yisaer))
- Allow users to define resources for initContainers in TiDB initializer job ([#1938](https://github.com/pingcap/tidb-operator/pull/1938), [@tfulcrand](https://github.com/tfulcrand))
- Add TLS support for Pump and Drainer ([#1979](https://github.com/pingcap/tidb-operator/pull/1979), [@weekface](https://github.com/weekface))
- Add documents and examples for auto-scaler and initializer ([#1772](https://github.com/pingcap/tidb-operator/pull/1772), [@Yisaer](https://github.com/Yisaer))
- 1. Add check to guarantee the NodePort won't be changed if the serviceType of TidbMonitor is NodePort
  2. Add EnvVar sort to avoid the monitor rendering different results from the same TidbMonitor spec
  3. Fix the problem that the TidbMonitor LoadBalancer IP is not used ([#1962](https://github.com/pingcap/tidb-operator/pull/1962), [@Yisaer](https://github.com/Yisaer))
- Make tidb-initializer support TLS ([#1931](https://github.com/pingcap/tidb-operator/pull/1931), [@weekface](https://github.com/weekface))
- 1. Fix the problem that Advanced StatefulSet cannot work with webhook
  2. Change the Reaction for the Down State TiKV pod during deleting request in webhook from admit to reject ([#1963](https://github.com/pingcap/tidb-operator/pull/1963), [@Yisaer](https://github.com/Yisaer))
- Fix the drainer installation error when `drainerName` is set ([#1961](https://github.com/pingcap/tidb-operator/pull/1961), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix some TiKV configuration keys in toml ([#1887](https://github.com/pingcap/tidb-operator/pull/1887), [@aylei](https://github.com/aylei))
- Support using a remote directory as data source for tidb-lightning ([#1629](https://github.com/pingcap/tidb-operator/pull/1629), [@aylei](https://github.com/aylei))
- Add the API document and a script that generates documentation ([#1945](https://github.com/pingcap/tidb-operator/pull/1945), [@Yisaer](https://github.com/Yisaer))
- Add the tikv-importer chart ([#1910](https://github.com/pingcap/tidb-operator/pull/1910), [@shonge](https://github.com/shonge))
- Fix the Prometheus scrape config issue while TLS is enabled ([#1919](https://github.com/pingcap/tidb-operator/pull/1919), [@weekface](https://github.com/weekface))
- Enable TLS between TiDB components ([#1870](https://github.com/pingcap/tidb-operator/pull/1870), [@weekface](https://github.com/weekface))
- Fix the timeout error when `.Values.admission.validation.pods` is `true` during the TiKV upgrade ([#1875](https://github.com/pingcap/tidb-operator/pull/1875), [@Yisaer](https://github.com/Yisaer))
- Enable TLS for MySQL clients ([#1878](https://github.com/pingcap/tidb-operator/pull/1878), [@weekface](https://github.com/weekface))
- Fix the bug which would cause broken TiDB image property ([#1860](https://github.com/pingcap/tidb-operator/pull/1860), [@Yisaer](https://github.com/Yisaer))
- TidbMonitor would use its namespace for the targetRef if it is not defined ([#1834](https://github.com/pingcap/tidb-operator/pull/1834), [@Yisaer](https://github.com/Yisaer))
- Support starting tidb-server with `--advertise-address` parameter ([#1859](https://github.com/pingcap/tidb-operator/pull/1859), [@LinuxGit](https://github.com/LinuxGit))
- Backup/Restore: support configuring TiKV GC life time ([#1835](https://github.com/pingcap/tidb-operator/pull/1835), [@LinuxGit](https://github.com/LinuxGit))
- Support no secret for S3/Ceph when the OIDC authentication is used ([#1817](https://github.com/pingcap/tidb-operator/pull/1817), [@tirsen](https://github.com/tirsen))
- 1. Change the setting from the previous `admission.hookEnabled.pods` to the `admission.validation.pods`
  2. Change the setting from the previous `admission.hookEnabled.statefulSets` to the `admission.validation.statefulSets`
  3. Change the setting from the previous `admission.hookEnabled.validating` to the `admission.validation.pingcapResources`
  4. Change the setting from the previous `admission.hookEnabled.defaulting` to the `admission.mutation.pingcapResources`
  5. Change the setting from the previous `admission.failurePolicy.defaulting` to the `admission.failurePolicy.mutation`
  6. Change the setting from the previous `admission.failurePolicy.*` to the `admission.failurePolicy.validation` ([#1832](https://github.com/pingcap/tidb-operator/pull/1832), [@Yisaer](https://github.com/Yisaer))
- Enable TidbCluster defaulting mutation by default which is recommended when admission webhook is used ([#1816](https://github.com/pingcap/tidb-operator/pull/1816), [@Yisaer](https://github.com/Yisaer))
- Fix a bug that TiKV fails to start while creating the cluster using CR with cluster TLS enabled ([#1808](https://github.com/pingcap/tidb-operator/pull/1808), [@weekface](https://github.com/weekface))
- Support using prefix in remote storage during backup/restore ([#1790](https://github.com/pingcap/tidb-operator/pull/1790), [@DanielZhangQD](https://github.com/DanielZhangQD))


# TiDB Operator v1.1.0-beta.2 Release Notes

This is a pre-release of `v1.1.0`, which focuses on the usability, extensibility and security of TiDB Operator. While we encourage usage in non-critical environments, it is **NOT** recommended to use this version in critical environments.

## Changes since v1.1.0-beta.1

## Action Required

- `--default-storage-class-name` and `--default-backup-storage-class-name `are abandoned, and the storage class defaults to Kubernetes default storage class right now. If you have set default storage class different than Kubernetes default storage class, please set them explicitly in your TiDB cluster helm or YAML files. ([#1581](https://github.com/pingcap/tidb-operator/pull/1581), [@cofyc](https://github.com/cofyc))


## Other Notable Changes

- Allow users to configure affinity and tolerations for `Backup` and `Restore`. ([#1737](https://github.com/pingcap/tidb-operator/pull/1737), [@Smana](https://github.com/Smana))
- Allow AdvancedStatefulSet and Admission Webhook to work together. ([#1640](https://github.com/pingcap/tidb-operator/pull/1640), [@Yisaer](https://github.com/Yisaer))
- Add a basic deployment example of managing TiDB cluster with custom resources only. ([#1573](https://github.com/pingcap/tidb-operator/pull/1573), [@aylei](https://github.com/aylei))
- Support TidbCluster Auto-scaling feature based on CPU average utilization load. ([#1731](https://github.com/pingcap/tidb-operator/pull/1731), [@Yisaer](https://github.com/Yisaer))
- Support user-defined TiDB server/client certificate ([#1714](https://github.com/pingcap/tidb-operator/pull/1714), [@weekface](https://github.com/weekface))
- Add an option for tidb-backup chart to allow reusing existing PVC or not for restore ([#1708](https://github.com/pingcap/tidb-operator/pull/1708), [@mightyguava](https://github.com/mightyguava))
- Add `resources`, `imagePullPolicy` and `nodeSelector` field for tidb-backup chart ([#1705](https://github.com/pingcap/tidb-operator/pull/1705), [@mightyguava](https://github.com/mightyguava))
- Add more SANs (Subject Alternative Name) to TiDB server certificate ([#1702](https://github.com/pingcap/tidb-operator/pull/1702), [@weekface](https://github.com/weekface))
- Support automatically migrating existing Kubernetes StatefulSets to Advanced StatefulSets when AdvancedStatfulSet feature is enabled ([#1580](https://github.com/pingcap/tidb-operator/pull/1580), [@cofyc](https://github.com/cofyc))
- Fix the bug in admission webhook which causes PD pod deleting error and allow the deleting pod to request for PD and TiKV when PVC is not found. ([#1568](https://github.com/pingcap/tidb-operator/pull/1568), [@Yisaer](https://github.com/Yisaer))
- Limit the restart rate for PD and TiKV - only one instance would be restarted each time ([#1532](https://github.com/pingcap/tidb-operator/pull/1532), [@Yisaer](https://github.com/Yisaer))
- Add default ClusterRef namespace for TidbMonitor as the same as it is deployed and fix the bug that TidbMonitor's Pod can't be created when Spec.PrometheusSpec.logLevel is missing. ([#1500](https://github.com/pingcap/tidb-operator/pull/1500), [@Yisaer](https://github.com/Yisaer))
- Refine logs for `TidbMonitor` and `TidbInitializer` controller ([#1493](https://github.com/pingcap/tidb-operator/pull/1493), [@aylei](https://github.com/aylei))
- Avoid unnecessary updates to `Service` and `Deployment` of discovery ([#1499](https://github.com/pingcap/tidb-operator/pull/1499), [@aylei](https://github.com/aylei))
- Remove some update events that are not very useful ([#1486](https://github.com/pingcap/tidb-operator/pull/1486), [@weekface](https://github.com/weekface))


# TiDB Operator v1.1.0-beta.1 Release Notes

This is a pre-release of `v1.1.0`, which focuses on the usability, extensibility and security of TiDB Operator. While we encourage usage in non-critical environments, it is **NOT** recommended to use this version in critical environments.

## Changes since v1.0.0

### Action Required

- ACTION REQUIRED: Add the `timezone` support for [all charts](https://github.com/pingcap/tidb-operator/tree/master/charts) ([#1122](https://github.com/pingcap/tidb-operator/pull/1122), [@weekface](https://github.com/weekface)).

  For the `tidb-cluster` chart, we already have the `timezone` option (`UTC` by default). If the user does not change it to a different value (for example: `Aisa/Shanghai`), all Pods will not be recreated.
  If the user changes it to another value (for example: `Aisa/Shanghai`), all the related Pods (add a `TZ` env) will be recreated (rolling update).

  Regarding other charts, we don't have a `timezone` option in their `values.yaml`. We add the `timezone` option in this PR. No matter whether the user uses the old `values.yaml` or the new `values.yaml`, all the related Pods (add a `TZ` env) will not be recreated (rolling update).

  The related Pods include `pump`, `drainer`, `dicovery`, `monitor`, `scheduled backup`, `tidb-initializer`, and `tikv-importer`.

  All images' time zone maintained by `tidb-operator` is `UTC`. If you use your own images, you need to make sure that the time zone inside your images is `UTC`.

### Other Notable Changes

- Support backup to S3 with [Backup & Restore (BR)](https://github.com/pingcap/br) ([#1280](https://github.com/pingcap/tidb-operator/pull/1280), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add basic defaulting and validating for `TidbCluster` ([#1429](https://github.com/pingcap/tidb-operator/pull/1429), [@aylei](https://github.com/aylei))
- Support scaling in/out with deleted slots feature of advanced StatefulSets ([#1361](https://github.com/pingcap/tidb-operator/pull/1361), [@cofyc](https://github.com/cofyc))
- Support initializing the TiDB cluster with TidbInitializer Custom Resource ([#1403](https://github.com/pingcap/tidb-operator/pull/1403), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Refine the configuration schema of PD/TiKV/TiDB ([#1411](https://github.com/pingcap/tidb-operator/pull/1411), [@aylei](https://github.com/aylei))
- Set the default name of the instance label key for `tidbcluster`-owned resources to the cluster name ([#1419](https://github.com/pingcap/tidb-operator/pull/1419), [@aylei](https://github.com/aylei))
- Extend the custom resource `TidbCluster` to support managing the Pump cluster ([#1269](https://github.com/pingcap/tidb-operator/pull/1269), [@aylei](https://github.com/aylei))
- Fix the default TiKV-importer configuration ([#1415](https://github.com/pingcap/tidb-operator/pull/1415), [@aylei](https://github.com/aylei))
- Expose ephemeral-storage in resource configuration ([#1398](https://github.com/pingcap/tidb-operator/pull/1398), [@aylei](https://github.com/aylei))
- Add e2e case of operating tidb-cluster without helm ([#1396](https://github.com/pingcap/tidb-operator/pull/1396), [@aylei](https://github.com/aylei))
- Expose terraform Aliyun ACK version and specify the default version to '1.14.8-aliyun.1' ([#1284](https://github.com/pingcap/tidb-operator/pull/1284), [@shonge](https://github.com/shonge))
- Refine error messages for the scheduler ([#1373](https://github.com/pingcap/tidb-operator/pull/1373), [@weekface](https://github.com/weekface))
- Bind the cluster-role `system:kube-scheduler` to the service account `tidb-scheduler` ([#1355](https://github.com/pingcap/tidb-operator/pull/1355), [@shonge](https://github.com/shonge))
- Add a new CRD TidbInitializer ([#1391](https://github.com/pingcap/tidb-operator/pull/1391), [@aylei](https://github.com/aylei))
- Upgrade the default backup image to pingcap/tidb-cloud-backup:20191217 and facilitate the `-r` option ([#1360](https://github.com/pingcap/tidb-operator/pull/1360), [@aylei](https://github.com/aylei))
- Fix Docker ulimit configuring for the latest EKS AMI ([#1349](https://github.com/pingcap/tidb-operator/pull/1349), [@aylei](https://github.com/aylei))
- Support sync pump status to tidb-cluster ([#1292](https://github.com/pingcap/tidb-operator/pull/1292), [@shonge](https://github.com/shonge))
- Support automatically creating and reconciling the tidb-discovery-service for `tidb-controller-manager` ([#1322](https://github.com/pingcap/tidb-operator/pull/1322), [@aylei](https://github.com/aylei))
- Make backup and restore more universal and secure ([#1276](https://github.com/pingcap/tidb-operator/pull/1276), [@onlymellb](https://github.com/onlymellb))
- Manage PD and TiKV configurations in the `TidbCluster` resource ([#1330](https://github.com/pingcap/tidb-operator/pull/1330), [@aylei](https://github.com/aylei))
- Support managing the configuration of tidb-server in the `TidbCluster` resource ([#1291](https://github.com/pingcap/tidb-operator/pull/1291), [@aylei](https://github.com/aylei))
- Add schema for configuration of TiKV ([#1306](https://github.com/pingcap/tidb-operator/pull/1306), [@aylei](https://github.com/aylei))
- Wait for the TiDB `host:port` to be opened before processing to initialize TiDB to speed up TiDB initialization ([#1296](https://github.com/pingcap/tidb-operator/pull/1296), [@cofyc](https://github.com/cofyc))
- Remove DinD related scripts ([#1283](https://github.com/pingcap/tidb-operator/pull/1283), [@shonge](https://github.com/shonge))
- Allow retrieving credentials from metadata on AWS and GCP ([#1248](https://github.com/pingcap/tidb-operator/pull/1248), [@gregwebs](https://github.com/gregwebs))
- Add the privilege to operate configmap for tidb-controller-manager ([#1275](https://github.com/pingcap/tidb-operator/pull/1275), [@aylei](https://github.com/aylei))
- Manage TiDB service in tidb-controller-manager ([#1242](https://github.com/pingcap/tidb-operator/pull/1242), [@aylei](https://github.com/aylei))
- Support the cluster-level setting for components ([#1193](https://github.com/pingcap/tidb-operator/pull/1193), [@aylei](https://github.com/aylei))
- Get the time string from the current time instead of the Pod name ([#1229](https://github.com/pingcap/tidb-operator/pull/1229), [@weekface](https://github.com/weekface))
- Operator will not resign the ddl owner anymore when upgrading tidb-servers because tidb-server will transfer ddl owner automatically on shutdown ([#1239](https://github.com/pingcap/tidb-operator/pull/1239), [@aylei](https://github.com/aylei))
- Fix the Google terraform module `use_ip_aliases` error ([#1206](https://github.com/pingcap/tidb-operator/pull/1206), [@tennix](https://github.com/tennix))
- Upgrade the default TiDB version to v3.0.5 ([#1179](https://github.com/pingcap/tidb-operator/pull/1179), [@shonge](https://github.com/shonge))
- Upgrade the base system of Docker images to the latest stable ([#1178](https://github.com/pingcap/tidb-operator/pull/1178), [@AstroProfundis](https://github.com/AstroProfundis))
- `tkctl get TiKV` now can show store state for each TiKV Pod ([#916](https://github.com/pingcap/tidb-operator/pull/916), [@Yisaer](https://github.com/Yisaer))
- Add an option to monitor across namespaces ([#907](https://github.com/pingcap/tidb-operator/pull/907), [@gregwebs](https://github.com/gregwebs))
- Add the `STOREID` column to show the store ID for each TiKV Pod in `tkctl get TiKV` ([#842](https://github.com/pingcap/tidb-operator/pull/842), [@Yisaer](https://github.com/Yisaer))
- Users can designate permitting host in chart values.tidb.permitHost ([#779](https://github.com/pingcap/tidb-operator/pull/779), [@shonge](https://github.com/shonge))
- Add the zone label and reserved resources arguments to kubelet ([#871](https://github.com/pingcap/tidb-operator/pull/871), [@aylei](https://github.com/aylei))
- Fix an issue that kubeconfig may be destroyed in the apply phrase ([#861](https://github.com/pingcap/tidb-operator/pull/861), [@cofyc](https://github.com/cofyc))
- Support canary release for the TiKV component ([#869](https://github.com/pingcap/tidb-operator/pull/869), [@onlymellb](https://github.com/onlymellb))
- Make the latest charts compatible with the old controller manager ([#856](https://github.com/pingcap/tidb-operator/pull/856), [@onlymellb](https://github.com/onlymellb))
- Add the basic support of TLS encrypted connections in the TiDB cluster ([#750](https://github.com/pingcap/tidb-operator/pull/750), [@AstroProfundis](https://github.com/AstroProfundis))
- Support tidb-operator to spec nodeSelector, affinity and tolerations ([#855](https://github.com/pingcap/tidb-operator/pull/855), [@shonge](https://github.com/shonge))
- Support configuring resources requests and limits for all containers of the TiDB cluster ([#853](https://github.com/pingcap/tidb-operator/pull/853), [@aylei](https://github.com/aylei))
- Support using Kind (Kubernetes IN Docker) to set up a testing environment ([#791](https://github.com/pingcap/tidb-operator/pull/791), [@xiaojingchen](https://github.com/xiaojingchen))
- Support add-hoc data source to be restored with the tidb-lightning chart ([#827](https://github.com/pingcap/tidb-operator/pull/827), [@tennix](https://github.com/tennix))
- Add the `tikvGCLifeTime` option ([#835](https://github.com/pingcap/tidb-operator/pull/835), [@weekface](https://github.com/weekface))
- Update the default backup image to pingcap/tidb-cloud-backup:20190828 ([#846](https://github.com/pingcap/tidb-operator/pull/846), [@aylei](https://github.com/aylei))
- Fix the Pump/Drainer data directory to avoid potential data loss ([#826](https://github.com/pingcap/tidb-operator/pull/826), [@aylei](https://github.com/aylei))
- Fix the issue that`tkctl` ouputs nothing with the `-oyaml` or `-ojson` flag and support viewing details of a specific Pod or PV, also improve the output of the `tkctl get` command ([#822](https://github.com/pingcap/tidb-operator/pull/822), [@onlymellb](https://github.com/onlymellb))
- Add recommendations options to mydumper: `-t 16 -F 64 --skip-tz-utc` ([#828](https://github.com/pingcap/tidb-operator/pull/828), [@weekface](https://github.com/weekface))
- Support zonal and multi-zonal clusters in deploy/gcp ([#809](https://github.com/pingcap/tidb-operator/pull/809), [@cofyc](https://github.com/cofyc))
- Fix ad-hoc backup when the default backup name is used ([#836](https://github.com/pingcap/tidb-operator/pull/836), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add the support for tidb-lightning ([#817](https://github.com/pingcap/tidb-operator/pull/817), [@tennix](https://github.com/tennix))
- Support restoring the TiDB cluster from a specified scheduled backup directory ([#804](https://github.com/pingcap/tidb-operator/pull/804), [@onlymellb](https://github.com/onlymellb))
- Fix an exception in the log of `tkctl` ([#797](https://github.com/pingcap/tidb-operator/pull/797), [@onlymellb](https://github.com/onlymellb))
- Add the `hostNetwork` field in PD/TiKV/TiDB spec to make it possible to run TiDB components in host network ([#774](https://github.com/pingcap/tidb-operator/pull/774), [@cofyc](https://github.com/cofyc))
- Use mdadm and RAID rather than LVM when it is available on GKE ([#789](https://github.com/pingcap/tidb-operator/pull/789), [@gregwebs](https://github.com/gregwebs))
- Users can now expand cloud storage PV dynamically by increasing the PVC storage size ([#772](https://github.com/pingcap/tidb-operator/pull/772), [@tennix](https://github.com/tennix))
- Support configuring node image types for PD/TiDB/TiKV node pools ([#776](https://github.com/pingcap/tidb-operator/pull/776), [@cofyc](https://github.com/cofyc))
- Add a script to delete unused disk for GKE ([#771](https://github.com/pingcap/tidb-operator/pull/771), [@gregwebs](https://github.com/gregwebs))
- Support `binlog.pump.config` and `binlog.drainer.config` configurations for Pump and Drainer ([#693](https://github.com/pingcap/tidb-operator/pull/693), [@weekface](https://github.com/weekface))
- Prevent the Pump progress from exiting with 0 if the Pump becomes `offline` ([#769](https://github.com/pingcap/tidb-operator/pull/769), [@weekface](https://github.com/weekface))
- Introduce a new helm chart, tidb-drainer, to facilitate multiple Drainers management ([#744](https://github.com/pingcap/tidb-operator/pull/744), [@aylei](https://github.com/aylei))
- Add the backup-manager tool to support backing up, restoring, and cleaning backup data ([#694](https://github.com/pingcap/tidb-operator/pull/694), [@onlymellb](https://github.com/onlymellb))
- Add `affinity` to Pump/Drainer configration ([#741](https://github.com/pingcap/tidb-operator/pull/741), [@weekface](https://github.com/weekface))
- Fix the TiKV scaling failure in some cases after TiKV failover ([#726](https://github.com/pingcap/tidb-operator/pull/726), [@onlymellb](https://github.com/onlymellb))
- Fix error handling for UpdateService ([#718](https://github.com/pingcap/tidb-operator/pull/718), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Reduce e2e run time from 60 m to 20 m ([#713](https://github.com/pingcap/tidb-operator/pull/713), [@weekface](https://github.com/weekface))
- Add the `AdvancedStatefulset` feature to use advanced StatefulSet instead of Kubernetes builtin StatefulSet ([#1108](https://github.com/pingcap/tidb-operator/pull/1108), [@cofyc](https://github.com/cofyc))
- Enable auto generate certificates for the TiDB cluster ([#782](https://github.com/pingcap/tidb-operator/pull/782), [@AstroProfundis](https://github.com/AstroProfundis))
- Support backup to gcs ([#1127](https://github.com/pingcap/tidb-operator/pull/1127), [@onlymellb](https://github.com/onlymellb))
- Support configuring `net.ipv4.tcp_keepalive_time` and `net.core.somaxconn` for TiDB and configuring `net.core.somaxconn` for TiKV ([#1107](https://github.com/pingcap/tidb-operator/pull/1107), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add basic e2e tests for aggregated apiserver ([#1109](https://github.com/pingcap/tidb-operator/pull/1109), [@aylei](https://github.com/aylei))
- Add the `enablePVReclaim` option to reclaim PV when tidb-operator scales in TiKV or PD ([#1037](https://github.com/pingcap/tidb-operator/pull/1037), [@onlymellb](https://github.com/onlymellb))
- Unify all S3 compliant storage to support backup and restore ([#1088](https://github.com/pingcap/tidb-operator/pull/1088), [@onlymellb](https://github.com/onlymellb))
- Set podSecuriyContext to nil by default ([#1079](https://github.com/pingcap/tidb-operator/pull/1079), [@aylei](https://github.com/aylei))
- Add tidb-apiserver in the tidb-operator chart ([#1083](https://github.com/pingcap/tidb-operator/pull/1083), [@aylei](https://github.com/aylei))
- Add new component TiDB aggregated apiserver ([#1048](https://github.com/pingcap/tidb-operator/pull/1048), [@aylei](https://github.com/aylei))
- Fix the issue that the tkctl version does not work when the release name is un-wanted ([#1065](https://github.com/pingcap/tidb-operator/pull/1065), [@aylei](https://github.com/aylei))
- Support pause for backup schedule ([#1047](https://github.com/pingcap/tidb-operator/pull/1047), [@onlymellb](https://github.com/onlymellb))
- Fix the issue that TiDB Loadbalancer is empty in terraform output ([#1045](https://github.com/pingcap/tidb-operator/pull/1045), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Fix that the `create_tidb_cluster_release` variable in AWS terraform script does not work ([#1062](https://github.com/pingcap/tidb-operator/pull/1062), [@aylei](https://github.com/aylei))
- Enable `ConfigMapRollout` by default in the stability test ([#1036](https://github.com/pingcap/tidb-operator/pull/1036), [@aylei](https://github.com/aylei))
- Migrate to use app/v1 and do not support Kubernetes before 1.9 anymore ([#1012](https://github.com/pingcap/tidb-operator/pull/1012), [@Yisaer](https://github.com/Yisaer))
- Suspend the ReplaceUnhealthy process for AWS TiKV auto-scaling-group ([#1014](https://github.com/pingcap/tidb-operator/pull/1014), [@aylei](https://github.com/aylei))
- Change the tidb-monitor-reloader image to pingcap/tidb-monitor-reloader:v1.0.1 ([#898](https://github.com/pingcap/tidb-operator/pull/898), [@qiffang](https://github.com/qiffang))
- Add some sysctl kernel parameter settings for tuning ([#1016](https://github.com/pingcap/tidb-operator/pull/1016), [@tennix](https://github.com/tennix))
- Support maximum retention time backups for backup schedule ([#979](https://github.com/pingcap/tidb-operator/pull/979), [@onlymellb](https://github.com/onlymellb))
- Upgrade the default TiDB version to v3.0.4 ([#837](https://github.com/pingcap/tidb-operator/pull/837), [@shonge](https://github.com/shonge))
- Fix values file customization for tidb-operator on Aliyun ([#971](https://github.com/pingcap/tidb-operator/pull/971), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Add the `maxFailoverCount` limit to TiKV ([#965](https://github.com/pingcap/tidb-operator/pull/965), [@weekface](https://github.com/weekface))
- Support setting custom tidb-operator values in terraform script for AWS ([#946](https://github.com/pingcap/tidb-operator/pull/946), [@aylei](https://github.com/aylei))
- Convert the TiKV capacity into MiB when it is not a multiple of GiB ([#942](https://github.com/pingcap/tidb-operator/pull/942), [@cofyc](https://github.com/cofyc))
- Fix Drainer misconfiguration ([#939](https://github.com/pingcap/tidb-operator/pull/939), [@weekface](https://github.com/weekface))
- Support correctly deploying tidb-operator and tidb-cluster with customized `values.yaml` ([#959](https://github.com/pingcap/tidb-operator/pull/959), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Support specifying SecurityContext for PD, TiKV and TiDB Pods and enable tcp keepalive for AWS ([#915](https://github.com/pingcap/tidb-operator/pull/915), [@aylei](https://github.com/aylei))
