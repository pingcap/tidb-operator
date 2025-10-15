# Multicluster DNS Setup for TiDB Operator

This document describes how to enable and use multicluster DNS naming in TiDB Operator according to [Kubernetes Enhancement Proposal (KEP) 2577](https://github.com/kubernetes/enhancements/pull/2577).

## Overview

Multicluster DNS enables DNS records that work across multiple Kubernetes clusters within a ClusterSet. The DNS format follows the pattern:

```
<clusterid>.<svc>.<ns>.svc.<clustersetzone>
```

For example:
```
cluster1.basic-pd-0.basic-pd-peer.tidb.svc.clusterset.local
```

This format allows PD pods in one cluster to discover and communicate with PD pods in other clusters using a standardized naming convention.

## Prerequisites

1. **CoreDNS Multicluster Plugin**: Install and configure the [CoreDNS multicluster plugin](https://github.com/coredns/multicluster) in your Kubernetes clusters.

2. **Network Connectivity**: Ensure that Pod networks are connected across all Kubernetes clusters in the ClusterSet.

3. **Service Export/Import**: Set up Kubernetes [Multi-Cluster Services (MCS) API](https://github.com/kubernetes-sigs/mcs-api) to export and import services across clusters.

## Configuration

### Enabling Multicluster DNS

Add the `multiclusterDNS` section to your TidbCluster spec:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
  namespace: tidb
spec:
  version: v8.5.2
  
  # Enable multicluster DNS
  multiclusterDNS:
    # Unique identifier for this cluster within the ClusterSet
    clusterID: "cluster1"
    # ClusterSet domain name
    clusterSetZone: "clusterset.local"
  
  pd:
    replicas: 3
    # ... other PD configuration
  
  tikv:
    replicas: 3
    # ... other TiKV configuration
  
  tidb:
    replicas: 2
    # ... other TiDB configuration
```

### Parameters

- **`clusterID`** (required): A unique identifier for the Kubernetes cluster within the ClusterSet. This should be a valid DNS label (lowercase alphanumeric characters and hyphens).

- **`clusterSetZone`** (required): The domain name suffix for the ClusterSet. This is typically configured in your CoreDNS multicluster plugin.

## Multi-Cluster Deployment Example

### Cluster 1 Configuration

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
  namespace: tidb
spec:
  version: v8.5.2
  
  multiclusterDNS:
    clusterID: "cluster1"
    clusterSetZone: "clusterset.local"
  
  pd:
    replicas: 2
    requests:
      storage: "20Gi"
  
  tikv:
    replicas: 2
    requests:
      storage: "100Gi"
  
  tidb:
    replicas: 1
    service:
      type: ClusterIP
```

### Cluster 2 Configuration

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
  namespace: tidb
spec:
  version: v8.5.2
  
  multiclusterDNS:
    clusterID: "cluster2"
    clusterSetZone: "clusterset.local"
  
  # Reference the cluster in cluster1
  cluster:
    namespace: tidb
    name: basic
    clusterDomain: "clusterset.local"
  
  pd:
    replicas: 1
    requests:
      storage: "20Gi"
  
  tikv:
    replicas: 1
    requests:
      storage: "100Gi"
  
  tidb:
    replicas: 1
    service:
      type: ClusterIP
```

### Cluster 3 Configuration

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: basic
  namespace: tidb
spec:
  version: v8.5.2
  
  multiclusterDNS:
    clusterID: "cluster3"
    clusterSetZone: "clusterset.local"
  
  # Reference the cluster in cluster1
  cluster:
    namespace: tidb
    name: basic
    clusterDomain: "clusterset.local"
  
  pd:
    replicas: 1
    requests:
      storage: "20Gi"
  
  tikv:
    replicas: 1
    requests:
      storage: "100Gi"
  
  tidb:
    replicas: 1
    service:
      type: ClusterIP
```

## DNS Resolution

With multicluster DNS enabled, PD pods will be addressable using the following format:

```
<clusterid>.<pod-name>.<service-name>.<namespace>.svc.<clustersetzone>
```

Examples:
- `cluster1.basic-pd-0.basic-pd-peer.tidb.svc.clusterset.local`
- `cluster2.basic-pd-0.basic-pd-peer.tidb.svc.clusterset.local`
- `cluster3.basic-pd-0.basic-pd-peer.tidb.svc.clusterset.local`

## Verification

1. Check that PD pods are using multicluster DNS format:

```bash
kubectl -n tidb logs basic-pd-0 | grep "advertise-peer-urls"
```

Expected output should show URLs with the multicluster DNS format.

2. Verify DNS resolution from within a pod:

```bash
kubectl -n tidb exec -it basic-pd-0 -- nslookup cluster1.basic-pd-0.basic-pd-peer.tidb.svc.clusterset.local
```

3. Check PD member list:

```bash
kubectl -n tidb exec -it basic-pd-0 -- pd-ctl member
```

The member names should follow the multicluster DNS format.

## Differences from Standard Deployment

When multicluster DNS is enabled:

1. **DNS Names**: PD member names use the full multicluster DNS format instead of simple hostnames or cluster-domain-suffixed names.

2. **Discovery**: The discovery component is enhanced to parse and handle multicluster DNS format.

3. **Start Scripts**: PD start scripts are modified to use multicluster DNS names for advertise URLs.

4. **Cross-Cluster Communication**: Pods can communicate across clusters using the standardized multicluster DNS names.

## Comparison with AcrossK8s Mode

| Feature | `acrossK8s: true` | `multiclusterDNS` |
|---------|-------------------|-------------------|
| DNS Format | `<pod>.<svc>.<ns>.svc[.<domain>]` | `<clusterid>.<pod>.<svc>.<ns>.svc.<zone>` |
| Standard | Custom | KEP-2577 |
| CoreDNS Plugin | Optional | Required |
| MCS API | Not required | Required |

## Troubleshooting

### DNS Resolution Fails

- Verify CoreDNS multicluster plugin is installed and configured
- Check that the `clusterSetZone` matches your CoreDNS configuration
- Ensure services are properly exported using MCS API

### PD Cluster Formation Issues

- Check PD logs for DNS-related errors
- Verify network connectivity between clusters
- Ensure firewall rules allow cross-cluster communication

### Invalid Configuration

- Ensure `clusterID` is a valid DNS label (lowercase, alphanumeric, hyphens only)
- Verify `clusterSetZone` is a valid domain name
- Both `clusterID` and `clusterSetZone` are required when using multicluster DNS

## References

- [Kubernetes Enhancement Proposal (KEP) 2577: Multicluster Services DNS](https://github.com/kubernetes/enhancements/pull/2577)
- [CoreDNS Multicluster Plugin](https://github.com/coredns/multicluster)
- [Kubernetes Multi-Cluster Services API](https://github.com/kubernetes-sigs/mcs-api)
- [TiDB Operator Multi-Cluster Deployment](https://docs.pingcap.com/tidb-in-kubernetes/stable/deploy-tidb-cluster-across-multiple-kubernetes)

