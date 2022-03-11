# TidbMonitor with Thanos

This document is to show how to integrate TidbMonitor with [Thanos](https://thanos.io/tip/thanos/design.md/).

## Install TidbMonitor

The following commands is assumed to be executed in this directory.

Install the monitor with thanos sidecar:

```bash
> kubectl -n <namespace> apply -f tidb-monitor.yaml
```

Wait for the monitor Pod ready:

```bash
watch kubectl -n <namespace> get pod
```

If you need to store historical data, you can create the corresponding secret and configure the `objectStorageConfig` field:

```bash
> kubectl -n <namespace> apply -f objectstorage-secret.yaml
```

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
  - name: basic
  thanos:
    baseImage: thanosio/thanos
    version: v0.17.2
    objectStorageConfig:
      key: objectstorage.yaml
      name: thanos-objectstorage
```

Of course, you can also not configure it.

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
