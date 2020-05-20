# A example describe monitoring with external configMap

> **Note:**
>
> This setup is for teach user how to monitor with an external configMap

## Install

The following commands is assumed to be executed in this directory.

Create external configMap:

```bash
> kubectl apply -f external-configMap.yaml
```

Install tidb monitor:

```bash
kubectl apply -f tidb-monitor.yaml
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```


Explore the monitoring dashboards:

```bash
> kubectl -n <namespace> port-forward svc/basic-prometheus  9090:9090 &>/tmp/pf-prometheus.log &
```

Browse [localhost:9090](http://localhost:9090), check rules configuration.
