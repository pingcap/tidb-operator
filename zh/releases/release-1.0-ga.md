---
title: TiDB Operator 1.0 GA Release Notes
---

# TiDB Operator 1.0 GA Release Notes

Release date: July 30, 2019

TiDB Operator version: 1.0.0

## v1.0.0 What's New

### Action Required

- ACTION REQUIRED: `tikv.storeLabels` was removed from `values.yaml`. You can directly set it with `location-labels` in `pd.config`.
- ACTION REQUIRED: the `--features` flag of tidb-scheduler has been updated to the `key={true,false}` format. You can enable the feature by appending `=true`.
- ACTION REQUIRED: you need to change the configurations in `values.yaml` of previous chart releases to the new `values.yaml` of the new chart. Otherwise, the configurations will be ignored when upgrading the TiDB cluster with the new chart.

The `pd` section in old `values.yaml`:

```
pd:
  logLevel: info
  maxStoreDownTime: 30m
  maxReplicas: 3
```

The `pd` section in new `values.yaml`:

```
pd:
  config: |
    [log]
    level = "info"
    [schedule]
    max-store-down-time = "30m"
    [replication]
    max-replicas = 3
```

The `tikv` section in old `values.yaml`:

```
tikv:
  logLevel: info
  syncLog: true
  readpoolStorageConcurrency: 4
  readpoolCoprocessorConcurrency: 8
  storageSchedulerWorkerPoolSize: 4
```

The `tikv` section in new `values.yaml`:

```
tikv:
  config: |
    log-level = "info"
    [server]
    status-addr = "0.0.0.0:20180"
    [raftstore]
    sync-log = true
    [readpool.storage]
    high-concurrency = 4
    normal-concurrency = 4
    low-concurrency = 4
    [readpool.coprocessor]
    high-concurrency = 8
    normal-concurrency = 8
    low-concurrency = 8
    [storage]
    scheduler-worker-pool-size = 4
```

The `tidb` section in old `values.yaml`:

```
tidb:
  logLevel: info
  preparedPlanCacheEnabled: false
  preparedPlanCacheCapacity: 100
  txnLocalLatchesEnabled: false
  txnLocalLatchesCapacity: "10240000"
  tokenLimit: "1000"
  memQuotaQuery: "34359738368"
  txnEntryCountLimit: "300000"
  txnTotalSizeLimit: "104857600"
  checkMb4ValueInUtf8: true
  treatOldVersionUtf8AsUtf8mb4: true
  lease: 45s
  maxProcs: 0
```

The `tidb` section in new `values.yaml`:

```
tidb:
  config: |
    token-limit = 1000
    mem-quota-query = 34359738368
    check-mb4-value-in-utf8 = true
    treat-old-version-utf8-as-utf8mb4 = true
    lease = "45s"
    [log]
    level = "info"
    [prepared-plan-cache]
    enabled = false
    capacity = 100
    [txn-local-latches]
    enabled = false
    capacity = 10240000
    [performance]
    txn-entry-count-limit = 300000
    txn-total-size-limit = 104857600
    max-procs = 0
```

The `monitor` section in old `values.yaml`:

```
monitor:
  create: true
  ...
```

The `monitor` section in new `values.yaml`:

```
monitor:
  create: true
   initializer:
     image: pingcap/tidb-monitor-initializer:v3.0.5
     imagePullPolicy: IfNotPresent
   reloader:
     create: true
     image: pingcap/tidb-monitor-reloader:v1.0.0
     imagePullPolicy: IfNotPresent
     service:
       type: NodePort
  ...
```

Please check [cluster configuration](https://pingcap.com/docs/v3.0/tidb-in-kubernetes/reference/configuration/tidb-cluster/) for detailed configuration.

### Stability Test Cases Added

- Stop all etcds and kubelets

### Improvements

- Simplify GKE SSD setup
- Modularization for AWS Terraform scripts
- Turn on the automatic failover feature by default
- Enable configmap rollout by default
- Enable stable scheduling by default
- Support multiple TiDB clusters management in Alibaba Cloud
- Enable AWS NLB cross zone load balancing by default

### Bug Fixes

- Fix sysbench installation on bastion machine of AWS deployment
- Fix TiKV metrics monitoring in default setup

## Detailed Bug Fixes and Changes

- Allow upgrading TiDB monitor along with TiDB version ([#666](https://github.com/pingcap/tidb-operator/pull/666))
- Specify the TiKV status address to fix monitoring ([#695](https://github.com/pingcap/tidb-operator/pull/695))
- Fix sysbench installation on bastion machine for AWS deployment ([#688](https://github.com/pingcap/tidb-operator/pull/688))
- Update the `git add upstream` command to use `https` in contributing document ([#690](https://github.com/pingcap/tidb-operator/pull/690))
- Stability cases: stop kubelet and etcd ([#665](https://github.com/pingcap/tidb-operator/pull/665))
- Limit test cover packages ([#687](https://github.com/pingcap/tidb-operator/pull/687))
- Enable nlb cross zone load balancing by default ([#686](https://github.com/pingcap/tidb-operator/pull/686))
- Add TiKV raftstore parameters ([#681](https://github.com/pingcap/tidb-operator/pull/681))
- Support multiple TiDB clusters management for Alibaba Cloud ([#658](https://github.com/pingcap/tidb-operator/pull/658))
- Adjust the `EndEvictLeader` function ([#680](https://github.com/pingcap/tidb-operator/pull/680))
- Add more logs ([#676](https://github.com/pingcap/tidb-operator/pull/676))
- Update feature gates to support `key={true,false}` syntax ([#677](https://github.com/pingcap/tidb-operator/pull/677))
- Fix the typo meke to make ([#679](https://github.com/pingcap/tidb-operator/pull/679))
- Enable configmap rollout by default and quote configmap digest suffix ([#678](https://github.com/pingcap/tidb-operator/pull/678))
- Turn automatic failover on ([#667](https://github.com/pingcap/tidb-operator/pull/667))
- Sets node count for default pool equal to total desired node count ([#673](https://github.com/pingcap/tidb-operator/pull/673))
- Upgrade default TiDB version to v3.0.1 ([#671](https://github.com/pingcap/tidb-operator/pull/671))
- Remove storeLabels ([#663](https://github.com/pingcap/tidb-operator/pull/663))
- Change the way to configure TiDB/TiKV/PD in charts ([#638](https://github.com/pingcap/tidb-operator/pull/638))
- Modularize for AWS terraform scripts ([#650](https://github.com/pingcap/tidb-operator/pull/650))
- Change the `DeferClose` function ([#653](https://github.com/pingcap/tidb-operator/pull/653))
- Increase the default storage size for Pump from 10Gi to 20Gi in response to `stop-write-at-available-space` ([#657](https://github.com/pingcap/tidb-operator/pull/657))
- Simplify local SDD setup ([#644](https://github.com/pingcap/tidb-operator/pull/644))
