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

package restore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	manager "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/restore"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/restore/tasks"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestRestoreRunnerSyncsExistingJobWithoutCluster(t *testing.T) {
	for _, failed := range []bool{false, true} {
		restore := &br.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "source", UID: "restore-uid"}, Spec: br.RestoreSpec{BR: &br.BRConfig{Cluster: "missing", ClusterNamespace: "target"}}}
		condition := batchv1.JobComplete
		expected := br.RestoreComplete
		if failed {
			condition = batchv1.JobFailed
			expected = br.RestoreFailed
		}
		job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: restore.GetRestoreJobName(), Namespace: "source", OwnerReferences: []metav1.OwnerReference{br.GetRestoreOwnerRef(restore)}}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: condition, Status: corev1.ConditionTrue}}}}
		cli := client.NewFakeClient(restore, job)
		m := manager.NewRestoreManager(cli, record.NewFakeRecorder(100), "")
		state := &tasks.ReconcileContext{State: tasks.NewState(restore), RestoreManager: m}
		r := &Reconciler{Client: cli}
		_, err := r.NewRunner(state, task.NewTableTaskReporter("test")).Run(context.Background())
		require.NoError(t, err)
		got := &br.Restore{}
		require.NoError(t, cli.Get(context.Background(), types.NamespacedName{Namespace: "source", Name: "restore"}, got))
		require.Equal(t, expected, got.Status.Phase)
		job.OwnerReferences[0].UID = "foreign"
		require.NoError(t, cli.Update(context.Background(), job))
		_, err = r.NewRunner(state, task.NewTableTaskReporter("test")).Run(context.Background())
		require.ErrorContains(t, err, "not controlled")
	}
}
