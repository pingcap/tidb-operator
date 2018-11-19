# TiDB Operator User Guide

This document serves as a generic user guide for TiDB Operator. Basic understanding of [Kubernetes](https://k8s.io) and [TiDB](https://github.com/pingcap/tidb) is required. For quick start, please reference one of the following tutorials:

* [Local DinD tutorial](./local-dind-tutorial.md)
* [Google GKE tutorial](./google-kubernetes-tutorial.md)
* [AWS EKS tutorial](./aws-eks-tutorial.md)

## Requirements

Before deploying the TiDB Operator, make sure the following requirements are satisfied:

* Kubernetes v1.10 or later
* [Helm](https://helm.sh) v2.8.2 or later
* [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
* [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) enabled (optional)

> **Note:** Though TiDB Operator can use network volume to persist TiDB data, it is highly recommended to set up [local volume](https://kubernetes.io/docs/concepts/storage/volumes/#local) for better performance. Because TiDB already replicates data, network volume will add extra replicas which is redundant.

### Helm

You can follow Helm official [documentation](https://helm.sh) to install Helm in your Kubernetes cluster. The following instructions are listed here for quick reference:

1. Install helm client

    ```
    $ curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | bash
    ```

    Or if you use macOS, you can use homebrew to install Helm by `brew install kubernetes-helm`

2. Install helm server

    ```shell
    $ kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/tiller-rbac.yaml
    $ helm init --service-account=tiller --upgrade
    $ kubectl get po -n kube-system -l name=tiller # make sure tiller pod is running
    ```

### Local Persistent Volume

Local disks are recommended to be formatted as ext4 filesystem.

Mount local ssd disks of your Kubernetes nodes at subdirectory of /mnt/disks. For example if your data disk is `/dev/nvme0n1`, you can format and mount with the following commands:

```shell
$ sudo mkdir -p /mnt/disks/disk0
$ sudo mkfs.ext4 /dev/nvme0n1
$ sudo mount -t ext4 -o nodelalloc /dev/nvme0n1 /mnt/disks/disk0
```

To auto-mount disks when your operating system is booted, you should edit `/etc/fstab` to include these mounting info.

After mounting all data disks on Kubernetes nodes, you can deploy [local-volume-provisioner](https://github.com/kubernetes-incubator/external-storage/tree/master/local-volume) to automatically provision the mounted disks as Local PersistentVolumes.

```shell
$ kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/local-dind/local-volume-provisioner.yaml
$ kubectl get po -n kube-system -l app=local-volume-provisioner
$ kubectl get pv | grep local-storage
```

## Install TiDB Operator

```shell
$ kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml
$ helm install charts/tidb-operator --name=tidb-operator --namespace=tidb-admin
$ kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
```

## Manage TiDB cluster

TiDB Operator can manage multiple clusters in the same Kubernetes cluster. Clusters are qualified by `namespace` and `clusterName`, namely different clusters may have same `namespace` or `clusterName` but not both.

The default `clusterName` is `demo` which is defined in charts/tidb-cluster/values.yaml. The following variables will be used in the rest of the document:

```shell
$ releaseName="tidb-cluster"
$ namespace="tidb"
$ clusterName="demo" # Make sure this is the same as variable defined in charts/tidb-cluster/values.yaml
```

### Deploy TiDB cluster

After TiDB Operator and Helm is deployed correctly, TiDB cluster can be deployed using following command:

```shell
$ helm install charts/tidb-cluster --name=${releaseName} --namespace=${namespace}
$ kubectl get po -n ${namespace} -l app.kubernetes.io/name=tidb-operator
```

For customized deployment, you can modify `charts/tidb-cluster/values.yaml` before installing the charts. Most of the variables are self-explanatory with comments.

### Access TiDB cluster

By default TiDB service is exposed using [`NodePort`](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport). You can modify it to `ClusterIP` which will disable access from outside of the cluster. Or modify it to [`LoadBalancer`](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) if the underlining Kubernetes supports this kind of service.

By default TiDB cluster is deployed with a random generated password. You can specify a password by setting `tidb.password` in charts/tidb-cluster/values.yaml before deploying. Whether you specify the password or not, you can retrieve the password through `Secret`:

```shell
$ PASSWORD=$(kubectl get secret -n ${namespace} ${clusterName}-tidb -ojsonpath="{.data.password}" base64 -c | awk '{print $6}')
$ echo ${PASSWORD}
$ kubectl get svc -n ${namespace} # check the available services
```

* Access inside of the Kubernetes cluster

    When your application is deployed in the same Kubernetes cluster, you can access TiDB via domain name `demo-tidb.tidb.svc` with port `4000`. Here `demo` is the `clusterName` which can be modified in charts/tidb-cluster/values.yaml. And the latter `tidb` is the namespace you specified when using `helm install` to deploy TiDB cluster.

* Access outside of the Kubernetes cluster

    * Using kubectl portforward

        ```shell
        $ kubectl port-forward -n ${namespace} svc/${clusterName}-tidb 4000:4000 &>/tmp/portforward-tidb.log
        $ mysql -h 127.0.0.1 -P 4000 -u root -p
        ```

    * Using LoadBalancer

        When you set `tidb.service.type` to `LoadBalancer` and the underlining Kubernetes support LoadBalancer, then a LoadBalancer will be created for TiDB service. You can access it via the external IP with port `4000`. Some cloud platforms support internal load balancer via service annotations, for example you can add annotation `cloud.google.com/load-balancer-type: Internal` to `tidb.service.annotations` to create an internal load balancer for TiDB on GKE.

    * Using NodePort

        You can access TiDB via any node's IP with tidb service node port. The node port is the port after `4000`, usually greater than `30000`.

### Scale TiDB cluster

To scale TiDB cluster, just modify the `replicas` of PD, TiKV and TiDB. And then run the following command:

```shell
$ helm upgrade ${releaseName} charts/tidb-cluster
```

### Upgrade TiDB cluster

Upgrade TiDB cluster is similar to scale TiDB cluster, but by changing `image` of PD, TiKV and TiDB to different image versions. And then run the following command:

```shell
$ helm upgrade ${releaseName} charts/tidb-cluster
```

### Destroy TiDB cluster

To destroy TiDB cluster, run the following command:

```shell
$ helm delete ${releaseName} --purge
```

The above command only delete the running pods, the data is persistent. If you do not need the data anymore, you can run the following command to clean the data:

```shell
$ kubectl delete pvc -n ${namespace} -l app.kubernetes.io/instance=${releaseName}
$ kubectl get pv -l app.kubernetes.io/namespace=${namespace} -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

> **Note:** the above command will delete the data permanently. Think twice before executing them.

### Monitor

TiDB cluster is monitored with Prometheus and Grafana. When TiDB cluster is created, a Prometheus and Grafana pod will be created and configured to scrape and visualize metrics.

By default the monitor data is not persistent, when the monitor pod is killed for some reason, the data will be lost. This can be avoided by specifying `monitor.persistent` to `true` which will create a persistent volume for storing monitor data.

You can view the dashboard using `kubectl portforward`:

```shell
$ kubectl port-forward -n ${namespace} svc/${clusterName}-grafana 3000:3000 &>/tmp/portforward-grafana.log
```

Then open your browser at http://localhost:3000 The default username and password are both `admin`

The Grafana service is exposed as `NodePort` by default, you can change it to `LoadBalancer` if the underlining Kubernetes has load balancer support. And then view the dashboard via load balancer endpoint.

### Backup

Currently, TiDB Operator supports two kinds of backup: full backup via [Mydumper](https://github.com/maxbube/mydumper) and incremental backup via binlog.

#### Full backup

Full backup can be done periodically just like crontab job. Currently, full backup requires a PersistentVolume, the backup job will create a PVC to store backup data.

To create a full backup job, modify charts/tidb-cluster/values.yaml `fullbackup` section.

* `create` must be set to `true`
* Set `storageClassName` to the PV storage class name used for backup data
* `schedule` takes the [Cron](https://en.wikipedia.org/wiki/Cron) format
* `user` and `password` must be set to the correct user which has the permission to read the database to be backuped.

If TiDB cluster is running on GKE, the backup data can be uploaded to GCS bucket. A bucket name and base64 encoded service account credential that has bucket read/write access must be provided. The comments in charts/tidb-cluster/values.yaml is self-explanatory for GCP backup.

#### Incremental backup

To enable incremental backup, set `binlog.pump.create` and `binlog.drainer.create` to `true`. By default the incremental backup data is stored in protobuffer format in a PV. You can change `binlog.drainer.destDBType` from `pb` to `mysql` or `kafka` and configure the corresponding downstream.

### Restore

Currently, tidb-operator only supports restoring from full backup in GCS bucket. The `restore` section in charts/tidb-cluster/values.yaml should have enough comments as document.

## Troubleshooting

### Some pods are pending for a long time

When a pod is pending, it means the required resources are not satisfied. The most common cases are:

* CPU, memory or storage insufficient

  Check the detail info of the pod by:

  ```shell
  $ kubectl describe po -n <ns> <pod-name>
  ```

  When this happens, either reduce the resource requests of the TiDB cluster and then using `helm` to upgrade the cluster. If the storage request is larger than any of the available volumes, you have to delete the pod and corresponding pending PVC.

* Storage class not exist or no PV available

  You can check this by:

  ```shell
  $ kubectl get pvc -n <ns>
  $ kubectl get pv | grep <storage-class-name> | grep Available
  ```

  When this happens, you can change the `storageClassName` and then using `helm` to upgrade the cluster. After that, delete the pending pods and the corresponding pending PVC and waiting new pod and pvc to be created.
