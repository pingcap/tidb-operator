---
title: Enable Admission Controller in TiDB Operator
summary: Learn how to enable the admission controller in TiDB Operator and the functionality of the admission controller.
category: how-to
---

# Enable Admission Controller in TiDB Operator

Kubernetes v1.9 introduces the [dynamic admission control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) to modify and validate resources. TiDB Operator also supports the dynamic admission control to modify, validate, and maintain resources. This document describes how to enable the admission controller and introduces the functionality of the admission controller.

## Enable the admission controller

With a default installation, TiDB Operator disables the admission controller. Take the following steps to manually turn it on.

1. Edit the `values.yaml` file in TiDB Operator.

    Enable the `Operator Webhook` feature:

    ```yaml
    admissionWebhook:
      create: true
    ```

2. Configure the failure policy.

    Prior to Kubernetes v1.15, the management mechanism of the dynamic admission control is coarser-grained and is inconvenient to use. To prevent the impact of the dynamic admission control on the global cluster, you need to configure the [Failure Policy](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy).

    * For Kubernetes versions earlier than v1.15, it is recommended to set the `failurePolicy` of TiDB Operator to `Ignore`. This avoids the influence on the global cluster in case of `admission webhook` exception in TiDB Operator .

        ```yaml
        ......
        failurePolicy:
            validation: Ignore
            mutation: Ignore
        ```

    * For Kubernetes v1.15 and later versions, it is recommended to set the `failurePolicy` of TiDB Operator to `Failure`. The exception occurs in `admission webhook` does not effect the whole cluster, because the dynamic admission control supports the label-based filtering mechanism.

        ```yaml
        ......
        failurePolicy:
            validation: Fail
            mutation: Fail
        ```

3. Install or update TiDB Operator.

    To install or update TiDB Operator, see [Deploy TiDB Operator in Kubernetes](deploy-tidb-operator.md).

## Functionality of admission controller

TiDB Operator implements many functions using the admission controller. This section introduces the admission controller for each resource and its corresponding functions.

* Admission controller for Pod validation

    The admission controller for Pod validation guarantees the safe logon and safe logoff of the PD/TiKV/TiDB component. You can [restart a TiDB cluster in Kubernetes](restart-a-tidb-cluster.md) using this controller. The component is enabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      validation:
        pods: true
    ```

* Admission controller for StatefulSet validation

    The admission controller for StatefulSet validation supports the gated launch of the TiDB/TiKV component in a TiDB cluster. The component is disabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      validation:
        statefulSets: false
    ```

    You can control the gated launch of the TiDB/TiKV component in a TiDB cluster through two annotations, `tidb.pingcap.com/tikv-partition` and `tidb.pingcap.com/tidb-partition`. To set the gated launch of the TiKV component in a TiDB cluster, execute the following commands. The effect of `partition=2` is the same as that of [StatefulSet Partitions](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#partitions).

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl annotate tidbcluster ${name} -n ${namespace} tidb.pingcap.com/tikv-partition=2 &&
    tidbcluster.pingcap.com/${name} annotated
    ```

    Execute the following commands to unset the gated launch:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl annotate tidbcluster ${name} -n ${namespace} tidb.pingcap.com/tikv-partition- &&
    tidbcluster.pingcap.com/${name} annotated
    ```

    This also applies to the TiDB component.

* Admission controller for TiDB Operator resources validation

    The admission controller for TiDB Operator resources validation supports validating customized resources such as `TidbCluster` and `TidbMonitor` in TiDB Operator. The component is disabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      validation:
        pingcapResources: false
    ```

    For example, regarding `TidbCluster` resources, the admission controller for TiDB Operator resources validation checks the required fields of the `spec` field. When you create or update `TidbCluster`, if the check is not passed (for example, neither of the `spec.pd.image` filed and the `spec.pd.baseImage` field are defined), this admission controller refuses the request.

* Admission controller for Pod modification

    The admission controller for Pod modification supports the hotspot scheduling of TiKV in the auto-scaling scenario. To [enable TidbCluster auto-scaling](enable-tidb-cluster-auto-scaling.md), you need to enable this controller. The component is enabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      mutation:
        pods: true
    ```

* Admission controller for TiDB Operator resources modification

    The admission controller for TiDB Operator resources modification supports filling in the default values of customized resources, such as `TidbCluster` and `TidbMonitor` in TiDB Operator. The component is enabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      mutation:
        pingcapResources: true
    ```
