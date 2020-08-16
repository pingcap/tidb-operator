# Add TiDB Monitor to Thanos Query

The following steps will create a basic TiDB cluster , then we can add it to  Thanos Query.

## Install basic cluster

The following commands is assumed to be executed in this directory.

Install the basic cluster:

```bash
> kubectl -n <namespace> apply -f  tidb-cluster.yaml
```

Wait for cluster Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Install basic monitor 

The following commands is assumed to be executed in this directory.

Install the external configuration for monitor:

```bash
> kubectl -n <namespace> apply -f external-configMap.yaml
```

Install the TiDB Monitor with thanos sidecar:

```bash
> kubectl -n <namespace> apply -f tidbmonitor.yaml
```

Wait for monitor Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

## Install thanos query
The following commands is assumed to be executed in this directory.

Install thanos query component

```bash
> kubectl -n <namespace> apply -f thanos-query/thanos-query.yaml
```

Wait for thanos Pods ready:

```bash
watch kubectl -n <namespace> get pod
```

Install thanos query service:

```bash
> kubectl -n <namespace> apply -f thanos-query/thanos-query-service.yaml
```

Access thanos query ui by thanos query service ip and port:

```bash
> kubectl port-forward service/thanos-query  <port>:9090
```

Validate thanos query have basic cluster store info:

```
Browser to http://<ip>:<port>/stores
```


## Destroy

```bash
> kubectl -n <namespace> delete -f ./
> kubectl -n <namespace> delete -f ./thanos-query/
```

The PVCs used by TiDB cluster will not be deleted in the above process, therefore, the PVs will not be released either. You can delete PVCs and release the PVs by the following command:
```bash
> kubectl -n <namespace> delete pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/managed-by=tidb-operator
```

