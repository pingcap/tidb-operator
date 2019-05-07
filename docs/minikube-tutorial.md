# Deploy TiDB in the minikube cluster

This document describes how to deploy a TiDB cluster in the [minikube](https://kubernetes.io/docs/setup/minikube/) cluster.

## Table of Contents

- [Start a Kubernetes cluster with minikube](#start-a-kubernetes-cluster-with-minikube)
  * [What is minikube?](#what-is-minikube)
  * [Install minikube and start a Kubernetes cluster](#install-minikube-and-start-a-kubernetes-cluster)
  * [Install kubectl to access the cluster](#install-kubectl-to-access-the-cluster)
- [Install TiDB operator and run a TiDB cluster with it](#install-tidb-operator-and-run-a-tidb-cluster-with-it)
  * [Install helm](#install-helm)
  * [Install TiDB operator in the Kubernetes cluster](#install-tidb-operator-in-the-kubernetes-cluster)
  * [Launch a TiDB cluster](#launch-a-tidb-cluster)
  * [Test TiDB cluster](#test-tidb-cluster)
  * [Delete TiDB cluster](#delete-tidb-cluster)
- [FAQs](#faqs)
  * [TiDB cluster in minikube is not responding or responds slow](#tidb-cluster-in-minikube-is-not-responding-or-responds-slow)

## Start a Kubernetes cluster with minikube

### What is minikube?

[Minikube](https://kubernetes.io/docs/setup/minikube/) can start a local
Kubernetes cluster inside a VM on your laptop. It works on macOS, Linux, and
Windows.

> **Note:** Although Minikube supports `--vm-driver=none` that uses host docker instead of VM, it is not fully tested with TiDB Operator and may not work. If you want to try TiDB Operator on a system without virtualization support (e.g., on a VPS), you may consider using [DinD](local-dind-tutorial.md) instead.

### Install minikube and start a Kubernetes cluster

See [Installing Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) to install
minikube (1.0.0+) on your machine.

After you installed minikube, you can run the following command to start a
Kubernetes cluster.

```
minikube start
```

For Chinese mainland users, you may use local gcr.io mirrors such as
`registry.cn-hangzhou.aliyuncs.com/google_containers`.

```
minikube start --image-repository registry.cn-hangzhou.aliyuncs.com/google_containers
```

or configure HTTP/HTTPS proxy environments in your docker, e.g.

```
# change 127.0.0.1:1086 to your http/https proxy server IP:PORT
minikube start --docker-env https_proxy=http://127.0.0.1:1086 \
  --docker-env http_proxy=http://127.0.0.1:1086
```

> **Note:** If you are running minikube with VMs (default), the `127.0.0.1` is the VM itself, you might want to use your real IP address of the host machine in some cases.

See [minikube setup](https://kubernetes.io/docs/setup/minikube/) for more options to
configure your virtual machine and Kubernetes cluster.

### Install kubectl to access the cluster

The Kubernetes command-line tool,
[kubectl](https://kubernetes.io/docs/user-guide/kubectl/), allows you to run
commands against Kubernetes clusters.

Install kubectl according to the instructions in [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

After kubectl is installed, test your minikube Kubernetes cluster:

```
kubectl cluster-info
```

## Install TiDB operator and run a TiDB cluster with it

### Install helm

Helm is the package manager for Kubernetes and is what allows us to install all of the distributed components of TiDB in a single step. Helm requires both a server-side and a client-side component to be installed.

```
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get | bash
```

Install helm tiller:

```
helm init
```

If you have limited access to gcr.io, you can try a mirror, e.g.

```
helm init --upgrade --tiller-image registry.cn-hangzhou.aliyuncs.com/google_containers/tiller:$(helm version --client --short | grep -P -o 'v\d+\.\d+\.\d')
```

Once it is installed, running `helm version` should show you both the client
and server version, e.g.

```
$ helm version
Client: &version.Version{SemVer:"v2.13.1",
GitCommit:"618447cbf203d147601b4b9bd7f8c37a5d39fbb4", GitTreeState:"clean"}
Server: &version.Version{SemVer:"v2.13.1",
GitCommit:"618447cbf203d147601b4b9bd7f8c37a5d39fbb4", GitTreeState:"clean"}
```

If it shows only the client version, `helm` cannot yet connect to the server. Use
`kubectl` to see if any tiller pods are running.

```
kubectl -n kube-system get pods -l app=helm
```

### Install TiDB operator in the Kubernetes cluster

Clone tidb-operator repository:

```
git clone --depth=1 https://github.com/pingcap/tidb-operator
cd tidb-operator
kubectl apply -f ./manifests/crd.yaml
helm install charts/tidb-operator --name tidb-operator --namespace tidb-admin
```

Now, we can watch the operator come up with:

```
watch kubectl get pods --namespace tidb-admin -o wide
```

If you have limited access to gcr.io (pods failed with ErrImagePull), you can
try a mirror of kube-scheduler image. You can upgrade tidb-operator like this:

```
helm upgrade tidb-operator charts/tidb-operator --namespace tidb-admin --set \
  scheduler.kubeSchedulerImageName=registry.cn-hangzhou.aliyuncs.com/google_containers/kube-scheduler
```

When you see both tidb-scheduler and tidb-controller-manager are running, you
can process to launch a TiDB cluster!

### Launch a TiDB cluster

```
helm install charts/tidb-cluster --name tidb-cluster --set \
  schedulerName=default-scheduler,pd.storageClassName=standard,tikv.storageClassName=standard,pd.replicas=1,tikv.replicas=1,tidb.replicas=1
```

Watch tidb-cluster up and running:

```
watch kubectl get pods --namespace default -l app.kubernetes.io/instance=tidb-cluster -o wide
```

### Test TiDB cluster

There can be a small delay between the pod is up and running, and the service
is available. You can watch list services available with:

```
watch kubectl get svc
```

When you see `tidb-cluster-tidb` appear, it's ready to connect to TiDB server.

After first, forward local port to tidb port:

```
kubectl port-forward svc/tidb-cluster-tidb 4000:4000
```

In another terminal, connect to TiDB server with MySQL client:

```
mysql -h 127.0.0.1 -P 4000 -uroot
```

Or run SQL command directly:

```
mysql -h 127.0.0.1 -P 4000 -uroot -e 'select tidb_version();'
```

### Delete TiDB cluster

```
helm delete --purge tidb-cluster

# update reclaim policy of PVs used by tidb-cluster to Delete
kubectl get pv -l app.kubernetes.io/instance=tidb-cluster -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'

# delete PVCs
kubectl delete pvc -l app.kubernetes.io/managed-by=tidb-operator
```

## FAQs

### TiDB cluster in minikube is not responding or responds slow

The minikube VM is configured by default to only use 2048MB of memory and 2
CPUs. You can pass more during `minikube start` with the `--memory` and `--cpus` flag.
Note that you'll need to recreate minikube VM for this to take effect.

```
minikube delete
minikube start --cpus 4 --memory 4096 ...
```
