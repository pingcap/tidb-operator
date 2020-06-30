---
title: Tools in Kubernetes
summary: Learn about operation tools for TiDB in Kubernetes.
category: reference
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

* **Local Mode**：In this mode, `tikv-ctl` accesses data files of TiKV, and the running TiKV instances must be stopped. To operate in the local mode, first you need to enter the [Diagnostic Mode](troubleshoot.md#use-the-diagnostic-mode) to turn off automatic re-starting for the TiKV instance, stop the TiKV process, and use the `tkctl debug` command to start in the target TiKV Pod a new container that contains the `tikv-ctl` executable. The steps are as follows:

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

1. Refer to [Helm official documentation](https://v2.helm.sh/docs/using_helm/#installing-helm) to install Helm client.

    If the server to install the Helm client cannot access the Internet, you have to download the Helm client package on a server or computer that has Internet access, and then upload it to the server. Take installing Helm client `2.16.7` for example:
 
   {{< copyable "shell-regular" >}}
 
   ```shell
   wget https://get.helm.sh/helm-v2.16.7-linux-amd64.tar.gz && \
   tar zxvf helm-v2.16.7-linux-amd64.tar.gz
   ```
 
   Uncompress the package and you will see the following files:
 
   ```shell
   linux-amd64/
   linux-amd64/README.md
   linux-amd64/tiller
   linux-amd64/helm
   linux-amd64/LICENSE
   ```
 
   Upload the `linux-amd64/helm` file to the server and put it in the `/usr/local/bin/` directory.
 
   Then execute `helm version -c`. If the command prints the output normally, it means that the Helm client has been installed successfully：
 
   {{< copyable "shell-regular" >}}
 
   ```shell
   helm version -c
   ```
 
   ```shell
   Client: &version.Version{SemVer:"v2.16.7", GitCommit:"5f2584fd3d35552c4af26036f0c464191287986b", GitTreeState:"clean"}
   ```

2. Install Helm server.

    Apply the `RBAC` rule required by the `tiller` component in the cluster and install `tiller`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.0/manifests/tiller-rbac.yaml && \
    helm init --service-account=tiller --upgrade
    ```

    If the server cannot access the Internet, you have to download the `tiller-rbac.yaml` file on a server or computer that has Internet access:

    {{< copyable "shell-regular" >}}

    ```shell
    wget https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.0/manifests/tiller-rbac.yaml
    ```

    Upload the `tiller-rbac.yaml` file to the server and then install `tiller`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f tiller-rbac.yaml
    helm init --service-account=tiller --skip-refresh
    ```

    The Helm server is a service named `tiller`, running as a Pod in the Kubernetes cluster. The Pod uses the image `gcr.io/kubernetes-helm/tiller:v2.16.7` by default.

    If the server cannot access `gcr.io`, try using the mirror repository:

    {{< copyable "shell-regular" >}}

    ``` shell
    helm init --service-account=tiller --upgrade --tiller-image registry.cn-hangzhou.aliyuncs.com/google_containers/tiller:$(helm version --client --short | grep -Eo 'v[0-9]\.[0-9]+\.[0-9]+')
    ```

    If the server cannot access the Internet, or it can access neither the `gcr.io` nor the `registry.cn-hangzhou.aliyuncs.com`, you have to download the image of `tiller` on a server or computer that has Internet access:

    {{< copyable "shell-regular" >}}

    ``` shell
    docker pull gcr.io/kubernetes-helm/tiller:v2.16.7 && \
    docker save -o tiller-v2.16.7.tar gcr.io/kubernetes-helm/tiller:v2.16.7
    ```

    Upload the `tiller-v2.16.7.tar` file to the server and execute the `docker load` command to load the image to the server:

    {{< copyable "shell-regular" >}}

    ``` shell
    docker load -i tiller-v2.16.7.tar
    ```

    Confirm that the tiller pod is in the `running` state by the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n kube-system -l name=tiller
    ```

    If `RBAC` is not enabled for the Kubernetes cluster, use the following command to install `tiller`:

    {{< copyable "shell-regular" >}}

    ```shell
    helm init --upgrade
    ```

3. Configure the Helm Repository

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

    After adding the repository, use `helm search` to search for the charts provided by PingCAP:

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

    When a new version of chart has been released, you can use `helm repo update` to update the repository cached locally:

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    Common Helm operations include `helm install`, `helm upgrade`, `helm del`, and `helm ls`. The Helm chart usually contains many configurable parameters which could be tedious to configure manually. For convenience, it is recommended that you configure these parameters using a YAML file. Based on the conventions in the Helm community, the YAML file used for Helm configuration is named `values.yaml` in this document.

    Before the operations of installation, upgrade, deletion, and so on, you can execute `helm ls` to view the applications that have been installed in the cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    helm ls
    ```

    When performing a deployment or upgrade, you must specify the chart name (`chart-name`) and the name for the deployed application (`release-name`). You can also specify one or multiple `values.yaml` files to configure charts. In addition, you can specify `chart-version` by using the `--version` flag to choose a specific chart version (by default the latest GA is used). The steps in command line are as follows:

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

4. Use Helm chart offline

    If the server cannot access the Internet, you cannot install TiDB Operator or other applications through configuring the Helm repository. In this case, you have to download the Helm charts required during the cluster installation on a server or computer that has Internet access, and then upload them to the server.
    
    Execute the following commands to download the charts required during the cluster installation:

    {{< copyable "shell-regular" >}}

    ```shell
    wget http://charts.pingcap.org/tidb-operator-v1.1.0.tgz && \
    wget http://charts.pingcap.org/tidb-drainer-v1.1.0.tgz && \
    wget http://charts.pingcap.org/tidb-lightning-v1.1.0.tgz
    ```

    Upload the charts to the server and uncompress them. You can install the components accordingly through the `helm install` command. Take `tidb-operator` for example:

    {{< copyable "shell-regular" >}}

    ```shell
    tar zxvf tidb-operator.v1.1.0.tgz && \
    helm install ./tidb-operator --name=${release_name} --namespace=${namespace}
    ```

## Use Terraform

[Terraform](https://www.terraform.io/) is a Infrastructure as Code management tool. It enables users to define their own infrastructure in a  manifestation style, based on which execution plans are generated to create or schedule real world compute resources. TiDB in Kubernetes use Terraform to create and manage TiDB clusters on public clouds.

Follow the steps in [Terraform Documentation](https://www.terraform.io/downloads.html) to install Terraform.
