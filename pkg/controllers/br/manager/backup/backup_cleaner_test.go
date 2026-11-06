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
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestStopLogBackupJobUsesClientTLSPath(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
	cluster.Spec.TLSCluster = &core.TLSCluster{Enabled: true}
	kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "test"}}
	kv.Spec.Cluster.Name = "cluster"
	pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: "test"}}
	pd.Spec.Cluster.Name = "cluster"
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "test"}, Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster"}}}
	backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
	cleaner := &backupCleaner{cli: client.NewFakeClient(cluster, kv, pd)}
	job, reason, err := cleaner.makeStopLogBackupJob(t.Context(), backup)
	require.NoError(t, err, reason)
	container := job.Spec.Template.Spec.Containers[0]
	require.Contains(t, container.Args, "--cluster-tls=true")
	require.Contains(t, container.VolumeMounts, corev1.VolumeMount{Name: core.VolumeNameClusterClientTLS, ReadOnly: true, MountPath: core.DirPathClusterClientTLS})
	require.Contains(t, job.Spec.Template.Spec.Volumes, corev1.Volume{Name: core.VolumeNameClusterClientTLS, VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: coreutil.LegacyTLSClusterClientSecretName("cluster")}}})
}
func TestStopLogBackupJobMustSucceedBeforeCleanup(t *testing.T) {
	for _, condition := range []batchv1.JobConditionType{batchv1.JobFailed, batchv1.JobComplete, "", "Missing"} {
		t.Run(string(condition), func(t *testing.T) {
			backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "test"}, Spec: br.BackupSpec{Mode: br.BackupModeLog}, Status: br.BackupStatus{CommitTs: "1"}}
			job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: backup.GetStopLogBackupJobName(), Namespace: backup.Namespace}}
			if condition != "" {
				job.Status.Conditions = []batchv1.JobCondition{{Type: condition, Status: corev1.ConditionTrue}}
			}
			objects := []client.Object{}
			if condition != "Missing" {
				objects = append(objects, job)
			}
			cleaner := &backupCleaner{cli: client.NewFakeClient(objects...)}
			finished, err := cleaner.ensureBackupJobFinished(t.Context(), backup)
			if condition != "Missing" {
				require.NoError(t, cleaner.cli.Get(t.Context(), client.ObjectKeyFromObject(job), job))
				require.Nil(t, job.DeletionTimestamp, "cleanup must not cancel its stop Job")
			} else {
				require.True(t, errors.IsNotFound(cleaner.cli.Get(t.Context(), client.ObjectKeyFromObject(job), job)))
			}
			if condition == batchv1.JobFailed {
				require.ErrorContains(t, err, "stop log backup")
				require.False(t, finished)
			} else {
				require.NoError(t, err)
				require.Equal(t, condition == batchv1.JobComplete, finished)
			}
		})
	}
}

func TestRecordedLogStopAllowsCleanupWithoutJob(t *testing.T) {
	for _, jobExists := range []bool{false, true} {
		name := "already missing"
		if jobExists {
			name = "removed between checks"
		}
		t.Run(name, func(t *testing.T) {
			now := metav1.Now()
			backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "test", DeletionTimestamp: &now}, Spec: br.BackupSpec{Mode: br.BackupModeLog, CleanPolicy: br.CleanPolicyTypeDelete, BR: &br.BRConfig{Cluster: "db"}}, Status: br.BackupStatus{Phase: br.BackupRunning, CommitTs: "1", BackupPath: "s3://bucket/backup"}}
			backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
			stopJob := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: backup.GetStopLogBackupJobName(), Namespace: backup.Namespace}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}}
			objects := []client.Object{backup}
			if jobExists {
				objects = append(objects, stopJob)
			}
			cli := client.NewFakeClient(objects...)
			updater := NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))
			require.NoError(t, updater.Update(t.Context(), backup, &br.BackupCondition{Command: br.LogStopCommand, Condition: metav1.Condition{Type: string(br.BackupComplete), Status: metav1.ConditionTrue, Reason: "StopCompleted"}}, nil))
			require.Equal(t, br.BackupComplete, backup.Status.Phase)
			require.Equal(t, br.BackupComplete, backup.Status.LogSubCommandStatuses[br.LogStopCommand].Phase)
			cleaner := &backupCleaner{cli: cli, statusUpdater: updater}
			require.NoError(t, cleaner.StopLogBackup(t.Context(), backup))
			if jobExists {
				require.NoError(t, cli.Delete(t.Context(), stopJob))
			}
			require.NoError(t, cleaner.CleanData(t.Context(), backup))
			jobs := &batchv1.JobList{}
			require.NoError(t, cli.List(t.Context(), jobs))
			require.Len(t, jobs.Items, 1)
			require.Equal(t, backup.GetCleanJobName(), jobs.Items[0].Name)
		})
	}
}

func TestNotStartedStopRecordDoesNotCoverLaterStartup(t *testing.T) {
	now := metav1.Now()
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "test", DeletionTimestamp: &now}, Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "db"}}}
	cli := client.NewFakeClient(backup)
	updater := NewRealBackupConditionUpdater(cli, record.NewFakeRecorder(100))
	require.NoError(t, updater.Update(t.Context(), backup, &br.BackupCondition{Command: br.LogStopCommand, Condition: metav1.Condition{Type: string(br.BackupComplete), Status: metav1.ConditionTrue, Reason: "LogBackupNotStarted"}}, nil))
	commitTs := "1"
	require.NoError(t, updater.Update(t.Context(), backup, &br.BackupCondition{Command: br.LogStartCommand, Condition: metav1.Condition{Type: string(br.BackupComplete), Status: metav1.ConditionTrue, Reason: "StartCompleted"}}, &BackupUpdateStatus{CommitTs: &commitTs}))
	require.True(t, br.IsLogBackupAlreadyStart(backup))
	cleaner := &backupCleaner{cli: cli, statusUpdater: updater}
	finished, err := cleaner.isLogStopJobFinished(t.Context(), backup)
	require.NoError(t, err)
	require.False(t, finished, "an earlier not-started observation does not prove the new task stopped")
}
