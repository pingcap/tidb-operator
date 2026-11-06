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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
)

func TestStopBackupConsentAndTLS(t *testing.T) {
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
	require.Error(t, cleaner.StopLogBackup(ctx, backup))
	got := &br.Backup{}
	require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: "source", Name: "backup"}, got))
	condition := got.Status.LogSubCommandStatuses[br.LogStopCommand]
	require.NotEmpty(t, condition.Conditions)
	require.Equal(t, "StopLogBackupBlocked", condition.Conditions[0].Reason)
	require.Contains(t, got.Finalizers, "protect-data")
	jobs := &batchv1.JobList{}
	require.NoError(t, cli.List(ctx, jobs))
	require.Empty(t, jobs.Items)
	cluster.Annotations = map[string]string{util.AllowedBackupNamespaces: "source"}
	require.NoError(t, cli.Update(ctx, cluster))
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

// TestRunningLogBackupReenrollsAfterConsent verifies retry without an Operator restart.
func TestRunningLogBackupReenrollsAfterConsent(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "target"}}
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "source"}, Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster", ClusterNamespace: "target"}}, Status: br.BackupStatus{CommitTs: "1", Phase: br.BackupRunning, LogSubCommandStatuses: map[br.LogSubCommandType]br.LogSubCommandStatus{br.LogStartCommand: {Phase: br.BackupComplete}}}}
	cli := client.NewFakeClient(cluster, backup)
	tracker := &backupTracker{cli: cli, logBackups: map[string]bool{}}
	manager := &backupManager{cli: cli, backupTracker: tracker}
	err := manager.Sync(context.Background(), backup)
	require.Error(t, err)
	require.Empty(t, tracker.logBackups)
	cluster.Annotations = map[string]string{util.AllowedBackupNamespaces: "source"}
	require.NoError(t, cli.Update(context.Background(), cluster))
	err = manager.Sync(context.Background(), backup)
	require.NoError(t, err)
	tracker.operateLock.Lock()
	require.True(t, tracker.logBackups[genLogBackupKey("source", "backup")])
	tracker.operateLock.Unlock()
	// The object disappears before the first scheduled checkpoint, so the worker will retire.
	require.NoError(t, cli.Delete(context.Background(), backup))
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
