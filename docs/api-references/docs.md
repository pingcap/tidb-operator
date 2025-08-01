---
title: TiDB Operator API Document
summary: Reference of TiDB Operator API
category: how-to
---
<h1>API Document</h1>
<h2 id="pingcap.com/v1alpha1">pingcap.com/v1alpha1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#backup">Backup</a>
</li><li>
<a href="#backupschedule">BackupSchedule</a>
</li><li>
<a href="#dmcluster">DMCluster</a>
</li><li>
<a href="#restore">Restore</a>
</li><li>
<a href="#tidbcluster">TidbCluster</a>
</li><li>
<a href="#tidbinitializer">TidbInitializer</a>
</li><li>
<a href="#tidbmonitor">TidbMonitor</a>
</li></ul>
<h3 id="backup">Backup</h3>
<p>
<p>Backup is a backup of tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Backup</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#backupspec">
BackupSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following builtin env vars will be overwritten by values set here
- S3_PROVIDER
- S3_ENDPOINT
- AWS_REGION
- AWS_ACL
- AWS_STORAGE_CLASS
- AWS_DEFAULT_REGION
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCS_PROJECT_ID
- GCS_OBJECT_ACL
- GCS_BUCKET_ACL
- GCS_LOCATION
- GCS_STORAGE_CLASS
- GCS_SERVICE_ACCOUNT_JSON_KEY
- BR_LOG_TO_TERM</p>
</td>
</tr>
<tr>
<td>
<code>from</code></br>
<em>
<a href="#tidbaccessconfig">
TiDBAccessConfig
</a>
</em>
</td>
<td>
<p>From is the tidb cluster that needs to backup.</p>
</td>
</tr>
<tr>
<td>
<code>backupType</code></br>
<em>
<a href="#backuptype">
BackupType
</a>
</em>
</td>
<td>
<p>Type is the backup type for tidb cluster and only used when Mode = snapshot, such as full, db, table.</p>
</td>
</tr>
<tr>
<td>
<code>backupMode</code></br>
<em>
<a href="#backupmode">
BackupMode
</a>
</em>
</td>
<td>
<p>Mode is the backup mode, such as snapshot backup or log backup.</p>
</td>
</tr>
<tr>
<td>
<code>tikvGCLifeTime</code></br>
<em>
string
</em>
</td>
<td>
<p>TikvGCLifeTime is to specify the safe gc life time for backup.
The time limit during which data is retained for each GC, in the format of Go Duration.
When a GC happens, the current time minus this value is the safe point.</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<p>StorageProvider configures where and how backups should be stored.
*** Note: This field should generally not be left empty, unless you are certain the storage provider
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Backup data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageSize is the request storage size for backup job</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<p>BRConfig is the configs for BR
*** Note: This field should generally not be left empty, unless you are certain the BR config
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>commitTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.
Default is current timestamp.</p>
</td>
</tr>
<tr>
<td>
<code>logSubcommand</code></br>
<em>
<a href="#logsubcommandtype">
LogSubCommandType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subcommand is the subcommand for BR, such as start, stop, pause etc.</p>
</td>
</tr>
<tr>
<td>
<code>logTruncateUntil</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogTruncateUntil is log backup truncate until timestamp.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>logStop</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogStop indicates that will stop the log backup.</p>
</td>
</tr>
<tr>
<td>
<code>calcSizeLevel</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CalcSizeLevel determines how to size calculation of snapshots for EBS volume snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>federalVolumeBackupPhase</code></br>
<em>
<a href="#federalvolumebackupphase">
FederalVolumeBackupPhase
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FederalVolumeBackupPhase indicates which phase to execute in federal volume backup</p>
</td>
</tr>
<tr>
<td>
<code>resumeGcSchedule</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResumeGcSchedule indicates whether resume gc and pd scheduler for EBS volume snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>dumpling</code></br>
<em>
<a href="#dumplingconfig">
DumplingConfig
</a>
</em>
</td>
<td>
<p>DumplingConfig is the configs for dumpling</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of backup Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>toolImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ToolImage specifies the tool image used in <code>Backup</code>, which supports BR and Dumpling images.
For examples <code>spec.toolImage: pingcap/br:v4.0.8</code> or <code>spec.toolImage: pingcap/dumpling:v4.0.8</code>
For BR image, if it does not contain tag, Pod will use image &lsquo;ToolImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>tableFilter</code></br>
<em>
[]string
</em>
</td>
<td>
<p>TableFilter means Table filter expression for &lsquo;db.table&rsquo; matching. BR supports this from v4.0.3.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of backup Pods</p>
</td>
</tr>
<tr>
<td>
<code>useKMS</code></br>
<em>
bool
</em>
</td>
<td>
<p>Use KMS to decrypt the secrets</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify service account of backup</p>
</td>
</tr>
<tr>
<td>
<code>cleanPolicy</code></br>
<em>
<a href="#cleanpolicytype">
CleanPolicyType
</a>
</em>
</td>
<td>
<p>CleanPolicy denotes whether to clean backup data when the object is deleted from the cluster, if not set, the backup data will be retained</p>
</td>
</tr>
<tr>
<td>
<code>cleanOption</code></br>
<em>
<a href="#cleanoption">
CleanOption
</a>
</em>
</td>
<td>
<p>CleanOption controls the behavior of clean.</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName of Backup Job Pods</p>
</td>
</tr>
<tr>
<td>
<code>backoffRetryPolicy</code></br>
<em>
<a href="#backoffretrypolicy">
BackoffRetryPolicy
</a>
</em>
</td>
<td>
<p>BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>volumeBackupInitJobMaxActiveSeconds</code></br>
<em>
int
</em>
</td>
<td>
<p>VolumeBackupInitJobMaxActiveSeconds represents the deadline (in seconds) of the vbk init job</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#backupstatus">
BackupStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="backupschedule">BackupSchedule</h3>
<p>
<p>BackupSchedule is a backup schedule of tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>BackupSchedule</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#backupschedulespec">
BackupScheduleSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>schedule</code></br>
<em>
string
</em>
</td>
<td>
<p>Schedule specifies the cron string used for backup scheduling.</p>
</td>
</tr>
<tr>
<td>
<code>pause</code></br>
<em>
bool
</em>
</td>
<td>
<p>Pause means paused backupSchedule</p>
</td>
</tr>
<tr>
<td>
<code>maxBackups</code></br>
<em>
int32
</em>
</td>
<td>
<p>MaxBackups is to specify how many backups we want to keep
0 is magic number to indicate un-limited backups.
if MaxBackups and MaxReservedTime are set at the same time, MaxReservedTime is preferred
and MaxBackups is ignored.</p>
</td>
</tr>
<tr>
<td>
<code>maxReservedTime</code></br>
<em>
string
</em>
</td>
<td>
<p>MaxReservedTime is to specify how long backups we want to keep.</p>
</td>
</tr>
<tr>
<td>
<code>compactInterval</code></br>
<em>
string
</em>
</td>
<td>
<p>CompactInterval is to specify how long backups we want to compact.</p>
</td>
</tr>
<tr>
<td>
<code>backupTemplate</code></br>
<em>
<a href="#backupspec">
BackupSpec
</a>
</em>
</td>
<td>
<p>BackupTemplate is the specification of the backup structure to get scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>logBackupTemplate</code></br>
<em>
<a href="#backupspec">
BackupSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogBackupTemplate is the specification of the log backup structure to get scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>compactBackupTemplate</code></br>
<em>
<a href="#compactspec">
CompactSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CompactBackupTemplate is the specification of the compact backup structure to get scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Backup data storage if not storage class name set in BackupSpec.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageSize is the request storage size for backup job</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BRConfig is the configs for BR</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>StorageProvider configures where and how backups should be stored.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#backupschedulestatus">
BackupScheduleStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="dmcluster">DMCluster</h3>
<p>
<p>DMCluster is the control script&rsquo;s spec</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>DMCluster</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#dmclusterspec">
DMClusterSpec
</a>
</em>
</td>
<td>
<p>Spec defines the behavior of a dm cluster</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>discovery</code></br>
<em>
<a href="#dmdiscoveryspec">
DMDiscoverySpec
</a>
</em>
</td>
<td>
<p>Discovery spec</p>
</td>
</tr>
<tr>
<td>
<code>master</code></br>
<em>
<a href="#masterspec">
MasterSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm-master cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>worker</code></br>
<em>
<a href="#workerspec">
WorkerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm-worker cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>paused</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Indicates that the dm cluster is paused and will not be processed by
the controller.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm cluster version</p>
</td>
</tr>
<tr>
<td>
<code>schedulerName</code></br>
<em>
string
</em>
</td>
<td>
<p>SchedulerName of DM cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by DM cluster</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of DM cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>configUpdateStrategy</code></br>
<em>
<a href="#configupdatestrategy">
ConfigUpdateStrategy
</a>
</em>
</td>
<td>
<p>ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
cluster component is needed to reload the configuration change.
UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
related components to use the new ConfigMap, that is, the new configuration will be applied automatically.</p>
</td>
</tr>
<tr>
<td>
<code>enablePVReclaim</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>tlsCluster</code></br>
<em>
<a href="#tlscluster">
TLSCluster
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable the TLS connection between DM server components
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretNames</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretNames are the names of secrets which stores mysql/tidb server client certificates
that used by dm-master and dm-worker.</p>
</td>
</tr>
<tr>
<td>
<code>hostNetwork</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether Hostnetwork is enabled for DM cluster Pods
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of DM cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PriorityClassName of DM cluster Pods
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base node selectors of DM cluster Pods, components may add or override selectors upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional annotations for the dm cluster
Can be overrode by annotations in master spec or worker spec</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional labels for the dm cluster
Can be overrode by labels in master spec or worker spec</p>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of DM cluster Pods
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of DM cluster Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>dnsConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#poddnsconfig-v1-core">
Kubernetes core/v1.PodDNSConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSConfig Specifies the DNS parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>dnsPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#dnspolicy-v1-core">
Kubernetes core/v1.DNSPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSPolicy Specifies the DNSPolicy parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>statefulSetUpdateStrategy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetupdatestrategytype-v1-apps">
Kubernetes apps/v1.StatefulSetUpdateStrategyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StatefulSetUpdateStrategy of DM cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>podManagementPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podmanagementpolicytype-v1-apps">
Kubernetes apps/v1.PodManagementPolicyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodManagementPolicy of DM cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code></br>
<em>
<a href="#topologyspreadconstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TopologySpreadConstraints describes how a group of pods ought to spread across topology
domains. Scheduler will schedule pods in a way which abides by the constraints.
This field is is only honored by clusters that enables the EvenPodsSpread feature.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>suspendAction</code></br>
<em>
<a href="#suspendaction">
SuspendAction
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SuspendAction defines the suspend actions for all component.</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#dmclusterstatus">
DMClusterStatus
</a>
</em>
</td>
<td>
<p>Most recently observed status of the dm cluster</p>
</td>
</tr>
</tbody>
</table>
<h3 id="restore">Restore</h3>
<p>
<p>Restore represents the restoration of backup of a tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Restore</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#restorespec">
RestoreSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following builtin env vars will be overwritten by values set here
- S3_PROVIDER
- S3_ENDPOINT
- AWS_REGION
- AWS_ACL
- AWS_STORAGE_CLASS
- AWS_DEFAULT_REGION
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCS_PROJECT_ID
- GCS_OBJECT_ACL
- GCS_BUCKET_ACL
- GCS_LOCATION
- GCS_STORAGE_CLASS
- GCS_SERVICE_ACCOUNT_JSON_KEY
- BR_LOG_TO_TERM</p>
</td>
</tr>
<tr>
<td>
<code>to</code></br>
<em>
<a href="#tidbaccessconfig">
TiDBAccessConfig
</a>
</em>
</td>
<td>
<p>To is the tidb cluster that needs to restore.</p>
</td>
</tr>
<tr>
<td>
<code>backupType</code></br>
<em>
<a href="#backuptype">
BackupType
</a>
</em>
</td>
<td>
<p>Type is the backup type for tidb cluster and only used when Mode = snapshot, such as full, db, table.</p>
</td>
</tr>
<tr>
<td>
<code>restoreMode</code></br>
<em>
<a href="#restoremode">
RestoreMode
</a>
</em>
</td>
<td>
<p>Mode is the restore mode. such as snapshot or pitr.</p>
</td>
</tr>
<tr>
<td>
<code>pitrRestoredTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PitrRestoredTs is the pitr restored ts.</p>
</td>
</tr>
<tr>
<td>
<code>prune</code></br>
<em>
<a href="#prunetype">
PruneType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Prune is the prune type for restore, it is optional and can only have two valid values: afterFailed/alreadyFailed</p>
</td>
</tr>
<tr>
<td>
<code>logRestoreStartTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogRestoreStartTs is the start timestamp which log restore from.</p>
</td>
</tr>
<tr>
<td>
<code>federalVolumeRestorePhase</code></br>
<em>
<a href="#federalvolumerestorephase">
FederalVolumeRestorePhase
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FederalVolumeRestorePhase indicates which phase to execute in federal volume restore</p>
</td>
</tr>
<tr>
<td>
<code>volumeAZ</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>VolumeAZ indicates which AZ the volume snapshots restore to.
it is only valid for mode of volume-snapshot</p>
</td>
</tr>
<tr>
<td>
<code>tikvGCLifeTime</code></br>
<em>
string
</em>
</td>
<td>
<p>TikvGCLifeTime is to specify the safe gc life time for restore.
The time limit during which data is retained for each GC, in the format of Go Duration.
When a GC happens, the current time minus this value is the safe point.</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<p>StorageProvider configures where and how backups should be stored.</p>
</td>
</tr>
<tr>
<td>
<code>pitrFullBackupStorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PitrFullBackupStorageProvider configures where and how pitr dependent full backup should be stored.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Restore data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageSize is the request storage size for backup job</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<p>BR is the configs for BR.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of restore Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of restore Pods</p>
</td>
</tr>
<tr>
<td>
<code>useKMS</code></br>
<em>
bool
</em>
</td>
<td>
<p>Use KMS to decrypt the secrets</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify service account of restore</p>
</td>
</tr>
<tr>
<td>
<code>toolImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ToolImage specifies the tool image used in <code>Restore</code>, which supports BR and TiDB Lightning images.
For examples <code>spec.toolImage: pingcap/br:v4.0.8</code> or <code>spec.toolImage: pingcap/tidb-lightning:v4.0.8</code>
For BR image, if it does not contain tag, Pod will use image &lsquo;ToolImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>tableFilter</code></br>
<em>
[]string
</em>
</td>
<td>
<p>TableFilter means Table filter expression for &lsquo;db.table&rsquo; matching. BR supports this from v4.0.3.</p>
</td>
</tr>
<tr>
<td>
<code>warmup</code></br>
<em>
<a href="#restorewarmupmode">
RestoreWarmupMode
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Warmup represents whether to initialize TiKV volumes after volume snapshot restore</p>
</td>
</tr>
<tr>
<td>
<code>warmupImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>WarmupImage represents using what image to initialize TiKV volumes</p>
</td>
</tr>
<tr>
<td>
<code>warmupStrategy</code></br>
<em>
<a href="#restorewarmupstrategy">
RestoreWarmupStrategy
</a>
</em>
</td>
<td>
<p>WarmupStrategy</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName of Restore Job Pods</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>tolerateSingleTiKVOutage</code></br>
<em>
bool
</em>
</td>
<td>
<p>TolerateSingleTiKVOutage indicates whether to tolerate a single failure of a store without data loss</p>
</td>
</tr>
<tr>
<td>
<code>backoffLimit</code></br>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#restorestatus">
RestoreStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbcluster">TidbCluster</h3>
<p>
<p>TidbCluster is the control script&rsquo;s spec</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>TidbCluster</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#tidbclusterspec">
TidbClusterSpec
</a>
</em>
</td>
<td>
<p>Spec defines the behavior of a tidb cluster</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>discovery</code></br>
<em>
<a href="#discoveryspec">
DiscoverySpec
</a>
</em>
</td>
<td>
<p>Discovery spec</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account</p>
</td>
</tr>
<tr>
<td>
<code>pd</code></br>
<em>
<a href="#pdspec">
PDSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PD cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>pdms</code></br>
<em>
<a href="#pdmsspec">
[]PDMSSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PDMS cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tidb</code></br>
<em>
<a href="#tidbspec">
TiDBSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiDB cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tikv</code></br>
<em>
<a href="#tikvspec">
TiKVSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiKV cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tiflash</code></br>
<em>
<a href="#tiflashspec">
TiFlashSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiFlash cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>ticdc</code></br>
<em>
<a href="#ticdcspec">
TiCDCSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiCDC cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tiproxy</code></br>
<em>
<a href="#tiproxyspec">
TiProxySpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiProxy cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>pump</code></br>
<em>
<a href="#pumpspec">
PumpSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Pump cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>helper</code></br>
<em>
<a href="#helperspec">
HelperSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Helper spec</p>
</td>
</tr>
<tr>
<td>
<code>paused</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Indicates that the tidb cluster is paused and will not be processed by
the controller.</p>
</td>
</tr>
<tr>
<td>
<code>recoveryMode</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether RecoveryMode is enabled for TiDB cluster to restore
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiDB cluster version</p>
</td>
</tr>
<tr>
<td>
<code>schedulerName</code></br>
<em>
string
</em>
</td>
<td>
<p>SchedulerName of TiDB cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of TiDB cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>configUpdateStrategy</code></br>
<em>
<a href="#configupdatestrategy">
ConfigUpdateStrategy
</a>
</em>
</td>
<td>
<p>ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
cluster component is needed to reload the configuration change.
UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
related components to use the new ConfigMap, that is, the new configuration will be applied automatically.</p>
</td>
</tr>
<tr>
<td>
<code>enablePVReclaim</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>enablePVCReplace</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable PVC replace to recreate the PVC with different specs
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>tlsCluster</code></br>
<em>
<a href="#tlscluster">
TLSCluster
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable the TLS connection between TiDB server components
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>hostNetwork</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether Hostnetwork is enabled for TiDB cluster Pods
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of TiDB cluster Pods.
Will be overwritten by each cluster component&rsquo;s specific affinity setting, e.g. <code>spec.tidb.affinity</code></p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PriorityClassName of TiDB cluster Pods
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base node selectors of TiDB cluster Pods, components may add or override selectors upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base annotations for TiDB cluster, all Pods in the cluster should have these annotations.
Can be overrode by annotations in the specific component spec.</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base labels for TiDB cluster, all Pods in the cluster should have these labels.
Can be overrode by labels in the specific component spec.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of TiDB cluster Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>dnsConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#poddnsconfig-v1-core">
Kubernetes core/v1.PodDNSConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSConfig Specifies the DNS parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>dnsPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#dnspolicy-v1-core">
Kubernetes core/v1.DNSPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSPolicy Specifies the DNSPolicy parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TiDB cluster Pods
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>services</code></br>
<em>
<a href="#service">
[]Service
</a>
</em>
</td>
<td>
<p>(Deprecated) Services list non-headless services type used in TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>enableDynamicConfiguration</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableDynamicConfiguration indicates whether to append <code>--advertise-status-addr</code> to the startup parameters of TiKV.</p>
</td>
</tr>
<tr>
<td>
<code>clusterDomain</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterDomain is the Kubernetes Cluster Domain of TiDB cluster
Optional: Defaults to &ldquo;&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>acrossK8s</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>AcrossK8s indicates whether deploy TiDB cluster across multiple Kubernetes clusters</p>
</td>
</tr>
<tr>
<td>
<code>cluster</code></br>
<em>
<a href="#tidbclusterref">
TidbClusterRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Cluster is the external cluster, if configured, the components in this TidbCluster will join to this configured cluster.</p>
</td>
</tr>
<tr>
<td>
<code>pdAddresses</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PDAddresses are the external PD addresses, if configured, the PDs in this TidbCluster will join to the configured PD cluster.</p>
</td>
</tr>
<tr>
<td>
<code>statefulSetUpdateStrategy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetupdatestrategytype-v1-apps">
Kubernetes apps/v1.StatefulSetUpdateStrategyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StatefulSetUpdateStrategy of TiDB cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>podManagementPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podmanagementpolicytype-v1-apps">
Kubernetes apps/v1.PodManagementPolicyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodManagementPolicy of TiDB cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code></br>
<em>
<a href="#topologyspreadconstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TopologySpreadConstraints describes how a group of pods ought to spread across topology
domains. Scheduler will schedule pods in a way which abides by the constraints.
This field is is only honored by clusters that enables the EvenPodsSpread feature.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>startScriptVersion</code></br>
<em>
<a href="#startscriptversion">
StartScriptVersion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StartScriptVersion is the version of start script
When PD enables microservice mode, pd and pd microservice component will use start script v2.</p>
<p>default to &ldquo;v1&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>suspendAction</code></br>
<em>
<a href="#suspendaction">
SuspendAction
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SuspendAction defines the suspend actions for all component.</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
<tr>
<td>
<code>startScriptV2FeatureFlags</code></br>
<em>
<a href="#startscriptv2featureflag">
[]StartScriptV2FeatureFlag
</a>
</em>
</td>
<td>
<p>Feature flags used by v2 startup script to enable various features.
Examples of supported feature flags:
- WaitForDnsNameIpMatch indicates whether PD and TiKV has to wait until local IP address matches the one published to external DNS
- PreferPDAddressesOverDiscovery advises start script to use TidbClusterSpec.PDAddresses (if supplied) as argument for pd-server, tikv-server and tidb-server commands</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#tidbclusterstatus">
TidbClusterStatus
</a>
</em>
</td>
<td>
<p>Most recently observed status of the tidb cluster</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbinitializer">TidbInitializer</h3>
<p>
<p>TidbInitializer is a TiDB cluster initializing job</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>TidbInitializer</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#tidbinitializerspec">
TidbInitializerSpec
</a>
</em>
</td>
<td>
<p>Spec defines the desired state of TidbInitializer</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cluster</code></br>
<em>
<a href="#tidbclusterref">
TidbClusterRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>permitHost</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>permitHost is the host which will only be allowed to connect to the TiDB.</p>
</td>
</tr>
<tr>
<td>
<code>initSql</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InitSql is the SQL statements executed after the TiDB cluster is bootstrapped.</p>
</td>
</tr>
<tr>
<td>
<code>initSqlConfigMap</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InitSqlConfigMapName reference a configmap that provide init-sql, take high precedence than initSql if set</p>
</td>
</tr>
<tr>
<td>
<code>passwordSecret</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TiDB initializer Pods</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretName is the name of secret which stores tidb server client certificate
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tolerations of the TiDB initializer Pod</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Node selectors of TiDB initializer Pod</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#tidbinitializerstatus">
TidbInitializerStatus
</a>
</em>
</td>
<td>
<p>Most recently observed status of the TidbInitializer</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbmonitor">TidbMonitor</h3>
<p>
<p>TidbMonitor encode the spec and status of the monitoring component of a TiDB cluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>TidbMonitor</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#tidbmonitorspec">
TidbMonitorSpec
</a>
</em>
</td>
<td>
<p>Spec defines the desired state of TidbMonitor</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#tidbclusterref">
[]TidbClusterRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>monitored TiDB cluster info</p>
</td>
</tr>
<tr>
<td>
<code>prometheus</code></br>
<em>
<a href="#prometheusspec">
PrometheusSpec
</a>
</em>
</td>
<td>
<p>Prometheus spec</p>
</td>
</tr>
<tr>
<td>
<code>grafana</code></br>
<em>
<a href="#grafanaspec">
GrafanaSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Grafana spec</p>
</td>
</tr>
<tr>
<td>
<code>reloader</code></br>
<em>
<a href="#reloaderspec">
ReloaderSpec
</a>
</em>
</td>
<td>
<p>Reloader spec</p>
</td>
</tr>
<tr>
<td>
<code>initializer</code></br>
<em>
<a href="#initializerspec">
InitializerSpec
</a>
</em>
</td>
<td>
<p>Initializer spec</p>
</td>
</tr>
<tr>
<td>
<code>dm</code></br>
<em>
<a href="#dmmonitorspec">
DMMonitorSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>monitored DM cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>thanos</code></br>
<em>
<a href="#thanosspec">
ThanosSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Thanos spec</p>
</td>
</tr>
<tr>
<td>
<code>prometheusReloader</code></br>
<em>
<a href="#prometheusreloaderspec">
PrometheusReloaderSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PrometheusReloader set prometheus reloader configuration</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of TidbMonitor Pods</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>persistent</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>If Persistent enabled, storageClassName must be set to an existing storage.
Defaults to false.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for TidbMonitor data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Size of the persistent volume.</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NodeSelector of the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations for the TidbMonitor.
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Labels for the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tolerations of the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>kubePrometheusURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>kubePrometheusURL is where tidb-monitoring get the  common metrics of kube-prometheus.
Ref: <a href="https://github.com/coreos/kube-prometheus">https://github.com/coreos/kube-prometheus</a></p>
</td>
</tr>
<tr>
<td>
<code>alertmanagerURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>alertmanagerURL is where tidb-monitoring push alerts to.
Ref: <a href="https://prometheus.io/docs/alerting/alertmanager/">https://prometheus.io/docs/alerting/alertmanager/</a></p>
</td>
</tr>
<tr>
<td>
<code>alertManagerRulesVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>alertManagerRulesVersion is the version of the tidb cluster that used for alert rules.
default to current tidb cluster version, for example: v3.0.15</p>
</td>
</tr>
<tr>
<td>
<code>additionalContainers</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core">
[]Kubernetes core/v1.Container
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional containers of the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>clusterScoped</code></br>
<em>
bool
</em>
</td>
<td>
<p>ClusterScoped indicates whether this monitor should manage Kubernetes cluster-wide TiDB clusters</p>
</td>
</tr>
<tr>
<td>
<code>externalLabels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>The labels to add to any time series or alerts when communicating with
external systems (federation, remote storage, Alertmanager).</p>
</td>
</tr>
<tr>
<td>
<code>replicaExternalLabelName</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of Prometheus external label used to denote replica name.
Defaults to the value of <code>prometheus_replica</code>. External label will
<em>not</em> be added when value is set to empty string (<code>&quot;&quot;</code>).</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Replicas is the number of desired replicas.
Defaults to 1.</p>
</td>
</tr>
<tr>
<td>
<code>shards</code></br>
<em>
int32
</em>
</td>
<td>
<p>EXPERIMENTAL: Number of shards to distribute targets onto. Number of
replicas multiplied by shards is the total number of Pods created. Note
that scaling down shards will not reshard data onto remaining instances,
it must be manually moved. Increasing shards will not reshard data
either but it will continue to be available from the same instances. To
query globally use Thanos sidecar and Thanos querier or remote write
data to a central location. Sharding is done on the content of the
<code>__address__</code> target meta-label.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of TidbMonitor pod.</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of TidbMonitor pod.</p>
</td>
</tr>
<tr>
<td>
<code>enableAlertRules</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableAlertRules adds alert rules to the Prometheus config even
if <code>AlertmanagerURL</code> is not configured.</p>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TidbMonitor
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#tidbmonitorstatus">
TidbMonitorStatus
</a>
</em>
</td>
<td>
<p>Most recently observed status of the TidbMonitor</p>
</td>
</tr>
</tbody>
</table>
<h3 id="azblobstorageprovider">AzblobStorageProvider</h3>
<p>
(<em>Appears on:</em>
<a href="#storageprovider">StorageProvider</a>)
</p>
<p>
<p>AzblobStorageProvider represents the azure blob storage for storing backups.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path is the full path where the backup is saved.
The format of the path must be: &ldquo;<container-name>/<path-to-backup-file>&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>container</code></br>
<em>
string
</em>
</td>
<td>
<p>Container in which to store the backup data.</p>
</td>
</tr>
<tr>
<td>
<code>accessTier</code></br>
<em>
string
</em>
</td>
<td>
<p>Access tier of the uploaded objects.</p>
</td>
</tr>
<tr>
<td>
<code>secretName</code></br>
<em>
string
</em>
</td>
<td>
<p>SecretName is the name of secret which stores the
azblob service account credentials.</p>
</td>
</tr>
<tr>
<td>
<code>storageAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageAccount is the storage account of the azure blob storage
If this field is set, then use this to set backup-manager env
Otherwise retrieve the storage account from secret</p>
</td>
</tr>
<tr>
<td>
<code>sasToken</code></br>
<em>
string
</em>
</td>
<td>
<p>SasToken is the sas token of the storage account</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<p>Prefix of the data path.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="brconfig">BRConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#backupschedulespec">BackupScheduleSpec</a>, 
<a href="#backupspec">BackupSpec</a>, 
<a href="#compactspec">CompactSpec</a>, 
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>BRConfig contains config for BR</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>cluster</code></br>
<em>
string
</em>
</td>
<td>
<p>ClusterName of backup/restore cluster</p>
</td>
</tr>
<tr>
<td>
<code>clusterNamespace</code></br>
<em>
string
</em>
</td>
<td>
<p>Namespace of backup/restore cluster</p>
</td>
</tr>
<tr>
<td>
<code>db</code></br>
<em>
string
</em>
</td>
<td>
<p>Deprecated from BR v4.0.3. Please use <code>Spec.TableFilter</code> instead. DB is the specific DB which will be backed-up or restored</p>
</td>
</tr>
<tr>
<td>
<code>table</code></br>
<em>
string
</em>
</td>
<td>
<p>Deprecated from BR v4.0.3. Please use <code>Spec.TableFilter</code> instead. Table is the specific table which will be backed-up or restored</p>
</td>
</tr>
<tr>
<td>
<code>logLevel</code></br>
<em>
string
</em>
</td>
<td>
<p>LogLevel is the log level</p>
</td>
</tr>
<tr>
<td>
<code>statusAddr</code></br>
<em>
string
</em>
</td>
<td>
<p>StatusAddr is the HTTP listening address for the status report service. Set to empty string to disable</p>
</td>
</tr>
<tr>
<td>
<code>concurrency</code></br>
<em>
uint32
</em>
</td>
<td>
<p>Concurrency is the size of thread pool on each node that execute the backup task</p>
</td>
</tr>
<tr>
<td>
<code>rateLimit</code></br>
<em>
uint
</em>
</td>
<td>
<p>RateLimit is the rate limit of the backup task, MB/s per node</p>
</td>
</tr>
<tr>
<td>
<code>timeAgo</code></br>
<em>
string
</em>
</td>
<td>
<p>TimeAgo is the history version of the backup task, e.g. 1m, 1h</p>
</td>
</tr>
<tr>
<td>
<code>checksum</code></br>
<em>
bool
</em>
</td>
<td>
<p>Checksum specifies whether to run checksum after backup</p>
</td>
</tr>
<tr>
<td>
<code>checkRequirements</code></br>
<em>
bool
</em>
</td>
<td>
<p>CheckRequirements specifies whether to check requirements</p>
</td>
</tr>
<tr>
<td>
<code>sendCredToTikv</code></br>
<em>
bool
</em>
</td>
<td>
<p>SendCredToTikv specifies whether to send credentials to TiKV</p>
</td>
</tr>
<tr>
<td>
<code>onLine</code></br>
<em>
bool
</em>
</td>
<td>
<p>OnLine specifies whether online during restore</p>
</td>
</tr>
<tr>
<td>
<code>options</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Options means options for backup data to remote storage with BR. These options has highest priority.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="backoffretrypolicy">BackoffRetryPolicy</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>)
</p>
<p>
<p>BackoffRetryPolicy is the backoff retry policy, currently only valid for snapshot backup.
When backup job or pod failed, it will retry in the following way:
first time: retry after MinRetryDuration
second time: retry after MinRetryDuration * 2
third time: retry after MinRetryDuration * 2 * 2
&hellip;
as the limit:
1. the number of retries can not exceed MaxRetryTimes
2. the time from discovery failure can not exceed RetryTimeout</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>minRetryDuration</code></br>
<em>
string
</em>
</td>
<td>
<p>MinRetryDuration is the min retry duration, the retry duration will be MinRetryDuration &lt;&lt; (retry num -1)
format reference, <a href="https://golang.org/pkg/time/#ParseDuration">https://golang.org/pkg/time/#ParseDuration</a></p>
</td>
</tr>
<tr>
<td>
<code>maxRetryTimes</code></br>
<em>
int
</em>
</td>
<td>
<p>MaxRetryTimes is the max retry times</p>
</td>
</tr>
<tr>
<td>
<code>retryTimeout</code></br>
<em>
string
</em>
</td>
<td>
<p>RetryTimeout is the retry timeout
format reference, <a href="https://golang.org/pkg/time/#ParseDuration">https://golang.org/pkg/time/#ParseDuration</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="backoffretryrecord">BackoffRetryRecord</h3>
<p>
(<em>Appears on:</em>
<a href="#backupstatus">BackupStatus</a>)
</p>
<p>
<p>BackoffRetryRecord is the record of backoff retry</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>retryNum</code></br>
<em>
int
</em>
</td>
<td>
<p>RetryNum is the number of retry</p>
</td>
</tr>
<tr>
<td>
<code>detectFailedAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>DetectFailedAt is the time when detect failure</p>
</td>
</tr>
<tr>
<td>
<code>expectedRetryAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>ExpectedRetryAt is the time we calculate and expect retry after it</p>
</td>
</tr>
<tr>
<td>
<code>realRetryAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>RealRetryAt is the time when the retry was actually initiated</p>
</td>
</tr>
<tr>
<td>
<code>retryReason</code></br>
<em>
string
</em>
</td>
<td>
<p>Reason is the reason of retry</p>
</td>
</tr>
<tr>
<td>
<code>originalReason</code></br>
<em>
string
</em>
</td>
<td>
<p>OriginalReason is the original reason of backup job or pod failed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="backupcondition">BackupCondition</h3>
<p>
(<em>Appears on:</em>
<a href="#backupstatus">BackupStatus</a>, 
<a href="#logsubcommandstatus">LogSubCommandStatus</a>)
</p>
<p>
<p>BackupCondition describes the observed state of a Backup at a certain point.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>command</code></br>
<em>
<a href="#logsubcommandtype">
LogSubCommandType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#backupconditiontype">
BackupConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="backupconditiontype">BackupConditionType</h3>
<p>
(<em>Appears on:</em>
<a href="#backupcondition">BackupCondition</a>, 
<a href="#backupstatus">BackupStatus</a>, 
<a href="#logsubcommandstatus">LogSubCommandStatus</a>)
</p>
<p>
<p>BackupConditionType represents a valid condition of a Backup.</p>
</p>
<h3 id="backupmode">BackupMode</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>)
</p>
<p>
<p>BackupType represents the backup mode, such as snapshot backup or log backup.</p>
</p>
<h3 id="backupschedulespec">BackupScheduleSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#backupschedule">BackupSchedule</a>)
</p>
<p>
<p>BackupScheduleSpec contains the backup schedule specification for a tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>schedule</code></br>
<em>
string
</em>
</td>
<td>
<p>Schedule specifies the cron string used for backup scheduling.</p>
</td>
</tr>
<tr>
<td>
<code>pause</code></br>
<em>
bool
</em>
</td>
<td>
<p>Pause means paused backupSchedule</p>
</td>
</tr>
<tr>
<td>
<code>maxBackups</code></br>
<em>
int32
</em>
</td>
<td>
<p>MaxBackups is to specify how many backups we want to keep
0 is magic number to indicate un-limited backups.
if MaxBackups and MaxReservedTime are set at the same time, MaxReservedTime is preferred
and MaxBackups is ignored.</p>
</td>
</tr>
<tr>
<td>
<code>maxReservedTime</code></br>
<em>
string
</em>
</td>
<td>
<p>MaxReservedTime is to specify how long backups we want to keep.</p>
</td>
</tr>
<tr>
<td>
<code>compactInterval</code></br>
<em>
string
</em>
</td>
<td>
<p>CompactInterval is to specify how long backups we want to compact.</p>
</td>
</tr>
<tr>
<td>
<code>backupTemplate</code></br>
<em>
<a href="#backupspec">
BackupSpec
</a>
</em>
</td>
<td>
<p>BackupTemplate is the specification of the backup structure to get scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>logBackupTemplate</code></br>
<em>
<a href="#backupspec">
BackupSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogBackupTemplate is the specification of the log backup structure to get scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>compactBackupTemplate</code></br>
<em>
<a href="#compactspec">
CompactSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CompactBackupTemplate is the specification of the compact backup structure to get scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Backup data storage if not storage class name set in BackupSpec.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageSize is the request storage size for backup job</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BRConfig is the configs for BR</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>StorageProvider configures where and how backups should be stored.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="backupschedulestatus">BackupScheduleStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#backupschedule">BackupSchedule</a>)
</p>
<p>
<p>BackupScheduleStatus represents the current state of a BackupSchedule.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>lastBackup</code></br>
<em>
string
</em>
</td>
<td>
<p>LastBackup represents the last backup.</p>
</td>
</tr>
<tr>
<td>
<code>lastCompact</code></br>
<em>
string
</em>
</td>
<td>
<p>LastCompact represents the last compact</p>
</td>
</tr>
<tr>
<td>
<code>logBackup</code></br>
<em>
string
</em>
</td>
<td>
<p>logBackup represents the name of log backup.</p>
</td>
</tr>
<tr>
<td>
<code>logBackupStartTs</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LogBackupStartTs represents the start time of log backup</p>
</td>
</tr>
<tr>
<td>
<code>lastBackupTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastBackupTime represents the last time the backup was successfully created.</p>
</td>
</tr>
<tr>
<td>
<code>lastCompactProgress</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastCompactProgress represents the endTs of the last compact</p>
</td>
</tr>
<tr>
<td>
<code>lastCompactExecutionTs</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastCompactExecutionTs represents the execution time of the last compact</p>
</td>
</tr>
<tr>
<td>
<code>allBackupCleanTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>AllBackupCleanTime represents the time when all backup entries are cleaned up</p>
</td>
</tr>
</tbody>
</table>
<h3 id="backupspec">BackupSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#backup">Backup</a>, 
<a href="#backupschedulespec">BackupScheduleSpec</a>)
</p>
<p>
<p>BackupSpec contains the backup specification for a tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following builtin env vars will be overwritten by values set here
- S3_PROVIDER
- S3_ENDPOINT
- AWS_REGION
- AWS_ACL
- AWS_STORAGE_CLASS
- AWS_DEFAULT_REGION
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCS_PROJECT_ID
- GCS_OBJECT_ACL
- GCS_BUCKET_ACL
- GCS_LOCATION
- GCS_STORAGE_CLASS
- GCS_SERVICE_ACCOUNT_JSON_KEY
- BR_LOG_TO_TERM</p>
</td>
</tr>
<tr>
<td>
<code>from</code></br>
<em>
<a href="#tidbaccessconfig">
TiDBAccessConfig
</a>
</em>
</td>
<td>
<p>From is the tidb cluster that needs to backup.</p>
</td>
</tr>
<tr>
<td>
<code>backupType</code></br>
<em>
<a href="#backuptype">
BackupType
</a>
</em>
</td>
<td>
<p>Type is the backup type for tidb cluster and only used when Mode = snapshot, such as full, db, table.</p>
</td>
</tr>
<tr>
<td>
<code>backupMode</code></br>
<em>
<a href="#backupmode">
BackupMode
</a>
</em>
</td>
<td>
<p>Mode is the backup mode, such as snapshot backup or log backup.</p>
</td>
</tr>
<tr>
<td>
<code>tikvGCLifeTime</code></br>
<em>
string
</em>
</td>
<td>
<p>TikvGCLifeTime is to specify the safe gc life time for backup.
The time limit during which data is retained for each GC, in the format of Go Duration.
When a GC happens, the current time minus this value is the safe point.</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<p>StorageProvider configures where and how backups should be stored.
*** Note: This field should generally not be left empty, unless you are certain the storage provider
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Backup data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageSize is the request storage size for backup job</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<p>BRConfig is the configs for BR
*** Note: This field should generally not be left empty, unless you are certain the BR config
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>commitTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.
Default is current timestamp.</p>
</td>
</tr>
<tr>
<td>
<code>logSubcommand</code></br>
<em>
<a href="#logsubcommandtype">
LogSubCommandType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subcommand is the subcommand for BR, such as start, stop, pause etc.</p>
</td>
</tr>
<tr>
<td>
<code>logTruncateUntil</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogTruncateUntil is log backup truncate until timestamp.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>logStop</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogStop indicates that will stop the log backup.</p>
</td>
</tr>
<tr>
<td>
<code>calcSizeLevel</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CalcSizeLevel determines how to size calculation of snapshots for EBS volume snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>federalVolumeBackupPhase</code></br>
<em>
<a href="#federalvolumebackupphase">
FederalVolumeBackupPhase
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FederalVolumeBackupPhase indicates which phase to execute in federal volume backup</p>
</td>
</tr>
<tr>
<td>
<code>resumeGcSchedule</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResumeGcSchedule indicates whether resume gc and pd scheduler for EBS volume snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>dumpling</code></br>
<em>
<a href="#dumplingconfig">
DumplingConfig
</a>
</em>
</td>
<td>
<p>DumplingConfig is the configs for dumpling</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of backup Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>toolImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ToolImage specifies the tool image used in <code>Backup</code>, which supports BR and Dumpling images.
For examples <code>spec.toolImage: pingcap/br:v4.0.8</code> or <code>spec.toolImage: pingcap/dumpling:v4.0.8</code>
For BR image, if it does not contain tag, Pod will use image &lsquo;ToolImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>tableFilter</code></br>
<em>
[]string
</em>
</td>
<td>
<p>TableFilter means Table filter expression for &lsquo;db.table&rsquo; matching. BR supports this from v4.0.3.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of backup Pods</p>
</td>
</tr>
<tr>
<td>
<code>useKMS</code></br>
<em>
bool
</em>
</td>
<td>
<p>Use KMS to decrypt the secrets</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify service account of backup</p>
</td>
</tr>
<tr>
<td>
<code>cleanPolicy</code></br>
<em>
<a href="#cleanpolicytype">
CleanPolicyType
</a>
</em>
</td>
<td>
<p>CleanPolicy denotes whether to clean backup data when the object is deleted from the cluster, if not set, the backup data will be retained</p>
</td>
</tr>
<tr>
<td>
<code>cleanOption</code></br>
<em>
<a href="#cleanoption">
CleanOption
</a>
</em>
</td>
<td>
<p>CleanOption controls the behavior of clean.</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName of Backup Job Pods</p>
</td>
</tr>
<tr>
<td>
<code>backoffRetryPolicy</code></br>
<em>
<a href="#backoffretrypolicy">
BackoffRetryPolicy
</a>
</em>
</td>
<td>
<p>BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>volumeBackupInitJobMaxActiveSeconds</code></br>
<em>
int
</em>
</td>
<td>
<p>VolumeBackupInitJobMaxActiveSeconds represents the deadline (in seconds) of the vbk init job</p>
</td>
</tr>
</tbody>
</table>
<h3 id="backupstatus">BackupStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#backup">Backup</a>)
</p>
<p>
<p>BackupStatus represents the current status of a backup.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>backupPath</code></br>
<em>
string
</em>
</td>
<td>
<p>BackupPath is the location of the backup.</p>
</td>
</tr>
<tr>
<td>
<code>timeStarted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeStarted is the time at which the backup was started.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
<tr>
<td>
<code>timeCompleted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeCompleted is the time at which the backup was completed.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
<tr>
<td>
<code>timeTaken</code></br>
<em>
string
</em>
</td>
<td>
<p>TimeTaken is the time that backup takes, it is TimeCompleted - TimeStarted</p>
</td>
</tr>
<tr>
<td>
<code>backupSizeReadable</code></br>
<em>
string
</em>
</td>
<td>
<p>BackupSizeReadable is the data size of the backup.
the difference with BackupSize is that its format is human readable</p>
</td>
</tr>
<tr>
<td>
<code>backupSize</code></br>
<em>
int64
</em>
</td>
<td>
<p>BackupSize is the data size of the backup.</p>
</td>
</tr>
<tr>
<td>
<code>incrementalBackupSizeReadable</code></br>
<em>
string
</em>
</td>
<td>
<p>the difference with IncrementalBackupSize is that its format is human readable</p>
</td>
</tr>
<tr>
<td>
<code>incrementalBackupSize</code></br>
<em>
int64
</em>
</td>
<td>
<p>IncrementalBackupSize is the incremental data size of the backup, it is only used for volume snapshot backup
it is the real size of volume snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>commitTs</code></br>
<em>
string
</em>
</td>
<td>
<p>CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.</p>
</td>
</tr>
<tr>
<td>
<code>logSuccessTruncateUntil</code></br>
<em>
string
</em>
</td>
<td>
<p>LogSuccessTruncateUntil is log backup already successfully truncate until timestamp.</p>
</td>
</tr>
<tr>
<td>
<code>logCheckpointTs</code></br>
<em>
string
</em>
</td>
<td>
<p>LogCheckpointTs is the ts of log backup process.</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#backupconditiontype">
BackupConditionType
</a>
</em>
</td>
<td>
<p>Phase is a user readable state inferred from the underlying Backup conditions</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#backupcondition">
[]BackupCondition
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>logSubCommandStatuses</code></br>
<em>
<a href="#logsubcommandstatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.LogSubCommandType]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.LogSubCommandStatus
</a>
</em>
</td>
<td>
<p>LogSubCommandStatuses is the detail status of log backup subcommands, record each command separately, but only record the last command.</p>
</td>
</tr>
<tr>
<td>
<code>progresses</code></br>
<em>
<a href="#progress">
[]Progress
</a>
</em>
</td>
<td>
<p>Progresses is the progress of backup.</p>
</td>
</tr>
<tr>
<td>
<code>backoffRetryStatus</code></br>
<em>
<a href="#backoffretryrecord">
[]BackoffRetryRecord
</a>
</em>
</td>
<td>
<p>BackoffRetryStatus is status of the backoff retry, it will be used when backup pod or job exited unexpectedly</p>
</td>
</tr>
</tbody>
</table>
<h3 id="backupstoragetype">BackupStorageType</h3>
<p>
<p>BackupStorageType represents the backend storage type of backup.</p>
</p>
<h3 id="backuptype">BackupType</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>, 
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>BackupType represents the backup type.</p>
</p>
<h3 id="basicauth">BasicAuth</h3>
<p>
(<em>Appears on:</em>
<a href="#remotewritespec">RemoteWriteSpec</a>)
</p>
<p>
<p>BasicAuth allow an endpoint to authenticate over basic authentication
More info: <a href="https://prometheus.io/docs/operating/configuration/#endpoints">https://prometheus.io/docs/operating/configuration/#endpoints</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>username</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The secret in the service monitor namespace that contains the username
for authentication.</p>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The secret in the service monitor namespace that contains the password
for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="batchdeleteoption">BatchDeleteOption</h3>
<p>
(<em>Appears on:</em>
<a href="#cleanoption">CleanOption</a>)
</p>
<p>
<p>BatchDeleteOption controls the options to delete the objects in batches during the cleanup of backups</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>disableBatchConcurrency</code></br>
<em>
bool
</em>
</td>
<td>
<p>DisableBatchConcurrency disables the batch deletions with S3 API and the deletion will be done by goroutines.</p>
</td>
</tr>
<tr>
<td>
<code>batchConcurrency</code></br>
<em>
uint32
</em>
</td>
<td>
<p>BatchConcurrency represents the number of batch deletions in parallel.
It is used when the storage provider supports the batch delete API, currently, S3 only.
default is 10</p>
</td>
</tr>
<tr>
<td>
<code>routineConcurrency</code></br>
<em>
uint32
</em>
</td>
<td>
<p>RoutineConcurrency represents the number of goroutines that used to delete objects
default is 100</p>
</td>
</tr>
</tbody>
</table>
<h3 id="binlog">Binlog</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Binlog is the config for binlog.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enable</code></br>
<em>
bool
</em>
</td>
<td>
<p>optional</p>
</td>
</tr>
<tr>
<td>
<code>write-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 15s</p>
</td>
</tr>
<tr>
<td>
<code>ignore-error</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>If IgnoreError is true, when writing binlog meets error, TiDB would
ignore the error.</p>
</td>
</tr>
<tr>
<td>
<code>binlog-socket</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Use socket file to write binlog, for compatible with kafka version tidb-binlog.</p>
</td>
</tr>
<tr>
<td>
<code>strategy</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The strategy for sending binlog to pump, value can be &ldquo;range,omitempty&rdquo; or &ldquo;hash,omitempty&rdquo; now.
Optional: Defaults to range</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cdcconfigwraper">CDCConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#ticdcspec">TiCDCSpec</a>)
</p>
<p>
<p>CDCConfigWraper simply wrapps a GenericConfig</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cleanoption">CleanOption</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>)
</p>
<p>
<p>CleanOption defines the configuration for cleanup backup</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>pageSize</code></br>
<em>
uint64
</em>
</td>
<td>
<p>PageSize represents the number of objects to clean at a time.
default is 10000</p>
</td>
</tr>
<tr>
<td>
<code>retryCount</code></br>
<em>
int
</em>
</td>
<td>
<p>RetryCount represents the number of retries in pod when the cleanup fails.</p>
</td>
</tr>
<tr>
<td>
<code>backoffEnabled</code></br>
<em>
bool
</em>
</td>
<td>
<p>BackoffEnabled represents whether to enable the backoff when a deletion API fails.
It is useful when the deletion API is rate limited.</p>
</td>
</tr>
<tr>
<td>
<code>BatchDeleteOption</code></br>
<em>
<a href="#batchdeleteoption">
BatchDeleteOption
</a>
</em>
</td>
<td>
<p>
(Members of <code>BatchDeleteOption</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>snapshotsDeleteRatio</code></br>
<em>
float64
</em>
</td>
<td>
<p>SnapshotsDeleteRatio represents the number of snapshots deleted per second</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cleanpolicytype">CleanPolicyType</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>)
</p>
<p>
<p>CleanPolicyType represents the clean policy of backup data in remote storage</p>
</p>
<h3 id="cluster">Cluster</h3>
<p>
</p>
<h3 id="clusterref">ClusterRef</h3>
<p>
(<em>Appears on:</em>
<a href="#dmmonitorspec">DMMonitorSpec</a>)
</p>
<p>
<p>ClusterRef reference to a TidbCluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace is the namespace that TidbCluster object locates,
default to the same namespace as TidbMonitor/TidbCluster/TidbNGMonitoring/TidbDashboard</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of TidbCluster object</p>
</td>
</tr>
<tr>
<td>
<code>clusterDomain</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterDomain is the domain of TidbCluster object</p>
</td>
</tr>
</tbody>
</table>
<h3 id="commonconfig">CommonConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashconfig">TiFlashConfig</a>)
</p>
<p>
<p>CommonConfig is the configuration of TiFlash process.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>tmp_path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to &ldquo;/data0/tmp&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>path_realtime_mode</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>mark_cache_size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 5368709120</p>
</td>
</tr>
<tr>
<td>
<code>minmax_index_cache_size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 5368709120</p>
</td>
</tr>
<tr>
<td>
<code>flash</code></br>
<em>
<a href="#flash">
Flash
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>logger</code></br>
<em>
<a href="#flashlogger">
FlashLogger
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>security</code></br>
<em>
<a href="#flashsecurity">
FlashSecurity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="compactbackup">CompactBackup</h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#compactspec">
CompactSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following builtin env vars will be overwritten by values set here
- S3_PROVIDER
- S3_ENDPOINT
- AWS_REGION
- AWS_ACL
- AWS_STORAGE_CLASS
- AWS_DEFAULT_REGION
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCS_PROJECT_ID
- GCS_OBJECT_ACL
- GCS_BUCKET_ACL
- GCS_LOCATION
- GCS_STORAGE_CLASS
- GCS_SERVICE_ACCOUNT_JSON_KEY
- BR_LOG_TO_TERM</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<p>StorageProvider configures where and how backups should be stored.
*** Note: This field should generally not be left empty, unless you are certain the storage provider
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>startTs</code></br>
<em>
string
</em>
</td>
<td>
<p>StartTs is the start ts of the compact backup.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>endTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>EndTs is the end ts of the compact backup.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.
Default is current timestamp.</p>
</td>
</tr>
<tr>
<td>
<code>concurrency</code></br>
<em>
int
</em>
</td>
<td>
<p>Concurrency is the concurrency of compact backup job</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of backup Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>toolImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ToolImage specifies the br image used in compact <code>Backup</code>.
For examples <code>spec.toolImage: pingcap/br:v4.0.8</code>
For BR image, if it does not contain tag, Pod will use the same version in tc &lsquo;BrImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>tikvImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TikvImage specifies the tikv image used in compact <code>Backup</code>.
For examples <code>spec.tikvImage: pingcap/tikv:v9.0.0</code>
For TiKV image, if it does not contain tag, Pod will use the same version in tc &lsquo;TiKVImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<p>BRConfig is the configs for BR
*** Note: This field should generally not be left empty, unless you are certain the BR config
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of backup Pods</p>
</td>
</tr>
<tr>
<td>
<code>useKMS</code></br>
<em>
bool
</em>
</td>
<td>
<p>Use KMS to decrypt the secrets</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify service account of backup</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName of Backup Job Pods</p>
</td>
</tr>
<tr>
<td>
<code>maxRetryTimes</code></br>
<em>
int32
</em>
</td>
<td>
<p>BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#compactstatus">
CompactStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="compactretryrecord">CompactRetryRecord</h3>
<p>
(<em>Appears on:</em>
<a href="#compactstatus">CompactStatus</a>)
</p>
<p>
<p>CompactRetryRecord is the record of compact backoff retry</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>retryNum</code></br>
<em>
int
</em>
</td>
<td>
<p>RetryNum is the number of retry</p>
</td>
</tr>
<tr>
<td>
<code>detectFailedAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>DetectFailedAt is the time when detect failure</p>
</td>
</tr>
<tr>
<td>
<code>retryReason</code></br>
<em>
string
</em>
</td>
<td>
<p>Reason is the reason of retry</p>
</td>
</tr>
</tbody>
</table>
<h3 id="compactspec">CompactSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#backupschedulespec">BackupScheduleSpec</a>, 
<a href="#compactbackup">CompactBackup</a>)
</p>
<p>
<p>CompactSpec contains the backup specification for a tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following builtin env vars will be overwritten by values set here
- S3_PROVIDER
- S3_ENDPOINT
- AWS_REGION
- AWS_ACL
- AWS_STORAGE_CLASS
- AWS_DEFAULT_REGION
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCS_PROJECT_ID
- GCS_OBJECT_ACL
- GCS_BUCKET_ACL
- GCS_LOCATION
- GCS_STORAGE_CLASS
- GCS_SERVICE_ACCOUNT_JSON_KEY
- BR_LOG_TO_TERM</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<p>StorageProvider configures where and how backups should be stored.
*** Note: This field should generally not be left empty, unless you are certain the storage provider
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>startTs</code></br>
<em>
string
</em>
</td>
<td>
<p>StartTs is the start ts of the compact backup.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>endTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>EndTs is the end ts of the compact backup.
Format supports TSO or datetime, e.g. &lsquo;400036290571534337&rsquo;, &lsquo;2018-05-11 01:42:23&rsquo;.
Default is current timestamp.</p>
</td>
</tr>
<tr>
<td>
<code>concurrency</code></br>
<em>
int
</em>
</td>
<td>
<p>Concurrency is the concurrency of compact backup job</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of backup Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>toolImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ToolImage specifies the br image used in compact <code>Backup</code>.
For examples <code>spec.toolImage: pingcap/br:v4.0.8</code>
For BR image, if it does not contain tag, Pod will use the same version in tc &lsquo;BrImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>tikvImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TikvImage specifies the tikv image used in compact <code>Backup</code>.
For examples <code>spec.tikvImage: pingcap/tikv:v9.0.0</code>
For TiKV image, if it does not contain tag, Pod will use the same version in tc &lsquo;TiKVImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<p>BRConfig is the configs for BR
*** Note: This field should generally not be left empty, unless you are certain the BR config
*** can be obtained from another source, such as a schedule CR.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of backup Pods</p>
</td>
</tr>
<tr>
<td>
<code>useKMS</code></br>
<em>
bool
</em>
</td>
<td>
<p>Use KMS to decrypt the secrets</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify service account of backup</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName of Backup Job Pods</p>
</td>
</tr>
<tr>
<td>
<code>maxRetryTimes</code></br>
<em>
int32
</em>
</td>
<td>
<p>BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="compactstatus">CompactStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#compactbackup">CompactBackup</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>state</code></br>
<em>
string
</em>
</td>
<td>
<p>State is the current state of the backup</p>
</td>
</tr>
<tr>
<td>
<code>progress</code></br>
<em>
string
</em>
</td>
<td>
<p>Progress is the detailed progress of a running backup</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>Message is the error message of the backup</p>
</td>
</tr>
<tr>
<td>
<code>endTs</code></br>
<em>
string
</em>
</td>
<td>
<p>endTs is the real endTs processed by the compact backup</p>
</td>
</tr>
<tr>
<td>
<code>backoffRetryStatus</code></br>
<em>
<a href="#compactretryrecord">
[]CompactRetryRecord
</a>
</em>
</td>
<td>
<p>RetryStatus is status of the backoff retry, it will be used when backup pod or job exited unexpectedly</p>
</td>
</tr>
</tbody>
</table>
<h3 id="componentaccessor">ComponentAccessor</h3>
<p>
<p>ComponentAccessor is the interface to access component details, which respects the cluster-level properties
and component-level overrides</p>
</p>
<h3 id="componentspec">ComponentSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#dmdiscoveryspec">DMDiscoverySpec</a>, 
<a href="#discoveryspec">DiscoverySpec</a>, 
<a href="#masterspec">MasterSpec</a>, 
<a href="#ngmonitoringspec">NGMonitoringSpec</a>, 
<a href="#pdmsspec">PDMSSpec</a>, 
<a href="#pdspec">PDSpec</a>, 
<a href="#pumpspec">PumpSpec</a>, 
<a href="#ticdcspec">TiCDCSpec</a>, 
<a href="#tidbspec">TiDBSpec</a>, 
<a href="#tiflashspec">TiFlashSpec</a>, 
<a href="#tikvspec">TiKVSpec</a>, 
<a href="#tiproxyspec">TiProxySpec</a>, 
<a href="#tidbdashboardspec">TidbDashboardSpec</a>, 
<a href="#tidbngmonitoringspec">TidbNGMonitoringSpec</a>, 
<a href="#workerspec">WorkerSpec</a>)
</p>
<p>
<p>ComponentSpec is the base spec of each component, the fields should always accessed by the Basic<Component>Spec() method to respect the cluster-level properties</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>(Deprecated) Image of the component
Use <code>baseImage</code> and <code>version</code> instead</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Version of the component. Override the cluster-level version if non-empty
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>hostNetwork</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether Hostnetwork of the component is enabled. Override the cluster-level setting if present
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of the component. Override the cluster-level setting if present.
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PriorityClassName of the component. Override the cluster-level one if present
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>schedulerName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SchedulerName of the component. Override the cluster-level one if present
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NodeSelector of the component. Merged into the cluster-level nodeSelector if non-empty
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations for the component. Merge into the cluster-level annotations if non-empty
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Labels for the component. Merge into the cluster-level labels if non-empty
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tolerations of the component. Override the cluster-level tolerations if non-empty
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>configUpdateStrategy</code></br>
<em>
<a href="#configupdatestrategy">
ConfigUpdateStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigUpdateStrategy of the component. Override the cluster-level updateStrategy if present
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following env names cannot be used and will be overridden by TiDB Operator builtin envs
- NAMESPACE
- TZ
- SERVICE_NAME
- PEER_SERVICE_NAME
- HEADLESS_SERVICE_NAME
- SET_NAME
- HOSTNAME
- CLUSTER_NAME
- POD_NAME
- BINLOG_ENABLED
- SLOW_LOG_FILE</p>
</td>
</tr>
<tr>
<td>
<code>envFrom</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core">
[]Kubernetes core/v1.EnvFromSource
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Extend the use scenarios for env</p>
</td>
</tr>
<tr>
<td>
<code>initContainers</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core">
[]Kubernetes core/v1.Container
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Init containers of the components</p>
</td>
</tr>
<tr>
<td>
<code>additionalContainers</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core">
[]Kubernetes core/v1.Container
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional containers of the component.
If the container names in this field match with the ones generated by
TiDB Operator, the container configurations will be merged into the
containers generated by TiDB Operator via strategic merge patch.
If the container names in this field do not match with the ones
generated by TiDB Operator, the container configurations will be
appended to the Pod container spec directly.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>dnsConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#poddnsconfig-v1-core">
Kubernetes core/v1.PodDNSConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSConfig Specifies the DNS parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>dnsPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#dnspolicy-v1-core">
Kubernetes core/v1.DNSPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSPolicy Specifies the DNSPolicy parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>terminationGracePeriodSeconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.
Value must be non-negative integer. The value zero indicates delete immediately.
If this value is nil, the default grace period will be used instead.
The grace period is the duration in seconds after the processes running in the pod are sent
a termination signal and the time when the processes are forcibly halted with a kill signal.
Set this value longer than the expected cleanup time for your process.
Defaults to 30 seconds.</p>
</td>
</tr>
<tr>
<td>
<code>statefulSetUpdateStrategy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetupdatestrategytype-v1-apps">
Kubernetes apps/v1.StatefulSetUpdateStrategyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StatefulSetUpdateStrategy indicates the StatefulSetUpdateStrategy that will be
employed to update Pods in the StatefulSet when a revision is made to
Template.</p>
</td>
</tr>
<tr>
<td>
<code>podManagementPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podmanagementpolicytype-v1-apps">
Kubernetes apps/v1.PodManagementPolicyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodManagementPolicy of TiDB cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code></br>
<em>
<a href="#topologyspreadconstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TopologySpreadConstraints describes how a group of pods ought to spread across topology
domains. Scheduler will schedule pods in a way which abides by the constraints.
This field is is only honored by clusters that enables the EvenPodsSpread feature.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>suspendAction</code></br>
<em>
<a href="#suspendaction">
SuspendAction
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SuspendAction defines the suspend actions for all component.</p>
</td>
</tr>
<tr>
<td>
<code>readinessProbe</code></br>
<em>
<a href="#probe">
Probe
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReadinessProbe describes actions that probe the components&rsquo; readiness.
the default behavior is like setting type as &ldquo;tcp&rdquo;</p>
</td>
</tr>
</tbody>
</table>
<h3 id="componentstatus">ComponentStatus</h3>
<p>
</p>
<h3 id="configmapref">ConfigMapRef</h3>
<p>
(<em>Appears on:</em>
<a href="#prometheusconfiguration">PrometheusConfiguration</a>)
</p>
<p>
<p>ConfigMapRef is the external configMap</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>if the namespace is omitted, the operator controller would use the Tidbmonitor&rsquo;s namespace instead.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="configupdatestrategy">ConfigUpdateStrategy</h3>
<p>
(<em>Appears on:</em>
<a href="#componentspec">ComponentSpec</a>, 
<a href="#dmclusterspec">DMClusterSpec</a>, 
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>ConfigUpdateStrategy represents the strategy to update configuration</p>
</p>
<h3 id="containername">ContainerName</h3>
<p>
</p>
<h3 id="coprocessorcache">CoprocessorCache</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvclient">TiKVClient</a>)
</p>
<p>
<p>CoprocessorCache is the config for coprocessor cache.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enable</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether to enable the copr cache. The copr cache saves the result from TiKV Coprocessor in the memory and
reuses the result when corresponding data in TiKV is unchanged, on a region basis.</p>
</td>
</tr>
<tr>
<td>
<code>capacity-mb</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The capacity in MB of the cache.</p>
</td>
</tr>
<tr>
<td>
<code>admission-max-result-mb</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Only cache requests whose result set is small.</p>
</td>
</tr>
<tr>
<td>
<code>admission-min-process-ms</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Only cache requests takes notable time to process.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="crdkind">CrdKind</h3>
<p>
(<em>Appears on:</em>
<a href="#crdkinds">CrdKinds</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Kind</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Plural</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>SpecName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ShortNames</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>AdditionalPrinterColums</code></br>
<em>
[]k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.CustomResourceColumnDefinition
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="crdkinds">CrdKinds</h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>KindsString</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>TiDBCluster</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>DMCluster</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Backup</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Restore</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>BackupSchedule</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>TiDBMonitor</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>TiDBInitializer</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>TiDBNGMonitoring</code></br>
<em>
<a href="#crdkind">
CrdKind
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="customizedprobe">CustomizedProbe</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>Image is the image of the probe binary.</p>
</td>
</tr>
<tr>
<td>
<code>binaryName</code></br>
<em>
string
</em>
</td>
<td>
<p>BinaryName is the name of the probe binary.</p>
</td>
</tr>
<tr>
<td>
<code>args</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Args is the arguments of the probe binary.</p>
</td>
</tr>
<tr>
<td>
<code>initialDelaySeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Number of seconds after the container has started before liveness probes are initiated.</p>
</td>
</tr>
<tr>
<td>
<code>timeoutSeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Number of seconds after which the probe times out.
Defaults to 1 second. Minimum value is 1.</p>
</td>
</tr>
<tr>
<td>
<code>periodSeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>How often (in seconds) to perform the probe.
Default to 10 seconds. Minimum value is 1.</p>
</td>
</tr>
<tr>
<td>
<code>successThreshold</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Minimum consecutive successes for the probe to be considered successful after having failed.
Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.</p>
</td>
</tr>
<tr>
<td>
<code>failureThreshold</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Minimum consecutive failures for the probe to be considered failed after having succeeded.
Defaults to 3. Minimum value is 1.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dmclustercondition">DMClusterCondition</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterstatus">DMClusterStatus</a>)
</p>
<p>
<p>DMClusterCondition is dm cluster condition</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#dmclusterconditiontype">
DMClusterConditionType
</a>
</em>
</td>
<td>
<p>Type of the condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>The last time this condition was updated.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dmclusterconditiontype">DMClusterConditionType</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclustercondition">DMClusterCondition</a>)
</p>
<p>
<p>DMClusterConditionType represents a dm cluster condition value.</p>
</p>
<h3 id="dmclusterspec">DMClusterSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#dmcluster">DMCluster</a>)
</p>
<p>
<p>DMClusterSpec describes the attributes that a user creates on a dm cluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>discovery</code></br>
<em>
<a href="#dmdiscoveryspec">
DMDiscoverySpec
</a>
</em>
</td>
<td>
<p>Discovery spec</p>
</td>
</tr>
<tr>
<td>
<code>master</code></br>
<em>
<a href="#masterspec">
MasterSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm-master cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>worker</code></br>
<em>
<a href="#workerspec">
WorkerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm-worker cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>paused</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Indicates that the dm cluster is paused and will not be processed by
the controller.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm cluster version</p>
</td>
</tr>
<tr>
<td>
<code>schedulerName</code></br>
<em>
string
</em>
</td>
<td>
<p>SchedulerName of DM cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by DM cluster</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of DM cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>configUpdateStrategy</code></br>
<em>
<a href="#configupdatestrategy">
ConfigUpdateStrategy
</a>
</em>
</td>
<td>
<p>ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
cluster component is needed to reload the configuration change.
UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
related components to use the new ConfigMap, that is, the new configuration will be applied automatically.</p>
</td>
</tr>
<tr>
<td>
<code>enablePVReclaim</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>tlsCluster</code></br>
<em>
<a href="#tlscluster">
TLSCluster
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable the TLS connection between DM server components
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretNames</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretNames are the names of secrets which stores mysql/tidb server client certificates
that used by dm-master and dm-worker.</p>
</td>
</tr>
<tr>
<td>
<code>hostNetwork</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether Hostnetwork is enabled for DM cluster Pods
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of DM cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PriorityClassName of DM cluster Pods
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base node selectors of DM cluster Pods, components may add or override selectors upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional annotations for the dm cluster
Can be overrode by annotations in master spec or worker spec</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional labels for the dm cluster
Can be overrode by labels in master spec or worker spec</p>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of DM cluster Pods
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of DM cluster Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>dnsConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#poddnsconfig-v1-core">
Kubernetes core/v1.PodDNSConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSConfig Specifies the DNS parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>dnsPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#dnspolicy-v1-core">
Kubernetes core/v1.DNSPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSPolicy Specifies the DNSPolicy parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>statefulSetUpdateStrategy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetupdatestrategytype-v1-apps">
Kubernetes apps/v1.StatefulSetUpdateStrategyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StatefulSetUpdateStrategy of DM cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>podManagementPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podmanagementpolicytype-v1-apps">
Kubernetes apps/v1.PodManagementPolicyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodManagementPolicy of DM cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code></br>
<em>
<a href="#topologyspreadconstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TopologySpreadConstraints describes how a group of pods ought to spread across topology
domains. Scheduler will schedule pods in a way which abides by the constraints.
This field is is only honored by clusters that enables the EvenPodsSpread feature.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>suspendAction</code></br>
<em>
<a href="#suspendaction">
SuspendAction
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SuspendAction defines the suspend actions for all component.</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dmclusterstatus">DMClusterStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#dmcluster">DMCluster</a>)
</p>
<p>
<p>DMClusterStatus represents the current status of a dm cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>master</code></br>
<em>
<a href="#masterstatus">
MasterStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>worker</code></br>
<em>
<a href="#workerstatus">
WorkerStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#dmclustercondition">
[]DMClusterCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a dm cluster&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dmdiscoveryspec">DMDiscoverySpec</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterspec">DMClusterSpec</a>)
</p>
<p>
<p>DMDiscoverySpec contains details of Discovery members for dm</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>livenessProbe</code></br>
<em>
<a href="#probe">
Probe
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LivenessProbe describes actions that probe the discovery&rsquo;s liveness.
the default behavior is like setting type as &ldquo;tcp&rdquo;
NOTE: only used for TiDB Operator discovery now,
for other components, the auto failover feature may be used instead.</p>
</td>
</tr>
<tr>
<td>
<code>address</code></br>
<em>
string
</em>
</td>
<td>
<p>(Deprecated) Address indicates the existed TiDB discovery address</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dmexperimental">DMExperimental</h3>
<p>
(<em>Appears on:</em>
<a href="#masterconfig">MasterConfig</a>)
</p>
<p>
<p>DM experimental config</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>openapi</code></br>
<em>
bool
</em>
</td>
<td>
<p>OpenAPI was introduced in DM V5.3.0</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dmmonitorspec">DMMonitorSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#clusterref">
[]ClusterRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>initializer</code></br>
<em>
<a href="#initializerspec">
InitializerSpec
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="dmsecurityconfig">DMSecurityConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#masterconfig">MasterConfig</a>, 
<a href="#workerconfig">WorkerConfig</a>)
</p>
<p>
<p>DM common security config</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ssl-ca</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SSLCA is the path of file that contains list of trusted SSL CAs.</p>
</td>
</tr>
<tr>
<td>
<code>ssl-cert</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SSLCert is the path of file that contains X509 certificate in PEM format.</p>
</td>
</tr>
<tr>
<td>
<code>ssl-key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SSLKey is the path of file that contains X509 key in PEM format.</p>
</td>
</tr>
<tr>
<td>
<code>cert-allowed-cn</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertAllowedCN is the Common Name that allowed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dashboardconfig">DashboardConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>DashboardConfig is the configuration for tidb-dashboard.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>tidb-cacert-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tidb-cert-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tidb-key-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>public-path-prefix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>internal-proxy</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>disable-telemetry</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>When not disabled, usage data will be sent to PingCAP for improving user experience.
Optional: Defaults to false
Deprecated in PD v4.0.3, use EnableTelemetry instead</p>
</td>
</tr>
<tr>
<td>
<code>enable-telemetry</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>When enabled, usage data will be sent to PingCAP for improving user experience.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>enable-experimental</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>When enabled, experimental TiDB Dashboard features will be available.
These features are incomplete or not well tested. Suggest not to enable in
production.
Optional: Defaults to false</p>
</td>
</tr>
</tbody>
</table>
<h3 id="deploymentstoragestatus">DeploymentStorageStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorstatus">TidbMonitorStatus</a>)
</p>
<p>
<p>DeploymentStorageStatus is the storage information of the deployment</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>pvName</code></br>
<em>
string
</em>
</td>
<td>
<p>PV name</p>
</td>
</tr>
</tbody>
</table>
<h3 id="discoveryspec">DiscoverySpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>DiscoverySpec contains details of Discovery members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>livenessProbe</code></br>
<em>
<a href="#probe">
Probe
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LivenessProbe describes actions that probe the discovery&rsquo;s liveness.
the default behavior is like setting type as &ldquo;tcp&rdquo;
NOTE: only used for TiDB Operator discovery now,
for other components, the auto failover feature may be used instead.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="dumplingconfig">DumplingConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>)
</p>
<p>
<p>DumplingConfig contains config for dumpling</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>options</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Options means options for backup data to remote storage with dumpling.</p>
</td>
</tr>
<tr>
<td>
<code>tableFilter</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Deprecated. Please use <code>Spec.TableFilter</code> instead. TableFilter means Table filter expression for &lsquo;db.table&rsquo; matching</p>
</td>
</tr>
</tbody>
</table>
<h3 id="emptystruct">EmptyStruct</h3>
<p>
(<em>Appears on:</em>
<a href="#pdfailuremember">PDFailureMember</a>, 
<a href="#tikvfailurestore">TiKVFailureStore</a>, 
<a href="#unjoinedmember">UnjoinedMember</a>)
</p>
<p>
<p>EmptyStruct is defined to delight controller-gen tools
Only named struct is allowed by controller-gen</p>
</p>
<h3 id="evictleaderstatus">EvictLeaderStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvstatus">TiKVStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podCreateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>beginTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="experimental">Experimental</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Experimental controls the features that are still experimental: their semantics, interfaces are subject to change.
Using these features in the production environment is not recommended.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>allow-auto-random</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable the syntax like <code>auto_random(3)</code> on the primary key column.
Imported from TiDB v3.1.0.
Deprecated in TiDB v4.0.3, please check detail in <a href="https://docs.pingcap.com/tidb/dev/release-4.0.3#improvements">https://docs.pingcap.com/tidb/dev/release-4.0.3#improvements</a>.</p>
</td>
</tr>
<tr>
<td>
<code>allow-expression-index</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable creating expression index.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="failover">Failover</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>, 
<a href="#tikvspec">TiKVSpec</a>, 
<a href="#workerspec">WorkerSpec</a>)
</p>
<p>
<p>Failover contains the failover specification.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>recoverByUID</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
<em>(Optional)</em>
<p>RecoverByUID indicates that TiDB Operator will recover the failover by this UID,
it takes effect only when set <code>spec.recoverFailover=false</code></p>
</td>
</tr>
</tbody>
</table>
<h3 id="federalvolumebackupphase">FederalVolumeBackupPhase</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>)
</p>
<p>
<p>FederalVolumeBackupPhase represents a phase to execute in federal volume backup</p>
</p>
<h3 id="federalvolumerestorephase">FederalVolumeRestorePhase</h3>
<p>
(<em>Appears on:</em>
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>FederalVolumeRestorePhase represents a phase to execute in federal volume restore</p>
</p>
<h3 id="filelogconfig">FileLogConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#log">Log</a>, 
<a href="#pdlogconfig">PDLogConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>filename</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log filename, leave empty to disable file log.</p>
</td>
</tr>
<tr>
<td>
<code>log-rotate</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0
Is log rotate enabled.</p>
</td>
</tr>
<tr>
<td>
<code>max-size</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Max size for a single file, in MB.</p>
</td>
</tr>
<tr>
<td>
<code>max-days</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Max log keep days, default is never deleting.</p>
</td>
</tr>
<tr>
<td>
<code>max-backups</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Maximum number of old log files to retain.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="flash">Flash</h3>
<p>
(<em>Appears on:</em>
<a href="#commonconfig">CommonConfig</a>)
</p>
<p>
<p>Flash is the configuration of [flash] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>overlap_threshold</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0.6</p>
</td>
</tr>
<tr>
<td>
<code>compact_log_min_period</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 200</p>
</td>
</tr>
<tr>
<td>
<code>flash_cluster</code></br>
<em>
<a href="#flashcluster">
FlashCluster
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>proxy</code></br>
<em>
<a href="#flashproxy">
FlashProxy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="flashcluster">FlashCluster</h3>
<p>
(<em>Appears on:</em>
<a href="#flash">Flash</a>)
</p>
<p>
<p>FlashCluster is the configuration of [flash.flash_cluster] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>log</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to /data0/logs/flash_cluster_manager.log</p>
</td>
</tr>
<tr>
<td>
<code>refresh_interval</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 20</p>
</td>
</tr>
<tr>
<td>
<code>update_rule_interval</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10</p>
</td>
</tr>
<tr>
<td>
<code>master_ttl</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 60</p>
</td>
</tr>
</tbody>
</table>
<h3 id="flashlogger">FlashLogger</h3>
<p>
(<em>Appears on:</em>
<a href="#commonconfig">CommonConfig</a>)
</p>
<p>
<p>FlashLogger is the configuration of [logger] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>errorlog</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to /data0/logs/error.log</p>
</td>
</tr>
<tr>
<td>
<code>size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 100M</p>
</td>
</tr>
<tr>
<td>
<code>log</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to /data0/logs/server.log</p>
</td>
</tr>
<tr>
<td>
<code>level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to information</p>
</td>
</tr>
<tr>
<td>
<code>count</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10</p>
</td>
</tr>
</tbody>
</table>
<h3 id="flashproxy">FlashProxy</h3>
<p>
(<em>Appears on:</em>
<a href="#flash">Flash</a>)
</p>
<p>
<p>FlashProxy is the configuration of [flash.proxy] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>addr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0.0.0.0:20170</p>
</td>
</tr>
<tr>
<td>
<code>advertise-addr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to {clusterName}-tiflash-POD_NUM.{clusterName}-tiflash-peer.{namespace}:20170</p>
</td>
</tr>
<tr>
<td>
<code>data-dir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to /data0/proxy</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to /data0/proxy.toml</p>
</td>
</tr>
<tr>
<td>
<code>log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to /data0/logs/proxy.log</p>
</td>
</tr>
</tbody>
</table>
<h3 id="flashsecurity">FlashSecurity</h3>
<p>
(<em>Appears on:</em>
<a href="#commonconfig">CommonConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ca_path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Be set automatically by Operator</p>
</td>
</tr>
<tr>
<td>
<code>cert_path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Be set automatically by Operator</p>
</td>
</tr>
<tr>
<td>
<code>key_path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Be set automatically by Operator</p>
</td>
</tr>
<tr>
<td>
<code>cert_allowed_cn</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertAllowedCN is the Common Name that allowed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="flashserverconfig">FlashServerConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>)
</p>
<p>
<p>FlashServerConfig is the configuration of Proxy server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>engine-addr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default to {clusterName}-tiflash-POD_NUM.{clusterName}-tiflash-peer.{namespace}:3930</p>
</td>
</tr>
<tr>
<td>
<code>status-addr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default to 0.0.0.0:20292</p>
</td>
</tr>
<tr>
<td>
<code>advertise-status-addr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default to {clusterName}-tiflash-POD_NUM.{clusterName}-tiflash-peer.{namespace}:20292</p>
</td>
</tr>
<tr>
<td>
<code>TiKVServerConfig</code></br>
<em>
<a href="#tikvserverconfig">
TiKVServerConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>TiKVServerConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="gcsstorageprovider">GcsStorageProvider</h3>
<p>
(<em>Appears on:</em>
<a href="#storageprovider">StorageProvider</a>)
</p>
<p>
<p>GcsStorageProvider represents the google cloud storage for storing backups.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectId represents the project that organizes all your Google Cloud Platform resources</p>
</td>
</tr>
<tr>
<td>
<code>location</code></br>
<em>
string
</em>
</td>
<td>
<p>Location in which the gcs bucket is located.</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path is the full path where the backup is saved.
The format of the path must be: &ldquo;<bucket-name>/<path-to-backup-file>&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>bucket</code></br>
<em>
string
</em>
</td>
<td>
<p>Bucket in which to store the backup data.</p>
</td>
</tr>
<tr>
<td>
<code>storageClass</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageClass represents the storage class</p>
</td>
</tr>
<tr>
<td>
<code>objectAcl</code></br>
<em>
string
</em>
</td>
<td>
<p>ObjectAcl represents the access control list for new objects</p>
</td>
</tr>
<tr>
<td>
<code>bucketAcl</code></br>
<em>
string
</em>
</td>
<td>
<p>BucketAcl represents the access control list for new buckets</p>
</td>
</tr>
<tr>
<td>
<code>secretName</code></br>
<em>
string
</em>
</td>
<td>
<p>SecretName is the name of secret which stores the
gcs service account credentials JSON.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<p>Prefix of the data path.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="grafanaspec">GrafanaSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
<p>GrafanaSpec is the desired state of grafana</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>MonitorContainer</code></br>
<em>
<a href="#monitorcontainer">
MonitorContainer
</a>
</em>
</td>
<td>
<p>
(Members of <code>MonitorContainer</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>logLevel</code></br>
<em>
string
</em>
</td>
<td>
<p>Grafana log level</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<p>Service defines a Kubernetes service of Grafana.</p>
</td>
</tr>
<tr>
<td>
<code>usernameSecret</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>if <code>UsernameSecret</code> is not set, <code>username</code> will be used.</p>
</td>
</tr>
<tr>
<td>
<code>passwordSecret</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>if <code>passwordSecret</code> is not set, <code>password</code> will be used.</p>
</td>
</tr>
<tr>
<td>
<code>username</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v1.3.0 for security concerns, planned for removal in v1.4.0. Use <code>usernameSecret</code> instead.</p>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v1.3.0 for security concerns, planned for removal in v1.4.0. Use <code>passwordSecret</code> instead.</p>
</td>
</tr>
<tr>
<td>
<code>envs</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ingress</code></br>
<em>
<a href="#ingressspec">
IngressSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ingress configuration of Prometheus</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<p>Additional volume mounts of grafana pod.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="helperspec">HelperSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>HelperSpec contains details of helper component</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Image used to tail slow log and set kernel parameters if necessary, must have <code>tail</code> and <code>sysctl</code> installed
Optional: Defaults to busybox:1.26.2. Recommended to set to 1.34.1 for new installations.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
Optional: Defaults to the cluster-level setting</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ingressspec">IngressSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#grafanaspec">GrafanaSpec</a>, 
<a href="#prometheusspec">PrometheusSpec</a>)
</p>
<p>
<p>IngressSpec describe the ingress desired state for the target component</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>hosts</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Hosts describe the hosts for the ingress</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations describe the desired annotations for the ingress</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#ingresstls-v1-networking">
[]Kubernetes networking/v1.IngressTLS
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLS configuration. Currently the Ingress only supports a single TLS
port, 443. If multiple members of this list specify different hosts, they
will be multiplexed on the same port according to the hostname specified
through the SNI TLS extension, if the ingress controller fulfilling the
ingress supports SNI.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="initcontainerspec">InitContainerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>)
</p>
<p>
<p>InitContainerSpec contains basic spec about a init container</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="initializephase">InitializePhase</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbinitializerstatus">TidbInitializerStatus</a>)
</p>
<p>
</p>
<h3 id="initializerspec">InitializerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#dmmonitorspec">DMMonitorSpec</a>, 
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
<p>InitializerSpec is the desired state of initializer</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>MonitorContainer</code></br>
<em>
<a href="#monitorcontainer">
MonitorContainer
</a>
</em>
</td>
<td>
<p>
(Members of <code>MonitorContainer</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>envs</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="interval">Interval</h3>
<p>
(<em>Appears on:</em>
<a href="#quota">Quota</a>)
</p>
<p>
<p>Interval is the configuration of [quotas.default.interval] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>duration</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3600</p>
</td>
</tr>
<tr>
<td>
<code>queries</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>errors</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>result_rows</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>read_rows</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>execution_time</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
</tbody>
</table>
<h3 id="isolationread">IsolationRead</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>IsolationRead is the config for isolation read.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>engines</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Engines filters tidb-server access paths by engine type.
imported from v3.1.0</p>
</td>
</tr>
</tbody>
</table>
<h3 id="localstorageprovider">LocalStorageProvider</h3>
<p>
(<em>Appears on:</em>
<a href="#storageprovider">StorageProvider</a>)
</p>
<p>
<p>LocalStorageProvider defines local storage options, which can be any k8s supported mounted volume</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>volume</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumeMount</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="log">Log</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Log is the log section of config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log level.
Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log format. one of json, text, or console.
Optional: Defaults to text</p>
</td>
</tr>
<tr>
<td>
<code>disable-timestamp</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v3.0.5. Use EnableTimestamp instead
Disable automatic timestamps in output.</p>
</td>
</tr>
<tr>
<td>
<code>enable-timestamp</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableTimestamp enables automatic timestamps in log output.</p>
</td>
</tr>
<tr>
<td>
<code>enable-error-stack</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableErrorStack enables annotating logs with the full stack error
message.</p>
</td>
</tr>
<tr>
<td>
<code>file</code></br>
<em>
<a href="#filelogconfig">
FileLogConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>File log config.</p>
</td>
</tr>
<tr>
<td>
<code>enable-slow-log</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>slow-query-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>slow-threshold</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 300</p>
</td>
</tr>
<tr>
<td>
<code>expensive-threshold</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10000</p>
</td>
</tr>
<tr>
<td>
<code>query-log-max-len</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2048</p>
</td>
</tr>
<tr>
<td>
<code>record-plan-in-slow-log</code></br>
<em>
uint32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 1</p>
</td>
</tr>
</tbody>
</table>
<h3 id="logsubcommandstatus">LogSubCommandStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#backupstatus">BackupStatus</a>)
</p>
<p>
<p>LogSubCommandStatus is the log backup subcommand&rsquo;s status.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>command</code></br>
<em>
<a href="#logsubcommandtype">
LogSubCommandType
</a>
</em>
</td>
<td>
<p>Command is the log backup subcommand.</p>
</td>
</tr>
<tr>
<td>
<code>timeStarted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeStarted is the time at which the command was started.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
<tr>
<td>
<code>timeCompleted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeCompleted is the time at which the command was completed.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
<tr>
<td>
<code>logTruncatingUntil</code></br>
<em>
string
</em>
</td>
<td>
<p>LogTruncatingUntil is log backup truncate until timestamp which is used to mark the truncate command.</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#backupconditiontype">
BackupConditionType
</a>
</em>
</td>
<td>
<p>Phase is the command current phase.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#backupcondition">
[]BackupCondition
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="logsubcommandtype">LogSubCommandType</h3>
<p>
(<em>Appears on:</em>
<a href="#backupcondition">BackupCondition</a>, 
<a href="#backupspec">BackupSpec</a>, 
<a href="#logsubcommandstatus">LogSubCommandStatus</a>)
</p>
<p>
<p>LogSubCommandType is the log backup subcommand type.</p>
</p>
<h3 id="logtailerspec">LogTailerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>, 
<a href="#tikvspec">TiKVSpec</a>)
</p>
<p>
<p>LogTailerSpec represents an optional log tailer sidecar container</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>useSidecar</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>If true, we use native sidecar feature to tail log
It requires enable feature gate &ldquo;SidecarContainers&rdquo;
This feature is introduced at 1.28, default enabled at 1.29, and GA at 1.33
See <a href="https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/">https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/</a>
and <a href="https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/">https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterconfig">MasterConfig</h3>
<p>
<p>MasterConfig is the configuration of dm-master-server</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>log-level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log level.
Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>File log config.</p>
</td>
</tr>
<tr>
<td>
<code>log-format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log format. one of json or text.</p>
</td>
</tr>
<tr>
<td>
<code>rpc-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>RPC timeout when dm-master request to dm-worker
Optional: Defaults to 30s</p>
</td>
</tr>
<tr>
<td>
<code>rpc-rate-limit</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>RPC agent rate limit when dm-master request to dm-worker
Optional: Defaults to 10</p>
</td>
</tr>
<tr>
<td>
<code>rpc-rate-burst</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>RPC agent rate burst when dm-master request to dm-worker
Optional: Defaults to 40</p>
</td>
</tr>
<tr>
<td>
<code>DMSecurityConfig</code></br>
<em>
<a href="#dmsecurityconfig">
DMSecurityConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>DMSecurityConfig</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>dm-master&rsquo;s security config</p>
</td>
</tr>
<tr>
<td>
<code>experimental</code></br>
<em>
<a href="#dmexperimental">
DMExperimental
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>dm-master&rsquo;s experimental config</p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterconfigwraper">MasterConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#masterspec">MasterSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterfailuremember">MasterFailureMember</h3>
<p>
(<em>Appears on:</em>
<a href="#masterstatus">MasterStatus</a>)
</p>
<p>
<p>MasterFailureMember is the dm-master failure member information</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>memberID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pvcUID</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>memberDeleted</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="masterkeyfileconfig">MasterKeyFileConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvmasterkeyconfig">TiKVMasterKeyConfig</a>, 
<a href="#tikvsecurityconfigencryptionmasterkey">TiKVSecurityConfigEncryptionMasterKey</a>, 
<a href="#tikvsecurityconfigencryptionpreviousmasterkey">TiKVSecurityConfigEncryptionPreviousMasterKey</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>method</code></br>
<em>
string
</em>
</td>
<td>
<p>Encrypyion method, use master key encryption data key
Possible values: plaintext, aes128-ctr, aes192-ctr, aes256-ctr
Optional: Default to plaintext
optional</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Text file containing the key in hex form, end with &lsquo;\n&rsquo;</p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterkeykmsconfig">MasterKeyKMSConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvmasterkeyconfig">TiKVMasterKeyConfig</a>, 
<a href="#tikvsecurityconfigencryptionmasterkey">TiKVSecurityConfigEncryptionMasterKey</a>, 
<a href="#tikvsecurityconfigencryptionpreviousmasterkey">TiKVSecurityConfigEncryptionPreviousMasterKey</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key-id</code></br>
<em>
string
</em>
</td>
<td>
<p>AWS CMK key-id it can be find in AWS Console or use aws cli
This field is required</p>
</td>
</tr>
<tr>
<td>
<code>access-key</code></br>
<em>
string
</em>
</td>
<td>
<p>AccessKey of AWS user, leave empty if using other authrization method
optional</p>
</td>
</tr>
<tr>
<td>
<code>secret-access-key</code></br>
<em>
string
</em>
</td>
<td>
<p>SecretKey of AWS user, leave empty if using other authrization method
optional</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region of this KMS key
Optional: Default to us-east-1
optional</p>
</td>
</tr>
<tr>
<td>
<code>endpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>Used for KMS compatible KMS, such as Ceph, minio, If use AWS, leave empty
optional</p>
</td>
</tr>
</tbody>
</table>
<h3 id="mastermember">MasterMember</h3>
<p>
(<em>Appears on:</em>
<a href="#masterstatus">MasterStatus</a>)
</p>
<p>
<p>MasterMember is dm-master member status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>member id is actually a uint64, but apimachinery&rsquo;s json only treats numbers as int64/float64
so uint64 may overflow int64 and thus convert to float64</p>
</td>
</tr>
<tr>
<td>
<code>clientURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>health</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the health transitioned from one to another.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterservicespec">MasterServiceSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#masterspec">MasterSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ServiceSpec</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ServiceSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>externalTrafficPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#serviceexternaltrafficpolicy-v1-core">
Kubernetes core/v1.ServiceExternalTrafficPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalTrafficPolicy of the service
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>masterNodePort</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterspec">MasterSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterspec">DMClusterSpec</a>)
</p>
<p>
<p>MasterSpec contains details of dm-master members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#masterservicespec">
MasterServiceSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service defines a Kubernetes service of Master cluster.
Optional: Defaults to <code>.spec.services</code> in favor of backward compatibility</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for dm-master data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageSize is the request storage size for dm-master.
Defaults to &ldquo;10Gi&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>dataSubDir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subdirectory within the volume to store dm-master Data. By default, the data
is stored in the root directory of volume which is mounted at
/var/lib/dm-master.
Specifying this will change the data directory to a subdirectory, e.g.
/var/lib/dm-master/data if you set the value to &ldquo;data&rdquo;.
It&rsquo;s dangerous to change this value for a running cluster as it will
upgrade your cluster to use a new storage directory.
Defaults to &ldquo;&rdquo; (volume&rsquo;s root).</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#masterconfigwraper">
MasterConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of dm-master-servers</p>
</td>
</tr>
<tr>
<td>
<code>startUpScriptVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Start up script version</p>
</td>
</tr>
</tbody>
</table>
<h3 id="masterstatus">MasterStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterstatus">DMClusterStatus</a>)
</p>
<p>
<p>MasterStatus is dm-master status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
<a href="#mastermember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.MasterMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>leader</code></br>
<em>
<a href="#mastermember">
MasterMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureMembers</code></br>
<em>
<a href="#masterfailuremember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.MasterFailureMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>unjoinedMembers</code></br>
<em>
<a href="#unjoinedmember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.UnjoinedMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="memberphase">MemberPhase</h3>
<p>
(<em>Appears on:</em>
<a href="#masterstatus">MasterStatus</a>, 
<a href="#ngmonitoringstatus">NGMonitoringStatus</a>, 
<a href="#pdmsstatus">PDMSStatus</a>, 
<a href="#pdstatus">PDStatus</a>, 
<a href="#pumpstatus">PumpStatus</a>, 
<a href="#ticdcstatus">TiCDCStatus</a>, 
<a href="#tidbstatus">TiDBStatus</a>, 
<a href="#tikvstatus">TiKVStatus</a>, 
<a href="#tiproxystatus">TiProxyStatus</a>, 
<a href="#tidbdashboardstatus">TidbDashboardStatus</a>, 
<a href="#workerstatus">WorkerStatus</a>)
</p>
<p>
<p>MemberPhase is the current state of member</p>
</p>
<h3 id="membertype">MemberType</h3>
<p>
<p>MemberType represents member type</p>
</p>
<h3 id="metadataconfig">MetadataConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#remotewritespec">RemoteWriteSpec</a>)
</p>
<p>
<p>Configures the sending of series metadata to remote storage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>send</code></br>
<em>
bool
</em>
</td>
<td>
<p>Whether metric metadata is sent to remote storage or not.</p>
</td>
</tr>
<tr>
<td>
<code>sendInterval</code></br>
<em>
string
</em>
</td>
<td>
<p>How frequently metric metadata is sent to remote storage.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="monitorcomponentaccessor">MonitorComponentAccessor</h3>
<p>
</p>
<h3 id="monitorcontainer">MonitorContainer</h3>
<p>
(<em>Appears on:</em>
<a href="#grafanaspec">GrafanaSpec</a>, 
<a href="#initializerspec">InitializerSpec</a>, 
<a href="#prometheusreloaderspec">PrometheusReloaderSpec</a>, 
<a href="#prometheusspec">PrometheusSpec</a>, 
<a href="#reloaderspec">ReloaderSpec</a>, 
<a href="#thanosspec">ThanosSpec</a>)
</p>
<p>
<p>MonitorContainer is the common attributes of the container of monitoring</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="ngmonitoringspec">NGMonitoringSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbngmonitoringspec">TidbNGMonitoringSpec</a>)
</p>
<p>
<p>NGMonitoringSpec is spec of ng monitoring</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageClassName is the persistent volume for ng monitoring.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<p>StorageVolumes configures additional storage for NG Monitoring pods.</p>
</td>
</tr>
<tr>
<td>
<code>retentionPeriod</code></br>
<em>
string
</em>
</td>
<td>
<p>Retention period to store ng monitoring data</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>Config is the configuration of ng monitoring</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ngmonitoringstatus">NGMonitoringStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbngmonitoringstatus">TidbNGMonitoringStatus</a>)
</p>
<p>
<p>NGMonitoringStatus is latest status of ng monitoring</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="networks">Networks</h3>
<p>
(<em>Appears on:</em>
<a href="#user">User</a>)
</p>
<p>
<p>Networks is the configuration of [users.readonly.networks] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ip</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="observedstoragevolumestatus">ObservedStorageVolumeStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#storagevolumestatus">StorageVolumeStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>boundCount</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>BoundCount is the count of bound volumes.</p>
</td>
</tr>
<tr>
<td>
<code>currentCount</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>CurrentCount is the count of volumes whose capacity is equal to <code>currentCapacity</code>.</p>
</td>
</tr>
<tr>
<td>
<code>modifiedCount</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>ModifiedCount is the count of modified volumes.</p>
</td>
</tr>
<tr>
<td>
<code>currentCapacity</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<em>(Optional)</em>
<p>CurrentCapacity is the current capacity of the volume.
If any volume is resizing, it is the capacity before resizing.
If all volumes are resized, it is the resized capacity and same as desired capacity.</p>
</td>
</tr>
<tr>
<td>
<code>modifiedCapacity</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<em>(Optional)</em>
<p>ModifiedCapacity is the modified capacity of the volume.</p>
</td>
</tr>
<tr>
<td>
<code>currentStorageClass</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CurrentStorageClass is the modified capacity of the volume.</p>
</td>
</tr>
<tr>
<td>
<code>modifiedStorageClass</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ModifiedStorageClass is the modified storage calss of the volume.</p>
</td>
</tr>
<tr>
<td>
<code>resizedCapacity</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<em>(Optional)</em>
<p>(Deprecated) ResizedCapacity is the desired capacity of the volume.</p>
</td>
</tr>
<tr>
<td>
<code>resizedCount</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>(Deprecated) ResizedCount is the count of volumes whose capacity is equal to <code>resizedCapacity</code>.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="opentracing">OpenTracing</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>OpenTracing is the opentracing section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enable</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>sampler</code></br>
<em>
<a href="#opentracingsampler">
OpenTracingSampler
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>reporter</code></br>
<em>
<a href="#opentracingreporter">
OpenTracingReporter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>rpc-metrics</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="opentracingreporter">OpenTracingReporter</h3>
<p>
(<em>Appears on:</em>
<a href="#opentracing">OpenTracing</a>)
</p>
<p>
<p>OpenTracingReporter is the config for opentracing reporter.
See <a href="https://godoc.org/github.com/uber/jaeger-client-go/config#ReporterConfig">https://godoc.org/github.com/uber/jaeger-client-go/config#ReporterConfig</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>queue-size</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>buffer-flush-interval</code></br>
<em>
time.Duration
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>log-spans</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>local-agent-host-port</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="opentracingsampler">OpenTracingSampler</h3>
<p>
(<em>Appears on:</em>
<a href="#opentracing">OpenTracing</a>)
</p>
<p>
<p>OpenTracingSampler is the config for opentracing sampler.
See <a href="https://godoc.org/github.com/uber/jaeger-client-go/config#SamplerConfig">https://godoc.org/github.com/uber/jaeger-client-go/config#SamplerConfig</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>param</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>sampling-server-url</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-operations</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>sampling-refresh-interval</code></br>
<em>
time.Duration
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="pdconfig">PDConfig</h3>
<p>
<p>PDConfig is the configuration of pd-server</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>force-new-cluster</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-grpc-gateway</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>initial-cluster-token</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>set different tokens to prevent communication between PDs in different clusters.</p>
</td>
</tr>
<tr>
<td>
<code>lease</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>LeaderLease time, if leader doesn&rsquo;t update its TTL
in etcd after lease time, etcd will expire the leader key
and other servers can campaign the leader again.
Etcd only supports seconds TTL, so here is second too.
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>log</code></br>
<em>
<a href="#pdlogconfig">
PDLogConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log related config.</p>
</td>
</tr>
<tr>
<td>
<code>log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Backward compatibility.</p>
</td>
</tr>
<tr>
<td>
<code>log-level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tso-save-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TsoSaveInterval is the interval to save timestamp.
Optional: Defaults to 3s</p>
</td>
</tr>
<tr>
<td>
<code>metric</code></br>
<em>
<a href="#pdmetricconfig">
PDMetricConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>schedule</code></br>
<em>
<a href="#pdscheduleconfig">
PDScheduleConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>replication</code></br>
<em>
<a href="#pdreplicationconfig">
PDReplicationConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
<a href="#pdnamespaceconfig">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.PDNamespaceConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pd-server</code></br>
<em>
<a href="#pdserverconfig">
PDServerConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cluster-version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>quota-backend-bytes</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>QuotaBackendBytes Raise alarms when backend size exceeds the given quota. 0 means use the default quota.
the default size is 2GB, the maximum is 8GB.</p>
</td>
</tr>
<tr>
<td>
<code>auto-compaction-mode</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AutoCompactionMode is either &lsquo;periodic&rsquo; or &lsquo;revision&rsquo;. The default value is &lsquo;periodic&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>auto-compaction-retention-v2</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AutoCompactionRetention is either duration string with time unit
(e.g. &lsquo;5m&rsquo; for 5-minute), or revision unit (e.g. &lsquo;5000&rsquo;).
If no time unit is provided and compaction mode is &lsquo;periodic&rsquo;,
the unit defaults to hour. For example, &lsquo;5&rsquo; translates into 5-hour.
The default retention is 1 hour.
Before etcd v3.3.x, the type of retention is int. We add &lsquo;v2&rsquo; suffix to make it backward compatible.</p>
</td>
</tr>
<tr>
<td>
<code>tikv-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TickInterval is the interval for etcd Raft tick.</p>
</td>
</tr>
<tr>
<td>
<code>election-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ElectionInterval is the interval for etcd Raft election.</p>
</td>
</tr>
<tr>
<td>
<code>enable-prevote</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Prevote is true to enable Raft Pre-Vote.
If enabled, Raft runs an additional election phase
to check whether it would get enough votes to win
an election, thus minimizing disruptions.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>security</code></br>
<em>
<a href="#pdsecurityconfig">
PDSecurityConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>label-property</code></br>
<em>
<a href="#pdlabelpropertyconfig">
PDLabelPropertyConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>namespace-classifier</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0
NamespaceClassifier is for classifying stores/regions into different
namespaces.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>dashboard</code></br>
<em>
<a href="#dashboardconfig">
DashboardConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="pdconfigwraper">PDConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#pdmsspec">PDMSSpec</a>, 
<a href="#pdspec">PDSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdfailuremember">PDFailureMember</h3>
<p>
(<em>Appears on:</em>
<a href="#pdstatus">PDStatus</a>)
</p>
<p>
<p>PDFailureMember is the pd failure member information</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>memberID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pvcUID</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pvcUIDSet</code></br>
<em>
<a href="#emptystruct">
map[k8s.io/apimachinery/pkg/types.UID]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.EmptyStruct
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>memberDeleted</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>hostDown</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="pdlabelpropertyconfig">PDLabelPropertyConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
</p>
<h3 id="pdlogconfig">PDLogConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>PDLogConfig serializes log related config in toml/json.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log level.
Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log format. one of json, text, or console.</p>
</td>
</tr>
<tr>
<td>
<code>disable-timestamp</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Disable automatic timestamps in output.</p>
</td>
</tr>
<tr>
<td>
<code>file</code></br>
<em>
<a href="#filelogconfig">
FileLogConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>File log config.</p>
</td>
</tr>
<tr>
<td>
<code>development</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Development puts the logger in development mode, which changes the
behavior of DPanicLevel and takes stacktraces more liberally.</p>
</td>
</tr>
<tr>
<td>
<code>disable-caller</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableCaller stops annotating logs with the calling function&rsquo;s file
name and line number. By default, all logs are annotated.</p>
</td>
</tr>
<tr>
<td>
<code>disable-stacktrace</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableStacktrace completely disables automatic stacktrace capturing. By
default, stacktraces are captured for WarnLevel and above logs in
development and ErrorLevel and above in production.</p>
</td>
</tr>
<tr>
<td>
<code>disable-error-verbose</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableErrorVerbose stops annotating logs with the full verbose error
message.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdmsspec">PDMSSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>PDMSSpec contains details of PD microservice</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the PD microservice</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for pd ms</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service defines a Kubernetes service of PD microservice cluster.
Optional: Defaults to <code>.spec.services</code> in favor of backward compatibility</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#pdconfigwraper">
PDConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the configuration of PD microservice servers</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretName is the name of secret which stores tidb server client certificate
which used by Dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>mountClusterClientSecret</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>MountClusterClientSecret indicates whether to mount <code>cluster-client-secret</code> to the Pod</p>
</td>
</tr>
<tr>
<td>
<code>startUpScriptVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Start up script version</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for PD microservice log storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageVolumes configure additional storage for PD microservice pods.</p>
</td>
</tr>
<tr>
<td>
<code>startTimeout</code></br>
<em>
int
</em>
</td>
<td>
<p>Timeout threshold when pd get started</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdmsstatus">PDMSStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>PDMSStatus is PD microservice status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Members contains other service in current TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdmember">PDMember</h3>
<p>
(<em>Appears on:</em>
<a href="#pdstatus">PDStatus</a>)
</p>
<p>
<p>PDMember is PD member</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>member id is actually a uint64, but apimachinery&rsquo;s json only treats numbers as int64/float64
so uint64 may overflow int64 and thus convert to float64</p>
</td>
</tr>
<tr>
<td>
<code>clientURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>health</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the health transitioned from one to another.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdmetricconfig">PDMetricConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>job</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>address</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="pdnamespaceconfig">PDNamespaceConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>PDNamespaceConfig is to overwrite the global setting for specific namespace</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>leader-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>LeaderScheduleLimit is the max coexist leader schedules.</p>
</td>
</tr>
<tr>
<td>
<code>region-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegionScheduleLimit is the max coexist region schedules.</p>
</td>
</tr>
<tr>
<td>
<code>replica-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReplicaScheduleLimit is the max coexist replica schedules.</p>
</td>
</tr>
<tr>
<td>
<code>merge-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>MergeScheduleLimit is the max coexist merge schedules.</p>
</td>
</tr>
<tr>
<td>
<code>hot-region-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>HotRegionScheduleLimit is the max coexist hot region schedules.</p>
</td>
</tr>
<tr>
<td>
<code>max-replicas</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxReplicas is the number of replicas for each region.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdreplicationconfig">PDReplicationConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>PDReplicationConfig is the replication configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>max-replicas</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxReplicas is the number of replicas for each region.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>location-labels</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The label keys specified the location of a store.
The placement priorities is implied by the order of label keys.
For example, [&ldquo;zone&rdquo;, &ldquo;rack&rdquo;] means that we should place replicas to
different zones first, then to different racks if we don&rsquo;t have enough zones.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>strictly-match-label</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>StrictlyMatchLabel strictly checks if the label of TiKV is matched with LocaltionLabels.
Immutable, change should be made through pd-ctl after cluster creation.
Imported from v3.1.0</p>
</td>
</tr>
<tr>
<td>
<code>enable-placement-rules</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>When PlacementRules feature is enabled. MaxReplicas and LocationLabels are not used anymore.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdscheduleconfig">PDScheduleConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>ScheduleConfig is the schedule configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>max-snapshot-count</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>If the snapshot count of one store is greater than this value,
it will never be used as a source or target store.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>max-pending-peer-count</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 16</p>
</td>
</tr>
<tr>
<td>
<code>max-merge-region-size</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>If both the size of region is smaller than MaxMergeRegionSize
and the number of rows in region is smaller than MaxMergeRegionKeys,
it will try to merge with adjacent regions.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 20</p>
</td>
</tr>
<tr>
<td>
<code>max-merge-region-keys</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 200000</p>
</td>
</tr>
<tr>
<td>
<code>split-merge-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SplitMergeInterval is the minimum interval time to permit merge after split.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 1h</p>
</td>
</tr>
<tr>
<td>
<code>patrol-region-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PatrolRegionInterval is the interval for scanning region during patrol.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>max-store-down-time</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxStoreDownTime is the max duration after which
a store will be considered to be down if it hasn&rsquo;t reported heartbeats.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 30m</p>
</td>
</tr>
<tr>
<td>
<code>leader-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>LeaderScheduleLimit is the max coexist leader schedules.
Immutable, change should be made through pd-ctl after cluster creation.
Optional: Defaults to 4.
Imported from v3.1.0</p>
</td>
</tr>
<tr>
<td>
<code>region-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegionScheduleLimit is the max coexist region schedules.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 2048</p>
</td>
</tr>
<tr>
<td>
<code>replica-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReplicaScheduleLimit is the max coexist replica schedules.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 64</p>
</td>
</tr>
<tr>
<td>
<code>merge-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>MergeScheduleLimit is the max coexist merge schedules.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 8</p>
</td>
</tr>
<tr>
<td>
<code>hot-region-schedule-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>HotRegionScheduleLimit is the max coexist hot region schedules.
Immutable, change should be made through pd-ctl after cluster creation
Optional: Defaults to 4</p>
</td>
</tr>
<tr>
<td>
<code>hot-region-cache-hits-threshold</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
If the number of times a region hits the hot cache is greater than this
threshold, it is considered a hot region.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>tolerant-size-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>TolerantSizeRatio is the ratio of buffer size for balance scheduler.
Immutable, change should be made through pd-ctl after cluster creation.
Imported from v3.1.0</p>
</td>
</tr>
<tr>
<td>
<code>low-space-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<pre><code> high space stage         transition stage           low space stage
</code></pre>
<p>|&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&ndash;|&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&ndash;|&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;&mdash;-|
^                    ^                             ^                         ^
0       HighSpaceRatio * capacity       LowSpaceRatio * capacity          capacity</p>
<p>LowSpaceRatio is the lowest usage ratio of store which regraded as low space.
When in low space, store region score increases to very large and varies inversely with available size.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>high-space-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>HighSpaceRatio is the highest usage ratio of store which regraded as high space.
High space means there is a lot of spare capacity, and store region score varies directly with used size.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-raft-learner</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableLearner is the option to disable using AddLearnerNode instead of AddNode
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-remove-down-replica</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableRemoveDownReplica is the option to prevent replica checker from
removing down replicas.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-replace-offline-replica</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableReplaceOfflineReplica is the option to prevent replica checker from
repalcing offline replicas.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-make-up-replica</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableMakeUpReplica is the option to prevent replica checker from making up
replicas when replica count is less than expected.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-remove-extra-replica</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableRemoveExtraReplica is the option to prevent replica checker from
removing extra replicas.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-location-replacement</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableLocationReplacement is the option to prevent replica checker from
moving replica to a better location.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable-namespace-relocation</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableNamespaceRelocation is the option to prevent namespace checker
from moving replica to the target namespace.
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>schedulers-v2</code></br>
<em>
<a href="#pdschedulerconfigs">
PDSchedulerConfigs
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Schedulers support for loding customized schedulers
Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>schedulers-payload</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Only used to display</p>
</td>
</tr>
<tr>
<td>
<code>enable-one-way-merge</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableOneWayMerge is the option to enable one way merge. This means a Region can only be merged into the next region of it.
Imported from v3.1.0</p>
</td>
</tr>
<tr>
<td>
<code>enable-cross-table-merge</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableCrossTableMerge is the option to enable cross table merge. This means two Regions can be merged with different table IDs.
This option only works when key type is &ldquo;table&rdquo;.
Imported from v3.1.0</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdschedulerconfig">PDSchedulerConfig</h3>
<p>
<p>PDSchedulerConfig is customized scheduler configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>args</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
<tr>
<td>
<code>disable</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, change should be made through pd-ctl after cluster creation</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdschedulerconfigs">PDSchedulerConfigs</h3>
<p>
(<em>Appears on:</em>
<a href="#pdscheduleconfig">PDScheduleConfig</a>)
</p>
<p>
</p>
<h3 id="pdsecurityconfig">PDSecurityConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>PDSecurityConfig is the configuration for supporting tls.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>cacert-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CAPath is the path of file that contains list of trusted SSL CAs.</p>
</td>
</tr>
<tr>
<td>
<code>cert-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertPath is the path of file that contains X509 certificate in PEM format.</p>
</td>
</tr>
<tr>
<td>
<code>key-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeyPath is the path of file that contains X509 key in PEM format.</p>
</td>
</tr>
<tr>
<td>
<code>cert-allowed-cn</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertAllowedCN is the Common Name that allowed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdserverconfig">PDServerConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#pdconfig">PDConfig</a>)
</p>
<p>
<p>PDServerConfig is the configuration for pd server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>use-region-storage</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>UseRegionStorage enables the independent region storage.</p>
</td>
</tr>
<tr>
<td>
<code>metric-storage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>MetricStorage is the cluster metric storage.
Currently we use prometheus as metric storage, we may use PD/TiKV as metric storage later.
Imported from v3.1.0</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdspec">PDSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>PDSpec contains details of PD members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for pd</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service defines a Kubernetes service of PD cluster.
Optional: Defaults to <code>.spec.services</code> in favor of backward compatibility</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for PD data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageVolumes configure additional storage for PD pods.</p>
</td>
</tr>
<tr>
<td>
<code>dataSubDir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subdirectory within the volume to store PD Data. By default, the data
is stored in the root directory of volume which is mounted at
/var/lib/pd.
Specifying this will change the data directory to a subdirectory, e.g.
/var/lib/pd/data if you set the value to &ldquo;data&rdquo;.
It&rsquo;s dangerous to change this value for a running cluster as it will
upgrade your cluster to use a new storage directory.
Defaults to &ldquo;&rdquo; (volume&rsquo;s root).</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#pdconfigwraper">
PDConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of pd-servers</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretName is the name of secret which stores tidb server client certificate
which used by Dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>enableDashboardInternalProxy</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>(Deprecated) EnableDashboardInternalProxy would directly set <code>internal-proxy</code> in the <code>PdConfig</code>.
Note that this is deprecated, we should just set <code>dashboard.internal-proxy</code> in <code>pd.config</code>.</p>
</td>
</tr>
<tr>
<td>
<code>mountClusterClientSecret</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>MountClusterClientSecret indicates whether to mount <code>cluster-client-secret</code> to the Pod</p>
</td>
</tr>
<tr>
<td>
<code>startUpScriptVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Start up script version</p>
</td>
</tr>
<tr>
<td>
<code>startTimeout</code></br>
<em>
int
</em>
</td>
<td>
<p>Timeout threshold when pd get started</p>
</td>
</tr>
<tr>
<td>
<code>initWaitTime</code></br>
<em>
int
</em>
</td>
<td>
<p>Wait time before pd get started. This wait time is to allow the new DNS record to propagate,
ensuring that the PD DNS resolves to the same IP address as the pod.</p>
</td>
</tr>
<tr>
<td>
<code>mode</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Mode is the mode of PD cluster</p>
</td>
</tr>
<tr>
<td>
<code>spareVolReplaceReplicas</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>The default number of spare replicas to scale up when using VolumeReplace feature.
In multi-az deployments with topology spread constraints you may need to set this to number of zones to avoid
zone skew after volume replace (total replicas always whole multiples of zones).
Optional: Defaults to 1</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdstatus">PDStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>PDStatus is PD status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
<a href="#pdmember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.PDMember
</a>
</em>
</td>
<td>
<p>Members contains PDs in current TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>peerMembers</code></br>
<em>
<a href="#pdmember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.PDMember
</a>
</em>
</td>
<td>
<p>PeerMembers contains PDs NOT in current TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>leader</code></br>
<em>
<a href="#pdmember">
PDMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureMembers</code></br>
<em>
<a href="#pdfailuremember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.PDFailureMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>unjoinedMembers</code></br>
<em>
<a href="#unjoinedmember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.UnjoinedMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
<tr>
<td>
<code>volReplaceInProgress</code></br>
<em>
bool
</em>
</td>
<td>
<p>Indicates that a Volume replace using VolumeReplacing feature is in progress.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pdstorelabel">PDStoreLabel</h3>
<p>
<p>PDStoreLabel is the config item of LabelPropertyConfig.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="pdstorelabels">PDStoreLabels</h3>
<p>
</p>
<h3 id="performance">Performance</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Performance is the performance section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>max-procs</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-memory</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>stats-lease</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3s</p>
</td>
</tr>
<tr>
<td>
<code>stmt-count-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 5000</p>
</td>
</tr>
<tr>
<td>
<code>feedback-probability</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0.05</p>
</td>
</tr>
<tr>
<td>
<code>query-feedback-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 512</p>
</td>
</tr>
<tr>
<td>
<code>pseudo-estimate-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0.8</p>
</td>
</tr>
<tr>
<td>
<code>force-priority</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to NO_PRIORITY</p>
</td>
</tr>
<tr>
<td>
<code>bind-info-lease</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3s</p>
</td>
</tr>
<tr>
<td>
<code>txn-total-size-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 104857600</p>
</td>
</tr>
<tr>
<td>
<code>tcp-keep-alive</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>cross-join</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>run-auto-analyze</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>agg-push-down-join</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>committer-concurrency</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-txn-ttl</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>txn-entry-count-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0
Optional: Defaults to 300000</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pessimistictxn">PessimisticTxn</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>PessimisticTxn is the config for pessimistic transaction.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enable</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Enable must be true for &lsquo;begin lock&rsquo; or session variable to start a pessimistic transaction.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>max-retry-count</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>The max count of retry for a single statement in a pessimistic transaction.
Optional: Defaults to 256</p>
</td>
</tr>
</tbody>
</table>
<h3 id="plancache">PlanCache</h3>
<p>
<p>PlanCache is the PlanCache section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>shards</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="plugin">Plugin</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Plugin is the config for plugin</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>dir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>load</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="preparedplancache">PreparedPlanCache</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>PreparedPlanCache is the PreparedPlanCache section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 100</p>
</td>
</tr>
<tr>
<td>
<code>memory-guard-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0.1</p>
</td>
</tr>
</tbody>
</table>
<h3 id="probe">Probe</h3>
<p>
(<em>Appears on:</em>
<a href="#componentspec">ComponentSpec</a>, 
<a href="#dmdiscoveryspec">DMDiscoverySpec</a>, 
<a href="#discoveryspec">DiscoverySpec</a>)
</p>
<p>
<p>Probe contains details of probing tidb.
default probe by TCPPort on tidb 4000 / tikv 20160 / pd 2349.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>&ldquo;tcp&rdquo; will use TCP socket to connect component port.</p>
<p>&ldquo;command&rdquo; will probe the status api of tidb.
This will use curl command to request tidb, before v4.0.9 there is no curl in the image,
So do not use this before v4.0.9.</p>
</td>
</tr>
<tr>
<td>
<code>initialDelaySeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Number of seconds after the container has started before liveness probes are initiated.
Default to 10 seconds.</p>
</td>
</tr>
<tr>
<td>
<code>periodSeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>How often (in seconds) to perform the probe.
Default to Kubernetes default (10 seconds). Minimum value is 1.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="profile">Profile</h3>
<p>
<p>Profile is the configuration profiles.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>readonly</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max_memory_usage</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>use_uncompressed_cache</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>load_balancing</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="progress">Progress</h3>
<p>
(<em>Appears on:</em>
<a href="#backupstatus">BackupStatus</a>, 
<a href="#restorestatus">RestoreStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>step</code></br>
<em>
string
</em>
</td>
<td>
<p>Step is the step name of progress</p>
</td>
</tr>
<tr>
<td>
<code>progress</code></br>
<em>
float64
</em>
</td>
<td>
<p>Progress is the backup progress value</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastTransitionTime is the update time</p>
</td>
</tr>
</tbody>
</table>
<h3 id="prometheusconfiguration">PrometheusConfiguration</h3>
<p>
(<em>Appears on:</em>
<a href="#prometheusspec">PrometheusSpec</a>)
</p>
<p>
<p>Config  is the the desired state of Prometheus Configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>configMapRef</code></br>
<em>
<a href="#configmapref">
ConfigMapRef
</a>
</em>
</td>
<td>
<p>User can mount prometheus config with external configMap. The external configMap must contain <code>prometheus-config</code> key in data.</p>
</td>
</tr>
<tr>
<td>
<code>commandOptions</code></br>
<em>
[]string
</em>
</td>
<td>
<p>user can  use it specify prometheus command options</p>
</td>
</tr>
<tr>
<td>
<code>ruleConfigRef</code></br>
<em>
<a href="#configmapref">
ConfigMapRef
</a>
</em>
</td>
<td>
<p>User can mount prometheus rule config with external configMap. The external configMap must use the key with suffix <code>.rules.yml</code>.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="prometheusreloaderspec">PrometheusReloaderSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
<p>PrometheusReloaderSpec is the desired state of prometheus configuration reloader</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>MonitorContainer</code></br>
<em>
<a href="#monitorcontainer">
MonitorContainer
</a>
</em>
</td>
<td>
<p>
(Members of <code>MonitorContainer</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="prometheusspec">PrometheusSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
<p>PrometheusSpec is the desired state of prometheus</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>MonitorContainer</code></br>
<em>
<a href="#monitorcontainer">
MonitorContainer
</a>
</em>
</td>
<td>
<p>
(Members of <code>MonitorContainer</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>logLevel</code></br>
<em>
string
</em>
</td>
<td>
<p>Prometheus log level</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<p>Service defines a Kubernetes service of Prometheus.</p>
</td>
</tr>
<tr>
<td>
<code>reserveDays</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReserveDays defines Prometheus Configuration for <code>--storage.tsdb.retention.time</code> of units d.
reserveDays will be used if retentionTime not defined.</p>
</td>
</tr>
<tr>
<td>
<code>retentionTime</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Configuration for <code>--storage.tsdb.retention.time</code>, Units Supported: y, w, d, h, m, s, ms.
If set to non empty values, it will override the value of <code>ReserveDays</code>.</p>
</td>
</tr>
<tr>
<td>
<code>ingress</code></br>
<em>
<a href="#ingressspec">
IngressSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ingress configuration of Prometheus</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#prometheusconfiguration">
PrometheusConfiguration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of Prometheus include Prometheus config/Cli options/custom rules.</p>
</td>
</tr>
<tr>
<td>
<code>disableCompaction</code></br>
<em>
bool
</em>
</td>
<td>
<p>Disable prometheus compaction.
Defaults to false.</p>
</td>
</tr>
<tr>
<td>
<code>remoteWrite</code></br>
<em>
<a href="#remotewritespec">
[]RemoteWriteSpec
</a>
</em>
</td>
<td>
<p>If specified, the remote_write spec. This is an experimental feature, it may change in any upcoming release in a breaking way.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<p>Additional volume mounts of prometheus pod.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="proxyconfig">ProxyConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashconfig">TiFlashConfig</a>)
</p>
<p>
<p>ProxyConfig is the configuration of TiFlash proxy process.
All the configurations are same with those of TiKV except adding <code>engine-addr</code> in the TiKVServerConfig</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>log-level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>log-rotation-timespan</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 24h</p>
</td>
</tr>
<tr>
<td>
<code>panic-when-unexpected-key-or-data</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#flashserverconfig">
FlashServerConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>storage</code></br>
<em>
<a href="#tikvstorageconfig">
TiKVStorageConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raftstore</code></br>
<em>
<a href="#tikvraftstoreconfig">
TiKVRaftstoreConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>rocksdb</code></br>
<em>
<a href="#tikvdbconfig">
TiKVDbConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>coprocessor</code></br>
<em>
<a href="#tikvcoprocessorconfig">
TiKVCoprocessorConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>readpool</code></br>
<em>
<a href="#tikvreadpoolconfig">
TiKVReadPoolConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raftdb</code></br>
<em>
<a href="#tikvraftdbconfig">
TiKVRaftDBConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>import</code></br>
<em>
<a href="#tikvimportconfig">
TiKVImportConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>gc</code></br>
<em>
<a href="#tikvgcconfig">
TiKVGCConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pd</code></br>
<em>
<a href="#tikvpdconfig">
TiKVPDConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>security</code></br>
<em>
<a href="#tikvsecurityconfig">
TiKVSecurityConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="proxyprotocol">ProxyProtocol</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>ProxyProtocol is the PROXY protocol section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>networks</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PROXY protocol acceptable client networks.
Empty *string means disable PROXY protocol,
* means all networks.</p>
</td>
</tr>
<tr>
<td>
<code>header-timeout</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>PROXY protocol header read timeout, Unit is second.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="prunetype">PruneType</h3>
<p>
(<em>Appears on:</em>
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>PruneType represents the prune type for restore.</p>
</p>
<h3 id="pumpnodestatus">PumpNodeStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#pumpstatus">PumpStatus</a>)
</p>
<p>
<p>PumpNodeStatus represents the status saved in etcd.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeId</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>state</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="pumpspec">PumpSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>PumpSpec contains details of Pump members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for pump</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Pump data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<em>(Optional)</em>
<p>The configuration of Pump cluster.</p>
</td>
</tr>
<tr>
<td>
<code>setTimeZone</code></br>
<em>
bool
</em>
</td>
<td>
<p>For backward compatibility with helm chart</p>
</td>
</tr>
</tbody>
</table>
<h3 id="pumpstatus">PumpStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>PumpStatus is Pump status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
<a href="#pumpnodestatus">
[]PumpNodeStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="queueconfig">QueueConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#remotewritespec">RemoteWriteSpec</a>)
</p>
<p>
<p>QueueConfig allows the tuning of remote_write queue_config parameters. This object
is referenced in the RemoteWriteSpec object.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>capacity</code></br>
<em>
int
</em>
</td>
<td>
<p>Number of samples to buffer per shard before we start dropping them.</p>
</td>
</tr>
<tr>
<td>
<code>minShards</code></br>
<em>
int
</em>
</td>
<td>
<p>MinShards is the minimum number of shards, i.e. amount of concurrency.
Only valid in Prometheus versions 2.6.0 and newer.</p>
</td>
</tr>
<tr>
<td>
<code>maxShards</code></br>
<em>
int
</em>
</td>
<td>
<p>Max number of shards, i.e. amount of concurrency.</p>
</td>
</tr>
<tr>
<td>
<code>maxSamplesPerSend</code></br>
<em>
int
</em>
</td>
<td>
<p>Maximum number of samples per send.</p>
</td>
</tr>
<tr>
<td>
<code>batchSendDeadline</code></br>
<em>
time.Duration
</em>
</td>
<td>
<p>Maximum time sample will wait in buffer.</p>
</td>
</tr>
<tr>
<td>
<code>maxRetries</code></br>
<em>
int
</em>
</td>
<td>
<p>Max number of times to retry a batch on recoverable errors.</p>
</td>
</tr>
<tr>
<td>
<code>minBackoff</code></br>
<em>
time.Duration
</em>
</td>
<td>
<p>On recoverable errors, backoff exponentially.</p>
</td>
</tr>
<tr>
<td>
<code>maxBackoff</code></br>
<em>
time.Duration
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="quota">Quota</h3>
<p>
<p>Quota is the configuration of [quotas.default] section.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>interval</code></br>
<em>
<a href="#interval">
Interval
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="relabelconfig">RelabelConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#remotewritespec">RemoteWriteSpec</a>)
</p>
<p>
<p>RelabelConfig allows dynamic rewriting of the label set, being applied to samples before ingestion.
It defines <code>&lt;metric_relabel_configs&gt;</code>-section of Prometheus configuration.
More info: <a href="https://prometheus.io/docs/prometheus/latest/configuration/configuration/#metric_relabel_configs">https://prometheus.io/docs/prometheus/latest/configuration/configuration/#metric_relabel_configs</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>sourceLabels</code></br>
<em>
github.com/prometheus/common/model.LabelNames
</em>
</td>
<td>
<p>A list of labels from which values are taken and concatenated
with the configured separator in order.</p>
</td>
</tr>
<tr>
<td>
<code>separator</code></br>
<em>
string
</em>
</td>
<td>
<p>Separator is the string between concatenated values from the source labels.</p>
</td>
</tr>
<tr>
<td>
<code>regex</code></br>
<em>
string
</em>
</td>
<td>
<p>Regular expression against which the extracted value is matched. Default is &lsquo;(.*)&rsquo;</p>
</td>
</tr>
<tr>
<td>
<code>modulus</code></br>
<em>
uint64
</em>
</td>
<td>
<p>Modulus to take of the hash of concatenated values from the source labels.</p>
</td>
</tr>
<tr>
<td>
<code>targetLabel</code></br>
<em>
string
</em>
</td>
<td>
<p>TargetLabel is the label to which the resulting string is written in a replacement.
Regexp interpolation is allowed for the replace action.</p>
</td>
</tr>
<tr>
<td>
<code>replacement</code></br>
<em>
string
</em>
</td>
<td>
<p>Replacement is the regex replacement pattern to be used.</p>
</td>
</tr>
<tr>
<td>
<code>action</code></br>
<em>
github.com/prometheus/prometheus/model/relabel.Action
</em>
</td>
<td>
<p>Action is the action to be performed for the relabeling.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="reloaderspec">ReloaderSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
<p>ReloaderSpec is the desired state of reloader</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>MonitorContainer</code></br>
<em>
<a href="#monitorcontainer">
MonitorContainer
</a>
</em>
</td>
<td>
<p>
(Members of <code>MonitorContainer</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="remotewritespec">RemoteWriteSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#prometheusspec">PrometheusSpec</a>)
</p>
<p>
<p>RemoteWriteSpec defines the remote_write configuration for prometheus.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>The URL of the endpoint to send samples to.</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the remote write queue, must be unique if specified. The
name is used in metrics and logging in order to differentiate queues.
Only valid in Prometheus versions 2.15.0 and newer.</p>
</td>
</tr>
<tr>
<td>
<code>remoteTimeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>writeRelabelConfigs</code></br>
<em>
<a href="#relabelconfig">
[]RelabelConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The list of remote write relabel configurations.</p>
</td>
</tr>
<tr>
<td>
<code>basicAuth</code></br>
<em>
<a href="#basicauth">
BasicAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BasicAuth for the URL.</p>
</td>
</tr>
<tr>
<td>
<code>bearerToken</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>File to read bearer token for remote write.</p>
</td>
</tr>
<tr>
<td>
<code>bearerTokenFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>File to read bearer token for remote write.</p>
</td>
</tr>
<tr>
<td>
<code>tlsConfig</code></br>
<em>
<a href="#tlsconfig">
TLSConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLS Config to use for remote write.</p>
</td>
</tr>
<tr>
<td>
<code>proxyUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Proxy url</p>
</td>
</tr>
<tr>
<td>
<code>queueConfig</code></br>
<em>
<a href="#queueconfig">
QueueConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>metadataConfig</code></br>
<em>
<a href="#metadataconfig">
MetadataConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MetadataConfig configures the sending of series metadata to remote storage.
Only valid in Prometheus versions 2.23.0 and newer.</p>
</td>
</tr>
<tr>
<td>
<code>headers</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Custom HTTP headers to be sent along with each remote write request.
Be aware that headers that are set by Prometheus itself can&rsquo;t be overwritten.
Only valid in Prometheus versions 2.25.0 and newer.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="restorecondition">RestoreCondition</h3>
<p>
(<em>Appears on:</em>
<a href="#restorestatus">RestoreStatus</a>)
</p>
<p>
<p>RestoreCondition describes the observed state of a Restore at a certain point.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#restoreconditiontype">
RestoreConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="restoreconditiontype">RestoreConditionType</h3>
<p>
(<em>Appears on:</em>
<a href="#restorecondition">RestoreCondition</a>, 
<a href="#restorestatus">RestoreStatus</a>)
</p>
<p>
<p>RestoreConditionType represents a valid condition of a Restore.</p>
</p>
<h3 id="restoremode">RestoreMode</h3>
<p>
(<em>Appears on:</em>
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>RestoreMode represents the restore mode, such as snapshot or pitr.</p>
</p>
<h3 id="restorespec">RestoreSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#restore">Restore</a>)
</p>
<p>
<p>RestoreSpec contains the specification for a restore of a tidb cluster backup.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like v1.Container.Env.
Note that the following builtin env vars will be overwritten by values set here
- S3_PROVIDER
- S3_ENDPOINT
- AWS_REGION
- AWS_ACL
- AWS_STORAGE_CLASS
- AWS_DEFAULT_REGION
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCS_PROJECT_ID
- GCS_OBJECT_ACL
- GCS_BUCKET_ACL
- GCS_LOCATION
- GCS_STORAGE_CLASS
- GCS_SERVICE_ACCOUNT_JSON_KEY
- BR_LOG_TO_TERM</p>
</td>
</tr>
<tr>
<td>
<code>to</code></br>
<em>
<a href="#tidbaccessconfig">
TiDBAccessConfig
</a>
</em>
</td>
<td>
<p>To is the tidb cluster that needs to restore.</p>
</td>
</tr>
<tr>
<td>
<code>backupType</code></br>
<em>
<a href="#backuptype">
BackupType
</a>
</em>
</td>
<td>
<p>Type is the backup type for tidb cluster and only used when Mode = snapshot, such as full, db, table.</p>
</td>
</tr>
<tr>
<td>
<code>restoreMode</code></br>
<em>
<a href="#restoremode">
RestoreMode
</a>
</em>
</td>
<td>
<p>Mode is the restore mode. such as snapshot or pitr.</p>
</td>
</tr>
<tr>
<td>
<code>pitrRestoredTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PitrRestoredTs is the pitr restored ts.</p>
</td>
</tr>
<tr>
<td>
<code>prune</code></br>
<em>
<a href="#prunetype">
PruneType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Prune is the prune type for restore, it is optional and can only have two valid values: afterFailed/alreadyFailed</p>
</td>
</tr>
<tr>
<td>
<code>logRestoreStartTs</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogRestoreStartTs is the start timestamp which log restore from.</p>
</td>
</tr>
<tr>
<td>
<code>federalVolumeRestorePhase</code></br>
<em>
<a href="#federalvolumerestorephase">
FederalVolumeRestorePhase
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FederalVolumeRestorePhase indicates which phase to execute in federal volume restore</p>
</td>
</tr>
<tr>
<td>
<code>volumeAZ</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>VolumeAZ indicates which AZ the volume snapshots restore to.
it is only valid for mode of volume-snapshot</p>
</td>
</tr>
<tr>
<td>
<code>tikvGCLifeTime</code></br>
<em>
string
</em>
</td>
<td>
<p>TikvGCLifeTime is to specify the safe gc life time for restore.
The time limit during which data is retained for each GC, in the format of Go Duration.
When a GC happens, the current time minus this value is the safe point.</p>
</td>
</tr>
<tr>
<td>
<code>StorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<p>
(Members of <code>StorageProvider</code> are embedded into this type.)
</p>
<p>StorageProvider configures where and how backups should be stored.</p>
</td>
</tr>
<tr>
<td>
<code>pitrFullBackupStorageProvider</code></br>
<em>
<a href="#storageprovider">
StorageProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PitrFullBackupStorageProvider configures where and how pitr dependent full backup should be stored.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for Restore data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageSize is the request storage size for backup job</p>
</td>
</tr>
<tr>
<td>
<code>br</code></br>
<em>
<a href="#brconfig">
BRConfig
</a>
</em>
</td>
<td>
<p>BR is the configs for BR.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of restore Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of restore Pods</p>
</td>
</tr>
<tr>
<td>
<code>useKMS</code></br>
<em>
bool
</em>
</td>
<td>
<p>Use KMS to decrypt the secrets</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify service account of restore</p>
</td>
</tr>
<tr>
<td>
<code>toolImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ToolImage specifies the tool image used in <code>Restore</code>, which supports BR and TiDB Lightning images.
For examples <code>spec.toolImage: pingcap/br:v4.0.8</code> or <code>spec.toolImage: pingcap/tidb-lightning:v4.0.8</code>
For BR image, if it does not contain tag, Pod will use image &lsquo;ToolImage:${TiKV_Version}&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>tableFilter</code></br>
<em>
[]string
</em>
</td>
<td>
<p>TableFilter means Table filter expression for &lsquo;db.table&rsquo; matching. BR supports this from v4.0.3.</p>
</td>
</tr>
<tr>
<td>
<code>warmup</code></br>
<em>
<a href="#restorewarmupmode">
RestoreWarmupMode
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Warmup represents whether to initialize TiKV volumes after volume snapshot restore</p>
</td>
</tr>
<tr>
<td>
<code>warmupImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>WarmupImage represents using what image to initialize TiKV volumes</p>
</td>
</tr>
<tr>
<td>
<code>warmupStrategy</code></br>
<em>
<a href="#restorewarmupstrategy">
RestoreWarmupStrategy
</a>
</em>
</td>
<td>
<p>WarmupStrategy</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName of Restore Job Pods</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volume mounts of component pod.</p>
</td>
</tr>
<tr>
<td>
<code>tolerateSingleTiKVOutage</code></br>
<em>
bool
</em>
</td>
<td>
<p>TolerateSingleTiKVOutage indicates whether to tolerate a single failure of a store without data loss</p>
</td>
</tr>
<tr>
<td>
<code>backoffLimit</code></br>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="restorestatus">RestoreStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#restore">Restore</a>)
</p>
<p>
<p>RestoreStatus represents the current status of a tidb cluster restore.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>timeStarted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeStarted is the time at which the restore was started.</p>
</td>
</tr>
<tr>
<td>
<code>timeCompleted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeCompleted is the time at which the restore was completed.</p>
</td>
</tr>
<tr>
<td>
<code>timeTaken</code></br>
<em>
string
</em>
</td>
<td>
<p>TimeTaken is the time that restore takes, it is TimeCompleted - TimeStarted</p>
</td>
</tr>
<tr>
<td>
<code>commitTs</code></br>
<em>
string
</em>
</td>
<td>
<p>CommitTs is the snapshot time point of tidb cluster.</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#restoreconditiontype">
RestoreConditionType
</a>
</em>
</td>
<td>
<p>Phase is a user readable state inferred from the underlying Restore conditions</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#restorecondition">
[]RestoreCondition
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>progresses</code></br>
<em>
<a href="#progress">
[]Progress
</a>
</em>
</td>
<td>
<p>Progresses is the progress of restore.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="restorewarmupmode">RestoreWarmupMode</h3>
<p>
(<em>Appears on:</em>
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>RestoreWarmupMode represents when to initialize TiKV volumes</p>
</p>
<h3 id="restorewarmupstrategy">RestoreWarmupStrategy</h3>
<p>
(<em>Appears on:</em>
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>RestoreWarmupStrategy represents how to initialize TiKV volumes</p>
</p>
<h3 id="s3storageprovider">S3StorageProvider</h3>
<p>
(<em>Appears on:</em>
<a href="#storageprovider">StorageProvider</a>)
</p>
<p>
<p>S3StorageProvider represents a S3 compliant storage for storing backups.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#s3storageprovidertype">
S3StorageProviderType
</a>
</em>
</td>
<td>
<p>Provider represents the specific storage provider that implements the S3 interface</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region in which the S3 compatible bucket is located.</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path is the full path where the backup is saved.
The format of the path must be: &ldquo;<bucket-name>/<path-to-backup-file>&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>bucket</code></br>
<em>
string
</em>
</td>
<td>
<p>Bucket in which to store the backup data.</p>
</td>
</tr>
<tr>
<td>
<code>endpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>Endpoint of S3 compatible storage service</p>
</td>
</tr>
<tr>
<td>
<code>storageClass</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageClass represents the storage class</p>
</td>
</tr>
<tr>
<td>
<code>acl</code></br>
<em>
string
</em>
</td>
<td>
<p>Acl represents access control permissions for this bucket</p>
</td>
</tr>
<tr>
<td>
<code>secretName</code></br>
<em>
string
</em>
</td>
<td>
<p>SecretName is the name of secret which stores
S3 compliant storage access key and secret key.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<p>Prefix of the data path.</p>
</td>
</tr>
<tr>
<td>
<code>sse</code></br>
<em>
string
</em>
</td>
<td>
<p>SSE Sever-Side Encryption.</p>
</td>
</tr>
<tr>
<td>
<code>options</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Options Rclone options for backup and restore with dumpling and lightning.</p>
</td>
</tr>
<tr>
<td>
<code>forcePathStyle</code></br>
<em>
bool
</em>
</td>
<td>
<p>ForcePathStyle for the backup and restore to connect s3 with path style(true) or virtual host(false).</p>
</td>
</tr>
</tbody>
</table>
<h3 id="s3storageprovidertype">S3StorageProviderType</h3>
<p>
(<em>Appears on:</em>
<a href="#s3storageprovider">S3StorageProvider</a>)
</p>
<p>
<p>S3StorageProviderType represents the specific storage provider that implements the S3 interface</p>
</p>
<h3 id="safetlsconfig">SafeTLSConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tlsconfig">TLSConfig</a>)
</p>
<p>
<p>SafeTLSConfig specifies safe TLS configuration parameters.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ca</code></br>
<em>
<a href="#secretorconfigmap">
SecretOrConfigMap
</a>
</em>
</td>
<td>
<p>Struct containing the CA cert to use for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>cert</code></br>
<em>
<a href="#secretorconfigmap">
SecretOrConfigMap
</a>
</em>
</td>
<td>
<p>Struct containing the client cert file for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>keySecret</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Secret containing the client key file for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>serverName</code></br>
<em>
string
</em>
</td>
<td>
<p>Used to verify the hostname for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>insecureSkipVerify</code></br>
<em>
bool
</em>
</td>
<td>
<p>Disable target certificate validation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scalepolicy">ScalePolicy</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>, 
<a href="#tiflashspec">TiFlashSpec</a>, 
<a href="#tikvspec">TiKVSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>scaleInParallelism</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScaleInParallelism configures max scale in replicas for TiKV stores.</p>
</td>
</tr>
<tr>
<td>
<code>scaleOutParallelism</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScaleOutParallelism configures max scale out replicas for TiKV stores.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="secretorconfigmap">SecretOrConfigMap</h3>
<p>
(<em>Appears on:</em>
<a href="#safetlsconfig">SafeTLSConfig</a>)
</p>
<p>
<p>SecretOrConfigMap allows to specify data as a Secret or ConfigMap. Fields are mutually exclusive.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secret</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Secret containing data to use for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>configMap</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#configmapkeyselector-v1-core">
Kubernetes core/v1.ConfigMapKeySelector
</a>
</em>
</td>
<td>
<p>ConfigMap containing data to use for the targets.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="security">Security</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Security is the security section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>skip-grant-table</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ssl-ca</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ssl-cert</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ssl-key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cluster-ssl-ca</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cluster-ssl-cert</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cluster-ssl-key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cluster-verify-cn</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterVerifyCN is the Common Name that allowed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="service">Service</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>(Deprecated) Service represent service type used in TidbCluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="servicespec">ServiceSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#grafanaspec">GrafanaSpec</a>, 
<a href="#masterservicespec">MasterServiceSpec</a>, 
<a href="#pdmsspec">PDMSSpec</a>, 
<a href="#pdspec">PDSpec</a>, 
<a href="#prometheusspec">PrometheusSpec</a>, 
<a href="#reloaderspec">ReloaderSpec</a>, 
<a href="#tidbservicespec">TiDBServiceSpec</a>, 
<a href="#tidbdashboardspec">TidbDashboardSpec</a>)
</p>
<p>
<p>ServiceSpec specifies the service object in k8s</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#servicetype-v1-core">
Kubernetes core/v1.ServiceType
</a>
</em>
</td>
<td>
<p>Type of the real kubernetes service</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional annotations for the service</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional labels for the service</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerIP</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LoadBalancerIP is the loadBalancerIP of service
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>clusterIP</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterIP is the clusterIP of service</p>
</td>
</tr>
<tr>
<td>
<code>portName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PortName is the name of service port</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>The port that will be exposed by this service.</p>
<p>NOTE: only used for TiDB</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerSourceRanges</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LoadBalancerSourceRanges is the loadBalancerSourceRanges of service
If specified and supported by the platform, this will restrict traffic through the cloud-provider
load-balancer will be restricted to the specified client IPs. This field will be ignored if the
cloud-provider does not support the feature.&rdquo;
More info: <a href="https://kubernetes.io/docs/concepts/services-networking/service/#aws-nlb-support">https://kubernetes.io/docs/concepts/services-networking/service/#aws-nlb-support</a>
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerClass</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>loadBalancerClass is the class of the load balancer implementation this Service belongs to.
If specified, the value of this field must be a label-style identifier, with an optional prefix,
e.g. &ldquo;internal-vip&rdquo; or &ldquo;example.com/internal-vip&rdquo;. Unprefixed names are reserved for end-users.
This field can only be set when the Service type is &lsquo;LoadBalancer&rsquo;. If not set, the default load
balancer implementation is used, today this is typically done through the cloud provider integration,
but should apply for any default implementation. If set, it is assumed that a load balancer
implementation is watching for Services with a matching class. Any default load balancer
implementation (e.g. cloud providers) should ignore Services that set this field.
This field can only be set when creating or updating a Service to type &lsquo;LoadBalancer&rsquo;.
Once set, it can not be changed. This field will be wiped when a service is updated to a non &lsquo;LoadBalancer&rsquo; type.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="startscriptv2featureflag">StartScriptV2FeatureFlag</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
</p>
<h3 id="startscriptversion">StartScriptVersion</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
</p>
<h3 id="status">Status</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>Status is the status section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metrics-addr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>metrics-interval</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 15</p>
</td>
</tr>
<tr>
<td>
<code>report-status</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>record-db-qps</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
</tbody>
</table>
<h3 id="stmtsummary">StmtSummary</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>StmtSummary is the config for statement summary.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enable</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Enable statement summary or not.</p>
</td>
</tr>
<tr>
<td>
<code>enable-internal-query</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Enable summary internal query.</p>
</td>
</tr>
<tr>
<td>
<code>max-stmt-count</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>The maximum number of statements kept in memory.
Optional: Defaults to 100</p>
</td>
</tr>
<tr>
<td>
<code>max-sql-length</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>The maximum length of displayed normalized SQL and sample SQL.
Optional: Defaults to 4096</p>
</td>
</tr>
<tr>
<td>
<code>refresh-interval</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>The refresh interval of statement summary.</p>
</td>
</tr>
<tr>
<td>
<code>history-size</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>The maximum history size of statement summary.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="storageclaim">StorageClaim</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>)
</p>
<p>
<p>StorageClaim contains details of TiFlash storages</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Resources represents the minimum resources the volume should have.
More info: <a href="https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources">https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources</a></p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name of the StorageClass required by the claim.
More info: <a href="https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1">https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="storageprovider">StorageProvider</h3>
<p>
(<em>Appears on:</em>
<a href="#backupschedulespec">BackupScheduleSpec</a>, 
<a href="#backupspec">BackupSpec</a>, 
<a href="#compactspec">CompactSpec</a>, 
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>StorageProvider defines the configuration for storing a backup in backend storage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>s3</code></br>
<em>
<a href="#s3storageprovider">
S3StorageProvider
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gcs</code></br>
<em>
<a href="#gcsstorageprovider">
GcsStorageProvider
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>azblob</code></br>
<em>
<a href="#azblobstorageprovider">
AzblobStorageProvider
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>local</code></br>
<em>
<a href="#localstorageprovider">
LocalStorageProvider
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="storagevolume">StorageVolume</h3>
<p>
(<em>Appears on:</em>
<a href="#ngmonitoringspec">NGMonitoringSpec</a>, 
<a href="#pdmsspec">PDMSSpec</a>, 
<a href="#pdspec">PDSpec</a>, 
<a href="#ticdcspec">TiCDCSpec</a>, 
<a href="#tidbspec">TiDBSpec</a>, 
<a href="#tikvspec">TiKVSpec</a>, 
<a href="#tiproxyspec">TiProxySpec</a>, 
<a href="#tidbdashboardspec">TidbDashboardSpec</a>)
</p>
<p>
<p>StorageVolume configures additional PVC template for StatefulSets and volumeMount for pods that mount this PVC.
Note:
If <code>MountPath</code> is not set, volumeMount will not be generated. (You may not want to set this field when you inject volumeMount
in somewhere else such as Mutating Admission Webhook)
If <code>StorageClassName</code> is not set, default to the <code>spec.${component}.storageClassName</code></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>mountPath</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="storagevolumename">StorageVolumeName</h3>
<p>
(<em>Appears on:</em>
<a href="#storagevolumestatus">StorageVolumeStatus</a>)
</p>
<p>
<p>StorageVolumeName is the volume name which is same as <code>volumes.name</code> in Pod spec.</p>
</p>
<h3 id="storagevolumestatus">StorageVolumeStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#masterstatus">MasterStatus</a>, 
<a href="#pdmsstatus">PDMSStatus</a>, 
<a href="#pdstatus">PDStatus</a>, 
<a href="#pumpstatus">PumpStatus</a>, 
<a href="#ticdcstatus">TiCDCStatus</a>, 
<a href="#tidbstatus">TiDBStatus</a>, 
<a href="#tikvstatus">TiKVStatus</a>, 
<a href="#tiproxystatus">TiProxyStatus</a>, 
<a href="#workerstatus">WorkerStatus</a>)
</p>
<p>
<p>StorageVolumeStatus is the actual status for a storage</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ObservedStorageVolumeStatus</code></br>
<em>
<a href="#observedstoragevolumestatus">
ObservedStorageVolumeStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>ObservedStorageVolumeStatus</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
<a href="#storagevolumename">
StorageVolumeName
</a>
</em>
</td>
<td>
<p>Name is the volume name which is same as <code>volumes.name</code> in Pod spec.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="suspendaction">SuspendAction</h3>
<p>
(<em>Appears on:</em>
<a href="#componentspec">ComponentSpec</a>, 
<a href="#dmclusterspec">DMClusterSpec</a>, 
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>SuspendAction defines the suspend actions for a component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>suspendStatefulSet</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tlscluster">TLSCluster</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterspec">DMClusterSpec</a>, 
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TLSCluster can enable mutual TLS connection between TiDB cluster components
<a href="https://pingcap.com/docs/stable/how-to/secure/enable-tls-between-components/">https://pingcap.com/docs/stable/how-to/secure/enable-tls-between-components/</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Enable mutual TLS connection between TiDB cluster components
Once enabled, the mutual authentication applies to all components,
and it does not support applying to only part of the components.
The steps to enable this feature:
1. Generate TiDB cluster components certificates and a client-side certifiacete for them.
There are multiple ways to generate these certificates:
- user-provided certificates: <a href="https://pingcap.com/docs/stable/how-to/secure/generate-self-signed-certificates/">https://pingcap.com/docs/stable/how-to/secure/generate-self-signed-certificates/</a>
- use the K8s built-in certificate signing system signed certificates: <a href="https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/">https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/</a>
- or use cert-manager signed certificates: <a href="https://cert-manager.io/">https://cert-manager.io/</a>
2. Create one secret object for one component which contains the certificates created above.
The name of this Secret must be: <clusterName>-<componentName>-cluster-secret.
For PD: kubectl create secret generic <clusterName>-pd-cluster-secret &ndash;namespace=<namespace> &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
For TiKV: kubectl create secret generic <clusterName>-tikv-cluster-secret &ndash;namespace=<namespace> &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
For TiDB: kubectl create secret generic <clusterName>-tidb-cluster-secret &ndash;namespace=<namespace> &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
For Client: kubectl create secret generic <clusterName>-cluster-client-secret &ndash;namespace=<namespace> &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
Same for other components.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tlsconfig">TLSConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#remotewritespec">RemoteWriteSpec</a>, 
<a href="#thanosspec">ThanosSpec</a>)
</p>
<p>
<p>TLSConfig extends the safe TLS configuration with file parameters.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>SafeTLSConfig</code></br>
<em>
<a href="#safetlsconfig">
SafeTLSConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>SafeTLSConfig</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>caFile</code></br>
<em>
string
</em>
</td>
<td>
<p>Path to the CA cert in the Prometheus container to use for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>certFile</code></br>
<em>
string
</em>
</td>
<td>
<p>Path to the client cert file in the Prometheus container for the targets.</p>
</td>
</tr>
<tr>
<td>
<code>keyFile</code></br>
<em>
string
</em>
</td>
<td>
<p>Path to the client key file in the Prometheus container for the targets.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="thanosspec">ThanosSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
</p>
<p>
<p>ThanosSpec is the desired state of thanos sidecar</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>MonitorContainer</code></br>
<em>
<a href="#monitorcontainer">
MonitorContainer
</a>
</em>
</td>
<td>
<p>
(Members of <code>MonitorContainer</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>objectStorageConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>ObjectStorageConfig configures object storage in Thanos.
Alternative to ObjectStorageConfigFile, and lower order priority.</p>
</td>
</tr>
<tr>
<td>
<code>objectStorageConfigFile</code></br>
<em>
string
</em>
</td>
<td>
<p>ObjectStorageConfigFile specifies the path of the object storage configuration file.
When used alongside with ObjectStorageConfig, ObjectStorageConfigFile takes precedence.</p>
</td>
</tr>
<tr>
<td>
<code>listenLocal</code></br>
<em>
bool
</em>
</td>
<td>
<p>ListenLocal makes the Thanos sidecar listen on loopback, so that it
does not bind against the Pod IP.</p>
</td>
</tr>
<tr>
<td>
<code>tracingConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>TracingConfig configures tracing in Thanos. This is an experimental feature, it may change in any upcoming release in a breaking way.</p>
</td>
</tr>
<tr>
<td>
<code>tracingConfigFile</code></br>
<em>
string
</em>
</td>
<td>
<p>TracingConfig specifies the path of the tracing configuration file.
When used alongside with TracingConfig, TracingConfigFile takes precedence.</p>
</td>
</tr>
<tr>
<td>
<code>grpcServerTlsConfig</code></br>
<em>
<a href="#tlsconfig">
TLSConfig
</a>
</em>
</td>
<td>
<p>GRPCServerTLSConfig configures the gRPC server from which Thanos Querier reads
recorded rule data.
Note: Currently only the CAFile, CertFile, and KeyFile fields are supported.
Maps to the &lsquo;&ndash;grpc-server-tls-*&rsquo; CLI args.</p>
</td>
</tr>
<tr>
<td>
<code>logLevel</code></br>
<em>
string
</em>
</td>
<td>
<p>LogLevel for Thanos sidecar to be configured with.</p>
</td>
</tr>
<tr>
<td>
<code>logFormat</code></br>
<em>
string
</em>
</td>
<td>
<p>LogFormat for Thanos sidecar to be configured with.</p>
</td>
</tr>
<tr>
<td>
<code>minTime</code></br>
<em>
string
</em>
</td>
<td>
<p>MinTime for Thanos sidecar to be configured with. Option can be a constant time in RFC3339 format or time duration relative to current time, such as -1d or 2h45m. Valid duration units are ms, s, m, h, d, w, y.</p>
</td>
</tr>
<tr>
<td>
<code>routePrefix</code></br>
<em>
string
</em>
</td>
<td>
<p>RoutePrefix is prometheus prefix url</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<p>Additional volume mounts of thanos pod.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ticdccapture">TiCDCCapture</h3>
<p>
(<em>Appears on:</em>
<a href="#ticdcstatus">TiCDCStatus</a>)
</p>
<p>
<p>TiCDCCapture is TiCDC Capture status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>isOwner</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ready</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="ticdcconfig">TiCDCConfig</h3>
<p>
<p>TiCDCConfig is the configuration of tidbcdc
ref <a href="https://github.com/pingcap/ticdc/blob/a28d9e43532edc4a0380f0ef87314631bf18d866/pkg/config/config.go#L176">https://github.com/pingcap/ticdc/blob/a28d9e43532edc4a0380f0ef87314631bf18d866/pkg/config/config.go#L176</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TiCDC
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>gcTTL</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>CDC GC safepoint TTL duration, specified in seconds
Optional: Defaults to 86400</p>
</td>
</tr>
<tr>
<td>
<code>logLevel</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogLevel is the log level
Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>logFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogFile is the log file
Optional: Defaults to /dev/stderr</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ticdcspec">TiCDCSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TiCDCSpec contains details of TiCDC members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for TiCDC</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretNames</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretNames are the names of secrets that store the
client certificates for the downstream.</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#cdcconfigwraper">
CDCConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of tidbcdc servers</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageVolumes configure additional storage for TiCDC pods.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for TiCDC data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>gracefulShutdownTimeout</code></br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GracefulShutdownTimeout is the timeout of gracefully shutdown a TiCDC pod.
Encoded in the format of Go Duration.
Defaults to 10m</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ticdcstatus">TiCDCStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>TiCDCStatus is TiCDC status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>captures</code></br>
<em>
<a href="#ticdccapture">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiCDCCapture
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbaccessconfig">TiDBAccessConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>, 
<a href="#restorespec">RestoreSpec</a>)
</p>
<p>
<p>TiDBAccessConfig defines the configuration for access tidb cluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
<p>Host is the tidb cluster access address</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
int32
</em>
</td>
<td>
<p>Port is the port number to use for connecting tidb cluster</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
string
</em>
</td>
<td>
<p>User is the user for login tidb cluster</p>
</td>
</tr>
<tr>
<td>
<code>secretName</code></br>
<em>
string
</em>
</td>
<td>
<p>SecretName is the name of secret which stores tidb cluster&rsquo;s password.</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretName is the name of secret which stores tidb server client certificate
Optional: Defaults to nil</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbconfig">TiDBConfig</h3>
<p>
<p>TiDBConfig is the configuration of tidb-server
For more detail, refer to <a href="https://pingcap.com/docs/stable/reference/configuration/tidb-server/configuration/">https://pingcap.com/docs/stable/reference/configuration/tidb-server/configuration/</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>cors</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>socket</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>lease</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 45s</p>
</td>
</tr>
<tr>
<td>
<code>run-ddl</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>split-table</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>token-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 1000</p>
</td>
</tr>
<tr>
<td>
<code>oom-use-tmp-storage</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tmp-storage-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>oom-action</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to log</p>
</td>
</tr>
<tr>
<td>
<code>max-index-length</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3072</p>
</td>
</tr>
<tr>
<td>
<code>mem-quota-query</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 34359738368</p>
</td>
</tr>
<tr>
<td>
<code>tmp-storage-quota</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>TempStorageQuota describe the temporary storage Quota during query exector when OOMUseTmpStorage is enabled
If the quota exceed the capacity of the TempStoragePath, the tidb-server would exit with fatal error</p>
</td>
</tr>
<tr>
<td>
<code>enable-streaming</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>enable-batch-dml</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>txn-local-latches</code></br>
<em>
<a href="#txnlocallatches">
TxnLocalLatches
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0</p>
</td>
</tr>
<tr>
<td>
<code>lower-case-table-names</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>log</code></br>
<em>
<a href="#log">
Log
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>security</code></br>
<em>
<a href="#security">
Security
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#status">
Status
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>performance</code></br>
<em>
<a href="#performance">
Performance
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>prepared-plan-cache</code></br>
<em>
<a href="#preparedplancache">
PreparedPlanCache
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>opentracing</code></br>
<em>
<a href="#opentracing">
OpenTracing
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>proxy-protocol</code></br>
<em>
<a href="#proxyprotocol">
ProxyProtocol
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tikv-client</code></br>
<em>
<a href="#tikvclient">
TiKVClient
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>binlog</code></br>
<em>
<a href="#binlog">
Binlog
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>compatible-kill-query</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>plugin</code></br>
<em>
<a href="#plugin">
Plugin
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pessimistic-txn</code></br>
<em>
<a href="#pessimistictxn">
PessimisticTxn
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>check-mb4-value-in-utf8</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>alter-primary-key</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>treat-old-version-utf8-as-utf8mb4</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>split-region-max-num</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 1000</p>
</td>
</tr>
<tr>
<td>
<code>stmt-summary</code></br>
<em>
<a href="#stmtsummary">
StmtSummary
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>repair-mode</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>RepairMode indicates that the TiDB is in the repair mode for table meta.</p>
</td>
</tr>
<tr>
<td>
<code>repair-table-list</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>isolation-read</code></br>
<em>
<a href="#isolationread">
IsolationRead
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IsolationRead indicates that the TiDB reads data from which isolation level(engine and label).</p>
</td>
</tr>
<tr>
<td>
<code>max-server-connections</code></br>
<em>
uint32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxServerConnections is the maximum permitted number of simultaneous client connections.</p>
</td>
</tr>
<tr>
<td>
<code>new_collations_enabled_on_first_bootstrap</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>NewCollationsEnabledOnFirstBootstrap indicates if the new collations are enabled, it effects only when a TiDB cluster bootstrapped on the first time.</p>
</td>
</tr>
<tr>
<td>
<code>experimental</code></br>
<em>
<a href="#experimental">
Experimental
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Experimental contains parameters for experimental features.</p>
</td>
</tr>
<tr>
<td>
<code>enable-dynamic-config</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableDynamicConfig enables the TiDB to fetch configs from PD and update itself during runtime.
see <a href="https://github.com/pingcap/tidb/pull/13660">https://github.com/pingcap/tidb/pull/13660</a> for more details.</p>
</td>
</tr>
<tr>
<td>
<code>enable-table-lock</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>imported from v3.1.0</p>
</td>
</tr>
<tr>
<td>
<code>delay-clean-table-lock</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>imported from v3.1.0</p>
</td>
</tr>
<tr>
<td>
<code>skip-register-to-dashboard</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>imported from v4.0.5
SkipRegisterToDashboard tells TiDB don&rsquo;t register itself to the dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>enable-telemetry</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>When enabled, usage data (for example, instance versions) will be reported to PingCAP periodically for user experience analytics.
If this config is set to <code>false</code> on all TiDB servers, telemetry will be always disabled regardless of the value of the global variable <code>tidb_enable_telemetry</code>.
See PingCAP privacy policy for details: <a href="https://pingcap.com/en/privacy-policy/">https://pingcap.com/en/privacy-policy/</a>.
Imported from v4.0.2.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Labels are labels for TiDB server</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbconfigwraper">TiDBConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
<p>
<p>TiDBConfigWraper simply wrapps a GenericConfig</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbfailuremember">TiDBFailureMember</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbstatus">TiDBStatus</a>)
</p>
<p>
<p>TiDBFailureMember is the tidb failure member information</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbinitializer">TiDBInitializer</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>createPassword</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbmember">TiDBMember</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbstatus">TiDBStatus</a>)
</p>
<p>
<p>TiDBMember is TiDB member</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>health</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the health transitioned from one to another.</p>
</td>
</tr>
<tr>
<td>
<code>node</code></br>
<em>
string
</em>
</td>
<td>
<p>Node hosting pod of this TiDB member.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbservicespec">TiDBServiceSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
<p>
<p>TiDBServiceSpec defines <code>.tidb.service</code> field of <code>TidbCluster.spec</code>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ServiceSpec</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ServiceSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>externalTrafficPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#serviceexternaltrafficpolicy-v1-core">
Kubernetes core/v1.ServiceExternalTrafficPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalTrafficPolicy of the service
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>exposeStatus</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether expose the status port
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>mysqlNodePort</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Expose the tidb cluster mysql port to MySQLNodePort
Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>statusNodePort</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Expose the tidb status node port to StatusNodePort
Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>additionalPorts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#serviceport-v1-core">
[]Kubernetes core/v1.ServicePort
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Expose additional ports for TiDB
Optional: Defaults to omitted</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbslowlogtailerspec">TiDBSlowLogTailerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
<p>
<p>TiDBSlowLogTailerSpec represents an optional log tailer sidecar with TiDB</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>(Deprecated) Image used for slowlog tailer.
Use <code>spec.helper.image</code> instead</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>(Deprecated) ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
Use <code>spec.helper.imagePullPolicy</code> instead</p>
</td>
</tr>
<tr>
<td>
<code>useSidecar</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>If true, we use native sidecar feature to tail log
It requires enable feature gate &ldquo;SidecarContainers&rdquo;
This feature is introduced at 1.28, default enabled at 1.29, and GA at 1.33
See <a href="https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/">https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/</a>
and <a href="https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/">https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbspec">TiDBSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TiDBSpec contains details of TiDB members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for tidb</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#tidbservicespec">
TiDBServiceSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service defines a Kubernetes service of TiDB cluster.
Optional: No kubernetes service will be created by default.</p>
</td>
</tr>
<tr>
<td>
<code>binlogEnabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable TiDB Binlog, it is encouraged to not set this field and rely on the default behavior
Optional: Defaults to true if PumpSpec is non-nil, otherwise false</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>separateSlowLog</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether output the slow log in an separate sidecar container
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>slowLogVolumeName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional volume name configuration for slow query log.</p>
</td>
</tr>
<tr>
<td>
<code>slowLogTailer</code></br>
<em>
<a href="#tidbslowlogtailerspec">
TiDBSlowLogTailerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The specification of the slow log tailer sidecar</p>
</td>
</tr>
<tr>
<td>
<code>tlsClient</code></br>
<em>
<a href="#tidbtlsclient">
TiDBTLSClient
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable the TLS connection between the SQL client and TiDB server
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>tokenBasedAuthEnabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable <code>tidb_auth_token</code> authentication method. The tidb_auth_token authentication method is used only for the internal operation of TiDB Cloud.
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>plugins</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Plugins is a list of plugins that are loaded by TiDB server, empty means plugin disabled</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#tidbconfigwraper">
TiDBConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of tidb-servers</p>
</td>
</tr>
<tr>
<td>
<code>lifecycle</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#lifecycle-v1-core">
Kubernetes core/v1.Lifecycle
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Lifecycle describes actions that the management system should take in response to container lifecycle
events. For the PostStart and PreStop lifecycle handlers, management of the container blocks
until the action is complete, unless the container process fails, in which case the handler is aborted.</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageVolumes configure additional storage for TiDB pods.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for TiDB data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>initializer</code></br>
<em>
<a href="#tidbinitializer">
TiDBInitializer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Initializer is the init configurations of TiDB</p>
</td>
</tr>
<tr>
<td>
<code>bootstrapSQLConfigMapName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>BootstrapSQLConfigMapName is the name of the ConfigMap which contains the bootstrap SQL file with the key <code>bootstrap-sql</code>,
which will only be executed when a TiDB cluster bootstrap on the first time.
The field should be set ONLY when create a TC, since it only take effect on the first time bootstrap.
Only v6.5.1+ supports this feature.</p>
</td>
</tr>
<tr>
<td>
<code>scalePolicy</code></br>
<em>
<a href="#scalepolicy">
ScalePolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScalePolicy is the scale configuration for TiDB.</p>
</td>
</tr>
<tr>
<td>
<code>customizedStartupProbe</code></br>
<em>
<a href="#customizedprobe">
CustomizedProbe
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CustomizedStartupProbe is the customized startup probe for TiDB.
You can provide your own startup probe for TiDB.
The image will be an init container, and the tidb-server container will copy the probe binary from it, and execute it.
The probe binary in the image should be placed under the root directory, i.e., <code>/your-probe</code>.</p>
</td>
</tr>
<tr>
<td>
<code>arguments</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Arguments is the extra command line arguments for TiDB server.</p>
</td>
</tr>
<tr>
<td>
<code>serverLabels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>ServerLabels defines the server labels of the TiDB server.
Using both this field and config file to manage the labels is an undefined behavior.
Note these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
- region, topology.kubernetes.io/region
- zone, topology.kubernetes.io/zone
- host</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbstatus">TiDBStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>TiDBStatus is TiDB status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
<a href="#tidbmember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiDBMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureMembers</code></br>
<em>
<a href="#tidbfailuremember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiDBFailureMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resignDDLOwnerRetryCount</code></br>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>passwordInitialized</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
<tr>
<td>
<code>volReplaceInProgress</code></br>
<em>
bool
</em>
</td>
<td>
<p>Indicates that a Volume replace using VolumeReplacing feature is in progress.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbtlsclient">TiDBTLSClient</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
<p>
<p>TiDBTLSClient can enable TLS connection between TiDB server and MySQL client</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>When enabled, TiDB will accept TLS encrypted connections from MySQL client
The steps to enable this feature:
1. Generate a TiDB server-side certificate and a client-side certifiacete for the TiDB cluster.
There are multiple ways to generate certificates:
- user-provided certificates: <a href="https://pingcap.com/docs/stable/how-to/secure/enable-tls-clients/">https://pingcap.com/docs/stable/how-to/secure/enable-tls-clients/</a>
- use the K8s built-in certificate signing system signed certificates: <a href="https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/">https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/</a>
- or use cert-manager signed certificates: <a href="https://cert-manager.io/">https://cert-manager.io/</a>
2. Create a K8s Secret object which contains the TiDB server-side certificate created above.
The name of this Secret must be: <clusterName>-tidb-server-secret.
kubectl create secret generic <clusterName>-tidb-server-secret &ndash;namespace=<namespace> &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
3. Create a K8s Secret object which contains the TiDB client-side certificate created above which will be used by TiDB Operator.
The name of this Secret must be: <clusterName>-tidb-client-secret.
kubectl create secret generic <clusterName>-tidb-client-secret &ndash;namespace=<namespace> &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
4. Set Enabled to <code>true</code>.</p>
</td>
</tr>
<tr>
<td>
<code>disableClientAuthn</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableClientAuthn will skip client&rsquo;s certificate validation from the TiDB server.
Optional: defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>skipInternalClientCA</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>SkipInternalClientCA will skip TiDB server&rsquo;s certificate validation for internal components like Initializer, Dashboard, etc.
Optional: defaults to false</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiflashcommonconfigwraper">TiFlashCommonConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashconfigwraper">TiFlashConfigWraper</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiflashconfig">TiFlashConfig</h3>
<p>
<p>TiFlashConfig is the configuration of TiFlash.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#commonconfig">
CommonConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>commonConfig is the Configuration of TiFlash process</p>
</td>
</tr>
<tr>
<td>
<code>proxy</code></br>
<em>
<a href="#proxyconfig">
ProxyConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>proxyConfig is the Configuration of proxy process</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiflashconfigwraper">TiFlashConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#tiflashcommonconfigwraper">
TiFlashCommonConfigWraper
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>proxy</code></br>
<em>
<a href="#tiflashproxyconfigwraper">
TiFlashProxyConfigWraper
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tiflashproxyconfigwraper">TiFlashProxyConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashconfigwraper">TiFlashConfigWraper</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiflashspec">TiFlashSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TiFlashSpec contains details of TiFlash members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for TiFlash</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>privileged</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether create the TiFlash container in privileged mode, it is highly discouraged to enable this in
critical environment.
Optional: defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>storageClaims</code></br>
<em>
<a href="#storageclaim">
[]StorageClaim
</a>
</em>
</td>
<td>
<p>The persistent volume claims of the TiFlash data storages.
TiFlash supports multiple disks.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#tiflashconfigwraper">
TiFlashConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of TiFlash</p>
</td>
</tr>
<tr>
<td>
<code>initializer</code></br>
<em>
<a href="#initcontainerspec">
InitContainerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Initializer is the configurations of the init container for TiFlash</p>
</td>
</tr>
<tr>
<td>
<code>logTailer</code></br>
<em>
<a href="#logtailerspec">
LogTailerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogTailer is the configurations of the log tailers for TiFlash</p>
</td>
</tr>
<tr>
<td>
<code>recoverFailover</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>RecoverFailover indicates that Operator can recover the failover Pods</p>
</td>
</tr>
<tr>
<td>
<code>failover</code></br>
<em>
<a href="#failover">
Failover
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Failover is the configurations of failover</p>
</td>
</tr>
<tr>
<td>
<code>scalePolicy</code></br>
<em>
<a href="#scalepolicy">
ScalePolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScalePolicy is the scale configuration for TiFlash</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvbackupconfig">TiKVBackupConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>num-threads</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvblockcacheconfig">TiKVBlockCacheConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvstorageconfig">TiKVStorageConfig</a>)
</p>
<p>
<p>TiKVBlockCacheConfig is the config of a block cache</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>shared</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>num-shard-bits</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>strict-capacity-limit</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>high-pri-pool-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>memory-allocator</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvcfconfig">TiKVCfConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvdbconfig">TiKVDbConfig</a>, 
<a href="#tikvraftdbconfig">TiKVRaftDBConfig</a>)
</p>
<p>
<p>TiKVCfConfig is the config of a cf</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>block-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>block-cache-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>disable-block-cache</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cache-index-and-filter-blocks</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pin-l0-filter-and-index-blocks</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>use-bloom-filter</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>optimize-filters-for-hits</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>whole-key-filtering</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>bloom-filter-bits-per-key</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>block-based-bloom-filter</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>read-amp-bytes-per-bit</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>compression-per-level</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>write-buffer-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-write-buffer-number</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>min-write-buffer-number-to-merge</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-bytes-for-level-base</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>target-file-size-base</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>level0-file-num-compaction-trigger</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>level0-slowdown-writes-trigger</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>level0-stop-writes-trigger</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-compaction-bytes</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>compaction-pri</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>dynamic-level-bytes</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>num-levels</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-bytes-for-level-multiplier</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>compaction-style</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>disable-auto-compactions</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>soft-pending-compaction-bytes-limit</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>hard-pending-compaction-bytes-limit</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>force-consistency-checks</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>prop-size-index-distance</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>prop-keys-index-distance</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-doubly-skiplist</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>titan</code></br>
<em>
<a href="#tikvtitancfconfig">
TiKVTitanCfConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvclient">TiKVClient</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>TiKVClient is the config for tikv client.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>grpc-connection-count</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>GrpcConnectionCount is the max gRPC connections that will be established
with each tikv-server.
Optional: Defaults to 16</p>
</td>
</tr>
<tr>
<td>
<code>grpc-keepalive-time</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>After a duration of this time in seconds if the client doesn&rsquo;t see any activity it pings
the server to see if the transport is still alive.
Optional: Defaults to 10</p>
</td>
</tr>
<tr>
<td>
<code>grpc-keepalive-timeout</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>After having pinged for keepalive check, the client waits for a duration of Timeout in seconds
and if no activity is seen even after that the connection is closed.
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>commit-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CommitTimeout is the max time which command &lsquo;commit&rsquo; will wait.
Optional: Defaults to 41s</p>
</td>
</tr>
<tr>
<td>
<code>max-txn-time-use</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0
MaxTxnTimeUse is the max time a Txn may use (in seconds) from its startTS to commitTS.
Optional: Defaults to 590</p>
</td>
</tr>
<tr>
<td>
<code>max-batch-size</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxBatchSize is the max batch size when calling batch commands API.
Optional: Defaults to 128</p>
</td>
</tr>
<tr>
<td>
<code>overload-threshold</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>If TiKV load is greater than this, TiDB will wait for a while to avoid little batch.
Optional: Defaults to 200</p>
</td>
</tr>
<tr>
<td>
<code>max-batch-wait-time</code></br>
<em>
time.Duration
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxBatchWaitTime in nanosecond is the max wait time for batch.
Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>batch-wait-size</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>BatchWaitSize is the max wait size for batch.
Optional: Defaults to 8</p>
</td>
</tr>
<tr>
<td>
<code>region-cache-ttl</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>If a Region has not been accessed for more than the given duration (in seconds), it
will be reloaded from the PD.
Optional: Defaults to 600</p>
</td>
</tr>
<tr>
<td>
<code>store-limit</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>If a store has been up to the limit, it will return error for successive request to
prevent the store occupying too much token in dispatching level.
Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>store-liveness-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>StoreLivenessTimeout is the timeout for store liveness check request.</p>
</td>
</tr>
<tr>
<td>
<code>copr-cache</code></br>
<em>
<a href="#coprocessorcache">
CoprocessorCache
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvconfig">TiKVConfig</h3>
<p>
<p>TiKVConfig is the configuration of TiKV.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>log-level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>log-format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>slow-log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>slow-log-threshold</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>log-rotation-timespan</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 24h</p>
</td>
</tr>
<tr>
<td>
<code>log-rotation-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>refresh-config-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>panic-when-unexpected-key-or-data</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#tikvserverconfig">
TiKVServerConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>storage</code></br>
<em>
<a href="#tikvstorageconfig">
TiKVStorageConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raftstore</code></br>
<em>
<a href="#tikvraftstoreconfig">
TiKVRaftstoreConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>rocksdb</code></br>
<em>
<a href="#tikvdbconfig">
TiKVDbConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>coprocessor</code></br>
<em>
<a href="#tikvcoprocessorconfig">
TiKVCoprocessorConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>readpool</code></br>
<em>
<a href="#tikvreadpoolconfig">
TiKVReadPoolConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raftdb</code></br>
<em>
<a href="#tikvraftdbconfig">
TiKVRaftDBConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>import</code></br>
<em>
<a href="#tikvimportconfig">
TiKVImportConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>gc</code></br>
<em>
<a href="#tikvgcconfig">
TiKVGCConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pd</code></br>
<em>
<a href="#tikvpdconfig">
TiKVPDConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>security</code></br>
<em>
<a href="#tikvsecurityconfig">
TiKVSecurityConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pessimistic-txn</code></br>
<em>
<a href="#tikvpessimistictxn">
TiKVPessimisticTxn
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>backup</code></br>
<em>
<a href="#tikvbackupconfig">
TiKVBackupConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvconfigwraper">TiKVConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvspec">TiKVSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvcoprocessorconfig">TiKVCoprocessorConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
<p>TiKVCoprocessorConfig is the configuration of TiKV Coprocessor component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>split-region-on-table</code></br>
<em>
bool
</em>
</td>
<td>
<p>When it is set to <code>true</code>, TiKV will try to split a Region with table prefix if that Region
crosses tables.
It is recommended to turn off this option if there will be a large number of tables created.
Optional: Defaults to false
optional</p>
</td>
</tr>
<tr>
<td>
<code>batch-split-limit</code></br>
<em>
int64
</em>
</td>
<td>
<p>One split check produces several split keys in batch. This config limits the number of produced
split keys in one batch.
optional</p>
</td>
</tr>
<tr>
<td>
<code>region-max-size</code></br>
<em>
string
</em>
</td>
<td>
<p>When Region [a,e) size exceeds <code>region-max-size</code>, it will be split into several Regions [a,b),
[b,c), [c,d), [d,e) and the size of [a,b), [b,c), [c,d) will be <code>region-split-size</code> (or a
little larger). See also: region-split-size
Optional: Defaults to 144MB
optional</p>
</td>
</tr>
<tr>
<td>
<code>region-split-size</code></br>
<em>
string
</em>
</td>
<td>
<p>When Region [a,e) size exceeds <code>region-max-size</code>, it will be split into several Regions [a,b),
[b,c), [c,d), [d,e) and the size of [a,b), [b,c), [c,d) will be <code>region-split-size</code> (or a
little larger). See also: region-max-size
Optional: Defaults to 96MB
optional</p>
</td>
</tr>
<tr>
<td>
<code>region-max-keys</code></br>
<em>
int64
</em>
</td>
<td>
<p>When the number of keys in Region [a,e) exceeds the <code>region-max-keys</code>, it will be split into
several Regions [a,b), [b,c), [c,d), [d,e) and the number of keys in [a,b), [b,c), [c,d) will be
<code>region-split-keys</code>. See also: region-split-keys
Optional: Defaults to 1440000
optional</p>
</td>
</tr>
<tr>
<td>
<code>region-split-keys</code></br>
<em>
int64
</em>
</td>
<td>
<p>When the number of keys in Region [a,e) exceeds the <code>region-max-keys</code>, it will be split into
several Regions [a,b), [b,c), [c,d), [d,e) and the number of keys in [a,b), [b,c), [c,d) will be
<code>region-split-keys</code>. See also: region-max-keys
Optional: Defaults to 960000
optional</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvcoprocessorreadpoolconfig">TiKVCoprocessorReadPoolConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvreadpoolconfig">TiKVReadPoolConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>high-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 8</p>
</td>
</tr>
<tr>
<td>
<code>normal-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 8</p>
</td>
</tr>
<tr>
<td>
<code>low-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 8</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker-high</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2000</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker-normal</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2000</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker-low</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2000</p>
</td>
</tr>
<tr>
<td>
<code>stack-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10MB</p>
</td>
</tr>
<tr>
<td>
<code>use-unified-pool</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvdbconfig">TiKVDbConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
<p>TiKVDbConfig is the rocksdb config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>wal-recovery-mode</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2</p>
</td>
</tr>
<tr>
<td>
<code>wal-ttl-seconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wal-size-limit</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-total-wal-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 4GB</p>
</td>
</tr>
<tr>
<td>
<code>max-background-jobs</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 8</p>
</td>
</tr>
<tr>
<td>
<code>max-manifest-file-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 128MB</p>
</td>
</tr>
<tr>
<td>
<code>create-if-missing</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>max-open-files</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 40960</p>
</td>
</tr>
<tr>
<td>
<code>enable-statistics</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>stats-dump-period</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10m</p>
</td>
</tr>
<tr>
<td>
<code>compaction-readahead-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>info-log-max-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-roll-time</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-keep-log-file-num</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-dir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>rate-bytes-per-sec</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>rate-limiter-mode</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>auto-tuned</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>bytes-per-sync</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wal-bytes-per-sync</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-sub-compactions</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>writable-file-max-buffer-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>use-direct-io-for-flush-and-compaction</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-pipelined-write</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>defaultcf</code></br>
<em>
<a href="#tikvcfconfig">
TiKVCfConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>writecf</code></br>
<em>
<a href="#tikvcfconfig">
TiKVCfConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>lockcf</code></br>
<em>
<a href="#tikvcfconfig">
TiKVCfConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raftcf</code></br>
<em>
<a href="#tikvcfconfig">
TiKVCfConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>titan</code></br>
<em>
<a href="#tikvtitandbconfig">
TiKVTitanDBConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvencryptionconfig">TiKVEncryptionConfig</h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>data-encryption-method</code></br>
<em>
string
</em>
</td>
<td>
<p>Encrypyion method, use data key encryption raw rocksdb data
Possible values: plaintext, aes128-ctr, aes192-ctr, aes256-ctr
Optional: Default to plaintext
optional</p>
</td>
</tr>
<tr>
<td>
<code>data-key-rotation-period</code></br>
<em>
string
</em>
</td>
<td>
<p>The frequency of datakey rotation, It managered by tikv
Optional: default to 7d
optional</p>
</td>
</tr>
<tr>
<td>
<code>master-key</code></br>
<em>
<a href="#tikvmasterkeyconfig">
TiKVMasterKeyConfig
</a>
</em>
</td>
<td>
<p>Master key config</p>
</td>
</tr>
<tr>
<td>
<code>previous-master-key</code></br>
<em>
<a href="#tikvmasterkeyconfig">
TiKVMasterKeyConfig
</a>
</em>
</td>
<td>
<p>Previous master key config
It used in master key rotation, the data key should decryption by previous master key and  then encrypytion by new master key</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvfailurestore">TiKVFailureStore</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvstatus">TiKVStatus</a>)
</p>
<p>
<p>TiKVFailureStore is the tikv failure store information</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>storeID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pvcUIDSet</code></br>
<em>
<a href="#emptystruct">
map[k8s.io/apimachinery/pkg/types.UID]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.EmptyStruct
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>storeDeleted</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>hostDown</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvgcconfig">TiKVGCConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>batch-keys</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 512</p>
</td>
</tr>
<tr>
<td>
<code>max-write-bytes-per-sec</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-compaction-filter</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>compaction-filter-skip-version-check</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvimportconfig">TiKVImportConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>import-dir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>num-threads</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>num-import-jobs</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>num-import-sst-jobs</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-prepare-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>region-split-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>stream-channel-window</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-open-engines</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>upload-speed-limit</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvmasterkeyconfig">TiKVMasterKeyConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvencryptionconfig">TiKVEncryptionConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<p>Use KMS encryption or use file encryption, possible values: kms, file
If set to kms, kms MasterKeyKMSConfig should be filled, if set to file MasterKeyFileConfig should be filled
optional</p>
</td>
</tr>
<tr>
<td>
<code>MasterKeyFileConfig</code></br>
<em>
<a href="#masterkeyfileconfig">
MasterKeyFileConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>MasterKeyFileConfig</code> are embedded into this type.)
</p>
<p>Master key file config
If the type set to file, this config should be filled</p>
</td>
</tr>
<tr>
<td>
<code>MasterKeyKMSConfig</code></br>
<em>
<a href="#masterkeykmsconfig">
MasterKeyKMSConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>MasterKeyKMSConfig</code> are embedded into this type.)
</p>
<p>Master key KMS config
If the type set to kms, this config should be filled</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvpdconfig">TiKVPDConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>endpoints</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The PD endpoints for the client.</p>
<p>Default is empty.</p>
</td>
</tr>
<tr>
<td>
<code>retry-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The interval at which to retry a PD connection initialization.</p>
<p>Default is 300ms.
Optional: Defaults to 300ms</p>
</td>
</tr>
<tr>
<td>
<code>retry-max-count</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The maximum number of times to retry a PD connection initialization.</p>
<p>Default is isize::MAX, represented by -1.
Optional: Defaults to -1</p>
</td>
</tr>
<tr>
<td>
<code>retry-log-every</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>If the client observes the same error message on retry, it can repeat the message only
every <code>n</code> times.</p>
<p>Default is 10. Set to 1 to disable this feature.
Optional: Defaults to 10</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvpessimistictxn">TiKVPessimisticTxn</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wait-for-lock-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The default and maximum delay before responding to TiDB when pessimistic
transactions encounter locks</p>
</td>
</tr>
<tr>
<td>
<code>wake-up-delay-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>If more than one transaction is waiting for the same lock, only the one with smallest
start timestamp will be waked up immediately when the lock is released. Others will
be waked up after <code>wake_up_delay_duration</code> to reduce contention and make the oldest
one more likely acquires the lock.</p>
</td>
</tr>
<tr>
<td>
<code>pipelined</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvraftdbconfig">TiKVRaftDBConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>wal-recovery-mode</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wal-dir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wal-ttl-seconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wal-size-limit</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-total-wal-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-background-jobs</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-manifest-file-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>create-if-missing</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-open-files</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-statistics</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>stats-dump-period</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>compaction-readahead-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-max-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-roll-time</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-keep-log-file-num</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>info-log-dir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-sub-compactions</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>writable-file-max-buffer-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>use-direct-io-for-flush-and-compaction</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-pipelined-write</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>allow-concurrent-memtable-write</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>bytes-per-sync</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>wal-bytes-per-sync</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>defaultcf</code></br>
<em>
<a href="#tikvcfconfig">
TiKVCfConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvraftstoreconfig">TiKVRaftstoreConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
<p>TiKVRaftstoreConfig is the configuration of TiKV raftstore component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>sync-log</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>true for high reliability, prevent data loss when power failure.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>prevote</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>raft-base-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>raft-base-tick-interval is a base tick interval (ms).</p>
</td>
</tr>
<tr>
<td>
<code>raft-heartbeat-ticks</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raft-election-timeout-ticks</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raft-entry-max-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>When the entry exceed the max size, reject to propose it.
Optional: Defaults to 8MB</p>
</td>
</tr>
<tr>
<td>
<code>raft-max-size-per-msg</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Limit the max size of each append message.
Optional: Defaults to 1MB</p>
</td>
</tr>
<tr>
<td>
<code>raft-max-inflight-msgs</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Limit the max number of in-flight append messages during optimistic
replication phase.
Optional: Defaults to 256</p>
</td>
</tr>
<tr>
<td>
<code>raft-log-gc-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval to gc unnecessary raft log (ms).
Optional: Defaults to 10s</p>
</td>
</tr>
<tr>
<td>
<code>raft-log-gc-threshold</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>A threshold to gc stale raft log, must &gt;= 1.
Optional: Defaults to 50</p>
</td>
</tr>
<tr>
<td>
<code>raft-log-gc-count-limit</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>When entry count exceed this value, gc will be forced trigger.
Optional: Defaults to 72000</p>
</td>
</tr>
<tr>
<td>
<code>raft-log-gc-size-limit</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>When the approximate size of raft log entries exceed this value
gc will be forced trigger.
Optional: Defaults to 72MB</p>
</td>
</tr>
<tr>
<td>
<code>raft-entry-cache-life-time</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>When a peer is not responding for this time, leader will not keep entry cache for it.</p>
</td>
</tr>
<tr>
<td>
<code>raft-reject-transfer-leader-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>When a peer is newly added, reject transferring leader to the peer for a while.</p>
</td>
</tr>
<tr>
<td>
<code>split-region-check-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval (ms) to check region whether need to be split or not.
Optional: Defaults to 10s</p>
</td>
</tr>
<tr>
<td>
<code>region-split-check-diff</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ When size change of region exceed the diff since last check, it
/ will be checked again whether it should be split.
Optional: Defaults to 6MB</p>
</td>
</tr>
<tr>
<td>
<code>region-compact-check-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Interval (ms) to check whether start compaction for a region.
Optional: Defaults to 5m</p>
</td>
</tr>
<tr>
<td>
<code>clean-stale-peer-delay</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>delay time before deleting a stale peer
Optional: Defaults to 10m</p>
</td>
</tr>
<tr>
<td>
<code>region-compact-check-step</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Number of regions for each time checking.
Optional: Defaults to 100</p>
</td>
</tr>
<tr>
<td>
<code>region-compact-min-tombstones</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Minimum number of tombstones to trigger manual compaction.
Optional: Defaults to 10000</p>
</td>
</tr>
<tr>
<td>
<code>region-compact-tombstones-percent</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Minimum percentage of tombstones to trigger manual compaction.
/ Should between 1 and 100.
Optional: Defaults to 30</p>
</td>
</tr>
<tr>
<td>
<code>pd-heartbeat-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 60s</p>
</td>
</tr>
<tr>
<td>
<code>pd-store-heartbeat-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10s</p>
</td>
</tr>
<tr>
<td>
<code>snap-mgr-gc-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>snap-gc-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>lock-cf-compact-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10m</p>
</td>
</tr>
<tr>
<td>
<code>lock-cf-compact-bytes-threshold</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 256MB</p>
</td>
</tr>
<tr>
<td>
<code>notify-capacity</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>messages-per-tick</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-peer-down-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ When a peer is not active for max-peer-down-duration
/ the peer is considered to be down and is reported to PD.
Optional: Defaults to 5m</p>
</td>
</tr>
<tr>
<td>
<code>max-leader-missing-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ If the leader of a peer is missing for longer than max-leader-missing-duration
/ the peer would ask pd to confirm whether it is valid in any region.
/ If the peer is stale and is not valid in any region, it will destroy itself.</p>
</td>
</tr>
<tr>
<td>
<code>abnormal-leader-missing-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Similar to the max-leader-missing-duration, instead it will log warnings and
/ try to alert monitoring systems, if there is any.</p>
</td>
</tr>
<tr>
<td>
<code>peer-stale-state-check-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>leader-transfer-max-log-lag</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>snap-apply-batch-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>consistency-check-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval (ms) to check region whether the data is consistent.
Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>report-region-flow-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>raft-store-max-leader-lease</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The lease provided by a successfully proposed and applied entry.</p>
</td>
</tr>
<tr>
<td>
<code>right-derive-when-split</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Right region derive origin region id when split.</p>
</td>
</tr>
<tr>
<td>
<code>allow-remove-leader</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>merge-max-log-gap</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Max log gap allowed to propose merge.</p>
</td>
</tr>
<tr>
<td>
<code>merge-check-tick-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>/ Interval to re-propose merge.</p>
</td>
</tr>
<tr>
<td>
<code>use-delete-range</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cleanup-import-sst-interval</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10m</p>
</td>
</tr>
<tr>
<td>
<code>apply-max-batch-size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>apply-pool-size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2</p>
</td>
</tr>
<tr>
<td>
<code>store-max-batch-size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>store-pool-size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2</p>
</td>
</tr>
<tr>
<td>
<code>store-reschedule-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3s</p>
</td>
</tr>
<tr>
<td>
<code>apply-yield-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 500ms</p>
</td>
</tr>
<tr>
<td>
<code>hibernate-regions</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>apply-early</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>perf-level</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 0</p>
</td>
</tr>
<tr>
<td>
<code>dev-assert</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to false</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvreadpoolconfig">TiKVReadPoolConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>unified</code></br>
<em>
<a href="#tikvunifiedreadpoolconfig">
TiKVUnifiedReadPoolConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>coprocessor</code></br>
<em>
<a href="#tikvcoprocessorreadpoolconfig">
TiKVCoprocessorReadPoolConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>storage</code></br>
<em>
<a href="#tikvstoragereadpoolconfig">
TiKVStorageReadPoolConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvsecurityconfig">TiKVSecurityConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ca-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cert-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>key-path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cert-allowed-cn</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertAllowedCN is the Common Name that allowed</p>
</td>
</tr>
<tr>
<td>
<code>override-ssl-target</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>cipher-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>encryption</code></br>
<em>
<a href="#tikvsecurityconfigencryption">
TiKVSecurityConfigEncryption
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvsecurityconfigencryption">TiKVSecurityConfigEncryption</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvsecurityconfig">TiKVSecurityConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>data-encryption-method</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Encryption method to use for data files.
Possible values are &ldquo;plaintext&rdquo;, &ldquo;aes128-ctr&rdquo;, &ldquo;aes192-ctr&rdquo; and &ldquo;aes256-ctr&rdquo;. Value other than
&ldquo;plaintext&rdquo; means encryption is enabled, in which case master key must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>data-key-rotation-period</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies how often TiKV rotates data encryption key.</p>
</td>
</tr>
<tr>
<td>
<code>master-key</code></br>
<em>
<a href="#tikvsecurityconfigencryptionmasterkey">
TiKVSecurityConfigEncryptionMasterKey
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies master key if encryption is enabled. There are three types of master key:</p>
<ul>
<li><p>&ldquo;plaintext&rdquo;:</p>
<p>Plaintext as master key means no master key is given and only applicable when
encryption is not enabled, i.e. data-encryption-method = &ldquo;plaintext&rdquo;. This type doesn&rsquo;t
have sub-config items. Example:</p>
<p>[security.encryption.master-key]
type = &ldquo;plaintext&rdquo;</p></li>
<li><p>&ldquo;kms&rdquo;:</p>
<p>Use a KMS service to supply master key. Currently only AWS KMS is supported. This type of
master key is recommended for production use. Example:</p>
<p>[security.encryption.master-key]
type = &ldquo;kms&rdquo;</p>
<h2>KMS CMK key id. Must be a valid KMS CMK where the TiKV process has access to.</h2>
<h2>In production is recommended to grant access of the CMK to TiKV using IAM.</h2>
<p>key-id = &ldquo;1234abcd-12ab-34cd-56ef-1234567890ab&rdquo;</p>
<h2>AWS region of the KMS CMK.</h2>
<p>region = &ldquo;us-west-2&rdquo;</p>
<h2>(Optional) AWS KMS service endpoint. Only required when non-default KMS endpoint is</h2>
<h2>desired.</h2>
<p>endpoint = &ldquo;<a href="https://kms.us-west-2.amazonaws.com&quot;">https://kms.us-west-2.amazonaws.com&rdquo;</a></p></li>
<li><p>&ldquo;file&rdquo;:</p>
<p>Supply a custom encryption key stored in a file. It is recommended NOT to use in production,
as it breaks the purpose of encryption at rest, unless the file is stored in tempfs.
The file must contain a 256-bits (32 bytes, regardless of key length implied by
data-encryption-method) key encoded as hex string and end with newline (&ldquo;\n&rdquo;). Example:</p>
<p>[security.encryption.master-key]
type = &ldquo;file&rdquo;
path = &ldquo;/path/to/master/key/file&rdquo;</p></li>
</ul>
</td>
</tr>
<tr>
<td>
<code>previous-master-key</code></br>
<em>
<a href="#tikvsecurityconfigencryptionpreviousmasterkey">
TiKVSecurityConfigEncryptionPreviousMasterKey
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies the old master key when rotating master key. Same config format as master-key.
The key is only access once during TiKV startup, after that TiKV do not need access to the key.
And it is okay to leave the stale previous-master-key config after master key rotation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvsecurityconfigencryptionmasterkey">TiKVSecurityConfigEncryptionMasterKey</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvsecurityconfigencryption">TiKVSecurityConfigEncryption</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>MasterKeyFileConfig</code></br>
<em>
<a href="#masterkeyfileconfig">
MasterKeyFileConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>MasterKeyFileConfig</code> are embedded into this type.)
</p>
<p>Master key file config
If the type set to file, this config should be filled</p>
</td>
</tr>
<tr>
<td>
<code>MasterKeyKMSConfig</code></br>
<em>
<a href="#masterkeykmsconfig">
MasterKeyKMSConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>MasterKeyKMSConfig</code> are embedded into this type.)
</p>
<p>Master key KMS config
If the type set to kms, this config should be filled</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvsecurityconfigencryptionpreviousmasterkey">TiKVSecurityConfigEncryptionPreviousMasterKey</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvsecurityconfigencryption">TiKVSecurityConfigEncryption</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>MasterKeyFileConfig</code></br>
<em>
<a href="#masterkeyfileconfig">
MasterKeyFileConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>MasterKeyFileConfig</code> are embedded into this type.)
</p>
<p>Master key file config
If the type set to file, this config should be filled</p>
</td>
</tr>
<tr>
<td>
<code>MasterKeyKMSConfig</code></br>
<em>
<a href="#masterkeykmsconfig">
MasterKeyKMSConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>MasterKeyKMSConfig</code> are embedded into this type.)
</p>
<p>Master key KMS config
If the type set to kms, this config should be filled</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvserverconfig">TiKVServerConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#flashserverconfig">FlashServerConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
<p>TiKVServerConfig is the configuration of TiKV server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>status-thread-pool-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 1</p>
</td>
</tr>
<tr>
<td>
<code>max-grpc-send-msg-len</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10485760</p>
</td>
</tr>
<tr>
<td>
<code>grpc-compression-type</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to none</p>
</td>
</tr>
<tr>
<td>
<code>grpc-concurrency</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 4</p>
</td>
</tr>
<tr>
<td>
<code>grpc-concurrent-stream</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 1024</p>
</td>
</tr>
<tr>
<td>
<code>grpc-memory-pool-quota</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 32G</p>
</td>
</tr>
<tr>
<td>
<code>grpc-raft-conn-num</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10</p>
</td>
</tr>
<tr>
<td>
<code>grpc-stream-initial-window-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2MB</p>
</td>
</tr>
<tr>
<td>
<code>grpc-keepalive-time</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10s</p>
</td>
</tr>
<tr>
<td>
<code>grpc-keepalive-timeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 3s</p>
</td>
</tr>
<tr>
<td>
<code>concurrent-send-snap-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 32</p>
</td>
</tr>
<tr>
<td>
<code>concurrent-recv-snap-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 32</p>
</td>
</tr>
<tr>
<td>
<code>end-point-recursion-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 1000</p>
</td>
</tr>
<tr>
<td>
<code>end-point-stream-channel-size</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>end-point-batch-row-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>end-point-stream-batch-row-limit</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>end-point-enable-batch-if-possible</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>end-point-request-max-handle-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>snap-max-write-bytes-per-sec</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 100MB</p>
</td>
</tr>
<tr>
<td>
<code>snap-max-total-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>stats-concurrency</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>heavy-load-threshold</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>heavy-load-wait-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 60s</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>enable-request-batch</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>request-batch-enable-cross-command</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>request-batch-wait-duration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvspec">TiKVSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TiKVSpec contains details of TiKV members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for tikv</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>privileged</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether create the TiKV container in privileged mode, it is highly discouraged to enable this in
critical environment.
Optional: defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>separateRocksDBLog</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether output the RocksDB log in a separate sidecar container
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>separateRaftLog</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether output the Raft log in a separate sidecar container
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>rocksDBLogVolumeName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional volume name configuration for rocksdb log.</p>
</td>
</tr>
<tr>
<td>
<code>raftLogVolumeName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional volume name configuration for raft log.</p>
</td>
</tr>
<tr>
<td>
<code>logTailer</code></br>
<em>
<a href="#logtailerspec">
LogTailerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LogTailer is the configurations of the log tailers for TiKV</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for TiKV data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>dataSubDir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subdirectory within the volume to store TiKV Data. By default, the data
is stored in the root directory of volume which is mounted at
/var/lib/tikv.
Specifying this will change the data directory to a subdirectory, e.g.
/var/lib/tikv/data if you set the value to &ldquo;data&rdquo;.
It&rsquo;s dangerous to change this value for a running cluster as it will
upgrade your cluster to use a new storage directory.
Defaults to &ldquo;&rdquo; (volume&rsquo;s root).</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#tikvconfigwraper">
TiKVConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of tikv-servers</p>
</td>
</tr>
<tr>
<td>
<code>recoverFailover</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>RecoverFailover indicates that Operator can recover the failed Pods</p>
</td>
</tr>
<tr>
<td>
<code>failover</code></br>
<em>
<a href="#failover">
Failover
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Failover is the configurations of failover</p>
</td>
</tr>
<tr>
<td>
<code>mountClusterClientSecret</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>MountClusterClientSecret indicates whether to mount <code>cluster-client-secret</code> to the Pod</p>
</td>
</tr>
<tr>
<td>
<code>evictLeaderTimeout</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>EvictLeaderTimeout indicates the timeout to evict tikv leader, in the format of Go Duration.
Defaults to 1500min</p>
</td>
</tr>
<tr>
<td>
<code>waitLeaderTransferBackTimeout</code></br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>WaitLeaderTransferBackTimeout indicates the timeout to wait for leader transfer back before
the next tikv upgrade.</p>
<p>Defaults to 400s</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageVolumes configure additional storage for TiKV pods.</p>
</td>
</tr>
<tr>
<td>
<code>storeLabels</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>StoreLabels configures additional labels for TiKV stores.</p>
</td>
</tr>
<tr>
<td>
<code>enableNamedStatusPort</code></br>
<em>
bool
</em>
</td>
<td>
<p>EnableNamedStatusPort enables status port(20180) in the Pod spec.
If you set it to <code>true</code> for an existing cluster, the TiKV cluster will be rolling updated.</p>
</td>
</tr>
<tr>
<td>
<code>scalePolicy</code></br>
<em>
<a href="#scalepolicy">
ScalePolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScalePolicy is the scale configuration for TiKV</p>
</td>
</tr>
<tr>
<td>
<code>spareVolReplaceReplicas</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>The default number of spare replicas to scale up when using VolumeReplace feature.
In multi-az deployments with topology spread constraints you may need to set this to number of zones to avoid
zone skew after volume replace (total replicas always whole multiples of zones).
Optional: Defaults to 1</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvstatus">TiKVStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>TiKVStatus is TiKV status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bootStrapped</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>stores</code></br>
<em>
<a href="#tikvstore">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiKVStore
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>peerStores</code></br>
<em>
<a href="#tikvstore">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiKVStore
</a>
</em>
</td>
<td>
<p>key: store id</p>
</td>
</tr>
<tr>
<td>
<code>tombstoneStores</code></br>
<em>
<a href="#tikvstore">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiKVStore
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureStores</code></br>
<em>
<a href="#tikvfailurestore">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiKVFailureStore
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failoverUID</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>evictLeader</code></br>
<em>
<a href="#evictleaderstatus">
map[string]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.EvictLeaderStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
<tr>
<td>
<code>volReplaceInProgress</code></br>
<em>
bool
</em>
</td>
<td>
<p>Indicates that a Volume replace using VolumeReplacing feature is in progress.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvstorageconfig">TiKVStorageConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#proxyconfig">ProxyConfig</a>, 
<a href="#tikvconfig">TiKVConfig</a>)
</p>
<p>
<p>TiKVStorageConfig is the config of storage</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>max-key-size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>scheduler-notify-capacity</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0</p>
</td>
</tr>
<tr>
<td>
<code>scheduler-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2048000</p>
</td>
</tr>
<tr>
<td>
<code>scheduler-worker-pool-size</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 4</p>
</td>
</tr>
<tr>
<td>
<code>scheduler-pending-write-threshold</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 100MB</p>
</td>
</tr>
<tr>
<td>
<code>block-cache</code></br>
<em>
<a href="#tikvblockcacheconfig">
TiKVBlockCacheConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>reserve-space</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The size of the temporary file that preoccupies the extra space when
TiKV is started. The name of temporary file is <code>space_placeholder_file</code>,
located in the <code>storage.data-dir</code> directory. When TiKV runs out of disk
space and cannot be started normally, you can delete this file as an
emergency intervention and set it to <code>0MB</code>. Default value is 2GB.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvstoragereadpoolconfig">TiKVStorageReadPoolConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvreadpoolconfig">TiKVReadPoolConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>high-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 4</p>
</td>
</tr>
<tr>
<td>
<code>normal-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 4</p>
</td>
</tr>
<tr>
<td>
<code>low-concurrency</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 4</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker-high</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2000</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker-normal</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2000</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker-low</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 2000</p>
</td>
</tr>
<tr>
<td>
<code>stack-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 10MB</p>
</td>
</tr>
<tr>
<td>
<code>use-unified-pool</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to true</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvstore">TiKVStore</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvstatus">TiKVStatus</a>)
</p>
<p>
<p>TiKVStores is either Up/Down/Offline/Tombstone</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>store id is also uint64, due to the same reason as pd id, we store id as string</p>
</td>
</tr>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ip</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>leaderCount</code></br>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>state</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the health transitioned from one to another.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
<tr>
<td>
<code>leaderCountBeforeUpgrade</code></br>
<em>
int32
</em>
</td>
<td>
<p>LeaderCountBeforeUpgrade records the leader count before upgrade.</p>
<p>It is set when evicting leader and used to wait for most leaders to transfer back after upgrade.
It is unset after leader transfer is completed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvtitancfconfig">TiKVTitanCfConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvcfconfig">TiKVCfConfig</a>)
</p>
<p>
<p>TiKVTitanCfConfig is the titian config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>min-blob-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>blob-file-compression</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>blob-cache-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>min-gc-batch-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-gc-batch-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>discardable-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>sample-ratio</code></br>
<em>
float64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>merge-small-file-threshold</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>blob-run-mode</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>level_merge</code></br>
<em>
bool
</em>
</td>
<td>
<p>optional</p>
</td>
</tr>
<tr>
<td>
<code>gc-merge-rewrite</code></br>
<em>
bool
</em>
</td>
<td>
<p>optional</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvtitandbconfig">TiKVTitanDBConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvdbconfig">TiKVDbConfig</a>)
</p>
<p>
<p>TiKVTitanDBConfig is the config a titian db.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>dirname</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>disable-gc</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-background-gc</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>purge-obsolete-files-period</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value of this field will be truncated to seconds.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvunifiedreadpoolconfig">TiKVUnifiedReadPoolConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvreadpoolconfig">TiKVReadPoolConfig</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>min-thread-count</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>max-thread-count</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>stack-size</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deprecated in v4.0.0</p>
</td>
</tr>
<tr>
<td>
<code>max-tasks-per-worker</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tiproxycertlayout">TiProxyCertLayout</h3>
<p>
(<em>Appears on:</em>
<a href="#tiproxyspec">TiProxySpec</a>)
</p>
<p>
</p>
<h3 id="tiproxyconfigwraper">TiProxyConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#tiproxyspec">TiProxySpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiproxymember">TiProxyMember</h3>
<p>
(<em>Appears on:</em>
<a href="#tiproxystatus">TiProxyStatus</a>)
</p>
<p>
<p>TiProxyMember is TiProxy member</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>health</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>info</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional healthinfo if it is healthy.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the health transitioned from one to another.
TODO: remove nullable, <a href="https://github.com/kubernetes/kubernetes/issues/86811">https://github.com/kubernetes/kubernetes/issues/86811</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiproxyspec">TiProxySpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TiProxySpec contains details of TiProxy members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account for TiProxy</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>sslEnableTiDB</code></br>
<em>
bool
</em>
</td>
<td>
<p>Whether enable SSL connection between tiproxy and TiDB server</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretName is the name of secret which stores tidb server client certificate
used by TiProxy to check health status.</p>
</td>
</tr>
<tr>
<td>
<code>certLayout</code></br>
<em>
<a href="#tiproxycertlayout">
TiProxyCertLayout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiProxyCertLayout is the certificate layout of TiProxy that determines how tidb-operator mount cert secrets
and how configure TLS configurations for tiproxy.</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#tiproxyconfigwraper">
TiProxyConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of tiproxy-servers</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageVolumes configure additional storage for TiProxy pods.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for TiProxy data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>serverLabels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>ServerLabels defines the server labels of the TiProxy.
Using both this field and config file to manage the labels is an undefined behavior.
Note these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
- region, topology.kubernetes.io/region
- zone, topology.kubernetes.io/zone
- host</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiproxystatus">TiProxyStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>TiProxyStatus is TiProxy status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
<a href="#tiproxymember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.TiProxyMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclustercondition">TidbClusterCondition</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterstatus">TidbClusterStatus</a>)
</p>
<p>
<p>TidbClusterCondition describes the state of a tidb cluster at a certain point.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#tidbclusterconditiontype">
TidbClusterConditionType
</a>
</em>
</td>
<td>
<p>Type of the condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>The last time this condition was updated.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclusterconditiontype">TidbClusterConditionType</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclustercondition">TidbClusterCondition</a>)
</p>
<p>
<p>TidbClusterConditionType represents a tidb cluster condition value.</p>
</p>
<h3 id="tidbclusterref">TidbClusterRef</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>, 
<a href="#tidbdashboardspec">TidbDashboardSpec</a>, 
<a href="#tidbinitializerspec">TidbInitializerSpec</a>, 
<a href="#tidbmonitorspec">TidbMonitorSpec</a>, 
<a href="#tidbngmonitoringspec">TidbNGMonitoringSpec</a>)
</p>
<p>
<p>TidbClusterRef reference to a TidbCluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace is the namespace that TidbCluster object locates,
default to the same namespace as TidbMonitor/TidbCluster/TidbNGMonitoring/TidbDashboard</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of TidbCluster object</p>
</td>
</tr>
<tr>
<td>
<code>clusterDomain</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterDomain is the domain of TidbCluster object</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclusterspec">TidbClusterSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbcluster">TidbCluster</a>)
</p>
<p>
<p>TidbClusterSpec describes the attributes that a user creates on a tidb cluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>discovery</code></br>
<em>
<a href="#discoveryspec">
DiscoverySpec
</a>
</em>
</td>
<td>
<p>Discovery spec</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify a Service Account</p>
</td>
</tr>
<tr>
<td>
<code>pd</code></br>
<em>
<a href="#pdspec">
PDSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PD cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>pdms</code></br>
<em>
<a href="#pdmsspec">
[]PDMSSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PDMS cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tidb</code></br>
<em>
<a href="#tidbspec">
TiDBSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiDB cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tikv</code></br>
<em>
<a href="#tikvspec">
TiKVSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiKV cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tiflash</code></br>
<em>
<a href="#tiflashspec">
TiFlashSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiFlash cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>ticdc</code></br>
<em>
<a href="#ticdcspec">
TiCDCSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiCDC cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>tiproxy</code></br>
<em>
<a href="#tiproxyspec">
TiProxySpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiProxy cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>pump</code></br>
<em>
<a href="#pumpspec">
PumpSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Pump cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>helper</code></br>
<em>
<a href="#helperspec">
HelperSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Helper spec</p>
</td>
</tr>
<tr>
<td>
<code>paused</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Indicates that the tidb cluster is paused and will not be processed by
the controller.</p>
</td>
</tr>
<tr>
<td>
<code>recoveryMode</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether RecoveryMode is enabled for TiDB cluster to restore
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiDB cluster version</p>
</td>
</tr>
<tr>
<td>
<code>schedulerName</code></br>
<em>
string
</em>
</td>
<td>
<p>SchedulerName of TiDB cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of TiDB cluster Pods</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>configUpdateStrategy</code></br>
<em>
<a href="#configupdatestrategy">
ConfigUpdateStrategy
</a>
</em>
</td>
<td>
<p>ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
cluster component is needed to reload the configuration change.
UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
related components to use the new ConfigMap, that is, the new configuration will be applied automatically.</p>
</td>
</tr>
<tr>
<td>
<code>enablePVReclaim</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>enablePVCReplace</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable PVC replace to recreate the PVC with different specs
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>tlsCluster</code></br>
<em>
<a href="#tlscluster">
TLSCluster
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether enable the TLS connection between TiDB server components
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>hostNetwork</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether Hostnetwork is enabled for TiDB cluster Pods
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of TiDB cluster Pods.
Will be overwritten by each cluster component&rsquo;s specific affinity setting, e.g. <code>spec.tidb.affinity</code></p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PriorityClassName of TiDB cluster Pods
Optional: Defaults to omitted</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base node selectors of TiDB cluster Pods, components may add or override selectors upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base annotations for TiDB cluster, all Pods in the cluster should have these annotations.
Can be overrode by annotations in the specific component spec.</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base labels for TiDB cluster, all Pods in the cluster should have these labels.
Can be overrode by labels in the specific component spec.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base tolerations of TiDB cluster Pods, components may add more tolerations upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>dnsConfig</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#poddnsconfig-v1-core">
Kubernetes core/v1.PodDNSConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSConfig Specifies the DNS parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>dnsPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#dnspolicy-v1-core">
Kubernetes core/v1.DNSPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSPolicy Specifies the DNSPolicy parameters of a pod.</p>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TiDB cluster Pods
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>services</code></br>
<em>
<a href="#service">
[]Service
</a>
</em>
</td>
<td>
<p>(Deprecated) Services list non-headless services type used in TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>enableDynamicConfiguration</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableDynamicConfiguration indicates whether to append <code>--advertise-status-addr</code> to the startup parameters of TiKV.</p>
</td>
</tr>
<tr>
<td>
<code>clusterDomain</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterDomain is the Kubernetes Cluster Domain of TiDB cluster
Optional: Defaults to &ldquo;&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>acrossK8s</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>AcrossK8s indicates whether deploy TiDB cluster across multiple Kubernetes clusters</p>
</td>
</tr>
<tr>
<td>
<code>cluster</code></br>
<em>
<a href="#tidbclusterref">
TidbClusterRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Cluster is the external cluster, if configured, the components in this TidbCluster will join to this configured cluster.</p>
</td>
</tr>
<tr>
<td>
<code>pdAddresses</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PDAddresses are the external PD addresses, if configured, the PDs in this TidbCluster will join to the configured PD cluster.</p>
</td>
</tr>
<tr>
<td>
<code>statefulSetUpdateStrategy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetupdatestrategytype-v1-apps">
Kubernetes apps/v1.StatefulSetUpdateStrategyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StatefulSetUpdateStrategy of TiDB cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>podManagementPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podmanagementpolicytype-v1-apps">
Kubernetes apps/v1.PodManagementPolicyType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodManagementPolicy of TiDB cluster StatefulSets</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code></br>
<em>
<a href="#topologyspreadconstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TopologySpreadConstraints describes how a group of pods ought to spread across topology
domains. Scheduler will schedule pods in a way which abides by the constraints.
This field is is only honored by clusters that enables the EvenPodsSpread feature.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>startScriptVersion</code></br>
<em>
<a href="#startscriptversion">
StartScriptVersion
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StartScriptVersion is the version of start script
When PD enables microservice mode, pd and pd microservice component will use start script v2.</p>
<p>default to &ldquo;v1&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>suspendAction</code></br>
<em>
<a href="#suspendaction">
SuspendAction
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SuspendAction defines the suspend actions for all component.</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
<tr>
<td>
<code>startScriptV2FeatureFlags</code></br>
<em>
<a href="#startscriptv2featureflag">
[]StartScriptV2FeatureFlag
</a>
</em>
</td>
<td>
<p>Feature flags used by v2 startup script to enable various features.
Examples of supported feature flags:
- WaitForDnsNameIpMatch indicates whether PD and TiKV has to wait until local IP address matches the one published to external DNS
- PreferPDAddressesOverDiscovery advises start script to use TidbClusterSpec.PDAddresses (if supplied) as argument for pd-server, tikv-server and tidb-server commands</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclusterstatus">TidbClusterStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbcluster">TidbCluster</a>)
</p>
<p>
<p>TidbClusterStatus represents the current status of a tidb cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clusterID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pd</code></br>
<em>
<a href="#pdstatus">
PDStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pdms</code></br>
<em>
<a href="#pdmsstatus">
map[string]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.PDMSStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tikv</code></br>
<em>
<a href="#tikvstatus">
TiKVStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tidb</code></br>
<em>
<a href="#tidbstatus">
TiDBStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pump</code></br>
<em>
<a href="#pumpstatus">
PumpStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tiflash</code></br>
<em>
<a href="#tiflashstatus">
TiFlashStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tiproxy</code></br>
<em>
<a href="#tiproxystatus">
TiProxyStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ticdc</code></br>
<em>
<a href="#ticdcstatus">
TiCDCStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#tidbclustercondition">
[]TidbClusterCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a tidb cluster&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbdashboard">TidbDashboard</h3>
<p>
<p>TidbDashboard contains the spec and status of tidb dashboard.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#tidbdashboardspec">
TidbDashboardSpec
</a>
</em>
</td>
<td>
<p>Spec contains all spec about tidb dashboard.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
<p>ComponentSpec is common spec.</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#tidbclusterref">
[]TidbClusterRef
</a>
</em>
</td>
<td>
<p>Clusters reference TiDB cluster.</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by tidb dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<p>Base image of the component (image tag is now allowed during validation).</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageClassName is the default PVC storage class for tidb dashboard.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<p>StorageVolumes configures additional PVC for tidb dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>pathPrefix</code></br>
<em>
string
</em>
</td>
<td>
<p>PathPrefix is public URL path prefix for reverse proxies.</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<p>Service defines a Kubernetes service of tidb dashboard web access.</p>
</td>
</tr>
<tr>
<td>
<code>telemetry</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Telemetry is whether to enable telemetry.
When enabled, usage data will be sent to PingCAP for improving user experience.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>experimental</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Experimental is whether to enable experimental features.
When enabled, experimental TiDB Dashboard features will be available.
These features are incomplete or not well tested. Suggest not to enable in
production.
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
<tr>
<td>
<code>listenOnLocalhostOnly</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ListenOnLocalhostOnly whether to expose dashboard to 0.0.0.0 or limit it to localhost only
which means it will be accessible only via port-forwarding
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>disableKeyVisualizer</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableKeyVisualizer is whether to disable Key Visualizer.
Optional: Defaults to false</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#tidbdashboardstatus">
TidbDashboardStatus
</a>
</em>
</td>
<td>
<p>Status is most recently observed status of tidb dashboard.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbdashboardspec">TidbDashboardSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbdashboard">TidbDashboard</a>)
</p>
<p>
<p>TidbDashboardSpec is spec of tidb dashboard.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
<p>ComponentSpec is common spec.</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#tidbclusterref">
[]TidbClusterRef
</a>
</em>
</td>
<td>
<p>Clusters reference TiDB cluster.</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by tidb dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<p>Base image of the component (image tag is now allowed during validation).</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>StorageClassName is the default PVC storage class for tidb dashboard.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageVolumes</code></br>
<em>
<a href="#storagevolume">
[]StorageVolume
</a>
</em>
</td>
<td>
<p>StorageVolumes configures additional PVC for tidb dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>pathPrefix</code></br>
<em>
string
</em>
</td>
<td>
<p>PathPrefix is public URL path prefix for reverse proxies.</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#servicespec">
ServiceSpec
</a>
</em>
</td>
<td>
<p>Service defines a Kubernetes service of tidb dashboard web access.</p>
</td>
</tr>
<tr>
<td>
<code>telemetry</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Telemetry is whether to enable telemetry.
When enabled, usage data will be sent to PingCAP for improving user experience.
Optional: Defaults to true</p>
</td>
</tr>
<tr>
<td>
<code>experimental</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Experimental is whether to enable experimental features.
When enabled, experimental TiDB Dashboard features will be available.
These features are incomplete or not well tested. Suggest not to enable in
production.
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
<tr>
<td>
<code>listenOnLocalhostOnly</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ListenOnLocalhostOnly whether to expose dashboard to 0.0.0.0 or limit it to localhost only
which means it will be accessible only via port-forwarding
Optional: Defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>disableKeyVisualizer</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DisableKeyVisualizer is whether to disable Key Visualizer.
Optional: Defaults to false</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbdashboardstatus">TidbDashboardStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbdashboard">TidbDashboard</a>)
</p>
<p>
<p>TidbDashboardStatus is status of tidb dashboard.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbinitializerspec">TidbInitializerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbinitializer">TidbInitializer</a>)
</p>
<p>
<p>TidbInitializer spec encode the desired state of tidb initializer Job</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cluster</code></br>
<em>
<a href="#tidbclusterref">
TidbClusterRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of the component</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>permitHost</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>permitHost is the host which will only be allowed to connect to the TiDB.</p>
</td>
</tr>
<tr>
<td>
<code>initSql</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InitSql is the SQL statements executed after the TiDB cluster is bootstrapped.</p>
</td>
</tr>
<tr>
<td>
<code>initSqlConfigMap</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InitSqlConfigMapName reference a configmap that provide init-sql, take high precedence than initSql if set</p>
</td>
</tr>
<tr>
<td>
<code>passwordSecret</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TiDB initializer Pods</p>
</td>
</tr>
<tr>
<td>
<code>tlsClientSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSClientSecretName is the name of secret which stores tidb server client certificate
Optional: Defaults to nil</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tolerations of the TiDB initializer Pod</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Node selectors of TiDB initializer Pod</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbinitializerstatus">TidbInitializerStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbinitializer">TidbInitializer</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>JobStatus</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#jobstatus-v1-batch">
Kubernetes batch/v1.JobStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>JobStatus</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#initializephase">
InitializePhase
</a>
</em>
</td>
<td>
<p>Phase is a user readable state inferred from the underlying Job status and TidbCluster status</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbmonitorspec">TidbMonitorSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitor">TidbMonitor</a>)
</p>
<p>
<p>TidbMonitor spec encode the desired state of tidb monitoring component</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#tidbclusterref">
[]TidbClusterRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>monitored TiDB cluster info</p>
</td>
</tr>
<tr>
<td>
<code>prometheus</code></br>
<em>
<a href="#prometheusspec">
PrometheusSpec
</a>
</em>
</td>
<td>
<p>Prometheus spec</p>
</td>
</tr>
<tr>
<td>
<code>grafana</code></br>
<em>
<a href="#grafanaspec">
GrafanaSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Grafana spec</p>
</td>
</tr>
<tr>
<td>
<code>reloader</code></br>
<em>
<a href="#reloaderspec">
ReloaderSpec
</a>
</em>
</td>
<td>
<p>Reloader spec</p>
</td>
</tr>
<tr>
<td>
<code>initializer</code></br>
<em>
<a href="#initializerspec">
InitializerSpec
</a>
</em>
</td>
<td>
<p>Initializer spec</p>
</td>
</tr>
<tr>
<td>
<code>dm</code></br>
<em>
<a href="#dmmonitorspec">
DMMonitorSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>monitored DM cluster spec</p>
</td>
</tr>
<tr>
<td>
<code>thanos</code></br>
<em>
<a href="#thanosspec">
ThanosSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Thanos spec</p>
</td>
</tr>
<tr>
<td>
<code>prometheusReloader</code></br>
<em>
<a href="#prometheusreloaderspec">
PrometheusReloaderSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PrometheusReloader set prometheus reloader configuration</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of TidbMonitor Pods</p>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.</p>
</td>
</tr>
<tr>
<td>
<code>persistent</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>If Persistent enabled, storageClassName must be set to an existing storage.
Defaults to false.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for TidbMonitor data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Size of the persistent volume.</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NodeSelector of the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations for the TidbMonitor.
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Labels for the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tolerations of the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>kubePrometheusURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>kubePrometheusURL is where tidb-monitoring get the  common metrics of kube-prometheus.
Ref: <a href="https://github.com/coreos/kube-prometheus">https://github.com/coreos/kube-prometheus</a></p>
</td>
</tr>
<tr>
<td>
<code>alertmanagerURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>alertmanagerURL is where tidb-monitoring push alerts to.
Ref: <a href="https://prometheus.io/docs/alerting/alertmanager/">https://prometheus.io/docs/alerting/alertmanager/</a></p>
</td>
</tr>
<tr>
<td>
<code>alertManagerRulesVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>alertManagerRulesVersion is the version of the tidb cluster that used for alert rules.
default to current tidb cluster version, for example: v3.0.15</p>
</td>
</tr>
<tr>
<td>
<code>additionalContainers</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core">
[]Kubernetes core/v1.Container
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional containers of the TidbMonitor.</p>
</td>
</tr>
<tr>
<td>
<code>clusterScoped</code></br>
<em>
bool
</em>
</td>
<td>
<p>ClusterScoped indicates whether this monitor should manage Kubernetes cluster-wide TiDB clusters</p>
</td>
</tr>
<tr>
<td>
<code>externalLabels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>The labels to add to any time series or alerts when communicating with
external systems (federation, remote storage, Alertmanager).</p>
</td>
</tr>
<tr>
<td>
<code>replicaExternalLabelName</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of Prometheus external label used to denote replica name.
Defaults to the value of <code>prometheus_replica</code>. External label will
<em>not</em> be added when value is set to empty string (<code>&quot;&quot;</code>).</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>Replicas is the number of desired replicas.
Defaults to 1.</p>
</td>
</tr>
<tr>
<td>
<code>shards</code></br>
<em>
int32
</em>
</td>
<td>
<p>EXPERIMENTAL: Number of shards to distribute targets onto. Number of
replicas multiplied by shards is the total number of Pods created. Note
that scaling down shards will not reshard data onto remaining instances,
it must be manually moved. Increasing shards will not reshard data
either but it will continue to be available from the same instances. To
query globally use Thanos sidecar and Thanos querier or remote write
data to a central location. Sharding is done on the content of the
<code>__address__</code> target meta-label.</p>
</td>
</tr>
<tr>
<td>
<code>additionalVolumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional volumes of TidbMonitor pod.</p>
</td>
</tr>
<tr>
<td>
<code>podSecurityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSecurityContext of TidbMonitor pod.</p>
</td>
</tr>
<tr>
<td>
<code>enableAlertRules</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableAlertRules adds alert rules to the Prometheus config even
if <code>AlertmanagerURL</code> is not configured.</p>
</td>
</tr>
<tr>
<td>
<code>timezone</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Time zone of TidbMonitor
Optional: Defaults to UTC</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbmonitorstatus">TidbMonitorStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitor">TidbMonitor</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>deploymentStorageStatus</code></br>
<em>
<a href="#deploymentstoragestatus">
DeploymentStorageStatus
</a>
</em>
</td>
<td>
<p>Storage status for deployment</p>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbngmonitoring">TidbNGMonitoring</h3>
<p>
<p>TidbNGMonitoring contains the spec and status of tidb ng monitor</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#tidbngmonitoringspec">
TidbNGMonitoringSpec
</a>
</em>
</td>
<td>
<p>Spec contains all spec about tidb ng monitor</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
<p>ComponentSpec is common spec.
NOTE: the same field will be overridden by component&rsquo;s spec.</p>
</td>
</tr>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#tidbclusterref">
[]TidbClusterRef
</a>
</em>
</td>
<td>
<p>Clusters reference TiDB cluster</p>
</td>
</tr>
<tr>
<td>
<code>paused</code></br>
<em>
bool
</em>
</td>
<td>
<p>Paused pause controller if it is true</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by tidb ng monitoring</p>
</td>
</tr>
<tr>
<td>
<code>clusterDomain</code></br>
<em>
string
</em>
</td>
<td>
<p>ClusterDomain is the Kubernetes Cluster Domain of tidb ng monitoring</p>
</td>
</tr>
<tr>
<td>
<code>ngMonitoring</code></br>
<em>
<a href="#ngmonitoringspec">
NGMonitoringSpec
</a>
</em>
</td>
<td>
<p>NGMonitoring is spec of ng monitoring</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#tidbngmonitoringstatus">
TidbNGMonitoringStatus
</a>
</em>
</td>
<td>
<p>Status is most recently observed status of tidb ng monitor</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbngmonitoringspec">TidbNGMonitoringSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbngmonitoring">TidbNGMonitoring</a>)
</p>
<p>
<p>TidbNGMonitoringSpec is spec of tidb ng monitoring</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
<p>ComponentSpec is common spec.
NOTE: the same field will be overridden by component&rsquo;s spec.</p>
</td>
</tr>
<tr>
<td>
<code>clusters</code></br>
<em>
<a href="#tidbclusterref">
[]TidbClusterRef
</a>
</em>
</td>
<td>
<p>Clusters reference TiDB cluster</p>
</td>
</tr>
<tr>
<td>
<code>paused</code></br>
<em>
bool
</em>
</td>
<td>
<p>Paused pause controller if it is true</p>
</td>
</tr>
<tr>
<td>
<code>pvReclaimPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#persistentvolumereclaimpolicy-v1-core">
Kubernetes core/v1.PersistentVolumeReclaimPolicy
</a>
</em>
</td>
<td>
<p>Persistent volume reclaim policy applied to the PVs that consumed by tidb ng monitoring</p>
</td>
</tr>
<tr>
<td>
<code>clusterDomain</code></br>
<em>
string
</em>
</td>
<td>
<p>ClusterDomain is the Kubernetes Cluster Domain of tidb ng monitoring</p>
</td>
</tr>
<tr>
<td>
<code>ngMonitoring</code></br>
<em>
<a href="#ngmonitoringspec">
NGMonitoringSpec
</a>
</em>
</td>
<td>
<p>NGMonitoring is spec of ng monitoring</p>
</td>
</tr>
<tr>
<td>
<code>preferIPv6</code></br>
<em>
bool
</em>
</td>
<td>
<p>PreferIPv6 indicates whether to prefer IPv6 addresses for all components.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbngmonitoringstatus">TidbNGMonitoringStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbngmonitoring">TidbNGMonitoring</a>)
</p>
<p>
<p>TidbNGMonitoringStatus is status of tidb ng monitoring</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ngMonitoring</code></br>
<em>
<a href="#ngmonitoringstatus">
NGMonitoringStatus
</a>
</em>
</td>
<td>
<p>NGMonitoring is status of ng monitoring</p>
</td>
</tr>
</tbody>
</table>
<h3 id="topologyspreadconstraint">TopologySpreadConstraint</h3>
<p>
(<em>Appears on:</em>
<a href="#componentspec">ComponentSpec</a>, 
<a href="#dmclusterspec">DMClusterSpec</a>, 
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TopologySpreadConstraint specifies how to spread matching pods among the given topology.
It is a minimal version of corev1.TopologySpreadConstraint to avoid to add too many fields of API
Refer to <a href="https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints">https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>topologyKey</code></br>
<em>
string
</em>
</td>
<td>
<p>TopologyKey is the key of node labels. Nodes that have a label with this key
and identical values are considered to be in the same topology.
We consider each <key, value> as a &ldquo;bucket&rdquo;, and try to put balanced number
of pods into each bucket.
WhenUnsatisfiable is default set to DoNotSchedule
LabelSelector is generated by component type
See pkg/apis/pingcap/v1alpha1/tidbcluster_component.go#TopologySpreadConstraints()</p>
</td>
</tr>
<tr>
<td>
<code>maxSkew</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxSkew describes the degree to which pods may be unevenly distributed.
When <code>whenUnsatisfiable=DoNotSchedule</code>, it is the maximum permitted difference
between the number of matching pods in the target topology and the global minimum.
The global minimum is the minimum number of matching pods in an eligible domain
or zero if the number of eligible domains is less than MinDomains.
For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same
labelSelector spread as 2/2/1:
In this case, the global minimum is 1.
| zone1 | zone2 | zone3 |
|  P P  |  P P  |   P   |
- if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 2/2/2;
scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2)
violate MaxSkew(1).
- if MaxSkew is 2, incoming pod can be scheduled onto any zone.
When <code>whenUnsatisfiable=ScheduleAnyway</code>, it is used to give higher precedence
to topologies that satisfy it.
Default value is 1.</p>
</td>
</tr>
<tr>
<td>
<code>minDomains</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MinDomains indicates a minimum number of eligible domains.
When the number of eligible domains with matching topology keys is less than minDomains,
Pod Topology Spread treats &ldquo;global minimum&rdquo; as 0, and then the calculation of Skew is performed.
And when the number of eligible domains with matching topology keys equals or greater than minDomains,
this value has no effect on scheduling.
As a result, when the number of eligible domains is less than minDomains,
scheduler won&rsquo;t schedule more than maxSkew Pods to those domains.
If value is nil, the constraint behaves as if MinDomains is equal to 1.
Valid values are integers greater than 0.
When value is not nil, WhenUnsatisfiable must be DoNotSchedule.</p>
<p>For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same
labelSelector spread as 2/2/2:
| zone1 | zone2 | zone3 |
|  P P  |  P P  |  P P  |
The number of domains is less than 5(MinDomains), so &ldquo;global minimum&rdquo; is treated as 0.
In this situation, new pod with the same labelSelector cannot be scheduled,
because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones,
it will violate MaxSkew.</p>
</td>
</tr>
<tr>
<td>
<code>nodeAffinityPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#nodeinclusionpolicy-v1-core">
Kubernetes core/v1.NodeInclusionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NodeAffinityPolicy indicates how we will treat Pod&rsquo;s nodeAffinity/nodeSelector
when calculating pod topology spread skew. Options are:
- Honor: only nodes matching nodeAffinity/nodeSelector are included in the calculations.
- Ignore: nodeAffinity/nodeSelector are ignored. All nodes are included in the calculations.</p>
<p>If this value is nil, the behavior is equivalent to the Honor policy.</p>
</td>
</tr>
<tr>
<td>
<code>matchLabels</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/label.Label
</em>
</td>
<td>
<em>(Optional)</em>
<p>MatchLabels is used to overwrite generated corev1.TopologySpreadConstraints.LabelSelector
corev1.TopologySpreadConstraint generated in component_spec.go will set a
LabelSelector automatically with some KV.
Historically, it is l[&ldquo;comp&rdquo;] = &ldquo;&rdquo; for component tiproxy. And we will use
MatchLabels to keep l[&ldquo;comp&rdquo;] = &ldquo;&rdquo; for old clusters with tiproxy</p>
</td>
</tr>
</tbody>
</table>
<h3 id="txnlocallatches">TxnLocalLatches</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbconfig">TiDBConfig</a>)
</p>
<p>
<p>TxnLocalLatches is the TxnLocalLatches section of the config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="unjoinedmember">UnjoinedMember</h3>
<p>
(<em>Appears on:</em>
<a href="#masterstatus">MasterStatus</a>, 
<a href="#pdstatus">PDStatus</a>)
</p>
<p>
<p>UnjoinedMember is the pd unjoin cluster member information</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pvcUID</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pvcUIDSet</code></br>
<em>
<a href="#emptystruct">
map[k8s.io/apimachinery/pkg/types.UID]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.EmptyStruct
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="user">User</h3>
<p>
<p>User is the configuration of users.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>password</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>profile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>quota</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>networks</code></br>
<em>
<a href="#networks">
Networks
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="workerconfig">WorkerConfig</h3>
<p>
<p>WorkerConfig is the configuration of dm-worker-server</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>log-level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log level.
Optional: Defaults to info</p>
</td>
</tr>
<tr>
<td>
<code>log-file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>File log config.</p>
</td>
</tr>
<tr>
<td>
<code>log-format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Log format. one of json or text.</p>
</td>
</tr>
<tr>
<td>
<code>keepalive-ttl</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeepAliveTTL is the keepalive TTL when dm-worker writes to the embedded etcd of dm-master
Optional: Defaults to 10</p>
</td>
</tr>
<tr>
<td>
<code>DMSecurityConfig</code></br>
<em>
<a href="#dmsecurityconfig">
DMSecurityConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>DMSecurityConfig</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>dm-worker&rsquo;s security config</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workerconfigwraper">WorkerConfigWraper</h3>
<p>
(<em>Appears on:</em>
<a href="#workerspec">WorkerSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/apis/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workerfailuremember">WorkerFailureMember</h3>
<p>
(<em>Appears on:</em>
<a href="#workerstatus">WorkerStatus</a>)
</p>
<p>
<p>WorkerFailureMember is the dm-worker failure member information</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workermember">WorkerMember</h3>
<p>
(<em>Appears on:</em>
<a href="#workerstatus">WorkerStatus</a>)
</p>
<p>
<p>WorkerMember is dm-worker member status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>addr</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>stage</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the health transitioned from one to another.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workerspec">WorkerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterspec">DMClusterSpec</a>)
</p>
<p>
<p>WorkerSpec contains details of dm-worker members</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ComponentSpec</code></br>
<em>
<a href="#componentspec">
ComponentSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>ComponentSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>ResourceRequirements</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceRequirements</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>The desired ready replicas</p>
</td>
</tr>
<tr>
<td>
<code>baseImage</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base image of the component, image tag is now allowed during validation</p>
</td>
</tr>
<tr>
<td>
<code>maxFailoverCount</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
Optional: Defaults to 3</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The storageClassName of the persistent volume for dm-worker data storage.
Defaults to Kubernetes default storage class.</p>
</td>
</tr>
<tr>
<td>
<code>storageSize</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageSize is the request storage size for dm-worker.
Defaults to &ldquo;10Gi&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>dataSubDir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subdirectory within the volume to store dm-worker Data. By default, the data
is stored in the root directory of volume which is mounted at
/var/lib/dm-worker.
Specifying this will change the data directory to a subdirectory, e.g.
/var/lib/dm-worker/data if you set the value to &ldquo;data&rdquo;.
It&rsquo;s dangerous to change this value for a running cluster as it will
upgrade your cluster to use a new storage directory.
Defaults to &ldquo;&rdquo; (volume&rsquo;s root).</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#workerconfigwraper">
WorkerConfigWraper
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of dm-worker-servers</p>
</td>
</tr>
<tr>
<td>
<code>recoverFailover</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>RecoverFailover indicates that Operator can recover the failover Pods</p>
</td>
</tr>
<tr>
<td>
<code>failover</code></br>
<em>
<a href="#failover">
Failover
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Failover is the configurations of failover</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workerstatus">WorkerStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#dmclusterstatus">DMClusterStatus</a>)
</p>
<p>
<p>WorkerStatus is dm-worker status</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>synced</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#memberphase">
MemberPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>members</code></br>
<em>
<a href="#workermember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.WorkerMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failureMembers</code></br>
<em>
<a href="#workerfailuremember">
map[string]github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.WorkerFailureMember
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>failoverUID</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="#storagevolumestatus">
map[github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeName]*github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.StorageVolumeStatus
</a>
</em>
</td>
<td>
<p>Volumes contains the status of all volumes.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Represents the latest available observations of a component&rsquo;s state.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>
