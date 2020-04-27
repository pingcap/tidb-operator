# A Basic Tidbcluster with Dashboard Ingress Enabled

> **Note:**
>
> This setup is for test or demo purpose only and **IS NOT** applicable for critical environment. Refer to the [Documents](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/prerequisites/) for production setup.

The following steps will create a TiDB cluster with Dashboard Ingress Enabled.

**Prerequisites**: 
- Has TiDB operator `v1.1.0-rc.3` or higher version installed. [Doc](https://pingcap.com/docs/stable/tidb-in-kubernetes/deploy/tidb-operator/)
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

## Verify

Verify that the Dashboard Ingress has been deployed:
```bash
> kubectl -n <namespace> get ingress dashboard-view-dashboard
```

## Support TLS

To enable TLS, you could set it with following configuration:
```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: dashboard-view
spec:
  dashboard:
    ingress:
      hosts:
        - exmaple.com
        - hello.com
      annotations:
        foo: "bar"
      tls:
        - hosts:
          - sslexample.foo.com
          secretName: testsecret-tls
```

Note that as Ingress assume the TLS termination, if the `spec.TlsCluster.enabled` is true, the Dashboard wouldn't be visited through Ingress.
For more detail about Ingress Tls, see: https://kubernetes.io/docs/concepts/services-networking/ingress/#tls
