---
title: Restore Data from PV Using BR
summary: Learn how to restore data from Persistent Volume (PV) using BR.
---

# Restore Data from PV Using BR

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. [BR](https://docs.pingcap.com/tidb/dev/backup-and-restore-tool) is used to perform the restore.

PVs in this documentation can be any [Kubernetes supported Persistent Volume types](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes). This document uses NFS as an example PV type, and shows an example in which the backup data stored in the specified path on NFS is restored to the TiDB cluster.

The restore method described in this document is implemented based on Custom Resource Definition (CRD) in TiDB Operator v1.1 or later versions.

## Prerequisites

> **Note:**
>
> If TiDB Operator >= v1.1.10 && TiDB >= v4.0.8, BR will automatically adjust `tikv_gc_life_time`. You do not need to configure `spec.to` fields in the `Restore` CR. In addition, you can skip the steps of creating the `restore-demo2-tidb-secret` secret and [configuring database account privileges](#required-database-account-privileges).

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `restore-demo2-tidb-secret` secret which stores the root account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=<password> --namespace=test2
    ```

3. Ensure that the NFS server is accessible from your Kubernetes cluster.

## Required database account privileges

- The `SELECT` and `UPDATE` privileges of the `mysql.tidb` table: Before and after the restore, the `Restore` CR needs a database account with these privileges to adjust the GC time.

## Restore process

1. Create the `Restore` custom resource (CR), and restore the specified data to your cluster:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f restore.yaml
    ```

    The content of `restore.yaml` file is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-nfs
      namespace: test2
    spec:
      # backupType: full
      br:
        cluster: demo2
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status-addr}
        # concurrency: 4
        # rateLimit: 0
        # checksum: true
      # # Only needed for TiDB Operator < v1.1.10 or TiDB < v4.0.8
      # to:
      #   host: ${tidb_host}
      #   port: ${tidb_port}
      #   user: ${tidb_user}
      #   secretName: restore-demo2-tidb-secret
      local:
        prefix: backup-nfs
        volume:
          name: nfs
          nfs:
            server: ${nfs_server_if}
            path: /nfs
        volumeMount:
          name: nfs
          mountPath: /nfs
    ```

2. After creating the `Restore` CR, execute the following command to check the restore status:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get rt -n test2 -owide
    ```

The example above restores data from the `local://${.spec.local.volumeMount.mountPath}/${.spec.local.prefix}/` directory on NFS to the `demo2` TiDB cluster in the `test2` namespace. For more information about local storage configuration, refer to [Local storage fields](backup-restore-overview.md#local-storage-fields).

In the example above, some parameters in `spec.br` can be ignored, such as `logLevel`, `statusAddr`, `concurrency`, `rateLimit`, `checksum`, `timeAgo`, and `sendCredToTikv`. For more information about BR configuration, refer to [BR fields](backup-restore-overview.md#br-fields).

For more information about the `Restore` CR fields, refer to [Restore CR fields](backup-restore-overview.md#restore-cr-fields).

## Troubleshooting

If you encounter any problem during the restore process, refer to [Common Deployment Failures](deploy-failures.md).
