# Heterogeneous design for TidbCluster

## Summary

This document presents a design to deploy heterogeneous components, e.g. TiDB, TiKV, etc to an existing TiDB Cluster.

## Motivation

Currently, the spec for the `TidbCluster` describes a group of `PD`/`TiKV`/`TiDB` instances. For each kind of components, 
their specs are all the same. This design is easy to use and also covers the most cases for using the TiDB cluster.

However, as a distributed database with clear and multilayers, the components in each layer could be different from 
each other to meet the different requirements.

For example, as the SQL layer, the TiDB component could be composed of multiple instances with different resource requests and configurations to handle the different workloads (AP/TP query).

For the storage layer, the TiKV component could be composed of multiple instances with different store labels which determine the data distribution among different instances.

And this is also useful for the auto-scaling case, where we may want to scale out TiDBs/TiKVs with different resources or configurations from the original TidbCluster.

### Goals

* Deploy heterogeneous components, e.g. TiDB, TiKV, etc to an existing TiDB Cluster

### Non-Goals

* Heterogeneous deployment for PD, Pump, and TiCDC.

## Proposal

* Make the `spec.pd`, `spec.tidb`, and `spec.tikv` optional in the TidbCluster CR, e.g. change them to pointer. 
* Add cluster reference in the `spec` field of the TidbCluster CR so that the components can start with the PD address in the cluster reference.

  ```
  // +optional
  Cluster *TidbClusterRef `json:"cluster,omitempty"`
  ```

* The status of each TidbCluster should only include the components managed by this TidbCluster and based on this, the failureMembers/failureStores will only include the Pods managed by this TidbCluster so that failover function can work normally
* TLS logic should be updated to support the heterogeneous deployment
* Monitoring should work as expected with the heterogeneous deployment

### Test Plan

* Deploy TidbCluster CR with `spec.tikv` + `spec.cluster` to deploy a TiKV cluster with different configurations and join the existing TiDB Cluster
* Deploy TidbCluster CR with `spec.tidb` + `spec.cluster` to deploy a TiDB cluster with different configurations and join the existing TiDB Cluster
* Deploy TidbCluster CR with `spec.tiflash` + `spec.cluster` to deploy a TiFlash cluster with different configurations and join the existing TiDB Cluster
* Upgrading, scaling in, and scaling out of each TidbCluster CR work as expected.
* Failover of each TidbCluster CR works as expected.
* Existing TiDB Clusters should not roll update after upgrading the TiDB Operator.
* TLS works as expected.
* Monitoring works as expected.

## Drawbacks

* One TidbCluster does not necessarily indicate a full TiDB cluster anymore.

## Alternatives

* TiKVGroup/TiDBGroup, which would include more work for the CR definitions and controllers. You can check detail in the [deprecated design](2020-06-12-heterogeneous-design-for-tidb-cluister.md).
