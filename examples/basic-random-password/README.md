# A Basic TiDB cluster with random password initialized

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment.

The following steps will create a TiDB cluster with random password initialized.

## Install

The following commands is assumed to be executed in this directory.

Install the cluster:

```bash
kubectl -n <namespace> apply -f ./
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Explore

Get the password from secret:

```bash
kubectl get secret basic-secret -o=jsonpath='{.data.root}' -n <namespace>  | base64 --decode
```

Explore the TiDB SQL interface:

```bash
kubectl -n <namespace> port-forward svc/basic-tidb 4000:4000
```

Test connection successfully:

```bash
mysql -h 127.0.0.1 -P 4000 -u root -p <password> --comments
```

## Destroy

```bash
kubectl -n <namespace> delete -f ./
```

The PVCs used by TiDB cluster will not be deleted in the above process, therefore, the PVs will be not be released neither. You can delete PVCs and release the PVs by the following command:

```bash
kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator
```
