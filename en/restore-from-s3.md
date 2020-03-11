---
title: Restore Data From S3-Compatible Storage
summary: Learn how to restore data from the S3-compatible storage.
category: how-to
---

# Restore Data From S3-Compatible Storage

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. For the underlying implementation, [`loader`](https://pingcap.com/docs/stable/reference/tools/loader) is used to perform the restoration.

The restoration method described in this document is implemented based on CustomResourceDefinition (CRD) in TiDB Operator v1.1 or later versions. For the restoration method implemented based on Helm Charts, refer to [Back up and Restore TiDB Cluster Data Based on Helm Charts](backup-and-restore-using-helm-charts.md).

This document shows an example in which the backup data stored in the specified path on the S3-compatible storage is restored to the TiDB cluster.

## Prerequisites

1. Download [`backup-rbac.yaml`](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml) and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `restore-demo2-tidb-secret` secret which stores the root account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=<password> --namespace=test2
    ```

## Restoration process

1. Create the restore custom resource (CR) and restore the backup data to the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore.yaml
    ```

    The `restore.yaml` file has the following content:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore
      namespace: test2
    spec:
      to:
        host: <tidb-host-ip>
        port: <tidb-port>
        user: <tidb-user>
        secretName: restore-demo2-tidb-secret
      s3:
        provider: ceph
        endpoint: http://10.233.2.161
        secretName: ceph-secret
        path: s3://<path-to-backup>
      storageClassName: local-storage
      storageSize: 1Gi
    ```

2. After creating the `Restore` CR, execute the following command to check the restoration status:

    {{< copyable "shell-regular" >}}

     ```shell
     kubectl get rt -n test2 -owide
     ```

In the above example, the backup data stored in the `spec.s3.path` path on the S3-compatible storage is restored to the `spec.to.host` TiDB cluster. For the configuration of the S3-compatible storage, refer to [backup-s3.yaml](backup-to-s3.md#ad-hoc-backup-process).

More `Restore` CRs are as described follows:

* `.spec.metadata.namespace`: the namespace where the `Restore` CR is located.
* `.spec.to.host`: the address of the TiDB cluster to be restored.
* `.spec.to.port`: the port of the TiDB cluster to be restored.
* `.spec.to.user`: the accessing user of the TiDB cluster to be restored.
* `.spec.to.tidbSecretName`: the secrete of the credential needed by the TiDB cluster to be restored.
* `.spec.storageClassName`: the persistent volume (PV) type specified for the restoration. If this item is not specified, the value of the `default-backup-storage-class-name` parameter (`standard` by default, specified when TiDB Operator is started) is used by default.
* `.spec.storageSize`: the PV size specified for the restoration. This value must be greater than size of the backed up TiDB cluster.
