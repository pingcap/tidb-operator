---
title: Tools in Kubernetes
summary: Learn about operation tools for TiDB in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/tidb-toolkit/']
---

# Tools in Kubernetes

Operations on TiDB in Kubernetes require some open source tools. In the meantime, there are some special requirements for operations using TiDB tools in the Kubernetes environment. This documents introduces in details the related operation tools for TiDB in Kubernetes.

## Use PD Control in Kubernetes

[PD Control](https://pingcap.com/docs/stable/reference/tools/pd-control) is the command-line tool for PD (Placement Driver). To use PD Control to operate on TiDB clusters in Kubernetes, firstly you need to establish the connection from local to the PD service using `kubectl port-forward`:

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379 &>/tmp/portforward-pd.log &
```

After the above command is executed, you can access the PD service via `127.0.0.1:2379`, and then use the default parameters of `pd-ctl` to operate. For example:

{{< copyable "shell-regular" >}}

```shell
pd-ctl -d config show
```

Assume that your local port `2379` has been occupied and you want to switch to another port:

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-pd ${local_port}:2379 &>/tmp/portforward-pd.log &
```

Then you need to explicitly assign a PD port for `pd-ctl`:

{{< copyable "shell-regular" >}}

```shell
pd-ctl -u 127.0.0.1:${local_port} -d config show
```

## Use TiKV Control in Kubernetes

[TiKV Control](https://pingcap.com/docs/stable/reference/tools/tikv-control) is the command-line tool for TiKV. When using TiKV Control for TiDB clusters in Kubernetes, be aware that each operation mode involves different steps, as described below:

* **Remote Mode**: In this mode, `tikv-ctl` accesses the TiKV service or the PD service through network. Firstly you need to establish the connection from local to the PD service and the target TiKV node using `kubectl port-forward`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379 &>/tmp/portforward-pd.log &
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl port-forward -n ${namespace} ${pod_name} 20160:20160 &>/tmp/portforward-tikv.log &
    ```

    After the connection is established, you can access the PD service and the TiKV node via the corresponding port in local:

    {{< copyable "shell-regular" >}}

    ```shell
    tikv-ctl --host 127.0.0.1:20160 ${subcommands}
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    tikv-ctl --pd 127.0.0.1:2379 compact-cluster
    ```

* **Local Mode**：In this mode, `tikv-ctl` accesses data files of TiKV, and the running TiKV instances must be stopped. To operate in the local mode, first you need to enter the [Diagnostic Mode](tips.md#use-the-diagnostic-mode) to turn off automatic re-starting for the TiKV instance, stop the TiKV process, and use the `tkctl debug` command to start in the target TiKV Pod a new container that contains the `tikv-ctl` executable. The steps are as follows:

    1. Enter the Diagnostic mode:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl annotate pod ${pod_name} -n ${namespace} runmode=debug
        ```

    2. Stop the TiKV process:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl exec ${pod_name} -n ${namespace} -c tikv -- kill -s TERM 1
        ```

    3. Start the debug container:

        {{< copyable "shell-regular" >}}

        ```shell
        tkctl debug ${pod_name} -c tikv
        ```

    4. Start using `tikv-ctl` in local mode. It should be noted that the root file system of `tikv` is under `/proc/1/root`, so you need to adjust the path of the data directory accordingly when executing a command:

        {{< copyable "shell-regular" >}}

        ```shell
        tikv-ctl --db /path/to/tikv/db size -r 2
        ```

        > **Note:**
        >
        >   The default db path of TiKV instances in the debug container is `/proc/1/root/var/lib/tikv/db`

## Use TiDB Control in Kubernetes

[TiDB Control](https://pingcap.com/docs/stable/reference/tools/tidb-control) is the command-line tool for TiDB. To use TiDB Control in Kubernetes, you need to access the TiDB node and the PD service from local. It is suggested you turn on the connection from local to the TiDB node and the PD service using `kubectl port-forward`:

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} svc/${cluster_name}-pd 2379:2379 &>/tmp/portforward-pd.log &
```

{{< copyable "shell-regular" >}}

```shell
kubectl port-forward -n ${namespace} ${pod_name} 10080:10080 &>/tmp/portforward-tidb.log &
```

Then you can use the `tidb-ctl`:

{{< copyable "shell-regular" >}}

```shell
tidb-ctl schema in mysql
```

## Use Helm

[Helm](https://helm.sh/) is a package management tool for Kubernetes. Make sure your Helm version >= 2.11.0 && < 3.0.0 && != [2.16.4](https://github.com/helm/helm/issues/7797). The installation steps are as follows:

### Install the Helm client

Refer to [Helm official documentation](https://v2.helm.sh/docs/using_helm/#installing-helm) to install the Helm client.

If the server does not have access to the Internet, you need to download the Helm client on a machine with Internet access, and then copy it to the server. Here is an example of installing the Helm client `2.16.7`:

{{< copyable "shell-regular" >}}

```shell
wget https://get.helm.sh/helm-v2.16.7-linux-amd64.tar.gz
tar zxvf helm-v2.16.7-linux-amd64.tar.gz
```

After decompression, you can see the following files:

```shell
linux-amd64/
linux-amd64/README.md
linux-amd64/tiller
linux-amd64/helm
linux-amd64/LICENSE
```

Copy the `linux-amd64/helm` file to the server and place it under the `/usr/local/bin/` directory.

Then execute `helm verison -c`. If the command outputs normally, the Helm client installation is successful:

{{< copyable "shell-regular" >}}

```shell
helm version -c
```

```shell
Client: &version.Version{SemVer:"v2.16.7", GitCommit:"5f2584fd3d35552c4af26036f0c464191287986b", GitTreeState:"clean"}
```

### Install the Helm server

#### Install RBAC

If `RBAC` is not enabled in the Kubernetes cluster, skip this section and [install Tiller](#install-tiller) directly.

The Helm server is a service called `tiller`, first install the `RBAC` rules required by `tiller`:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.5/manifests/tiller-rbac.yaml
```

If the server cannot access the Internet, download the `tiller-rbac.yaml` file on a machine with Internet access:

{{< copyable "shell-regular" >}}

```shell
wget https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.5/manifests/tiller-rbac.yaml
```

Copy the file `tiller-rbac.yaml` to the server and install the `RBAC`:

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f tiller-rbac.yaml
```

#### Install Tiller

The Helm server is a service called `tiller`, which runs as a pod in the Kubernetes cluster. To install `tiller`, run the following command:

{{< copyable "shell-regular" >}}

```shell
helm init --service-account=tiller --upgrade
```

The image used by Pod `tiller` is `gcr.io/kubernetes-helm/tiller:v2.16.7`. If your server cannot access `gcr.io`, you can try to use the mirror registry:

{{< copyable "shell-regular" >}}

``` shell
helm init --service-account=tiller --upgrade --tiller-image registry.cn-hangzhou.aliyuncs.com/google_containers/tiller:$(helm version --client --short | grep -Eo 'v[0-9]\.[0-9]+\.[0-9]+')
```

If the server cannot access the Internet, you need to download the Docker image used by `tiller` on a machine which can access the Internet:

{{< copyable "shell-regular" >}}

``` shell
docker pull gcr.io/kubernetes-helm/tiller:v2.16.7
docker save -o tiller-v2.16.7.tar gcr.io/kubernetes-helm/tiller:v2.16.7
```

Copy the file `tiller-v2.16.7.tar` to the server, and use the command `docker load` to load the image:

{{< copyable "shell-regular" >}}

``` shell
docker load -i tiller-v2.16.7.tar
```

Finally, install `tiller` with the following command and confirm that the `tiller` Pod is in the running state:

{{< copyable "shell-regular" >}}

```shell
helm init --service-account=tiller --skip-refresh
kubectl get po -n kube-system -l name=tiller
```

#### Configure the Helm repo

Kubernetes applications are packed as charts in Helm. PingCAP provides the following Helm charts for TiDB in Kubernetes:

* `tidb-operator`: used to deploy TiDB Operator;
* `tidb-cluster`: used to deploy TiDB clusters;
* `tidb-backup`: used to back up or restore TiDB clusters;
* `tidb-lightning`: used to import data into a TiDB cluster;
* `tidb-drainer`: used to deploy TiDB Drainer;
* `tikv-importer`: used to deploy TiKV Importer.

These charts are hosted in the Helm chart repository `https://charts.pingcap.org/` maintained by PingCAP. You can add this repository to your local server or computer using the following command:

{{< copyable "shell-regular" >}}

```shell
helm repo add pingcap https://charts.pingcap.org/
```

- If the Helm version < 2.16.0:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l
    ```

- If the Helm version >= 2.16.0:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search pingcap -l --devel
    ```

    ```
    NAME                    CHART VERSION   APP VERSION DESCRIPTION
    pingcap/tidb-backup     v1.0.0                      A Helm chart for TiDB Backup or Restore
    pingcap/tidb-cluster    v1.0.0                      A Helm chart for TiDB Cluster
    pingcap/tidb-operator   v1.0.0                      tidb-operator Helm chart for Kubernetes
    ...
    ```

    ```
    NAME                    CHART VERSION   APP VERSION DESCRIPTION
    pingcap/tidb-backup     v1.0.0                      A Helm chart for TiDB Backup or Restore
    pingcap/tidb-cluster    v1.0.0                      A Helm chart for TiDB Cluster
    pingcap/tidb-operator   v1.0.0                      tidb-operator Helm chart for Kubernetes
    ...
    ```

    When a new version of chart has been released, you can use `helm repo update` to update the repository cached locally:

    {{< copyable "shell-regular" >}}

#### Helm common operations

Common Helm operations include `helm install`, `helm upgrade`, and `helm del`. Helm chart usually contains many configurable parameters which could be tedious to configure manually. For convenience, it is recommended that you configure using a YAML file. Based on the conventions in the Helm community, the YAML file used for Helm configuration is named `values.yaml` in this document.

Before performing the deploy, upgrade and deploy, you can view the deployed applications via `helm ls`:

{{< copyable "shell-regular" >}}

```shell
helm ls
```

When performing a deployment or upgrade, you must specify the chart name (`chart-name`) and the name for the deployed application (`release-name`). You can also specify one or multiple `values.yaml` files to configure charts. In addition, you can use `chart-version` to specify the chart version (by default the latest GA is used). The steps in command line are as follows:

* Install:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${chart_name} --name=${release_name} --namespace=${namespace} --version=${chart_version} -f ${values_file}
    ```

* Upgrade (the upgrade can be either modifying the `chart-version` to upgrade to the latest chart version, or editing the `values.yaml` file to update the configuration):

    {{< copyable "shell-regular" >}}

    ```shell
    helm upgrade ${release_name} ${chart_name} --version=${chart_version} -f ${values_file}
    ```

* Delete:

    To delete the application deployed by Helm, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    helm del --purge ${release_name}
    ```

For more information on Helm, refer to [Helm Documentation](https://helm.sh/docs/).

#### Use Helm chart offline

If the server has no Internet access, you cannot configure the Helm repo to install the TiDB Operator component and other applications. At this time, you need to download the chart file needed for cluster installation on a machine with Internet access, and then copy it to the server.

Use the following command to download the chart file required for cluster installation:

{{< copyable "shell-regular" >}}

```shell
wget http://charts.pingcap.org/tidb-operator-v1.1.5.tgz
wget http://charts.pingcap.org/tidb-drainer-v1.1.5.tgz
wget http://charts.pingcap.org/tidb-lightning-v1.1.5.tgz
```

Copy these chart files to the server and decompress them. You can use these charts to install the corresponding components by running the `helm install` command. Take `tidb-operator` as an example:

{{< copyable "shell-regular" >}}

```shell
tar zxvf tidb-operator.v1.1.5.tgz
helm install ./tidb-operator --name=${release_name} --namespace=${namespace}
```

## Use Terraform

[Terraform](https://www.terraform.io/) is a Infrastructure as Code management tool. It enables users to define their own infrastructure in a  manifestation style, based on which execution plans are generated to create or schedule real world compute resources. TiDB in Kubernetes use Terraform to create and manage TiDB clusters on public clouds.

Follow the steps in [Terraform Documentation](https://www.terraform.io/downloads.html) to install Terraform.
