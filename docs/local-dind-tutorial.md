# Deploy TiDB to Kubernetes on Your Laptop

This document describes how to deploy a TiDB cluster to Kubernetes on your laptop (Linux or macOS) for development or testing.

[Docker in Docker](https://hub.docker.com/_/docker/) (DinD) runs Docker containers as virtual machines and runs another layer of Docker containers inside the first layer of Docker containers. [kubeadm-dind-cluster](https://github.com/kubernetes-sigs/kubeadm-dind-cluster) uses this technology to run the Kubernetes cluster in Docker containers. TiDB Operator uses a modified DinD script to manage the DinD Kubernetes cluster.

## Prerequisites

Before deploying a TiDB cluster to Kubernetes, make sure the following requirements are satisfied:

- Resources requirement: CPU 2+, Memory 4G+

    > **Note:** For macOS, you need to allocate 2+ CPU and 4G+ Memory to Docker. For details, see [Docker for Mac configuration](https://docs.docker.com/docker-for-mac/#advanced).

- [Docker](https://docs.docker.com/install/): 17.03 or later

    > **Note:** [Legacy Docker Toolbox](https://docs.docker.com/toolbox/toolbox_install_mac/) users must migrate to [Docker for Mac](https://store.docker.com/editions/community/docker-ce-desktop-mac) by uninstalling Legacy Docker Toolbox and installing Docker for Mac, because DinD cannot run on Docker Toolbox and Docker Machine.

    > **Note:** `kubeadm` validates installed Docker version during the installation process. If you are using Docker later than 18.06, there would be warning messages. The cluster might still be working, but it is recommended to use a Docker version between 17.03 and 18.06 for better compatibility.

- [Helm Client](https://github.com/helm/helm/blob/master/docs/install.md#installing-the-helm-client): 2.9.0 or later
- [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl): 1.10 at least, 1.13 or later recommended

    > **Note:** The outputs of different versions of `kubectl` might be slightly different.

- For Linux users, `kubeadm` might produce warning messages during the installation process if you are using kernel 5.x or later versions. The cluster might still be working, but it is recommended to use kernel version 3.10+ or 4.x for better compatibility.

- `root` access or permissions to operate with the Docker daemon.

## Step 1: Deploy a Kubernetes cluster using DinD

There is a script in our repository that can help you install and set up a Kubernetes cluster (version 1.12) using DinD for TiDB Operator.

```sh
$ git clone --depth=1 https://github.com/pingcap/tidb-operator
$ cd tidb-operator
$ manifests/local-dind/dind-cluster-v1.12.sh up
```

> **Note:** If the cluster fails to pull Docker images during the startup due to the firewall, you can set the environment variable `KUBE_REPO_PREFIX` to `uhub.ucloud.cn/pingcap` before running the script `dind-cluster-v1.12.sh` as follows (the Docker images used are pulled from [UCloud Docker Registry](https://docs.ucloud.cn/compute/uhub/index)):

```
$ KUBE_REPO_PREFIX=uhub.ucloud.cn/pingcap manifests/local-dind/dind-cluster-v1.12.sh up
```

## Step 2: Install TiDB Operator in the DinD Kubernetes cluster

```sh
$ # Install TiDB Operator into Kubernetes
$ helm install charts/tidb-operator --name=tidb-operator --namespace=tidb-admin --set scheduler.kubeSchedulerImageName=mirantis/hypokube --set scheduler.kubeSchedulerImageTag=final
$ # wait operator running
$ kubectl get pods --namespace tidb-admin -l app.kubernetes.io/instance=tidb-operator
NAME                                       READY     STATUS    RESTARTS   AGE
tidb-controller-manager-5cd94748c7-jlvfs   1/1       Running   0          1m
tidb-scheduler-56757c896c-clzdg            2/2       Running   0          1m
```

## Step 3: Deploy a TiDB cluster in the DinD Kubernetes cluster

```sh
$ helm install charts/tidb-cluster --name=demo --namespace=tidb
$ watch kubectl get pods --namespace tidb -l app.kubernetes.io/instance=demo -o wide
$ # wait a few minutes to get all TiDB components get created and ready

$ kubectl get tidbcluster -n tidb
NAME   PD                  STORAGE   READY   DESIRE   TIKV                  STORAGE   READY   DESIRE   TIDB                  READY   DESIRE
demo   pingcap/pd:v2.1.8   1Gi       3       3        pingcap/tikv:v2.1.8   10Gi      3       3        pingcap/tidb:v2.1.8   2       2

$ kubectl get statefulset -n tidb
NAME        DESIRED   CURRENT   AGE
demo-pd     3         3         1m
demo-tidb   2         2         1m
demo-tikv   3         3         1m

$ kubectl get service -n tidb
NAME              TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                          AGE
demo-discovery    ClusterIP   10.96.146.139    <none>        10261/TCP                        1m
demo-grafana      NodePort    10.111.80.73     <none>        3000:32503/TCP                   1m
demo-pd           ClusterIP   10.110.192.154   <none>        2379/TCP                         1m
demo-pd-peer      ClusterIP   None             <none>        2380/TCP                         1m
demo-prometheus   NodePort    10.104.97.84     <none>        9090:32448/TCP                   1m
demo-tidb         NodePort    10.102.165.13    <none>        4000:32714/TCP,10080:32680/TCP   1m
demo-tidb-peer    ClusterIP   None             <none>        10080/TCP                        1m
demo-tikv-peer    ClusterIP   None             <none>        20160/TCP                        1m

$ kubectl get configmap -n tidb
NAME                     DATA   AGE
demo-monitor             5      1m
demo-monitor-dashboard   0      1m
demo-pd                  2      1m
demo-tidb                2      1m
demo-tikv                2      1m

$ kubectl get pod -n tidb
NAME                              READY     STATUS      RESTARTS   AGE
demo-discovery-649c7bcbdc-t5r2k   1/1       Running     0          1m
demo-monitor-58745cf54f-gb8kd     2/2       Running     0          1m
demo-pd-0                         1/1       Running     0          1m
demo-pd-1                         1/1       Running     0          1m
demo-pd-2                         1/1       Running     0          1m
demo-tidb-0                       1/1       Running     0          1m
demo-tidb-1                       1/1       Running     0          1m
demo-tikv-0                       1/1       Running     0          1m
demo-tikv-1                       1/1       Running     0          1m
demo-tikv-2                       1/1       Running     0          1m
```

To access the TiDB cluster, use `kubectl port-forward` to expose services to the host. The port numbers in command are in `<host machine port>:<k8s service port>` format.

> **Note:** If you are deploying DinD on a remote machine rather than a local PC, there might be problems accessing "localhost" of that remote system. When you use `kubectl` 1.13 or later, it is possible to expose the port on `0.0.0.0` instead of the default `127.0.0.1` by adding `--address 0.0.0.0` to the `kubectl port-forward` command.

- Access TiDB using the MySQL client

    1. Use `kubectl` to forward the host machine port to the TiDB service port:

        ```sh
        $ kubectl port-forward svc/demo-tidb 4000:4000 --namespace=tidb
        ```

    2. To connect to TiDB using the MySQL client, open a new terminal tab or window and run the following command:

        ```sh
        $ mysql -h 127.0.0.1 -P 4000 -u root
        ```

- View the monitor dashboard

    1. Use `kubectl` to forward the host machine port to the Grafana service port:

        ```sh
        $ kubectl port-forward svc/demo-grafana 3000:3000 --namespace=tidb
        ```

    2. Open your web browser at http://localhost:3000 to access the Grafana monitoring interface.

        * Default username: admin
        * Default password: admin

- Permanent remote access

    Although this is a very simple demo cluster and does not apply to any serious usage, it is useful if it can be accessed remotely without `kubectl port-forward`, which might require an open terminal.

    TiDB, Prometheus, and Grafana are exposed as `NodePort` Services by default, so it is possible to set up a reverse proxy for them.

    1. Find their listing port numbers using the following command:

        ```sh
        $ kubectl get service -n tidb | grep NodePort
        demo-grafana      NodePort    10.111.80.73     <none>        3000:32503/TCP                   1m
        demo-prometheus   NodePort    10.104.97.84     <none>        9090:32448/TCP                   1m
        demo-tidb         NodePort    10.102.165.13    <none>        4000:32714/TCP,10080:32680/TCP   1m
        ```

        In this sample output, the ports are: 32503 for Grafana, 32448 for Prometheus, and 32714 for TiDB.

    2. Find the host IP addresses of the cluster.

        DinD is a K8s cluster running inside Docker containers, so Services expose ports to the containers' address, instead of the real host machine. We can find IP addresses of Docker containers by `kubectl get nodes -o yaml | grep address`.

    3. Set up a reverse proxy.

        Either (or all) of the container IPs can be used as upstream for a reverse proxy. You can use any reverse proxy server that supports TCP (for TiDB) or HTTP (for Grafana and Prometheus) to provide remote access. HAProxy and NGINX are two common choices.

## Scale the TiDB cluster

You can scale out or scale in the TiDB cluster simply by modifying the number of `replicas`.

1. Configure the `charts/tidb-cluster/values.yaml` file.

    For example, to scale out the cluster, you can modify the number of TiKV `replicas` from 3 to 5, or the number of TiDB `replicas` from 2 to 3.

2. Run the following command to apply the changes:

    ```sh
    helm upgrade demo charts/tidb-cluster --namespace=tidb
    ```

> **Note:** If you need to scale in TiKV, the consumed time depends on the volume of your existing data, because the data needs to be migrated safely.

## Upgrade the TiDB cluster

1. Configure the `charts/tidb-cluster/values.yaml` file.

    For example, change the version of PD/TiKV/TiDB `image` to `v2.1.9`.

2. Run the following command to apply the changes:

    ```sh
    helm upgrade demo charts/tidb-cluster --namespace=tidb
    ```

## Destroy the TiDB cluster

When you are done with your test, use the following command to destroy the TiDB cluster:

```sh
$ helm delete demo --purge
```

> **Note:** This only deletes the running pods and other resources, the data is persisted. If you do not need the data anymore, run the following commands to clean up the data. (Be careful, this permanently deletes the data).

```sh
$ kubectl get pv -l app.kubernetes.io/namespace=tidb -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
$ kubectl delete pvc --namespace tidb --all
```

## Stop and Re-start the Kubernetes cluster

* If you want to stop the DinD Kubernetes cluster, run the following command:

    ```sh
    $ manifests/local-dind/dind-cluster-v1.12.sh stop

    ```

* If you want to restart the DinD Kubernetes after you stop it, run the following command:

    ```
    $ manifests/local-dind/dind-cluster-v1.12.sh start
    ```

## Destroy the DinD Kubernetes cluster

If you want to clean up the DinD Kubernetes cluster and bring up a new cluster, run the following commands:

```sh
$ manifests/local-dind/dind-cluster-v1.12.sh clean
$ sudo rm -rf data/kube-node-*
$ manifests/local-dind/dind-cluster-v1.12.sh up
```

> **Warning:** You must clean the data after you destroy the DinD Kubernetes cluster, otherwise the TiDB cluster would fail to start when you try to bring it up again.
