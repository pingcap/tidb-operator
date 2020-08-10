# Deprecated - Heterogeneous design for TidbCluster

This document presents a design for the Heterogeneous components in one kind for `TidbCluster`.

## Motivation

Currently, the spec for the `TidbCluster` describes a group of `PD`/`TiKV`/`TiDB` instances. For each kind of components, 
their specs are all the same. This design is easy to use and also covers the most cases for using the tidb cluster.

However, as a distributed database with clear and multilayers, the components in each layer could be different from 
each other to meet the different requirements.

For example, as the SQL layer, tidb components could be composed of multiple instances with different resource requests 
and configurations to handle the different workloads (AP/TP query). For the storage layer, tikv components could be 
composed of multiple instances with different store labels which decided the data distribution as a whole storage system.

## Proposal

Add 2 new CRD as `TiKVGroup` and `TiDBGroup` which represents a group of TiDB and TiKV instances. As a whole system, one
TiDB Cluster can be composed of one `TidbCluster` and multiple `TiKVGroup` and `TiDBGroup`. Here is the basic example:

For `TiKVGroup`:

```golang
type TiKVGroup struct {
	Spec   TiKVGroupSpec
	Status TiKVGroupStatus
}

type TiKVGroupSpec struct {
    TiKVSpec
    ClusterName string
}

type TiKVGroupStatus struct {
    TiKVStatus
}
```

As you can see, `TiKVGroupSpec` and `TiKVGroupStatus` reuse the `Spec` and `Status` spec in the `TiKV`. If you already
have a `TidbCluster` running in the kubernetes, and here is one example to show how to create a heterogeneous `TiKVGroup`
joining in your `TidbCluster`:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: TiKVGroup
metadata:
  name: tikv-instances
  namespace: <same-as-your-tidbcluster>
spec:
  clusterName: <your-tidbcluster-name>
  <some tikvspec.....>
```


# See also
 
This proposal is based on prior work in this [google doc](https://docs.google.com/document/d/1MV2bcsCjyYvfCCtwyc8-E18Z69qP_3-FKztpGj4ByEg/edit?usp=sharing)(in Chinese).
