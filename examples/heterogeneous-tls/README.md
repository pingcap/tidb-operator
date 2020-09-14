# A basic heterogeneous TLS cluster example


The following steps will create a basic tls TiDB cluster , then we can create a heterogeneous cluster contains one tikv and one tidb.

## Install tls certificate
The following commands is assumed to be executed in this directory.

Install the basic certificate:
```bash
> kubectl -n <namespace> apply -f cert-manager/basic/
```

Install the heterogeneous certificate:
```bash
> kubectl -n <namespace> apply -f cert-manager/heterogeneous/
```


## Install basic cluster

The following commands is assumed to be executed in this directory.

Install the basic cluster:

```bash
> kubectl -n <namespace> apply -f tidb-cluster.yaml
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Install heterogeneous cluster

The following commands is assumed to be executed in this directory.

Install the heterogeneous cluster:

```bash
> kubectl -n <namespace> apply -f heterogeneous-cluster.yaml
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```

The PVCs used by TiDB cluster will not be deleted in the above process, therefore, the PVs will not be released either. You can delete PVCs and release the PVs by the following command:
```bash
> kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator
```

