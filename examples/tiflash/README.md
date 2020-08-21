# A Basic TiDB cluster with TiFlash and Monitoring

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.

The following steps will create a TiDB cluster with TiFlash deployed and monitoring.

**Prerequisites**: 
- TiDB operator `v1.1.0-rc.3` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
- Available `StorageClass` configured, and there are enough PVs (by default, 9 PVs are required) of that storageClass:
  
  The available `StorageClass` can by checked with the following command:
  
  ```bash
  > kubectl get storageclass
  ```
  
  The output is similar to the following:
  
  ```bash
  NAME                 PROVISIONER               AGE
  standard (default)   kubernetes.io/gce-pd      1d
  gold                 kubernetes.io/gce-pd      1d
  local-storage   kubernetes.io/no-provisioner   189d
  ```
  
  The default storageClassName in `tidb-cluster.yaml` and `tidb-monitor.yaml` is set to `local-storage`, please update them to your available storageClass.

## Install

The following commands is assumed to be executed in this directory.

Install the cluster:

```bash
> kubectl create ns <namespace>
> kubectl -n <namespace> apply -f ./
```

Wait for cluster Pods ready:

```bash
> watch kubectl -n <namespace> get pod
```

## Explore

Explore the TiDB SQL interface:

```bash
> kubectl -n <namespace> port-forward svc/demo-tidb 4000:4000 &>/tmp/pf-tidb.log &
> mysql -h 127.0.0.1 -P 4000 -u root
```
Refer to the [doc](https://pingcap.com/docs/stable/reference/tiflash/use-tiflash/) to try TiFlash.

Explore the monitoring dashboards:

```bash
> kubectl -n <namespace> port-forward svc/demo-grafana 3000:3000 &>/tmp/pf-grafana.log &
```

Browse [localhost:3000](http://localhost:3000).

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```

The PVCs used by the TiDB cluster will not be deleted in the above command, therefore, the PVs will be not be released either. You can delete PVCs and release the PVs with the following command:

```bash
> kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=demo,app.kubernetes.io/managed-by=tidb-operator
```

