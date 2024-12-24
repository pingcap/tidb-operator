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
	"k8s.io/apimachinery/pkg/types"

	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	// TODO: replace all fields in ReconcileContext by State
	State
	Key types.NamespacedName

	PDClient pdm.PDClient
	// this means whether pd is available
	IsAvailable bool
	// This is single truth whether pd is initialized
	Initialized bool
	Healthy     bool
	MemberID    string
	IsLeader    bool

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
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

		state.MemberID = m.ID
		state.IsLeader = m.IsLeader

		// set available and trust health info only when member info is valid
		if !m.Invalid {
			state.IsAvailable = true
			state.Healthy = m.Health
		}

		return task.Complete().With("pd is ready")
	})
}

func CondPDClientIsNotRegisterred(state *ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		// TODO: do not use HasSynced twice, it may return different results
		return state.PDClient == nil || !state.PDClient.HasSynced()
	})
}
