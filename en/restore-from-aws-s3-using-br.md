---
title: Restore Data from S3-Compatible Storage Using BR
summary: Learn how to restore data from Amazon S3 using BR.
category: how-to
aliases: ['/docs/tidb-in-kubernetes/dev/restore-from-aws-s3-using-br/']
---

# Restore Data from S3-Compatible Storage Using BR

This document describes how to restore the TiDB cluster data backed up using TiDB Operator in Kubernetes. [BR](https://pingcap.com/docs/stable/br/backup-and-restore-tool/) is used to perform the restoration.

The restoration method described in this document is implemented based on Custom Resource Definition (CRD) in TiDB Operator v1.1 or later versions.

This document shows an example in which the backup data stored in the specified path on the Amazon S3 storage is restored to the TiDB cluster.

## Three methods to grant AWS account permissions

Refer to [Back up Data to Amazon S3 using BR](backup-to-aws-s3-using-br.md#three-methods-to-grant-aws-account-permissions).

## Prerequisites

Before you restore data from Amazon S3 storage, you need to grant AWS account permissions. This section describes three methods to grant AWS account permissions.

### Grant permissions by importing AccessKey and SecretKey

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `s3-secret` secret which stores the credential used to access the S3-compatible storage:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic s3-secret --from-literal=access_key=xxx --from-literal=secret_key=yyy --namespace=test2
    ```

3. Create the `restore-demo2-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=password=${password} --namespace=test2
    ```

### Grant permissions by associating IAM with Pod

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

     {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `restore-demo2-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=password=${password} --namespace=test2
    ```

3. Create the IAM role:

    - To create an IAM role for the account, refer to [Create an IAM User](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_create.html).
    - Give the required permission to the IAM role you have created  (refer to [access policies manage](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_manage-attach-detach.html) for details). Because `Restore` needs to access the Amazon S3 storage, the IAM here is given the `AmazonS3FullAccess` permission.

4. Associate IAM with TiKV Pod:

    - In the restoration process using BR, both the TiKV Pod and the BR Pod need to perform read and write operations on the S3 storage. Therefore, you need to add the annotation to the TiKV Pod to associate the Pod with the IAM role:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl edit tc demo2 -n test2
        ```

    - Find `spec.tikv.annotations`, append the `arn:aws:iam::123456789012:role/user` annotation, and then exit. After the TiKV Pod is restarted, check whether the annotation is added to the TiKV Pod.

    > **Note:**
    >
    > `arn:aws:iam::123456789012:role/user` is the IAM role created in Step 4.

### Grant permissions by associating IAM with ServiceAccount

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test2` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `restore-demo2-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic restore-demo2-tidb-secret --from-literal=password=${password} --namespace=test2
    ```

3. Enable the IAM role for the service account on the cluster:

    - To enable the IAM role on your EKS cluster, refer to [Amazon EKS Documentation](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html).

4. Create the IAM role:

    - Create an IAM role and give the `AmazonS3FullAccess` permission to the role. Modify `Trust relationships` of the role. For details, refer to [Creating an IAM Role and Policy](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html).

5. Associate IAM with the ServiceAccount resources:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl annotate sa tidb-backup-manager -n eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/user --namespace=test2
    ```

6. Bind ServiceAccount to TiKV Pod:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit tc demo2 -n test2
    ```

    Modify the value of `spec.tikv.serviceAccount` to `tidb-backup-manager`. After the TiKV Pod is restarted, check whether the `serviceAccountName` of the TiKV Pod has changed.

    > **Note:**
    >
    > `arn:aws:iam::123456789012:role/user` is the IAM role created in Step 4.

## Restoration process

+ If you grant permissions by importing AccessKey and SecretKey, create the `Restore` CR, and restore cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f resotre-aws-s3.yaml
    ```

    The content of `restore-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
    spec:
      br:
        cluster: demo2
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # sendCredToTikv: true
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        secretName: s3-secret
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

+ If you grant permissions by associating IAM with Pod, create the `Restore` CR, and restore cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore-aws-s3.yaml
    ```

    The content of `restore-aws-s3.yaml` is as follows:

     ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      br:
        cluster: demo2
        sendCredToTikv: false
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

+ If you grant permissions by associating IAM with ServiceAccount, create the `Restore` CR, and restore cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f restore-aws-s3.yaml
    ```

    The content of `restore-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Restore
    metadata:
      name: demo2-restore-s3
      namespace: test2
    spec:
      serviceAccount: tidb-backup-manager
      br:
        cluster: demo2
        sendCredToTikv: false
        clusterNamespace: test2
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      to:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: restore-demo2-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

After creating the `Restore` CR, execute the following command to check the restoration status:

{{< copyable "shell-regular" >}}

```shell
kubectl get rt -n test2 -o wide
```

More `Restore` CR fields are described as follows:

* `.spec.metadata.namespace`: the namespace where the `Restore` CR is located.
* `.spec.to.host`: the address of the TiDB cluster to be restored.
* `.spec.to.port`: the port of the TiDB cluster to be restored.
* `.spec.to.user`: the accessing user of the TiDB cluster to be restored.
* `.spec.to.tidbSecretName`: the secret of the user password of the `.spec.to.tidbSecretName` TiDB cluster.
* `.spec.to.tlsClient.tlsSecret`: the secret of the certificate used during the restoration.

    If [TLS](enable-tls-between-components.md) is enabled for the TiDB cluster, but you do not want to restore data using the `${cluster_name}-cluster-client-secret` as described in [Enable TLS between TiDB Components](enable-tls-between-components.md), you can use the `.spec.from.tlsClient.tlsSecret` parameter to specify a secret for the restoration. To generate the secret, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```
