---
title: Deploy TiDB Operator in Kubernetes
summary: Learn how to deploy TiDB Operator in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-tidb-operator/']
---

# Deploy TiDB Operator in Kubernetes

This document describes how to deploy TiDB Operator in Kubernetes.

## Prerequisites

Before deploying TiDB Operator, make sure the following items are installed on your machine:

* Kubernetes >= v1.12
* [DNS addons](https://kubernetes.io/docs/tasks/access-application-cluster/configure-dns-cluster/)
* [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
* [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) enabled (optional)
* [Helm 3](https://helm.sh)

### Deploy the Kubernetes cluster

TiDB Operator runs in the Kubernetes cluster. You can refer to [the document of how to set up Kubernetes](https://kubernetes.io/docs/setup/) to set up a Kubernetes cluster. Make sure that the Kubernetes version is v1.12 or higher. If you want to deploy a very simple Kubernetes cluster for testing purposes, consult the [Get Started](get-started.md) document.

For some public cloud environments, refer to the following documents:

- [Deploy on AWS EKS](deploy-on-aws-eks.md)
- [Deploy on GCP GKE](deploy-on-gcp-gke.md)
- [Deploy on Alibaba Cloud ACK](deploy-on-alibaba-cloud.md)

TiDB Operator uses [Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to persist the data of TiDB cluster (including the database, monitoring data, and backup data), so the Kubernetes cluster must provide at least one kind of persistent volumes. For better performance, it is recommended to use local SSD disks as the volumes. Follow [this step](#configure-local-persistent-volumes) to provision local persistent volumes.

It is recommended to enable [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) in the Kubernetes cluster.

### Install Helm

Refer to [Use Helm](tidb-toolkit.md#use-helm) to install Helm and configure it with the official PingCAP chart Repo.

## Configure local persistent volumes

Refer to [Local PV Configuration](configure-storage-class.md) to set up local persistent volumes in your Kubernetes cluster.

## Deploy TiDB Operator

### Create CRD

TiDB Operator uses [Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) to extend Kubernetes. Therefore, to use TiDB Operator, you must first create the `TidbCluster` CRD, which is a one-time job in your Kubernetes cluster.

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml
```

If the server cannot access the Internet, you need to download the `crd.yaml` file on a machine with Internet access before installing:

{{< copyable "shell-regular" >}}

```shell
wget https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml
kubectl apply -f ./crd.yaml
```

If the following message is displayed, the CRD installation is successful:

{{< copyable "shell-regular" >}}

```shell
kubectl get crd
```

```shell
NAME                                 CREATED AT
backups.pingcap.com                  2020-06-11T07:59:40Z
backupschedules.pingcap.com          2020-06-11T07:59:41Z
restores.pingcap.com                 2020-06-11T07:59:40Z
tidbclusterautoscalers.pingcap.com   2020-06-11T07:59:42Z
tidbclusters.pingcap.com             2020-06-11T07:59:38Z
tidbinitializers.pingcap.com         2020-06-11T07:59:42Z
tidbmonitors.pingcap.com             2020-06-11T07:59:41Z
```

### Customize TiDB Operator deployment

To deploy TiDB Operator quickly, you can refer to [Deploy TiDB Operator](get-started.md#deploy-tidb-operator). This section describes how to customize the deployment of TiDB Operator.

After creating CRDs in the step above, there are two methods to deploy TiDB Operator on your Kubernetes cluster: online and offline.

#### Online deployment

1. Get the `values.yaml` file of the `tidb-operator` chart you want to deploy:

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p ${HOME}/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=${chart_version} > ${HOME}/tidb-operator/values-tidb-operator.yaml
    ```

    > **Note:**
    >
    > `${chart_version}` represents the chart version of TiDB Operator. For example, `v1.2.3`. You can view the currently supported versions by running the `helm search repo -l tidb-operator` command.

2. Configure TiDB Operator

    TiDB Operator will use the `k8s.gcr.io/kube-scheduler` image. If you cannot download the image, you can modify the `scheduler.kubeSchedulerImageName` in the `${HOME}/tidb-operator/values-tidb-operator.yaml` file to `registry.cn-hangzhou.aliyuncs.com/google_containers/kube-scheduler`.

    TiDB Operator manages all TiDB clusters in the Kubernetes cluster by default. If you only need it to manage clusters in a specific namespace, you can set `clusterScoped: false` in `values.yaml`.

    > **Note:**
    >
    > After setting `clusterScoped: false`, TiDB Operator will still operate Nodes, Persistent Volumes, and Storage Classe in the Kubernetes cluster by default. If the role that deploys TiDB Operator does not have the permissions to operate these resources, you can set the corresponding permission request under `controllerManager.clusterPermissions` to `false` to disable TiDB Operator's operations on these resources.

    You can modify other items such as `limits`, `requests`, and `replicas` as needed.

3. Deploy TiDB Operator

    {{< copyable "shell-regular" >}}

    ```shell
    helm install tidb-operator pingcap/tidb-operator --namespace=tidb-admin --version=${chart_version} -f ${HOME}/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

    > **Note:**
    >
    > If the corresponding `tidb-admin` namespace does not exist, you can create the namespace first by running the `kubectl create namespace tidb-admin` command.

4. Upgrade TiDB Operator

    If you need to upgrade the TiDB Operator, modify the `${HOME}/tidb-operator/values-tidb-operator.yaml` file, and then execute the following command to upgrade:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator pingcap/tidb-operator --namespace=tidb-admin -f ${HOME}/tidb-operator/values-tidb-operator.yaml
    ```

#### Offline installation

If your server cannot access the Internet, install TiDB Operator offline by the following steps:

1. Download the `tidb-operator` chart

    If the server has no access to the Internet, you cannot configure the Helm repository to install the TiDB Operator component and other applications. At this time, you need to download the chart file needed for cluster installation on a machine with Internet access, and then copy it to the server.

    Use the following command to download the `tidb-operator` chart file:

    {{< copyable "shell-regular" >}}

    ```shell
    wget http://charts.pingcap.org/tidb-operator-v1.2.3.tgz
    ```

    Copy the `tidb-operator-v1.2.3.tgz` file to the target server and extract it to the current directory:

    {{< copyable "shell-regular" >}}

    ```shell
    tar zxvf tidb-operator.v1.2.3.tgz
    ```

2. Download the Docker images used by TiDB Operator

    If the server has no access to the Internet, you need to download all Docker images used by TiDB Operator on a machine with Internet access and upload them to the server, and then use `docker load` to install the Docker image on the server.

    The Docker images used by TiDB Operator are:

    {{< copyable "" >}}

    ```shell
    pingcap/tidb-operator:v1.2.3
    pingcap/tidb-backup-manager:v1.2.3
    bitnami/kubectl:latest
    pingcap/advanced-statefulset:v0.3.3
    k8s.gcr.io/kube-scheduler:v1.16.9
    ```

    Among them, `k8s.gcr.io/kube-scheduler:v1.16.9` should be consistent with the version of your Kubernetes cluster. You do not need to download it separately.

    Next, download all these images using the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    docker pull pingcap/tidb-operator:v1.2.3
    docker pull pingcap/tidb-backup-manager:v1.2.3
    docker pull bitnami/kubectl:latest
    docker pull pingcap/advanced-statefulset:v0.3.3

    docker save -o tidb-operator-v1.2.3.tar pingcap/tidb-operator:v1.2.3
    docker save -o tidb-backup-manager-v1.2.3.tar pingcap/tidb-backup-manager:v1.2.3
    docker save -o bitnami-kubectl.tar bitnami/kubectl:latest
    docker save -o advanced-statefulset-v0.3.3.tar pingcap/advanced-statefulset:v0.3.3
    ```

    Next, upload these Docker images to the server, and execute `docker load` to install these Docker images on the server:

    {{< copyable "shell-regular" >}}

    ```shell
    docker load -i tidb-operator-v1.2.3.tar
    docker load -i tidb-backup-manager-v1.2.3.tar
    docker load -i bitnami-kubectl.tar
    docker load -i advanced-statefulset-v0.3.3.tar
    ```

3. Configure TiDB Operator

    TiDB Operator embeds a `kube-scheduler` to implement a custom scheduler. To configure the Docker image's name and version of this built-in `kube-scheduler` component, modify the `./tidb-operator/values.yaml` file. For example, if `kube-scheduler` in your Kubernetes cluster uses the image `k8s.gcr.io/kube-scheduler:v1.16.9`, set `./tidb-operator/values.yaml` as follows:

    ```shell
    ...
    scheduler:
      serviceAccount: tidb-scheduler
      logLevel: 2
      replicas: 1
      schedulerName: tidb-scheduler
      resources:
        limits:
          cpu: 250m
          memory: 150Mi
        requests:
          cpu: 80m
          memory: 50Mi
      kubeSchedulerImageName: k8s.gcr.io/kube-scheduler
      kubeSchedulerImageTag: v1.16.9
    ...
    ```

    You can modify other items such as `limits`, `requests`, and `replicas` as needed.

4. Install TiDB Operator

    Install TiDB Operator using the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install tidb-operator ./tidb-operator --namespace=tidb-admin
    ```

    > **Note:**
    >
    > If the corresponding `tidb-admin` namespace does not exist, you can create the namespace first by running the `kubectl create namespace tidb-admin` command.

5. Upgrade TiDB Operator

    If you need to upgrade TiDB Operator, modify the `./tidb-operator/values.yaml` file, and then execute the following command to upgrade:

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade tidb-operator ./tidb-operator --namespace=tidb-admin
    ```

## Customize TiDB Operator

To customize TiDB Operator, modify `${HOME}/tidb-operator/values-tidb-operator.yaml`. The rest sections of the document use `values.yaml` to refer to `${HOME}/tidb-operator/values-tidb-operator.yaml`

TiDB Operator contains two components:

* tidb-controller-manager
* tidb-scheduler

These two components are stateless and deployed via `Deployment`. You can customize resource `limit`, `request`, and `replicas` in the `values.yaml` file.

After modifying `values.yaml`, run the following command to apply this modification:

{{< copyable "shell-regular" >}}

```shell
helm upgrade tidb-operator pingcap/tidb-operator --version=${chart_version} --namespace=tidb-admin -f ${HOME}/tidb-operator/values-tidb-operator.yaml
```
