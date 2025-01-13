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

	kvcfg "github.com/pingcap/tidb-operator/pkg/configs/tikv"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	PDClient pdm.PDClient

	StoreExists    bool
	StoreID        string
	StoreState     string
	LeaderEvicting bool

	Store *pdv1.Store

	IsPDAvailable bool

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		c, ok := cm.Get(pdm.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Complete().With("pd client is not registered")
		}
		state.PDClient = c

		if !c.HasSynced() {
			return task.Complete().With("store info is not synced, just wait for next sync")
		}
		state.IsPDAvailable = true

		s, err := c.Stores().Get(kvcfg.GetAdvertiseClientURLs(state.TiKV()))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %w", err)
			}
			return task.Complete().With("store does not exist")
		}
		state.Store, state.StoreID, state.StoreState = s, s.ID, string(s.NodeState)

		// TODO: cache evict leader scheduler info, then we don't need to check suspend here
		if state.Cluster().ShouldSuspendCompute() {
			return task.Complete().With("cluster is suspending")
		}
		scheduler, err := state.PDClient.Underlay().GetEvictLeaderScheduler(ctx, state.StoreID)
		if err != nil {
			return task.Fail().With("pd is unexpectedly crashed: %w", err)
		}
		if scheduler != "" {
			state.LeaderEvicting = true
		}

		return task.Complete().With("get store info")
	})
}
