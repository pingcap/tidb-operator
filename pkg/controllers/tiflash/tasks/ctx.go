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

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tiflashapi/v1"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	fm "github.com/pingcap/tidb-operator/v2/pkg/timanager/tiflash"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/third_party/kubernetes/pkg/controller/statefulset"
)

type ReconcileContext struct {
	State

	Store *pdv1.Store

	// PDSynced means the cache is ready to use
	// TODO: combine with store state
	PDSynced bool
}

// GetStoreID returns the store ID for PD operations
func (r *ReconcileContext) GetStoreID() string {
	if r.Store == nil {
		return ""
	}
	return r.Store.ID
}

// StoreNotExists returns true if the store does not exist in PD
func (r *ReconcileContext) StoreNotExists() bool {
	return r.Store == nil
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager, fcm fm.TiFlashClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx context.Context) task.Result {
		c, ok := state.GetPDClient(cm)
		if !ok {
			// We have to retry here because the store may be removed and cannot trigger the changes
			return task.Retry(task.DefaultRequeueAfter).With("pd client is not registered")
		}

		state.PDSynced = true

		ck := state.Cluster()
		f := state.Object()
		if err := fcm.Register(f); err != nil {
			return task.Fail().With("cannot register tiflash client")
		}

		fc, ok := fcm.Get(fm.Key(f))
		if !ok {
			return task.Fail().With("tiflash client is not registered")
		}

		addr := coreutil.InstanceAdvertiseAddress[scope.TiFlash](ck, f, coreutil.TiFlashFlashPort(f))
		s, err := c.Stores().Get(addr)
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %v", err)
			}
			return task.Complete().With("store does not exist")
		}

		state.Store = s
		state.SetStoreState(s.NodeState)
		state.SetRegionCount(s.RegionCount)
		state.SetStoreBusy(s.IsBusy)

		if state.FeatureGates().Enabled(metav1alpha1.UseTiFlashReadyAPI) {
			state.SetHealthy()
		} else {
			pod := state.Pod()

			var status tiflashapi.Status
			if pod != nil && statefulset.IsPodReady(pod) {
				s, err := fc.GetStoreStatus(ctx)
				if err != nil {
					return task.Fail().With("failed to get store status: %v", err)
				}
				status = s
				// TODO: check IsBusy ?
				if status == tiflashapi.Running {
					state.SetHealthy()
				}
			}
		}

		return task.Complete().With("get store info, state: %s, leader count: %v, region count: %v, busy: %v, health: %v",
			state.GetStoreState(),
			state.GetLeaderCount(),
			state.GetRegionCount(),
			state.IsStoreBusy(),
			state.IsHealthy(),
		)
	})
}
