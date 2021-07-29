---
title: Replicate Data to TLS-enabled Downstream Services Using TiCDC Components
summary: Learn how to get TiCDC components to replicate data to TLS-enabled downstream services.
---

# Replicate Data to TLS-enabled Downstream Services with TiCDC Components

This document describes how to get the TiCDC components to replicate data to TLS-enabled downstream services on Kubernetes.

## Preparations

Before you begin, do the following preparations:

- Deploy a downstream service, and enable the TLS authentication on the client.
- Generate the key file required for the client to access the downstream service.

## Configure TiCDC to replicate data to a TLS-enabled downstream service

1. Create a Kubernetes Secret object that contains a client TLS certificate used to access the downstream service. You can get the certificate from the key file you generated for the client.

    {{< copyable "shell-regular" >}}

    ```shell
      kubectl create secret generic ${secret_name} --namespace=${cluster_namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

2. Mount the certificate file to the TiCDC Pod.

    * If you have not deployed a TiDB cluster yet, add the `spec.ticdc.tlsClientSecretNames` field to the TidbCluster CR definition, and then deploy the TiDB cluster.

    * If you have already deployed a TiDB cluster, run `kubectl edit tc ${cluster_name} -n ${cluster_namespace}`, add the `spec.tiddc.tlsClientSecretNames` field, and then wait for the TiCDC pod to automatically roll over for updates.

    ```yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
      name: ${cluster_name}
      namespace: ${cluster_namespace}
    spec:
      # ...
      ticdc:
        baseImage: pingcap/ticdc
        version: "v5.0.1"
        # ...
        tlsClientSecretNames:
        - ${secret_name}
    ```

    Once the TiCDC Pod is running, the created Kubernetes Secret object is mounted to the TiCDC Pod. You can get the mounted key file in the `/var/lib/sink-tls/${secret_name}` directory of the Pod.

3. Create a replication task using the `cdc cli` tool.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl exec ${cluster_name}-ticdc-0 -- /cdc cli changefeed create --pd=https://${cluster_name}-pd:2379 --sink-uri="mysql://${user}:{$password}@${downstream_service}/?ssl-ca=/var/lib/sink-tls/${secret_name}/ca.crt&ssl-cert=/var/lib/sink-tls/${secret_name}/tls.crt&ssl-key=/var/lib/sink-tls/${secret_name}/tls.key"
    ```
