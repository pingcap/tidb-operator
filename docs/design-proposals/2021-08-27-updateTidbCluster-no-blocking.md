# updateTidbCluster not block when error occurs

## Summary

This document presents a design to make updateTidbCluster sync one component not block other components when error occurs except for Upgrading.

## Motivation

- Fix part of the issues mentioned in <https://github.com/pingcap/tidb-operator/issues/1245>. Upgrading or scaling of one component may block the scaling of the other components.  
- Fix the issues mentioned in <https://github.com/pingcap/tidb-operator/issues/3033>.

### Goals

* Make updateTidbCluster logic continue when error occurs on syncing one component.
* If the component is upgrading,subsequent components will not be syned.

### Non-Goals

## Proposal

- Use error Aggregate when error occurs on syncing.
- If the component is not upgrading,updateTidbCluster sync logic will continue when error occurs.
- If the component is upgrading,will break updateTidbCluster sync logic and requeue.
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

- Add more e2e-test with syncing components error.
- Test Upgrade components. Be not able to sync subsequent components when a component is upgrading. 
- Test Sync components with one component occurs error and other component success. Be able to sync subsequent components when component occurs error and is not upgrading.this may contains many cases:
  1. With PD and TiKV Sepc changed.Sync PD error, sync TiKV success
  2. With PD and TiKV TiDB changed.Sync PD error, sync TiDB success. 
  3. With TiKV and TiDB Sepc changed.Sync TiKV success,sync TiDB success.
  4. ...
- Test if other components' Services/Statefulsets/PVCs... are normal or not when one component occurs error.  
- Manual tests.

## Drawbacks

- This proposal won't fix the block in one component.eg:  <https://github.com/pingcap/tidb-operator/issues/3612> and <https://github.com/pingcap/tidb-operator/pull/1242#discussion_r351140739>.

## Alternatives

- It's complex and hard to deal with scenario of upgrade by Using Parallelize control-loops of same TiDB cluster. 
