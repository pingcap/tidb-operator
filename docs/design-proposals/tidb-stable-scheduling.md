# TiDB Stable Scheduling

This document presents a design to schedule new pod of TiDB member to its
previous node in certain circumstances.

## Table of Contents

- [Glossy](#glossy)
- [Motivation](#motivation)
- [Goals](#goals)
  * [Non-Goals](#non-goals)
- [Use Cases](#use-cases)
  * [No need to update IP addresses of TiDB in load balancer outside of the Kubernetes cluster](#no-need-to-update-ip-addresses-of-tidb-in-load-balancer-outside-of-the-kubernetes-cluster)
- [Proposal](#proposal)
  * [Feature gate](#feature-gate)
- [Implementation](#implementation)
- [Alternatives](#alternatives)
  * [Deploy TiDB members on all nodes](#deploy-tidb-members-on-all-nodes)
  * [Leverage load balancer health check mechanism](#leverage-load-balancer-health-check-mechanism)
  * [Use local PV to bind TiDB member to a node](#use-local-pv-to-bind-tidb-member-to-a-node)
- [Limitations](#limitations)
  * [No guarantee if there is another scheduler to schedule pods to TiDB nodes](#no-guarantee-if-there-is-another-scheduler-to-schedule-pods-to-tidb-nodes)
    + [Workarounds](#workarounds)
  * [Cannot schedule new pod of TiDB member back to its node if the node does not meet new requirements](#cannot-schedule-new-pod-of-tidb-member-back-to-its-node-if-the-node-does-not-meet-new-requirements)

## Glossy

- Pod: A deployable object in Kubernetes.
- Node: A node is worker machine to run pods in Kubernetes.
- TiDB cluster: TiDB cluster is database cluster which is constructed by TiDB
  server, PD server and TiKV server. Each TiDB/PD/TiKV server is a cluster too
  which consist of one or more members.
- TiDB server: One of key components of TiDB cluster. It's access point of TiDB
  cluster.
- TiDB member: member of TiDB server which holds a unique network identifier.
- TiDB pod: for each TiDB member, there is at most one running pod. It may
  crash or be replaced by a new pod.
- Load balancer: A component which proxies traffic for applications, e.g. LVS,
  HAProxy, F5, etc.

## Motivation

There are use cases that users need to access TiDB server from outside the
Kubernetes cluster. But in some environments (e.g. on bare-metal machines), we
may lack load balancer solution or need to configure IP addresses of TiDB
services in existing load balancer.

In this scenario, we need to use `NodePort` service. By default, NodePort
services are available cluster-wide on all nodes but cannot propagate client's
IP address to the end pods and may cause a second hop. 

For propagating client's IP address and better performance, we can
set `externalTrafficPolicy` of service to `Local`. A side-effect is the service of
TiDB will be accessible only on the nodes which a running TiDB pod. To avoid
manual intervention to update IP addresses in load balancer when performing
a rolling update, we prefer to schedule new pod of TiDB member to its previous node.

## Goals

- Able to schedule new pod of TiDB member to its previous node

### Non-Goals

- Stable scheduling for TiKV/PD pods
- Guarantee that new pod of TiDB member will be scheduled to its node if
  there is another scheduler in cluster which may schedule pods to its node
- Guarantee that new pod of TiDB member will be scheduled to its node if some
  scheduling requirements changed

## Use Cases

### No need to update IP addresses of TiDB in load balancer outside of the Kubernetes cluster

When a TiDB cluster is running in a dedicated Kubernetes cluster or nodes of
TiDB cluster are reserved for it. After a rolling update of TiDB cluster is
done, new pods of TiDB members will be scheduled to their previous nodes. User
does not need to update IP addresses in load balancer.

## Proposal

Currently, we have tidb-scheduler to schedule all pods of TiDB cluster. We can
write a new predicate function for pods of TiDB server. In this new predicate
function we can choose the previous node of TiDB member if it exists in candidate
nodes.

Note that it's not possible for tidb-scheduler to schedule the new pod of TiDB
member back to its node if the node does not meet the new scheduling
requirements (e.g. CPU/Memory, Taints).

Here is the workflow when the user performs a rolling update for TiDB cluster
`demo` which has 3 replicas of TiDB members:

- `demo-tidb-2` is running on the node kube-node-2
- the user performs an update
- after the pod of `demo-tidb-2` is terminated, the new pod of `demo-tidb-2` is
  created
- kube-scheduler sends feasible nodes which can run the pod `demo-tidb-2` to
  tidb-scheduler (scheduler extender)
- tidb-scheduler filters out other nodes if the original node exists in these
  nodes, kube-scheduler will choose `kube-node-2` to run `demo-tidb-2`
  - note that if `kube-node-2` exist in the nodes sent from kube-scheduler, it
    meets all criteria to `demo-tidb-2`
- tidb-scheduler does nothing if the original node does not exist in these
  nodes (e.g. not enough resources left for demo-tidb-2 if another pod is
  assigned to kube-node-2 after demo-tidb-2 is deleted), kube-scheduler will
  prioritize all feasible nodes to find the best match

### Feature gate

Add a new flag in tidb-scheduler which accepts a comma-separated list of
string.

```
tidb-scheduler --features StableScheduling=true
```

tidb-scheduler will enable this functionality only when `StableScheduling`
feature is enabled.

## Implementation

At first, we track assigned node of TiDB member in status of TiDBCluster.

```
type TiDBMember struct {
  ...
	// Node hosting pod of this TiDB member.
	NodeName string `json:"node,omitempty"`
}
```

In new predicate `StableScheduling` in tidb-scheduler, we filter out other
nodes for TiDB pod if previous node for this TiDB member exists in candidate
nodes.

## Alternatives

### Deploy TiDB members on all nodes

If TiDB members are running on all nodes, clients can access TiDB server with
any node IP address.

We can achieve this by using DaemonSet or Deployment with pod anti-affinity.
But if the Kubernetes cluster is large, to deploy TiDB members of each TiDB
cluster on every node is very inefficient and will consume too many resources.

### Leverage load balancer health check mechanism

Load balancers often have health checks on its backends, it will remove invalid
backend automatically. We can add all nodes into load balancer. Drawbacks with
this solution are:

- maybe too heavy for LB if the Kubernetes cluster is large
  - `NumberOfTiDBClusters` x `NumberOfNodes` ports should be health checked
- need to add every new node into the backend of LB
- hard to monitor LB (not all failed backends must be fixed)

Some of them can be alleviated by restricting TiDB members in a fixed set of
nodes (by using NodeSelector/NodeAffinity/Taints&Tolerations). This requires
the user to pre-select the nodes to run TiDB pods.

### Use local PV to bind TiDB member to a node

Local PVs are local resources of a node. If a TiDB member is using a local PV
on a node, this node is the only available node for it.

This solution is like using a fixed set of nodes to run TiDB pods and does not
require the user to pre-select nodes. But every pod is bound to one node, its
new pod will be pending forever if resources of the node are consumed by other
nodes.

Another drawback is it requires to bind a dummy PV for each TiDB pod which is
complex to manage and will confuse the user.

## Limitations

### No guarantee if there is another scheduler to schedule pods to TiDB nodes

This is because when old pod of TiDB member is terminated, resources of its
node are released and other schedulers may schedule other pods onto this node.
When `tidb-scheduler` schedules new pod of TiDB member, its previous node may
not fit.

#### Workarounds

- Reserve nodes for TiDB members (e.g. by taints)

### Cannot schedule new pod of TiDB member back to its node if the node does not meet new requirements

If we upgrade TiDB pods to request more resources, it is possible that its node
may not have enough resources for the new pod.

It applies if some other scheduling requirements are changed, e.g.

- NodeSelector
- Tolerations
