---
title: RBAC rules required by TiDB Operator
summary: Introduces the RBAC rules required by TiDB Operator.
---

# RBAC rules required by TiDB Operator

The [role-based access control (RBAC)](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) rules implemented in Kubernetes use Role or ClusterRole for management, and use RoleBinding or ClusterRoleBinding to grant permissions to a user or a group of users.

## Manage TiDB clusters at the cluster level

If the default setting `clusterScoped=true` is unchanged during the TiDB Operator deployment, TiDB Operator manages all TiDB clusters within a Kubernetes cluster.

To check the ClusterRole created for TiDB Operator, run the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl get clusterrole | grep tidb
```

The example output is as follows:

```shell
tidb-operator:tidb-controller-manager                                  2021-05-04T13:08:55Z
tidb-operator:tidb-scheduler                                           2021-05-04T13:08:55Z
```

In the output above:

* `tidb-operator:tidb-controller-manager` is the ClusterRole created for the `tidb-controller-manager` Pod.
* `tidb-operator:tidb-scheduler` is the ClusterRole created for the `tidb-scheduler` Pod.

### `tidb-controller-manager` ClusterRole permissions

The following table lists the permissions corresponding to the `tidb-controller-manager` ClusterRole.

| Resource       | Non-resource URLs        | Resource name          | Action                        | Explanation |
| --------- | ---------     | ----------------- | --------------  | ------- |
| events                                        | -                 | -              | [*]                                     | Exports event information |
| services                                      | -                 | -              | [*]                                     | Control the access of the service resources |
| statefulsets.apps.pingcap.com/status          | -                 | -              | [*]      | Control the access of the StatefulSet resource when `AdvancedStatefulSet=true`. For more information, see [Advanced StatefulSet Controller](advanced-statefulset.md). |
| statefulsets.apps.pingcap.com                 | -                 | -              | [*]                                              | Control the access of the StatefulSet resource when `AdvancedStatefulSet=true`. For more information, see [Advanced StatefulSet Controller](advanced-statefulset.md). |
| controllerrevisions.apps                      | -                 | -              | [*]                                              | Control the version of Kubernetes StatefulSet/Daemonset |
| deployments.apps                              | -                 | -              | [*]                                              | Control the access of the Deployment resource |
| statefulsets.apps                             | -                 | -              | [*]                                              | Control the access of the Statefulset resource |
| ingresses.extensions                          | -                 | -              | [*]                                              | Control the access of the Ingress resource for the monitoring system  |
| *.pingcap.com                                 | -                 | -              | [*]                                              | Control the access of all customized resources under pingcap.com |
| configmaps                                    | -                 | -              | [create get list watch update delete]            | Control the access of the ConfigMap resource |
| endpoints                                     | -                 | -              | [create get list watch update delete]            | Control the access of the Endpoints resource |
| serviceaccounts                               | -                 | -              | [create get update delete]                       | Create ServiceAccount for the TidbMonitor/Discovery service|
| clusterrolebindings.rbac.authorization.k8s.io | -                 | -              | [create get update delete]                       | Create ClusterRoleBinding for the TidbMonitor service|
| rolebindings.rbac.authorization.k8s.io        | -                 | -              | [create get update delete]                       | Create RoleBinding for the TidbMonitor/Discovery service |
| secrets                                       | -                 | -              | [create update get list watch delete]            | Control the access of the Secret resource |
| clusterroles.rbac.authorization.k8s.io        | -                 | -              | [escalate create get update delete]              | Create ClusterRole for the TidbMonitor service  |
| roles.rbac.authorization.k8s.io               | -                 | -              | [escalate create get update delete]              | Create Role for the TidbMonitor/Discovery service |
| persistentvolumeclaims                        | -                 | -              | [get list watch create update delete patch]      | Control the access of the PVC resource |
| jobs.batch                                    | -                 | -              | [get list watch create update delete]            | Use jobs to perform TiDB cluster initialization, backup, and restore operations |
| persistentvolumes                             | -                 | -              | [get list watch patch update]                    | Perform operations such as adding labels related to cluster information for PV and modifying `persistentVolumeReclaimPolicy` |
| pods                                          | -                 | -              | [get list watch update delete]                   | Control the access of the Pod resource |
| nodes                                         | -                 | -              | [get list watch]                                 | Read node labels and set store labels for TiKV and TiFlash accordingly |
| storageclasses.storage.k8s.io                 | -                 | -              | [get list watch]                                 | Verify whether StorageClass supports `VolumeExpansion` before expanding PVC storage |
| -                                             |[/metrics]         | -              | [get]                                            | Read monitoring metrics |

> **Note:**
>
> * In the **Non-resource URLs** column, `-` indicates that the item does not have non-resource URLs. 
> * In the **Resource name** column, `-` indicates that the item does not have a resource name.
> * In the **Actions** column, `*` indicates that the resource supports all actions that can be performed on a Kubernetes cluster.

### `tidb-scheduler` ClusterRole permissions

The following table lists the permissions corresponding to the `tidb-scheduler` ClusterRole.

| Resource                   | Non-resource URLs | Resource name     | Action                        | Explanation |
| ---------                  | ----------------- | --------------   | -----                           | ------- |
| leases.coordination.k8s.io | -                 | -                | [create]                        | Create lease resource locks for leader election |
| endpoints                  | -                 | -                | [delete get patch update]       | Control the access of the Endpoints resource |
| persistentvolumeclaims     | -                 | -                | [get list update]               | Read PVC information of PD/TiKV and update the scheduling information to the PVC label |
| configmaps                 | -                 | -                | [get list watch]                | Read the ConfigMap resource |
| pods                       | -                 | -                | [get list watch]                | Read Pod information |
| nodes                      | -                 | -                | [get list]                      | Read node information |
| leases.coordination.k8s.io | -                 | [tidb-scheduler] | [get update]                    | Read and update lease resource locks for leader election |
| tidbclusters.pingcap.com   | -                 | -                | [get]                           | Read Tidbcluster information |

> **Note:**
>
> * In the **Non-resource URLs** column, `-` indicates that the item does not have non-resource URLs. 
> * In the **Resource name** column, `-` indicates that the item does not have a resource name.

## Manage TiDB clusters at the namespace level 

If `clusterScoped=false` is set during the TiDB Operator deployment, TiDB Operator manages TiDB clusters at the Namespace level.

- To check the ClusterRole created for TiDB Operator, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get clusterrole | grep tidb
    ```

    The output is as follows:

    ```shell
    tidb-operator:tidb-controller-manager                                  2021-05-04T13:08:55Z
    ```

    `tidb-operator:tidb-controller-manager` is the ClusterRole created for the `tidb-controller-manager` Pod.

    > **Note:**
    >
    > During the TiDB Operator deployment, if `controllerManager.clusterPermissions.nodes`, `controllerManager.clusterPermissions.persistentvolumes`, `controllerManager.clusterPermissions.storageclasses` are all set to `false`, TiDB operator will not create this ClusterRole.

