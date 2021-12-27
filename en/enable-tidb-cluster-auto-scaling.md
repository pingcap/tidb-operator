---
title: Enable TidbCluster Auto-scaling
summary: Learn how to use the TidbCluster auto-scaling feature.
aliases: ['/docs/tidb-in-kubernetes/dev/enable-tidb-cluster-auto-scaling/']
---

# Enable TidbCluster Auto-scaling

Kubernetes provides [Horizontal Pod Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/), a native API for elastic scaling. Based on Kubernetes, TiDB 5.0 has implemented an elastic scheduling mechanism. Correspondingly, in TiDB Operator 1.2 and later versions, you can enable the auto-scaling feature to enable elastic scheduling. This document introduces how to enable and use the auto-scaling feature of `TidbCluster`.

## Enable the auto-scaling feature

> **Warning:**
>
> * The auto-scaling feature is in the alpha stage. It is highly **not recommended** to enable this feature in the critical production environment.
> * It is recommended to try this feature in a test environment. PingCAP welcomes your comments and suggestions to help improve this feature.
> * Currently the auto-scaling feature is based solely on CPU utilization.

To turn this feature on, you need to enable some related configurations in TiDB Operator. The auto-scaling feature is disabled by default.

Take the following steps to manually enable auto-scaling:

1. Edit the `values.yaml` file in TiDB Operator.

    Enable `AutoScaling` in the `features` option:

    ```yaml
    features:
    - AutoScaling=true
    ```

2. Install or update TiDB Operator.

    To install or update TiDB Operator, see [Deploy TiDB Operator in Kubernetes](deploy-tidb-operator.md).

3. Confirm the resource configuration of the target TiDB cluster.

    Before using the auto-scaling feature on the target TiDB cluster, first you need to configure the CPU setting of the corresponding components. For example, you need to configure `spec.tikv.requests.cpu` in TiKV:

    ```yaml
    spec:
      tikv:
        requests:
          cpu: "1"
      tidb:
        requests:
          cpu: "1"
    ```

## `TidbClusterAutoScaler`

The `TidbClusterAutoScaler` CR object is used to control the auto-scaling behavior in the TiDB cluster.

The following is an example.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbClusterAutoScaler
metadata:
  name: auto-scaling-demo
spec:
  cluster:
    name: auto-scaling-demo
  tikv:
    resources:
      storage_small:
        cpu: 1000m
        memory: 2Gi
        storage: 10Gi
        count: 3
    rules:
      cpu:
        max_threshold: 0.8
        min_threshold: 0.2
        resource_types:
        - storage_small
    scaleInIntervalSeconds: 500
    scaleOutIntervalSeconds: 300
  tidb:
    resources:
      compute_small:
        cpu: 1000m
        memory: 2Gi
        count: 3
    rules:
      cpu:
        max_threshold: 0.8
        min_threshold: 0.2
        resource_types:
        - compute_small
