## Overview

[![Build Status](http://107.150.125.75:8080/job/build_tidb_operator_master/badge/icon)](http://107.150.125.75:8080/job/build_tidb_operator_master/)

The TiDB operator manages TiDB clusters deployed to [Kubernetes](https://kubernetes.io) and automates tasks related to operating a TiDB cluster.

Document List:

- [Requirements](#requirements)
- [Deploy the TiDB operator](#deploy-the-tidb-operator)
- [Create a TiDB cluster](#create-a-tidb-cluster)
- [Resize and apply a rolling update to the cluster](#resize-the-cluster)

## Requirements

Before deploying the TiDB operator, make sure the following requirements are satisfied:

* Kubernetes >= v1.10
* [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) is enabled
* [Helm](https://helm.sh) is [installed](https://github.com/kubernetes/helm/releases).
* Your user has permission to deploy

Your helm install needs proper permissions, e.g.

    kubectl create serviceaccount tiller --namespace kube-system
    kubectl create -f charts/lib/tiller-clusterrolebinding.yaml
    helm init --service-account tiller --upgrade


## Deploy the TiDB operator

The TiDB operator is composed of `tidb-controller-manager`.

Use the following command to deploy the TiDB operator:

    helm install charts/tidb-operator -n tidb-operator --namespace=tidb-operator

The TiDB operator automatically creates the following components:

    $ kubectl get deployment -n tidb-operator -l "app=tidb-operator"
    NAME                      DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
    tidb-controller-manager   1         1         1            0           1m

    $ kubectl get pod -n tidb-operator
    NAME                                       READY     STATUS    RESTARTS   AGE
    tidb-controller-manager-3999554014-1h171   1/1       Running   0          1m

## Create a TiDB cluster

After deploying the TiDB operator, follow the following steps to create a TiDB cluster:

    kubectl apply -f manifests/crd.yaml

This creates the custom resource for the cluster that the operator uses.

    $ kubectl get customresourcedefinitions
    NAME                             KIND
    tidbclusters.pingcap.com         CustomResourceDefinition.v1beta1.apiextensions.k8s.io

Now you can create your cluster

    helm install charts/tidb-cluster -n tidb-demo --namespace=tidb

Wait a few minutes to get all TiDB components running:

    $ kubectl get tidbcluster -n tidb
    NAME           AGE
    demo-cluster   13h

    $ kubectl get statefulset -n tidb
    NAME                DESIRED   CURRENT   AGE
    demo-cluster-pd     3         3         13h
    demo-cluster-tidb   2         2         13h
    demo-cluster-tikv   3         3         13h

    $ kubectl get service -n tidb
    NAME                      TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                          AGE
    demo-cluster-grafana      NodePort    10.105.174.22    <none>        3000:31479/TCP                   14m
    demo-cluster-pd           ClusterIP   10.101.138.121   <none>        2379/TCP                         11m
    demo-cluster-pd-peer      ClusterIP   None             <none>        2380/TCP                         11m
    demo-cluster-prometheus   NodePort    10.108.106.219   <none>        9090:32303/TCP                   14m
    demo-cluster-tidb         NodePort    10.104.195.140   <none>        4000:31155/TCP,10080:30595/TCP   11m
    demo-cluster-tikv-peer    ClusterIP   None             <none>        20160/TCP                        11m

    $ kubectl get configmap -n tidb
    NAME                             DATA      AGE
    demo-cluster-monitor-configmap   3         12h
    demo-cluster-pd                  2         12h
    demo-cluster-tidb                2         12h
    demo-cluster-tikv                2         12h


    $ kubectl get pod -n tidb
    NAME                                       READY     STATUS      RESTARTS   AGE
    demo-cluster-configure-grafana-724x5       0/1       Completed   0          13h
    demo-cluster-pd-0                          1/1       Running     0          13h
    demo-cluster-pd-1                          1/1       Running     0          13h
    demo-cluster-pd-2                          1/1       Running     0          13h
    demo-cluster-prometheus-6fd9b54dcf-b7q9w   2/2       Running     0          13h
    demo-cluster-tidb-0                        1/1       Running     0          13h
    demo-cluster-tidb-1                        1/1       Running     0          13h
    demo-cluster-tikv-0                        2/2       Running     0          13h
    demo-cluster-tikv-1                        2/2       Running     0          13h
    demo-cluster-tikv-2                        2/2       Running     0          13h



## Resize the cluster

First configure `charts/tidb-cluster/values.yaml` file, for example to change the number of `pd` replicas to 3, and change the number of `tikv` replicas to 5.
Then run the following command to update the object.


    helm upgrade tidb-demo charts/tidb-cluster --namespace=tidb

### Contributing

See the [contributing guide](./docs/CONTRIBUTING.md).
