# A Basic DM cluster

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.

The following steps will create a DM cluster.

**Prerequisites**: 
- Has TiDB operator `v1.2.0` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
- Has default `StorageClass` configured, and there are enough PVs (by default, 2 PVs are required) of that storageClass:
  
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
  
  Alternatively, you could specify the storageClass explicitly by modifying `dm-cluster.yaml`.

## Install

The following commands is assumed to be executed in this directory.

Install the cluster:

```bash
> kubectl -n <namespace> apply -f ./
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Explore

Explore the DM master interface:

```bash
> kubectl exec -ti -n <namespace> <cluster_name>-dm-master-0 -- /bin/sh
> /dmctl --master-addr 127.0.0.1:8261 list-member
```

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```

The PVCs used by DM cluster will not be deleted in the above process, therefore, the PVs will be not be released neither. You can delete PVCs and release the PVs by the following command:

```bash
> kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator
```

