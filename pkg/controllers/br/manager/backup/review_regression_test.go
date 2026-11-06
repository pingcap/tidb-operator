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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/constants"
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
	backup.Status.LogClusterIdentity = util.LogBackupClusterIdentity(cluster)
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
	backup.Status.LogClusterIdentity = util.LogBackupClusterIdentity(cluster)
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

func TestClusterAccessObservationPreservesLifecycle(t *testing.T) {
	for _, mode := range []br.BackupMode{br.BackupModeSnapshot, br.BackupModeLog} {
		for _, phase := range []br.BackupConditionType{
			"", br.BackupScheduled, br.BackupPrepare, br.BackupRunning, br.BackupComplete,
			br.BackupFailed, br.BackupClean, br.BackupCleanFailed, br.BackupRetryTheFailed,
			br.BackupInvalid, br.BackupPaused, br.BackupStopped, br.BackupRepeatable, br.BackupRestart,
		} {
			backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"},
				Spec: br.BackupSpec{Mode: mode}, Status: br.BackupStatus{Phase: phase,
					LogSubCommandStatuses: map[br.LogSubCommandType]br.LogSubCommandStatus{br.LogStartCommand: {Phase: br.BackupComplete}},
				}}
			cli := client.NewFakeClient(backup)
			updater := NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))
			original := backup.Status.DeepCopy()
			for _, accessible := range []metav1.ConditionStatus{metav1.ConditionFalse, metav1.ConditionTrue} {
				require.NoError(t, updater.Update(t.Context(), backup, &br.BackupCondition{Condition: metav1.Condition{
					Type: constants.ConditionClusterAccessible, Status: accessible, Reason: "AccessObservation",
				}}, nil))
				got := &br.Backup{}
				require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(backup), got))
				require.Equal(t, phase, got.Status.Phase)
				require.Equal(t, original.LogSubCommandStatuses, got.Status.LogSubCommandStatuses)
				condition := meta.FindStatusCondition(got.Status.Conditions, constants.ConditionClusterAccessible)
				require.NotNil(t, condition)
				require.Equal(t, accessible, condition.Status)
			}
		}
	}
}

func TestLogCommandsRejectChangedCluster(t *testing.T) {
	ctx := t.Context()
	original := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "original", UID: "original", Annotations: map[string]string{util.AllowedBackupNamespaces: "source"}}}
	replacement := original.DeepCopy()
	replacement.Namespace, replacement.UID = "replacement", "replacement"
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "source"},
		Spec:   br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster", ClusterNamespace: "original"}},
		Status: br.BackupStatus{CommitTs: "1", LogClusterIdentity: util.LogBackupClusterIdentity(original)},
	}
	backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
	cli := client.NewFakeClient(original, replacement)
	for _, ns := range []string{"original", "replacement"} {
		kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: ns}}
		kv.Spec.Cluster.Name = "cluster"
		pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: ns}}
		pd.Spec.Cluster.Name = "cluster"
		require.NoError(t, cli.Create(ctx, kv))
		require.NoError(t, cli.Create(ctx, pd))
	}
	cleaner := &backupCleaner{cli: cli}
	tracker := &backupTracker{cli: cli}
	_, reason, err := cleaner.makeStopLogBackupJob(ctx, backup)
	require.NoError(t, err, reason)
	backup.Spec.BR.ClusterNamespace = "replacement"
	_, _, err = cleaner.makeStopLogBackupJob(ctx, backup)
	require.ErrorContains(t, err, "log backup cluster has changed")
	_, err = tracker.getLogBackupTC(ctx, backup)
	require.ErrorContains(t, err, "log backup cluster has changed")
	backup.Spec.BR.ClusterNamespace = "original"
	require.NoError(t, cli.Delete(ctx, original))
	original.ResourceVersion, original.UID = "", "recreated"
	require.NoError(t, cli.Create(ctx, original))
	_, _, err = cleaner.makeStopLogBackupJob(ctx, backup)
	require.ErrorContains(t, err, "log backup cluster has changed")
}

