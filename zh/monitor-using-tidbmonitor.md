---
title: 通过 TidbMonitor 监控 TiDB 集群
aliases: ['/docs-cn/tidb-in-kubernetes/dev/monitor-using-tidbmonitor/']
---

# 通过 TidbMonitor 监控 TiDB 集群

在 v1.1 及更高版本的 TiDB Operator 中，我们可以通过简单的 CR 文件（即 TidbMonitor）来快速建立对 Kubernetes 集群上的 TiDB 集群的监控。

## 快速上手

### 前置条件

1. 已经安装了 Operator `v1.1.0` 及以上版本，并且已经更新了相关版本的 CRD 文件
2. 已经设置了默认的 storageClass，并保证其有足够的 PV（默认情况下需要 6 个 PV）。这可以通过以下指令来验证：

```shell
$ kubectl get storageClass
NAME                 PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
standard (default)   rancher.io/local-path   Delete          WaitForFirstConsumer   false                  14h
```

### 安装

你可以在 Kubernetes 集群上通过 CR 文件快速建立起一个 TidbMonitor 监控 TiDB 集群。接下来我们可以将以下内容存为 yaml 文件，通过 `kubectl apply -f` 的方式部署一个 TidbMonitor 组件。

> **注意：**
>
> * 一个 TidbMonitor 只支持监控一个 TidbCluster。
> * `spec.clusters[0].name` 需要配置为 TiDB 集群 TidbCluster 的名字。

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
    version: v4.0.6
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

或者你可以通过以下方式快速部署 TidbMonitor:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n ${namespace}
```

如果服务器没有外网，请参考 [部署 TiDB 集群](deploy-on-general-kubernetes.md#部署-tidb-集群) 在有外网的机器上将用到的 Docker 镜像下载下来并上传到服务器上。

然后我们通过 kubectl get pod 命令来检查 TidbMonitor 启动完毕:

{{< copyable "shell-regular" >}}

```shell
$ kubectl get pod -l app.kubernetes.io/instance=basic -n ${namespace} | grep monitor
basic-monitor-85fcf66bc4-cwpcn     3/3     Running   0          117s
```

### 查看监控面板

运行以下命令查看监控面板：

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} port-forward svc/basic-grafana 3000:3000 &>/tmp/pf-grafana.log &
```

然后访问 localhost:3000。

### 删除监控

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n ${namespace}
```

## 持久化监控数据

如果我们需要将 TidbMonitor 的监控数据持久化存储，我们需要在 TidbMonitor 中开启持久化选项:

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
    version: v4.0.6
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

你可以通过以下命令来确认 PVC 情况:

```shell
$ kubectl get pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/component=monitor -n ${namespace}
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
basic-monitor   Bound    pvc-6db79253-cc9e-4730-bbba-ba987c29db6f   5G         RWO            standard       51s
```

### 设置 kube-prometheus 与 AlertManager

在部分情况下，你可能需要 TidbMonitor 同时获取 Kubernetes 上的监控指标。你可以通过设置 TidbMonitor.Spec.kubePrometheusURL 来使其获取 kube-prometheus metrics，了解 [kube-prometheus](https://github.com/coreos/kube-prometheus)。

同样的，你可以通过设置 TidbMonitor 来将监控推送警报至指定的 AlertManager，了解 [AlertManager](https://prometheus.io/docs/alerting/alertmanager/)。

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
    version: v4.0.6
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

## 开启 Ingress

### 环境准备

使用 `Ingress` 前需要 Kubernetes 集群安装有 `Ingress` 控制器，仅创建 `Ingress` 资源无效。您可能需要部署 `Ingress` 控制器，例如 [ingress-nginx](https://kubernetes.github.io/ingress-nginx/deploy/)。您可以从许多 [Ingress 控制器](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) 中进行选择。

更多关于 `Ingress` 环境准备，可以参考 [Ingress 环境准备](https://kubernetes.io/zh/docs/concepts/services-networking/ingress/#%E7%8E%AF%E5%A2%83%E5%87%86%E5%A4%87)

### 使用 Ingress 访问 TidbMonitor 

目前, `TidbMonitor` 提供了通过 Ingress 将 Prometheus/Grafana 服务暴露出去的方式，你可以通过 [Ingress 文档](https://kubernetes.io/zh/docs/concepts/services-networking/ingress/)了解更多关于 Ingress 的详情。

以下是一个开启了 Prometheus 与 Grafana Ingress 的 `TidbMonitor` 例子：

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
    version: v4.0.6
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

你可以通过 `spec.prometheus.ingress.annotations` 与 `spec.grafana.ingress.annotations` 来设置对应的 Ingress Annotations 的设置。如果你使用的是默认的 Nginx Ingress 方案，你可以在 [Nginx Ingress Controller Annotation](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/) 了解更多关于 Annotations 的详情。

`TidbMonitor` 的 Ingress 设置同样支持设置 TLS，以下是一个为 Ingress 设置 TLS 的例子。你可以通过 [Ingress TLS](https://kubernetes.io/zh/docs/concepts/services-networking/ingress/#tls) 来了解更多关于 Ingress TLS 的资料。

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
    version: v4.0.6
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

 TLS Secret 必须包含名为 tls.crt 和 tls.key 的密钥，这些密钥包含用于 TLS 的证书和私钥，例如：

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

## 参考

了解 TidbMonitor 更为详细的 API 设置，可以参考 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)。
