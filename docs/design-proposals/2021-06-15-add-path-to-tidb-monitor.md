# Pause TiDB Cluster

This document presents a design to add path of tidb monitor in
operator.

## Motivation

At present, when we want to expose monitoring services (including grafana and prothumes) through Ingress, our design is to use different hosts for different tidb-monitors to distinguish which specific monitoring service is being accessed.This brings some inflexible experiences. If we want to distinguish between different monitoring services through one host but different paths, it is impossible to do it now, but in fact this is also a feasible solution

## Proposal

We can add path to the spec and apply it to the ingress spec application, which can bring us more flexible path parameters [spec code](https://github.com/pingcap/tidb-operator/blob/master/pkg/apis/pingcap/v1alpha1/types.go#L1709)

### Spec

Add a new field to `Path` to Ingress Spec:

```
    // Indicates ingress path variable
    // +optional
    Path string `json:"path,omitempty"`
```

### Implementation

If not specified the value is '/'

If value specified,it will take effect on [operator code](https://github.com/pingcap/tidb-operator/blob/master/pkg/monitor/monitor/util.go#L942)

- tidb monitor ingress

## Testing plan

### TiDB cluster can be paused and unpaused

- Deploy a tidb cluster
- Deploy a have 'path' tidb monitor
- Verify ingress spec
- Verify monitor service connectivity

## Open Questions
