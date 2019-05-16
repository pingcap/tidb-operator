# TiDB Cluster Operation Guide

TiDB Operator can manage multiple clusters in the same Kubernetes cluster.

The following variables will be used in the rest of the document:

```shell
$ releaseName="demo"
$ namespace="tidb"
```

> **Note:** The rest of the document will use `values.yaml` to reference `charts/tidb-cluster/values.yaml`

## Deploy TiDB cluster

After TiDB Operator and Helm are deployed correctly, TiDB cluster can be deployed using following command:

```shell
$ helm install charts/tidb-cluster --name=${releaseName} --namespace=${namespace}
$ kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${releaseName}
```

The default deployment doesn't set CPU and memory requests or limits for any of the pods, and the storage used is `local-storage` with minimal size. These settings can make TiDB cluster run on a small Kubernetes cluster like DinD or the default GKE cluster for testing. But for production deployment, you would likely to adjust the cpu, memory and storage resources according to the [recommendations](https://github.com/pingcap/docs/blob/master/op-guide/recommendation.md).

The resource limits should be equal or bigger than the resource requests, it is suggested to set limit and request equal to get [`Guaranteed` QoS]( https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed).

For other settings, the variables in `values.yaml` are self-explanatory with comments. You can modify them according to your need before installing the charts.

## Access TiDB cluster

By default TiDB service is exposed using [`NodePort`](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport). You can modify it to `ClusterIP` which will disable access from outside of the cluster. Or modify it to [`LoadBalancer`](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) if the underlining Kubernetes supports this kind of service.

```shell
$ kubectl get svc -n ${namespace} # check the available services
```

By default the TiDB cluster has no root password set. Setting a password in helm is insecure. Instead you can set the name of a K8s secret as `tidb.passwordSecretName` in `values.yaml`. Note that this is only used to initialize users: once your tidb cluster is initialized you may delete the secret. The format of the secret is `user=password`, so you can set the root user password with:

```
kubectl create namespace ${namespace}
kubectl create secret generic tidb-secret --from-literal=root=<root-password> --namespace=${namespace}
```

You can retrieve the password from the initialization `Secret`:

```shell
$ PASSWORD=$(kubectl get secret -n ${namespace} tidb-secret -ojsonpath="{.data.root}" | base64 --decode)
$ echo ${PASSWORD}
```

* Access inside of the Kubernetes cluster

    When your application is deployed in the same Kubernetes cluster, you can access TiDB via domain name `demo-tidb.tidb.svc` with port `4000`. Here `demo` is the `releaseName`. And the latter `tidb` is the namespace you specified when using `helm install` to deploy TiDB cluster.

* Access outside of the Kubernetes cluster

    * Using kubectl portforward

        ```shell
        $ kubectl port-forward -n ${namespace} svc/${releaseName}-tidb 4000:4000 &>/tmp/portforward-tidb.log
        $ mysql -h 127.0.0.1 -P 4000 -u root -p
        ```

    * Using LoadBalancer

        When you set `tidb.service.type` to `LoadBalancer` and the underlining Kubernetes support LoadBalancer, then a LoadBalancer will be created for TiDB service. You can access it via the external IP with port `4000`. Some cloud platforms support internal load balancer via service annotations, for example you can add annotation `cloud.google.com/load-balancer-type: Internal` to `tidb.service.annotations` to create an internal load balancer for TiDB on GKE.

    * Using NodePort

        You can access TiDB via any node's IP with tidb service node port. The node port is the port after `4000`, usually greater than `30000`.

## Scale TiDB cluster

TiDB Operator supports both horizontal and vertical scaling, but there are some caveats for storage vertical scaling.

* Kubernetes is v1.11 or later, please reference [the official blog](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/)
* Backend storage class supports resizing. (Currently only a limited of network storage class supports resizing)

When using local persistent volumes, even CPU and memory vertical scaling can cause problems because there may be not enough resources on the node.

Due to the above reasons, it's recommended to do horizontal scaling other than vertical scaling when workload increases.

### Horizontal scaling

To scale in/out TiDB cluster, just modify the `replicas` of PD, TiKV and TiDB in `values.yaml` file. And then run the following command:

```shell
$ helm upgrade ${releaseName} charts/tidb-cluster
```

### Vertical scaling

To scale up/down TiDB cluster, modify the cpu/memory/storage limits and requests of PD, TiKV and TiDB in `values.yaml` file. And then run the same command as above.

