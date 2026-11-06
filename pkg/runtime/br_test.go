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

package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

func TestLogBackupFinalizerRequiresStopCompletion(t *testing.T) {
	for _, policy := range []br.CleanPolicyType{br.CleanPolicyTypeRetain, br.CleanPolicyTypeOnFailure, br.CleanPolicyTypeDelete} {
		for _, phase := range []br.BackupConditionType{br.BackupInvalid, br.BackupRetryTheFailed, br.BackupStopped, br.BackupClean} {
			for _, stopPhase := range []br.BackupConditionType{br.BackupRunning, br.BackupFailed, br.BackupComplete} {
				backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "test", DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time}, Finalizers: []string{meta.Finalizer}}, Spec: br.BackupSpec{Mode: br.BackupModeLog, CleanPolicy: policy}, Status: br.BackupStatus{CommitTs: "1", Phase: phase, Conditions: []metav1.Condition{{Type: string(br.BackupClean), Status: metav1.ConditionTrue}}, LogSubCommandStatuses: map[br.LogSubCommandType]br.LogSubCommandStatus{br.LogStopCommand: {Phase: stopPhase}}}}
				cli := client.NewFakeClient(backup)
				wantRemove := phase == br.BackupStopped || stopPhase == br.BackupComplete
				require.Equal(t, wantRemove, runtime.FromBackup(backup).NeedRemoveFinalizer(), "policy=%s phase=%s stop=%s", policy, phase, stopPhase)
				require.NoError(t, common.JobLifecycleManager.Sync(t.Context(), runtime.FromBackup(backup), cli))
				require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(backup), backup))
				require.Equal(t, !wantRemove, len(backup.Finalizers) > 0, "policy=%s phase=%s stop=%s", policy, phase, stopPhase)
			}
		}
	}
}
