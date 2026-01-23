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

package tasks

import (
	"context"

	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ReconcileContext struct {
	// TODO: replace all fields in ReconcileContext by State
	State

	PDClient pdm.PDClient
}

func TaskContextClient(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextClient", func(_ context.Context) task.Result {
		ck := state.Cluster()
		key := timanager.PrimaryKey(ck.Namespace, ck.Name)
		pc, ok := cm.Get(key)
		if !ok {
			return task.Wait().With("pd client has not been registered yet")
		}
		state.PDClient = pc
		return task.Complete().With("pd client is ready")
	})
}
