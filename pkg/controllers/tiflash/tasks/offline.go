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
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	// removingWaitInterval is the interval to wait for store removal.
	removingWaitInterval = 10 * time.Second
)

func TaskOfflineStore(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("OfflineStore", func(ctx context.Context) task.Result {
		if state.Store == nil {
			return task.Complete().With("store is not exists, no need to offline")
		}

		if state.PDClient == nil {
			return task.Fail().With("pd client is not registered")
		}
		if err := state.PDClient.Underlay().DeleteStore(ctx, state.Store.ID); err != nil {
			return task.Fail().With("cannot delete store %s: %w", state.Store.ID, err)
		}
		state.SetStoreState(v1alpha1.StoreStateRemoving)
		return task.Retry(removingWaitInterval).With("the store is removing")
	})
}
