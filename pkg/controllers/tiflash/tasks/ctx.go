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
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	PDClient pdm.PDClient

	Store       *pdv1.Store
	StoreLabels []*metapb.StoreLabel
}

// GetStoreID returns the store ID for PD operations
func (r *ReconcileContext) GetStoreID() string {
	if r.Store == nil {
		return ""
	}
	return r.Store.ID
}

// GetStoreNotExists returns true if the store does not exist in PD
func (r *ReconcileContext) GetStoreNotExists() bool {
	return r.Store == nil
}

// GetPDClient returns the PD client for API operations
func (r *ReconcileContext) GetPDClient() pdm.PDClient {
	return r.PDClient
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(context.Context) task.Result {
		ck := state.Cluster()
		c, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Fail().With("pd client is not registered")
		}
		state.PDClient = c

		if !c.HasSynced() {
			return task.Fail().With("store info is not synced, just wait for next sync")
		}

		s, err := c.Stores().Get(tiflashconfig.GetServiceAddr(state.TiFlash()))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %w", err)
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
		return task.Complete().With("got store info")
	})
}
