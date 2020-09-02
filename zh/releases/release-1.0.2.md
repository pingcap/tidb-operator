---
title: TiDB Operator 1.0.2 Release Notes
---

# TiDB Operator 1.0.2 Release Notes

Release date: November 1, 2019

TiDB Operator version: 1.0.2

## v1.0.2 What's New

### Action Required

The AWS Terraform script uses auto-scaling-group for all components (PD/TiKV/TiDB/monitor). When an ec2 instance fails the health check, the instance will be replaced. This is helpful for those applications that are stateless or use EBS volumes to store data.

But a TiKV Pod uses instance store to store its data. When an instance is replaced, all the data on its store will be lost. TiKV has to resync all data to the newly added instance. Though TiDB is a distributed database and can work when a node fails, resyncing data can cost much if the dataset is large. Besides, the ec2 instance may be recovered to a healthy state by rebooting.

So we disabled the auto-scaling-group's replacing behavior in `v1.0.2`.

Auto-scaling-group scaling process can also be suspended according to its [documentation](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-suspend-resume-processes.html) if you are using `v1.0.1` or prior versions.

### Improvements

- Suspend ReplaceUnhealthy process for AWS TiKV auto-scaling-group
- Add a new VM manager `qm` in stability test
- Add `tikv.maxFailoverCount` limit to TiKV
- Set the default `externalTrafficPolicy` to be `Local` for TiDB service in AWS/GCP/Aliyun
- Add provider and module versions for AWS

### Bug Fixes

- Fix the issue that tkctl version does not work when the release name is un-wanted
- Migrate statefulsets apiVersion to `app/v1` which fixes compatibility with Kubernetes 1.16 and above versions
- Fix the issue that the `create_tidb_cluster_release` variable in AWS Terraform script does not work
- Fix compatibility issues by adding `v1beta1` to statefulset apiVersions
- Fix the issue that TiDB Loadbalancer is empty in Terraform output
- Fix a compatibility issue of TiKV `maxFailoverCount`
- Fix Terraform providers version constraint issues for GCP and Aliyun
- Fix values file customization for tidb-operator on Aliyun
- Fix tidb-operator crash when users modify statefulset upgrade strategy improperly
- Fix drainer misconfiguration

## Detailed Bug Fixes and Changes

- Fix the issue that tkctl version does not work when the release name is un-wanted ([#1065](https://github.com/pingcap/tidb-operator/pull/1065))
- Fix the issue that the `create_tidb_cluster_release` variable in AWS terraform script does not work ([#1062](https://github.com/pingcap/tidb-operator/pull/1062))
- Fix compatibility issues for ([#1012](https://github.com/pingcap/tidb-operator/pull/1012)): add `v1beta1` to statefulset apiVersions ([#1054](https://github.com/pingcap/tidb-operator/pull/1054))
- Enable ConfigMapRollout by default in stability test ([#1036](https://github.com/pingcap/tidb-operator/pull/1036))
- Fix the issue that TiDB Loadbalancer is empty in Terraform output ([#1045](https://github.com/pingcap/tidb-operator/pull/1045))
- Migrate statefulsets apiVersion to `app/v1` which fixes compatibility with Kubernetes 1.16 and above versions ([#1012](https://github.com/pingcap/tidb-operator/pull/1012))
- Only expect TiDB cluster upgrade to be complete when rolling back wrong configuration in stability test ([#1030](https://github.com/pingcap/tidb-operator/pull/1030))
- Suspend ReplaceUnhealthy process for AWS TiKV auto-scaling-group ([#1014](https://github.com/pingcap/tidb-operator/pull/1014))
- Add a new VM manager `qm` in stability test ([#896](https://github.com/pingcap/tidb-operator/pull/896))
- Fix provider versions constraint issues for GCP and Aliyun ([#959](https://github.com/pingcap/tidb-operator/pull/959))
- Fix values file customization for tidb-operator on Aliyun ([#971](https://github.com/pingcap/tidb-operator/pull/971))
- Fix a compatibility issue of TiKV `tikv.maxFailoverCount` ([#977](https://github.com/pingcap/tidb-operator/pull/977))
- Add `tikv.maxFailoverCount` limit to TiKV ([#965](https://github.com/pingcap/tidb-operator/pull/965))
- Fix tidb-operator crash when users modify statefulset upgrade strategy improperly ([#912](https://github.com/pingcap/tidb-operator/pull/912))
- Set the default `externalTrafficPolicy` to be `Local` for TiDB service in AWS/GCP/Aliyun ([#947](https://github.com/pingcap/tidb-operator/pull/947))
- Add note about setting PV reclaim policy to retain ([#911](https://github.com/pingcap/tidb-operator/pull/911))
- Fix drainer misconfiguration ([#939](https://github.com/pingcap/tidb-operator/pull/939))
- Add provider and module versions for AWS ([#926](https://github.com/pingcap/tidb-operator/pull/926))
