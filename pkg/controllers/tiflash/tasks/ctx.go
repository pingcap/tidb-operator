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

	"github.com/pingcap/kvproto/pkg/metapb"

	tiflashconfig "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	PDClient pdapi.PDClient

	Store       *pdv1.Store
	StoreID     string
	StoreState  string
	StoreLabels []*metapb.StoreLabel

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(context.Context) task.Result {
		ck := state.Cluster()
		c, ok := cm.Get(pdm.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Complete().With("pd client is not registered")
		}
		state.PDClient = c.Underlay()

		if !c.HasSynced() {
			return task.Complete().With("store info is not synced, just wait for next sync")
		}

		s, err := c.Stores().Get(tiflashconfig.GetServiceAddr(state.TiFlash()))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %w", err)
			}
			return task.Complete().With("store does not exist")
		}

		state.Store, state.StoreID, state.StoreState = s, s.ID, string(s.NodeState)
		state.StoreLabels = make([]*metapb.StoreLabel, len(s.Labels))
		for k, v := range s.Labels {
			state.StoreLabels = append(state.StoreLabels, &metapb.StoreLabel{Key: k, Value: v})
		}
		return task.Complete().With("got store info")
	})
}
