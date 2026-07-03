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

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func CondObjectIsOfflineForGracefulScaleIn(state State) task.Condition {
	return task.CondFunc(func() bool {
		tiproxy := state.Object()
		if tiproxy == nil || !tiproxy.GetDeletionTimestamp().IsZero() {
			return false
		}
		return coreutil.IsOffline[scope.TiProxy](tiproxy) && coreutil.GracefulScaleInEnabled(tiproxy)
	})
}

func CondOfflineScaleInDrainComplete(state State) task.Condition {
	return task.CondFunc(func() bool {
		tiproxy := state.Object()
		if tiproxy == nil || !tiproxy.GetDeletionTimestamp().IsZero() {
			return false
		}
		if !coreutil.IsOffline[scope.TiProxy](tiproxy) || !coreutil.GracefulScaleInEnabled(tiproxy) {
			return false
		}
		complete, err := offlineScaleInDrainComplete(tiproxy, state.Pod())
		if err != nil {
			return false
		}
		return complete
	})
}

func TaskOfflineScaleInDrain(state State, c client.Client) task.Task {
	return task.NameTaskFunc("OfflineScaleInDrain", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Wait().With("wait for tiproxy pod before graceful scale-in drain")
		}

		retryAfter, err := drainPodForGracefulShutdown(ctx, c, state, pod, true)
		if err != nil {
			return task.Fail().With("cannot drain tiproxy pod for graceful scale-in: %v", err)
		}
		if retryAfter > 0 {
			return task.Retry(retryAfter).With("wait for tiproxy pod to finish graceful scale-in drain")
		}
		return task.Complete().With("tiproxy pod graceful scale-in drain is complete")
	})
}

func TaskDeleteOfflinedTiProxy(state State, c client.Client) task.Task {
	return task.NameTaskFunc("DeleteOfflinedTiProxy", func(ctx context.Context) task.Result {
		tiproxy := state.Object()
		if tiproxy.GetDeletionTimestamp().IsZero() {
			if err := c.Delete(ctx, tiproxy); err != nil {
				return task.Fail().With("cannot delete offlined tiproxy: %v", err)
			}
		}
		return task.Wait().With("wait until offlined tiproxy deletion is watched")
	})
}

func TaskReviveFromScaleIn(state State, c client.Client) task.Task {
	return task.NameTaskFunc("ReviveFromScaleIn", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		tiproxy := state.Object()
		if coreutil.IsOffline[scope.TiProxy](tiproxy) {
			return task.Complete().With("tiproxy is still marked offline")
		}

		pod := state.Pod()
		if !needsScaleInRevive(tiproxy, pod) {
			return task.Complete().With("tiproxy does not need scale-in revive")
		}

		newTiProxy := tiproxy.DeepCopy()
		if clearGracefulDrainAnnotationsOnCR(newTiProxy) {
			if err := c.Update(ctx, newTiProxy); err != nil {
				return task.Fail().With("cannot clear tiproxy graceful shutdown state on CR: %v", err)
			}
			state.SetObject(newTiProxy)
			tiproxy = newTiProxy
		}

		// If the pod is gone, there is no live TiProxy process holding a health override. Skip the
		// API call (which would otherwise keep failing and block the later TaskPod from recreating
		// the pod) and let the normal process recreate it; a fresh pod starts healthy without any
		// override.
		if pod == nil {
			return task.Complete().With("tiproxy pod is gone, cleared graceful scale-in drain state and will recreate pod")
		}

		if !ensureTiProxyHealthOverrideCleared(ctx, state, c, logger) {
			return task.Retry(task.DefaultRequeueAfter).With("wait for tiproxy health override to be cleared")
		}

		if pod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] != "" {
			newPod := pod.DeepCopy()
			delete(newPod.Annotations, v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime)
			if err := c.Update(ctx, newPod); err != nil {
				return task.Fail().With("cannot clear tiproxy graceful scale-in drain state on pod: %v", err)
			}
			state.SetPod(newPod)
		}

		state.SetHealthy()
		return task.Complete().With("tiproxy is revived from graceful scale-in drain")
	})
}
