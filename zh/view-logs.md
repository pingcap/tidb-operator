---
title: 查看日志
summary: 介绍如何查看 TiDB 集群各组件日志以及 TiDB 慢查询日志。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/view-logs/']
---

# 查看日志

本文档介绍如何查看 TiDB 集群各组件日志，以及 TiDB 慢查询日志。

## TiDB 集群各组件日志

通过 TiDB Operator 部署的 TiDB 各组件默认将日志输出在容器的 `stdout` 和 `stderr` 中。可以通过下面的方法查看单个 Pod 的日志：

{{< copyable "shell-regular" >}}

```bash
kubectl logs -n ${namespace} ${pod_name}
```

如果这个 Pod 由多个 Container 组成，可以查看这个 Pod 内某个 Container 的日志：

{{< copyable "shell-regular" >}}

```bash
kubectl logs -n ${namespace} ${pod_name} -c ${container_name}
```

请通过 `kubectl logs --help` 获取更多查看 Pod 日志的方法。

## TiDB 组件慢查询日志

TiDB 3.0 及以上的版本中，慢查询日志和应用日志区分开，可以通过名为 `slowlog` 的 sidecar 容器查看慢查询日志：

{{< copyable "shell-regular" >}}

```bash
kubectl logs -n ${namespace} ${pod_name} -c slowlog
```

> **注意：**
>
> 慢查询日志的格式与 MySQL 的慢查询日志相同，但由于 TiDB 自身的特点，其中的一些具体字段可能存在差异，因此解析 MySQL 慢查询日志的工具不一定能完全兼容 TiDB 的慢查询日志。
