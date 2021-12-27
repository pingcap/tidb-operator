---
title: Aggregate Monitoring Data of Multiple TiDB Clusters
summary: Learn how to aggregate monitoring data of multiple TiDB clusters by Thanos query.
---

# Aggregate Monitoring Data of Multiple TiDB Clusters

This document describes how to aggregate the monitoring data of multiple TiDB clusters by Thanos to provide centralized monitoring service.

## Thanos

[Thanos](https://thanos.io/tip/thanos/design.md/) is a high availability solution for Prometheus that simplifies the availability guarantee of Prometheus.

Thanos provides [Thanos Query](https://thanos.io/tip/components/query.md/) component as a unified query solution across multiple Prometheus clusters. You can use this feature to aggregate monitoring data of multiple TiDB clusters.

## Aggregate monitoring data via Thanos Query

### Configure Thanos Query

1. Configure a Thanos Sidecar container for each TidbMonitor.

    A configuration example is as follows.

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl -n ${namespace} apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/tidb-monitor.yaml
    ```

2. Deploy the Thanos Query component.

    1. Download the `thanos-query.yaml` file for Thanos Query deployment:

        {{< copyable "shell-regular" >}}

        ```
        curl -sl -O https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/monitor-with-thanos/thanos-query.yaml
        ```

    2. Manually modify the `--store` parameter in the `thanos-query.yaml` file by updating `basic-prometheus:10901` to `basic-prometheus.${namespace}:10901`.

        `${namespace}` is the namespace where TidbMonitor is deployed.

    3. Execute the `kubectl apply` command for deployment.

        {{< copyable "shell-regular" >}}

        ```
        kubectl -n ${thanos_namespace} apply -f thanos-query.yaml
        ```

       In the command above, `${thanos_namespace}` is the namespace where the Thanos Query component is deployed.

In Thanos Query, a Prometheus instance corresponds to a store and also corresponds to a TidbMonitor. After deploying Thanos Query, you can provide a uniform query interface for monitoring data through Thanos Query's API.

### Access the Thanos Query panel

To access the Thanos Query panel, execute the following command, and then access <http://127.0.0.1:9090> in your browser:

{{< copyable "shell-regular" >}}

```bash
kubectl port-forward -n ${thanos_namespace} svc/thanos-query 9090
```

If you want to access the Thanos Query panel using NodePort or LoadBalancer, refer to the following documents:

- [NodePort method](access-tidb.md#nodeport)
- [LoadBalancer way](access-tidb.md#loadbalancer)

### Configure Grafana

After deploying Thanos Query, to query the monitoring data of multiple TidbMonitors, take the following steps:

1. Log in to Grafana.
2. In the left navigation bar, select `Configuration` > `Data Sources`.
3. Add or modify a DataSource in the Prometheus type.
4. Set the URL under HTTP to `http://thanos-query.${thanos_namespace}:9090`.

### Add or remove TidbMonitor

In Thanos Query, a Prometheus instance corresponds to a monitor store and also corresponds to a TidbMonitor. If you need to add, update, or remove a monitor store from the Thanos Query, update the `--store` configuration of the Thanos Query component, and perform a rolling update to the Thanos Query component.

```yaml
spec:
 containers:
   - args:
       - query
       - --grpc-address=0.0.0.0:10901
       - --http-address=0.0.0.0:9090
       - --log.level=debug
       - --log.format=logfmt
       - --query.replica-label=prometheus_replica
       - --query.replica-label=rule_replica
       - --store=<TidbMonitorName1>-prometheus.<TidbMonitorNs1>:10901
       - --store=<TidbMonitorName2>-prometheus.<TidbMonitorNs2>:10901
```

### Configure archives and storage of Thanos Sidecar

> **Note:**
>
> To ensure successful configuration, you must first create the S3 bucket. If you choose AWS S3, refer to [AWS documentation - Create AWS S3 Bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html) and [AWS documentation - AWS S3 Endpoint List](https://docs.aws.amazon.com/general/latest/gr/s3.html) for instructions.

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

## RemoteWrite mode

Besides aggregating data via Thanos Query, you can also push monitoring data to Thanos using Prometheus' RemoteWrite feature.

To enable the RemoteWrite mode, specify the Prometheus RemoteWrite configuration when you create the TidbMonitor CR. For example:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
  - name: basic
  prometheus:
    baseImage: prom/prometheus
    version: v2.27.1
    remoteWrite:
      - url: "http://thanos-receiver:19291/api/v1/receive"
  grafana:
    baseImage: grafana/grafana
    version: 7.5.11
  initializer:
    baseImage: registry.cn-beijing.aliyuncs.com/tidb/tidb-monitor-initializer
    version: v5.2.1
  reloader:
    baseImage: registry.cn-beijing.aliyuncs.com/tidb/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

After RemoteWrite is enabled, Prometheus pushes the monitoring data to [Thanos Receiver](https://thanos.io/tip/components/receive.md/). For more information, refer to [the design of Thanos Receiver](https://thanos.io/v0.8/proposals/201812_thanos-remote-receive/).

For details on the deployment, refer to [this example of integrating TidbMonitor with Thanos Receiver](https://github.com/pingcap/tidb-operator/tree/master/examples/monitor-prom-remotewrite).
