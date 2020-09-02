---
title: TiDB Operator 1.1 Beta.1 Release Notes
---

# TiDB Operator 1.1 Beta.1 Release Notes

Release date: January 8, 2020

TiDB Operator version: 1.1.0-beta.1

## Action Required

- ACTION REQUIRED: Add the `timezone` support for [all charts](https://github.com/pingcap/tidb-operator/tree/master/charts) ([#1122](https://github.com/pingcap/tidb-operator/pull/1122), [@weekface](https://github.com/weekface)).

    For the `tidb-cluster` chart, we already have the `timezone` option (`UTC` by default). If the user does not change it to a different value (for example: `Aisa/Shanghai`), all Pods will not be recreated.

    If the user changes it to another value (for example: `Aisa/Shanghai`), all the related Pods (add a `TZ` env) will be recreated (rolling update).

    Regarding other charts, we don't have a `timezone` option in their `values.yaml`. We add the `timezone` option in this PR. No matter whether the user uses the old `values.yaml` or the new `values.yaml`, all the related Pods (add a `TZ` env) will not be recreated (rolling update).

    The related Pods include `pump`, `drainer`, `discovery`, `monitor`, `scheduled backup`, `tidb-initializer`, and `tikv-importer`.

    All images' time zone maintained by `tidb-operator` is `UTC`. If you use your own images, you need to make sure that the time zone inside your images is `UTC`.

## Other Notable Changes

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
- Support ad-hoc data source to be restored with the tidb-lightning chart ([#827](https://github.com/pingcap/tidb-operator/pull/827), [@tennix](https://github.com/tennix))
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
