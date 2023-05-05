---
title: TiDB Operator API Document
summary: Reference of TiDB Operator API
category: how-to
---
<h1>API Document</h1>
<h2 id="federation.pingcap.com/v1alpha1">federation.pingcap.com/v1alpha1</h2>
Resource Types:
<ul><li>
<a href="#volumebackup">VolumeBackup</a>
</li><li>
<a href="#volumebackupschedule">VolumeBackupSchedule</a>
</li><li>
<a href="#volumerestore">VolumeRestore</a>
</li></ul>
<h3 id="volumebackup">VolumeBackup</h3>
<p>
<p>VolumeBackup is the control script&rsquo;s spec</p>
</p>
<table>
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
federation.pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>VolumeBackup</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta">
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
<a href="#volumebackupspec">
VolumeBackupSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#volumebackupstatus">
VolumeBackupStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="volumebackupschedule">VolumeBackupSchedule</h3>
<p>
<p>VolumeBackupSchedule is the control script&rsquo;s spec</p>
</p>
<table>
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
federation.pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>VolumeBackupSchedule</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta">
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
<a href="#volumebackupschedulespec">
VolumeBackupScheduleSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#volumebackupschedulestatus">
VolumeBackupScheduleStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="volumerestore">VolumeRestore</h3>
<p>
<p>VolumeRestore is the control script&rsquo;s spec</p>
</p>
<table>
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
federation.pingcap.com/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>VolumeRestore</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta">
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
<a href="#volumerestorespec">
VolumeRestoreSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#volumerestorestatus">
VolumeRestoreStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="volumebackupschedulespec">VolumeBackupScheduleSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackupschedule">VolumeBackupSchedule</a>)
</p>
<p>
<p>VolumeBackupScheduleSpec describes the attributes that a user creates on a volume backup schedule.</p>
</p>
<h3 id="volumebackupschedulestatus">VolumeBackupScheduleStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackupschedule">VolumeBackupSchedule</a>)
</p>
<p>
<p>VolumeBackupScheduleStatus represents the current status of a volume backup schedule.</p>
</p>
<h3 id="volumebackupspec">VolumeBackupSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackup">VolumeBackup</a>)
</p>
<p>
<p>VolumeBackupSpec describes the attributes that a user creates on a volume backup.</p>
</p>
<h3 id="volumebackupstatus">VolumeBackupStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackup">VolumeBackup</a>)
</p>
<p>
<p>VolumeBackupStatus represents the current status of a volume backup.</p>
</p>
<h3 id="volumerestorespec">VolumeRestoreSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#volumerestore">VolumeRestore</a>)
</p>
<p>
<p>VolumeRestoreSpec describes the attributes that a user creates on a volume restore.</p>
</p>
<h3 id="volumerestorestatus">VolumeRestoreStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#volumerestore">VolumeRestore</a>)
</p>
<p>
<p>VolumeRestoreStatus represents the current status of a volume restore.</p>
</p>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>
