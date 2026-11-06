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
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type observingManager struct {
	manager.BackupManager
	called bool
}

func (m *observingManager) Sync(context.Context, *br.Backup) error { m.called = true; return nil }

type idleCleaner struct{ manager.BackupCleaner }

func (*idleCleaner) Clean(context.Context, *br.Backup) error { return nil }
func TestRunnerUsesConfiguredTargetWithoutConsent(t *testing.T) {
	for _, withSource := range []bool{false, true} {
		for _, paused := range []bool{false, true} {
			target := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "target"}}
			target.Spec.Paused = paused
			cli := client.NewFakeClient(target)
			if withSource {
				source := target.DeepCopy()
				source.Namespace = "source"
				source.ResourceVersion = ""
				source.Spec.Paused = !paused
				require.NoError(t, cli.Create(t.Context(), source))
			}
			backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "source"}, Spec: br.BackupSpec{BR: &br.BRConfig{Cluster: "cluster", ClusterNamespace: "target"}}}
			m := &observingManager{}
			state := &tasks.ReconcileContext{State: tasks.NewState(backup), BackupManager: m, BackupCleaner: &idleCleaner{}}
			r := &Reconciler{Client: cli}
			_, err := r.NewRunner(state, task.NewTableTaskReporter("test")).Run(t.Context())
			require.NoError(t, err)
			require.Equal(t, !paused, m.called)
			require.Equal(t, "target", state.Cluster().Namespace)
		}
	}
}
