---
title: What's New in TiDB Operator 1.2
---

# What's New in TiDB Operator 1.2

TiDB Operator 1.2 introduces the following key features, which helps you manage TiDB clusters and the tools more easily in terms of extensibility, usability, and security.

## Extensibility

- Support customizing labels and annotations for Pods and services in TidbCluster
- Support configuring podSecurityContext and topologySpreadConstraints for all TiDB components
- Support full lifecycle management of [Pump](https://docs.pingcap.com/tidb/stable/tidb-binlog-overview#pump)
- TidbMonitor supports [monitoring multiple TidbClusters](monitor-a-tidb-cluster.md#monitor-multiple-clusters)
- TidbMonitor supports remotewrite
- TidbMonitor supports [configuring Thanos sidecar](aggregate-multiple-cluster-monitor-data.md)
- TidbMonitor management resource changes from Deployment to StatefulSet to provide more flexibility
- Support [deploying TiDB Operator with namespace permissions only](deploy-tidb-operator.md#customize-tidb-operator-deployment)
- Support [deploying multiple sets of TiDB Operator](deploy-multiple-tidb-operator.md) to manage different TiDB clusters separately

## Usability

- Add [DMCluster CR to manage DM 2.0](deploy-tidb-dm.md)
- Support customizing store labels for TiKV and TiFlash
- Support [customizing storage for TiDB slow log](configure-a-tidb-cluster.md#configure-pv-for-tidb-slow-logs)
- TiDB Lightning chart supports [configuring local-backend and persisting the checkpoint information](restore-data-using-tidb-lightning.md)

## Security

- Support [configuring TLS for the TiDB Lightning chart and TiKV Importer chart](enable-tls-between-components.md)

## Experimental features

- Support [deploying a TiDB cluster across multiple Kubernetes clusters](deploy-tidb-cluster-across-multiple-kubernetes.md)
