---
title: TiDB Operator 1.0.6 Release Notes
---

# TiDB Operator 1.0.6 Release Notes

Release date: December 27, 2019

TiDB Operator version: 1.0.6

## v1.0.6 What's New

Action required: Users should migrate the configs in `values.yaml` of previous chart releases to the new `values.yaml` of the new chart. Otherwise, the monitor pods might fail when you upgrade the monitor with the new chart.

For example, configs in the old `values.yaml` file:

```
monitor:
  ...
  initializer:
    image: pingcap/tidb-monitor-initializer:v3.0.5
    imagePullPolicy: IfNotPresent
  ...
```

After migration, configs in the new `values.yaml` file should be as follows:

```
monitor:
  ...
  initializer:
    image: pingcap/tidb-monitor-initializer:v3.0.5
    imagePullPolicy: Always
    config:
      K8S_PROMETHEUS_URL: http://prometheus-k8s.monitoring.svc:9090
  ...
```

### Monitor

- Enable alert rule persistence ([#898](https://github.com/pingcap/tidb-operator/pull/898))
- Add node & pod info in TiDB Grafana ([#885](https://github.com/pingcap/tidb-operator/pull/885))

### TiDB Scheduler

- Refine scheduler error messages ([#1373](https://github.com/pingcap/tidb-operator/pull/1373))

### Compatibility

- Fix the compatibility issue in Kubernetes v1.17 ([#1241](https://github.com/pingcap/tidb-operator/pull/1241))
- Bind the `system:kube-scheduler` ClusterRole to the `tidb-scheduler` service account ([#1355](https://github.com/pingcap/tidb-operator/pull/1355))

### TiKV Importer

- Fix the default `tikv-importer` configuration ([#1415](https://github.com/pingcap/tidb-operator/pull/1415))

### E2E

- Ensure pods unaffected when upgrading ([#955](https://github.com/pingcap/tidb-operator/pull/955))

### CI

- Move the release CI script from Jenkins into the tidb-operator repository ([#1237](https://github.com/pingcap/tidb-operator/pull/1237))
- Adjust the release CI script for the `release-1.0` branch ([#1320](https://github.com/pingcap/tidb-operator/pull/1320))
