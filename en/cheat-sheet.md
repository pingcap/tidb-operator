---
title: Command Cheat Sheet for TiDB Cluster Management
summary: Learn the commonly used commands for managing TiDB clusters.
aliases: ['/docs/tidb-in-kubernetes/dev/cheat-sheet/']
---

# Command Cheat Sheet for TiDB Cluster Management

This document is an overview of the commands used for TiDB cluster management.

## kubectl

### View resources

* View CRD:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get crd
    ```

* View TidbCluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get tc ${name}
    ```

* View TidbMonitor:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get tidbmonitor ${name}
    ```

* View Backup:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get bk ${name}
    ```

* View BackupSchedule:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get bks ${name}
    ```

* View Restore:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get restore ${name}
    ```

* View TidbClusterAutoScaler:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get tidbclusterautoscaler ${name}
    ```

* View TidbInitializer:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get tidbinitializer ${name}
    ```

* View Advanced StatefulSet:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get asts ${name}
    ```

* View a Pod:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pod ${name}
    ```

    View a TiKV Pod:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pod -l app.kubernetes.io/component=tikv
    ```

    View the continuous status change of a Pod:

    ```shell
    watch kubectl -n ${namespace} get pod
    ```

    View the detailed information of a Pod:

    ```shell
    kubectl -n ${namespace} describe pod ${name}
    ```

* View the node on which Pods are located:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pods -l "app.kubernetes.io/component=tidb,app.kubernetes.io/instance=${cluster_name}" -ojsonpath="{range .items[*]}{.spec.nodeName}{'\n'}{end}"
    ```

* View Service:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get service ${name}
    ```

* View ConfigMap:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get cm ${name}
    ```

* View a PersistentVolume (PV):

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pv ${name}
    ```

    View the PV used by the cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get pv -l app.kubernetes.io/namespace=${namespace},app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=${cluster_name}
    ```

* View a PersistentVolumeClaim (PVC):

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pvc ${name}
    ```

* View StorageClass:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get sc
    ```

* View StatefulSet:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get sts ${name}
    ```

    View the detailed information of StatefulSet:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} describe sts ${name}
    ```

### Update resources

* Add an annotation for TiDBCluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} annotate tc ${cluster_name} ${key}=${value}
    ```

    Add a force-upgrade annotation for TiDBCluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} annotate --overwrite tc ${cluster_name} tidb.pingcap.com/force-upgrade=true
    ```

    Delete a force-upgrade annotation for TiDBCluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} annotate tc ${cluster_name} tidb.pingcap.com/force-upgrade-
    ```

    Enable the debug mode for Pods:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} annotate pod ${pod_name} runmode=debug
    ```

### Edit resources

* Edit TidbCluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} edit tc ${name}
    ```

### Patch Resources

* Patch PV ReclaimPolicy:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl patch pv ${name} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
    ```

* Patch a PVC:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} patch pvc ${name} -p '{"spec": {"resources": {"requests": {"storage": "100Gi"}}}'
    ```

* Patch StorageClass:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl patch storageclass ${name} -p '{"allowVolumeExpansion": true}'
    ```

### Create resources

* Create a cluster using the YAML file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} apply -f ${file}
    ```

* Create Namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create ns ${namespace}
    ```

* Create Secret:

    Create Secret of the certificate:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} create secret generic ${secret_name} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

    Create Secret of the user id and password:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} create secret generic ${secret_name} --from-literal=user=${user} --from-literal=password=${password}
    ```

### Interact with running Pods

* View the PD configuration file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/pd/pd.toml
    ```

* View the TiDB configuration file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/tidb/tidb.toml
    ```

* View the TiKV configuration file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/tikv/tikv.toml
    ```

* View Pod logs:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} logs ${pod_name} -f
    ```

    View logs of the previous container:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} logs ${pod_name} -p
    ```

    If there are multiple containers in a Pod, view logs of one container:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} logs ${pod_name} -c ${container_name}
    ```

* Expose services:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} port-forward svc/${service_name} ${local_port}:${port_in_pod}
    ```

    Expose PD services:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} port-forward svc/${cluster_name}-pd 2379:2379
    ```

### Interact with nodes

* Mark the node as unschedulable:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl cordon ${node_name}
    ```

* Mark the node as schedulable:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl uncordon ${node_name}
    ```

### Delete resources

* Delete a Pod:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

* Delete a PVC:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} pvc ${pvc_name}
    ```

* Delete TidbCluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} tc ${tc_name}
    ```

* Delete TidbMonitor:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete -n ${namespace} tidbmonitor ${tidb_monitor_name}
    ```

* Delete TidbClusterAutoScaler:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} delete tidbclusterautoscaler ${name}
    ```

### More

See [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/) for more kubectl usage.

## Helm

### Add Helm repository

{{< copyable "shell-regular" >}}

```shell
helm repo add pingcap https://charts.pingcap.org/
```

### Update Helm repository

{{< copyable "shell-regular" >}}

```shell
helm repo update
```

### View available Helm chart

- View charts in Helm Hub:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search hub ${chart_name}
    ```

    For example:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search hub mysql
    ```

- View charts in other Repos:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search repo ${chart_name} -l --devel
    ```

    For example:

    {{< copyable "shell-regular" >}}

    ```shell
    helm search repo tidb-operator -l --devel
    ```

### Get the default `values.yaml` of the Helm chart

{{< copyable "shell-regular" >}}

```shell
helm inspect values ${chart_name} --version=${chart_version} > values.yaml
```

For example:

{{< copyable "shell-regular" >}}

```shell
helm inspect values pingcap/tidb-operator --version=v1.2.0 > values-tidb-operator.yaml
```

### Deploy using Helm chart

{{< copyable "shell-regular" >}}

```shell
helm install ${name} ${chart_name} --namespace=${namespace} --version=${chart_version} -f ${values_file}
```

For example:

{{< copyable "shell-regular" >}}

```shell
helm install tidb-operator pingcap/tidb-operator --namespace=tidb-admin --version=v1.2.0 -f values-tidb-operator.yaml
```

### View the deployed Helm release

{{< copyable "shell-regular" >}}

```shell
helm ls
```

### Update Helm release

{{< copyable "shell-regular" >}}

```shell
helm upgrade ${name} ${chart_name} --version=${chart_version} -f ${values_file}
```

For example:

{{< copyable "shell-regular" >}}

```shell
helm upgrade tidb-operator pingcap/tidb-operator --version=v1.2.0 -f values-tidb-operator.yaml
```

### Delete Helm release

{{< copyable "shell-regular" >}}

```shell
helm uninstall ${name} -n ${namespace}
```

For example:

{{< copyable "shell-regular" >}}

```shell
helm uninstall tidb-operator -n tidb-admin
```

### More

See [Helm Commands](https://helm.sh/docs/helm/) for more Helm usage.
