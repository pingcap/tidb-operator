---
title: Migrate TiDB to Kubernetes
summary: Learn how to migrate a TiDB cluster deployed in the physical or virtual machine to a Kubernetes cluster.
---

# Migrate TiDB to Kubernetes

This document describes how to migrate a TiDB cluster deployed in the physical or virtual machine to a Kubernetes cluster, without using any backup and restore tool.

## Prerequisites

- The physical or virtual machines outside Kubernetes have network access to the Pods in Kubernetes.
- The physical or virtual machines outside Kubernetes can resolve the domain name of the Pods in Kubernetes. (See [Step 1](#step-1-configure-dns-service-in-all-nodes-of-the-cluster-to-be-migrated) for configuration details.)
- The cluster to be migrated (that is, the source cluster) does not [enable TLS between components](enable-tls-between-components.md).

## Step 1: Configure DNS service in all nodes of the cluster to be migrated

1. Get the Pod IP address list of the endpoints of the CoreDNS or kube-dns service of the Kubernetes cluster:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl describe svc/kube-dns -n kube-system
    ```

2. Modify the `/etc/resolv.conf` configuration of the source cluster node, and add the following content to the configuration file:

    {{< copyable "" >}}

    ```
    search default.svc.cluster.local svc.cluster.local cluster.local
    nameserver <CoreDNS Pod_IP_1>
    nameserver <CoreDNS Pod_IP_2>
    nameserver <CoreDNS Pod_IP_n>
    ```

3. Test whether the node can successfully resolve the domain name of the Pods in Kubernetes:

    ```bash
    $ ping basic-pd-2.basic-pd-peer.blade.svc
    PING basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178) 56(84) bytes of data.
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=1 ttl=61 time=0.213 ms
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=2 ttl=61 time=0.175 ms
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=3 ttl=61 time=0.188 ms
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=4 ttl=61 time=0.157 ms
    ```

## Step 2: Create a TiDB cluster in Kubernetes

1. Get the PD node address and port of the source cluster via [PD Control](https://docs.pingcap.com/tidb/stable/pd-control):

    {{< copyable "shell-regular" >}}

    ```bash
    pd-ctl -u http://<address>:<port> member | jq '.members | .[] | .client_urls'
    ```

2. Create the target TiDB cluster in Kubernetes, which must have at least 3 TiKV nodes. Specify the PD node address of the source cluster in the `spec.pdAddresses` field (starting with `http://`):

    ```yaml
    spec
      ...
      pdAddresses:
      - http://pd1_addr:port
      - http://pd2_addr:port
      - http://pd3_addr:port
    ```

3. Confirm that the source cluster and the target cluster compose of a new cluster that runs normally:

    - Get the number and state of stores in the new cluster:

        {{< copyable "shell-regular" >}}

        ```bash
        # Get the number of stores
        pd-ctl -u http://<address>:<port> store | jq '.count'
        # Get the state of stores
        pd-ctl -u http://<address>:<port> store | jq '.stores | .[] | .store.state_name'
        ```

    - [Access the TiDB cluster in Kubernetes](access-tidb.md) via MySQL client.

## Step 3: Scale in the TiDB nodes of the source cluster

Remove all TiDB nodes of the source cluster:

- If the source cluster is deployed using TiUP, refer to [Scale in a TiDB/PD/TiKV cluster](https://docs.pingcap.com/tidb/stable/scale-tidb-using-tiup#scale-in-a-tidbpdtikv-cluster).

- If the source cluster is deployed using TiDB Ansible, refer to [Decrease the capacity of a TiDB node](https://docs.pingcap.com/tidb/stable/scale-tidb-using-ansible#decrease-the-capacity-of-a-tidb-node).

> **Note:**
>
> If you access the source TiDB cluster via load balancer or database middleware, you need to first modify the configuration to route your application traffic to the target TiDB cluster. Otherwise, your application might be affected.

## Step 4: Scale in the TiKV nodes of the source cluster

Remove all TiKV nodes of the source cluster:

- If the source cluster is deployed using TiUP, refer to [Scale in a TiDB/PD/TiKV cluster](https://docs.pingcap.com/tidb/stable/scale-tidb-using-tiup#scale-in-a-tidbpdtikv-cluster).

- If the source cluster is deployed using TiDB Ansible, refer to [Decrease the capacity of a TiKV node](https://docs.pingcap.com/tidb/stable/scale-tidb-using-ansible#decrease-the-capacity-of-a-tikv-node).

> **Note:**
>
> * You need to scale in the TiKV nodes one by one. Wait until the store state of one TiKV node becomes "tombstone" and then scale in the next TiKV node.
> * You can view the store state using PD Control.

## Step 5: Scale in the PD nodes of the source cluster

Remove all PD nodes of the source cluster:

- If the source cluster is deployed using TiUP, refer to [Scale in a TiDB/PD/TiKV cluster](https://docs.pingcap.com/tidb/stable/scale-tidb-using-tiup#scale-in-a-tidbpdtikv-cluster).

- If the source cluster is deployed using TiDB Ansible, refer to [Decrease the capacity of a PD node](https://docs.pingcap.com/tidb/stable/scale-tidb-using-ansible#decrease-the-capacity-of-a-pd-node).

## Step 6: Delete the `spec.pdAddresses` field

To avoid confusion for further operations on the cluster, it is recommended that you delete the `spec.pdAddresses` field in the new cluster after the migration.
