---
title: Deploy TiDB in Kubernetes Using kind
summary: Learn how to deploy a TiDB cluster in Kubernetes using kind.
category: how-to
aliases: ['/docs/dev/tidb-in-kubernetes/deploy-tidb-from-kubernetes-dind/']
---

# Deploy TiDB in Kubernetes Using kind

This tutorial shows how to deploy [TiDB Operator](https://github.com/pingcap/tidb-operator) and a TiDB cluster in Kubernetes on your laptop (Linux or macOS) using [kind](https://kind.sigs.k8s.io/).

kind is a tool for running local Kubernetes clusters using Docker containers as cluster nodes. It is developed for testing local Kubernetes clusters, initially targeting the conformance tests. The Kubernetes cluster version depends on the node image that your kind uses, and you can specify the image to be used for the nodes and choose any other published version. Refer to [Docker hub](https://hub.docker.com/r/kindest/node/tags) to see available tags.

> **Warning:**
>
> This deployment is for testing only. DO NOT USE in production!

## Prerequisites

Before deployment, make sure the following requirements are satisfied:

- [Docker](https://docs.docker.com/install/): version >= 17.03
- [Helm](https://helm.sh/docs/intro/install/): Helm 2 or the latest stable version of Helm 3
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl): version >= 1.12
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/): version >= 0.7.0 (the latest version recommended)
- The value of [net.ipv4.ip_forward](https://linuxconfig.org/how-to-turn-on-off-ip-forwarding-in-linux) should be set to `1`

## Step 1: Create a Kubernetes cluster using kind

Refer to [`kind` Quick Start](https://kind.sigs.k8s.io/docs/user/quick-start) to install `kind` and create a cluster.

The following is an example of using `kind` v0.8.1:

```
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.8.1/kind-$(uname)-amd64
chmod +x ./kind
./kind create cluster
```

Check whether the cluster is successfully created:

{{< copyable "shell-regular" >}}

```
kubectl cluster-info
```

## Step 2: Deploy TiDB Operator

1. Install Helm. You can either refer to [Installing Helm](https://helm.sh/docs/intro/install/) or install the latest stable version of Helm 3:

    {{< copyable "shell-regular" >}}

    ```shell
    curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
    ```

2. Create [CRDs (Custom Resource Definition)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/). TiDB Operator uses CRDs to extend Kubernetes. Therefore, to use TiDB Operator, you must first create CRDs, such as `TidbCluster`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.0-rc.3/manifests/crd.yaml && \
    kubectl get crd tidbclusters.pingcap.com
    ```

3. Install TiDB Operator.

    - The following is an example of installing TiDB Operator using Helm 3:

        {{< copyable "shell-regular" >}}

        ```shell
        helm repo add pingcap https://charts.pingcap.org/
        kubectl create ns pingcap
        helm install --namespace pingcap tidb-operator pingcap/tidb-operator --version v1.1.0-rc.3
        ```

    - If you use Helm 2, refer to [Install Helm](tidb-toolkit.md#use-helm) to initialize Helm. After that, deploy TiDB Operator by running the following command:

        {{< copyable "shell-regular" >}}

        ```shell
        helm repo add pingcap https://charts.pingcap.org/
        helm install --namespace pingcap --name tidb-operator pingcap/tidb-operator --version v1.1.0-rc.3
        ```

4. Check the status of TiDB Operator:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n pingcap get pods
    ```

## Step 3: Deploy the TiDB cluster

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

## Access the database and monitoring dashboards

To access the TiDB cluster, use the `kubectl port-forward` command to expose services to the host. The ports in the command are in `<host machine port>:<k8s service port>` format.

- Access TiDB using the MySQL client

    Before you start testing your TiDB cluster, make sure you have installed a MySQL client.

    1. Use kubectl to forward the host machine port to the TiDB service port:

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl port-forward svc/basic-tidb 4000:4000 --namespace=demo
        ```

        If the output is similar to `Forwarding from 0.0.0.0:4000 -> 4000`, then the proxy is set up.

    2. To access TiDB using the MySQL client, open a **new** terminal tab or window and run the following command:

        {{< copyable "shell-regular" >}}

        ``` shell
        mysql -h 127.0.0.1 -P 4000 -u root
        ```

        When the testing finishes, press <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop the proxy and exit.

- View the monitoring dashboard

    1. Use kubectl to forward the host machine port to the Grafana service port:

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl port-forward svc/basic-grafana 3000:3000 --namespace=demo
        ```

        If the output is similar to `Forwarding from 0.0.0.0:4000 -> 4000`, then the proxy is set up.

    2. Open your web browser at <http://localhost:3000> to access the Grafana monitoring dashboard.

        - default username: admin
        - default password: admin

        When the testing finishes, press <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop the proxy and exit.

    > **Note:**
    >
    > If you are deploying kind on a remote machine rather than a local PC, there might be problems accessing the monitoring dashboard of the remote system through "localhost".
    >
    > When you use kubectl 1.13 or later versions, you can expose the port on `0.0.0.0` instead of the default `127.0.0.1` by adding `--address 0.0.0.0` to the `kubectl port-forward` command.
    >
    > {{< copyable "shell-regular" >}}
    >
    > ```
    > kubectl port-forward --address 0.0.0.0 -n tidb svc/basic-grafana 3000:3000
    > ```
    >
    > Then, open your browser at `http://<VM's IP address>:3000` to access the Grafana monitoring dashboard.

## Destroy the TiDB and Kubernetes cluster

To destroy the local TiDB cluster, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl delete tc basic -n demo
```

To destroy the monitor component, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl delete tidbmonitor basic -n demo
```

To destroy the Kubernetes cluster, run the following command:

{{< copyable "shell-regular" >}}

``` shell
kind delete cluster
```
