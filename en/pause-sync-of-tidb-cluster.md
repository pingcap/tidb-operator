---
title: Pause Sync of a TiDB Cluster in Kubernetes
summary: Introduce how to pause sync of a TiDB cluster in Kubernetes
---

# Pause Sync of a TiDB Cluster in Kubernetes

This document introduces how to pause sync of a TiDB cluster in Kubernetes using configuration.

## What is sync in TiDB Operator

In TiDB Operator, controller regulates the state of the TiDB cluster in Kubernetes. The controller constantly compares the desired state recorded in the `TidbCluster` object with the actual state of the TiDB cluster. This process is referred to as **sync** generally. For more details, refer to [TiDB Operator Architecture](architecture.md).

## Use scenarios

Here are some cases where you might need to pause sync of a TiDB cluster in Kubernetes.

- Avoid unexpected rolling update

    To prevent new versions of TiDB Operator from introducing compatibility issues into the clusters, before updating TiDB Operator, you can pause sync of TiDB clusters. After updating TiDB Operator, you can resume syncing clusters one by one, or specify a time for resume. In this way, you can observe how the rolling update of TiDB Operator would affect the cluster.

- Avoid multiple rolling restarts

    In some cases, you might need to continuously modify the cluster over a period of time, but do not want to restart the TiDB cluster many times. To avoid multiple rolling restarts, you can pause sync of the cluster. During the sync pausing, any change of the configuration does not take effect on the cluster. After you finish the modification, resume sync of the TiDB cluster. All changes can be applied in a single rolling restart.

- Maintenance window

    In some situations, you can update or restart the TiDB cluster only during a maintenance window. When outside the maintenance window, you can pause sync of the TiDB cluster, so that any modification to the specs does not take effect. When inside the maintenance window, you can resume sync of the TiDB cluster to allow TiDB cluster to rolling update or restart.

## Pause sync

1. Execute the following command to edit the TiDB cluster configuration. `${cluster_name}` represents the name of the TiDB cluster, and `${namespace}` refers to the TiDB cluster namespace.

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. Configure `spec.paused: true` as follows, and save changes. The sync of TiDB cluster components (PD, TiKV, TiDB, TiFlash, TiCDC, Pump) is then paused. 

    {{< copyable "" >}}
    
    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      ...
    spec:
      ...
      paused: true  # Pausing sync of TiDB cluster
      pd:
        ...
      tikv:
        ...
      tidb:
        ...
    ```

3. To confirm the sync status of a TiDB cluster, execute the following command. `${pod_name}` is the name of tidb-controller-manager Pod, and `${namespace}` is the namespace of TiDB Operator.

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl logs ${pod_name} -n ${namespace} | grep paused
    ```
    
    The expected output is as follows. The sync of all components in the TiDB cluster is paused.
    
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

## Resume sync

To resume the sync of the TiDB cluster, configure `spec.paused: false` in the TidbCluster CR.

1. Execute the following command to edit the TiDB cluster configuration. `${cluster_name}` represents the name of the TiDB cluster, and `${namespace}` refers to the TiDB cluster namespace.

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl edit tc ${cluster_name} -n ${namespace}
    ```

2. Configure `spec.paused: false` as follows, and save changes. The sync of TiDB cluster components (PD, TiKV, TiDB, TiFlash, TiCDC, Pump) is then resumed. 

    {{< copyable "" >}}
    
    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      ...
    spec:
      ...
      paused: false  # Resuming sync of TiDB cluster
      pd:
        ...
      tikv:
        ...
      tidb:
        ...
    ```

3. To confirm the sync status of a TiDB cluster, execute the following command. `${pod_name}` represents the name of the tidb-controller-manager Pod, and `${namespace}` represents the namespace of TiDB Operator.

    {{< copyable "shell-regular" >}}
    
    ```bash
    kubectl logs ${pod_name} -n ${namespace} | grep "Finished syncing TidbCluster"
    ```
    
    The expected output is as follows. The `finished syncing` timestamp is later than the `paused` timestamp, which indicates that sync of the TiDB cluster has been resumed.
    
    ```
    I1207 11:14:59.361353       1 tidb_cluster_controller.go:136] Finished syncing TidbCluster "default/basic" (368.816685ms)
    I1207 11:15:28.982910       1 tidb_cluster_controller.go:136] Finished syncing TidbCluster "default/basic" (97.486818ms)
    I1207 11:15:29.360446       1 tidb_cluster_controller.go:136] Finished syncing TidbCluster "default/basic" (377.51187ms)
    ```
