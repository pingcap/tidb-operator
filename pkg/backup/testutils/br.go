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
