---
title: Upgrade TiDB Operator and Kubernetes
summary: Learn how to upgrade TiDB Operator and Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/upgrade-tidb-operator/']
---

# Upgrade TiDB Operator and Kubernetes

This document describes how to upgrade TiDB Operator and Kubernetes.

## Upgrade TiDB Operator

1. Update [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/):

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${version}/manifests/crd.yaml && \
    kubectl get crd tidbclusters.pingcap.com
    ```

    > **Note:**
    >
    > The `${version}` in this document represents the version of TiDB Operator, such as `v1.2.0-beta.2`. You can check the currently supported version using the `helm search repo -l tidb-operator` command.
    > If the command output does not include the latest version, update the repo using the `helm repo update` command. For details, refer to [Configure the Help repo](tidb-toolkit.md#configure-the-helm-repo).

2. Get the `values.yaml` file of the `tidb-operator` chart that you want to install:

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p ${HOME}/tidb-operator/${version} && \
    helm inspect values pingcap/tidb-operator --version=${version} > ${HOME}/tidb-operator/${version}/values-tidb-operator.yaml
    ```

3. Modify the `operatorImage` image in the `${HOME}/tidb-operator/${version}/values-tidb-operator.yaml` file. Merge the customized configuration in the old `values.yaml` file with the `${HOME}/tidb-operator/${version}/values-tidb-operator.yaml` file, and execute `helm upgrade`:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator pingcap/tidb-operator --version=${version} -f ${HOME}/tidb-operator/${version}/values-tidb-operator.yaml
    ```

    After all the Pods start normally, execute the following command to check the image of TiDB Operator:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n tidb-admin -l app.kubernetes.io/instance=tidb-operator -o yaml | grep 'image:.*operator:'
    ```

    If TiDB Operator is successfully upgraded, the expected output is as follows. `${version}` represents the desired version of TiDB Operator.

    ```
    image: pingcap/tidb-operator:${version}
    image: docker.io/pingcap/tidb-operator:${version}
    image: pingcap/tidb-operator:${version}
    image: docker.io/pingcap/tidb-operator:${version}
    ```

    > **Note:**
    >
    > After TiDB Operator is upgraded, the `discovery` deployment in all TiDB clusters will automatically upgrade to the specified version of TiDB Operator.

## Upgrade TiDB Operator from v1.0 to v1.1 or later releases

Since TiDB Operator v1.1.0, PingCAP no longer updates or maintains the tidb-cluster chart. The components and features that have been managed using the tidb-cluster chart will be managed by CR (Custom Resource) or dedicated charts in v1.1. For more details, refer to [TiDB Operator v1.1 Notes](notes-tidb-operator-v1.1.md).

> **Note:**
>
> By default, TiDB (starting from v4.0.2) periodically shares usage details with PingCAP to help understand how to improve the product. For details about what is shared and how to disable the sharing, see [Telemetry](https://docs.pingcap.com/tidb/stable/telemetry).
