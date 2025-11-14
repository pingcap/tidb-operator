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

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ReconcileContext struct {
	// TODO: replace all fields in ReconcileContext by State
	State

	PDClient  pdm.PDClient
	TSOClient tsom.TSOClient

	CacheSynced       bool
	TSOMemberNotFound bool
	IsLeader          bool
}

func TaskContextClient(state *ReconcileContext, cm pdm.PDClientManager, tsocm tsom.TSOClientManager) task.Task {
	return task.NameTaskFunc("ContextClient", func(_ context.Context) task.Result {
		ck := state.Cluster()
		key := timanager.PrimaryKey(ck.Namespace, ck.Name)
		tc, ok := tsocm.Get(key)
		if !ok {
			return task.Wait().With("tso client has not been registered yet")
		}
		state.TSOClient = tc

		pc, ok := cm.Get(key)
		if !ok {
			return task.Wait().With("pd client has not been registered yet")
		}

		if !pc.HasSynced() {
			return task.Complete().With("context without member info is completed, cache of tso info is not synced")
		}

		state.PDClient = pc
		state.CacheSynced = true

		tso := state.Object()
		m, err := pc.TSOMembers().Get(tso.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				state.TSOMemberNotFound = true
				return task.Complete().With("context without tso member info is completed, tso member is not found")
			}
			return task.Fail().With("cannot get member: %v", err)
		}

		state.IsLeader = m.IsLeader

		return task.Complete().With("pd client is ready")
	})
}
