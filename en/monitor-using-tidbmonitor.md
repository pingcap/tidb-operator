---
title: Monitor a TiDB Cluster Using TidbMonitor
summary: This document introduces how to monitor a TiDB cluster using TidbMonitor.
aliases: ['/docs/tidb-in-kubernetes/dev/monitor-using-tidbmonitor/']
---

# Monitor a TiDB Cluster Using TidbMonitor

In TiDB Operator v1.1 or later versions, you can monitor a TiDB cluster on a Kubernetes cluster by using a simple Custom Resource (CR) file called TidbMonitor.

## Quick start

### Prerequisites

- TiDB Operator **v1.1.0** (or later versions) is installed, and the related Custom Resource Definition (CRD) file is updated.

- The default storageClass is set, which has enough Persistent Volumes (PV). 6 PVs are needed by default. You can check this setting by executing the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get storageClass
    ```

    ```shell
    NAME                 PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
    standard (default)   rancher.io/local-path   Delete          WaitForFirstConsumer   false                  14h
    ```

### Install

To deploy a TidbMonitor component, save the following content as a `.yaml` file and execute the `kubectl apply -f` command.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
  - name: basic
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
    service:
      type: NodePort
  grafana:
    baseImage: grafana/grafana
    version: 6.0.1
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v4.0.4
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

> **Note:**
>
> * One TidbMonitor can monitor only one TidbCluster.
> * `spec.clusters[0].name` needs to be set to the TidbCluster name of your TiDB Cluster.

You can also quickly deploy TidbMonitor using the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n ${namespace}
```

If the server does not have an external network, refer to [Deploy the TiDB cluster](deploy-on-general-kubernetes.md#deploy-the-tidb-cluster) to download the required Docker image on the machine with an external network and upload it to the server.

After the deployment is finished, you can check whether TidbMonitor is started by executing the `kubectl get pod` command:

{{< copyable "shell-regular" >}}

```shell
kubectl get pod -l app.kubernetes.io/instance=basic -n ${namespace} | grep monitor
```

```
basic-monitor-85fcf66bc4-cwpcn     3/3     Running   0          117s
```

### View the monitoring dashboard

To view the monitoring dashboard, execute the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} port-forward svc/basic-grafana 3000:3000 &>/tmp/pf-grafana.log &
```

Visit `localhost:3000` to access the dashboard.

### Delete the monitor

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n ${namespace}
```

## Persist the monitoring data

If you want to persist the monitoring data in TidbMonitor, enable the `persist` option in TidbMonitor:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
    - name: basic
  persistent: true
  storageClassName: ${storageClassName}
  storage: 5G
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
    service:
      type: NodePort
  grafana:
    baseImage: grafana/grafana
    version: 6.0.1
    service:
      type: NodePort
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v4.0.4
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

To verify the status of Persistent Volume Claim (PVC), execute the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/component=monitor -n ${namespace}
```

```
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
basic-monitor   Bound    pvc-6db79253-cc9e-4730-bbba-ba987c29db6f   5G         RWO            standard       51s
```

### Set kube-prometheus and AlertManager

In some cases, TidbMonitor needs to obtain the monitoring metrics on Kubernetes. To obtain the kube-prometheus metrics, configure `TidbMonitor.Spec.kubePrometheusURL`. For details, see [kube-prometheus](https://github.com/coreos/kube-prometheus).

Similarly, you can configure TidbMonitor to push the monitoring alert to AlertManager. For details, see [AlertManager](https://prometheus.io/docs/alerting/alertmanager/).

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
    - name: basic
  kubePrometheusURL: "your-kube-prometheus-url"
  alertmanagerURL: "your-alert-manager-url"
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
    service:
      type: NodePort
  grafana:
    baseImage: grafana/grafana
    version: 6.0.1
    service:
      type: NodePort
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v4.0.4
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

## Enable Ingress

This section introduces how to enable Ingress for TidbMonitor. Ingress is an API object that exposes HTTP and HTTPS routes from outside the cluster to services within the cluster.

### Prerequisites

Before using Ingress, you need to install the Ingress controller. Simply creating the Ingress resource does not take effect.

You need to deploy the [NGINX Ingress controller](https://kubernetes.github.io/ingress-nginx/deploy/), or choose from various [Ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/).

For more information, see [Ingress Prerequisites](https://kubernetes.io/docs/concepts/services-networking/ingress/#prerequisites).

### Access TidbMonitor using Ingress

Currently, TidbMonitor provides a method to expose the Prometheus/Grafana service through Ingress. For details about Ingress, see [Ingress official documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/).

The following example shows how to enable Prometheus and Grafana in TidbMonitor:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: ingress-demo
spec:
  clusters:
    - name: demo
  persistent: false
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
    ingress:
      hosts:
      - exmaple.com
      annotations:
        foo: "bar"
  grafana:
    baseImage: grafana/grafana
    version: 6.0.1
    service:
      type: ClusterIP
    ingress:
      hosts:
        - exmaple.com
      annotations:
        foo: "bar"
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v4.0.4
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

To modify the setting of Ingress Annotations, configure `spec.prometheus.ingress.annotations` and `spec.grafana.ingress.annotations`. If you use the default Nginx Ingress, see [Nginx Ingress Controller Annotation](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/) for details.

The TidbMonitor Ingress setting also supports TLS. The following example shows how to configure TLS for Ingress. See [Ingress TLS](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls) for details.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: ingress-demo
spec:
  clusters:
    - name: demo
  persistent: false
  prometheus:
    baseImage: prom/prometheus
    version: v2.11.1
    ingress:
      hosts:
      - exmaple.com
      tls:
      - hosts:
        - exmaple.com
        secretName: testsecret-tls
  grafana:
    baseImage: grafana/grafana
    version: 6.0.1
    service:
      type: ClusterIP
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v4.0.4
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

TLS Secret must include the `tls.crt` and `tls.key` keys, which include the certificate and private key used for TLS. For example:

 ```yaml
apiVersion: v1
kind: Secret
metadata:
  name: testsecret-tls
  namespace: ${namespace}
data:
  tls.crt: base64 encoded cert
  tls.key: base64 encoded key
type: kubernetes.io/tls
 ```

## References

For more detailed API information of TidbMonitor, see [TiDB Operator API documentation](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md).
