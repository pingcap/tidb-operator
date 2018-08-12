# Deploy TiDB to Kubernetes on Your Laptop

This document describes how to deploy a TiDB cluster to Kubernetes on your laptop for development or testing.

Docker in Docker (DinD) runs Docker containers as virtual machines and runs another layer of Docker containers inside the first layer of Docker containers. [kubeadm-dind-cluster](https://github.com/kubernetes-sigs/kubeadm-dind-cluster) uses this technology to run the Kubernetes cluster in Docker containers.

## Prerequisites

Before deploying a TiDB cluster to Kubernetes, make sure the following requirements are satisfied:

* [Docker](https://docs.docker.com/install/) 17.03 or later
* [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl) 1.10 or later

> **Note:** The outputs of different versions of `kubectl` might be slightly different.

## Step 1: Deploy a Kubernetes cluster using DinD

1. Use DinD to install and deploy a multiple-node Kubernetes cluster:

    ```sh
    $ wget https://cdn.rawgit.com/kubernetes-sigs/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.10.sh
    $ chmod +x dind-cluster-v1.10.sh
    $ NUM_NODES=4 ./dind-cluster-v1.10.sh up
    ```

    > **Note:** If you fail to pull the Docker images due to GFW, you can try the following method (the Docker images used are pulled from [UCloud Docker Registry](https://docs.ucloud.cn/compute/uhub/index)):

    ```sh
    $ git clone https://github.com/pingcap/kubeadm-dind-cluster
    $ cd kubeadm-dind-cluster
    $ NUM_NODES=4 tools/multi_k8s_dind_cluster_manager.sh rebuild e2e-v1.10
    ```

2. After the DinD cluster bootstrap is done, use the following command to verify the Kubernetes cluster is up and running:

    ```sh
    $ kubectl get node,componentstatus
    $ kubectl get po -n kube-system
    ```

3. Now the cluster is up and running, you need to install the Kubernetes package manager [Helm](https://helm.sh) into the cluster, which is used to deploy and manage TiDB Operator and TiDB clusters later.

    ```sh
    $ os=linux # change `linux` to `darwin` if you use macOS
    $ wget "https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-${os}-amd64.tar.gz"
    $ tar xzf helm-v2.9.1-${os}-amd64.tar.gz
    $ sudo mv ${os}-amd64/helm /usr/local/bin

    $ git clone git@github.com:pingcap/tidb-operator.git
    $ cd tidb-operator
    $ kubectl apply -f manifests/tiller-rbac.yaml
    $ helm init --service-account=tiller --upgrade
    $ helm version # verify the Helm server is running
    ```

    > **Note:** If the tiller pod fails to start due to image pull failure because of GFW, you can replace the last command with `helm init --service-account=tiller --upgrade --tiller-image=uhub.ucloud.cn/pingcap/tiller:v2.9.1`

## Step 2: Configure local volumes

[LocalPersistentVolume](https://kubernetes.io/docs/concepts/storage/volumes/#local) is used to persist the PD/TiKV data. The [local persistent volume provisioner](https://github.com/kubernetes-incubator/external-storage/tree/master/local-volume) doesn't work out of the box in DinD, so you need to modify its deployment. And it doesn't support [dynamic provision](https://github.com/kubernetes/community/pull/1914) yet, so you need to manually mount disks or directories to mount points. To simplify this operation, there are some [scripts](../manifests/local-dind) to help configure the development environment.

```sh
$ # create directories for local volumes
$ ./manifests/local-dind/pv-hosts.sh
$ # deploy local volume provisioner
$ kubectl apply -f manifests/local-dind/local-volume-provisioner.yaml
$ # wait local-volume-provisioner pods running
$ kubectl get po -n kube-system -l app=local-volume-provisioner
$ # verify pv created
$ kubectl get pv
```

## Step 3: Install tidb-operator in the DinD Kubernetes cluster

```sh
$ kubectl apply -f manifests/crd.yaml

$ # This creates the custom resource for the cluster that the operator uses.
$ kubectl get customresourcedefinitions
NAME                             AGE
tidbclusters.pingcap.com         1m

$ # Install TiDB Operator into Kubernetes
$ helm install charts/tidb-operator --name=tidb-operator --namespace=pingcap

$ # wait operator running
$ kubectl get po -n pingcap -l app=tidb-operator
NAME                                       READY     STATUS    RESTARTS   AGE
tidb-controller-manager-5cd94748c7-jlvfs   1/1       Running   0          1m
```

## Step 4: Deploy TiDB clusters in the DinD Kubernetes cluster

```sh
$ helm install charts/tidb-cluster --name=tidb-cluster --namespace=tidb

$ # wait a few minutes to get all TiDB components created and ready
$ kubectl get tidbcluster -n tidb
NAME      AGE
demo      3m

$ kubectl get statefulset -n tidb
NAME        DESIRED   CURRENT   AGE
demo-pd     3         3         1m
demo-tidb   2         2         1m
demo-tikv   3         3         1m

$ kubectl get service -n tidb
NAME              TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                          AGE
demo-grafana      NodePort    10.111.80.73     <none>        3000:32503/TCP                   1m
demo-pd           ClusterIP   10.110.192.154   <none>        2379/TCP                         1m
demo-pd-peer      ClusterIP   None             <none>        2380/TCP                         1m
demo-prometheus   NodePort    10.104.97.84     <none>        9090:32448/TCP                   1m
demo-tidb         NodePort    10.102.165.13    <none>        4000:32714/TCP,10080:32680/TCP   1m
demo-tikv-peer    ClusterIP   None             <none>        20160/TCP                        1m

$ kubectl get configmap -n tidb
NAME           DATA      AGE
demo-monitor   3         1m
demo-pd        2         1m
demo-tidb      2         1m
demo-tikv      2         1m

$ kubectl get pod -n tidb
NAME                              READY     STATUS      RESTARTS   AGE
demo-monitor-58745cf54f-gb8kd     2/2       Running     0          1m
demo-monitor-configurator-stvw6   0/1       Completed   0          1m
demo-pd-0                         1/1       Running     0          1m
demo-pd-1                         1/1       Running     0          1m
demo-pd-2                         1/1       Running     0          1m
demo-tidb-0                       1/1       Running     0          1m
demo-tidb-1                       1/1       Running     0          1m
demo-tikv-0                       2/2       Running     0          1m
demo-tikv-1                       2/2       Running     0          1m
demo-tikv-2                       2/2       Running     0          1m
```

To access the TiDB cluster, use `kubectl port-forward` to expose the services to host.

* Access TiDB using the MySQL client

    1. Use `kubectl` to forward the host machine port to the TiDB service port:

        ```sh
        $ kubectl port-forward svc/demo-tidb 4000:4000 --namespace=tidb
        ```

    2. To connect to TiDB using the MySQL client, open a new terminal tab or window and run the following command:

        ```sh
        $ mysql -h 127.0.0.1 -P 4000 -u root
        ```

* View the monitor dashboard

    1. Use `kubectl` to forward the host machine port to the Grafana service port:

        ```sh
        $ kubectl port-forward svc/demo-grafana 3000:3000 --namespace=tidb
        ```

    2. Open your web browser at http://localhost:3000 to access the Grafana monitoring interface.

        * Default username: admin
        * Default password: admin

## Scale the TiDB cluster

First configure the `charts/tidb-cluster/values.yaml` file, for example, change the number of TiKV `replicas` to 5 and TiDB `replicas` to 3. Then run the following command:

```sh
helm upgrade tidb-cluster charts/tidb-cluster --namespace=tidb
```

> **Note:** Scale down TiKV needs to migrate data, so the scaling down time depends on your existing data.

## Upgrade the TiDB cluster

First configure the `charts/tidb-cluster/values.yaml` file, for example, changing PD/TiKV/TiDB `image` version from v2.0.4 to v2.0.5. Then run the following command:

```sh
helm upgrade tidb-cluster charts/tidb-cluster --namespace=tidb
```

## Destroy the TiDB cluster

When you're done with your test, use following commands to destroy the TiDB cluster:

```sh
$ helm delete tidb-cluster --purge
```

> **Note:** This only deletes running pods and other resources, the data is persisted. If you don't need the data anymore, run the following commands to cleanup the data. (Be careful, this permanently delete the data).

```sh
$ kubectl get pv -l cluster.pingcap.com/namespace=tidb -o name | xargs -i kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
$ kubectl delete pvc --namespace tidb --all
```

## Destroy the DinD Kubernetes cluster

Change to the directory where you put dind-cluster-v1.10.sh

```sh
$ ./dind-cluster-v1.10.sh clean
```
