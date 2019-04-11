# TiDB Operator Setup

## Requirements

Before deploying the TiDB Operator, make sure the following requirements are satisfied:

* Kubernetes v1.10 or greater
* [DNS addons](https://kubernetes.io/docs/tasks/access-application-cluster/configure-dns-cluster/)
* [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
* [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) enabled (optional)
* [Helm](https://helm.sh) v2.8.2 or greater
* Kubernetes v1.12 is required for zone-aware persistent volumes.

> **Note:** Allthough TiDB Operator can use network volume to persist TiDB data, this is slower due to redundant replication. It is highly recommended to set up [local volume](https://kubernetes.io/docs/concepts/storage/volumes/#local) for better performance.

> **Note:** Network volumes in a multi availability zone setup require Kubernetes v1.12 or greater. We do recommend using networked volumes for backup in the tidb-bakup chart.

## Kubernetes

TiDB Operator runs on top of Kubernetes cluster, you can use one of the methods listed [here](https://kubernetes.io/docs/setup/pick-right-solution/) to set up a Kubernetes cluster. Just make sure the Kubernetes cluster version is equal or greater than v1.10. If you want to use AWS, GKE or local machine, there are quick start tutorials:

* [Local DinD tutorial](./local-dind-tutorial.md)
* [Google GKE tutorial](./google-kubernetes-tutorial.md)
* [AWS EKS tutorial](./aws-eks-tutorial.md)

If you want to use a different envirnoment, a proper DNS addon must be installed in the Kubernetes cluster. You can follow the [official documentation](https://kubernetes.io/docs/tasks/access-application-cluster/configure-dns-cluster/) to set up a DNS addon.

TiDB Operator uses [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to persist TiDB cluster data (including the database, monitor data, backup data), so the Kubernetes must provide at least one kind of persistent volume. To achieve better performance, local SSD disk persistent volume is recommended. You can follow [this step](#local-persistent-volume) to auto provisioning local persistent volumes.

The Kubernetes cluster is suggested to enable [RBAC](https://kubernetes.io/docs/admin/authorization/rbac). Otherwise you may want to set `rbac.create` to `false` in the values.yaml of both tidb-operator and tidb-cluster charts.

Because TiDB by default will use lots of file descriptors, the [worker node](https://access.redhat.com/solutions/61334) and its Docker daemon's ulimit must be configured to greater than 1048576:

```shell
$ sudo vim /etc/systemd/system/docker.service
```

Set `LimitNOFILE` to equal or greater than 1048576.

Otherwise you have to change TiKV's `max-open-files` to match your work node `ulimit -n` in the configuration file `charts/tidb-cluster/templates/config/_tikv-config.tpl`, but this will impact TiDB performance.

## Helm

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

    If `RBAC` is not enabled for the Kubernetes cluster, then `helm init --upgrade` should be enough.

## Local Persistent Volume

Local disks are recommended to be formatted as ext4 filesystem. The local persistent volume directory must be a mount point: a whole disk mount or a [bind mount](https://unix.stackexchange.com/questions/198590/what-is-a-bind-mount):

### Disk mount

Mount local ssd disks of your Kubernetes nodes at subdirectory of /mnt/disks. For example if your data disk is `/dev/nvme0n1`, you can format and mount with the following commands:

```shell
$ sudo mkdir -p /mnt/disks/disk0
$ sudo mkfs.ext4 /dev/nvme0n1
$ sudo mount -t ext4 -o nodelalloc /dev/nvme0n1 /mnt/disks/disk0
```

### Bind mount

The disadvantages of bind mount for TiDB: all the volumes has the size of the whole disk and there is no quota and isolation of bind mount volumes. If your data directory is `/data`, you can create a bind mount with the following commands:

```shell
$ sudo mkdir -p /data/local-pv01
$ sudo mkdir -p /mnt/disks/local-pv01
$ sudo mount --bind /data/local-pv01 /mnt/disks/local-pv01
```

Use this command to confirm the mount point exist:

```shell
$ mount | grep /mnt/disks/local-pv01
```

### Auto-mount on reboot

To auto-mount disks when your operating system is booted, you should edit `/etc/fstab` to include these mounting info.

Disk mount:
```shell
$ echo "/dev/nvme0n1 /mnt/disks/disk01 ext4 defaults 0 0" >> /etc/fstab
```

Bind mount:
```shell
$ echo "/data/local-pv01 /mnt/disks/local-pv01 none bind 0 0" >> /etc/fstab
```

### Deploy local-static-provisioner

After mounting all data disks on Kubernetes nodes, you can deploy [local-volume-provisioner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner) to automatically provision the mounted disks as Local PersistentVolumes.

```shell
$ kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/local-dind/local-volume-provisioner.yaml
$ kubectl get po -n kube-system -l app=local-volume-provisioner
$ kubectl get pv | grep local-storage
```

### Remove a mount point

If we want to remove a mount point, first we need to `umount` the mount point, and then delete the related directories. For example:

```shell
$ umount /mnt/disks/local-pv01
$ rm -rf /mnt/disks/local-pv01 
$ rm -rf /data/local-pv01
```

> Note: The local-volume plugin expects paths to be stable, if you remove a previous mount-point in the discovery directory (default to `/mnt/disks/`), you should remove the PV manually to keep consistency.

## Install TiDB Operator

TiDB Operator uses [CRD](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to extend Kubernetes, so to use TiDB Operator, you should first create `TidbCluster` custom resource kind. This is a one-time job, namely you can only need to do this once in your Kubernetes cluster.

```shell
$ kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/manifests/crd.yaml
$ kubectl get crd tidbclusters.pingcap.com
```

After the `TidbCluster` custom resource is created, you can install TiDB Operator in your Kubernetes cluster.

Uncomment the `scheduler.kubeSchedulerImage` in `values.yaml`, set it to the same as your kubernetes cluster version.

```shell
$ git clone https://github.com/pingcap/tidb-operator.git
$ cd tidb-operator
$ helm install charts/tidb-operator --name=tidb-operator --namespace=tidb-admin
$ kubectl get po -n tidb-admin -l app.kubernetes.io/name=tidb-operator
```

## Custom TiDB Operator

Customizing is done by modifying `charts/tidb-operator/values.yaml`. The rest of the document will use `values.yaml` to reference `charts/tidb-operator/values.yaml`

TiDB Operator contains two components:

* tidb-controller-manager
* tidb-scheduler

This two components are stateless, so they are deployed via `Deployment`. You can customize the `replicas` and resource limits/requests as you wish in the values.yaml.

After editing values.yaml, run the following command to apply the modification:

```shell
$ helm upgrade tidb-operator charts/tidb-operator
```

## Upgrade TiDB Operator

Upgrading TiDB Operator itself is similar to customize TiDB Operator, modify the image version in values.yaml and then run `helm upgrade`:

```shell
$ helm upgrade tidb-operator charts/tidb-operator
```

When a new version of tidb-operator comes out, simply update the `operatorImage` in values.yaml and run the above command should be enough. But for safety reasons, you should get the new charts from tidb-operator repo and merge the old values.yaml with new values.yaml. And then upgrade as above.

TiDB Operator is for TiDB cluster maintenance, what this means is that when TiDB cluster is up and running, you can just stop TiDB Operator and TiDB cluster still works well unless you need to do TiDB cluster maintenance like scaling, upgrading etc.

## Upgrade Kubernetes

When you have a major version change of Kubernetes, you need to make sure that the kubeSchedulerImageTag matches it. By default, this value is generated by helm during install/upgrade so you need to perform a helm upgrade to reset it.

