---
title: Back up Data to S3-Compatible Storage Using BR
summary: Learn how to back up data to Amazon S3 using BR.
aliases: ['/docs/tidb-in-kubernetes/dev/backup-to-aws-s3-using-br/']
---

<!-- markdownlint-disable MD007 -->

# Back up Data to S3-Compatible Storage Using BR

This document describes how to back up the data of a TiDB cluster in AWS Kubernetes to the AWS storage using Helm charts. "Backup" in this document refers to full backup (ad-hoc full backup and scheduled full backup). [BR](https://docs.pingcap.com/tidb/stable/backup-and-restore-tool) is used to get the logic backup of the TiDB cluster, and then this backup data is sent to the AWS storage.

The backup method described in this document is implemented using Custom Resource Definition (CRD) in TiDB Operator v1.1 or later versions.

## Three methods to grant AWS account permissions

In the AWS cloud environment, different types of Kubernetes clusters provide different methods to grant AWS account permissions. This document describes the following three methods:

+ Import the AccessKey and SecretKey of the AWS account:

    - The AWS client supports reading `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in the process environment variables to get the permissions of the associated user or role.

+ Associate [IAM](https://aws.amazon.com/cn/iam/) with the Pod:

    - By associating the IAM role of the user with the running Pod resources, the process that runs in a Pod gets the permissions owned by the role.
    - This authorization method is provided by [`kube2iam`](https://github.com/jtblin/kube2iam).

    > **Note:**
    >
    > - When you use this method, refer to [`kube2iam` Usage](https://github.com/jtblin/kube2iam#usage) for instructions on how to create the `kube2iam` environment in the Kubernetes cluster, and then deploy TiDB Operator and the TiDB cluster.
    > - This method does not apply to [`hostNetwork`](https://kubernetes.io/docs/concepts/policy/pod-security-policy). Make sure that the `spec.tikv.hostNetwork` parameter is set to `false`.

+ Associate [IAM](https://aws.amazon.com/cn/iam/) with ServiceAccount:

    - By Associating the IAM role of the user with the [`serviceAccount`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#serviceaccount) resources in Kubernetes, the Pods of this ServiceAccount get the permissions owned by the role.
    - This method is provided by [`EKS Pod Identity Webhook`](https://github.com/aws/amazon-eks-pod-identity-webhook).

    > **Note:**
    >
    > When you use this method, refer to [AWS Documentation](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html) for instructions on how to create an EKS cluster, and then deploy TiDB Operator and the TiDB cluster.

## Ad-hoc full backup

Ad-hoc full backup describes the backup by creating a `Backup` Custom Resource (CR) object. TiDB Operator performs the specific backup operation based on this `Backup` object. If an error occurs during the backup process, TiDB Operator does not retry, and you need to handle this error manually.

Currently, the above three authorization methods are supported for the ad-hoc full backup. This document provides examples in which the data of the `demo1` TiDB cluster in the `test1` Kubernetes namespace is backed up to AWS storage and all the above methods are used in the examples.

### Prerequisites for ad-hoc full backup

Before you perform ad-hoc full backup, AWS account permissions need to be granted. This section describes three methods to grant AWS account permissions.

#### Grant permissions by importing AccessKey and SecretKey

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test1` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. Create the `s3-secret` secret which stores the credential used to access the S3-compatible storage:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic s3-secret --from-literal=access_key=xxx --from-literal=secret_key=yyy --namespace=test1
    ```

3. Create the `backup-demo1-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=${password} --namespace=test1
    ```

#### Grant permissions by associating IAM with Pod

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test1` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test1
    ```

2. Create the `backup-demo1-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=${password} --namespace=test1
    ```

3. Create the IAM role:

    - To create an IAM role for the account, refer to [Create an IAM User](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_users_create.html).
    - Give the required permission to the IAM role you have created. Refer to [Adding and Removing IAM Identity Permissions](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_manage-attach-detach.html) for details. Because `Backup` needs to access the Amazon S3 storage, IAM is granted the `AmazonS3FullAccess` permission.

4. Associate IAM with TiKV Pod:

    - In the backup process using BR, both the TiKV Pod and the BR Pod need to perform read and write operations on the S3 storage. Therefore, you need to add the annotation to the TiKV Pod to associate the Pod with the IAM role:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl edit tc demo1 -n test1
        ```

    - Find `spec.tikv.annotations`, append the `iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user` annotation, and then exit. After the TiKV Pod is restarted, check whether the annotation is added to the TiKV Pod.

    > **Note:**
    >
    > `arn:aws:iam::123456789012:role/user` is the IAM role created in Step 4.

#### Grant permissions by associating IAM with ServiceAccount

1. Download [backup-rbac.yaml](https://github.com/pingcap/tidb-operator/blob/master/manifests/backup/backup-rbac.yaml), and execute the following command to create the role-based access control (RBAC) resources in the `test1` namespace:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-rbac.yaml -n test2
    ```

2. Create the `backup-demo1-tidb-secret` secret which stores the account and password needed to access the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic backup-demo1-tidb-secret --from-literal=password=${password} --namespace=test1
    ```

3. Enable the IAM role for the service account on the cluster:

    - To enable the IAM role on your EKS cluster, refer to [Amazon EKS Documentation](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html).

4. Create the IAM role:

    - Create an IAM role and give the `AmazonS3FullAccess` permission to the role. Modify `Trust relationships` of the role. For details, refer to [Creating an IAM Role and Policy](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html).

5. Associate IAM with the ServiceAccount resources:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl annotate sa tidb-backup-manager -n eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/user --namespace=test1
    ```

6. Bind ServiceAccount to TiKV Pod:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl edit tc demo1 -n test1
    ```

    Modify the value of `spec.tikv.serviceAccount` to `tidb-backup-manager`. After the TiKV Pod is restarted, check whether the `serviceAccountName` of the TiKV Pod has changed.

    > **Note:**
    >
    > `arn:aws:iam::123456789012:role/user` is the IAM role created in Step 4.

### Process of ad-hoc full backup

- If you grant permissions by importing AccessKey and SecretKey, create the `Backup` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-aws-s3.yaml
    ```

    The content of `backup-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: test1
    spec:
      backupType: full
      br:
        cluster: demo1
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
        # sendCredToTikv: true
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        secretName: s3-secret
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

- If you grant permissions by associating IAM with Pod, create the `Backup` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-aws-s3.yaml
    ```

    The content of `backup-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: test1
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      backupType: full
      br:
        cluster: demo1
        sendCredToTikv: false
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

- If you grant permissions by associating IAM with ServiceAccount, create the `Backup` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-aws-s3.yaml
    ```

    The content of `backup-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: Backup
    metadata:
      name: demo1-backup-s3
      namespace: test1
    spec:
      backupType: full
      serviceAccount: tidb-backup-manager
      br:
        cluster: demo1
        sendCredToTikv: false
        clusterNamespace: test1
        # logLevel: info
        # statusAddr: ${status_addr}
        # concurrency: 4
        # rateLimit: 0
        # timeAgo: ${time}
        # checksum: true
      from:
        host: ${tidb_host}
        port: ${tidb_port}
        user: ${tidb_user}
        secretName: backup-demo1-tidb-secret
      s3:
        provider: aws
        region: us-west-1
        bucket: my-bucket
        prefix: my-folder
    ```

The above three examples uses three methods to grant permissions to back up data to Amazon S3 storage. The `acl`, `endpoint`, `storageClass` configuration items of Amazon S3 can be ignored.

<details>
<summary>Configure the access-control list (ACL) policy</summary>

Amazon S3 supports the following ACL policies:

- `private`
- `public-read`
- `public-read-write`
- `authenticated-read`
- `bucket-owner-read`
- `bucket-owner-full-control`

If the ACL policy is not configured, the `private` policy is used by default. For the detailed description of these access control policies, refer to [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html).

</details>

<details>
<summary>Configure <code>storageClass</code></summary>

Amazon S3 supports the following `storageClass` types:

- `STANDARD`
- `REDUCED_REDUNDANCY`
- `STANDARD_IA`
- `ONEZONE_IA`
- `GLACIER`
- `DEEP_ARCHIVE`

If `storageClass` is not configured, `STANDARD_IA` is used by default. For the detailed description of these storage types, refer to [AWS documentation](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html).

</details>

After creating the `Backup` CR, use the following command to check the backup status:

{{< copyable "shell-regular" >}}

 ```shell
 kubectl get bk -n test1 -o wide
 ```

<details>
<summary>More <code>Backup</code> CR parameter description</summary>

- `.spec.metadata.namespace`: the namespace where the `Backup` CR is located.
- `.spec.tikvGCLifeTime`: the temporary `tikv_gc_lifetime` time setting during the backup. Defaults to 72h.

    Before the backup begins, if the `tikv_gc_lifetime` setting in the TiDB cluster is smaller than `spec.tikvGCLifeTime` set by the user, TiDB Operator adjusts the value of `tikv_gc_lifetime` to the value of `spec.tikvGCLifeTime`. This operation makes sure that the backup data is not garbage-collected by TiKV.

    After the backup, no matter whether the backup is successful or not, as long as the previous `tikv_gc_lifetime` is smaller than `.spec.tikvGCLifeTime`, TiDB Operator will try to set `tikv_gc_lifetime` to the previous value.

    In extreme cases, if TiDB Operator fails to access the database, TiDB Operator cannot automatically recover the value of `tikv_gc_lifetime` and treats the backup as failed. At this time, you can view `tikv_gc_lifetime` of the current TiDB cluster using the following statement:

    {{< copyable "sql" >}}

    ```sql
    select VARIABLE_NAME, VARIABLE_VALUE from mysql.tidb where VARIABLE_NAME like "tikv_gc_life_time";
    ```

    In the output of the command above, if the value of `tikv_gc_lifetime` is still larger than expected (10m by default), it means TiDB Operator failed to automatically recover the value. Therefore, you need to set `tikv_gc_lifetime` back to the previous value manually:

    {{< copyable "sql" >}}

    ```sql
    update mysql.tidb set VARIABLE_VALUE = '10m' where VARIABLE_NAME = 'tikv_gc_life_time';

- * `.spec.cleanPolicy`: The clean policy of the backup data when the backup CR is deleted.

    Three clean policies are supported:

    * `Retain`: On any circumstances, retain the backup data when deleting the backup CR.
    * `Delete`: On any circumstances, delete the backup data when deleting the backup CR.
    * `OnFailure`: If the backup fails, delete the backup data when deleting the backup CR.

    If this field is not configured, or if you configure a value other than the three policies above, the backup data is retained.

    Note that in v1.1.2 and earlier versions, this field does not exist. The backup data is deleted along with the CR by default. For v1.1.3 or later versions, if you want to keep this behavior, set this field to `Delete`.

- `.spec.from.host`: the address of the TiDB cluster to be backed up, which is the service name of the TiDB cluster to be exported, such as `basic-tidb`.
- `.spec.from.port`: the port of the TiDB cluster to be backed up.
- `.spec.from.user`: the accessing user of the TiDB cluster to be backed up.
- `.spec.from.tidbSecretName`: the secret of the user password of the `.spec.from.user` TiDB cluster.
- `.spec.from.tlsClientSecretName`: the secret of the certificate used during the backup.

    If [TLS](enable-tls-between-components.md) is enabled for the TiDB cluster, but you do not want to back up data using the `${cluster_name}-cluster-client-secret` as described in [Enable TLS between TiDB Components](enable-tls-between-components.md), you can use the `.spec.from.tlsClient.tlsSecret` parameter to specify a secret for the backup. To generate the secret, run the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=${cert_path} --from-file=tls.key=${key_path} --from-file=ca.crt=${ca_path}
    ```

</details>

<details>
<summary>Supported S3-compatible <code>provider</code></summary>

- `alibaba`：Alibaba Cloud Object Storage System (OSS) formerly Aliyun
- `digitalocean`：Digital Ocean Spaces
- `dreamhost`：Dreamhost DreamObjects
- `ibmcos`：IBM COS S3
- `minio`：Minio Object Storage
- `netease`：Netease Object Storage (NOS)
- `wasabi`：Wasabi Object Storage
- `other`：Any other S3 compatible provider

</details>

## Scheduled full backup

You can set a backup policy to perform scheduled backups of the TiDB cluster, and set a backup retention policy to avoid excessive backup items. A scheduled full backup is described by a custom `BackupSchedule` CR object. A full backup is triggered at each backup time point. Its underlying implementation is the ad-hoc full backup.

### Prerequisites for scheduled full backup

The prerequisites for the scheduled full backup is the same as the [prerequisites for ad-hoc full backup](#prerequisites-for-ad-hoc-full-backup).

### Process of scheduled full backup

+ If you grant permissions by importing AccessKey and SecretKey, create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-scheduler-aws-s3.yaml
    ```

    The content of `backup-scheduler-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: test1
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        backupType: full
        br:
          cluster: demo1
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status_addr}
          # concurrency: 4
          # rateLimit: 0
          # timeAgo: ${time}
          # checksum: true
          # sendCredToTikv: true
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          secretName: s3-secret
          region: us-west-1
          bucket: my-bucket
          prefix: my-folder
    ```

+ If you grant permissions by associating IAM with the Pod, create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-scheduler-aws-s3.yaml
    ```

    The content of `backup-scheduler-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: test1
      annotations:
        iam.amazonaws.com/role: arn:aws:iam::123456789012:role/user
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      backupTemplate:
        backupType: full
        br:
          cluster: demo1
          sendCredToTikv: false
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status_addr}
          # concurrency: 4
          # rateLimit: 0
          # timeAgo: ${time}
          # checksum: true
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          region: us-west-1
          bucket: my-bucket
          prefix: my-folder
    ```

+ If you grant permissions by associating IAM with ServiceAccount, create the `BackupSchedule` CR, and back up cluster data as described below:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f backup-scheduler-aws-s3.yaml
    ```

    The content of `backup-scheduler-aws-s3.yaml` is as follows:

    ```yaml
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: BackupSchedule
    metadata:
      name: demo1-backup-schedule-s3
      namespace: test1
    spec:
      #maxBackups: 5
      #pause: true
      maxReservedTime: "3h"
      schedule: "*/2 * * * *"
      serviceAccount: tidb-backup-manager
      backupTemplate:
        backupType: full
        br:
          cluster: demo1
          sendCredToTikv: false
          clusterNamespace: test1
          # logLevel: info
          # statusAddr: ${status_addr}
          # concurrency: 4
          # rateLimit: 0
          # timeAgo: ${time}
          # checksum: true
        from:
          host: ${tidb_host}
          port: ${tidb_port}
          user: ${tidb_user}
          secretName: backup-demo1-tidb-secret
        s3:
          provider: aws
          region: us-west-1
          bucket: my-bucket
          prefix: my-folder
    ```

After creating the scheduled full backup, use the following command to check the backup status:

{{< copyable "shell-regular" >}}

```shell
kubectl get bks -n test1 -o wide
```

You can use the following command to check all the backup items:

{{< copyable "shell-regular" >}}

```shell
kubectl get bk -l tidb.pingcap.com/backup-schedule=demo1-backup-schedule-s3 -n test1
```

From the above two examples, you can see that the `backupSchedule` configuration consists of two parts. One is the unique configuration of `backupSchedule`, and the other is `backupTemplate`.

`backupTemplate` specifies the configuration related to the S3 storage, which is the same as the configuration of the ad-hoc full backup to S3 (refer to [S3 backup process](#process-of-ad-hoc-full-backup) for details). The following are the unique configuration items of `backupSchedule`:

- `.spec.maxBackups`: A backup retention policy, which determines the maximum number of backup items to be retained. When this value is exceeded, the outdated backup items will be deleted. If you set this configuration item to `0`, all backup items are retained.

- `.spec.maxReservedTime`: A backup retention policy based on time. For example, if you set the value of this configuration to `24h`, only backup items within the recent 24 hours are retained. All backup items out of this time are deleted. For the time format, refer to [`func ParseDuration`](https://golang.org/pkg/time/#ParseDuration). If you have set the maximum number of backup items and the longest retention time of backup items at the same time, the latter setting takes effect.

- `.spec.schedule`: The time scheduling format of Cron. Refer to [Cron](https://en.wikipedia.org/wiki/Cron) for details.

- `.spec.pause`: `false` by default. If this parameter is set to `true`, the scheduled scheduling is paused. In this situation, the backup operation will not be performed even if the scheduling time is reached. During this pause, the backup [Garbage Collection](https://pingcap.com/docs/stable/reference/garbage-collection/overview) (GC) runs normally. If you change `true` to `false`, the full backup process is restarted.

## Delete the backup CR

You can delete the full backup CR (`Backup`) and the scheduled backup CR (`BackupSchedule`) by the following commands:

{{< copyable "shell-regular" >}}

```shell
kubectl delete backup ${name} -n ${namespace}
kubectl delete backupschedule ${name} -n ${namespace}
```

If you use TiDB Operator v1.1.2 or earlier versions, or if you use TiDB Operator v1.1.3 or later versions and set the value of `spec.cleanPolicy` to `Delete`, TiDB Operator deletes the backup data when it deletes the CR. In such cases, if you need to delete the namespace as well, it is recommended that you first delete all the `Backup`/`BackupSchedule` CR and then delete the namespace.

If you delete the namespace before you delete the `Backup`/`BackupSchedule` CR, TiDB Operator continues to create jobs to clean the backup data. However, since the namespace is in `Terminating` state, TiDB Operator fails to create a job, which causes the namespace to be stuck in the state.

To address this issue, delete `finalizers` using the following command:

{{< copyable "shell-regular" >}}

```shell
kubectl edit backup ${name} -n ${namespace}
```

After deleting the `metadata.finalizers` configuration, you can delete CR normally.

## Troubleshooting

If you encounter any problem during the backup process, refer to [Common Deployment Failures](deploy-failures.md).
