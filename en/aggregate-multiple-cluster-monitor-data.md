---
title: Aggregate Monitoring Data of Multiple TiDB Clusters
summary: Learn how to aggregate monitoring data of multiple TiDB clusters by Thanos query.
---

# Aggregate Monitoring Data of Multiple TiDB Clusters

This document describes how to aggregate the monitoring data of multiple TiDB clusters by Thanos to provide centralized monitoring service.

## Thanos

[Thanos](https://thanos.io/design.md/) is a high availability solution for Prometheus that simplifies the availability guarantee of Prometheus.

Thanos provides [Thanos Query](https://thanos.io/tip/components/query.md/) component as a unified query solution across multiple Prometheus clusters. You can use this feature to aggregate monitoring data of multiple TiDB clusters.

## Configure Thanos Query

First, deploy the TidbMonitor with Thanos Sidecar container:

{{< copyable "shell-regular" >}}

```
kubectl -n ${namespace} apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/tidb-monitor.yaml
```

Then deploy the Thanos Query component:

{{< copyable "shell-regular" >}}

```
kubectl -n ${namespace} apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/thanos-query.yaml
```

In Thanos Query, a Prometheus corresponds to a Store, which corresponds to a TidbMonitor. After deploying Thanos Query, you can provide a unified query interface for monitoring data through Thanos Query's API.

## Configure Grafana

After you deploy Thanos Query, change Grafana's DataSource into Thanos to query the monitoring data of multiple `TidbMonitor` CRs.

## Add or remove TidbMonitor

If you need to add, update, or remove a monitor store from the Thanos Query, update the `--store` configuration of Thanos Query, and perform a rolling update to the Thanos Query component.

```shell
- query
- --http-address=0.0.0.0:9090
- --store=<store-api>:<grpc-port>
- --store=<store-api2>:<grpc-port>
```

## Configure archives and storage of Thanos Sidecar

Thanos Sidecar supports replicating monitoring data to S3 remote storage.

The configuration of the `TidbMonitor` CR is as follows:

```yaml
spec:
  thanos:
    baseImage: thanosio/thanos
    version: v0.17.2
    objectStorageConfig:
      key: objectstorage.yaml
      name: thanos-objectstorage
```

Meanwhile, you need to create a Secret. The example is as follows:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: thanos-objectstorage
type: Opaque
stringData:
  objectstorage.yaml: |
    type: S3
    config:
      bucket: "xxxxxx"
      endpoint: "xxxx"
      region: ""
      access_key: "xxxx"
      insecure: true
      signature_version2: true
      secret_key: "xxxx"
      put_user_metadata: {}
      http_config:
        idle_conn_timeout: 90s
        response_header_timeout: 2m
      trace:
        enable: true
      part_size: 41943040
```
