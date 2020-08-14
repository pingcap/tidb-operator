---
title: What's New in TiDB Operator 1.1
summary: Learn the new features in TiDB Operator 1.1.
aliases: ['/docs/tidb-in-kubernetes/dev/whats-new-in-v1.1/']
---

# What's New in TiDB Operator 1.1

Based on 1.0, TiDB Operator 1.1 has several new features, including TiDB 4.0 support, TiKV data encryption, and TLS certificate configuration. TiDB Operator v1.1 also supports deploying new components such as TiFlash and TiCDC.

TiDB Operator 1.1 also makes improvements in usability, providing the user experience that is consistent with the Kubernetes native resources.

## Extensibility

- `TidbCluster` CR supports deploying and managing the PD Discovery component, which is fully capable of replacing tidb-cluster chart to manage the TiDB cluster.

- `TidbCluster` CR adds support for Pump, TiFlash, TiCDC, and TiDB Dashboard.

- Add the [Admission Controller](enable-admission-webhook.md) (optional) to improve the user experience of upgrade and scaling, and to provide the canary release feature.

- `tidb-scheduler` supports high availability (HA) scheduling at any dimension and scheduler preemption.

- Support using tikv-importer chart to [deploy and manage tikv-importer](restore-data-using-tidb-lightning.md#deploy-tikv-importer).

## Usability

- Add `TidbMonitor` CR to deploy the cluster monitoring.

- Add `TidbInitializer` CR to initialize the cluster.

- Add `Backup`, `BackupSchedule`, and `Restore` CR to back up and restore the cluster, which supports using Amazon S3 or GCS as the remote storage.

- Support gracefully restart a component in the TiDB cluster.

## Security

- Support configuring TLS certificates for the TiDB cluster components and for MySQL clients.

- Support TiKV data encryption.

## Experimental features

- Add `TidbClusterAutoScaler` to implement [auto-scaling](enable-tidb-cluster-auto-scaling.md). You can enable this feature by turning on the `AutoScaling` switch.

- Add the optional [Advanced StatefulSet Controller](advanced-statefulset.md), which supports deleting a specific Pod. You can enable this feature by turning on the `AdvancedStatefulSet` switch.

For the full release notes, see [1.1 CHANGE LOG](https://github.com/pingcap/tidb-operator/blob/master/CHANGELOG-1.1.md).

To deploy TiDB Operator in Kubernetes, see [Deployment](deploy-tidb-operator.md). For CRD documentation, see [API references](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md).
