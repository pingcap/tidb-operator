// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// GenValidStorageProviders generates valid storage providers
func GenValidStorageProviders() []v1alpha1.StorageProvider {
	return []v1alpha1.StorageProvider{
		{
			S3: &v1alpha1.S3StorageProvider{
				Bucket:   "s3",
				Prefix:   "prefix-",
				Endpoint: "s3://localhost:80",
			},
		},
		{
			S3: &v1alpha1.S3StorageProvider{
				Bucket:     "s3",
				Prefix:     "prefix-",
				Endpoint:   "s3://localhost:80",
				SecretName: "s3",
			},
		},
		{
			Gcs: &v1alpha1.GcsStorageProvider{
				ProjectId: "gcs",
				Bucket:    "gcs",
				Prefix:    "prefix-",
			},
		},
		{
			Gcs: &v1alpha1.GcsStorageProvider{
				ProjectId:  "gcs",
				Bucket:     "gcs",
				Prefix:     "prefix-",
				SecretName: "gcs",
			},
		},
		{
			Local: &v1alpha1.LocalStorageProvider{
				Prefix: "prefix-",
				Volume: corev1.Volume{
					Name: "nfs",
					VolumeSource: corev1.VolumeSource{
						NFS: &corev1.NFSVolumeSource{
							Server:   "fake-server",
							Path:     "/some/path",
							ReadOnly: true,
						},
					},
				},
				VolumeMount: corev1.VolumeMount{
					Name:      "nfs",
					MountPath: "/some/path",
				},
			},
		},
	}
}

func ConstructRestoreMetaStr() string {
	return `{
		"tikv": {
			"replicas": 3,
			"stores": [{
				"store_id": 1,
				"volumes": [{
					"volume_id": "vol-0e65f40961a9f6244",
					"type": "",
					"mount_path": "",
					"snapshot_id": "snap-1234567890abcdef0",
					"restore_volume_id": "vol-0e65f40961a9f0001"
				}]
			}, {
				"store_id": 2,
				"volumes": [{
					"volume_id": "vol-0e65f40961a9f6245",
					"type": "",
					"mount_path": "",
					"snapshot_id": "snap-1234567890abcdef1",
					"restore_volume_id": "vol-0e65f40961a9f0002"
				}]
			}, {
				"store_id": 3,
				"volumes": [{
					"volume_id": "vol-0e65f40961a9f6246",
					"type": "",
					"mount_path": "",
					"snapshot_id": "snap-1234567890abcdef2",
					"restore_volume_id": "vol-0e65f40961a9f0003"
				}]
			}]
		},
		"pd": {
			"replicas": 0
		},
		"tidb": {
			"replicas": 0
		},
		"kubernetes": {
			"pvcs": [{
				"metadata": {
					"name": "pvc-1",
					"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3121",
					"resourceVersion": "1957",
					"creationTimestamp": null,
					"labels": {
						"test/label": "retained"
					},
					"annotations": {
						"pv.kubernetes.io/bind-completed": "yes",
						"pv.kubernetes.io/bound-by-controller": "yes",
						"test/annotation": "retained"
					},
					"finalizers": ["kubernetes.io/pvc-protection"]
				},
				"spec": {
					"resources": {},
					"volumeName": "pv-1"
				},
				"status": {
					"phase": "Bound"
				}
			}, {
				"metadata": {
					"name": "pvc-2",
					"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3123",
					"resourceVersion": "1959",
					"creationTimestamp": null,
					"labels": {
						"test/label": "retained"
					},
					"annotations": {
						"pv.kubernetes.io/bind-completed": "yes",
						"pv.kubernetes.io/bound-by-controller": "yes",
						"test/annotation": "retained"
					},
					"finalizers": ["kubernetes.io/pvc-protection"]
				},
				"spec": {
					"resources": {},
					"volumeName": "pv-2"
				},
				"status": {
					"phase": "Bound"
				}
			}, {
				"metadata": {
					"name": "pvc-3",
					"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3125",
					"resourceVersion": "1961",
					"creationTimestamp": null,
					"labels": {
						"test/label": "retained"
					},
					"annotations": {
						"pv.kubernetes.io/bind-completed": "yes",
						"pv.kubernetes.io/bound-by-controller": "yes",
						"test/annotation": "retained"
					},
					"finalizers": ["kubernetes.io/pvc-protection"]
				},
				"spec": {
					"resources": {},
					"volumeName": "pv-3"
				},
				"status": {
					"phase": "Bound"
				}
			}],
			"pvs": [{
				"metadata": {
					"name": "pv-1",
					"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3122",
					"resourceVersion": "1958",
					"creationTimestamp": null,
					"labels": {
						"test/label": "retained"
					},
					"annotations": {
						"pv.kubernetes.io/provisioned-by": "ebs.csi.aws.com",
						"temporary/volume-id": "vol-0e65f40961a9f6244",
						"test/annotation": "retained"
					},
					"finalizers": ["kubernetes.io/pv-protection"]
				},
				"spec": {
					"csi": {
						"driver": "ebs.csi.aws.com",
						"volumeHandle": "vol-0e65f40961a9f6244",
						"fsType": "ext4"
					},
					"claimRef": {
						"name": "pvc-1",
						"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3121",
						"resourceVersion": "1957"
					}
				},
				"status": {
					"phase": "Bound"
				}
			}, {
				"metadata": {
					"name": "pv-2",
					"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3124",
					"resourceVersion": "1960",
					"creationTimestamp": null,
					"labels": {
						"test/label": "retained"
					},
					"annotations": {
						"pv.kubernetes.io/provisioned-by": "ebs.csi.aws.com",
						"temporary/volume-id": "vol-0e65f40961a9f6245",
						"test/annotation": "retained"
					},
					"finalizers": ["kubernetes.io/pv-protection"]
				},
				"spec": {
					"csi": {
						"driver": "ebs.csi.aws.com",
						"volumeHandle": "vol-0e65f40961a9f6245",
						"fsType": "ext4"
					},
					"claimRef": {
						"name": "pvc-2",
						"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3123",
						"resourceVersion": "1959"
					}
				},
				"status": {
					"phase": "Bound"
				}
			}, {
				"metadata": {
					"name": "pv-3",
					"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3126",
					"resourceVersion": "1962",
					"creationTimestamp": null,
					"labels": {
						"test/label": "retained"
					},
					"annotations": {
						"pv.kubernetes.io/provisioned-by": "ebs.csi.aws.com",
						"temporary/volume-id": "vol-0e65f40961a9f6246",
						"test/annotation": "retained"
					},
					"finalizers": ["kubernetes.io/pv-protection"]
				},
				"spec": {
					"csi": {
						"driver": "ebs.csi.aws.com",
						"volumeHandle": "vol-0e65f40961a9f6246",
						"fsType": "ext4"
					},
					"claimRef": {
						"name": "pvc-3",
						"uid": "301b0e8b-3538-4f61-a0fd-a25abd9a3125",
						"resourceVersion": "1961"
					}
				},
				"status": {
					"phase": "Bound"
				}
			}],
			"crd_tidb_cluster": {
				"metadata": {
					"creationTimestamp": null
				},
				"spec": {
					"discovery": {},
					"version": ""
				},
				"status": {
					"pd": {
						"synced": false,
						"leader": {
							"name": "",
							"id": "",
							"clientURL": "",
							"health": false,
							"lastTransitionTime": null
						}
					},
					"tikv": {},
					"tidb": {},
					"pump": {},
					"tiflash": {},
					"ticdc": {}
				}
			},
			"options": null
		},
		"options": null
	}`
}
