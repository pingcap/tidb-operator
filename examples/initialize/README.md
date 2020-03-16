# The extra TiDB cluster Components

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.


The following steps will create a TiDB cluster with monitoring, the monitoring data is not persisted by default.

**Prerequisites**: 
- Has TiDB operator `v1.1.0-beta.2` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
  
  
## Initialize


> **Note:**
>
> The Initialization should be done once the TiDB Cluster was created 

The following commands is assumed to be executed in this directory.

You can create the root user and set its password by creating secret and link it to the Initializer:

```bash
> kubectl create secret generic tidb-secret --from-literal=root=<root-password> --namespace=<namespace>
```

You can aloso create other users and set their password:
```bash
> kubectl create secret generic tidb-secret --from-literal=root=<root-password> --from-literal=developer=<developer-passowrd> --namespace=<namespace>
```

Initialize the cluster to create the users and create the database named `test`

```bash
> kubectl -n <namespace> apply -f ./
```

Wait for Initialize job done:
```bash
$ kubectl get pod | grep basic-tidb-initializer
initialize-demo-tidb-initializer-whzn7               0/1     Completed   0          57s
```

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
