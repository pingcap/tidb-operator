---
title: Migrate from Helm 2 to Helm 3
summary: Learn how to migrate from Helm 2 to Helm 3.
---

# Migrate from Helm 2 to Helm 3

This document describes how to migrate component management from Helm 2 to Helm 3. This document takes TiDB Operator as an example. For other components, you can take similar steps to perform the migration.

For more information about migrating releases managed by Helm 2 to Helm 3, refer to [Helm documentation](https://helm.sh/docs/topics/v2_v3_migration/).

## Migration procedure

In this example, TiDB Operator (`tidb-operator`) managed by Helm 2 is installed in the `tidb-admin` namespace. A `basic` TidbCluster and `basic` TidbMonitor are deployed in the `tidb-cluster` namespace.

{{< copyable "shell-regular" >}}

```bash
helm list
```

```
NAME            REVISION        UPDATED                         STATUS          CHART                   APP VERSION     NAMESPACE
tidb-operator   1               Tue Jan  5 15:28:00 2021        DEPLOYED        tidb-operator-v1.1.8    v1.1.8          tidb-admin
```

1. [Install Helm 3](https://helm.sh/docs/intro/install/).

    Helm 3 takes a different approach from Helm 2 in configuration and data storage. Therefore, when you install Helm 3, there is no risk of overwriting the original configuration and data.

    > **Note:**
    >
    > During the installation, do not let the CLI binary of Helm 3 overwrite that of Helm 2.  For example, you can name Helm 3 CLI binary as `helm3`. (All following examples in this document uses `helm3`.)

2. Install [helm-2to3 plugin](https://github.com/helm/helm-2to3) for Helm 3.

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 plugin install https://github.com/helm/helm-2to3
    ```

    You can verify whether the plugin is successfully installed by running the following command:

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 plugin list
    ```

    ```
    NAME    VERSION DESCRIPTION
    2to3    0.8.0   migrate and cleanup Helm v2 configuration and releases in-place to Helm v3
    ```

3. Migrate the configuration such as repo and plugins from Helm 2 to Helm 3:

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 2to3 move config
    ```

    Before you migrate, you can learn about the possible operations and their consequences by executing `helm3 2to3 move config --dry-run`.

    After the migration, the PingCAP repo is already in Helm 3:

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 repo list
    ```

    ```
    NAME    URL
    pingcap https://charts.pingcap.org/
    ```

4. Migrate the releases from Helm 2 to Helm 3:

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 2to3 convert tidb-operator
    ```

    Before you migrate, you can learn about the possible operations and their consequences by executing `helm3 2to3 convert tidb-operator --dry-run`.

    After the migration, you can view the release corresponding to TiDB Operator in Helm 3:

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 list --namespace=tidb-admin
    ```

    ```
    NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART                   APP VERSION
    tidb-operator   tidb-admin      1               2021-01-05 07:28:00.3545941 +0000 UTC   deployed        tidb-operator-v1.1.8    v1.1.8
    ```

    > **Note:**
    >
    > If the original Helm 2 is Tillerless (Tiller is installed locally via plugins like [helm-tiller](https://github.com/rimusz/helm-tiller) rather than in the Kubernetes cluster), you can migrate the release by adding the `--tiller-out-cluster` flag in the command: `helm3 2to3 convert tidb-operator --tiller-out-cluster`.

5. Confirm that TiDB Operator, TidbCluster, and TidbMonitor run normally.

    To view the running status of TiDB Operator:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get pods --namespace=tidb-admin -l app.kubernetes.io/instance=tidb-operator
    ```

    If all Pods are in the Running state, TiDB Operator runs normally:

    ```
    NAME                                       READY   STATUS    RESTARTS   AGE
    tidb-controller-manager-6d8d5c6d64-b8lv4   1/1     Running   0          2m22s
    tidb-scheduler-644d59b46f-4f6sb            2/2     Running   0          2m22s
    ```

    To view the running status of TidbCluster and TidbMonitor:

    {{< copyable "shell-regular" >}}

    ``` shell
    watch kubectl get pods --namespace=tidb-cluster
    ```

    If all Pods are in the Running state, TidbCluster and TidbMonitor runs normally:

    ```
    NAME                              READY   STATUS    RESTARTS   AGE
    basic-discovery-6bb656bfd-xl5pb   1/1     Running   0          9m9s
    basic-monitor-5fc8589c89-gvgjj    3/3     Running   0          8m58s
    basic-pd-0                        1/1     Running   0          9m8s
    basic-tidb-0                      2/2     Running   0          7m14s
    basic-tikv-0                      1/1     Running   0          8m13s
    ```

6. Clean up Helm 2 data, such as configuration and releases:

    {{< copyable "shell-regular" >}}

    ```bash
    helm3 2to3 cleanup --name=tidb-operator
    ```

    Before you clean up the data, you can learn about the possible operations and their consequences by executing `helm3 2to3 cleanup --name=tidb-operator --dry-run`.

    > **Note:**
    >
    > After the cleanup, you can no longer manage the releases using Helm 2.
