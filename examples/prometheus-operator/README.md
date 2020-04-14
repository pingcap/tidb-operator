# How to monitor TiDB using prometheus-operator

> **Note:**
>
> This example describe ServiceMonitor and PrometheusRule yaml demo , you can use existing Prometheus CRD instance to load .

**Prerequisites**: 
- Has Prometheus operator installed. [Doc](https://github.com/coreos/kube-prometheus)

  This could by verified by the following command:
  
  ```bash
  >  kubectl get crd |grep coreos.com
  ```
  
  The output is similar to this:
  
  ```bash
        NAME                                       TIME
    alertmanagers.monitoring.coreos.com            2020-04-12T14:22:49Z
    podmonitors.monitoring.coreos.com              2020-04-12T14:22:50Z
    prometheuses.monitoring.coreos.com             2020-04-12T14:22:50Z
    prometheusrules.monitoring.coreos.com          2020-04-12T14:22:50Z
    servicemonitors.monitoring.coreos.com          2020-04-12T14:22:51Z
  ```
  
## Initialize

Initialize serviceMonitor with label ` app: tidb cluster: basic` and prometheus rules.

```bash
> kubectl -n <namespace> apply -f ./
```

Wait for Initialize job done:
```bash
$ kubectl get serviceMonitor -n <namespace>| grep basic
NAME         AGE
basic-pd     25h
basic-tidb   24h
basic-tikv   24h
```

```bash
$ kubectl get PrometheusRule -n <namespace>|grep basic
NAME         AGE
basic-prometheus-rules   23h
```


## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
