---
title: 启用 TidbCluster 弹性伸缩
summary: 介绍如何使用 TidbCluster 的弹性伸缩能力。
category: how-to
---

# 启用 TidbCluster 弹性伸缩

在 Kubernetes 平台上，有着基于 CPU 利用率进行负载的原生 API: [Horizontal Pod Autoscaler](https://kubernetes.io/zh/docs/tasks/run-application/horizontal-pod-autoscale/)。与之相应的，在 Operator 1.1 版本及以上，TiDB 集群可以凭借 Kubernetes 平台本身的特性来开启弹性调度的能力。本篇文章将会介绍如何开启并使用 TidbCluster 的弹性伸缩能力。

## 开启弹性伸缩特性

> **警告：**
>
> * TidbCluster 弹性伸缩目前仍处于 Alpha 阶段，我们极其不推荐在关键、生产环境开启这个特性
> * 我们推荐你在测试、内网环境对这个特性进行体验，并反馈相关的建议与问题给我们，帮助我们更好地提高这一特性能力。

开启弹性伸缩特性需要主动开启 Operator 相关配置，默认情况下 Operator 的弹性伸缩特性是关闭的。你可以通过以下方式来开启弹性调度特性:

1. 修改 Operator 的 `values.yaml`

在 `features` 选项中开启 AutoScaling：

```yaml
features:
  - AutoScaling=true
```

开启 Operator Webhook 特性:

```yaml
admissionWebhook:
  create: true
  mutation:
    pods: true
```

2. 安装 / 更新 operator

修改完 `values.yaml` 文件中上述配置项以后进行 TiDB-Operator 部署或者更新。 安装与更新 Operator 请参考 [在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md)


## 了解 TidbClusterAutoScaler

我们通过 `TidbClusterAutoScaler` CR 对象来控制 TiDB 集群的弹性伸缩行为，如果你曾经使用过 [Horizontal Pod Autoscaler](https://kubernetes.io/zh/docs/tasks/run-application/horizontal-pod-autoscale/)， 那么你一定会对这个概念感到非常熟悉。以下是一个 TiKV 的弹性伸缩例子。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbClusterAutoScaler
metadata:
  name: auto-scaling-demo
spec:
  cluster:
    name: auto-scaling-demo
    namespace: default
  monitor:
    name: auto-scaling-demo
    namespace: default
  tikv:
    minReplicas: 3
    maxReplicas: 4
    metrics:
      - type: "Resource"
        resource:
          name: "cpu"
          target:
            type: "Utilization"
            averageUtilization: 80
```

对于 TiDB 组件，你可以通过 `spec.tidb` 来进行配置，目前 TiKV 与 TiDB 的弹性伸缩 API 相同。

在 `TidbClusterAutoScaler` 对象中，`cluster` 属性代表了需要被弹性调度的 Tidb 集群，通过 name 与 namespace 来代表。 由于 `TidbClusterAutoScaler` 组件需要通过指标采集组件抓取相关资源使用情况，我们需要提供对应的指标采集与查询服务给 `TidbClusterAutoScaler`。`monitor` 属性则代表了与之相连的 TidbMonitor 对象。 如果你并不了解 TidbMonitor，可以参考 [通过 TidbMonitor 监控 TiDB 集群](monitor-using-tidbmonitor.md)

对于非 `TidbMonitor` 的外部 `Prometheus`, 你可以通过 `spec.metricsUrl` 来填写这个服务的 Host ,从而指定该 TiDB 集群的监控指标采集服务。对于使用 `helm` 部署 TiDB 集群监控的情况，可以通过以下方式来指定 `spec.metricsUrl`。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbClusterAutoScaler
metadata:
  name: auto-scaling-demo
spec:
  cluster:
    name: auto-scaling-demo
    namespace: default
  metricsUrl: "http://<release-name>-prometheus.<release-namespace>.svc:9090"
  ......
```

`minReplicas` 与 `maxReplicas` 则代表了对于目标组件弹性的上下限。

目前 `TidbClusterAutoScaler` 仅支持基于 CPU 负载的弹性伸缩，CPU 负载的描述性 API 如下所示。`averageUtilization` 则代表了 CPU 负载利用率的阈值。如果当前 CPU 利用率超过 80%，则会触发弹性扩容。

```yaml
    metrics:
      - type: "Resource"
        resource:
          name: "cpu"
          target:
            type: "Utilization"
            averageUtilization: 80
```


## 快速上手

我们将通过以下指令快速部署一个 3 PD、3 TiKV、2 TiDB，并带有监控与弹性伸缩能力的 TiDB 集群。

```shell
$ kubectl apply -f examples/auto-scale/  -n <namespace>
tidbclusterautoscaler.pingcap.com/auto-scaling-demo created
tidbcluster.pingcap.com/auto-scaling-demo created
tidbmonitor.pingcap.com/auto-scaling-demo created
```

当 TiDB 集群创建完毕以后，你可以通过 [sysbench](https://www.percona.com/blog/tag/sysbench/) 等数据库压测工具进行压测来体验弹性伸缩功能。

使用如下命令销毁环境：

```shell
kubectl delete tidbcluster auto-scaling-demo -n <namespace>
kubectl delete tidbmonitor auto-scaling-demo -n <namespace>
kubectl delete tidbclusterautoscaler auto-scaling-demo -n <namespace>
```

## 配置 TidbClusterAutoScaler

相比无状态的 Web 服务，一个分布式数据库软件对于实例的伸缩往往是非常敏感的。我们需要保证每次弹性伸缩之间存在一定的间隔，从而避免引起频繁的弹性伸缩。
你可以通过 `spec.tikv.scaleOutThreshold` 和 `spec.tikv.scaleInThreshold` 来配置每两次弹性伸缩之间的时间间隔，对于 TiDB 也同样如此。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbClusterAutoScaler
metadata:
  name: auto-sclaer
spec:
  tidb:
    scaleOutIntervalSeconds: 60
    scaleInIntervalSeconds: 60
  tikv:
    scaleOutIntervalSeconds: 10
    scaleInIntervalSeconds: 10
```
