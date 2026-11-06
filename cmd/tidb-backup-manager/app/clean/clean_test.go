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

package clean

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	bkutil "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
)

func TestValidateBRBackupPath(t *testing.T) {
	for _, tc := range []struct {
		name, bucket, prefix, recorded string
		valid                          bool
	}{
		{"valid", "bucket", "backup/one", "s3://bucket/backup/one", true},
		{"bucket with prefix", "bucket/backup", "one", "s3://bucket/backup/one", true},
		{"changed bucket", "other", "backup/one", "s3://bucket/backup/one", false},
		{"changed prefix", "bucket", "backup/two", "s3://bucket/backup/one", false},
		{"missing record", "bucket", "backup/one", "", false},
		{"bucket root", "bucket", "", "s3://bucket", false},
		{"slash root", "bucket", "/", "s3://bucket", false},
		{"double slash", "bucket", "backup//one", "s3://bucket/backup/one", false},
		{"dot segments", "bucket", "backup/../one", "s3://bucket/one", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backup := &v1alpha1.Backup{}
			backup.Spec.S3 = &v1alpha1.S3StorageProvider{Bucket: tc.bucket, Prefix: tc.prefix}
			backup.Status.BackupPath = tc.recorded
			err := validateBRBackupPath(backup)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				// The public operation must reject before credentials or storage are accessed.
				require.ErrorContains(t, (&Options{}).CleanBRRemoteBackupData(context.Background(), backup), "refusing cleanup")
			}
		})
	}
	for _, backup := range []*v1alpha1.Backup{
		{Spec: v1alpha1.BackupSpec{StorageProvider: v1alpha1.StorageProvider{Gcs: &v1alpha1.GcsStorageProvider{Bucket: "bucket", Prefix: "backup"}}}, Status: v1alpha1.BackupStatus{BackupPath: "gcs://bucket/backup/"}},
		{Spec: v1alpha1.BackupSpec{StorageProvider: v1alpha1.StorageProvider{Azblob: &v1alpha1.AzblobStorageProvider{Container: "bucket", Prefix: "backup"}}}, Status: v1alpha1.BackupStatus{BackupPath: "azure://bucket/backup/"}},
	} {
		require.NoError(t, validateBRBackupPath(backup))
	}
}

func TestCleanLocalBackup(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "backup", "nested"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "backup", "nested", "data"), []byte("nested"), 0600))
	require.NoError(t, os.Symlink("..", filepath.Join(root, "backup", "root-link")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "backup", "data"), []byte("backup"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep"), []byte("keep"), 0600))
	backup := &v1alpha1.Backup{}
	backup.Spec.Local = &v1alpha1.LocalStorageProvider{Prefix: "backup"}
	backup.Spec.Local.VolumeMount.MountPath = root
	backup.Status.BackupPath = "local://" + filepath.Join(root, "backup")
	var err error
	backup.Status.StorageIdentity, err = bkutil.StorageIdentity(backup)
	require.NoError(t, err)
	require.NoError(t, (&Options{}).CleanBRRemoteBackupData(context.Background(), backup))
	_, err = os.Stat(filepath.Join(root, "backup", "data"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root, "keep"))
	require.NoError(t, err)
	for _, prefix := range []string{"", ".", "..", "../outside", "/", "/outside", "a/../b"} {
		backup.Spec.Local.Prefix = prefix
		backup.Status.BackupPath = "local://" + path.Join(root, prefix)
		require.Error(t, validateBRBackupPath(backup))
	}
}

// TestLegacyCleanupCannotRedirectBRBackup rejects changing cleanup engines and rclone endpoint overrides.
func TestLegacyCleanupCannotRedirectBRBackup(t *testing.T) {
	backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{BR: &v1alpha1.BRConfig{Cluster: "cluster"}}}
	backup.Spec.S3 = &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "backup", Endpoint: "https://original"}
	identity, err := bkutil.StorageIdentity(backup)
	require.NoError(t, err)
	backup.Status.BackupPath = "s3://bucket/backup"
	backup.Status.StorageIdentity = identity
	backup.Spec.BR = nil
	backup.Spec.S3.Options = []string{"--s3-endpoint=https://other"}
	err = (&Manager{}).performCleanBackup(context.Background(), backup)
	require.ErrorContains(t, err, "storage identity has changed")
}

