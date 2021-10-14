# updateTidbCluster does not return when error occurs during syncing one component

## Summary

This document presents a design to make updateTidbCluster not return when error occurs during syncing one component, but if one component is upgrading, it will still block the upgrading of the following components.

### Goals

- Fix https://github.com/pingcap/tidb-operator/issues/1245#issuecomment-881129198.  

### Non-Goals

## Proposal

- Aggregate the errors occurring in updateTidbCluster during syncing.
- If one component is upgrading, it will still block the upgrading of the following components.
- eg:

```Go
var errs []error
if err := c.tikvMemberManager.Sync(tc); err != nil {
  errs = append(errs, err)
}
// syncing the pump cluster
if err := c.pumpMemberManager.Sync(tc); err != nil {
    errs = append(errs, err)
}
...
return errorutils.NewAggregate(errs)
```

### Test Plan

- Add more e2e cases with syncing components error.
  - One component is upgrading and error occurs during syncing, the following component cannot upgrade but can scale and failover.
  - One component is upgrading and no error occurs during syncing, the following component cannot upgrade but can scale and failover.
  - One component is not upgrading and error occurs during syncing, the following component can upgrade, scale and failover.
  - One component is not upgrading and no error occurs during syncing, the following component can upgrade, scale and failover.
