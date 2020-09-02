---
title: TiDB Operator 1.1 RC.1 Release Notes
---

# TiDB Operator 1.1 RC.1 Release Notes

Release date: April 1, 2020

TiDB Operator version: 1.1.0-rc.1

## Action Required

- `--advertise-address` will be configured for `tidb-server`, which would trigger rolling-upgrade for the `tidb-server` component. You can set `spec.paused` to `true` before upgrading tidb-operator to avoid the rolling upgrade, and set it back to `false` when you are ready to upgrade your TiDB server ([#2076](https://github.com/pingcap/tidb-operator/pull/2076), [@cofyc](https://github.com/cofyc))
- Add the `tlsClient.tlsSecret` field in the backup and restore spec, which supports specifying a secret name that includes the cert ([#2003](https://github.com/pingcap/tidb-operator/pull/2003), [@shuijing198799](https://github.com/shuijing198799))
- Remove `spec.br.pd`, `spec.br.ca`, `spec.br.cert`, `spec.br.key` and add `spec.br.cluster`, `spec.br.clusterNamespace` for the `Backup`, `Restore` and `BackupSchedule` custom resources, which makes the BR configuration more reasonable ([#1836](https://github.com/pingcap/tidb-operator/pull/1836), [@shuijing198799](https://github.com/shuijing198799))

## Other Notable Changes

- Use `tidb-lightning` in `Restore` instead of `loader` ([#2068](https://github.com/pingcap/tidb-operator/pull/2068), [@Yisaer](https://github.com/Yisaer))
- Add `cert-allowed-cn` support to TiDB components ([#2061](https://github.com/pingcap/tidb-operator/pull/2061), [@weekface](https://github.com/weekface))
- Fix the PD `location-labels` configuration ([#1941](https://github.com/pingcap/tidb-operator/pull/1941), [@aylei](https://github.com/aylei))
- Able to pause and unpause TiDB cluster deployment via `spec.paused` ([#2013](https://github.com/pingcap/tidb-operator/pull/2013), [@cofyc](https://github.com/cofyc))
- Default the `max-backups` for TiDB server configuration to `3` if the TiDB cluster is deployed by CR ([#2045](https://github.com/pingcap/tidb-operator/pull/2045), [@Yisaer](https://github.com/Yisaer))
- Able to configure custom environments for components ([#2052](https://github.com/pingcap/tidb-operator/pull/2052), [@cofyc](https://github.com/cofyc))
- Fix the error that `kubectl get tc` cannot show correct images ([#2031](https://github.com/pingcap/tidb-operator/pull/2031), [@Yisaer](https://github.com/Yisaer))
    1. Default the `spec.tikv.maxFailoverCount` and `spec.tidb.maxFailoverCount` to `3` when they are not defined
    2. Disable auto-failover when `maxFailoverCount` is set to `0` ([#2015](https://github.com/pingcap/tidb-operator/pull/2015), [@Yisaer](https://github.com/Yisaer))
- Support deploying TiDB clusters with TidbCluster and TidbMonitor CRs via Terraform on ACK ([#2012](https://github.com/pingcap/tidb-operator/pull/2012), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Update PDConfig for TidbCluster to PD v3.1.0 ([#1928](https://github.com/pingcap/tidb-operator/pull/1928), [@Yisaer](https://github.com/Yisaer))
- Support deploying TiDB clusters with TidbCluster and TidbMonitor CRs via Terraform on AWS ([#2004](https://github.com/pingcap/tidb-operator/pull/2004), [@DanielZhangQD](https://github.com/DanielZhangQD))
- Update TidbConfig for TidbCluster to TiDB v3.1.0 ([#1906](https://github.com/pingcap/tidb-operator/pull/1906), [@Yisaer](https://github.com/Yisaer))
- Allow users to define resources for initContainers in TiDB initializer job ([#1938](https://github.com/pingcap/tidb-operator/pull/1938), [@tfulcrand](https://github.com/tfulcrand))
- Add TLS support for Pump and Drainer ([#1979](https://github.com/pingcap/tidb-operator/pull/1979), [@weekface](https://github.com/weekface))
- Add documents and examples for auto-scaler and initializer ([#1772](https://github.com/pingcap/tidb-operator/pull/1772), [@Yisaer](https://github.com/Yisaer))
    1. Add check to guarantee the NodePort won't be changed if the serviceType of TidbMonitor is NodePort
    2. Add EnvVar sort to avoid the monitor rendering different results from the same TidbMonitor spec
    3. Fix the problem that the TidbMonitor LoadBalancer IP is not used ([#1962](https://github.com/pingcap/tidb-operator/pull/1962), [@Yisaer](https://github.com/Yisaer))
- Make tidb-initializer support TLS ([#1931](https://github.com/pingcap/tidb-operator/pull/1931), [@weekface](https://github.com/weekface))
    1. Fix the problem that Advanced StatefulSet cannot work with webhook
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
    1. Change the setting from the previous `admission.hookEnabled.pods` to the `admission.validation.pods`
    2. Change the setting from the previous `admission.hookEnabled.statefulSets` to the `admission.validation.statefulSets`
    3. Change the setting from the previous `admission.hookEnabled.validating` to the `admission.validation.pingcapResources`
    4. Change the setting from the previous `admission.hookEnabled.defaulting` to the `admission.mutation.pingcapResources`
    5. Change the setting from the previous `admission.failurePolicy.defaulting` to the `admission.failurePolicy.mutation`
    6. Change the setting from the previous `admission.failurePolicy.*` to the `admission.failurePolicy.validation` ([#1832](https://github.com/pingcap/tidb-operator/pull/1832), [@Yisaer](https://github.com/Yisaer))
- Enable TidbCluster defaulting mutation by default which is recommended when admission webhook is used ([#1816](https://github.com/pingcap/tidb-operator/pull/1816), [@Yisaer](https://github.com/Yisaer))
- Fix a bug that TiKV fails to start while creating the cluster using CR with cluster TLS enabled ([#1808](https://github.com/pingcap/tidb-operator/pull/1808), [@weekface](https://github.com/weekface))
- Support using prefix in remote storage during backup/restore ([#1790](https://github.com/pingcap/tidb-operator/pull/1790), [@DanielZhangQD](https://github.com/DanielZhangQD))
