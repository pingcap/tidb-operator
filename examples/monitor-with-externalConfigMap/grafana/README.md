# A example describe grafana mount custom dashboard json file.

> **Note:**
>
> This document is to show how to mount custom dashboard for grafana in TidbMonitor. 
> This would help to customize user's own grafana dashboards or others.

## Install

The following commands is assumed to be executed in this directory.

Create custom dashboard json with configMap:

```bash
> kubectl apply -f custom-dashboard.yaml -n ${namespace}
```

Install tidb monitor:

```bash
kubectl apply -f tidb-monitor.yaml -n ${namespace}
```

Wait for cluster Pods ready:

```bash
watch kubectl -n ${namespace} get pod
```


Explore the monitoring dashboards:

```bash
> kubectl -n  ${namespace} port-forward svc/basic-grafana  3000:3000
```

Browse [localhost:3000](http://localhost:3000), check custom dashboard json file .

## Uninstall


Uninstall tidb monitor:

```bash
kubectl delete -f tidb-monitor.yaml -n ${namespace}
```

Delete external configMap:
```bash
> kubectl delete -f custom-dashboard.yaml -n ${namespace}
```
