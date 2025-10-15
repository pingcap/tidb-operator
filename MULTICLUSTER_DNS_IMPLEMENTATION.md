# Multicluster DNS Support Implementation Summary

This document provides a summary of changes made to add multicluster DNS support to TiDB Operator according to [KEP-2577](https://github.com/kubernetes/enhancements/pull/2577).

## Overview

Added support for multicluster DNS naming format: `<clusterid>.<svc>.<ns>.svc.<clustersetzone>`

This enables TiDB clusters deployed across multiple Kubernetes clusters to use standardized multicluster DNS naming, compatible with CoreDNS multicluster plugin and Kubernetes MCS API.

## Modified Files

### 1. API Definition (`pkg/apis/pingcap/v1alpha1/types.go`)

**Added:**
- `MulticlusterDNSSpec` struct with `ClusterID` and `ClusterSetZone` fields
- `MulticlusterDNS` field in `TidbClusterSpec`

```go
type MulticlusterDNSSpec struct {
    ClusterID string `json:"clusterID"`
    ClusterSetZone string `json:"clusterSetZone"`
}
```

**Lines:** 393, 1254-1269

---

### 2. TidbCluster Helper Methods (`pkg/apis/pingcap/v1alpha1/tidbcluster.go`)

**Added methods:**
- `UseMulticlusterDNS()` - checks if multicluster DNS is enabled
- `GetMulticlusterDNSClusterID()` - returns the cluster ID
- `GetMulticlusterDNSClusterSetZone()` - returns the cluster set zone

**Lines:** 442-461

---

### 3. DNS Name Generation (`pkg/manager/member/utils.go`)

**Added functions:**
- `PdNameForMulticlusterDNS()` - generates PD names in multicluster DNS format
- `PDMSNameForMulticlusterDNS()` - generates PDMS names in multicluster DNS format

**Format:**
```
<clusterid>.<pod-name>.<svc-name>.<namespace>.svc.<clustersetzone>
```

**Lines:** 161-167, 184-189

---

### 4. PD Start Script (`pkg/manager/member/startscript/v2/pd_start_script.go`)

**Modified functions:**
- `RenderPDStartScript()` - uses multicluster DNS format when enabled
- `renderPDMSStartScript()` - uses multicluster DNS format for PDMS when enabled

**Changes:**
- PDDomain and PDMSDomain construction now checks `tc.UseMulticlusterDNS()`
- Generates multicluster DNS format: `<clusterid>.${POD_NAME}.<svc>.<ns>.svc.<zone>`

**Lines:** 66-77, 145-156

---

### 5. Discovery Component (`pkg/discovery/discovery.go`)

**Modified function:**
- `Discover()` - parses and handles multicluster DNS format

**Changes:**
- Detects multicluster DNS format (6+ segments with "svc" at position 4)
- Extracts clusterID, podName, serviceName, and namespace from DNS name
- Initializes PD cluster with correct name format

**Lines:** 84-143

---

## New Files

### 6. Example Configuration
**File:** `examples/multi-cluster/tidb-cluster-multicluster-dns.yaml`

Example TidbCluster CR with multicluster DNS enabled:
```yaml
spec:
  multiclusterDNS:
    clusterID: "cluster1"
    clusterSetZone: "clusterset.local"
```

---

### 7. Documentation
**File:** `docs/multicluster-dns-setup.md`

Comprehensive guide covering:
- Prerequisites (CoreDNS multicluster plugin, MCS API)
- Configuration examples
- Multi-cluster deployment scenarios
- DNS resolution verification
- Troubleshooting

---

## Feature Behavior

### When Multicluster DNS is Enabled

1. **DNS Names:** PD pods use format:
   ```
   cluster1.basic-pd-0.basic-pd-peer.tidb.svc.clusterset.local
   ```

2. **Start Scripts:** PD and PDMS start scripts use multicluster DNS format for:
   - `--name` parameter
   - `--advertise-peer-urls`
   - `--advertise-client-urls`

3. **Discovery:** Discovery component recognizes and parses multicluster DNS format

4. **PD Cluster Init:** Initial cluster formation uses FQDN with clusterID prefix

### Backward Compatibility

- Existing deployments continue to work unchanged
- `multiclusterDNS` is optional (can be nil)
- When not configured, behavior is identical to current implementation
- Compatible with existing `acrossK8s` and `clusterDomain` settings

---

## Configuration

### Required Fields

```yaml
multiclusterDNS:
  clusterID: "cluster1"        # Required: unique cluster identifier
  clusterSetZone: "clusterset.local"  # Required: ClusterSet domain
```

### Validation

- `clusterID` must be a valid DNS label (RFC 1123)
- `clusterSetZone` must be a valid domain name
- Both fields required when `multiclusterDNS` is specified

---

## Integration Points

### CoreDNS Multicluster Plugin
- Requires CoreDNS multicluster plugin installed
- Plugin must be configured with matching `clusterSetZone`

### Kubernetes MCS API
- Requires Multi-Cluster Services API
- Services must be exported/imported across clusters

### Network Requirements
- Pod networks must be connected across clusters
- Firewall rules must allow cross-cluster communication

---

## Testing Recommendations

1. **Unit Tests:** Test new helper functions and DNS name generation
2. **Integration Tests:** Test PD cluster formation with multicluster DNS
3. **E2E Tests:** Deploy across multiple clusters and verify:
   - DNS resolution
   - PD cluster formation
   - Cross-cluster communication
   - Failover scenarios

---

## Future Enhancements

Potential future improvements:
1. Support for other components (TiKV, TiDB, TiFlash) if needed
2. Automatic cluster ID generation
3. Integration with service mesh
4. Enhanced validation and error messages

---

## References

- [KEP-2577: Multicluster Services DNS](https://github.com/kubernetes/enhancements/pull/2577)
- [CoreDNS Multicluster](https://github.com/coredns/multicluster)
- [Kubernetes MCS API](https://github.com/kubernetes-sigs/mcs-api)
- [TiDB Operator Docs](https://docs.pingcap.com/tidb-in-kubernetes)

