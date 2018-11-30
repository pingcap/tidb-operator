# TiDB Cluster Operation Guide

TiDB Operator can manage multiple clusters in the same Kubernetes cluster. Clusters are qualified by `namespace` and `clusterName`, namely different clusters may have same `namespace` or `clusterName` but not both.

The default `clusterName` is `demo` which is defined in charts/tidb-cluster/values.yaml. The following variables will be used in the rest of the document:

```shell
$ releaseName="tidb-cluster"
$ namespace="tidb"
$ clusterName="demo" # Make sure this is the same as variable defined in charts/tidb-cluster/values.yaml
```

> **Note:** The rest of the document will use `values.yaml` to reference `charts/tidb-cluster/values.yaml`

## Deploy TiDB cluster

After TiDB Operator and Helm are deployed correctly, TiDB cluster can be deployed using following command:

```shell
$ helm install charts/tidb-cluster --name=${releaseName} --namespace=${namespace}
$ kubectl get po -n ${namespace} -l app.kubernetes.io/name=tidb-operator
```

The default deployment doesn't set CPU and memory requests or limits for any of the pods, and the storage used is `local-storage` with minimal size. These settings can make TiDB cluster run on a small Kubernetes cluster like DinD or the default GKE cluster for testing. But for production deployment, you would likely to adjust the cpu, memory and storage resources according to the [recommendations](https://github.com/pingcap/docs/blob/master/op-guide/recommendation.md).

The resource limits should be equal or bigger than the resource requests, it is suggested to set limit and request equal to get [`Guaranteed` QoS]( https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed).

For other settings, the variables in `values.yaml` are self-explanatory with comments. You can modify them according to your need before installing the charts.

## Access TiDB cluster

By default TiDB service is exposed using [`NodePort`](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport). You can modify it to `ClusterIP` which will disable access from outside of the cluster. Or modify it to [`LoadBalancer`](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) if the underlining Kubernetes supports this kind of service.

By default TiDB cluster is deployed with a random generated password. You can specify a password by setting `tidb.password` in `values.yaml` before deploying. Whether you specify the password or not, you can retrieve the password through `Secret`:

```shell
$ PASSWORD=$(kubectl get secret -n ${namespace} ${clusterName}-tidb -ojsonpath="{.data.password}" base64 -c | awk '{print $6}')
$ echo ${PASSWORD}
$ kubectl get svc -n ${namespace} # check the available services
```

* Access inside of the Kubernetes cluster

    When your application is deployed in the same Kubernetes cluster, you can access TiDB via domain name `demo-tidb.tidb.svc` with port `4000`. Here `demo` is the `clusterName` which can be modified in `values.yaml`. And the latter `tidb` is the namespace you specified when using `helm install` to deploy TiDB cluster.

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
$ kubectl port-forward -n ${namespace} svc/${clusterName}-grafana 3000:3000 &>/tmp/portforward-grafana.log
```

Then open your browser at http://localhost:3000 The default username and password are both `admin`

The Grafana service is exposed as `NodePort` by default, you can change it to `LoadBalancer` if the underlining Kubernetes has load balancer support. And then view the dashboard via load balancer endpoint.

## Backup

Currently, TiDB Operator supports two kinds of backup: full backup via [Mydumper](https://github.com/maxbube/mydumper) and incremental backup via binlog.

### Full backup

Full backup can be done periodically just like crontab job. Currently, full backup requires a PersistentVolume, the backup job will create a PVC to store backup data.

To create a full backup job, modify `fullbackup` section in `values.yaml` file.

* `create` must be set to `true`
* Set `storageClassName` to the PV storage class name used for backup data
* `schedule` takes the [Cron](https://en.wikipedia.org/wiki/Cron) format
* `user` and `password` must be set to the correct user which has the permission to read the database to be backuped.

If TiDB cluster is running on GKE, the backup data can be uploaded to GCS bucket. A bucket name and base64 encoded service account credential that has bucket read/write access must be provided. The comments in `values.yaml` is self-explanatory for GCP backup.

### Incremental backup

To enable incremental backup, set `binlog.pump.create` and `binlog.drainer.create` to `true`. By default the incremental backup data is stored in protobuffer format in a PV. You can change `binlog.drainer.destDBType` from `pb` to `mysql` or `kafka` and configure the corresponding downstream.

## Restore

Currently, tidb-operator only supports restoring from full backup in GCS bucket. The `restore` section in `values.yaml` should have enough comments as document.
