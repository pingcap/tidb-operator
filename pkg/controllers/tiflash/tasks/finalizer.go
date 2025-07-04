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

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	removingWaitInterval = 10 * time.Second
)

func TaskFinalizerDel(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		switch {
		case !state.Cluster().GetDeletionTimestamp().IsZero():
			return handleClusterDeletion(ctx, c, state)

		case state.GetStoreState() == v1alpha1.StoreStateRemoving:
			return task.Retry(removingWaitInterval).With("wait until the store is removed")

		case state.GetStoreState() == v1alpha1.StoreStateRemoved || state.StoreNotExists:
			return handleStoreRemovedOrNotExists(ctx, c, state)

		default:
			return handleStoreExists(ctx, state)
		}
	})
}

func handleClusterDeletion(ctx context.Context, c client.Client, state *ReconcileContext) task.Result {
	if pod := state.Pod(); pod != nil {
		if err := k8s.RemoveFinalizer(ctx, c, pod); err != nil {
			return task.Fail().With("cannot remove finalizer: %w", err)
		}
	}

	wait, err := EnsureSubResourcesDeleted(ctx, c, state.TiFlash())
	if err != nil {
		return task.Fail().With("cannot delete sub resources: %w", err)
	}

	if wait {
		return task.Retry(task.DefaultRequeueAfter).With("wait all subresources deleted")
	}

	if err := k8s.RemoveFinalizer(ctx, c, state.TiFlash()); err != nil {
		return task.Fail().With("cannot remove finalizer: %w", err)
	}
	return task.Complete().With("finalizer is removed")
}

func handleStoreRemovedOrNotExists(ctx context.Context, c client.Client, state *ReconcileContext) task.Result {
	if pod := state.Pod(); pod != nil {
		if err := k8s.RemoveFinalizer(ctx, c, pod); err != nil {
			return task.Fail().With("cannot remove finalizer: %w", err)
		}
		return task.Retry(task.DefaultRequeueAfter).With("pod finalizer removed, wait for pod to be deleted")
	}

	wait, err := EnsureSubResourcesDeleted(ctx, c, state.TiFlash())
	if err != nil {
		return task.Fail().With("cannot delete sub resources: %w", err)
	}

	if wait {
		return task.Retry(task.DefaultRequeueAfter).With("wait all subresources deleted")
	}

	if err := k8s.RemoveFinalizer(ctx, c, state.TiFlash()); err != nil {
		return task.Fail().With("cannot remove finalizer: %w", err)
	}
	return task.Complete().With("finalizer is removed")
}

func handleStoreExists(ctx context.Context, state *ReconcileContext) task.Result {
	if err := state.PDClient.Underlay().DeleteStore(ctx, state.StoreID); err != nil {
		return task.Fail().With("cannot delete store %s: %v", state.StoreID, err)
	}
	return task.Retry(removingWaitInterval).With("the store is removing")
}

func EnsureSubResourcesDeleted(ctx context.Context, c client.Client, f *v1alpha1.TiFlash) (wait bool, _ error) {
	wait1, err := k8s.DeleteInstanceSubresource(ctx, c, runtime.FromTiFlash(f), &corev1.PodList{})
	if err != nil {
		return false, err
	}
	wait2, err := k8s.DeleteInstanceSubresource(ctx, c, runtime.FromTiFlash(f), &corev1.ConfigMapList{})
	if err != nil {
		return false, err
	}
	wait3, err := k8s.DeleteInstanceSubresource(ctx, c, runtime.FromTiFlash(f), &corev1.PersistentVolumeClaimList{})
	if err != nil {
		return false, err
	}

	return wait1 || wait2 || wait3, nil
}
