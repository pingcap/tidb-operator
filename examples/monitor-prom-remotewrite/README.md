# TidbMonitor with Thanos Receiver

This document is to show how to integrate TidbMonitor with [Thanos Receiver](https://thanos.io/tip/components/receive.md/).

## Install Thanos Receiver and Thanos Query

Install Thanos receiver to receive data from the TidbMonitor Prometheus:

```bash
> kubectl -n <namespace> apply -f thanos-receiver.yaml
```

Install thanos query to integrate receiver:

```bash
> kubectl -n <namespace> apply -f thanos-query.yaml
```

Explore the thanos query dashboards:

```bash
> kubectl port-forward svc/thanos-query 9090:9090
```

Browse [localhost:9090](http://localhost:9090).

## Install TidbMonitor

The following commands is assumed to be executed in this directory.

Install the monitor with remotewrite configuration:

```bash
> kubectl -n <namespace> apply -f tidb-monitor.yaml
```

Wait for the monitor Pod ready:

```bash
watch kubectl -n <namespace> get pod
```

Then we can view metric data by thanos query ui.

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
