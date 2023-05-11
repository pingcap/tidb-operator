<h2>EBS Backup Restore Federation</h2>


<h2>Background</h2>


Some TiDB operator users need to deploy TiDB across multiple Kubernetes clusters. Each Kubernetes cluster will be deployed in a single AZ, and each Kubernetes can only manage the AWS resources in that AZ. However, EBS backup and recovery requires the operation of volumes and snapshots in different AZs. Therefore, a federated solution needs to be provided to users to coordinate backup/restore of volumes in their respective AZs across multiple Kubernetes clusters.

The key constraint is that operations involving AWS and Kubernetes resources can only be done within each Kubernetes, but TiDB components are interconnected across each Kubernetes, so TiDB-related operations are not limited by Kubernetes. Therefore, we need to categorize the EBS backup and restore process, distinguishing which processes can be done in only one Kubernetes, and which processes need to be done in each Kubernetes.

<h3>Current Implementation</h3>


In the diagrams below, the red lines represent AWS and Kubernetes related operations that need to be executed separately in each Kubernetes cluster, while the black lines represent TiDB related operations that can be executed in any Kubernetes cluster.

<h4>Backup</h4>

![current backup process .png](current backup process.png)

<h4>Restore</h4>

![current restore process.png](current restore process.png)


<h2>Scope and prerequisites</h2>


Goal for 6/30:

Support our current 3 az/k8s cluster setup with a TiDB cluster in each az/k8s cluster. Such that we restore into the same AZs in the same region.

Scope and assumptions:

1. EBS backup and restore support in the federated cluster includes all functions of those in the single Kubernetes cluster, including creating backups, deleting backups, automatic backup scheduling using backup schedules, automatic backup cleanup based on retention policies, and cluster restore functions.
2. Only multiple Kubernetes clusters deployed in different AZs within a single region are supported for federation, and cross-region federation is not supported.
3. Only EBS snapshot backup and restore is supported, and other types of backup and restore or TiDB-Operator's other features are not supported.
4. The federated cluster architecture for backup and restore must be consistent, that is, the number of AZs, Kubernetes clusters, and TiKV replicas for each Kubernetes cluster must be the same, but the number of TiDB/PD instances can be different.
5. EBS snapshot restore only supports restoring to volumes in the same region, with the same volume type.

Longer term(beyond 2023):

1. Cross-region federation of Kubernetes clusters deployed in different AZs across multiple AWS regions will be supported;
1. The federated cluster architecture for backup and restore must be consistent, that is, the number of AZs, Kubernetes clusters, and TiKV replicas for each Kubernetes cluster must be the same, but the number of TiDB/PD instances can be different. **_The number of TiKV nodes shall be the same between the backuped and restored clusters, however, the TiKV nodes in the restored cluster might have less CPU and memory._**
2. EBS snapshot restore will support restoring to volumes in a different region with different volume types.
3. EBS snapshot backup and restore will support backup and restoring from and to different k8s clusters with each k8s cluster under different AWS accounts._

<h2>Architecture Design</h2>

![ebs br multiple k8s support architecture.png](ebs br multiple k8s architecture.png)

<h2>Detailed Design</h2>


<h3>Volume-BR-Federation-Operator</h3>


The user needs to install Volume-BR-Federation-Operator into the control plane Kubernetes cluster, and ensure that Volume-BR-Federation-Operator can connect to the api-servers of each data plane Kubernetes cluster on the network.

The federal operator should cooperate tidb-operators in the data plane by Kubernetes API server. We only need to **get/create/update/delete** permissions on Backup/Restore CR to the users of the federation operator.

<h3>KubeConfig</h3>


1. The user configures the KubeConfig of all data plane Kubernetes clusters into one or multiple secrets in the control plane.
2. The user creates a KubeClientConfig CR, declaring that secret and the context of kubeConfig, then adds the Kubernetes cluster to the federation.

The user configures the KubeConfig of all data plane Kubernetes clusters into one secret in the control plane and mount the secret to the BR federation operator._

```json
apiVersion: pingcap.com/v1alpha1
kind: KubeClientConfig
metadata:
  name: conf-1
  namespace: default
spec:
  contextName: ctx-1
  secretRef:
    name: ctx-secret1
```

<h3>Backup</h3>


In the federal Operator, a new CRD, VolumeBackupFederation, will be created. The user needs to specify the kubeConfig and TiDBCluster information for each Kubernetes cluster so that the control plane can control the backup of TiKV volumes for each Kubernetes cluster.

```json
apiVersion: pingcap.com/v1alpha1
kind: VolumeBackupFederation
metadata:
  name: bk-federation
  namespace: default
spec:
  clusterList:
  - kubeConfig: conf-1
    cluster: tc-1
    clusterNamespace: ns1
  - kubeConfig: conf-2
    cluster: tc-2
    clusterNamespace: ns2
  - kubeConfig: conf-3
    cluster: tc-3
    clusterNamespace: ns3
  template:
    s3:
      bucket: bucket1
      region: region1
      prefix: prefix1
      secretName: secret1
    toolImage: br-image
    serviceAccount: sa1
    cleanPolicy: Delete
```
<h4>Backup Process</h4>


During EBS backup, as EBS snapshot is an asynchronous and time-consuming operation, to ensure data safety, we need to first stop PD scheduling and maintain GC safe point less than resolved ts before taking the EBS snapshot. In a multi-Kubernetes cluster scenario, as EBS snapshot needs to be taken in each Kubernetes cluster, this operation becomes a distributed operation. Therefore, we need to ensure that PD scheduling and GC are stopped before taking EBS snapshot in each Kubernetes cluster, and PD scheduling and GC are resumed only after EBS snapshot is taken in all Kubernetes clusters. Therefore, we divide the EBS backup process into three phases.

