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

package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
)

func TestLegacyStopBackupTLS(t *testing.T) {
	ctx := context.Background()
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "target"}}
	cluster.Spec.TLSCluster = &core.TLSCluster{Enabled: true}
	kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "target"}}
	kv.Spec.Cluster.Name = "cluster"
	pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: "target"}}
	pd.Spec.Cluster.Name = "cluster"
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "source", Finalizers: []string{"protect-data"}}, Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster", ClusterNamespace: "target"}}, Status: br.BackupStatus{TimeStarted: metav1.Now(), CommitTs: "1"}}
	backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
	cli := client.NewFakeClient(cluster, kv, pd, backup)
	updater := NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))
	cleaner := &backupCleaner{cli: cli, statusUpdater: updater}
	require.NoError(t, cleaner.StopLogBackup(ctx, backup))
	job, reason, err := cleaner.makeStopLogBackupJob(ctx, backup)
	require.NoError(t, err, reason)
	found := false
	for _, mount := range job.Spec.Template.Spec.Containers[0].VolumeMounts {
		if mount.Name == core.VolumeNameClusterClientTLS {
			require.Equal(t, core.DirPathClusterClientTLS, mount.MountPath)
			found = true
		}
	}
	require.True(t, found)
	for _, vol := range job.Spec.Template.Spec.Volumes {
		if vol.Name == core.VolumeNameClusterClientTLS {
			require.Equal(t, coreutil.LegacyTLSClusterClientSecretName("cluster"), vol.Secret.SecretName)
		}
	}
}

// TestStaleWorkerCannotReplaceStorageIdentity checks the atomic guard after reading the latest CR.
func TestStaleWorkerCannotReplaceStorageIdentity(t *testing.T) {
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"}, Status: br.BackupStatus{StorageIdentity: "original-identity", BackupPath: "s3://bucket/original"}}
	cli := client.NewFakeClient(backup)
	updater := NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))
	identity := "stale-identity"
	path := "s3://bucket/stale"
	err := updater.Update(context.Background(), backup.DeepCopy(), nil, &BackupUpdateStatus{StorageIdentity: &identity})
	require.ErrorContains(t, err, "identity cannot change")
	err = updater.Update(context.Background(), backup.DeepCopy(), nil, &BackupUpdateStatus{BackupPath: &path})
	require.ErrorContains(t, err, "path cannot change")
	got := &br.Backup{}
	require.NoError(t, cli.Get(context.Background(), types.NamespacedName{Namespace: "tenant", Name: "backup"}, got))
	require.Equal(t, "original-identity", got.Status.StorageIdentity)
	require.Equal(t, "s3://bucket/original", got.Status.BackupPath)
}

func TestCleanJobLocalAdditionalMountsAreBoundToIdentity(t *testing.T) {
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"}}
	backup.Spec.Local = &br.LocalStorageProvider{Prefix: "backup"}
	backup.Spec.Local.Volume.Name = "original"
	backup.Spec.Local.VolumeMount = corev1.VolumeMount{Name: "original", MountPath: "/data"}
	backup.Spec.AdditionalVolumes = []corev1.Volume{{Name: "overlay", VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "original"},
	}}}
	backup.Spec.AdditionalVolumeMounts = []corev1.VolumeMount{{Name: "overlay", MountPath: "/data/backup"}}
	var err error
	backup.Status.StorageIdentity, err = util.StorageIdentity(backup)
	require.NoError(t, err)
	backup.Status.BackupPath = "local:///data/backup"
	cleaner := &backupCleaner{cli: client.NewFakeClient()}
	job, reason, err := cleaner.makeCleanJob(t.Context(), backup)
	require.NoError(t, err, reason)
	require.Contains(t, job.Spec.Template.Spec.Containers[0].VolumeMounts, backup.Spec.AdditionalVolumeMounts[0])
	_, err = util.ValidateStorageIdentity(backup)
	require.NoError(t, err)
	backup.Spec.AdditionalVolumes[0].PersistentVolumeClaim.ClaimName = "other"
	_, err = util.ValidateStorageIdentity(backup)
	require.ErrorContains(t, err, "storage identity has changed")
}
