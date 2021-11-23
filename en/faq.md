---
title: TiDB FAQs in Kubernetes
summary: Learn about TiDB FAQs in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/faq/']
---

# TiDB FAQs in Kubernetes

This document collects frequently asked questions (FAQs) about the TiDB cluster in Kubernetes.

## How to modify time zone settings？

The default time zone setting for each component container of a TiDB cluster in Kubernetes is UTC. To modify this setting, take the steps below based on your cluster status:

### For the first deployment

Configure the `.spec.timezone` attribute in the TidbCluster CR. For example:

```shell
...
spec:
  timezone: Asia/Shanghai
...
```

Then deploy the TiDB cluster.

### For a running cluster

If the TiDB cluster is already running, first upgrade the cluster, and then configure it to support the new time zone.

1. Upgrade the TiDB cluster:

    Configure the `.spec.timezone` attribute in the TidbCluster CR. For example:

    ```shell
    ...
    spec:
      timezone: Asia/Shanghai
    ...
    ```

    Then upgrade the TiDB cluster.

2. Configure TiDB to support the new time zone:
    
    Refer to [Time Zone Support](https://docs.pingcap.com/tidb/v4.0/configure-time-zone) to modify TiDB service time zone settings.

## Can HPA or VPA be configured on TiDB components?

Currently, the TiDB cluster does not support HPA (Horizontal Pod Autoscaling) or VPA (Vertical Pod Autoscaling), because it is difficult to achieve autoscaling on stateful applications such as a database. Autoscaling can not be achieved merely by the monitoring data of CPU and memory.

## What scenarios require manual intervention when I use TiDB Operator to orchestrate a TiDB cluster?

Besides the operation of the Kubernetes cluster itself, there are the following two scenarios that might require manual intervention when using TiDB Operator:

* Adjusting the cluster after the auto-failover of TiKV. See [Auto-Failover](use-auto-failover.md) for details;
* Maintaining or dropping the specified Kubernetes nodes. See [Maintaining Nodes](maintain-a-kubernetes-node.md) for details.

## What is the recommended deployment topology when I use TiDB Operator to orchestrate a TiDB cluster on a public cloud?

To achieve high availability and data safety, it is recommended that you deploy the TiDB cluster in at least three availability zones in a production environment.

In terms of the deployment topology relationship between the TiDB cluster and TiDB services, TiDB Operator supports the following three deployment modes. Each mode has its own merits and demerits, so your choice must be based on actual application needs.

* Deploy the TiDB cluster and TiDB services in the same Kubernetes cluster of the same VPC;
* Deploy the TiDB cluster and TiDB services in different Kubernetes clusters of the same VPC;
* Deploy the TiDB cluster and TiDB services in different Kubernetes clusters of different VPCs.

## Does TiDB Operator support TiSpark?

TiDB Operator does not yet support automatically orchestrating TiSpark.

If you want to add the TiSpark component to TiDB in Kubernetes, you must maintain Spark on your own in **the same** Kubernetes cluster. You must ensure that Spark can access the IPs and ports of PD and TiKV instances, and install the TiSpark plugin for Spark. [TiSpark](https://pingcap.com/docs/stable/tispark-overview/#deploy-tispark-on-the-existing-spark-cluster) offers a detailed guide for you to install the TiSpark plugin.

To maintain Spark in Kubernetes, refer to [Spark on Kubernetes](http://spark.apache.org/docs/latest/running-on-kubernetes.html).

## How to check the configuration of the TiDB cluster?

To check the configuration of the PD, TiKV, and TiDB components of the current cluster, run the following command:

* Check the PD configuration file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec -it ${pod_name} -n ${namespace} -- cat /etc/pd/pd.toml
    ```

* Check the TiKV configuration file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec -it ${pod_name} -n ${namespace} -- cat /etc/tikv/tikv.toml
    ```

* Check the TiDB configuration file:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec -it ${pod_name} -c tidb -n ${namespace} -- cat /etc/tidb/tidb.toml
    ```

## Why does TiDB Operator fail to schedule Pods when I deploy the TiDB clusters?

Three possible reasons:

* Insufficient resource or HA Policy causes the Pod stuck in the `Pending` state. Refer to [Deployment Failures](deploy-failures.md#the-pod-is-in-the-pending-state) for more details.

* `taint` is applied to some nodes, which prevents the Pod from being scheduled to these nodes unless the Pod has the matching `toleration`. Refer to [taint & toleration](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/) for more details.

* Scheduling conflict, which causes the Pod stuck in the `ContainerCreating` state. In such cases, you can check if there is more than one TiDB Operator deployed in the Kubernetes cluster. Conflicts occur when custom schedulers in multiple TiDB Operators schedule the same Pod in different phases.

    You can execute the following command to verify whether there is more than one TiDB Operator deployed. If more than one record is returned, delete the extra TiDB Operator to resolve the scheduling conflict.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get deployment --all-namespaces |grep tidb-scheduler
    ```

## How does TiDB ensure data safety and reliability? 

To ensure persistent storage of data, TiDB clusters deployed by TiDB Operator use [Persistent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) provided by Kubernetes cluster as the storage.

To ensure data safety in case one node is down, PD and TiKV use [Raft Consistency Algorithm](https://raft.github.io/) to replicate the stored data as multiple replicas across nodes.

In the bottom layer, TiKV replicates data using the log replication and State Machine model. For write requests, data is written to the Leader node first, and then the Leader node replicates the command to its Follower nodes as a log. When most of the Follower nodes in the cluster receive this log from the Leader node, the log is committed and the State Machine changes accordingly.

## If the Ready field of a TidbCluster is false, does it mean that the corresponding TiDBCluster is unavailable?

After you execute the `kubectl get tc` command, if the output shows that the **Ready** field of a TiDBCluster is false, it does not mean that the corresponding TiDBCluster is unavailable, because the cluster might be in any of the following status:

* Upgrading
* Scaling
* Any Pod of PD, TiDB, TiKV, or TiFlash is not Ready

To check whether a TiDB Cluster is unavailable, you can try connecting to TiDB. If the connection fails, it means that the corresponding TiDBCluster is unavailable.