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
