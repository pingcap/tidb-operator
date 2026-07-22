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

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskPlacementPolicyRefBlock(state common.TiKVGroupState, c client.Client) task.Task {
	return task.NameTaskFunc("PlacementPolicyRefBlock", func(ctx context.Context) task.Result {
		kvg := state.TiKVGroup()
		policies, err := apicall.ListPlacementPoliciesReferencingTiKVGroup(ctx, c, kvg)
		if err != nil {
			return task.Fail().With("cannot list placement policies: %w", err)
		}

		names := make([]string, 0, len(policies))
		for _, policy := range policies {
			names = append(names, policy.Name)
		}
		if len(names) != 0 {
			return task.Wait().With(
				"TiKVGroup deletion is blocked by referencing placement policies: %v",
				names,
			)
		}

		return task.Complete().With("TiKVGroup has no referencing placement policies")
	})
}
