# The extra TiDB cluster Components

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.


The following steps will create a TiDB cluster with monitoring and auto-scaler, the monitoring data is not persisted by default.

**Prerequisites**: 
- Has TiDB operator `v1.1.0-beta.2` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
 
   
## Auto-scaling

> **Note:**
>
> The Auto-scaling feature is still in alpha, you should enable this feature in TiDB Operator by setting values.yaml:
 ```yaml
features:
  AutoScaling=true
```

Auto-scale the cluster based on CPU load
```bash
> kubectl -n <namespace> apply -f ./
```

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
