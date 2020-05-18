---
title: Resource and Disaster Recovery Configurations
summary: Learn the resource and disaster recovery configurations of a TiDB cluster in Kubernetes.
category: reference
---

# Resource and Disaster Recovery Configurations

This document introduces the following items of a TiDB cluster in Kubernetes:

+ The configuration of resources
+ The configuration of disaster recovery

## Resource configuration

Before deploying a TiDB cluster, it is necessary to configure the resources for each component of the cluster depending on your needs. PD, TiKV and TiDB are the core service components of a TiDB cluster. In a production environment, their resource configurations must be specified according to component needs. Detailed reference: [Hardware Recommendations](https://pingcap.com/docs/v3.0/how-to/deploy/hardware-recommendations/).

To ensure the proper scheduling and stable operation of the components of the TiDB cluster in Kubernetes, it is recommended to set Guaranteed-level QoS by letting `limits` equal to `requests` when configuring resources. Detailed reference: [Configure Quality of Service for Pods](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/).

If you are using a NUMA-based CPU, you need to enable `Static`'s CPU management policy on the node for better performance. In order to allow the TiDB cluster component to monopolize the corresponding CPU resources, the CPU quota must be an integer greater than or equal to `1` besides setting Guaranteed-level QoS as mentioned above. Detailed reference: [Control CPU Management Policies on the Node](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies).

## Disaster recovery configuration

TiDB is a distributed database and its disaster recovery must ensure that when any physical topology node fails, not only the service is unaffected, but also the data is complete and available. The two configurations of disaster recovery are described separately as follows.

### Disaster recovery of TiDB service

The disaster recovery of TiDB service is essentially based on Kubernetes' scheduling capabilities. To optimize scheduling, TiDB Operator provides a custom scheduler that guarantees the disaster recovery of TiDB service at the host level through the specified scheduling algorithm. Currently, the TiDB cluster uses this scheduler as the default scheduler, which is configured through the item `schedulerName` in the above table.

Disaster recovery at other levels (such as rack, zone, region) are guaranteed by Affinity's `PodAntiAffinity`. `PodAntiAffinity` can avoid the situation where different instances of the same component are deployed on the same physical topology node. In this way, disaster recovery is achieved. Detailed user guide for Affinity: [Affinity & AntiAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity).

The following is an example of a typical disaster recovery setup:

{{< copyable "" >}}

```shell
affinity:
 podAntiAffinity:
   preferredDuringSchedulingIgnoredDuringExecution:
   # this term works when the nodes have the label named region
   - weight: 10
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "region"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named zone
   - weight: 20
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "zone"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named rack
   - weight: 40
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "rack"
       namespaces:
       - ${namespace}
   # this term works when the nodes have the label named kubernetes.io/hostname
   - weight: 80
     podAffinityTerm:
       labelSelector:
         matchLabels:
           app.kubernetes.io/instance: ${cluster_name}
           app.kubernetes.io/component: "pd"
       topologyKey: "kubernetes.io/hostname"
       namespaces:
       - ${namespace}
```

### Disaster recovery of data

Before configuring the data disaster recovery, read [Information Configuration of the Cluster Typology](https://pingcap.com/docs/v3.0/how-to/deploy/geographic-redundancy/location-awareness/) which describes how the disaster recovery of the TiDB cluster is implemented.

To add the data disaster recovery feature in Kubernetes:

1. Set the label collection of topological location for PD

    Replace the `location-labels` information in the `pd.config` with the label collection that describes the topological location on the nodes in the Kubernetes cluster.

    > **Note:**
    >
    > * For PD versions < v3.0.9, the `/` in the label name is not supported.
    > * If you configure `host` in the `location-labels`, TiDB Operator will get the value from the `kubernetes.io/hostname` in the node label.

2. Set the topological information of the Node where the TiKV node is located.

    TiDB Operator automatically obtains the topological information of the Node for TiKV and calls the PD interface to set this information as the information of TiKV's store labels. Based on this topological information, the TiDB cluster schedules the replicas of the data.

    If the Node of the current Kubernetes cluster does not have a label indicating the topological location, or if the existing label name of topology contains `/`, you can manually add a label to the Node by running the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl label node ${node_name} region=${region_name} zone=${zone_name} rack=${rack_name} kubernetes.io/hostname=${host_name}
    ```

    In the command above, `region`, `zone`, `rack`, and `kubernetes.io/hostname` are just examples. The name and number of the label to be added can be arbitrarily defined, as long as it conforms to the specification and is consistent with the labels set by `location-labels` in `pd.config`.
