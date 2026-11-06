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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	manager "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
)

func TestInvalidLogBackupDeletionStartsStopJob(t *testing.T) {
	for _, policy := range []br.CleanPolicyType{br.CleanPolicyTypeRetain, br.CleanPolicyTypeOnFailure} {
		for _, clusterState := range []string{"normal", "paused", "missing"} {
			for _, deleting := range []bool{false, true} {
				name := string(policy) + "/" + clusterState
				if deleting {
					name += "/deleting"
				} else {
					name += "/active"
				}
				t.Run(name, func(t *testing.T) {
					cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "test"}}
					cluster.Spec.Paused = clusterState == "paused"
					kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "test"}}
					kv.Spec.Cluster.Name = "db"
					pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: "test"}}
					pd.Spec.Cluster.Name = "db"
					b := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "test", Finalizers: []string{meta.Finalizer}}, Spec: br.BackupSpec{Mode: br.BackupModeLog, CleanPolicy: policy, BR: &br.BRConfig{Cluster: "db"}, LogSubcommand: br.LogPauseCommand, LogTruncateUntil: "invalid"}, Status: br.BackupStatus{Phase: br.BackupRunning, CommitTs: "1"}}
					b.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
					objects := []client.Object{b, kv, pd}
					if clusterState != "missing" {
						objects = append(objects, cluster)
					}
					cli := client.NewFakeClient(objects...)
					recorder := record.NewFakeRecorder(100)
					updater := manager.NewRealBackupConditionUpdater(cli, recorder)
					require.NoError(t, updater.Update(t.Context(), b, &br.BackupCondition{Command: br.LogPauseCommand, Condition: metav1.Condition{Type: string(br.BackupInvalid), Status: metav1.ConditionTrue, Reason: "InvalidSpec"}}, nil))
					require.True(t, br.IsBackupInvalid(b))
					require.Equal(t, br.BackupRunning, b.Status.Phase)
					if deleting {
						now := metav1.Now()
						b.DeletionTimestamp = &now
						require.NoError(t, cli.Update(t.Context(), b))
					}
					r := &Reconciler{Client: cli, EventRecorder: recorder, BackupCleaner: manager.NewBackupCleaner(cli, updater, "")}
					_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(b)})
					if deleting && clusterState == "missing" {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
					jobs := &batchv1.JobList{}
					require.NoError(t, cli.List(t.Context(), jobs))
					if deleting && clusterState != "missing" {
						require.Len(t, jobs.Items, 1)
						require.Equal(t, b.GetStopLogBackupJobName(), jobs.Items[0].Name)
					} else {
						require.Empty(t, jobs.Items)
					}
					got := &br.Backup{}
					require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(b), got))
					require.Contains(t, got.Finalizers, meta.Finalizer)
				})
			}
		}
	}
}
