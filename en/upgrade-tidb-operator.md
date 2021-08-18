---
title: Upgrade TiDB Operator and Kubernetes
summary: Learn how to upgrade TiDB Operator and Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/upgrade-tidb-operator/']
---

# Upgrade TiDB Operator and Kubernetes

This document describes how to upgrade TiDB Operator and Kubernetes.

> **Note:**
>
> You can check the currently supported versions of TiDB Operator using the `helm search repo -l tidb-operator` command.
> If the command output does not include the latest version, update the repo using the `helm repo update` command. For details, refer to [Configure the Help repo](tidb-toolkit.md#configure-the-helm-repo).

## Upgrade TiDB Operator online

1. Update CustomResourceDefinition (CRD) for Kubernetes. For more information about CRD, see [CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/).

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.2.1/manifests/crd.yaml && \
    kubectl get crd tidbclusters.pingcap.com
    ```

2. Get the `values.yaml` file of the `tidb-operator` chart for the new TiDB Operator version. 

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p ${HOME}/tidb-operator/v1.2.1 && \
    helm inspect values pingcap/tidb-operator --version=v1.2.1 > ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
    ```

3. In the `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml` file, modify the `operatorImage` version to the new TiDB Operator version. Merge the customized configuration in the old `values.yaml` file to the `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml` file, and then execute `helm upgrade`:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator pingcap/tidb-operator --version=v1.2.1 -f ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
    ```

    After all the Pods start normally, execute the following command to check the image of TiDB Operator:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n tidb-admin -l app.kubernetes.io/instance=tidb-operator -o yaml | grep 'image:.*operator:'
    ```

    If TiDB Operator is successfully upgraded, the expected output is as follows. `v1.2.1` represents the desired version of TiDB Operator.

    ```
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    ```

    > **Note:**
    >
    > After TiDB Operator is upgraded, the `discovery` deployment in all TiDB clusters will be automatically upgraded to the corresponding version of TiDB Operator.

## Upgrade TiDB Operator offline

If your server cannot access the Internet, you can take the following steps to upgrade TiDB Operator offline:

1. Download the files and images required for the upgrade using a machine with the Internet access:

    1. Download the `crd.yaml` file for the new TiDB Operator version. For more information about CRD, see [CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/).

        {{< copyable "shell-regular" >}}

        ```shell
        wget https://raw.githubusercontent.com/pingcap/tidb-operator/v1.2.1/manifests/crd.yaml
        ```

    2. Download the `tidb-operator` chart package file.

        {{< copyable "shell-regular" >}}

        ```shell
        wget http://charts.pingcap.org/tidb-operator-v1.2.1.tgz
        ```

    3. Download the Docker images required for the new TiDB Operator version:

        {{< copyable "shell-regular" >}}
    
        ```shell
        docker pull pingcap/tidb-operator:v1.2.1
        docker pull pingcap/tidb-backup-manager:v1.2.1

        docker save -o tidb-operator-v1.2.1.tar pingcap/tidb-operator:v1.2.1
        docker save -o tidb-backup-manager-v1.2.1.tar pingcap/tidb-backup-manager:v1.2.1
        ```

2. Upload the downloaded files and images to the server that needs to be upgraded, and then take the following steps for installation:

    1. Install the `crd.yaml` file for TiDB Operator:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl apply -f . /crd.yaml
        ```

    2. Unpack the `tidb-operator` chart package file, and then copy the `values.yaml` file:

        {{< copyable "shell-regular" >}}

        ```shell
        tar zxvf tidb-operator-v1.2.1.tgz && \
        mkdir -p ${HOME}/tidb-operator/v1.2.1 &&
        cp tidb-operator/values.yaml ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
        ```

    3. Install the Docker images on the server:

        {{< copyable "shell-regular" >}}

        ```shell
        docker load -i tidb-operator-v1.2.1.tar
        docker load -i tidb-backup-manager-v1.2.1.tar
        ```

3. In the `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml` file, modify the `operatorImage` version to the new TiDB Operator version. Merge the customized configuration in the old `values.yaml` file to the `${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml` file, and then execute `helm upgrade`:

   {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator ./tidb-operator --version=v1.2.1 -f ${HOME}/tidb-operator/v1.2.1/values-tidb-operator.yaml
    ```

   After all the Pods start normally, execute the following command to check the image version of TiDB Operator:

   {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n tidb-admin -l app.kubernetes.io/instance=tidb-operator -o yaml | grep 'image:.*operator:'
    ```

   If TiDB Operator is successfully upgraded, the expected output is as follows. `v1.2.1` represents the new version of TiDB Operator.

    ```
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    image: pingcap/tidb-operator:v1.2.1
    image: docker.io/pingcap/tidb-operator:v1.2.1
    ```

   > **Note:**
   >
   > After TiDB Operator is upgraded, the `discovery` deployment in all TiDB clusters will be automatically upgraded to the specified version of TiDB Operator.

## Upgrade TiDB Operator from v1.0 to v1.1 or later releases

Since TiDB Operator v1.1.0, PingCAP no longer updates or maintains the tidb-cluster chart. The components and features that have been managed using the tidb-cluster chart will be managed by CR (Custom Resource) or dedicated charts in v1.1. For more details, refer to [TiDB Operator v1.1 Notes](notes-tidb-operator-v1.1.md).
