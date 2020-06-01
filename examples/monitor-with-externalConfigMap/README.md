# A example describe monitoring with external configMap

> **Note:**
>
> This document is to show how to mount external configmap for prometheus in TidbMonitor. 
> This would help to customize user's own prometheus configurations or rules.

## Install

The following commands is assumed to be executed in this directory.

Create external configMap:

```bash
> kubectl apply -f external-configMap.yaml -n ${namespace}
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
> kubectl -n  ${namespace} port-forward svc/basic-prometheus  9090:9090 &>/tmp/pf-prometheus.log &
```

Browse [localhost:9090](http://localhost:9090), check rules configuration.

## Uninstall


Uninstall tidb monitor:

```bash
kubectl delete -f tidb-monitor.yaml -n ${namespace}
```

Delete external configMap:
```bash
> kubectl delete -f external-configMap.yaml -n ${namespace}
```

