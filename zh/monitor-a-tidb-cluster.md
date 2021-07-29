---
title: TiDB 集群的监控与告警
summary: 介绍如何监控 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/monitor-a-tidb-cluster/', '/docs-cn/tidb-in-kubernetes/dev/monitor-using-tidbmonitor/', '/zh/tidb-in-kubernetes/dev/monitor-using-tidbmonitor/']
---

# TiDB 集群的监控与告警

本文介绍如何对通过 TiDB Operator 部署的 TiDB 集群进行监控及配置告警。

## TiDB 集群的监控

TiDB 通过 Prometheus 和 Grafana 监控 TiDB 集群。在通过 TiDB Operator 创建新的 TiDB 集群时，可以对于每个 TiDB 集群，创建、配置一套独立的监控系统，与 TiDB 集群运行在同一 Namespace，包括 Prometheus 和 Grafana 两个组件。

在 [TiDB 集群监控](https://pingcap.com/docs-cn/stable/deploy-monitoring-services/)中有一些监控系统配置的细节可供参考。

在 v1.1 及更高版本的 TiDB Operator 中，可以通过简单的 CR 文件（即 TidbMonitor，可参考 [tidb-operator 中的示例](https://github.com/pingcap/tidb-operator/blob/master/manifests/monitor/tidb-monitor.yaml)）来快速建立对 Kubernetes 集群上的 TiDB 集群的监控。

> **注意：**
>
> * `spec.clusters[].name` 需要配置为 TiDB 集群 TidbCluster 的名字。

### 持久化监控数据

可以在 `TidbMonitor` 中设置 `spec.persistent` 为 `true` 来持久化监控数据。开启此选项时应将 `spec.storageClassName` 设置为一个当前集群中已有的存储，并且此存储应当支持将数据持久化，否则会存在数据丢失的风险。配置示例如下：

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

你可以通过以下命令来确认 PVC 情况:

{{< copyable "shell-regular" >}}

```shell
kubectl get pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/component=monitor -n ${namespace}
```

```
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
basic-monitor   Bound    pvc-6db79253-cc9e-4730-bbba-ba987c29db6f   5G         RWO            standard       51s
```

### 自定义 Prometheus 配置

用户可以通过自定义配置文件或增加额外的命令行参数，来自定义 Prometheus 配置。

#### 使用自定义配置文件

1. 为用户自定义配置创建 ConfigMap 并将 `data` 部分的键名设置为 `prometheus-config`。
2. 设置 `spec.prometheus.config.configMapRef.name` 与 `spec.prometheus.config.configMapRef.namespace` 为自定义 ConfigMap 的名称与所属的 namespace。

如需了解完整的配置示例，可参考 [tidb-operator 中的示例](https://github.com/pingcap/tidb-operator/blob/master/examples/monitor-with-externalConfigMap/README.md)。

#### 增加额外的命令行参数

设置 `spec.prometheus.config.commandOptions` 为用于启动 Prometheus 的额外的命令行参数。

如需了解完整的配置示例，可参考 [tidb-operator 中的示例](https://github.com/pingcap/tidb-operator/blob/master/examples/monitor-with-externalConfigMap/README.md)。

> **注意：**
>
> 以下参数已由 TidbMonitor controller 自动设置，不支持通过 `commandOptions` 重复指定：
>
> - `config.file`
> - `log.level`
> - `web.enable-admin-api`
> - `web.enable-lifecycle`
> - `storage.tsdb.path`
> - `storage.tsdb.retention`
> - `storage.tsdb.max-block-duration`
> - `storage.tsdb.min-block-duration`

### 访问 Grafana 监控面板

可以通过 `kubectl port-forward` 访问 Grafana 监控面板：

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-grafana 3000:3000 &>/tmp/portforward-grafana.log &
```

然后在浏览器中打开 [http://localhost:3000](http://localhost:3000)，默认用户名和密码都为 `admin`。

也可以设置 `spec.grafana.service.type` 为 `NodePort` 或者 `LoadBalancer`，通过 `NodePort` 或者 `LoadBalancer` 查看监控面板。

如果不需要使用 Grafana，可以在部署时将 `TidbMonitor` 中的 `spec.grafana` 部分删除。这一情况下需要使用其他已有或新部署的数据可视化工具直接访问监控数据来完成可视化。

### 访问 Prometheus 监控数据

对于需要直接访问监控数据的情况，可以通过 `kubectl port-forward` 来访问 Prometheus：

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-prometheus 9090:9090 &>/tmp/portforward-prometheus.log &
```

然后在浏览器中打开 [http://localhost:9090](http://localhost:9090)，或通过客户端工具访问此地址即可。

也可以设置 `spec.prometheus.service.type` 为 `NodePort` 或者 `LoadBalancer`，通过 `NodePort` 或者 `LoadBalancer` 访问监控数据。

### 设置 kube-prometheus 与 AlertManager

在部分情况下，你可能需要 TidbMonitor 同时获取 Kubernetes 上的监控指标。你可以通过设置 `TidbMonitor.Spec.kubePrometheusURL` 来使其获取 [kube-prometheus](https://github.com/coreos/kube-prometheus) metrics。

同样的，你可以通过设置 TidbMonitor 来将监控推送警报至指定的 [AlertManager](https://prometheus.io/docs/alerting/alertmanager/)。

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

## 开启 Ingress

本节介绍如何为 TidbMonitor 开启 Ingress。[Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) 是一个 API 对象，负责管理集群中服务的外部访问。

### 环境准备

使用 `Ingress` 前，需要在 Kubernetes 集群中安装 `Ingress` 控制器，否则仅创建 `Ingress` 资源无效。你可能需要部署 `Ingress` 控制器，例如 [ingress-nginx](https://kubernetes.github.io/ingress-nginx/deploy/)。你可以从许多 [Ingress 控制器](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/)中进行选择。

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

在公有云 Kubernetes 集群中，通常可以[配置 Loadbalancer](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/) 通过域名访问 Ingress。如果无法配置 Loadbalancer 服务，比如使用了 NodePort 作为 Ingress 的服务类型，可通过与如下命令等价的方式访问服务：

```shell
curl -H "Host: example.com" ${node_ip}:${NodePort}
```

## 告警配置

在随 TiDB 集群部署 Prometheus 时，会自动导入一些默认的告警规则，可以通过浏览器访问 Prometheus 的 Alerts 页面查看当前系统中的所有告警规则和状态。

目前支持自定义配置告警规则，可以参考下面步骤修改告警规则：

1. 在为 TiDB 集群部署监控的过程中，设置 `spec.reloader.service.type` 为 `NodePort` 或者 `LoadBalancer`。
2. 通过 `NodePort` 或者 `LoadBalancer` 访问 reloader 服务，点击上方 `Files` 选择要修改的告警规则文件进行修改，修改完成后 `Save`。

默认的 Prometheus 和告警配置不能发送告警消息，如需发送告警消息，可以使用任意支持 Prometheus 告警的工具与其集成。推荐通过 [AlertManager](https://prometheus.io/docs/alerting/alertmanager/) 管理与发送告警消息。

如果在你的现有基础设施中已经有可用的 AlertManager 服务，可以参考[设置 kube-prometheus 与 AlertManager](#设置-kube-prometheus-与-alertmanager) 设置 `spec.alertmanagerURL` 配置其地址供 Prometheus 使用；如果没有可用的 AlertManager 服务，或者希望部署一套独立的服务，可以参考官方的[说明](https://github.com/prometheus/alertmanager)部署。

## 多集群监控

从 TiDB Operator 1.2 版本起，TidbMonitor 支持跨命名空间的多集群监控。

### 使用 YAML 文件配置多集群监控

无论要监控的集群是否已开启 `TLS`，你都可以通过配置 TidbMonitor 的 YAML 文件实现。

配置示例如下:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbMonitor
metadata:
  name: basic
spec:
  clusters:
    - name: ns1
      namespace: ns1
    - name: ns2
      namespace: ns2
  persistent: true
  storage: 5G
  prometheus:
    baseImage: prom/prometheus
    version: v2.18.1
    service:
      type: NodePort
  grafana:
    baseImage: grafana/grafana
    version: 6.7.6
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

如需了解完整的配置示例，可参考 TiDB Operator 仓库中的[示例](https://github.com/pingcap/tidb-operator/tree/master/examples/monitor-multiple-cluster-non-tls)。

### 使用 Grafana 查看多集群监控

当 `tidb-monitor-initializer` 镜像版本在 `< v4.0.14`、`< v5.0.3` 范围时，要使用 Grafana 查看多个集群的监控，请在每个 Grafana Dashboard 中进行以下操作：

1. 点击 Grafana Dashboard 中的 **Dashboard settings** 选项，打开 **Settings** 面板。
2. 在 **Settings** 面板中，选择 **Variables** 中的 **tidb_cluster** 变量，将 **tidb_cluster** 变量的 **Hide** 属性设置为空选项。
3. 返回当前 Grafana Dashboard (目前无法保存对于 **Hide** 属性的修改)，即可看到集群选择下拉框。下拉框中的集群名称格式为 `${namespace}-${name}`。

如果需要保存对 Grafana Dashboard 的修改， Grafana 必须为 `6.5` 及以上版本，TiDB Operator 必须为 v1.2.0-rc.2 及以上版本。
