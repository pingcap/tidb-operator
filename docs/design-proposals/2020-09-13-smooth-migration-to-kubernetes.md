# Smooth migration to Kubernetes

## Summary

This document presents a design to migrate TiDB cluster deployed in physical machine or virtual machine to Kubernetes without using migration tools.

## Motivation

Users may want to migrate TiDB deployed in physical machine or virtual machine to Kubernetes by creating a new TiDB cluster in Kubernetes joining in the existing
PD cluster.

### Goals

* Migrate an existing TiDB cluster without TLS enabled to Kubernetes by creating a new one in Kubernetes with specified PD addresses.

### Non-Goals

* Migrating a TiDB cluster with TLS enabled.

## Proposal

### Prerequisites

* Network between TiDB cluster in Kubernetes and external TiDB cluster must be connected, and domain name need to be resolved correctly by DNS if necessary.

### Defaulting & Validation

* Validation
  * The format of `spec.PDAddresses` must be a string array with the format of `http://{address}:{port}`

## Design Details

Users create a TiDB cluster in Kubernetes by specifying the `spec.PDAddresses` in the `TidbCluster` CR.
The `PDAddresses` is the PD addresses of the cluster you want to migrate data from. If the addresses are available, and the PD in Kubernetes can
connect to one of them and join in it successfully. Then, the new cluster will be created successfully. TiKV `region` will be migrated automatically by raft
to the new TiKV Pods. After you scale in TiKV deployed in physical machine or virtual machine, all data will be migrated to the new TiKV automatically.

### Test Plan

1. Creating a TiDB cluster in old version tidb-operator, and create 8 tables, each with 1000000 rows.
2. Upgrading tidb-operator to the new version including the `spec.PDAddresses` feature. The existing TiDB cluster is running without being affected, e.g. no rolling update.
3. Creating a TiDB cluster with `spec.PDAddresses` specified (If any of the PD addresses does not match the format of `http://{address}:{port}`, the cluster won't be created successfully).
4. Transferring PD leaders to the new TiDB cluster, and scaling in PD replicas to 0. It can still work normally.
5. Deleting all the stores in the old TiDB cluster, and then deleting the old cluster.
6. Notice that all the data are transferred successfully.
7. Removing the `spec.PDAddresses` from the new TiDB Cluster and it has no effect on the cluster, e.g. no rolling update.

