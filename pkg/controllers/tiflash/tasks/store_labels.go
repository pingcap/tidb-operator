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
	"fmt"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskStoreLabels(state *ReconcileContext, c client.Client, m pdm.PDClientManager) task.Task {
	return common.TaskServerLabels[scope.TiFlash](state, c, m, func(ctx context.Context, ls map[string]string) error {
		pc, ok := state.GetPDClient(m)
		if !ok {
			return fmt.Errorf("%w: cannot get pd client", task.ErrWait)
		}
		if !state.PDSynced || state.Store == nil {
			return fmt.Errorf("%w: pd is not synced or store is not found", task.ErrWait)
		}

		logger := logr.FromContextOrDiscard(ctx)
		if !maputil.Contains(state.Store.Labels, ls) {
			if err := pc.Underlay().SetStoreLabels(ctx, state.Store.ID, ls); err != nil {
				return err
			}

			logger.Info("set store labels", "id", state.Store.ID, "currentLabels", state.Store.Labels, "expectedLabels", ls)
			return nil
		}

		logger.Info("store labels has been set", "id", state.Store.ID, "currentLabels", state.Store.Labels, "expectedLabels", ls)
		return nil
	})
}
