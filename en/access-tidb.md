---
title: Access the TiDB Cluster in Kubernetes
summary: Learn how to access the TiDB cluster in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/access-tidb/']
---

# Access the TiDB Cluster in Kubernetes

This document describes how to access the TiDB cluster in Kubernetes.

+ To access the TiDB cluster within a Kubernetes cluster, use the TiDB service domain name `${cluster_name}-tidb.${namespace}`.
+ To access the TiDB cluster outside a Kubernetes cluster, expose the TiDB service port by editing the `spec.tidb.service` field configuration in the `TidbCluster` CR.

    {{< copyable "" >}}

    ```yaml
    spec:
        ...
        tidb:
        service:
          type: NodePort
          # externalTrafficPolicy: Cluster
          # annotations:
          #   cloud.google.com/load-balancer-type: Internal
    ```

## NodePort

If there is no LoadBalancer, expose the TiDB service port in the following two modes of NodePort:

- `externalTrafficPolicy=Cluster`: All machines in the Kubernetes cluster assign a NodePort to TiDB Pod, which is the default mode.

    When using the `Cluster` mode, you can access the TiDB service by using the IP address of any machine plus the same port. If there is no TiDB Pod on the machine, the corresponding request is forwarded to the machine with a TiDB Pod.

    > **Note:**
    >
    > In this mode, the request's source IP obtained by the TiDB server is the node IP, not the real client's source IP. Therefore, the access control based on the client's source IP is not available in this mode.

- `externalTrafficPolicy=Local`: Only those machines that run TiDB assign NodePort to TiDB Pod so that you can access local TiDB instances.

    When you use the `Local` mode, it is recommended to enable the `StableScheduling` feature of `tidb-scheduler`. `tidb-scheduler` tries to schedule the newly added TiDB instances to the existing machines during the upgrade process. With such scheduling, client outside the Kubernetes cluster does not need to upgrade configuration after TiDB is restarted.

### View the IP/PORT exposed in NodePort mode

To view the Node Port assigned by Service, run the following commands to obtain the Service object of TiDB:

{{< copyable "shell-regular" >}}

```shell
kubectl -n ${namespace} get svc ${cluster_name}-tidb -ojsonpath="{.spec.ports[?(@.name=='mysql-client')].nodePort}{'\n'}"
```

To check you can access TiDB services by using the IP of what nodes, see the following two cases:

- When `externalTrafficPolicy` is configured as `Cluster`, you can use the IP of any node to access TiDB services.
- When `externalTrafficPolicy` is configured as `Local`, use the following commands to get the nodes where the TiDB instance of a specified cluster is located:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get pods -l "app.kubernetes.io/component=tidb,app.kubernetes.io/instance=${cluster_name}" -ojsonpath="{range .items[*]}{.spec.nodeName}{'\n'}{end}"
    ```

## LoadBalancer

If Kubernetes is run in an environment with LoadBalancer, such as GCP/AWS platform, it is recommended to use the LoadBalancer feature of these cloud platforms by setting `tidb.service.type=LoadBalancer`.

See [Kubernetes Service Documentation](https://kubernetes.io/docs/concepts/services-networking/service/) to know more about the features of Service and what LoadBalancer in the cloud platform supports.

## Gracefully upgrade the TiDB cluster

When you perform a rolling update to the TiDB cluster, Kubernetes sends a [`TERM`](https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods) signal to the TiDB server before it stops the TiDB pod. When the TiDB server receives the `TERM` signal, it tries to wait for all connections to close. After 15 seconds, the TiDB server forcibly closes all the connections and exits the process.

Starting from v1.1.2, TiDB Operator supports gracefully upgrading the TiDB cluster. You can enable this feature by configuring the following items:

- `spec.tidb.terminationGracePeriodSeconds`: The longest tolerable duration to delete the old TiDB Pod during the rolling upgrade. If this duration is exceeded, the TiDB Pod will be deleted forcibly.
- `spec.tidb.lifecycle`: Sets the `preStop` hook for the TiDB Pod, which is the operation executed before the TiDB server stops.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
spec:
  version: v4.0.0
  pvReclaimPolicy: Retain
  discovery: {}
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
    terminationGracePeriodSeconds: 60
    lifecycle:
      preStop:
        exec:
          command:
          - /bin/sh
          - -c
          - "sleep 10 && kill -QUIT 1"
```

The YAML file above:

- Sets the longest tolerable duration to delete the TiDB Pod to 60 seconds. If the client does not close the connections after 60 seconds, these connections will be closed forcibly. You can adjust the value according to your needs.
- Sets the value of `preStop` hook to `sleep 10 && kill -QUIT 1`. Here `PID 1` refers to the PID of the TiDB server process in the TiDB Pod. When the TiDB server process receives the signal, it exits only after all the connections are closed by the client.

When Kubernetes deletes the TiDB Pod, it also removes the TiDB node from the service endpoints. This is to ensure that the new connection is not established to this TiDB node. However, because this process is asynchronous, you can make the system sleep for a few seconds before you send the `kill` signal, which makes sure that the TiDB node is removed from the endpoints.
