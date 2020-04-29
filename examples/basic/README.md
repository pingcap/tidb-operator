# A Basic TiDB cluster with monitoring

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.

The following steps will create a TiDB cluster with monitoring, the monitoring data is not persisted by default.

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

Explore the TiDB sql interface:

```bash
> kubectl -n <namespace> port-forward svc/basic-tidb 4000:4000 &>/tmp/pf-tidb.log &
> mysql -h 127.0.0.1 -P 4000 -u root
```

Explore the monitoring dashboards:

```bash
> kubectl -n <namespace> port-forward svc/basic-grafana 3000:3000 &>/tmp/pf-grafana.log &
```

Browse [localhost:3000](http://localhost:3000).

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```

The PVCs used by TiDB cluster will not be deleted in the above process, therefore, the PVs will be not be released neither. You can delete PVCs and release the PVs by the following command:

```bash
> kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator
```

