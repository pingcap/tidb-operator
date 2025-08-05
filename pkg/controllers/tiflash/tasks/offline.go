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

	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// TaskOfflineStore handles the two-step store deletion process based on spec.offline field.
// This implements the state machine for offline operations: Pending -> Active -> Completed/Failed/Canceled.
func TaskOfflineStore(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("OfflineTiFlashStore", func(ctx context.Context) task.Result {
		tiflash := state.Instance()
		tiflash.Spec.Offline = true
		return common.TaskOfflineStoreStateMachine(ctx, state, tiflash, "tiflash", &common.DummyStoreOfflineHook{})
	})
}
