---
title: TiDB Operator API Document
summary: Reference of TiDB Operator API
category: how-to
---
<h1>API Document</h1>
<h2 id="pingcap.com/v1alpha1">pingcap.com/v1alpha1</h2>
Resource Types:
<ul><li>
<a href="#volumebackupfederation">VolumeBackupFederation</a>
</li><li>
<a href="#volumebackupschedulefederation">VolumeBackupScheduleFederation</a>
</li><li>
<a href="#volumerestorefederation">VolumeRestoreFederation</a>
</li></ul>
<h3 id="volumebackupfederation">VolumeBackupFederation</h3>
<p>
<p>VolumeBackupFederation is the control script&rsquo;s spec</p>
</p>
<table>
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
<td><code>VolumeBackupFederation</code></td>
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
<a href="#volumebackupfederationspec">
VolumeBackupFederationSpec
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
<a href="#volumebackupfederationstatus">
VolumeBackupFederationStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="volumebackupschedulefederation">VolumeBackupScheduleFederation</h3>
<p>
<p>VolumeBackupScheduleFederation is the control script&rsquo;s spec</p>
</p>
<table>
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
<td><code>VolumeBackupScheduleFederation</code></td>
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
<a href="#volumebackupschedulefederationspec">
VolumeBackupScheduleFederationSpec
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
<a href="#volumebackupschedulefederationstatus">
VolumeBackupScheduleFederationStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="volumerestorefederation">VolumeRestoreFederation</h3>
<p>
<p>VolumeRestoreFederation is the control script&rsquo;s spec</p>
</p>
<table>
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
<td><code>VolumeRestoreFederation</code></td>
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
<a href="#volumerestorefederationspec">
VolumeRestoreFederationSpec
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
<a href="#volumerestorefederationstatus">
VolumeRestoreFederationStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="volumebackupfederationspec">VolumeBackupFederationSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackupfederation">VolumeBackupFederation</a>)
</p>
<p>
<p>VolumeBackupFederationSpec describes the attributes that a user creates on a volume backup federation.</p>
</p>
<h3 id="volumebackupfederationstatus">VolumeBackupFederationStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackupfederation">VolumeBackupFederation</a>)
</p>
<p>
<p>VolumeBackupFederationStatus represents the current status of a volume backup federation.</p>
</p>
<h3 id="volumebackupschedulefederationspec">VolumeBackupScheduleFederationSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackupschedulefederation">VolumeBackupScheduleFederation</a>)
</p>
<p>
<p>VolumeBackupScheduleFederationSpec describes the attributes that a user creates on a volume backup schedule federation.</p>
</p>
<h3 id="volumebackupschedulefederationstatus">VolumeBackupScheduleFederationStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#volumebackupschedulefederation">VolumeBackupScheduleFederation</a>)
</p>
<p>
<p>VolumeBackupScheduleFederationStatus represents the current status of a volume backup schedule federation.</p>
</p>
<h3 id="volumerestorefederationspec">VolumeRestoreFederationSpec</h3>
<p>
(<em>Appears on:</em>
<a href="#volumerestorefederation">VolumeRestoreFederation</a>)
</p>
<p>
<p>VolumeRestoreFederationSpec describes the attributes that a user creates on a volume restore federation.</p>
</p>
<h3 id="volumerestorefederationstatus">VolumeRestoreFederationStatus</h3>
<p>
(<em>Appears on:</em>
<a href="#volumerestorefederation">VolumeRestoreFederation</a>)
</p>
<p>
<p>VolumeRestoreFederationStatus represents the current status of a volume restore federation.</p>
</p>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>