func TestLogClusterBindingRecordedBeforeJob(t *testing.T) {
	ctx := t.Context()
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "tenant", UID: "original"}}
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"},
		Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster"}},
	}
	backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
	kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "tenant"}}
	kv.Spec.Cluster.Name = "cluster"
	pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: "tenant"}}
	pd.Spec.Cluster.Name = "cluster"
	cli := client.NewFakeClient(cluster, backup, kv, pd)
	updater := NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))
	manager := &backupManager{cli: cli, statusUpdater: updater}
	job, reason, err := manager.makeBRBackupJob(ctx, backup)
	require.NoError(t, err, reason)
	got := &br.Backup{}
	require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(backup), got))
	require.Equal(t, util.LogBackupClusterIdentity(cluster), got.Status.LogClusterIdentity)
	require.Contains(t, job.Spec.Template.Spec.Containers[0].Args, "--pd-addr=pd-pd.tenant:2379")
	changed := "other"
	require.ErrorContains(t, updater.Update(ctx, got, nil, &BackupUpdateStatus{LogClusterIdentity: &changed}), "cannot change")
	cluster.UID = "recreated"
	require.NoError(t, cli.Update(ctx, cluster))
	unbound := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "unbound", Namespace: "tenant"}, Spec: backup.Spec}
	require.NoError(t, cli.Create(ctx, unbound))
	original := backup.Status.LogClusterIdentity
	require.ErrorContains(t, updater.Update(ctx, unbound, nil, &BackupUpdateStatus{LogClusterIdentity: &original}), "cluster has changed before recording")
	require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(unbound), unbound))
	require.Empty(t, unbound.Status.LogClusterIdentity)
}

func TestExistingStopJobDoesNotBypassClusterBinding(t *testing.T) {
	original := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "original", UID: "original"}}
	target := original.DeepCopy()
	target.Namespace, target.UID = "target", "replacement"
	target.Annotations = map[string]string{util.AllowedBackupNamespaces: "source"}
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "source", Finalizers: []string{"protect"}},
		Spec:   br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster", ClusterNamespace: "target"}},
		Status: br.BackupStatus{CommitTs: "1", LogClusterIdentity: util.LogBackupClusterIdentity(original)},
	}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: backup.GetStopLogBackupJobName(), Namespace: "source"}}
	cli := client.NewFakeClient(target, backup, job)
	cleaner := &backupCleaner{cli: cli, statusUpdater: NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))}
	require.ErrorContains(t, cleaner.StopLogBackup(t.Context(), backup), "log backup cluster has changed")
	require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(backup), backup))
	require.Equal(t, util.LogBackupClusterIdentity(original), backup.Status.LogClusterIdentity)
	require.Contains(t, backup.Finalizers, "protect")
	require.Equal(t, "StopLogBackupBlocked", backup.Status.LogSubCommandStatuses[br.LogStopCommand].Conditions[0].Reason)
}

func TestLegacyRunningLogWithoutCommitTsCannotBeDeclaredStopped(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "tenant"}}
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant", Finalizers: []string{"protect"}},
		Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster"}},
		Status: br.BackupStatus{Phase: br.BackupRunning, BackupPath: "s3://bucket/backup",
			LogSubCommandStatuses: map[br.LogSubCommandType]br.LogSubCommandStatus{br.LogStartCommand: {Phase: br.BackupRunning}},
		},
	}
	cli := client.NewFakeClient(cluster, backup)
	cleaner := &backupCleaner{cli: cli, statusUpdater: NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))}
	require.ErrorContains(t, cleaner.StopLogBackup(t.Context(), backup), "original log backup cluster identity is unavailable")
	require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(backup), backup))
	require.Contains(t, backup.Finalizers, "protect")
	require.NotEqual(t, br.BackupComplete, backup.Status.LogSubCommandStatuses[br.LogStopCommand].Phase)
	require.Equal(t, "StopLogBackupBlocked", backup.Status.LogSubCommandStatuses[br.LogStopCommand].Conditions[0].Reason)
}
