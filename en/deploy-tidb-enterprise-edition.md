---
title: Deploy TiDB Enterprise Edition in Kubernetes
summary: Learn how to deploy TiDB Enterprise Edition in Kubernetes.
---

# Deploy TiDB Enterprise Edition in Kubernetes

This document describes how to deploy TiDB Enterprise Edition and related tools in Kubernetes. TiDB Enterprise Edition has the following features:

* Enterprise-level best practices
* Enterprise-level service support
* Fully-enhanced security features

## Prerequisites

* TiDB Operator is [deployed](deploy-tidb-operator.md).

## Deploy the Enterprise Edition

Currently, the difference between the deployment of TiDB Operator Enterprise Edition and Community Edition mainly lies in image naming. Compared with that of Community Edition, the image of Enterprise Edition has an extra `-enterprise` suffix.

```yaml
spec:
  version: v5.0.1
  ...
  pd:
    baseImage: pingcap/pd-enterprise
  ...
  tikv:
    baseImage: pingcap/tikv-enterprise
  ...
  tidb:
    baseImage: pingcap/tidb-enterprise
  ...
  tiflash:
    baseImage: pingcap/tiflash-enterprise
  ...
  pump:
    baseImage: pingcap/tidb-binlog-enterprise
  ...
  ticdc:
    baseImage: pingcap/ticdc-enterprise
```

+ If you are deploying a new cluster:

    Refer to [Configure a TiDB Cluster in Kubernetes](configure-a-tidb-cluster.md) to configure `tidb-cluster.yaml` and Enterprise Edition image as described above, and run the `kubectl apply -f tidb-cluster.yaml -n ${namespace}` command to deploy the TiDB Enterprise Edition cluster and related tools.

+ If you want to switch an existing Community Edition cluster to Enterprise Edition:

    Run the `kubectl edit tc ${name} -n ${namespace}` command to add the suffix "-enterprise" to each component's `baseImage` in the above format, and then update the cluster configuration.

    TiDB Operator will automatically update the cluster image to the enterprise image through a rolling upgrade.

## Switch back to the Community Edition

```yaml
spec:
  version: v5.0.1
  ...
  pd:
    baseImage: pingcap/pd
  ...
  tikv:
    baseImage: pingcap/tikv
  ...
  tidb:
    baseImage: pingcap/tidb
  ...
  tiflash:
    baseImage: pingcap/tiflash
  ...
  pump:
    baseImage: pingcap/tidb-binlog
  ...
  ticdc:
    baseImage: pingcap/ticdc
```

If you need to switch an existing Enterprise Edition cluster back to Community Edition:

- Method #1: Remove the "-enterprise" suffix from the `baseImage` item in the configuration file of the existing cluster in the above format, and run the `kubectl apply -f tidb-cluster.yaml -n ${namespace}` command to update the cluster configuration.
- Method #2: Run the `kubectl edit tc ${name} -n ${namespace}` command to remove the suffix "-enterprise" from each component's `baseImage` in the above format, and then update the cluster configuration.

After updating the configuration, TiDB Operator will automatically switch the cluster image to Community Edition image through a rolling upgrade.
