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
	"time"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

const (
	removingWaitInterval = 10 * time.Second
)

func TaskFinalizerDel(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("FinalizerDel", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		switch {
		case !rtx.Cluster.GetDeletionTimestamp().IsZero():
			if err := k8s.EnsureInstanceSubResourceDeleted(ctx, c,
				rtx.TiFlash.Namespace, rtx.TiFlash.Name, client.GracePeriodSeconds(1)); err != nil {
				return task.Fail().With("cannot delete sub resources: %w", err)
			}
			// whole cluster is deleting
			if err := k8s.RemoveFinalizer(ctx, c, rtx.TiFlash); err != nil {
				return task.Fail().With("cannot remove finalizer: %w", err)
			}

		case rtx.StoreState == v1alpha1.StoreStateRemoving:
			// TODO: Complete task and retrigger reconciliation by polling PD
			return task.Retry(removingWaitInterval).With("wait until the store is removed")

		case rtx.StoreState == v1alpha1.StoreStateRemoved || rtx.StoreID == "":
			if err := k8s.EnsureInstanceSubResourceDeleted(ctx, c,
				rtx.TiFlash.Namespace, rtx.TiFlash.Name, client.GracePeriodSeconds(1)); err != nil {
				return task.Fail().With("cannot delete subresources: %w", err)
			}
			// Store ID is empty may because of tiflash is not initialized
			// TODO: check whether tiflash is initialized
			if err := k8s.RemoveFinalizer(ctx, c, rtx.TiFlash); err != nil {
				return task.Fail().With("cannot remove finalizer: %w", err)
			}
		default:
			// get store info successfully and the store still exists
			if err := rtx.PDClient.DeleteStore(ctx, rtx.StoreID); err != nil {
				return task.Fail().With("cannot delete store %s: %v", rtx.StoreID, err)
			}

			return task.Retry(removingWaitInterval).With("the store is removing")
		}
		return task.Complete().With("finalizer is removed")
	})
}

func TaskFinalizerAdd(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("FinalizerAdd", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		if err := k8s.EnsureFinalizer(ctx, c, rtx.TiFlash); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %w", err)
		}
		return task.Complete().With("finalizer is added")
	})
}
