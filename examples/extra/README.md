# The extra TiDB cluster Components

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.


The following steps will create a TiDB cluster with monitoring, the monitoring data is not persisted by default.

**Prerequisites**: 
- Has TiDB operator `v1.1.0-beta.2` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
- Has Tidb Cluster has been installed in `../basic/*`
  

## Initialize


> **Note:**
>
> The Initialization should be done once the TiDB Cluster was created 

The following commands is assumed to be executed in this directory.

Initialize the cluster to create the database

```bash
> kubectl -n <namespace> apply -f ./tidb-initializer.yaml
```

Wait for Initialize job done:


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
> kubectl -n <namespace> apply -f ./tidb-cluster-auto-scaler.yaml
```

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
