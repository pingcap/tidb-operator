# TiDBMonitor with Thanos

This document is to show how to integrate TidbMonitor with [Thanos](https://thanos.io/design.md/).


## Install

The following commands is assumed to be executed in this directory.

Install the monitor with thanos sidecar:

```bash
> kubectl -n <namespace> apply -f tidb-monitor.yaml
```

Wait for monitor Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

If you need to store historical data, you can configure the `objectStorageConfig` field and create the corresponding secret.

## Install Thanos

Install thanos query component to integrate tidbmonitor :

```bash
> kubectl -n <namespace> apply -f thanos-query.yaml
```
Explore the thanos query dashboards:

```bash
> kubectl port-forward svc/thanos-query 9090:9090
```

Browse [localhost:9090](http://localhost:9090).

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
