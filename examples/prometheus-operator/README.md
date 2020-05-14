# How to monitor TiDB using prometheus-operator

> **Note:**
>
> this example describe how to monitor TidbCluster with existing Prometheus managed by [prometheus-operator](https://github.com/coreos/prometheus-operator) in the Kubernetes
> this case is only used to demo and not suggest to use in the prod env.

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

Then we can go to prometheus homepage to see if it works, like http://<IP>:9090/rules and http://<IP>:9090/targets URL.

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
