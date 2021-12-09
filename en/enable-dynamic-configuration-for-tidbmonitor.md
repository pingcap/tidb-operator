---
title: Enable Dynamic Configuration for TidbMonitor
summary: This document describes how to enable dynamic configuration for TidbMonitor.
---

# Enable Dynamic Configuration for TidbMonitor

This document describes how to enable dynamic configuration for TidbMonitor.

TidbMonitor supports monitoring across multiple clusters and shards. Without dynamic configuration, when the Prometheus configurations, rules, or targets are changed, such changes only take effect after a restart. If you are monitoring a large dataset, after the restart, it might take a long time to recover the Prometheus snapshot data.

With dynamic configuration enabled, any configuration change of TidbMonitor takes effect immediately.

## Enable the dynamic configuration feature

To enable the dynamic configuration feature, configure `prometheusReloader` in the `spec` field of TidbMonitor. For example:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: monitor
spec:
  clusterScoped: true
  clusters:
    - name: ns1
      namespace: ns1
    - name: ns2
      namespace: ns2
  prometheusReloader:
    baseImage: quay.io/prometheus-operator/prometheus-config-reloader
    version: v0.49.0
  imagePullPolicy: IfNotPresent
```

After you modify the `prometheusReloader` configuration, TidbMonitor restarts automatically. After the restart, the dynamic configuration feature is enabled. All configuration changes related to Prometheus are dynamically updated.

For more examples, refer to [monitor-dynamic-configmap](https://github.com/pingcap/tidb-operator/tree/master/examples/monitor-dynamic-configmap).

## Disable the dynamic configuration feature

To disable the dynamic configuration feature, remove the `prometheusReloader` configuration from the `spec` field of TidbMonitor.
