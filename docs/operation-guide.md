# TiDB Cluster Operation Guide

TiDB Operator can manage multiple clusters in the same Kubernetes cluster.

The following variables will be used in the rest of the document:

```shell
$ releaseName="demo"
$ namespace="tidb"
```

> **Note:** The rest of the document will use `values.yaml` to reference `charts/tidb-cluster/values.yaml`

## Configuration

TiDB Operator uses `values.yaml` as TiDB cluster configuration file. It provides the default basic configuration which you can use directly for quick deployment, but if you have specific configuration requirements or for production deployment, you need to manually modify the variables in the `values.yaml` file.

* Resource setting

    * CPU & Memory

        The default deployment doesn't set CPU and memory requests or limits for any of the pods, these settings can make TiDB cluster run on a small Kubernetes cluster like DinD or the default GKE cluster for testing. But for production deployment, you would likely to adjust the cpu, memory and storage resources according to the [recommendations](https://pingcap.com/docs/dev/how-to/deploy/hardware-recommendations/#software-and-hardware-recommendations).
        
        The resource limits should be equal or bigger than the resource requests, it is suggested to set limit and request equal to get [`Guaranteed` QoS]( https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed).

    * Storage

        The variables `pd.storageClassName` and `tikv.storageClassName` in `values.yaml` are used to set `StorageClass` of PD and TiKV,their default setting are `local-storage` with minimal size.
        
        If you don't want to use the default `StorageClass` or your Kubernetes cluster does not support `local-storage` class, please execute the following command to find an available `StorageClass` and select the ones you want to provide to TiDB cluster.

        ```shell
        $ kubectl get sc
        ```

* Disaster Tolerance setting

    TiDB is a distributed database. Its disaster tolerance means that when any physical node failed, not only to ensure TiDB server is available, but also ensure the data is complete and available.

    How to guarantee Disaster Tolerance of TiDB cluster on Kubernetes?

    We mainly solve the problem from the scheduling of services and data.

    * Disaster Tolerance of TiDB instance

        TiDB Operator provides an extended scheduler to guarantee PD/TiKV/TiDB instance disaster tolerance on host level. TiDB Cluster has set the extended scheduler as default scheduler, you will find the setting in the variable `schedulerName` of `values.yaml`.

        In the other hand use `PodAntiAffinity` term of `affinity` to ensure disaster tolerance on the other topology levels (e.g. rack, zone, region). 
        refer to the doc: [pod affnity & anti affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature), moreover `values.yaml` also provides a typical disaster tolerance setting example in the comments of `pd.affinity`.

    * Disaster Tolerance of data

        Disaster tolerance of data is guaranteed by TiDB Cluster itself. The only work Operator needs to do is that collects topology info from specific labels of Kubernetes nodes where TiKV Pod runs on and then PD will schedule data replicas auto according to the topology info.
        Because current TiDB Operator can only recognize some specific labels, so you can only set nodes topology info with the following particular labels

        * `region`: region where node is located
        * `zone`: zone where node is located
        * `rack`: rack where node is located
        * `kubernetes.io/hostname`: hostname of the node

        you need label topology info to nodes of Kubernetes cluster use the following command
        ```shell
        # Not all tags are required
        $ kubectl label node <nodeName> region=<regionName> zone=<zoneName> rack=<rackName> kubernetes.io/hostname=<hostName>
        ```

For other settings, the variables in `values.yaml` are self-explanatory with comments. You can modify them according to your need before installing the charts.

## Deploy TiDB cluster

After TiDB Operator and Helm are deployed correctly and configuration completed, TiDB cluster can be deployed using following command:

```shell
$ helm install charts/tidb-cluster --name=${releaseName} --namespace=${namespace}
$ kubectl get po -n ${namespace} -l app.kubernetes.io/instance=${releaseName}
```

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

> **Note**: currently, changing PD's `scheduler` and `replication` configurations(`maxStoreDownTime` and `maxReplicas` in `values.yaml`, and all the configuration key under `[scheduler]` and `[replication]` section if you override the pd config file) after cluster creation has no effect. You have to configure these variables via `pd-ctl` after the cluster creation, see: [pd-ctl](https://pingcap.com/docs/dev/reference/tools/pd-control/)

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

## Backup and Restore

TiDB Operator provides highly automated backup and recovery operations for TiDB cluster. You can easily take full backup or setup incremental backup of TiDB cluster, and restore the TiDB cluster when the cluster fails.

For detail operation guides of backup and restore, please refer to [Backup and Restore TiDB Cluster](./backup-restore.md).
