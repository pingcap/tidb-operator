# Pause TiDB Cluster

This document presents a design to pause reconciliation of tidb cluster in
operator.

## Motivation

As TiDB and operator evolves, it's unavoidable that we may need to introduce
break changes in statefulset spec of our components, e.g. 

- enable `--advertise-address` flag for all tidb servers
- change TiDB pod readiness probe from HTTPGet to TCPSocket 4000 port

These changes will trigger rolling-upgrade during the upgrade of tidb-operator.

We need to a mechanism which can help users control upgrade process.

In normal upgrades, users can also pause the tidb cluster and apply multiple
fixes in between pausing and resuming without triggering unnecessary rollouts.

## Proposal

This design is inspired by [deployment
paused](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#pausing-and-resuming-a-deployment) feature.

### Spec

Add a new field to `TidbCluster` which indicates whether the tidb cluster is
paused or not:

```
    // Indicates that the tidb cluster is paused and will not be processed by
    // the controller.
    // +optional
    Paused bool `json:"paused,omitempty"`
```

### Implementation

If `false` (default), nothing happens.

If `true`, the controller pauses the reconciliation for

- pd service and statefulset
- tikv service and statefulset
- tidb service and statefulset
- pump service and statefulset
- tiflash service and statefulset
- ticdc service and statefulset

The following are not affected:

- The reconciliation of discovery service
- The reconciliation of tidb cluster status

## Testing plan

### TiDB cluster can be paused and unpaused

- Deploy a tidb cluster
- Pause tidb cluster
- Make a change
- Verify that the change does not take effect
- Unpause tidb cluster
- Verify that the change takes effect now

## Open Questions

### Should auto-failover continue to work when the tidb cluster is paused

Currently, auto failover is paused too. This is easy to implement and we think
users should resume the tidb cluster once the work is done.
