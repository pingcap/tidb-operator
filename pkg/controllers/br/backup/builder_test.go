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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/backup/tasks"
	manager "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type observingManager struct {
	manager.BackupManager
	called bool
	reason string
}

func (m *observingManager) Sync(context.Context, *br.Backup) error { m.called = true; return nil }
func (m *observingManager) UpdateStatus(_ context.Context, _ *br.Backup, c *br.BackupCondition, _ *manager.BackupUpdateStatus) error {
	m.reason = c.Reason
	return nil
}

type idleCleaner struct{ manager.BackupCleaner }

func (*idleCleaner) Clean(context.Context, *br.Backup) error { return nil }
func TestRunnerUsesConsentingTargetCluster(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "target", Annotations: map[string]string{util.AllowedBackupNamespaces: "source"}}}
	misleading := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "source"}}
	misleading.Spec.Paused = true
	cli := client.NewFakeClient(cluster, misleading)
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "source"}, Spec: br.BackupSpec{BR: &br.BRConfig{Cluster: "cluster", ClusterNamespace: "target"}}}
	m := &observingManager{}
	state := &tasks.ReconcileContext{State: tasks.NewState(backup), BackupManager: m, BackupCleaner: &idleCleaner{}}
	r := &Reconciler{Client: cli}
	_, err := r.NewRunner(state, task.NewTableTaskReporter("test")).Run(context.Background())
	require.NoError(t, err)
	require.True(t, m.called)
	require.Equal(t, "target", state.Cluster().Namespace)
	cluster.Annotations = nil
	require.NoError(t, cli.Update(context.Background(), cluster))
	m.called = false
	_, err = r.NewRunner(state, task.NewTableTaskReporter("test")).Run(context.Background())
	require.Error(t, err)
	require.False(t, m.called)
	require.Equal(t, "ClusterAccessFailed", m.reason)
}
