# A Basic heterogeneous cluster example

The following steps will create a basic TiDB cluster , then we can create a heterogeneous cluster contains one tikv and one tidb.

## Install basic cluster

The following commands is assumed to be executed in this directory.

Install the basic cluster:

```bash
> kubectl -n <namespace> apply -f baisc/tidb-cluster.yaml
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Install heterogeneous cluster

The following commands is assumed to be executed in this directory.

Install the heterogeneous cluster:

```bash
> kubectl -n <namespace> apply -f heterogeneous/heterogeneous-cluster.yaml
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Aggregate basic and heterogeneous cluster monitor data by thanos query
The following commands is assumed to be executed in this directory.

Install basic and heterogeneous cluster monitor external configuration,
because we must to add `extra_labels` configuration due to thanos sidecar reuqirements.

```bash
> kubectl -n <namespace> apply -f basic/basic-monitor-external-config.yaml
> kubectl -n <namespace> apply -f heterogeneous/basic-monitor-external-config.yaml
```

Install the basic and heterogeneous cluster monitor and add a thanos query sidecar:

```bash
> kubectl -n <namespace> apply -f basic/tidb-monitor.yaml
> kubectl -n <namespace> apply -f heterogeneous/tidb-monitor.yaml
```

Wait for monitor Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

Install the thanos sidecar servcie for basic and heterogeneous cluster monitor:

```bash
> kubectl -n <namespace> apply -f basic/basic-sidecar-service.yaml
> kubectl -n <namespace> apply -f heterogeneous/heterogeneous-sidecar-service
```

Install the thanos query with basic and heterogeneous stores:
```bash
> kubectl -n <namespace> apply -f thanos-query.yaml
> kubectl -n <namespace> apply -f thanos-query-service.yaml
```

Wait for thanos Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

Access thanos query ui by thanos query ip and port:

```bash
> kubectl port-forward service/thanos-query  <port>:9090
```



## Destroy

```bash
> kubectl -n <namespace> delete -f ./
> kubectl -n <namespace> delete -f ./basic/
> kubectl -n <namespace> delete -f ./heterogeneous/
```

The PVCs used by TiDB cluster will not be deleted in the above process, therefore, the PVs will not be released either. You can delete PVCs and release the PVs by the following command:
```bash
> kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator
```

