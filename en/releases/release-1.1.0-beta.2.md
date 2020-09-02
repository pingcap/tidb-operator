---
title: TiDB Operator 1.1 Beta.2 Release Notes
---

# TiDB Operator 1.1 Beta.2 Release Notes

Release date: February 26, 2020

TiDB Operator version: 1.1.0-beta.2

## Action Required

- `--default-storage-class-name` and `--default-backup-storage-class-name` are abandoned, and the storage class defaults to Kubernetes default storage class right now. If you have set default storage class different than Kubernetes default storage class, please set them explicitly in your TiDB cluster helm or YAML files. ([#1581](https://github.com/pingcap/tidb-operator/pull/1581), [@cofyc](https://github.com/cofyc))

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
