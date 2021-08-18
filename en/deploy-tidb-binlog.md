---
title: Deploy TiDB Binlog
summary: Learn how to deploy TiDB Binlog for a TiDB cluster in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/deploy-tidb-binlog/']
---

# Deploy TiDB Binlog

This document describes how to maintain [TiDB Binlog](https://pingcap.com/docs/stable/tidb-binlog/tidb-binlog-overview/) of a TiDB cluster in Kubernetes.

## Prerequisites

- [Deploy TiDB Operator](deploy-tidb-operator.md);
- [Install Helm](tidb-toolkit.md#use-helm) and configure it with the official PingCAP chart.

## Deploy TiDB Binlog in a TiDB cluster

TiDB Binlog is disabled in the TiDB cluster by default. To create a TiDB cluster with TiDB Binlog enabled, or enable TiDB Binlog in an existing TiDB cluster, take the following steps.

### Deploy Pump

1. Modify the `TidbCluster` CR file to add the Pump configuration.

    For example:

    ```yaml
    spec:
      ...
      pump:
        baseImage: pingcap/tidb-binlog
        version: v5.1.1
        replicas: 1
        storageClassName: local-storage
        requests:
          storage: 30Gi
        schedulerName: default-scheduler
        config:
          addr: 0.0.0.0:8250
          gc: 7
          heartbeat-interval: 2
    ```

    Since v1.1.6, TiDB Operator supports passing raw TOML configuration to the component:

    ```yaml
    spec:
      ...
      pump:
        baseImage: pingcap/tidb-binlog
        version: v5.1.1
        replicas: 1
        storageClassName: local-storage
        requests:
          storage: 30Gi
        schedulerName: default-scheduler
        config: |
          addr = "0.0.0.0:8250"
          gc = 7
          heartbeat-interval = 2
    ```

    Edit `version`, `replicas`, `storageClassName`, and `requests.storage` according to your cluster.

    To deploy Enterprise Edition of Pump, edit the YAML file above to set `spec.pump.baseImage` to the enterprise image (`pingcap/tidb-binlog-enterprise`).

    For example:

    ```yaml
    spec:
      pump:
        baseImage: pingcap/tidb-binlog-enterprise
    ```

2. Set affinity and anti-affinity for TiDB and Pump.

    If you enable TiDB Binlog in the production environment, it is recommended to set affinity and anti-affinity for TiDB and the Pump component; if you enable TiDB Binlog in a test environment on the internal network, you can skip this step.

    By default, the affinity of TiDB and Pump is set to `{}`. Currently, each TiDB instance does not have a corresponding Pump instance by default. When TiDB Binlog is enabled, if Pump and TiDB are separately deployed and network isolation occurs, and `ignore-error` is enabled in TiDB components, TiDB loses binlogs.

    In this situation, it is recommended to deploy a TiDB instance and a Pump instance on the same node using the affinity feature, and to split Pump instances on different nodes using the anti-affinity feature. For each node, only one Pump instance is required. The steps are as follows:

    * Configure `spec.tidb.affinity` as follows:

        ```yaml
        spec:
          tidb:
            affinity:
              podAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - weight: 100
                  podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: "app.kubernetes.io/component"
                        operator: In
                        values:
                        - "pump"
                      - key: "app.kubernetes.io/managed-by"
                        operator: In
                        values:
                        - "tidb-operator"
                      - key: "app.kubernetes.io/name"
                        operator: In
                        values:
                        - "tidb-cluster"
                      - key: "app.kubernetes.io/instance"
                        operator: In
                        values:
                        - ${cluster_name}
                    topologyKey: kubernetes.io/hostname
        ```

    * Configure `spec.pump.affinity` as follows:

        ```yaml
        spec:
          pump:
            affinity:
              podAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - weight: 100
                  podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: "app.kubernetes.io/component"
                        operator: In
                        values:
                        - "tidb"
                      - key: "app.kubernetes.io/managed-by"
                        operator: In
                        values:
                        - "tidb-operator"
                      - key: "app.kubernetes.io/name"
                        operator: In
                        values:
                        - "tidb-cluster"
                      - key: "app.kubernetes.io/instance"
                        operator: In
                        values:
                        - ${cluster_name}
                    topologyKey: kubernetes.io/hostname
              podAntiAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - weight: 100
                  podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: "app.kubernetes.io/component"
                        operator: In
                        values:
                        - "pump"
                      - key: "app.kubernetes.io/managed-by"
                        operator: In
                        values:
                        - "tidb-operator"
                      - key: "app.kubernetes.io/name"
                        operator: In
                        values:
                        - "tidb-cluster"
                      - key: "app.kubernetes.io/instance"
                        operator: In
                        values:
                        - ${cluster_name}
                    topologyKey: kubernetes.io/hostname
        ```

    > **Note:**
    >
    > If you update the affinity configuration of the TiDB components, it will cause rolling updates of the TiDB components in the cluster.

## Deploy Drainer

To deploy multiple drainers using the `tidb-drainer` Helm chart for a TiDB cluster, take the following steps:

1. Make sure that the PingCAP Helm repository is up to date:

    {{< copyable "shell-regular" >}}

    ```shell
    helm repo update
    ```

    {{< copyable "shell-regular" >}}

    ```shell
    helm search repo tidb-drainer -l
    ```

2. Get the default `values.yaml` file to facilitate customization:

    {{< copyable "shell-regular" >}}

    ```shell
    helm inspect values pingcap/tidb-drainer --version=${chart_version} > values.yaml
    ```

3. Modify the `values.yaml` file to specify the source TiDB cluster and the downstream database of the drainer. Here is an example:

    ```yaml
    clusterName: example-tidb
    clusterVersion: v5.1.1
    baseImage:pingcap/tidb-binlog
    storageClassName: local-storage
    storage: 10Gi
    initialCommitTs: "-1"
    config: |
      detect-interval = 10
      [syncer]
      worker-count = 16
      txn-batch = 20
      disable-dispatch = false
      ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql"
      safe-mode = false
      db-type = "tidb"
      [syncer.to]
      host = "downstream-tidb"
      user = "root"
      password = ""
      port = 4000
    ```

    The `clusterName` and `clusterVersion` must match the desired source TiDB cluster.

    The `initialCommitTs` is the starting commit timestamp of data replication when Drainer has no checkpoint. The value must be set as a string type, such as `"424364429251444742"`.

    For complete configuration details, refer to [TiDB Binlog Drainer Configurations in Kubernetes](configure-tidb-binlog-drainer.md).

    To deploy Enterprise Edition of Drainer, edit the YAML file above to set `baseImage` to the enterprise image (`pingcap/tidb-binlog-enterprise`).

    For example:

    ```yaml
    ...
    clusterVersion: v5.1.1
    baseImage: pingcap/tidb-binlog-enterprise
    ...
    ```

4. Deploy Drainer:

    {{< copyable "shell-regular" >}}

    ```shell
    helm install ${release_name} pingcap/tidb-drainer --namespace=${namespace} --version=${chart_version} -f values.yaml
    ```

    If the server does not have an external network, refer to [deploy the TiDB cluster](deploy-on-general-kubernetes.md#deploy-the-tidb-cluster) to download the required Docker image on the machine with an external network and upload it to the server.

    > **Note:**
    >
    > This chart must be installed to the same namespace as the source TiDB cluster.

## Enable TLS

### Enable TLS between TiDB components

If you want to enable TLS for the TiDB cluster and TiDB Binlog, refer to [Enable TLS between Components](enable-tls-between-components.md).

After you have created a secret and started a TiDB cluster with Pump, edit the `values.yaml` file to set the `tlsCluster.enabled` value to `true`, and configure the corresponding `certAllowedCN`:

```yaml
...
tlsCluster:
  enabled: true
  # certAllowedCN:
  #  - TiDB
...
```

### Enable TLS between Drainer and the downstream database

If you set the downstream database of `tidb-drainer` to `mysql/tidb`, and if you want to enable TLS between Drainer and the downstream database, take the following steps.

1. Create a secret that contains the TLS information of the downstream database.

    ```bash
    kubectl create secret generic ${downstream_database_secret_name} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

    `tidb-drainer` saves the checkpoint in the downstream database by default, so you only need to configure `tlsSyncer.tlsClientSecretName` and the corresponding `cerAllowedCN`:

    ```yaml
    tlsSyncer:
      tlsClientSecretName: ${downstream_database_secret_name}
      # certAllowedCN:
      #  - TiDB
    ```

2. To save the checkpoint of `tidb-drainer` to **other databases that have enabled TLS**, create a secret that contains the TLS information of the checkpoint database:

    ```bash
    kubectl create secret generic ${checkpoint_tidb_client_secret} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

    Edit the `values.yaml` file to set the `tlsSyncer.checkpoint.tlsClientSecretName` value to `${checkpoint_tidb_client_secret}`, and configure the corresponding `certAllowedCN`:

    ```yaml
    ...
    tlsSyncer: {}
      tlsClientSecretName: ${downstream_database_secret_name}
      # certAllowedCN:
      #  - TiDB
      checkpoint:
        tlsClientSecretName: ${checkpoint_tidb_client_secret}
        # certAllowedCN:
        #  - TiDB
    ...
    ```

## Remove Pump/Drainer nodes

For details on how to maintain the node state of the TiDB Binlog cluster, refer to [Starting and exiting a Pump or Drainer process](https://docs.pingcap.com/tidb/stable/maintain-tidb-binlog-cluster#starting-and-exiting-a-pump-or-drainer-process).

If you want to remove the TiDB Binlog component completely, it is recommended that you first remove Pump nodes and then remove Drainer nodes.

If TLS is enabled for the TiDB Binlog component to be removed, write the following content into `binlog.yaml` and execute `kubectl apply -f binlog.yaml` to start a Pod that is mounted with the TLS file and the `binlogctl` tool.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: binlogctl
spec:
  containers:
  - name: binlogctl
    image: pingcap/tidb-binlog:${tidb_version}
    command: ['/bin/sh']
    stdin: true
    stdinOnce: true
    tty: true
    volumeMounts:
      - name: binlog-tls
        mountPath: /etc/binlog-tls
  volumes:
    - name: binlog-tls
      secret:
        secretName: ${cluster_name}-cluster-client-secret
```

### Scale in Pump

To scale in Pump, you need to take a single Pump node offline, and execute `kubectl edit tc ${cluster_name} -n ${namespace}` to reduce the value of `replicas` of Pump by 1. Repeat the operations on each node.

The steps are as follows:

1. Take the Pump node offline.

    Assume there are three Pump nodes in the cluster. You need to take the third node offline and replace `${ordinal_id}` with `2`. (`${tidb_version}` is the current TiDB version.)

    - If TLS is not enabled for Pump, create a Pod to take Pump offline:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl run offline-pump-${ordinal_id} --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd offline-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250
        ```

    - If TLS is enabled for Pump, use the previously started Pod to take Pump offline:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls "https://${cluster_name}-pd:2379" -cmd offline-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250 -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
        ```

    View the log of Pump by executing the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl logs -f -n ${namespace} ${release_name}-pump-${ordinal_id}
    ```

    If `pump offline, please delete my pod` is output, this node is successfully taken offline.

2. Delete the corresponding Pump Pod:

    Execute `kubectl edit tc ${cluster_name} -n ${namespace}` to change `spec.pump.replicas` to `2`, and wait until the Pump Pod is taken offline and deleted automatically.

3. (Optional) Force Pump to go offline:

    If the offline operation fails, the Pump Pod will not output `pump offline, please delete my pod`. At this time, you can force Pump to go offline, that is, taking Step 2 to reduce the value of `spec.pump.replicas` and mark Pump as `offline` after the Pump Pod is deleted completely.

    - If TLS is not enabled for Pump, mark Pump as `offline`:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl run update-pump-${ordinal_id} --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd update-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250 --state offline
        ```

    - If TLS is enabled for Pump, mark Pump as `offline` using the previously started Pod:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls=https://${cluster_name}-pd:2379 -cmd update-pump -node-id ${cluster_name}-pump-${ordinal_id}:8250 --state offline -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
        ```

### Remove Pump nodes completely

> **Note:**
>
> - Before performing the following steps, you need to have at least one Pump node in the cluster. If you have scaled in Pump nodes to `0`, you need to scale out Pump at least to `1` node before you perform the removing operation in this section.
> - To scale out the Pump to `1`, execute `kubectl edit tc ${tidb-cluster} -n ${namespace}` and modify the `spec.pump.replicas` to `1`.

1. Before removing Pump nodes, execute `kubectl edit tc ${cluster_name} -n ${namespace}` and set `spec.tidb.binlogEnabled` to `false`. After the TiDB Pods are rolling updated, you can remove the Pump nodes.

    If you directly remove Pump nodes, it might cause TiDB failure because TiDB has no Pump nodes to write into.

2. Refer to [Scale in Pump](#scale-in-pump) to scale in Pump to `0`.

3. Execute `kubectl edit tc ${cluster_name} -n ${namespace}` and delete all configuration items of `spec.pump`.

4. Execute `kubectl delete sts ${cluster_name}-pump -n ${namespace}` to delete the StatefulSet resources of Pump.

5. View PVCs used by the Pump cluster by executing `kubectl get pvc -n ${namespace} -l app.kubernetes.io/component=pump`. Then delete all the PVC resources of Pump by executing `kubectl delete pvc -l app.kubernetes.io/component=pump -n ${namespace}`.

### Remove Drainer nodes

1. Take Drainer nodes offline:

    In the following commands, `${drainer_node_id}` is the node ID of the Drainer node to be taken offline. If you have configured `drainerName` in `values.yaml` of Helm, the value of `${drainer_node_id}` is `${drainer_name}-0`; otherwise, the value of `${drainer_node_id}` is `${cluster_name}-${release_name}-drainer-0`.

    - If TLS is not enabled for Drainer, create a Pod to take Drainer offline:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl run offline-drainer-0 --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd offline-drainer -node-id ${drainer_node_id}:8249
        ```

    - If TLS is enabled for Drainer, use the previously started Pod to take Drainer offline:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls "https://${cluster_name}-pd:2379" -cmd offline-drainer -node-id ${drainer_node_id}:8249 -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
        ```

    View the log of Drainer by executing the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl logs -f -n ${namespace} ${drainer_node_id}
    ```

    If `drainer offline, please delete my pod` is output, this node is successfully taken offline.

2. Delete the corresponding Drainer Pod:

    Execute `helm uninstall ${release_name} -n ${namespace}` to delete the Drainer Pod.

    If you no longer need Drainer, execute `kubectl delete pvc data-${drainer_node_id} -n ${namespace}` to delete the PVC resources of Drainer.

3. (Optional) Force Drainer to go offline:

    If the offline operation fails, the Drainer Pod will not output `drainer offline, please delete my pod`. At this time, you can force Drainer to go offline, that is, taking Step 2 to delete the Drainer Pod and mark Drainer as `offline`.

    - If TLS is not enabled for Drainer, mark Drainer as `offline`:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl run update-drainer-${ordinal_id} --image=pingcap/tidb-binlog:${tidb_version} --namespace=${namespace} --restart=OnFailure -- /binlogctl -pd-urls=http://${cluster_name}-pd:2379 -cmd update-drainer -node-id ${drainer_node_id}:8249 --state offline
        ```

    - If TLS is enabled for Drainer, use the previously started Pod to take Drainer offline:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl exec binlogctl -n ${namespace} -- /binlogctl -pd-urls=https://${cluster_name}-pd:2379 -cmd update-drainer -node-id ${drainer_node_id}:8249 --state offline -ssl-ca "/etc/binlog-tls/ca.crt" -ssl-cert "/etc/binlog-tls/tls.crt" -ssl-key "/etc/binlog-tls/tls.key"
        ```
