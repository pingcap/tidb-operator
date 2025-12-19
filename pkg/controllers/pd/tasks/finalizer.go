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

	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskDeleteMember(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("DeleteMember", func(ctx context.Context) task.Result {
		// TODO: check whether quorum will be lost?
		if err := state.PDClient.Underlay().DeleteMember(ctx, state.PD().Name); err != nil {
			return task.Fail().With("cannot delete member: %v", err)
		}

		return task.Complete().With("member is deleted")
	})
}
