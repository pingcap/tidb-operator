# TidbMonitor with Thanos Receiver

This document is to show how to integrate TidbMonitor with [Thanos Receiver](https://thanos.io/tip/components/receive.md/).

## Install TiDB Cluster

```bash
kubectl apply -f tidb-cluster.yaml -n ${namespace}
```

Wait for Pods ready:

```bash
watch kubectl -n ${namespace} get pod
```

## Install TidbMonitor

Install the monitor with remotewrite configuration:

```bash
> kubectl -n <namespace> apply -f tidb-monitor.yaml
```

Wait for the monitor Pod ready:

```bash
watch kubectl -n <namespace> get pod
```

## Install Thanos Receiver and Thanos Query

Install Thanos receiver to receive data from the TidbMonitor Prometheus:

```bash
> kubectl -n <namespace> apply -f thanos-receiver.yaml
```

Install Thanos query to integrate with receiver:

```bash
> kubectl -n <namespace> apply -f thanos-query.yaml
```

Explore the Thanos query dashboards:

```bash
> kubectl -n <namespace> port-forward svc/thanos-query 9090:9090
```

Browse [localhost:9090](http://localhost:9090).

Then we can view the metrics with Thanos query UI.

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