- To check the roles created for TiDB Operator, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get role -n tidb-admin
    ```

    The example output is as follows:

    ```shell
    tidb-admin    tidb-operator:tidb-controller-manager            2021-05-04T13:08:55Z
    tidb-admin    tidb-operator:tidb-scheduler                     2021-05-04T13:08:55Z
    ```

    In the output:

    * `tidb-operator:tidb-controller-manager` is the role created for the `tidb-controller-manager` Pod.
    * `tidb-operator:tidb-scheduler` is the role created for the `tidb-scheduler` Pod.

### `tidb-controller-manager` ClusterRole permissions

The following table lists the permissions corresponding to the `tidb-controller-manager` ClusterRole.

| Resource                   | Non-resource URLs | Resource name     | Action                        | Explanation |
| ---------                     | ----------------- | -------------- | -----                            | ------- |
| persistentvolumes             | -                 | -   | [get list watch patch update]    | Perform operations such as adding labels related to cluster information for PV and modifying `persistentVolumeReclaimPolicy` |
| nodes                         | -                 | -   | [get list watch]                 | Read node Labels and set store Labels for TiKV and TiFlash accordingly|
| storageclasses.storage.k8s.io | -                 | -   | [get list watch]                 | Verify whether StorageClass supports `VolumeExpansion` before expanding PVC storage |

> **Note:**
>
> * In the **Non-resource URLs** column, `-` indicates that the item does not have non-resource URLs. 
> * In the **Resource name** column, `-` indicates that the item does not have a resource name.

### `tidb-controller-manager` Role permissions

The following table lists the permissions corresponding to the `tidb-controller-manager` Role.

| Resource                                      | Non-resource URLs | Resource name     | Action                        | Explanation |
| ---------                                     | ----------------- | -------------- | -----                                            | ------- |
| events                                        | -                 | -              | [*]                                              | Export event information  |
| services                                      | -                 | -              | [*]                                              | Control the access of the service resources |
| statefulsets.apps.pingcap.com/status          | -                 | -              | [*]                                              | Control the access of the StatefulSet resource when `AdvancedStatefulSet=true`. For more information, see [Advanced StatefulSet Controller](advanced-statefulset.md).  |
| statefulsets.apps.pingcap.com                 | -                 | -              | [*]                                              | Control the access of the StatefulSet resource when `AdvancedStatefulSet=true`. For more information, see [Advanced StatefulSet Controller](advanced-statefulset.md).  |
| controllerrevisions.apps                      | -                 | -              | [*]                                              | Control the version of Kubernetes StatefulSet/Daemonset |
| deployments.apps                              | -                 | -              | [*]                                              | Control the access of the Deployment resource |
| statefulsets.apps                             | -                 | -              | [*]                                              | Control the access of the Statefulset resource |
| ingresses.extensions                          | -                 | -              | [*]                                              | Control the access of the Ingress resource for the monitoring system |
| *.pingcap.com                                 | -                 | -              | [*]                                              | Control the access of all customized resources under pingcap.com |
| configmaps                                    | -                 | -              | [create get list watch update delete]            | Control the access of the ConfigMap resource |
| endpoints                                     | -                 | -              | [create get list watch update delete]            | Control the access of the Endpoints resource |
| serviceaccounts                               | -                 | -              | [create get update delete]                       | Create ServiceAccount for the TidbMonitor/Discovery service |
| rolebindings.rbac.authorization.k8s.io        | -                 | -              | [create get update delete]                       | Create ClusterRoleBinding for the TidbMonitor service |
| secrets                                       | -                 | -              | [create update get list watch delete]            | Control the access of the Secret resource |
| roles.rbac.authorization.k8s.io               | -                 | -              | [escalate create get update delete]              | Create Role for the TidbMonitor/Discovery service |
| persistentvolumeclaims                        | -                 | -              | [get list watch create update delete patch]      | Control the access of the PVC resource |
| jobs.batch                                    | -                 | -              | [get list watch create update delete]            | Use jobs to perform TiDB cluster initialization, backup, and restore operations |
| pods                                          | -                 | -              | [get list watch update delete]                   | Control the access of the Pod resource |

> **Note:**
>
> * In the **Non-resource URLs** column, `-` indicates that the item does not have non-resource URLs. 
> * In the **Resource names** column, `-` indicates that the item does not have a resource name.
> * In the **Actions** column, `*` indicates that the resource supports all actions that can be performed on a Kubernetes cluster.

### `tidb-scheduler` Role permissions

The following table lists the permissions corresponding to the `tidb-scheduler` Role.

| Resource                   | Non-resource URLs | Resource name     | Action                        | Explanation |
| ---------                  | ----------------- | --------------   | -----                           | ------- |
| leases.coordination.k8s.io | -                 | -                | [create]                        | Create lease resource locks for leader election |
| endpoints                  | -                 | -                | [delete get patch update]       | Control the access of the Endpoints resource |
| persistentvolumeclaims     | -                 | -                | [get list update]               | Read PVC information of PD/TiKV and update the scheduling information to the PVC label  |
| configmaps                 | -                 | -                | [get list watch]                | Read the ConfigMap resource |
| pods                       | -                 | -                | [get list watch]                | Read pod information  |
| nodes                      | -                 | -                | [get list]                      | Read node information |
| leases.coordination.k8s.io | -                 | [tidb-scheduler] | [get update]                    | Read and update lease resource locks for leader election |
| tidbclusters.pingcap.com   | -                 | -                | [get]                           | Read Tidbcluster information |

> **Note:**
>
> * In the **Non-resource URLs** column, `-` indicates that the item does not have non-resource URLs. 
> * In the **Resource name** column, `-` indicates that the item does not have a resource name.
