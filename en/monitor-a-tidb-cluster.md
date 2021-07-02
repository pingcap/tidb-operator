---
title: Deploy Monitoring and Alerts for a TiDB Cluster
summary: Learn how to monitor a TiDB cluster in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/monitor-a-tidb-cluster/','/docs/tidb-in-kubernetes/dev/monitor-using-tidbmonitor/','/tidb-in-kubernetes/dev/monitor-using-tidbmonitor/']
---

# Deploy Monitoring and Alerts for a TiDB Cluster

This document describes how to monitor a TiDB cluster deployed using TiDB Operator and configure alerts for the cluster.

## Monitor the TiDB cluster

You can monitor the TiDB cluster with Prometheus and Grafana. When you create a new TiDB cluster using TiDB Operator, you can deploy a separate monitoring system for the TiDB cluster. The monitoring system must run in the same namespace as the TiDB cluster, and includes two components: Prometheus and Grafana.

For configuration details on the monitoring system, refer to [TiDB Cluster Monitoring](https://pingcap.com/docs/stable/how-to/monitor/monitor-a-cluster).

In TiDB Operator v1.1 or later versions, you can monitor a TiDB cluster on a Kubernetes cluster by using a simple Custom Resource (CR) file called `TidbMonitor`.

> **Note:**
>
> * One `TidbMonitor` can only monitor one `TidbCluster`.
> * `spec.clusters[0].name` should be set to the `TidbCluster` name of the corresponding TiDB cluster.

### Persist monitoring data

The monitoring data is not persisted by default. To persist the monitoring data, you can set `spec.persistent` to `true` in `TidbMonitor`. When you enable this option, you need to set `spec.storageClassName` to an existing storage in the current cluster. This storage must support persisting data; otherwise, there is a risk of data loss.

A configuration example is as follows:

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
    version: v2.18.1
    service:
      type: NodePort
  grafana:
    baseImage: grafana/grafana
    version: 6.1.6
    service:
      type: NodePort
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.1.0
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

To verify the PVC status, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/component=monitor -n ${namespace}
```

```
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
basic-monitor   Bound    pvc-6db79253-cc9e-4730-bbba-ba987c29db6f   5G         RWO            standard       51s
```

### Customize the Prometheus configuration

You can customize the Prometheus configuration by using a customized configuration file or by adding extra options to the command.

#### Use a customized configuration file

1. Create a ConfigMap for your customized configuration, and set the key name of `data` to `prometheus-config`.
2. Set `spec.prometheus.config.configMapRef.name` and `spec.prometheus.config.configMapRef.namespace` to the name and namespace of the customized ConfigMap respectively.

For the complete configuration, refer to the [tidb-operator example](https://github.com/pingcap/tidb-operator/blob/master/examples/monitor-with-externalConfigMap/README.md).

#### Add extra options to the command

To add extra options to the command that starts Prometheus, configure `spec.prometheus.config.commandOptions`.

For the complete configuration, refer to the [tidb-operator example](https://github.com/pingcap/tidb-operator/blob/master/examples/monitor-with-externalConfigMap/README.md).

> **Note:**
>
> The following options are automatically configured by the TidbMonitor controller and cannot be specified again via `commandOptions`.
>
> - `config.file`
> - `log.level`
> - `web.enable-admin-api`
> - `web.enable-lifecycle`
> - `storage.tsdb.path`
> - `storage.tsdb.retention`
> - `storage.tsdb.max-block-duration`
> - `storage.tsdb.min-block-duration`

### Access the Grafana monitoring dashboard

You can run the `kubectl port-forward` command to access the Grafana monitoring dashboard:

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-grafana 3000:3000 &>/tmp/portforward-grafana.log &
```

Then open [http://localhost:3000](http://localhost:3000) in your browser and log on with the default username and password `admin`.

You can also set `spec.grafana.service.type` to `NodePort` or `LoadBalancer`, and then view the monitoring dashboard through `NodePort` or `LoadBalancer`.

If there is no need to use Grafana, you can delete the part of `spec.grafana` in `TidbMonitor` during deployment. In this case, you need to use other existing or newly deployed data visualization tools to directly access the monitoring data.

### Access the Prometheus monitoring data

To access the monitoring data directly, run the `kubectl port-forward` command to access Prometheus:

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-prometheus 9090:9090 &>/tmp/portforward-prometheus.log &
```

Then open [http://localhost:9090](http://localhost:9090) in your browser or access this address via a client tool.

You can also set `spec.prometheus.service.type` to `NodePort` or `LoadBalancer`, and then view the monitoring data through `NodePort` or `LoadBalancer`.

### Set kube-prometheus and AlertManager

In some cases, TidbMonitor needs to obtain the monitoring metrics on Kubernetes. To obtain the [kube-prometheus](https://github.com/coreos/kube-prometheus) metrics, configure `TidbMonitor.Spec.kubePrometheusURL`.

Similarly, you can configure TidbMonitor to push the monitoring alert to [AlertManager](https://prometheus.io/docs/alerting/alertmanager/).

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
    - name: basic
  kubePrometheusURL: http://prometheus-k8s.monitoring:9090
  alertmanagerURL: alertmanager-main.monitoring:9093
  prometheus:
    baseImage: prom/prometheus
    version: v2.18.1
    service:
      type: NodePort
  grafana:
    baseImage: grafana/grafana
    version: 6.1.6
    service:
      type: NodePort
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.1.0
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

## Enable Ingress

This section introduces how to enable Ingress for TidbMonitor. [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) is an API object that exposes HTTP and HTTPS routes from outside the cluster to services within the cluster.

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
    version: v2.18.1
    ingress:
      hosts:
      - example.com
      annotations:
        foo: "bar"
  grafana:
    baseImage: grafana/grafana
    version: 6.1.6
    service:
      type: ClusterIP
    ingress:
      hosts:
        - example.com
      annotations:
        foo: "bar"
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.1.0
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
    version: v2.18.1
    ingress:
      hosts:
      - example.com
      tls:
      - hosts:
        - example.com
        secretName: testsecret-tls
  grafana:
    baseImage: grafana/grafana
    version: 6.1.6
    service:
      type: ClusterIP
  initializer:
    baseImage: pingcap/tidb-monitor-initializer
    version: v5.1.0
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

In a public cloud-deployed Kubernetes cluster, you can usually [configure Loadbalancer](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/) to access Ingress through a domain name. If you cannot configure the Loadbalancer service (for example, when you use NodePort as the service type of Ingress), you can access the service in a way equivalent to the following command:

```shell
curl -H "Host: example.com" ${node_ip}:${NodePort}
```

## Configure alert

When Prometheus is deployed with a TiDB cluster, some default alert rules are automatically imported. You can view all alert rules and statuses in the current system by accessing the Alerts page of Prometheus through a browser.

The custom configuration of alert rules is supported. You can modify the alert rules by taking the following steps:

1. When deploying the monitoring system for the TiDB cluster, set `spec.reloader.service.type` to `NodePort` or `LoadBalancer`.
2. Access the `reloader` service through `NodePort` or `LoadBalancer`. Click the `Files` button above to select the alert rule file to be modified, and make the custom configuration. Click `Save` after the modification.

The default Prometheus and alert configuration do not support sending alert messages. To send an alert message, you can integrate Prometheus with any tool that supports Prometheus alerts. It is recommended to manage and send alert messages via [AlertManager](https://prometheus.io/docs/alerting/alertmanager/).

- If you already have an available AlertManager service in your existing infrastructure, you can set the value of `spec.alertmanagerURL` to the address of `AlertManager`, which will be used by Prometheus. For details, refer to [Set kube-prometheus and AlertManager](#set-kube-prometheus-and-alertmanager).

- If no AlertManager service is available, or if you want to deploy a separate AlertManager service, refer to the [Prometheus official document](https://github.com/prometheus/alertmanager).