![new backup process.png](new backup process in data plane.png)

**First phase:**



1. The backup controller of the control plane randomly selects a data plane Kubernetes cluster and creates a backup CR with “federalVolumeBackupPhase: initialize”.
2. The backup controller of the corresponding data plane Kubernetes cluster first creates backup pod1 to obtain the resolved ts and maintain the GC safepoint not exceeding the resolved ts and stop PD scheduling.
3. After stopping GC and PD scheduling, backup CR enters the VolumeBackupInitialized state.

**Second phase:**



4. The backup controller of the control plane discovers that backup CR is in VolumeBackupInitialized state, then sets federalVolumeBackupPhase of backup CR to “execute” and creates backup CRs for other data plane Kubernetes clusters with “federalVolumeBackupPhase: execute”.
5. The backup controller of the data plane collects information about TC and records it in the clustermeta file, creates backup pod2, and executes EBS backup.
6. After the EBS backup is completed, backup CRs enter the VolumeBackupComplete state.

**Third phase:**



7. The backup controller of the control plane discovers that all backup CRs are in VolumeBackupComplete state. At this time, all data planes have completed effective backups (if the condition is not met, the backup enters the failed state).
8. The backup controller of the control plane sets federalVolumeBackupPhase of the backup CR in step1 to “teardown”, the backup controller of the corresponding data plane Kubernetes cluster deletes backup pod1 is deleted to stop maintaining the GC safepoint and resume PD scheduling. The EBS backup is completed.

![new backup process in control plane.png](new backup process in control plane.png)

<h4>Delete Backup</h4>


When the VolumeBackupFederation CR is deleted, the backup controller in the control plane will delete all corresponding Backup CRs in the data plane. After all Backups have been deleted, VolumeBackupFederation will be deleted completely.

<h3>BackupSchedule</h3>


In the Federal Operator, a new CRD, VolumeBackupScheduleFederation, will be created to support automatic scheduling of backups and regular cleanup of backups based on maxReservedTime. Its implementation is similar to BackupSchedule and will automatically create and delete VolumeBackupFederation CRs based on the configuration.

```json
apiVersion: pingcap.com/v1alpha1
kind: VolumeBackupScheduleFederation
metadata:
  name: bks-federation
  namespace: default
spec:
  schedule: 0 0 1 * * ?
  pause: false
  maxReservedTime: 7d
  backupTemplate: {{spec of VolumeBackupFederation}}
```

<h3>Restore</h3>


In the Federal Operator, a new CRD, VolumeRestoreFederation, will be created. The user needs to specify the kubeConfig, tc, and backup storage path information for each Kubernetes cluster, and it is necessary to ensure that Kubernetes and backups are in the same AZ.

```json
apiVersion: pingcap.com/v1alpha1
kind: VolumeRestoreFederation
metadata:
  name: rt-federation
  namespace: default
spec:
  clusterList:
  - kubernetes: ctx-1
    cluster: tc-1
    clusterNamespace: ns1
    azName: az4
    backup:
      path: prefix/tc-1-bk-federation-az1
  - kubernetes: ctx-2
    cluster: tc-2
    clusterNamespace: ns2
    backup:
      path: prefix/tc-2-bk-federation
  - kubernetes: ctx-3
    cluster: tc-3
    clusterNamespace: ns3
    backup:
      path: prefix/tc-3-bk-federation
  template:
    s3:
      bucket: bucket1
      region: region1
      secretName: secret1
    toolImage: br-image
    serviceAccount: sa1
```

<h4>Restore Process</h4>


The restore process was originally divided into two phases, where the first phase involved restoring the TiKV disks and the second phase involved restoring TiKV data to the resolved ts state. The first phase needs to be executed in each Kubernetes cluster, while the second phase only needs to be executed in a single Kubernetes cluster. However, there is a follow-up task after the second phase, which involves restarting TiKV pods and setting TC recoveryMode to false. This task needs to be executed in each Kubernetes cluster, so we divide the restore process into three phases.

![new prpocess of restore in data plane.png](new process of restore in data plane.png)

**Before restoration:**


1. Users create TiDBCluster CR in each data plane Kubernetes cluster and set the recoveryMode of all TiDBCluster CR to true, and also ensure that the TiKV replicas in each AZ are consistent with the backup.

**First phase:**



2. The restore controller of the control plane creates restore CR in each Kubernetes cluster with “federalVolumeRestorePhase: restore-volume”.
3. The restore controller of the data plane restores the disks from EBS snapshots, correctly assigns each disk to the volume of each TiKV, and starts TiKV.
4. When the restore controller of the data discovers that all TiKV nodes in the current Kubernetes cluster have been successfully started, the restore CR1 enters the TikvComplete status.

**Second phase:**



5. The restore controller of the control plane randomly selects a Kubernetes cluster to set federalVolumeRestorePhase of restore CR to “restore-data” after all restore CRs have entered TikvComplete status (if any restore CR fails, the restore fails).
6. The restore controller of the data plane creates a restore pod, and performs the data restore stage of EBS restore. When BR completes execution, the restore CR enters DataComplete status.

**Third phase:**



7. The restore controller of the control plane updates the federalVolumeRestorePhase of all restore CRs to “restore-finish” after the restore CR in step5 enters the DataComplete status.
8. The restore controller of the data plane restarts the TiKV pods in the Kubernetes cluster where it is located, sets TC recoveryMode to false. Then the restore CR enters complete status.
9. The restore is completed when the restore controller of the control plane discovers that all restore CRs have entered complete status.

![new process of restore in control plane.png](new process of restore in control plane.png)

