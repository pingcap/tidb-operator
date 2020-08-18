# Deploy TiDB cluster across multiple Kubernetes clusters

## Summary

This document presents a design to deploy one TiDB cluster across multiple Kubernetes clusters.

## Motivation

Users may want to deploy one TiDB cluster across multiple regions on public cloud or across multiple data centers with self-managed Kubernetes clusters. However, due to various reasons, e.g. network latency, they can not deploy one Kubernetes cluster across the multiple regions or data centers, so we have to provide a way to deploy one TiDB cluster across multiple Kubernetes clusters.

### Goals

* Deploy one TiDB cluster across multiple Kubernetes clusters

### Non-Goals

* The network interworking across multiple Kubernetes clusters
* The Pod name discovery across multiple Kubernetes clusters

## Proposal

### Prerequisites

* Users have to ensure that the Pod networks are connected with each other across the Kubernetes clusters
* Users have to ensure that the Pod names can be discovered across the Kubernetes clusters

### High-Level Description

* Deploy a TidbCluster in each Kubernetes cluster.
* The distribution of the PD replicas in each Kubernetes cluster should ensure that the major members are not located in one Kubernetes cluster.
  For example, if there are 3 Kubernetes clusters, the distribution of PD replicas can be `1,1,1` or `2,2,1`, etc.
* Update `spec.cluster` in TidbCluster CR and add `spec.cluster.domain`.
* Discovery component is created in each cluster. The PD Pods in each cluster access the Discovery in the same cluster.
* Update the Discovery component
  * if `spec.cluster` is configured, Discovery get the members from the PD of the `spec.cluster`
    * if `spec.cluster.domain` is not configured, access the PD with service name
    * if `spec.cluster.domain` is configured, access the PD with the headless service name
    * if the PD of the `spec.cluster` is not available, try to access its own PD service
  * if `spec.cluster` is not configured, follow the current way   
* TiDB Operator, TiDB, TiKV, and other components in one cluster connect to the PD via the PD service in its own cluster.
* Applications in one cluster connect to the TiDB via the TiDB Service in its own cluster.

## Design Details

* TidbCluster CR supports configuring service domain
  For example, `domain: test.local`, the address of each component in the cluster should be ns0-pd-0.ns0-pd-peer.ns0.svc.test.local, ns0-tikv-0.ns0-tikv-peer.ns0.svc.test.local
* Update `spec.cluster` in TidbCluster CR and add `spec.cluster.domain`
* Update the Discovery component
  * if `spec.cluster` is configured, Discovery get the members from the PD of the `spec.cluster`
    * if `spec.cluster.domain` is not configured, access the PD with service name
    * if `spec.cluster.domain` is configured, access the PD with the headless service name
    * if the PD of the `spec.cluster` is not available, try to access its own PD service
  * if `spec.cluster` is not configured, follow the current way
* TLS support for the TidbClusters across Kubernetes clusters
* The status of each TidbCluster should only include the components managed by this TidbCluster so that the Failover process works as expected

### Test Plan

* Deploy TidbCluster across multiple Kubernetes clusters and all of the PD members and TiKV/TiFlash stores belong to one TiDB Cluster.
* Upgrading, scaling in, and scaling out work as expected.
* Failover in different clusters works as expected.
* Existing TiDB Clusters should not roll update after upgrading the TiDB Operator.
* TLS works as expected.
* Monitoring works as expected.

## Drawbacks

* The specs of the TidbCluster in different Kubernetes clusters are different.

## Alternatives

* Define new custom resources, such as TiKVGroup/TiDBGroup/TiFlashGroup, deploy them in other Kubernetes clusters, and join the existing TiDB Cluster.
  * This would include more work for the CR definition and controllers for the CRs
  * For across Kubernetes clusters deployment, the TiDB Cluster may need to provide services in all of the Kubernetes clusters, considering the network latency and other factors, we may end up with deploying all of the PDGroup/TiKVGroup/TiDBGroup/TiFlashGroup in each Kubernetes cluster, which is more complicated than using TidbCluster directly for users.
