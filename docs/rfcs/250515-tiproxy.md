# TiProxy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [TiProxy Group and TiDB Group Association](#tiproxy-group-and-tidb-group-association)
    - [1:1](#11)
    - [1:N](#1n)
  - [Configuration Management](#configuration-management)
  - [TLS Handling](#tls-handling)
    - [Cluster TLS](#cluster-tls)
    - [Server TLS](#server-tls)
    - [SQL TLS](#sql-tls)
  - [Test Plan](#test-plan)
  - [Feature Gate](#feature-gate)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a release*.

- [x] (R) This design doc has been discussed and approved
- [ ] (R) Test plan is in place
  - [ ] (R) e2e tests in kind
- [ ] (R) Graduation criteria is in place if required
- [ ] (R) User-facing documentation has been created in [pingcap/docs-tidb-operator]

## Summary

[TiProxy](https://docs.pingcap.com/tidb/stable/tiproxy-overview/) is the official proxy component of PingCAP. It should be supported by tidb operator.

## Motivation

### Goals

- Deploy and manage TiProxy

### Non-Goals

- Manage the relationship between TiProxy and TiDB

## Proposal

### User Stories (Optional)

#### Story 1

As a TiDB user, deploy TiProxy to provide load balancing, connection persistence, and service discovery.

### Risks and Mitigations

- It will be complex to manage the relationship between TiProxy and TiDB, like traffic routing and certificate management.
  - Let users do this by themselves.

## Design Details

### API

Similar to TiDBGroup, TiProxyGroup is a new CRD.

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: TiProxyGroup
metadata:
  name: tiproxy
spec:
  cluster:
    name: basic
  replicas: 3
  schedulePolicies: <>
  template:
    spec:
      image: <>
      version: <>
      volumes: []
      updateStrategy: <>
      resources: <>
      probes: <>
      config: <> 
      preStop:
        sleepSeconds: 10
      server:
        port: 
          sql: 6000
          api: 3080
      security:
        tls:
          # whether enable tls between client and tiproxy
          mysql:
            enabled: true
          # Whether enable tls between tiproxy and TiDB server
          backend:
            enabled: true
```

### TiProxy Group and TiDB Group Association

Within tidb operator, there may be more than one tidb groups in a tidb cluster, so does tiproxy groups. Because once TiProxy connects to PD, it dynamically discovers all TiDB servers within the associated TiDB cluster, it's important to define the relationship between TiProxyGroup and TiDBGroup.

By default, tiproxy will balance traffic between all tidb servers. However, users might have different groups of TiDB servers intended for different workloads (e.g., OLTP vs. OLAP), and they might want to use different TiProxy groups for different TiDB groups. 

To simplify management and implementation, tidb operator will NOT manage the relationship between TiProxyGroup and TiDBGroup, users need to do it by themselves to ensure a TiProxy group will only route traffic to its associated TiDB groups.

#### 1:1

If users want to each TiProxyGroup refer to a single TiDBGroup, they can config TiProxyGroup and TiDBGroup like this:

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: TiProxyGroup
metadata:
  name: tiproxy-oltp
spec:
  template:
    spec:
      config: |
        labels={"group":"oltp"}
        [balance]
        label-name="group"
```

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: TiDBGroup
metadata:
  name: tidb-oltp
spec:
  template:
    spec:
      server:
        # TiDBGroup should add a new field to set server labels dynamically without restarting TiDB.
        # Here is an example:
        labels:
          group: oltp
```

#### 1:N

Similar to 1:1, if users want to each TiProxyGroup refer to multiple TiDBGroups, they just need to config TiDBGroups to use the same label as the TiProxyGroup, e.g `group: oltp`.


### Configuration Management

Users can provide custom configurations via the config field in the CRD. This field accepts a multi-line string representing the TOML content.

The TiDB Operator will:

1. Take the user-provided config as a base.
2. Inject or Override critical configurations:
  - `proxy.pd-addrs`: Dynamically set to the service endpoint of the PD component for the current tidb cluster. This is crucial for TiProxy to discover TiDB servers.
  - `proxy.addr`: Set to listen on a suitable address within the container, typically 0.0.0.0:<proxy-port> (default <proxy-port> is 6000).
  - `proxy.advertise-addr`: Set to the address that TiProxy will advertise to TiDB servers.
  - `api.addr`: Set to listen on a suitable address within the container for its API, typically 0.0.0.0:<api-port> (default <api-port> is 3080).
  - `balance.label-name`: Set to the label name that TiProxy will use to balance traffic.
  - `security`: see "TLS Handling" section

### TLS Handling

#### Cluster TLS

It is used to access TiDB or PD. It's controlled by Cluster CR's `spec.tlsCluster.enabled`.

#### Server TLS

There are two TLS objects:
- `server-tls`: used to provide TLS on SQL port (6000).
- `server-http-tls`: used to provide TLS on HTTP status port (3080). Will use cluster tls.

#### SQL TLS

It's used to access TiDB SQL port. In fact, we don't present any client cert/key when tiproxy connects to tidb server. Instead, only the ca cert of tidb server-side is needed.

### Test Plan

### Feature Gate

No feature gate
