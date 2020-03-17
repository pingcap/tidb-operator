---
title: 通过 TidbMonitor 监控 TiDB 集群
category: how-to
---

# Kubernetes 上的 TiDB 集群监控

在 Operator 1.1 版本及以上，我们可以通过简单的 CR 文件，即 TidbMonitor 来快速建立对 Kubernetes 集群上的 TiDB 集群的监控。


## 快速上手

> **注意：**
>
> 快速上手只是用来作为演示或是测试，请勿将以下配置部署在关键、生产环境。

> 前置条件:

    1. 已经安装了 Operator `v1.1.0-beta.1` 及以上版本，并且已经更新了相关版本的 CRD 文件
    2. 已经设置了默认的 storageClass， 并保证其有足够的 PV (默认情况下需要 6 个PV)

    这可以通过以下指令来验证:

```shell
$ kubectl get storageClass
NAME                 PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
standard (default)   rancher.io/local-path   Delete          WaitForFirstConsumer   false                  14h
```


### 安装

你可以通过在 Kubernetes 集群上，通过 CR 文件快速建立起一个 TiDB 集群，以及与之相关联的 TidbMonitor 监控组件。

首先我们通过 CR 文件快速部署一个简单的 Tidb 集群:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml -n <namespace>
``` 

然后我们通过 kubectl get pod 命令来检查 TiDB 集群启动完毕:

```shell
$ kubectl get pod -l app.kubernetes.io/instance=basic -n <namespace>
NAME                               READY   STATUS    RESTARTS   AGE
basic-discovery-66bdb44dfc-mtlzb   1/1     Running   0          22m
basic-pd-0                         1/1     Running   0          22m
basic-pd-1                         1/1     Running   0          22m
basic-pd-2                         1/1     Running   1          22m
basic-tidb-0                       1/1     Running   0          19m
basic-tidb-1                       1/1     Running   0          19m
basic-tikv-0                       1/1     Running   0          21m
basic-tikv-1                       1/1     Running   0          21m
basic-tikv-2                       1/1     Running   0          21m
```

接下来我们可以将以下内容存为 yaml 文件，通过 kubectl apply -f 的方式部署部署一个 TidbMonitor 组件。

> **注意：**
>
> 目前一个 TidbMonitor 只支持监控一个 TidbCluster。

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
    version: v3.0.5
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

或者你可以通过以下方式快速部署 TidbMonitor:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n <namespace>
```

然后我们通过 kubectl get pod 命令来检查 TidbMonitor 启动完毕:

{{< copyable "shell-regular" >}}

```shell
$ kubectl get pod -l app.kubernetes.io/instance=basic | grep monitor 
basic-monitor-85fcf66bc4-cwpcn     3/3     Running   0          117s
```

### 查看监控面板

```shell
> kubectl -n <namespace> port-forward svc/basic-grafana 4000:4000 &>/tmp/pf-grafana.log &
```

然后访问 localhost:4000

### 删除集群与监控

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbcluster basic -n <namespace>
```

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n <namespace>
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
  storageClassName: <storageClassName>
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
    version: v3.0.5
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

你可以通过以下命令来确认 PVC 情况:

```shell
$ kubectl get pvc -l app.kubernetes.io/instance=basic,app.kubernetes.io/component=monitor
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
basic-monitor   Bound    pvc-6db79253-cc9e-4730-bbba-ba987c29db6f   5G         RWO            standard       51s
```

### 设置 kube-prometheus 与 AlertManager

你可以通过设置 TidbMonitor 来使其获取 kube-prometheus metrics，了解 [kube-prometheus](https://github.com/coreos/kube-prometheus)

同样的，你可以通过设置 TidbMonitor 来将监控推送警报至指定的 AlertManager，了解 [AlertManager](https://prometheus.io/docs/alerting/alertmanager/)

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
    version: v3.0.5
  reloader:
    baseImage: pingcap/tidb-monitor-reloader
    version: v1.0.1
  imagePullPolicy: IfNotPresent
```

## 参考

了解 TidbMonitor 更为详细的 API 设置，可以参考 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.html)。