```

### Implementation principles

According to the configuration of the `TidbClusterAutoScaler` CR, TiDB Operator sends requests to PD to query the result of scaling. Based on the result, TiDB Operator makes use of the [heterogeneous cluster](deploy-heterogeneous-tidb-cluster.md) feature to create, update, or delete the heterogeneous TiDB cluster (only the TiDB component or the TiKV component is configured). In this way, the auto-scaling of the TiDB cluster is achieved.

### Related fields

* `spec.cluster`: the TiDB cluster to be elastically scheduled.

    * `name`: the name of the TiDB cluster.
    * `namespace`: the namespace of the TiDB cluster. If not configured, this field is set to the same namespace as the `TidbClusterAutoScaler` CR by default.

* `spec.tikv`: the configuration related to TiKV elastic scheduling.
* `spec.tikv.resources`: the resource types that TiKV can use for elastic scheduling. If not configured, this field is set to the same value as `spec.tikv.requests` in the `TidbCluster` CR corresponding to `spec.cluster`.

    * `cpu`: CPU configuration.
    * `memory`: memory configuration.
    * `storage`: storage configuration.
    * `count`: the number of resources that the current configuration can use. If this field is not configured, there is no limit on resources.

* `spec.tikv.rules`: the rules of TiKV elastic scheduling. Currently only CPU-based rules are supported.

    * `max_threshold`: If the average CPU utilization of all Pods is higher than `max_threshold`, the scaling-out operation is triggered.
    * `min_threshold`: If the average CPU utilization of all Pods is lower than `min_threshold`, the scaling-in operation is triggered.
    * `resource_types`: the resource types that can be used for CPU-based elastic scheduling. This field corresponds to `key` in `spec.tikv.resources[]`. If not configured, this field is set to all `key`s in `spec.tikv.resources[]` by default.

* `spec.tikv.scaleInIntervalSeconds`: the interval between this scaling-in operation and the last scaling in/out operation. If not configured, the field is set to `500` by default, which means 500 seconds.
* `spec.tikv.scaleOutIntervalSeconds`: the interval between this scaling-out operation and the last scaling in/out operation. If not configured, the field is set to `300` by default, which means 300 seconds.
* `spec.tidb`: the configuration related to TiDB elastic scheduling. Other fields are the same as `spec.tikv`.

For more information about configuration fields, refer to [API references](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md#basicautoscalerspec).

## Example

1. Run the following commands to quickly deploy a TiDB cluster with 3 PD instances, 3 TiKV instances, 2 TiDB instances, and the monitoring and the auto-scaling features.

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/auto-scale/tidb-cluster.yaml -n ${namespace}
    ```

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/auto-scale/tidb-monitor.yaml -n ${namespace}
    ```

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/auto-scale/tidb-cluster-auto-scaler.yaml  -n ${namespace}
    ```

2. Prepare data using [sysbench](https://github.com/akopytov/sysbench).

    Copy the following content and paste it to the local `sysbench.config` file:

    {{< copyable "" >}}

    ```config
    mysql-host=${tidb_service_ip}
    mysql-port=4000
    mysql-user=root
    mysql-password=
    mysql-db=test
    time=120
    threads=20
    report-interval=5
    db-driver=mysql
    ```

    Prepare data by running the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    sysbench --config-file=${path}/sysbench.config oltp_point_select --tables=1 --table-size=20000 prepare
    ```

3. Start the stress test:

    {{< copyable "shell-regular" >}}

    ```bash
    sysbench --config-file=${path}/sysbench.config oltp_point_select --tables=1 --table-size=20000 run
    ```

    The command above will return the following result:

    ```sh
    Initializing worker threads...

    Threads started!

    [ 5s ] thds: 20 tps: 37686.35 qps: 37686.35 (r/w/o: 37686.35/0.00/0.00) lat (ms,95%): 0.99 err/s: 0.00 reconn/s: 0.00
    [ 10s ] thds: 20 tps: 38487.20 qps: 38487.20 (r/w/o: 38487.20/0.00/0.00) lat (ms,95%): 0.95 err/s: 0.00 reconn/s: 0.00
    ```

4. Create a new terminal session and view the Pod changing status of the TiDB cluster by running the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    watch -n1 "kubectl -n ${namespace} get pod"
    ```

    The output is as follows:

    ```sh
    auto-scaling-demo-discovery-fbd95b679-f4cb9   1/1     Running   0          17m
    auto-scaling-demo-monitor-6857c58564-ftkp4    3/3     Running   0          17m
    auto-scaling-demo-pd-0                        1/1     Running   0          17m
    auto-scaling-demo-tidb-0                      2/2     Running   0          15m
    auto-scaling-demo-tidb-1                      2/2     Running   0          15m
    auto-scaling-demo-tikv-0                      1/1     Running   0          15m
    auto-scaling-demo-tikv-1                      1/1     Running   0          15m
    auto-scaling-demo-tikv-2                      1/1     Running   0          15m
    ```

    View the changing status of Pods and the TPS and QPS of sysbench. When new Pods are created in TiKV and TiDB, the TPS and QPS of sysbench increase significantly.

    After sysbench finishes the test, the newly created Pods in TiKV and TiDB disappear automatically.

5. Destroy the environment by running the following commands:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl delete tidbcluster auto-scaling-demo -n ${namespace}
    kubectl delete tidbmonitor auto-scaling-demo -n ${namespace}
    kubectl delete tidbclusterautoscaler auto-scaling-demo -n ${namespace}
    ```
