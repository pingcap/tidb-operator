---
title: TiDB Operator 1.0 Beta.3 Release Notes
---

# TiDB Operator 1.0 Beta.3 Release Notes

Release date: June 6, 2019

TiDB Operator version: 1.0.0-beta.3

## v1.0.0-beta.3 What’s New

### Action Required

- ACTION REQUIRED: `nodeSelectorRequired` was removed from values.yaml.
- ACTION REQUIRED: Comma-separated values support in `nodeSelector` has been dropped, please use new-added `affinity` field which has a more expressive syntax.

### A lot of stability cases added

- ConfigMap rollout
- One PD replicas
- Stop TiDB Operator itself
- TiDB stable scheduling
- Disaster tolerance and data regions disaster tolerance
- Fix many bugs of stability test

### New Features

- Introduce ConfigMap rollout management. With the feature gate open, configuration file changes will be automatically applied to the cluster via a rolling update. Currently, the `scheduler` and `replication` configurations of PD can not be changed via ConfigMap rollout. You can use `pd-ctl` to change these values instead, see [#487](https://github.com/pingcap/tidb-operator/pull/487) for details.
- Support stable scheduling for pods of TiDB members in tidb-scheduler.
- Support adding additional pod annotations for PD/TiKV/TiDB,  e.g. [fluentbit.io/parser](https://docs.fluentbit.io/manual/filter/kubernetes#kubernetes-annotations).
- Support the affinity feature of k8s which can define the rule of assigning pods to nodes
- Allow pausing during TiDB upgrade

### Documentation Improvement

- GCP one-command deployment
- Refine user guides
- Improve GKE, AWS, Aliyun guide

### Pass User Acceptance Tests

### Other improvements

- Upgrade default TiDB version to v3.0.0-rc.1
- Fix a bug in reporting assigned nodes of TiDB members
- `tkctl get` can show cpu usage correctly now
- Adhoc backup now appends the start time to the PVC name by default.
- Add the privileged option for TiKV pod
- `tkctl upinfo` can show nodeIP podIP port now
- Get TS and use it before full backup using mydumper
- Fix capabilities issue for `tkctl debug` command

## Detailed Bug Fixes and Changes

- Add capabilities and privilege mode for debug container ([#537](https://github.com/pingcap/tidb-operator/pull/537))
- Note helm versions in deployment docs ([#553](https://github.com/pingcap/tidb-operator/pull/553))
- Split public and private subnets when using existing vpc ([#530](https://github.com/pingcap/tidb-operator/pull/530))
- Release v1.0.0-beta.3 ([#557](https://github.com/pingcap/tidb-operator/pull/557))
- GKE terraform upgrade to 0.12 and fix bastion instance zone to be region agnostic ([#554](https://github.com/pingcap/tidb-operator/pull/554))
- Get TS and use it before full backup using mydumper ([#534](https://github.com/pingcap/tidb-operator/pull/534))
- Add port podip nodeip to tkctl upinfo ([#538](https://github.com/pingcap/tidb-operator/pull/538))
- Fix disaster tolerance of stability test ([#543](https://github.com/pingcap/tidb-operator/pull/543))
- Add privileged option for TiKV pod template ([#550](https://github.com/pingcap/tidb-operator/pull/550))
- Use staticcheck instead of megacheck ([#548](https://github.com/pingcap/tidb-operator/pull/548))
- Refine backup and restore documentation ([#518](https://github.com/pingcap/tidb-operator/pull/518))
- Fix stability tidb pause case ([#542](https://github.com/pingcap/tidb-operator/pull/542))
- Fix tkctl get cpu info rendering ([#536](https://github.com/pingcap/tidb-operator/pull/536))
- Fix Aliyun tf output rendering and refine documents ([#511](https://github.com/pingcap/tidb-operator/pull/511))
- Make webhook configurable ([#529](https://github.com/pingcap/tidb-operator/pull/529))
- Add pods disaster tolerance and data regions disaster tolerance test cases ([#497](https://github.com/pingcap/tidb-operator/pull/497))
- Remove helm hook annotation for initializer job ([#526](https://github.com/pingcap/tidb-operator/pull/526))
- Add stable scheduling e2e test case ([#524](https://github.com/pingcap/tidb-operator/pull/524))
- Upgrade TiDB version in related documentations ([#532](https://github.com/pingcap/tidb-operator/pull/532))
- Fix a bug in reporting assigned nodes of TiDB members ([#531](https://github.com/pingcap/tidb-operator/pull/531))
- Reduce wait time and fix stability test ([#525](https://github.com/pingcap/tidb-operator/pull/525))
- Fix documentation usability issues in GCP document ([#519](https://github.com/pingcap/tidb-operator/pull/519))
- PD replicas 1 and stop tidb-operator ([#496](https://github.com/pingcap/tidb-operator/pull/496))
- Pause-upgrade stability test ([#521](https://github.com/pingcap/tidb-operator/pull/521))
- Fix restore script bug ([#510](https://github.com/pingcap/tidb-operator/pull/510))
- Retry truncating sst files upon failure ([#484](https://github.com/pingcap/tidb-operator/pull/484))
- Upgrade default TiDB to v3.0.0-rc.1 ([#520](https://github.com/pingcap/tidb-operator/pull/520))
- Add `--namespace` when creating backup secret ([#515](https://github.com/pingcap/tidb-operator/pull/515))
- New stability test case for ConfigMap rollout ([#499](https://github.com/pingcap/tidb-operator/pull/499))
- Fix issues found in Queeny's test ([#507](https://github.com/pingcap/tidb-operator/pull/507))
- Pause rolling-upgrade process of TiDB statefulset ([#470](https://github.com/pingcap/tidb-operator/pull/470))
- GKE terraform and guide ([#493](https://github.com/pingcap/tidb-operator/pull/493))
- Support the affinity feature of Kubernetes which defines the rule of assigning pods to nodes ([#475](https://github.com/pingcap/tidb-operator/pull/475))
- Support adding additional pod annotations for PD/TiKV/TiDB ([#500](https://github.com/pingcap/tidb-operator/pull/500))
- Document PD configuration issue ([#504](https://github.com/pingcap/tidb-operator/pull/504))
- Refine Aliyun and AWS cloud TiDB configurations ([#492](https://github.com/pingcap/tidb-operator/pull/492))
- Update wording and add note ([#502](https://github.com/pingcap/tidb-operator/pull/502))
- Support stable scheduling for TiDB ([#477](https://github.com/pingcap/tidb-operator/pull/477))
- Fix `make lint` ([#495](https://github.com/pingcap/tidb-operator/pull/495))
- Support updating configuration on the fly ([#479](https://github.com/pingcap/tidb-operator/pull/479))
- Update AWS deploy docs after testing ([#491](https://github.com/pingcap/tidb-operator/pull/491))
- Add release-note to pull_request_template.md ([#490](https://github.com/pingcap/tidb-operator/pull/490))
- Design proposal of stable scheduling in TiDB ([#466](https://github.com/pingcap/tidb-operator/pull/466))
- Update DinD image to make it possible to configure HTTP proxies ([#485](https://github.com/pingcap/tidb-operator/pull/485))
- Fix a broken link ([#489](https://github.com/pingcap/tidb-operator/pull/489))
- Fix typo ([#483](https://github.com/pingcap/tidb-operator/pull/483))
