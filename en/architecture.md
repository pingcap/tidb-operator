---
title: TiDB Operator Architecture
summary: Learn the architecture of TiDB Operator and how it works.
aliases: ['/docs/tidb-in-kubernetes/dev/architecture/']
---

# TiDB Operator Architecture

This document describes the architecture of TiDB Operator and how it works.

## Architecture

The following diagram is an overview of the architecture of TiDB Operator.

![TiDB Operator Overview](/media/tidb-operator-overview-1.1.png)

`TidbCluster`, `TidbMonitor`, `TidbInitializer`, `Backup`, `Restore`, `BackupSchedule`, and `TidbClusterAutoScaler` are custom resources defined by CRD (`CustomResourceDefinition`).

* `TidbCluster` describes the desired state of the TiDB cluster.
* `TidbMonitor` describes the monitoring components of the TiDB cluster.
* `TidbInitializer` describes the desired initialization Job of the TiDB cluster.
* `Backup` describes the desired backup of the TiDB cluster.
* `Restore` describes the desired restoration of the TiDB cluster.
* `BackupSchedule` describes the scheduled backup of the TiDB cluster.
* `TidbClusterAutoScaler` describes the automatic scaling of the TiDB cluster.

The following components are responsible for the orchestration and scheduling logic in a TiDB cluster:

* `tidb-controller-manager` is a set of custom controllers in Kubernetes. These controllers constantly compare the desired state recorded in the `TidbCluster` object with the actual state of the TiDB cluster. They adjust the resources in Kubernetes to drive the TiDB cluster to meet the desired state and complete the corresponding control logic according to other CRs;
* `tidb-scheduler` is a Kubernetes scheduler extension that injects the TiDB specific scheduling policies to the Kubernetes scheduler;
* `tidb-admission-webhook` is a dynamic admission controller in Kubernetes, which completes the modification, verification, operation, and maintenance of Pod, StatefulSet, and other related resources.

> **Note:**
>
> `tidb-scheduler` is not mandatory. Refer to [tidb-scheduler and default-scheduler](tidb-scheduler.md#tidb-scheduler-and-default-scheduler) for details.

## Control flow

The following diagram is the analysis of the control flow of TiDB Operator. Starting from TiDB Operator v1.1, the TiDB cluster, monitoring, initialization, backup, and other components are deployed and managed using CR.

![TiDB Operator Control Flow](/media/tidb-operator-control-flow-1.1.png)

The overall control flow is described as follows:

1. The user creates a `TidbCluster` object and other CR objects through kubectl, such as `TidbMonitor`;
2. TiDB Operator watches `TidbCluster` and other related objects, and constantly adjust the `StatefulSet`, `Deployment`, `Service`, and other objects of PD, TiKV, TiDB, Monitor or other components based on the actual state of the cluster;
3. Kubernetes' native controllers create, update, or delete the corresponding `Pod` based on objects such as `StatefulSet`, `Deployment`, and `Job`;
4. If you configure the components to use `tidb-scheduler` in the `TidbCluster` CR, the `Pod` declaration of PD, TiKV, and TiDB specifies `tidb-scheduler` as the scheduler. `tidb-scheduler` applies the specific scheduling logic of TiDB when scheduling the corresponding `Pod`.

Based on the above declarative control flow, TiDB Operator automatically performs health check and fault recovery for the cluster nodes. You can easily modify the `TidbCluster` object declaration to perform operations such as deployment, upgrade, and scaling.
