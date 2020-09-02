---
title: TiDB Operator 1.0.4 Release Notes
---

# TiDB Operator 1.0.4 Release Notes

Release date: November 23, 2019

TiDB Operator version: 1.0.4

## v1.0.4 What's New

### Action Required

There is no action required if you are upgrading from [v1.0.3](release-1.0.3.md).

### Highlights

[#1202](https://github.com/pingcap/tidb-operator/pull/1202) introduced `HostNetwork` support, which offers better performance compared to the Pod network. Check out our [benchmark report](https://pingcap.com/docs/dev/benchmark/sysbench-in-k8s/#pod-network-vs-host-network) for details.

> **Note:**
>
> Due to [this issue of Kubernetes](https://github.com/kubernetes/kubernetes/issues/78420), the Kubernetes cluster must be one of the following versions to enable `HostNetwork` of the TiDB cluster:
>
> - `v1.13.11` or later
> - `v1.14.7` or later
> - `v1.15.4` or later
> - any version since `v1.16.0`

[#1175](https://github.com/pingcap/tidb-operator/pull/1175) added the `podSecurityContext` support for TiDB cluster Pods. We recommend setting the namespaced kernel parameters for TiDB cluster Pods according to our [Environment Recommendation](https://pingcap.com/docs/dev/tidb-in-kubernetes/deploy/prerequisites/#the-configuration-of-kernel-parameters).

New Helm chart `tidb-lightning` brings [TiDB Lightning](https://pingcap.com/docs/stable/reference/tools/tidb-lightning/overview/) support for TiDB in Kubernetes. Check out the [document](https://pingcap.com/docs/dev/tidb-in-kubernetes/maintain/lightning/) for detailed user guide.

Another new Helm chart `tidb-drainer` brings multiple drainers support for TiDB Binlog in Kubernetes. Check out the [document](https://pingcap.com/docs/dev/tidb-in-kubernetes/maintain/tidb-binlog/#deploy-multiple-drainers) for detailed user guide.

### Improvements

- Support HostNetwork ([#1202](https://github.com/pingcap/tidb-operator/pull/1202))
- Support configuring sysctls for Pods and enable net.* ([#1175](https://github.com/pingcap/tidb-operator/pull/1175))
- Add tidb-lightning support ([#1161](https://github.com/pingcap/tidb-operator/pull/1161))
- Add new helm chart tidb-drainer to support multiple drainers ([#1160](https://github.com/pingcap/tidb-operator/pull/1160))

## Detailed Bug Fixes and Changes

- Add e2e scripts and simplify the e2e Jenkins file ([#1174](https://github.com/pingcap/tidb-operator/pull/1174))
- Fix the pump/drainer data directory to avoid data loss caused by bad configuration ([#1183](https://github.com/pingcap/tidb-operator/pull/1183))
- Add init sql case to e2e ([#1199](https://github.com/pingcap/tidb-operator/pull/1199))
- Keep the instance label of drainer same with the TiDB cluster in favor of monitoring ([#1170](https://github.com/pingcap/tidb-operator/pull/1170))
- Set `podSecuriyContext` to nil by default in favor of backward compatibility ([#1184](https://github.com/pingcap/tidb-operator/pull/1184))

## Additional Notes for Users of v1.1.0.alpha branch

For historical reasons, `v1.1.0.alpha` is a hot-fix branch and got this name by mistake. All fixes in that branch are cherry-picked to `v1.0.4` and the `v1.1.0.alpha` branch will be discarded to keep things clear.

We strongly recommend you to upgrade to `v1.0.4` if you are using any version under `v1.1.0.alpha`.

`v1.0.4` introduces the following fixes comparing to `v1.1.0.alpha.3`:

- Support HostNetwork ([#1202](https://github.com/pingcap/tidb-operator/pull/1202))
- Add the permit host option for tidb-initializer job ([#779](https://github.com/pingcap/tidb-operator/pull/779))
- Fix drainer misconfiguration in tidb-cluster chart ([#945](https://github.com/pingcap/tidb-operator/pull/945))
- Set the default `externalTrafficPolicy` to be Local for TiDB services ([#960](https://github.com/pingcap/tidb-operator/pull/960))
- Fix tidb-operator crash when users modify sts upgrade strategy improperly ([#969](https://github.com/pingcap/tidb-operator/pull/969))
- Add the `maxFailoverCount` limit to TiKV ([#976](https://github.com/pingcap/tidb-operator/pull/976))
- Fix values file customization for tidb-operator on Aliyun ([#983](https://github.com/pingcap/tidb-operator/pull/983))
- Do not limit failover count when maxFailoverCount = 0 ([#978](https://github.com/pingcap/tidb-operator/pull/978))
- Suspend the `ReplaceUnhealthy` process for TiKV auto-scaling-group on AWS ([#1027](https://github.com/pingcap/tidb-operator/pull/1027))
- Fix the issue that the `create_tidb_cluster_release` variable does not work ([#1066](https://github.com/pingcap/tidb-operator/pull/1066)))
- Add `v1` to statefulset apiVersions ([#1056](https://github.com/pingcap/tidb-operator/pull/1056))
- Add timezone support ([#1126](https://github.com/pingcap/tidb-operator/pull/1027))
