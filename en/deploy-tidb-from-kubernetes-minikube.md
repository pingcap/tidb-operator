---
title: Deploy TiDB in the Minikube Cluster
summary: Learn how to deploy TiDB in the minikube cluster.
category: how-to
---

# Deploy TiDB in the Minikube Cluster

This document describes how to deploy a TiDB cluster in the [minikube](https://kubernetes.io/docs/setup/minikube/) cluster.

> **Warning:**
>
> This is for testing only. DO NOT USE in production!

## Start a Kubernetes cluster with minikube

[Minikube](https://kubernetes.io/docs/setup/minikube/) can start a local Kubernetes cluster inside a VM on your laptop. It works on macOS, Linux, and Windows.

> **Note:**
>
> Although Minikube supports `--vm-driver=none` that uses host docker instead of VM, it is not fully tested with TiDB Operator and may not work. If you want to try TiDB Operator on a system without virtualization support (e.g., on a VPS), you might consider using [kind](deploy-tidb-from-kubernetes-kind.md) instead.

### Install minikube and start a Kubernetes cluster

See [Installing Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) to install minikube (1.0.0+) on your machine.

After you installed minikube, you can run the following command to start a Kubernetes cluster.

```shell
minikube start
```

For Chinese mainland users, you may use local gcr.io mirrors such as `registry.cn-hangzhou.aliyuncs.com/google_containers`.

```shell
minikube start --image-repository registry.cn-hangzhou.aliyuncs.com/google_containers
```

Or you can configure HTTP/HTTPS proxy environments in your Docker:

```shell
# change 127.0.0.1:1086 to your http/https proxy server IP:PORT
minikube start --docker-env https_proxy=http://127.0.0.1:1086 \
  --docker-env http_proxy=http://127.0.0.1:1086
```

> **Note:**
>
> As minikube is running with VMs (default), the `127.0.0.1` is the VM itself, you might want to use your real IP address of the host machine in some cases.

See [minikube setup](https://kubernetes.io/docs/setup/minikube/) for more options to configure your virtual machine and Kubernetes cluster.

### Install kubectl to access the cluster

The Kubernetes command-line tool, [kubectl](https://kubernetes.io/docs/user-guide/kubectl/), allows you to run commands against Kubernetes clusters.

Install kubectl according to the instructions in [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

After kubectl is installed, test your minikube Kubernetes cluster:

```shell
kubectl cluster-info
```

## Install Helm

[Helm](https://helm.sh/) is a package management tool for Kubernetes. Make sure your Helm version >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797). The installation steps are as follows:

1. Refer to [Helm official documentation](https://v2.helm.sh/docs/using_helm/#installing-helm) to install Helm client.

2. Install Helm server.

    Apply the `RBAC` rule required by the `tiller` component in the cluster and install `tiller`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/tiller-rbac.yaml && \
    helm init --service-account=tiller --upgrade
    ```

    To confirm that the `tiller` Pod is in the `running` state, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n kube-system -l name=tiller
    ```

3. Add the repository:

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo add pingcap https://charts.pingcap.org/
    ```

    Use `helm search` to search the chart provided by PingCAP:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l
    ```

## Deploy TiDB Operator

TiDB Operator uses [CRD (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to extend Kubernetes. Therefore, to use TiDB Operator, you must first create the `TidbCluster` CRD.

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml && \
kubectl get crd tidbclusters.pingcap.com
```

After `TidbCluster` CRD is created, install TiDB Operator in your Kubernetes cluster.

1. Get the `values.yaml` file of the `tidb-operator` chart you want to install:

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p /home/tidb/tidb-operator && \
    helm inspect values pingcap/tidb-operator --version=v1.1.0-rc.1 > /home/tidb/tidb-operator/values-tidb-operator.yaml
    ```

    Modify the configuration in `values.yaml` according to your needs.

2. Install TiDB Operator:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install pingcap/tidb-operator --name=tidb-operator --namespace=tidb-admin --version=v1.1.0-rc.1 -f /home/tidb/tidb-operator/values-tidb-operator.yaml && \
    kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
    ```

## Deploy the TiDB cluster

To deploy the TiDB cluster, perform the following steps:

1. Create `Namespace`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create namespace demo
    ```

2. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml -n demo
    ```

3. Deploy the TiDB cluster monitor:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-monitor.yaml -n demo
    ```

4. View the Pod status:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get po -n demo
    ```

## Test the TiDB cluster

Before you start testing your TiDB cluster, make sure you have installed a MySQL client. Note that there can be a small delay between the time when the pod is up and running, and when the service is available. You can watch the list of available services with:

```shell
kubectl get svc -n demo --watch
```

When you see `basic-tidb` appear, the service is ready to access. You can use <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop the process.

To connect your MySQL client to the TiDB server, take the following steps:

1. Forward a local port to the TiDB port.

    ```shell
    kubectl -n demo port-forward svc/basic-tidb 4000:4000
    ```

2. In another terminal window, connect the TiDB server with a MySQL client:

    ```shell
    mysql -h 127.0.0.1 -P 4000 -uroot
    ```

    Or you can run a SQL command directly:

    ```shell
    mysql -h 127.0.0.1 -P 4000 -uroot -e 'select tidb_version();'
    ```

## Monitor the TiDB cluster

To monitor the status of the TiDB cluster, take the following steps.

1. Forward a local port to the Grafana port.

    ```shell
    kubectl -n demo port-forward svc/basic-grafana 3000:3000
    ```

2. Open your browser, and access Grafana at `http://localhost:3000`.

    Alternatively, Minikube provides `minikube service` that exposes Grafana as a service for you to access more conveniently.

    ```shell
    minikube service basic-grafana -n demo
    ```

    And it will automatically set up the proxy and open the browser for Grafana.

## Delete the TiDB cluster

To destroy a TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl delete tc basic -n demo
```

To destroy the monitoring component, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n demo
```

Update the reclaim policy of PVs used by the demo cluster to `Delete`:

{{< copyable "shell-regular" >}}

```shell
kubectl get pv -l app.kubernetes.io/instance=basic -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

Delete PVCs:

{{< copyable "shell-regular" >}}

```shell
kubectl delete pvc -l app.kubernetes.io/managed-by=tidb-operator
```

## FAQs

### TiDB cluster in minikube is not responding or responds slow

The minikube VM is configured by default to only use 2048MB of memory and 2 CPUs. You can allocate more resources during `minikube start` using the `--memory` and `--cpus` flag. Note that you'll need to recreate minikube VM for this to take effect.

```shell
minikube delete
minikube start --cpus 4 --memory 4096 ...
```