func TestLocalCleanupCannotReachMountRootThroughSymlinks(t *testing.T) {
	for _, tc := range []struct {
		name, link, prefix string
	}{
		{"parent alias", "..", "backup/data"},
		{"root alias", ".", "backup"},
		{"nested root alias", "..", "nested/backup"},
		{"outside alias", "../outside", "backup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			root := filepath.Join(parent, "data")
			outside := filepath.Join(parent, "outside")
			require.NoError(t, os.MkdirAll(filepath.Join(root, "nested"), 0700))
			require.NoError(t, os.Mkdir(outside, 0700))
			for _, dir := range []string{root, outside} {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "keep"), []byte("unrelated"), 0600))
			}
			link := filepath.Join(root, "backup")
			if tc.name == "nested root alias" {
				link = filepath.Join(root, "nested", "backup")
			}
			require.NoError(t, os.Symlink(tc.link, link))
			backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{BR: &v1alpha1.BRConfig{Cluster: "cluster"}}}
			backup.Spec.Local = &v1alpha1.LocalStorageProvider{Prefix: tc.prefix}
			backup.Spec.Local.VolumeMount.MountPath = root
			backup.Status.BackupPath = "local://" + filepath.Join(root, tc.prefix)
			var err error
			backup.Status.StorageIdentity, err = bkutil.StorageIdentity(backup)
			require.NoError(t, err)
			err = (&Options{}).CleanBRRemoteBackupData(t.Context(), backup)
			require.Error(t, err)
			for _, dir := range []string{root, outside} {
				data, err := os.ReadFile(filepath.Join(dir, "keep"))
				require.NoError(t, err)
				require.Equal(t, "unrelated", string(data))
			}
		})
	}
}

func TestLocalCleanupAllowsContainedNonRootAlias(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "backups", "one"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "backups", "one", "data"), []byte("backup"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep"), []byte("keep"), 0600))
	require.NoError(t, os.Symlink("backups/one", filepath.Join(root, "backup")))
	backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{BR: &v1alpha1.BRConfig{Cluster: "cluster"}}}
	backup.Spec.Local = &v1alpha1.LocalStorageProvider{Prefix: "backup"}
	backup.Spec.Local.VolumeMount.MountPath = root
	backup.Status.BackupPath = "local://" + filepath.Join(root, "backup")
	var err error
	backup.Status.StorageIdentity, err = bkutil.StorageIdentity(backup)
	require.NoError(t, err)
	require.NoError(t, (&Options{}).CleanBRRemoteBackupData(t.Context(), backup))
	_, err = os.Stat(filepath.Join(root, "backups", "one", "data"))
	require.True(t, os.IsNotExist(err))
	data, err := os.ReadFile(filepath.Join(root, "keep"))
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}

func TestLocalCleanupRejectsAdditionalMountChanges(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "backup"), 0700))
	file := filepath.Join(root, "backup", "keep")
	require.NoError(t, os.WriteFile(file, []byte("keep"), 0600))
	backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{BR: &v1alpha1.BRConfig{Cluster: "cluster"}}}
	backup.Spec.Local = &v1alpha1.LocalStorageProvider{Prefix: "backup"}
	backup.Spec.Local.VolumeMount.MountPath = root
	backup.Status.BackupPath = "local://" + filepath.Join(root, "backup")
	var err error
	backup.Status.StorageIdentity, err = bkutil.StorageIdentity(backup)
	require.NoError(t, err)
	backup.Spec.AdditionalVolumes = []corev1.Volume{{Name: "overlay", VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "other"},
	}}}
	backup.Spec.AdditionalVolumeMounts = []corev1.VolumeMount{{Name: "overlay", MountPath: filepath.Join(root, "backup")}}
	err = (&Options{}).CleanBRRemoteBackupData(t.Context(), backup)
	require.ErrorContains(t, err, "storage identity has changed")
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}
