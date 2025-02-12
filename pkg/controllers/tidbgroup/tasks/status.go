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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskStatusAvailable(state State, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		dbg := state.TiDBGroup()
		dbs := state.TiDBSlice()

		status := metav1.ConditionFalse
		reason := "Unavailable"
		msg := "no tidb instance is available"

		for _, db := range dbs {
			if coreutil.IsHealthy[scope.TiDB](db) {
				status = metav1.ConditionTrue
				reason = "Available"
				msg = "tidb group is available"
			}
		}

		needUpdate := meta.SetStatusCondition(&dbg.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiDBGroupCondAvailable,
			Status:             status,
			ObservedGeneration: dbg.Generation,
			Reason:             reason,
			Message:            msg,
		})
		// TODO(liubo02): only update status once
		if needUpdate {
			if err := c.Status().Update(ctx, dbg); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}
		return task.Complete().With("status is synced")
	})
}
