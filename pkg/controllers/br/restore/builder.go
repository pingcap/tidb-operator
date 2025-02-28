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
	"github.com/pingcap/tidb-operator/pkg/controllers/br/restore/tasks"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get cluster
		common.TaskContextCluster[scope.Restore](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(
			common.CondClusterIsSuspending(state),
		),

		tasks.TaskRestoreManager(state, r.Client, r.EventRecorder),
	)

	return runner
}
