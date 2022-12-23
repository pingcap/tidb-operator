# A Basic TiDB cluster with monitoring deployed by a namespaced scoped user

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://docs.pingcap.com/tidb-in-kubernetes/stable/prerequisites/) for production setup.

1. create a namespaced scoped user, and a service account named `namespaced` will be created.

```bash
> kubectl create ns <namespace>
> kubectl -n <namespace> apply -f ./rbac.yaml
```

2. get the kube config for the `namespaced` user.

```bash
# get the user token
> kubectl -n <namespace> get secrets namespaced-token-***** -o "jsonpath={.data.token}" | base64 -d
# get the certificate
> kubectl -n <namespace> get secrets namespaced-token-***** -o "jsonpath={.data['ca\.crt']}"
```

```yaml
# save this block as a kube config file, like `namespaced.config`.
apiVersion: v1

clusters:
- cluster:
    certificate-authority-data: <PLACE CERTIFICATE HERE>
    server: https://<YOUR_KUBERNETES_API_ENDPOINT>
  name: namespaced

users:
- name: namespaced
  user:
    client-key-data: <PLACE CERTIFICATE HERE>
    token: <PLACE USER TOKEN HERE>

contexts:
- context:
    cluster: namespaced
    namespace: namespaced
    user: namespaced
  name: namespaced

current-context: namespaced
```

3. install a namespaced scope tidb-operator.

```bash
> helm template namespaced-operator ../../charts/tidb-operator --set clusterScoped=false --set scheduler.create=false > namespaced-operator.yaml
> KUBECONFIG=namespaced.config kubectl apply -f ./namespaced-operator.yaml
```

4. install TidbCluster and TidbMonitor.

```bash
> KUBECONFIG=namespaced.config kubectl apply -f ./tidb-cluster.yaml
> KUBECONFIG=namespaced.config kubectl apply -f ./tidb-monitor.yaml
```
