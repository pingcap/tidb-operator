---
title: Restore Data from GCS
summary: Learn how to restore the backup data from GCS.
aliases: ['/docs/tidb-in-kubernetes/dev/restore-from-gcs/']
---

# Restore Data from GCS

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. For the underlying implementation, [TiDB Lightning](https://pingcap.com/docs/stable/how-to/get-started/tidb-lightning/#tidb-lightning-tutorial) is used to perform the restoration.

The restoration method described in this document is implemented based on CustomResourceDefinition (CRD) in TiDB Operator v1.1 or later versions. For the restoration method implemented based on Helm Charts, refer to [Back up and Restore TiDB Cluster Data Based on Helm Charts](backup-and-restore-using-helm-charts.md).

This document shows an example in which the backup data stored in the specified path on [Google Cloud Storage (GCS)](https://cloud.google.com/storage/docs/) is restored to the TiDB cluster.

## Prerequisites

1. Download [`backup-rbac.yaml`](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml) and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `restore-demo2-tidb-secret` secret which stores the root account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=user=root --from-literal=password=${password} --namespace=test2
    ```

## Required database account privileges

| Privileges | Scope |
|:----|:------|
| SELECT | Tables |
| INSERT | Tables |
| UPDATE | Tables |
| DELETE | Tables |
| CREATE | Databases, tables |
| DROP | Databases, tables |
| ALTER | Tables |

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
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      gcs:
        projectId: ${project_id}
        secretName: gcs-secret
        path: gcs://${backup_path}
      # storageClassName: local-storage
      storageSize: 1Gi
    ```

2. After creating the `Restore` CR, execute the following command to check the restoration status:

    {{< copyable "shell-regular" >}}

     ```shell
     kubectl get rt -n test2 -owide
     ```

In the above example, the backup data stored in the specified `spec.gcs.path` path on GCS is restored to the `spec.to.host` TiDB cluster. For the configuration of GCS, refer to [backup-gcs.yaml](backup-to-gcs.md#ad-hoc-backup-process).

More `Restore` CRs are described as follows:

* `.spec.metadata.namespace`: the namespace where the `Restore` CR is located.
* `.spec.to.host`: the address of the TiDB cluster to be restored.
* `.spec.to.port`: the port of the TiDB cluster to be restored.
* `.spec.to.user`: the accessing user of the TiDB cluster to be restored.
* `.spec.to.tidbSecretName`: the secret of the credential needed by the TiDB cluster to be restored.
* `.spec.storageClassName`: the persistent volume (PV) type specified for the restoration.
* `.spec.storageSize`: the PV size specified for the restoration. This value must be greater than the size of the backed up TiDB cluster.

> **Note:**
>
> TiDB Operator creates a PVC for data recovery. The backup data is downloaded from the remote storage to the PV first, and then restored. If you want to delete this PVC after the recovery is completed, you can refer to [Delete Resource](cheat-sheet.md#delete-resources) to delete the recovery Pod first, and then delete the PVC.

## Troubleshooting

If you encounter any problem during the restore process, refer to [Common Deployment Failures](deploy-failures.md).
