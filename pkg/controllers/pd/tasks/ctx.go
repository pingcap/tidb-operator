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

	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	// TODO: replace all fields in ReconcileContext by State
	State

	PDClient pdm.PDClient
	// This is single truth whether pd is initialized
	Initialized bool

	ClusterID string
	MemberID  string
	IsLeader  bool
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(_ context.Context) task.Result {
		ck := state.Cluster()
		pc, ok := cm.Get(pdm.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Wait().With("pd client has not been registered yet")
		}

		state.PDClient = pc

		if !pc.HasSynced() {
			return task.Complete().With("context without member info is completed, cache of pd info is not synced")
		}

		state.Initialized = true

		m, err := pc.Members().Get(state.PD().Name)
		if err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("context without member info is completed, pd is not initialized")
			}
			return task.Fail().With("cannot get member: %w", err)
		}

		state.ClusterID = m.ClusterID
		state.MemberID = m.ID
		state.IsLeader = m.IsLeader

		// set available and trust health info only when member info is valid
		if !m.Invalid && m.Health {
			state.SetHealthy()
			return task.Complete().With("pd is ready")
		}

		return task.Wait().With("pd is unready, invalid: %v, health: %v", m.Invalid, m.Health)
	})
}

func CondPDClientIsNotRegisterred(state *ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		// TODO: do not use HasSynced twice, it may return different results
		return state.PDClient == nil || !state.PDClient.HasSynced()
	})
}
