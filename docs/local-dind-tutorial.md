This document describes how to deploy a TiDB cluster to Kubernetes on your laptop for development or testing.

Docker in Docker (DinD) runs Docker containers as virtual machines and runs another layer of Docker containers inside the first layer of Docker containers. [kubeadm-dind-cluster](https://github.com/kubernetes-sigs/kubeadm-dind-cluster) uses this technology to run the Kubernetes cluster in Docker containers.

# Prerequisites

Before deploying a TiDB cluster to Kubernetes, make sure the following requirements are satisfied:

* [Docker](https://docs.docker.com/install/) 17.03 or later
* [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl) 1.10 or later

# Deploy a Kubernetes cluster using DinD

1. Use DinD to install and deploy a multiple-node Kubernetes cluster:

```sh
$ wget https://cdn.rawgit.com/kubernetes-sigs/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.10.sh
$ chmod +x dind-cluster-v1.10.sh
$ NUM_NODES=4 ./dind-cluster-v1.10.sh up
```

*Note*: If you fail to pull the Docker images due to GFW, you can try the following method (the Docker images used are pulled from [UCloud Docker Registry](https://docs.ucloud.cn/compute/uhub/index)):

```sh
$ git clone https://github.com/pingcap/kubeadm-dind-cluster
$ cd kubeadm-dind-cluster
$ NUM_NODES=4 tools/multi_k8s_dind_cluster_manager.sh rebuild e2e-v1.10
```

2. After the DinD cluster bootstrap is done, use the following command to verify the kubernetes cluster is up and running:

```sh
$ kubectl get node,componentstatus
$ kubectl get po -n kube-system
```

3. Now the cluster is up and running, we need to install Kubernetes package manager [Helm](https://helm.sh) into the cluster.
Later we'll use Helm to deploy and manage TiDB Operator and TiDB clusters.

```sh
$ export os=linux # change `linux` to `darwin` if you use macOS
$ wget https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-${os}-amd64.tar.gz
$ tar xzf helm-v2.9.1-${os}-amd64.tar.gz
$ sudo mv ${os}-amd64/helm /usr/local/bin
$ kubectl apply -f manifests/tiller-rbac.yaml
$ helm init --service-account=tiller --upgrade
```

*Note*: If the tiller pod fail to start due to image pull failure because of GFW, you can replace the last command with `helm init --service-account=tiller --upgrade --tiller-image=uhub.ucloud.cn/pingcap/tiller:v2.9.1`

# Setup local volumes

We'll use [LocalPersistentVolume](https://kubernetes.io/docs/concepts/storage/volumes/#local) to persistent PD/TiKV data. The [local persistent volume provisioner](https://github.com/kubernetes-incubator/external-storage/tree/master/local-volume) doesn't work out of the box in DinD, we need to modify its deployment. And it doesn't support [dynamic provision](https://github.com/kubernetes/community/pull/1914) yet, we need to manually mount disks or directories to mount points. It's a little tricky so we provide some [scripts](../manifests/local-dind) to help setup the development environment.

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

# Install tidb-operator in DinD K8s

```sh
$ kubectl apply -f manifests/crd.yaml

$ # This creates the custom resource for the cluster that the operator uses.
$ kubectl get customresourcedefinitions
NAME                             AGE
tidbclusters.pingcap.com         1m

$ # Install TiDB Operator into K8s
$ helm install charts/tidb-operator --name=tidb-operator --namespace=pingcap

$ # wait operator running
$ kubectl get po -n pingcap -l app=tidb-operator
NAME                                       READY     STATUS    RESTARTS   AGE
tidb-controller-manager-3999554014-1h171   1/1       Running   0          1m
```

# Deploy TiDB clusters in DinD K8s

```sh
$ helm install charts/tidb-cluster --name=tidb-cluster --namespace=tidb

$ # wait a few minutes to get all TiDB components created and ready
$ kubectl get tidbcluster -n tidb
NAME           AGE
demo-cluster   1m

$ kubectl get statefulset -n tidb
NAME                DESIRED   CURRENT   AGE
demo-cluster-pd     3         3         1m
demo-cluster-tidb   2         2         1m
demo-cluster-tikv   3         3         1m

$ kubectl get service -n tidb
NAME                      TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                          AGE
demo-cluster-grafana      NodePort    10.105.174.22    <none>        3000:31479/TCP                   1m
demo-cluster-pd           ClusterIP   10.101.138.121   <none>        2379/TCP                         1m
demo-cluster-pd-peer      ClusterIP   None             <none>        2380/TCP                         1m
demo-cluster-prometheus   NodePort    10.108.106.219   <none>        9090:32303/TCP                   1m
demo-cluster-tidb         NodePort    10.104.195.140   <none>        4000:31155/TCP,10080:30595/TCP   1m
demo-cluster-tikv-peer    ClusterIP   None             <none>        20160/TCP                        1m

$ kubectl get configmap -n tidb
NAME                             DATA      AGE
demo-cluster-monitor-configmap   3         1m
demo-cluster-pd                  2         1m
demo-cluster-tidb                2         1m
demo-cluster-tikv                2         1m


$ kubectl get pod -n tidb
NAME                                       READY     STATUS      RESTARTS   AGE
demo-cluster-configure-grafana-724x5       0/1       Completed   0          1m
demo-cluster-pd-0                          1/1       Running     0          1m
demo-cluster-pd-1                          1/1       Running     0          1m
demo-cluster-pd-2                          1/1       Running     0          1m
demo-cluster-prometheus-6fd9b54dcf-b7q9w   2/2       Running     0          1m
demo-cluster-tidb-0                        1/1       Running     0          1m
demo-cluster-tidb-1                        1/1       Running     0          1m
demo-cluster-tikv-0                        2/2       Running     0          1m
demo-cluster-tikv-1                        2/2       Running     0          1m
demo-cluster-tikv-2                        2/2       Running     0          1m
```

To access TiDB cluster, we have to use `kubectl port-forward` to expose the services to host.

* Access TiDB using MySQL client

```sh
$ kubectl port-forward svc/demo-cluster-tidb 4000:4000 --namespace=tidb
$ mysql -h 127.0.0.1 -P 4000 -u root
```

* View monitor dashboard

```sh
$ kubectl port-forward svc/demo-cluster-grafana 3000:3000 --namespace=tidb
$ # then view the Grafana dashboard at http://localhost:3000 in your web browser
```

## Scale the TiDB cluster

First configure charts/tidb-cluster/values.yaml file, for example to change the number of TiKV `replicas` to 5 and TiDB `replicas` to 3. Then run the following command.

```sh
helm upgrade tidb-cluster charts/tidb-cluster --namespace=tidb
```

## Upgrade the TiDB cluster

First configure charts/tidb-cluster/values.yaml file, for example to change PD/TiKV/TiDB `image` version from v2.0.4 to v2.0.5. Then run the following command.

```sh
helm upgrade tidb-cluster charts/tidb-cluster --namespace=tidb
```
