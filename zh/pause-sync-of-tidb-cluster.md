---
title: 暂停同步 Kubernetes 上的 TiDB 集群
summary: 介绍如何暂停同步 Kubernetes 上的 TiDB 集群
---

# 暂停同步 Kubernetes 上的 TiDB 集群

本文介绍如何通过配置暂停同步 Kubernetes 上的 TiDB 集群。

## 什么是同步

在 TiDB Operator 中，控制器会不断对比 `TidbCluster` 对象中记录的期望状态与 TiDB 集群的实际状态，并调整 Kubernetes 中的资源以驱动 TiDB 集群满足期望状态。这个不断调整的过程通常被称为**同步**。更多细节参见[TiDB Operator架构](architecture.md)。

## 暂停同步的应用场景

以下为一些暂停同步的应用场景。

- 避免意外的滚动升级

    为防止 TiDB Operator 新版本的兼容性问题影响集群，升级 TiDB Operator 之前，可以先暂停同步集群。升级 TiDB Operator 之后，逐个恢复同步集群或者在指定时间恢复同步集群，以此来观察 TiDB Operator 版本升级对集群的影响。

- 避免多次滚动重启集群

    在某些情况下，一段时间内可能会多次修改 TiDB 集群配置，但是又不想多次滚动重启集群。为了避免多次滚动重启集群，可以先暂停同步集群，在此期间，对 TiDB 集群的任何配置都不会生效。集群配置修改完成后，恢复集群同步，此时暂停同步期间的所有配置修改都能在一次重启过程中被应用。

- 维护时间窗口

    在某些情况下，只允许在特定时间窗口内滚动升级或重启 TiDB 集群。因此可以在维护时间窗口之外的时间段暂停 TiDB 集群的同步过程，这样在维护时间窗口之外对 TiDB 集群的任何配置都不会生效；在维护时间窗口内，可以通过恢复 TiDB 集群同步来允许滚动升级或者重启 TiDB 集群。

## 暂停同步 TiDB 集群

1. 使用以下命令修改集群配置，其中 `${cluster_name}` 表示 TiDB 集群名称, `${namespace}` 表示 TiDB 集群所在的 namespace。

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. 在 TidbCluster CR 中以如下方式配置 `spec.paused: true`，保存配置并退出编辑器。TiDB 集群各组件 (PD、TiKV、TiDB、TiFlash、TiCDC、Pump) 的同步过程将会被暂停。

    {{< copyable "" >}}
    
    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      ...
    spec:
      ...
      paused: true  # 暂停同步
      pd:
        ...
      tikv:
        ...
      tidb:
        ...
    ```

3. TiDB 集群同步暂停后，可以使用以下命令查看 tidb-controller-manager Pod 日志确认 TiDB 集群同步状态。其中 `${pod_name}` 表示 tidb-controller-manager Pod 的名称，`${namespace}` 表示 TiDB Operator 所在的 namespace。

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl logs ${pod_name} -n ${namespace} | grep paused
    ```

    输出类似下方结果则表示 TiDB 集群同步已经暂停。
    
    ```
    I1207 11:09:59.029949       1 pd_member_manager.go:92] tidb cluster default/basic is paused, skip syncing for pd service
    I1207 11:09:59.029977       1 pd_member_manager.go:136] tidb cluster default/basic is paused, skip syncing for pd headless service
    I1207 11:09:59.035437       1 pd_member_manager.go:191] tidb cluster default/basic is paused, skip syncing for pd statefulset
    I1207 11:09:59.035462       1 tikv_member_manager.go:116] tikv cluster default/basic is paused, skip syncing for tikv service
    I1207 11:09:59.036855       1 tikv_member_manager.go:175] tikv cluster default/basic is paused, skip syncing for tikv statefulset
    I1207 11:09:59.036886       1 tidb_member_manager.go:132] tidb cluster default/basic is paused, skip syncing for tidb headless service
    I1207 11:09:59.036895       1 tidb_member_manager.go:258] tidb cluster default/basic is paused, skip syncing for tidb service
    I1207 11:09:59.039358       1 tidb_member_manager.go:188] tidb cluster default/basic is paused, skip syncing for tidb statefulset
    ```

## 恢复同步 TiDB 集群

如果想要恢复 TiDB 集群的同步，可以在 TidbCluster CR 中配置 `spec.paused: false`，恢复同步 TiDB 集群。

1. 使用以下命令修改集群配置，其中 `${cluster_name}` 表示 TiDB 集群名称, `${namespace}` 表示 TiDB 集群所在的 namespace。

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. 在 TidbCluster CR 中以如下方式配置 `spec.paused: false`，保存配置并退出编辑器。TiDB 集群各组件 (PD、TiKV、TiDB、TiFlash、TiCDC、Pump) 的同步过程将会被恢复。

    {{< copyable "" >}}
    
    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      ...
    spec:
      ...
      paused: false  # 恢复同步
      pd:
        ...
      tikv:
        ...
      tidb:
        ...
    ```

3. 恢复 TiDB 集群同步后，可以使用以下命令查看 tidb-controller-manager Pod 日志确认 TiDB 集群同步状态。其中 `${pod_name}` 表示 tidb-controller-manager Pod 的名称，`${namespace}` 表示 TiDB Operator 所在的 namespace。

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl logs ${pod_name} -n ${namespace} | grep "Finished syncing TidbCluster"
    ```
    
    输出类似下方结果，可以看到同步成功时间戳大于暂停同步日志中显示的时间戳，表示 TiDB 集群同步已经被恢复。
    
    ```
    I1207 11:14:59.361353       1 tidb_cluster_controller.go:136] Finished syncing TidbCluster "default/basic" (368.816685ms)
    I1207 11:15:28.982910       1 tidb_cluster_controller.go:136] Finished syncing TidbCluster "default/basic" (97.486818ms)
    I1207 11:15:29.360446       1 tidb_cluster_controller.go:136] Finished syncing TidbCluster "default/basic" (377.51187ms)
    ```
