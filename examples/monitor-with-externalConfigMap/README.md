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
