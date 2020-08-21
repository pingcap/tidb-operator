---
title: Deploy DM on Kubernetes
summary: Deploy DM on Kubernetes
category: how-to
---

# Deploy DM on Kubernetes

This document describes how to deploy DM of the new HA architecture with the yamls in this directory.

## Deploy dm-master

Update the rpc configs if necessary in `master/config/config.toml`.

{{< copyable "shell-regular" >}}

``` shell
kubectl apply -k master -n <namespace>
```

> **Note: **
>
> - `3` replicas are deployed by default.
> - `storageClassName` is set to `local-storage` by default.

## Deploy dm-worker

- If you only need to use DM for incremental data migration, no need to create PVC for dm-worker, just deploy it with below command:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -k worker/base -n <namespace>
    ```

- If you need to use DM for both full and incremental data migration, you have to create PVC for dm-worker, deploy it with below command:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -k worker/overlays/full -n <namespace>
    ```

> **Note: **
>
> - `3` replicas are deployed by default.
> - `storageClassName` is set to `local-storage` for PVC by default.
> - If PVCs are created, they are mounted to `/data` directory.

## Create a Task

After you've successfully deployed dm-master and dm-worker, you can define an upstream data source and create a data migration task: <https://docs.pingcap.com/tidb-data-migration/dev/manage-replication-tasks>.
