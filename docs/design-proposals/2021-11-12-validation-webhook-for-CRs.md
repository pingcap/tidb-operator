# Validation webhook for CRs

## Summary

This document presents a design to validate and admit webhook for CRs, which contains:

- TidbCluster
- TidbMonitor
- TidbInitializer
- Backup
- Restore
- BackupSchedule
- TidbClusterAutoScaler

### Goals

Fix issues:

- <https://github.com/pingcap/tidb-operator/issues/3949>
- <https://github.com/pingcap/tidb-operator/issues/4065>
- <https://github.com/pingcap/tidb-operator/issues/3468>
- <https://github.com/pingcap/tidb-operator/issues/3213>
- <https://github.com/pingcap/tidb-operator/issues/2534>
- <https://github.com/pingcap/tidb-operator/issues/2490>

### Non-Goals

## Proposal

- Add an admission controller named resource to admission server just like pod/statefulset/strategy.
- Create resource package in pkg/webhook, the directory tree may like this:

```text
pkg/webhook/resource/
|- resource.go                        // resource register and resource interface 
|- tidbcluster_resource.go            // validate or admit logic of TidbCluster
|- tidbcluster_resource_test.go
|- tidbmonitor_resource.go            // validate or admit logic of TidbMonitor
|- tidbmonitor_resource_test.go 
|- ...                                // other resources
|- webhook.go            // resourceAdmissionControl struct, implementation interfaces in generic-admission-server and initialization of Resources.
|- webhook_test.go        
```

- Interface `ValidateAdmitResource` implemented by every resource in `resource.go` may have the following methods:

```Go
type ValidateAdmitResource interface {
  // NewObject create an empty object that the resource validate or admit to
  NewObject() runtime.Object
  // ValidateCreate validate when resource applied first time
  ValidateCreate(ctx context.Context, obj runtime.Object) *admission.AdmissionResponse
  // ValidateUpdate validate updating of resource
  ValidateUpdate(ctx context.Context, obj, old runtime.Object) *admission.AdmissionResponse
  // AdmitCreate mutate resource when resource create 
  AdmitCreate(ctx context.Context, obj runtime.Object) *admission.AdmissionResponse
  // AdmitUpdate mutate resource when resource update
  AdmitUpdate(ctx context.Context, obj, old runtime.Object) *admission.AdmissionResponse
}
```

- Primary logic of function `Validate` in may like:
  - Initialize Object
  - Call function `ValidateCreate` or `ValidateUpdate` by different operation of AdmissionRequest

### Test Plan

- Add e2e cases with different kinds of resources.
