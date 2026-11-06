// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

func TestStorageDestinationIdentity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider br.StorageProvider
		mutate   func(*br.StorageProvider)
	}{
		{"S3 endpoint", br.StorageProvider{S3: &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup", Endpoint: "https://original"}}, func(p *br.StorageProvider) { p.S3.Endpoint = "https://other" }},
		{"Azure account", br.StorageProvider{Azblob: &br.AzblobStorageProvider{Container: "bucket", Prefix: "backup", StorageAccount: "original"}}, func(p *br.StorageProvider) { p.Azblob.StorageAccount = "other" }},
		{"local PVC", br.StorageProvider{Local: &br.LocalStorageProvider{Prefix: "backup", Volume: corev1.Volume{VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "original"}}}}}, func(p *br.StorageProvider) { p.Local.Volume.PersistentVolumeClaim.ClaimName = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backup := &br.Backup{Spec: br.BackupSpec{StorageProvider: tc.provider}}
			identity, err := ValidateStorageIdentity(backup)
			require.NoError(t, err)
			backup.Status.BackupPath = "recorded-path"
			backup.Status.StorageIdentity = identity
			_, err = ValidateStorageIdentity(backup)
			require.NoError(t, err)
			tc.mutate(&backup.Spec.StorageProvider)
			_, err = ValidateStorageIdentity(backup)
			require.ErrorContains(t, err, "storage identity has changed")
			backup.Status.StorageIdentity = ""
			_, err = ValidateStorageIdentity(backup)
			require.ErrorContains(t, err, "original storage identity is unavailable")
		})
	}
	t.Setenv("AZURE_STORAGE_ACCOUNT", "original")
	backup := &br.Backup{Spec: br.BackupSpec{StorageProvider: br.StorageProvider{Azblob: &br.AzblobStorageProvider{Container: "bucket", Prefix: "backup"}}}}
	identity, err := StorageIdentity(backup)
	require.NoError(t, err)
	backup.Status.BackupPath = "recorded"
	backup.Status.StorageIdentity = identity
	t.Setenv("AZURE_STORAGE_ACCOUNT", "other")
	_, err = ValidateStorageIdentity(backup)
	require.ErrorContains(t, err, "storage identity has changed")
}
