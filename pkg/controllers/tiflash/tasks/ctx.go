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

	"github.com/pingcap/kvproto/pkg/metapb"
	"k8s.io/apimachinery/pkg/api/errors"

	tiflashconfig "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
	"github.com/pingcap/tidb-operator/pkg/tiflashapi/v1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	fm "github.com/pingcap/tidb-operator/pkg/timanager/tiflash"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

type ReconcileContext struct {
	State

	PDClient pdm.PDClient

	Store       *pdv1.Store
	StoreLabels []*metapb.StoreLabel

	StoreStatus tiflashapi.Status

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

// GetPDClient returns the PD client for API operations
func (r *ReconcileContext) GetPDClient() pdm.PDClient {
	return r.PDClient
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager, fcm fm.TiFlashClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		c, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			// We have to retry here because the store may have been removed and cannot trigger the changes
			return task.Retry(defaultTaskWaitDuration).With("pd client is not registered")
		}
		state.PDClient = c

		if !c.HasSynced() {
			// We have to retry here because the store may have been removed and cannot trigger the changes
			return task.Retry(defaultTaskWaitDuration).With("store info is not synced, just wait for next sync")
		}

		state.PDSynced = true

		f := state.Object()
		if err := fcm.Register(f); err != nil {
			return task.Fail().With("cannot register tiflash client")
		}
		fc, ok := fcm.Get(fm.Key(f))
		if !ok {
			return task.Fail().With("tiflash client is not registered")
		}

		s, err := c.Stores().Get(tiflashconfig.GetServiceAddr(state.TiFlash()))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %v", err)
			}
			return task.Complete().With("store does not exist")
		}

		state.Store = s
		state.SetStoreState(string(s.NodeState))
		state.SetRegionCount(s.RegionCount)
		state.SetStoreBusy(s.IsBusy)

		state.StoreLabels = make([]*metapb.StoreLabel, len(s.Labels))
		for k, v := range s.Labels {
			state.StoreLabels = append(state.StoreLabels, &metapb.StoreLabel{Key: k, Value: v})
		}

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
			state.StoreStatus = status
		}

		return task.Complete().With("get store info, state: %s, leader count: %v, region count: %v, busy: %v, status: %v",
			state.GetStoreState(),
			state.GetLeaderCount(),
			state.GetRegionCount(),
			state.IsStoreBusy(),
			status,
		)
	})
}
