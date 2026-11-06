# TiBR API access

<!-- toc -->
<!-- /toc -->

The generated `tikv-worker` API binds to `127.0.0.1:19500` by default. A Pod-network client can no longer invoke the default backup/restore listener directly. This is independent of `Cluster.spec.tlsCluster`: the TLS flags for component connections are not evidence that every API operation is authenticated by the external binary.

An administrator can access the API through Kubernetes authenticated and authorized port forwarding:

```sh
kubectl -n <namespace> port-forward pod/<tibr-statefulset>-0 19500:19500
```

The local client then uses `127.0.0.1:19500`. Restrict `pods/portforward` and Pod exec/debug permissions to trusted backup administrators. Keep the port-forward listener on loopback. Containers in the same Pod share its network namespace and remain inside this trust boundary.

On upgrade, the Operator changes existing owned StatefulSet templates using the old default `0.0.0.0:19500` to loopback and waits for the rollout before reporting synchronization. It preserves other Pod fields. A normal RollingUpdate strategy restarts the API Pod and may interrupt requests; an explicit OnDelete strategy requires the administrator to restart the Pod to complete the rollout. Check TiBR status and the running Pod before treating the old listener as removed.

A Pod overlay that explicitly replaces the API container command remains an administrator override. Exposing a listener that way requires an authenticated gateway and appropriate network isolation; do not infer authentication from the presence of component TLS flags. Existing remote clients need the port-forward flow or a separately secured gateway. This patch does not add authentication behavior to the external `tikv-worker` binary.
