---
title: Advanced StatefulSet Controller
summary: Learn how to enable and use the advanced StatefulSet controller.
aliases: ['/docs/tidb-in-kubernetes/dev/advanced-statefulset/']
---

# Advanced StatefulSet Controller

**Feature Stage**: Alpha

Kubernetes has a built-in [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) that allocates consecutive serial numbers to Pods. For example, when there are three replicas, the Pods are named as pod-0, pod-1, and pod-2. When scaling out or scaling in, you must add a Pod at the end or delete the last pod. For example, when you scale out to four replicas, pod-3 is added. When you scale in to two replicas, pod-2 is deleted.

When you use local storage, Pods are associated with the Nodes storage resources and cannot be scheduled freely. If you want to delete one of the Pods in the middle to maintain its Node but no other Nodes can be migrated, or if you want to delete a Pod that fails and to create another Pod with a different serial number, you cannot implement such desired function by the built-in StatefulSet.

The [advanced StatefulSet controller](https://github.com/pingcap/advanced-statefulset) is implemented based on the built-in StatefulSet controller. It supports freely controlling the serial number of Pods. This document describes how to use the advanced StatefulSet controller in TiDB Operator.

## Enable

1. Load the Advanced StatefulSet CRD file:

    * For Kubernetes versions < 1.16:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/advanced-statefulset-crd.v1beta1.yaml
        ```

    * For Kubernetes versions >= 1.16:

        {{< copyable "shell-regular" >}}

        ```
        kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/advanced-statefulset-crd.v1.yaml
        ```

2. Enable the `AdvancedStatefulSet` feature in `values.yaml` of the TiDB Operator chart:

    {{< copyable "" >}}

    ```yaml
    features:
    - AdvancedStatefulSet=true
    advancedStatefulset:
      create: true
    ```

    Upgrade TiDB Operator. For details, refer to [Upgrade TiDB Operator](upgrade-tidb-operator.md).

> **Note:**
>
> If the `AdvancedStatefulSet` feature is enabled, TiDB Operator converts the current `StatefulSet` object into an `AdvancedStatefulSet` object. However, after the `AdvancedStatefulSet` feature is disabled, the `AdvancedStatefulSet` object cannot be automatically converted to the built-in `StatefulSet` object of Kubernetes.

## Usage

This section describes how to use the advanced StatefulSet controller.

### View the `AdvancedStatefulSet` Object by kubectl

The data format of `AdvancedStatefulSet` is the same as that of `StatefulSet`, but `AdvancedStatefulSet` is implemented based on CRD, with `asts` as the alias. You can view the `AdvancedStatefulSet` object in the namespace by running the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get -n ${namespace} asts
```

### Specify the Pod to be scaled in

With the advanced StatefulSet controller, when scaling in TidbCluster, you can not only reduce the number of replicas, but also specify the scaling in of any Pod in the PD, TiDB, or TiKV components by configuring annotations.

For example:

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: asts
spec:
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 4
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
```

The above configuration deploys 4 TiKV instances, namely `basic-tikv-0`, `basic-tikv-1`, ..., `basic-tikv-3`. If you want to delete `basic-tikv-1`, set `spec.tikv.replicas` to `3` and configure the following annotations:

{{< copyable "" >}}

```yaml
metadata:
  annotations:
    tikv.tidb.pingcap.com/delete-slots: '[1]'
```

> **Note:**
>
> When modifying `replicas` and `delete-slots annotation`, complete the modification in the same operation; otherwise, the controller operates the modification according to the general expectations.

The complete example is as follows:

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  annotations:
    tikv.tidb.pingcap.com/delete-slots: '[1]'
  name: asts
spec:
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
```

The supported annotations are as follows:

- `pd.tidb.pingcap.com/delete-slots`: Specifies the serial numbers of the Pods to be deleted in the PD component.
- `tidb.tidb.pingcap.com/delete-slots`: Specifies the serial number of the Pods to be deleted in the TiDB component.
- `tikv.tidb.pingcap.com/delete-slots`: Specifies the serial number of the Pods to be deleted in the TiKV component.

The value of Annotation is an integer array of JSON, such as `[0]`, `[0,1]`, `[1,3]`.

### Specify the location to scale out

You can reverse the above operation of scaling in to restore `basic-tikv-1`.

> **Note:**
>
> The specified scaling out performed by the advanced StatefulSet controller is the same as the regular StatefulSet scaling, which does not delete the Persistent Volume Claims (PVCs) associated with the Pod. If you want to avoid using the previous data, delete the associated PVCs before scaling out at the original location.

For example:

{{< copyable "" >}}

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  annotations:
    tikv.tidb.pingcap.com/delete-slots: '[]'
  name: asts
spec:
  version: v5.1.1
  timezone: UTC
  pvReclaimPolicy: Delete
  pd:
    baseImage: pingcap/pd
    replicas: 3
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 4
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 2
    service:
      type: ClusterIP
    config: {}
```

The delete-slots annotations can be left empty or deleted completely.