> **Note**: See the above caveats of vertical scaling. Before [#35](https://github.com/pingcap/tidb-operator/issues/35) is fixed, you have to manually configure the block cache size for TiKV in charts/tidb-cluster/templates/config/_tikv-config.tpl

## Upgrade TiDB cluster

Upgrade TiDB cluster is similar to scale TiDB cluster, but by changing `image` of PD, TiKV and TiDB to different image versions in `values.yaml`. And then run the following command:

```shell
$ helm upgrade ${releaseName} charts/tidb-cluster
```

For minor version upgrade, updating the `image` should be enough. When TiDB major version is out, the better way to update is to fetch the new charts from tidb-operator and then merge the old values.yaml with new values.yaml. And then upgrade as above.

## Change TiDB cluster Configuration

Since `v1.0.0`, TiDB operator can perform rolling-update on configuration updates. This feature is disabled by default in favor of backward compatibility, you can enable it by setting `enableConfigMapRollout` to `true` in your helm values file.

> WARN: changing this variable against a running cluster will trigger an rolling-update of PD/TiKV/TiDB pods even if there's no configuration change.

## Destroy TiDB cluster

To destroy TiDB cluster, run the following command:

```shell
$ helm delete ${releaseName} --purge
```

The above command only delete the running pods, the data is persistent. If you do not need the data anymore, you can run the following command to clean the data:

```shell
$ kubectl delete pvc -n ${namespace} -l app.kubernetes.io/instance=${releaseName},app.kubernetes.io/managed-by=tidb-operator
$ kubectl get pv -l app.kubernetes.io/namespace=${namespace},app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=${releaseName} -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

> **Note:** the above command will delete the data permanently. Think twice before executing them.


## Monitor

TiDB cluster is monitored with Prometheus and Grafana. When TiDB cluster is created, a Prometheus and Grafana pod will be created and configured to scrape and visualize metrics.

By default the monitor data is not persistent, when the monitor pod is killed for some reason, the data will be lost. This can be avoided by specifying `monitor.persistent` to `true` in `values.yaml` file.

You can view the dashboard using `kubectl portforward`:

```shell
$ kubectl port-forward -n ${namespace} svc/${releaseName}-grafana 3000:3000 &>/tmp/portforward-grafana.log
```

Then open your browser at http://localhost:3000 The default username and password are both `admin`

The Grafana service is exposed as `NodePort` by default, you can change it to `LoadBalancer` if the underlining Kubernetes has load balancer support. And then view the dashboard via load balancer endpoint.

### View TiDB Slow Query Log

For default setup, tidb is configured to export slow query log to STDOUT along with normal server logs. You can obtain the slow query log by `grep` the keyword `SLOW_QUERY`:

```shell
$ kubectl logs -n ${namespace} ${tidbPodName} | grep SLOW_QUERY
```

Optionally, you can output slow query log in a separate sidecar by enabling `separateSlowLog`:

```yaml
# Uncomment the following line to enable separate output of the slow query log
    # separateSlowLog: true
```

Run `helm upgrade` to apply the change, then you can obtain the slow query log from the sidecar named `slowlog`:

```shell
$ kubectl logs -n ${namespace} ${tidbPodName} -c slowlog
```

To retrieve logs from multiple pods, [`stern`](https://github.com/wercker/stern) is recommended.

```shell
$ stern -n ${namespace} tidb -c slowlog
```

## Backup

Currently, TiDB Operator supports two kinds of backup: incremental backup via binlog and full backup(scheduled or ad-hoc) via [Mydumper](https://github.com/maxbube/mydumper).

### Incremental backup

To enable incremental backup, set `binlog.pump.create` and `binlog.drainer.create` to `true`. By default the incremental backup data is stored in protobuffer format in a PV. You can change `binlog.drainer.destDBType` from `pb` to `mysql` or `kafka` and configure the corresponding downstream.

### Full backup

Currently, full backup requires a PersistentVolume. The backup job will create a PVC to store backup data.

By default, the backup uses PV to store the backup data.
> **Note:** You must set the ad-hoc full backup PV's [reclaim policy](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy) to `Retain` to keep your backup data safe.

You can also store the backup data to [Google Cloud Storage](https://cloud.google.com/storage/) bucket or [Ceph object storage](https://ceph.com/ceph-storage/object-storage/) by configuring the corresponding section in `values.yaml`. This way the PV temporarily stores backup data before it is placed in object storage.

The comments in `values.yaml` is self-explanatory for both GCP backup and Ceph backup.

### Scheduled full backup

Scheduled full backup can be ran periodically just like crontab job.

To create a scheduled full backup job, modify `scheduledBackup` section in `values.yaml` file.

* `create` must be set to `true`
* Set `storageClassName` to the PV storage class name used for backup data
* `schedule` takes the [Cron](https://en.wikipedia.org/wiki/Cron) format
* `user` and `password` must be set to the correct user which has the permission to read the database to be backuped.

> **Note:** You must set the scheduled full backup PV's [reclaim policy](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy) to `Retain` to keep your backup data safe.


### Ad-Hoc full backup

> **Note:** The rest of the document will use `values.yaml` to reference `charts/tidb-backup/values.yaml`

Ad-Hoc full backup can be done once just like job.

To create an ad-hoc full backup job, modify `backup` section in `values.yaml` file.

* `mode` must be set to `backup`
* Set `storage.className` to the PV storage class name used for backup data

Create a secret containing the user and password that has the permission to backup the database:

```shell
$ kubectl create secret generic backup-secret -n ${namespace} --from-literal=user=<user> --from-literal=password=<password>
```

Then run the following command to create an ad-hoc backup job:

```shell
$ helm install charts/tidb-backup --name=<backup-name> --namespace=${namespace}
```

## Restore

Restore is similar to backup. See the `values.yaml` file for details.

Modified the variables in `values.yaml` and then create restore job using the following command:

```shell
$ helm install charts/tidb-backup --name=<backup-name> --namespace=${namespace}
```
