---
title: Deploy TiDB Operator in Kubernetes
summary: Learn how to deploy TiDB Operator in Kubernetes.
category: how-to
---

# Deploy TiDB Operator in Kubernetes

This document describes how to deploy TiDB Operator in Kubernetes.

## Prerequisites

Before deploying TiDB Operator, make sure the following items are installed on your machine:

* Kubernetes >= v1.12
* [DNS addons](https://kubernetes.io/docs/tasks/access-application-cluster/configure-dns-cluster/)
* [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
* [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) enabled (optional)
* [Helm](https://helm.sh) version >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797)

## Deploy Kubernetes cluster

TiDB Operator runs in Kubernetes cluster. You can refer to [the document of how to set up Kubernetes](https://kubernetes.io/docs/setup/) to set up a Kubernetes cluster. Make sure that the Kubernetes version is v1.12 or higher. If you want to deploy a very simple Kubernetes cluster for testing purposes, consult the [Get Started](get-started.md) document.

TiDB Operator uses [Persistent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to persist the data of TiDB cluster (including the database, monitoring data, and backup data), so the Kubernetes cluster must provide at least one kind of persistent volume. For better performance, it is recommended to use local SSD disk as the volumes. Follow [this step](#configure-local-persistent-volume) to provision local persistent volumes.

It is recommended to enable [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) in the Kubernetes cluster.

## Install Helm

Refer to [Use Helm](tidb-toolkit.md#use-helm) to install Helm and configre it with the official PingCAP chart Repo.

## Configure local persistent volume

### Prepare local volumes

Refer to [Local PV Configuration](configure-storage-class.md) to set up local persistent volumes in your Kubernetes cluster.

## Install TiDB Operator

TiDB Operator uses [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to extend Kubernetes. Therefore, to use TiDB Operator, you must first create the `TidbCluster` custom resource type, which is a one-time job in your Kubernetes cluster.

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

After `TidbCluster` custom resource type is created, install TiDB Operator in your Kubernetes cluster.

1. Get the `values.yaml` file of the `tidb-operator` chart you want to install.

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p /home/tidb/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=${chart_version} > /home/tidb/tidb-operator/values-tidb-operator.yaml
    ```

    > **Note:**
    >
    > `${chart_version}` represents the chart version of TiDB Operator. For example, `v1.0.0`. You can view the currently supported versions by running the `helm search -l tidb-operator` command.

2. Install TiDB Operator.

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-operator --name=tidb-operator --namespace=tidb-admin --version=${chart_version} -f /home/tidb/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

## Customize TiDB Operator

To customize TiDB Operator, modify `/home/tidb/tidb-operator/values-tidb-operator.yaml`. The rest sections of the document use `values.yaml` to refer to `/home/tidb/tidb-operator/values-tidb-operator.yaml`

TiDB Operator contains two components:

* tidb-controller-manager
* tidb-scheduler

These two components are stateless and deployed via `Deployment`. You can customize resource `limit`, `request`, and `replicas` in the `values.yaml` file.

After modifying `values.yaml`, run the following command to apply this modification:

{{< copyable "shell-regular" >}}

```shell
helm upgrade tidb-operator pingcap/tidb-operator --version=${chart_version} -f /home/tidb/tidb-operator/values-tidb-operator.yaml
```
