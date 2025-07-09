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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	pdcfg "github.com/pingcap/tidb-operator/pkg/configs/pd"
	"github.com/pingcap/tidb-operator/pkg/timanager"
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
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		pc, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Wait().With("pd client has not been registered yet")
		}

		state.PDClient = pc

		if !pc.HasSynced() {
			return task.Complete().With("context without member info is completed, cache of pd info is not synced")
		}

		state.Initialized = true

		pd := state.PD()
		m, err := pc.Members().Get(pd.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("context without member info is completed, pd is not initialized")
			}
			return task.Fail().With("cannot get member: %w", err)
		}

		state.ClusterID = m.ClusterID
		state.MemberID = m.ID
		state.IsLeader = m.IsLeader

		logger := logr.FromContextOrDiscard(ctx)
		ready, err := state.PDClient.Underlay().GetMemberReady(ctx, getPDURL(ck, pd), pd.Spec.Version)
		if err != nil {
			// Do not return error here, because the pd pod may not be created yet.
			logger.Error(err, "failed to get member ready")
		}

		// set available and trust health info only when member info is valid
		if !m.Invalid && m.Health && ready {
			state.SetHealthy()
			return task.Complete().With("pd is ready")
		}

		return task.Wait().With("pd is unready, invalid: %v, health: %v, ready: %v", m.Invalid, m.Health, ready)
	})
}

func CondPDClientIsNotRegisterred(state *ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		// TODO: do not use HasSynced twice, it may return different results
		return state.PDClient == nil || !state.PDClient.HasSynced()
	})
}

func getPDURL(cluster *v1alpha1.Cluster, pd *v1alpha1.PD) string {
	scheme := "http"
	if coreutil.IsTLSClusterEnabled(cluster) {
		scheme = "https"
	}
	return pdcfg.GetAdvertiseClientURLs(pd, scheme)
}
