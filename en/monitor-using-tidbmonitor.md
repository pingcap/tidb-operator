---
title: Monitor a TiDB Cluster Using TidbMonitor
summary: This document introduces how to monitor a TiDB cluster using TidbMonitor.
category: how-to
---

# Monitor a TiDB Cluster Using TidbMonitor

In TiDB Operator v1.1 or later versions, you can monitor a TiDB cluster on a Kubernetes cluster by using a simple Custom Resource (CR) file called TidbMonitor.

## Quick start

> **Warning:**
>
> This quick start guide is only used for demonstration or testing purposes. Do not deploy the following configuration in your crucial production environment.

### Prerequisites

- TiDB Operator **v1.1.0-beta.1** (or later versions) is installed, and the related Custom Resource Definition (CRD) file is updated.

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
    version: v3.1.0
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

> **Note:**
>
> Currently, one TidbMonitor can monitor only one TidbCluster.

You can also quickly deploy TidbMonitor using the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n ${namespace}
```

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
    version: v3.1.0
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

In some cases, TidbMonitor needs to obtain the monitoring metrics on Kubernetes. To obtain the kube-prometheus metrics, configure `TidbMonitor.Spec.kubePrometheusURL`. For details, refer to [kube-prometheus](https://github.com/coreos/kube-prometheus).

Similarly, you can configure TidbMonitor to push the monitoring alert to AlertManager. For details, refer to [AlertManager](https://prometheus.io/docs/alerting/alertmanager/).

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
    version: v3.1.0
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

## References

For more detailed API information of TidbMonitor, refer to [TiDB Operator API documentation](api-references.md).
