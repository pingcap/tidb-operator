---
title: TiDB Operator API Document
summary: Reference of TiDB Operator API
category: how-to
aliases: ['/docs/tidb-in-kubernetes/dev/api-references/']
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
<a href="#restore">Restore</a>
</li><li>
<a href="#tidbcluster">TidbCluster</a>
</li><li>
<a href="#tidbclusterautoscaler">TidbClusterAutoScaler</a>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
<p>Type is the backup type for tidb cluster.</p>
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
<p>StorageProvider configures where and how backups should be stored.</p>
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
<p>BRConfig is the configs for BR</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
0 is magic number to indicate un-limited backups.</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
<p>Type is the backup type for tidb cluster.</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
<code>pd</code></br>
<em>
<a href="#pdspec">
PDSpec
</a>
</em>
</td>
<td>
<p>PD cluster spec</p>
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
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TODO: remove optional after defaulting logic introduced
TiDB cluster version</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#persistentvolumereclaimpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of TiDB cluster Pods</p>
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
<p>Base annotations of TiDB cluster Pods, components may add or override selectors upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<p>Services list non-headless services type used in TidbCluster
Deprecated</p>
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
<h3 id="tidbclusterautoscaler">TidbClusterAutoScaler</h3>
<p>
<p>TidbClusterAutoScaler is the control script&rsquo;s spec</p>
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
<td><code>TidbClusterAutoScaler</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
<a href="#tidbclusterautoscalerspec">
TidbClusterAutoScalerSpec
</a>
</em>
</td>
<td>
<p>Spec describes the state of the TidbClusterAutoScaler</p>
<br/>
<br/>
<table>
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
<p>TidbClusterRef describe the target TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>metricsUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>We used prometheus to fetch the metrics resources until the pd could provide it.
MetricsUrl represents the url to fetch the metrics info</p>
</td>
</tr>
<tr>
<td>
<code>monitor</code></br>
<em>
<a href="#tidbmonitorref">
TidbMonitorRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TidbMonitorRef describe the target TidbMonitor, when MetricsUrl and Monitor are both set,
Operator will use MetricsUrl</p>
</td>
</tr>
<tr>
<td>
<code>tikv</code></br>
<em>
<a href="#tikvautoscalerspec">
TikvAutoScalerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiKV represents the auto-scaling spec for tikv</p>
</td>
</tr>
<tr>
<td>
<code>tidb</code></br>
<em>
<a href="#tidbautoscalerspec">
TidbAutoScalerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiDB represents the auto-scaling spec for tidb</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#tidbclusterautosclaerstatus">
TidbClusterAutoSclaerStatus
</a>
</em>
</td>
<td>
<p>Status describe the status of the TidbClusterAutoScaler</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
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
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
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
<h3 id="brconfig">BRConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#backupspec">BackupSpec</a>, 
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
<p>DB is the specific DB which will be backed-up or restored</p>
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
<p>Table is the specific table which will be backed-up or restored</p>
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
</tbody>
</table>
<h3 id="backupcondition">BackupCondition</h3>
<p>
(<em>Appears on:</em>
<a href="#backupstatus">BackupStatus</a>)
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#conditionstatus-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<a href="#backupcondition">BackupCondition</a>)
</p>
<p>
<p>BackupConditionType represents a valid condition of a Backup.</p>
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
0 is magic number to indicate un-limited backups.</p>
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
<code>lastBackupTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<code>allBackupCleanTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<p>Type is the backup type for tidb cluster.</p>
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
<p>StorageProvider configures where and how backups should be stored.</p>
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
<p>BRConfig is the configs for BR</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeStarted is the time at which the backup was started.</p>
</td>
</tr>
<tr>
<td>
<code>timeCompleted</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>TimeCompleted is the time at which the backup was completed.</p>
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
<h3 id="basicautoscalerspec">BasicAutoScalerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbautoscalerspec">TidbAutoScalerSpec</a>, 
<a href="#tikvautoscalerspec">TikvAutoScalerSpec</a>)
</p>
<p>
<p>BasicAutoScalerSpec describes the basic spec for auto-scaling</p>
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
<code>maxReplicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale out.
It cannot be less than minReplicas.</p>
</td>
</tr>
<tr>
<td>
<code>minReplicas</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>minReplicas is the lower limit for the number of replicas to which the autoscaler
can scale down.  It defaults to 1 pod. Scaling is active as long as at least one metric value is
available.</p>
</td>
</tr>
<tr>
<td>
<code>scaleInIntervalSeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScaleInIntervalSeconds represents the duration seconds between each auto-scaling-in
If not set, the default ScaleInIntervalSeconds will be set to 500</p>
</td>
</tr>
<tr>
<td>
<code>scaleOutIntervalSeconds</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScaleOutIntervalSeconds represents the duration seconds between each auto-scaling-out
If not set, the default ScaleOutIntervalSeconds will be set to 300</p>
</td>
</tr>
<tr>
<td>
<code>metrics</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#metricspec-v2beta2-autoscaling">
[]Kubernetes autoscaling/v2beta2.MetricSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>metrics contains the specifications for which to use to calculate the
desired replica count (the maximum replica count across all metrics will
be used).  The desired replica count is calculated multiplying the
ratio between the target value and the current value by the current
number of pods.  Ergo, metrics used must decrease as the pod count is
increased, and vice-versa.  See the individual metric source types for
more information about how each type of metric must respond.
If not set, the default metric will be set to 80% average CPU utilization.</p>
</td>
</tr>
<tr>
<td>
<code>metricsTimeDuration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>MetricsTimeDuration describe the Time duration to be queried in the Prometheus</p>
</td>
</tr>
<tr>
<td>
<code>scaleOutThreshold</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScaleOutThreshold describe the consecutive threshold for the auto-scaling,
if the consecutive counts of the scale-out result in auto-scaling reach this number,
the auto-scaling would be performed.
If not set, the default value is 3.</p>
</td>
</tr>
<tr>
<td>
<code>scaleInThreshold</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>ScaleInThreshold describe the consecutive threshold for the auto-scaling,
if the consecutive counts of the scale-in result in auto-scaling reach this number,
the auto-scaling would be performed.
If not set, the default value is 5.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="basicautoscalerstatus">BasicAutoScalerStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbautoscalerstatus">TidbAutoScalerStatus</a>, 
<a href="#tikvautoscalerstatus">TikvAutoScalerStatus</a>)
</p>
<p>
<p>BasicAutoScalerStatus describe the basic auto-scaling status</p>
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
<code>metrics</code></br>
<em>
<a href="#metricsstatus">
[]MetricsStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MetricsStatusList describes the metrics status in the last auto-scaling reconciliation</p>
</td>
</tr>
<tr>
<td>
<code>currentReplicas</code></br>
<em>
int32
</em>
</td>
<td>
<p>CurrentReplicas describes the current replicas for the component(tidb/tikv)</p>
</td>
</tr>
<tr>
<td>
<code>recommendedReplicas</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>RecommendedReplicas describes the calculated replicas in the last auto-scaling reconciliation for the component(tidb/tikv)</p>
</td>
</tr>
<tr>
<td>
<code>lastAutoScalingTimestamp</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastAutoScalingTimestamp describes the last auto-scaling timestamp for the component(tidb/tikv)</p>
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
<a href="#pdspec">PDSpec</a>, 
<a href="#pumpspec">PumpSpec</a>, 
<a href="#tidbspec">TiDBSpec</a>, 
<a href="#tiflashspec">TiFlashSpec</a>, 
<a href="#tikvspec">TiKVSpec</a>)
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
<p>Image of the component, override baseImage and version if present
Deprecated</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of the component. Override the cluster-level one if present
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
<p>Annotations of the component. Merged into the cluster-level annotations if non-empty
Optional: Defaults to cluster-level setting</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#podsecuritycontext-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container, like
v1.Container.Env.
Note that following env names cannot be used and may be overrided by
tidb-operator built envs.
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
</tbody>
</table>
<h3 id="configupdatestrategy">ConfigUpdateStrategy</h3>
<p>
(<em>Appears on:</em>
<a href="#componentspec">ComponentSpec</a>, 
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>ConfigUpdateStrategy represents the strategy to update configuration</p>
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
<code>enabled</code></br>
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
[]k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1.CustomResourceColumnDefinition
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
<code>TidbClusterAutoScaler</code></br>
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
<code>tidb_cacert_path</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tidb_cert_path</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tidb_key_path</code></br>
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
imported from TiDB v3.1.0</p>
</td>
</tr>
</tbody>
</table>
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
<p>Is log rotate enabled.</p>
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
gcs service account credentials JSON .</p>
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
<tr>
<td>
<code>username</code></br>
<em>
string
</em>
</td>
<td>
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
Optional: Defaults to busybox:1.26.2</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
<p>Disable automatic timestamps in output.</p>
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
<h3 id="logtailerspec">LogTailerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>)
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<h3 id="masterkeyfileconfig">MasterKeyFileConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvmasterkeyconfig">TiKVMasterKeyConfig</a>)
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
</tbody>
</table>
<h3 id="masterkeykmsconfig">MasterKeyKMSConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tikvmasterkeyconfig">TiKVMasterKeyConfig</a>)
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
<h3 id="memberphase">MemberPhase</h3>
<p>
(<em>Appears on:</em>
<a href="#pdstatus">PDStatus</a>, 
<a href="#pumpstatus">PumpStatus</a>, 
<a href="#tidbstatus">TiDBStatus</a>, 
<a href="#tikvstatus">TiKVStatus</a>)
</p>
<p>
<p>MemberPhase is the current state of member</p>
</p>
<h3 id="membertype">MemberType</h3>
<p>
<p>MemberType represents member type</p>
</p>
<h3 id="metricsstatus">MetricsStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#basicautoscalerstatus">BasicAutoScalerStatus</a>)
</p>
<p>
<p>MetricsStatus describe the basic metrics status in the last auto-scaling reconciliation</p>
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
<p>Name indicates the metrics name</p>
</td>
</tr>
<tr>
<td>
<code>currentValue</code></br>
<em>
string
</em>
</td>
<td>
<p>CurrentValue indicates the value calculated in the last auto-scaling reconciliation</p>
</td>
</tr>
<tr>
<td>
<code>thresholdValue</code></br>
<em>
string
</em>
</td>
<td>
<p>TargetValue indicates the threshold value for this metrics in auto-scaling</p>
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
<a href="#prometheusspec">PrometheusSpec</a>, 
<a href="#reloaderspec">ReloaderSpec</a>)
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
<code>Resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>
(Members of <code>Resources</code> are embedded into this type.)
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
(<em>Appears on:</em>
<a href="#pdspec">PDSpec</a>)
</p>
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
<p>NamespaceClassifier is for classifying stores/regions into different
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<code>strictly-match-label,string</code></br>
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
<code>enable-placement-rules,string</code></br>
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
<code>disable-raft-learner,string</code></br>
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
<code>disable-remove-down-replica,string</code></br>
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
<code>disable-replace-offline-replica,string</code></br>
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
<code>disable-make-up-replica,string</code></br>
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
<code>disable-remove-extra-replica,string</code></br>
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
<code>disable-location-replacement,string</code></br>
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
<code>disable-namespace-relocation,string</code></br>
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
<code>enable-one-way-merge,string</code></br>
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
<code>enable-cross-table-merge,string</code></br>
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
<p>CAPath is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn&rsquo;t be empty</p>
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
<code>use-region-storage,string</code></br>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<p>TODO: remove optional after defaulting introduced
Base image of the component, image tag is now allowed during validation</p>
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
<code>config</code></br>
<em>
<a href="#pdconfig">
PDConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of pd-servers</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#statefulsetstatus-v1-apps">
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
<p>Optional: Defaults to 1024</p>
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
<code>txn-entry-count-limit</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional: Defaults to 300000</p>
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
<tr>
<td>
<code>reserveDays</code></br>
<em>
int
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<p>TODO: remove optional after defaulting introduced
Base image of the component, image tag is now allowed during validation</p>
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
<code>GenericConfig</code></br>
<em>
github.com/pingcap/tidb-operator/pkg/util/config.GenericConfig
</em>
</td>
<td>
<p>
(Members of <code>GenericConfig</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>TODO: add schema
The configuration of Pump cluster.</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#statefulsetstatus-v1-apps">
Kubernetes apps/v1.StatefulSetStatus
</a>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#conditionstatus-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<a href="#restorecondition">RestoreCondition</a>)
</p>
<p>
<p>RestoreConditionType represents a valid condition of a Restore.</p>
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
<p>Type is the backup type for tidb cluster.</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
</tbody>
</table>
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
<p>Prefix for the keys.</p>
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
<p>Deprecated
Service represent service type used in TidbCluster</p>
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
<a href="#pdspec">PDSpec</a>, 
<a href="#prometheusspec">PrometheusSpec</a>, 
<a href="#reloaderspec">ReloaderSpec</a>, 
<a href="#tidbservicespec">TiDBServiceSpec</a>)
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#servicetype-v1-core">
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
<p>Additional annotations of the kubernetes service object</p>
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
</tbody>
</table>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<a href="#backupspec">BackupSpec</a>, 
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
</tbody>
</table>
<h3 id="tlscluster">TLSCluster</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterspec">TidbClusterSpec</a>)
</p>
<p>
<p>TLSCluster can enable TLS connection between TiDB server components
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
<p>Enable mutual TLS authentication among TiDB components
Once enabled, the mutual authentication applies to all components,
and it does not support applying to only part of the components.
The steps to enable this feature:
1. Generate TiDB server components certificates and a client-side certifiacete for them.
There are multiple ways to generate these certificates:
- user-provided certificates: <a href="https://pingcap.com/docs/stable/how-to/secure/generate-self-signed-certificates/">https://pingcap.com/docs/stable/how-to/secure/generate-self-signed-certificates/</a>
- use the K8s built-in certificate signing system signed certificates: <a href="https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/">https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/</a>
- or use cert-manager signed certificates: <a href="https://cert-manager.io/">https://cert-manager.io/</a>
2. Create one secret object for one component which contains the certificates created above.
The name of this Secret must be: ${cluster_name}-<componentName>-cluster-secret.
For PD: kubectl create secret generic ${cluster_name}-pd-cluster-secret &ndash;namespace=${namespace} &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
For TiKV: kubectl create secret generic ${cluster_name}-tikv-cluster-secret &ndash;namespace=${namespace} &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
For TiDB: kubectl create secret generic ${cluster_name}-tidb-cluster-secret &ndash;namespace=${namespace} &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
For Client: kubectl create secret generic ${cluster_name}-cluster-client-secret &ndash;namespace=${namespace} &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
Same for other components.</p>
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
</tbody>
</table>
<h3 id="tidbconfig">TiDBConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbspec">TiDBSpec</a>)
</p>
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
<p>imported from v3.1.0
optional</p>
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
<p>imported from v3.1.0
optional</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
</td>
</tr>
<tr>
<td>
<code>externalTrafficPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#serviceexternaltrafficpolicytype-v1-core">
Kubernetes core/v1.ServiceExternalTrafficPolicyType
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<p>Image used for slowlog tailer
Deprecated, use TidbCluster.HelperImage instead</p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
Deprecated, use TidbCluster.HelperImagePullPolicy instead</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<p>TODO: remove optional after defaulting introduced
Base image of the component, image tag is now allowed during validation</p>
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
<code>slowLogTailer</code></br>
<em>
<a href="#tidbslowlogtailerspec">
TiDBSlowLogTailerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The spec of the slow log tailer sidecar</p>
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
<a href="#tidbconfig">
TiDBConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of tidb-servers</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#statefulsetstatus-v1-apps">
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
</tbody>
</table>
<h3 id="tidbtlsclient">TiDBTLSClient</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbaccessconfig">TiDBAccessConfig</a>, 
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
The name of this Secret must be: ${cluster_name}-tidb-server-secret.
kubectl create secret generic ${cluster_name}-tidb-server-secret &ndash;namespace=${namespace} &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
3. Create a K8s Secret object which contains the TiDB client-side certificate created above which will be used by TiDB Operator.
The name of this Secret must be: ${cluster_name}-tidb-client-secret.
kubectl create secret generic ${cluster_name}-tidb-client-secret &ndash;namespace=${namespace} &ndash;from-file=tls.crt=<path/to/tls.crt> &ndash;from-file=tls.key=<path/to/tls.key> &ndash;from-file=ca.crt=<path/to/ca.crt>
4. Set Enabled to <code>true</code>.</p>
</td>
</tr>
<tr>
<td>
<code>tlsSecret</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specify a secret of client cert for backup/restore
Optional: Defaults to <cluster>-tidb-client-secret
If you want to specify a secret for backup/restore, generate a Secret Object according to the third step of the above procedure, The difference is the Secret Name can be freely defined, and then copy the Secret Name to TLSSecret
this field only work in backup/restore process</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tiflashconfig">TiFlashConfig</h3>
<p>
(<em>Appears on:</em>
<a href="#tiflashspec">TiFlashSpec</a>)
</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<a href="#tiflashconfig">
TiFlashConfig
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
<p>MaxTxnTimeUse is the max time a Txn may use (in seconds) from its startTS to commitTS.
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
(<em>Appears on:</em>
<a href="#tikvspec">TiKVSpec</a>)
</p>
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
<code>encryption</code></br>
<em>
<a href="#tikvencryptionconfig">
TiKVEncryptionConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvcoprocessorconfig">TiKVCoprocessorConfig</h3>
<p>
(<em>Appears on:</em>
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
</tbody>
</table>
<h3 id="tikvdbconfig">TiKVDbConfig</h3>
<p>
(<em>Appears on:</em>
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
<code>method</code></br>
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
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
</tbody>
</table>
<h3 id="tikvimportconfig">TiKVImportConfig</h3>
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
<h3 id="tikvraftdbconfig">TiKVRaftDBConfig</h3>
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
<code>hibernate-regions</code></br>
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
<h3 id="tikvreadpoolconfig">TiKVReadPoolConfig</h3>
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
</tbody>
</table>
<h3 id="tikvserverconfig">TiKVServerConfig</h3>
<p>
(<em>Appears on:</em>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<p>TODO: remove optional after defaulting introduced
Base image of the component, image tag is now allowed during validation</p>
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
<code>config</code></br>
<em>
<a href="#tikvconfig">
TiKVConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Config is the Configuration of tikv-servers</p>
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
<code>statefulSet</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#statefulsetstatus-v1-apps">
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
<code>image</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvstorageconfig">TiKVStorageConfig</h3>
<p>
(<em>Appears on:</em>
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
<code>lastHeartbeatTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<h3 id="tidbautoscalerspec">TidbAutoScalerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautoscalerspec">TidbClusterAutoScalerSpec</a>)
</p>
<p>
<p>TidbAutoScalerSpec describes the spec for tidb auto-scaling</p>
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
<code>BasicAutoScalerSpec</code></br>
<em>
<a href="#basicautoscalerspec">
BasicAutoScalerSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>BasicAutoScalerSpec</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbautoscalerstatus">TidbAutoScalerStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautosclaerstatus">TidbClusterAutoSclaerStatus</a>)
</p>
<p>
<p>TidbAutoScalerStatus describe the auto-scaling status of tidb</p>
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
<code>BasicAutoScalerStatus</code></br>
<em>
<a href="#basicautoscalerstatus">
BasicAutoScalerStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>BasicAutoScalerStatus</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclusterautoscalerspec">TidbClusterAutoScalerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautoscaler">TidbClusterAutoScaler</a>)
</p>
<p>
<p>TidbAutoScalerSpec describes the state of the TidbClusterAutoScaler</p>
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
<a href="#tidbclusterref">
TidbClusterRef
</a>
</em>
</td>
<td>
<p>TidbClusterRef describe the target TidbCluster</p>
</td>
</tr>
<tr>
<td>
<code>metricsUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>We used prometheus to fetch the metrics resources until the pd could provide it.
MetricsUrl represents the url to fetch the metrics info</p>
</td>
</tr>
<tr>
<td>
<code>monitor</code></br>
<em>
<a href="#tidbmonitorref">
TidbMonitorRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TidbMonitorRef describe the target TidbMonitor, when MetricsUrl and Monitor are both set,
Operator will use MetricsUrl</p>
</td>
</tr>
<tr>
<td>
<code>tikv</code></br>
<em>
<a href="#tikvautoscalerspec">
TikvAutoScalerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiKV represents the auto-scaling spec for tikv</p>
</td>
</tr>
<tr>
<td>
<code>tidb</code></br>
<em>
<a href="#tidbautoscalerspec">
TidbAutoScalerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TiDB represents the auto-scaling spec for tidb</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclusterautosclaerstatus">TidbClusterAutoSclaerStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautoscaler">TidbClusterAutoScaler</a>)
</p>
<p>
<p>TidbClusterAutoSclaerStatus describe the whole status</p>
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
<code>tikv</code></br>
<em>
<a href="#tikvautoscalerstatus">
TikvAutoScalerStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tikv describes the status for the tikv in the last auto-scaling reconciliation</p>
</td>
</tr>
<tr>
<td>
<code>tidb</code></br>
<em>
<a href="#tidbautoscalerstatus">
TidbAutoScalerStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Tidb describes the status for the tidb in the last auto-scaling reconciliation</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tidbclusterref">TidbClusterRef</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautoscalerspec">TidbClusterAutoScalerSpec</a>, 
<a href="#tidbinitializerspec">TidbInitializerSpec</a>, 
<a href="#tidbmonitorspec">TidbMonitorSpec</a>)
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
default to the same namespace with TidbMonitor</p>
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
<code>pd</code></br>
<em>
<a href="#pdspec">
PDSpec
</a>
</em>
</td>
<td>
<p>PD cluster spec</p>
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
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TODO: remove optional after defaulting logic introduced
TiDB cluster version</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#persistentvolumereclaimpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#affinity-v1-core">
Kubernetes core/v1.Affinity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Affinity of TiDB cluster Pods</p>
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
<p>Base annotations of TiDB cluster Pods, components may add or override selectors upon this respectively</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
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
<p>Services list non-headless services type used in TidbCluster
Deprecated</p>
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
<code>Pump</code></br>
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
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#jobstatus-v1-batch">
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
<h3 id="tidbmonitorref">TidbMonitorRef</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautoscalerspec">TidbClusterAutoScalerSpec</a>)
</p>
<p>
<p>TidbMonitorRef reference to a TidbMonitor</p>
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
<p>Namespace is the namespace that TidbMonitor object locates,
default to the same namespace with TidbClusterAutoScaler</p>
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
<p>Name is the name of TidbMonitor object</p>
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
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
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
</tbody>
</table>
<h3 id="tidbmonitorstatus">TidbMonitorStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbmonitor">TidbMonitor</a>)
</p>
<p>
<p>TODO: sync status</p>
</p>
<h3 id="tikvautoscalerspec">TikvAutoScalerSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautoscalerspec">TidbClusterAutoScalerSpec</a>)
</p>
<p>
<p>TikvAutoScalerSpec describes the spec for tikv auto-scaling</p>
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
<code>BasicAutoScalerSpec</code></br>
<em>
<a href="#basicautoscalerspec">
BasicAutoScalerSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>BasicAutoScalerSpec</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="tikvautoscalerstatus">TikvAutoScalerStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#tidbclusterautosclaerstatus">TidbClusterAutoSclaerStatus</a>)
</p>
<p>
<p>TikvAutoScalerStatus describe the auto-scaling status of tikv</p>
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
<code>BasicAutoScalerStatus</code></br>
<em>
<a href="#basicautoscalerstatus">
BasicAutoScalerStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>BasicAutoScalerStatus</code> are embedded into this type.)
</p>
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
<code>createdAt</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
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
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>