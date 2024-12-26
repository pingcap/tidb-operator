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
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/client"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskFinalizerDel(state *ReconcileContext, c client.Client, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		errList := []error{}
		for _, peer := range state.PDSlice() {
			if err := c.Delete(ctx, peer); err != nil {
				if errors.IsNotFound(err) {
					continue
				}

				errList = append(errList, err)
				continue
			}

			// PD controller cannot clean up finalizer after quorum is lost
			// Forcely clean up all finalizers of pd instances
			if err := k8s.RemoveFinalizer(ctx, c, peer); err != nil {
				errList = append(errList, err)
			}
		}

		if len(errList) != 0 {
			return task.Fail().With("failed to delete all pd instances: %v", utilerr.NewAggregate(errList))
		}

		if len(state.PDSlice()) != 0 {
			return task.Fail().With("wait for all pd instances being removed, %v still exists", len(state.PDSlice()))
		}

		if err := k8s.EnsureGroupSubResourceDeleted(ctx, c,
			state.PDGroup().Namespace, state.PDGroup().Name); err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}

		if err := k8s.RemoveFinalizer(ctx, c, state.PDGroup()); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
		}

		m.Deregister(pdm.PrimaryKey(state.Cluster().Namespace, state.Cluster().Name))

		return task.Complete().With("finalizer has been removed")
	})
}

func TaskFinalizerAdd(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func(ctx context.Context) task.Result {
		if err := k8s.EnsureFinalizer(ctx, c, state.PDGroup()); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}
