# Canary deployments for TiDB-Operator

## Summary

This document presents a design to support canary deployments TiDB-Operator.

## Motivation

When upgrade TiDB-Operator, users may want to deploy multiple cluster-scoped TiDB-Operators in the same Kubernetes cluster.
And choose some specific tidb clusters will be managed by new  version TiDB-Operator, the others are still managed by old version. When the new version TiDB-Operator is fully tested, replace the old one.

### Goals

* Deploy multiple TiDB-Operator in one Kubernetes cluster
* Manage tidbcluster which be selected
* Don't manage `tidbcluster` which be excluded

### Non-Goals

* Upgrading `TiDB-Operator`
* Upgrading `tidbcluster`

## Proposal

Add `labelSelector` and `excludedLabelSelector`
When sync `tidbcluster` CRDs filter by `labelSelector` and  excluded by `excludedLabelSelector`.

### Example
1. Update the old version `TiDB-Operator` `spec.excludedLabelSelector` excluded `tidbcluster` which be labeled `tidb-operator/managed=v1.1.4`
2. Deploy a new version `TiDB-Operator` and specify `spec.labelSelector` which be labeled `tidb-operator/managed=v1.1.4`
3. Deploy `tidbcluster` with label `tidb-operator/managed=v1.1.4`
4. Full test new version `TiDB-Operator`
5. Label all old `tidbcluster` with `tidb-operator/managed=v1.1.4`
6. Offline the old version `TiDB-Operator`
