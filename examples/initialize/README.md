# Creating TidbCluster with Initialization

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.


The following steps will create a TiDB cluster with Initialization.

**Prerequisites**: 
- Has TiDB operator `v1.1.0-beta.1` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
- Has default `StorageClass` configured, and there are enough PVs (by default, 6 PVs are required) of that storageClass:
  
  This could by verified by the following command:
  
  ```bash
  > kubectl get storageclass
  ```
  
  The output is similar to this:
  
  ```bash
  NAME                 PROVISIONER               AGE
  standard (default)   kubernetes.io/gce-pd      1d
  gold                 kubernetes.io/gce-pd      1d
  ```
  
  Alternatively, you could specify the storageClass explicitly by modifying `tidb-cluster.yaml`.
 
  
## Initialize


> **Note:**
>
> The Initialization should be done once the TiDB Cluster was created 

The following commands is assumed to be executed in this directory.

You can create the root user and set its password by creating secret and link it to the Initializer:

```bash
> kubectl create secret generic tidb-secret --from-literal=root=<root-password> --namespace=<namespace>
```

You can also create other users and set their password:
```bash
> kubectl create secret generic tidb-secret --from-literal=root=<root-password> --from-literal=developer=<developer-passowrd> --namespace=<namespace>
```

Initialize the cluster to create the users and create the database named `hello`:

```bash
> kubectl -n <namespace> apply -f ./
```

Wait for Initialize job done:
```bash
$ kubectl get pod -n <namespace>| grep initialize-demo-tidb-initializer
initialize-demo-tidb-initializer-whzn7               0/1     Completed   0          57s
```

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
