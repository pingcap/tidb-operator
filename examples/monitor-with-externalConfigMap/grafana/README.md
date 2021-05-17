# An example describes how to mount a custom dashboard JSON file for Grafana.

> **Note:**
>
> This document is to show how to mount a custom dashboard for Grafana in TidbMonitor. 
> This would help to customize the user's Grafana dashboards.

## Install

The following commands are assumed to be executed in this directory.

Create custom dashboard JSON with configMap:

```bash
kubectl apply -f custom-dashboard.yaml -n ${namespace}
```

Install TidbMonitor:

```bash
kubectl apply -f tidb-monitor.yaml -n ${namespace}
```

Wait for Pods ready:

```bash
watch kubectl -n ${namespace} get pod
```

Explore the monitoring dashboards:

```bash
kubectl -n  ${namespace} port-forward svc/basic-grafana  3000:3000
```

Browse [localhost:3000](http://localhost:3000), and check the custom dashboard.

## Uninstall

Uninstall TidbMonitor:

```bash
kubectl delete -f tidb-monitor.yaml -n ${namespace}
```

Delete the external configMap:

```bash
kubectl delete -f custom-dashboard.yaml -n ${namespace}
```
