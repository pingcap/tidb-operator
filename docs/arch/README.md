# Architecture of the TiDB Operator(v2)

<!-- toc -->
- [Overview](#overview)
- [Goals](#goals)
- [Key changes](#key-changes)
- [Arch](#arch)
- [CRD](#crd)
  - [Cluster](#cluster)
  - [Component Group](#component-group)
  - [Instance](#instance)
<!-- /toc -->

## Overview

TiDB Operator is a software to manage multiple TiDB clusters in the Kubernetes platform. It's designed based on the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

The v2 version of TiDB Operator aims to resolve some painful issues and make the operator more extensible and user friendly.

## Goals

- Flexible. Advanced users can redefine many default behaviors of TiDB Operator to manage TiDBs in thier own way.
- Extensible. It's easy to extend TiDB Operator(v2) to manage a new component of TiDB.
- User friendly. The validation and kubectl plugin will be supported to make ops more secure and easy.

## Key changes

- Split the huge TidbCluster CR into multi-smaller CRs, including Cluster, PDGroup, PD, TiKVGroup, TiKV, ...
- Add more controllers for these new CRs
- Remove StatefulSet dependency

## Arch

[controller runtime](https://github.com/kubernetes-sigs/controller-runtime) is used as the underlying framework.

## CRD

TODO: need a picture

### Cluster

Cluster is a CRD that abstracts a TiDB Cluster. It contains some common configurations and feature gates of the TiDB Cluster and displays the overview status of the whole cluster. This CRD is designed like a "namespace". All components of the TiDB Cluster should refer to a Cluster CR.

### Component Group

Component Group means a set of instances for a specified component of TiDB Clusters. For example, PDGroup, TiKVGroup. In a cluster, one component can have more than one Group CR with a different name. A use case is one TiDBGroup for TP workloads and another TiDBGroup for AP workloads.

Normally, the controller of component groups is resposible for:

- Instance lifecycle management
- Control replicas
- Specify behavior policy
  - Schedule
  - Scale
  - Update

Users can specify the template of instance CRs by the field spec.template in component group CR to create or update instances.

### Instance

Instance means an instance of a component, for example, TiKV, PD. It manages an individual pod and its volumes. All "states" such as URI and volumes are bound with the instance CR. Users can easily create or remove a PD/TiKV/TiDB by creating or removing an instance CR.

The controller of instances is resposible for:
- Manage pods, configmaps and volumes
- Provide APIs at the instance level

Instance CRs are normally managed by their component group. Most fields of a instance CR are immutable for users.
