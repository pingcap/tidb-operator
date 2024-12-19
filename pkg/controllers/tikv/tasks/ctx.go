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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	kvcfg "github.com/pingcap/tidb-operator/pkg/configs/tikv"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	PDClient pdapi.PDClient

	Healthy bool

	StoreExists    bool
	StoreID        string
	StoreState     string
	LeaderEvicting bool

	Suspended bool

	Cluster *v1alpha1.Cluster
	TiKV    *v1alpha1.TiKV
	Pod     *corev1.Pod

	Store *pdv1.Store

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
}

func (ctx *ReconcileContext) Self() *ReconcileContext {
	return ctx
}

func TaskContextTiKV(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextTiKV", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var tikv v1alpha1.TiKV
		if err := c.Get(ctx, rtx.Key, &tikv); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get tikv instance: %w", err)
			}

			return task.Complete().With("tikv instance has been deleted")
		}
		rtx.TiKV = &tikv
		return task.Complete().With("tikv is set")
	})
}

func TaskContextInfoFromPD(cm pdm.PDClientManager) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		c, ok := cm.Get(pdm.PrimaryKey(rtx.TiKV.Namespace, rtx.TiKV.Spec.Cluster.Name))
		if !ok {
			return task.Complete().With("pd client is not registered")
		}
		rtx.PDClient = c.Underlay()

		if !c.HasSynced() {
			return task.Complete().With("store info is not synced, just wait for next sync")
		}

		s, err := c.Stores().Get(kvcfg.GetAdvertiseClientURLs(rtx.TiKV))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %w", err)
			}
			return task.Complete().With("store does not exist")
		}
		rtx.Store, rtx.StoreID, rtx.StoreState = s, s.ID, string(s.NodeState)

		// TODO: cache evict leader scheduler info, then we don't need to check suspend here
		if rtx.Cluster.ShouldSuspendCompute() {
			return task.Complete().With("cluster is suspending")
		}
		scheduler, err := rtx.PDClient.GetEvictLeaderScheduler(ctx, rtx.StoreID)
		if err != nil {
			return task.Fail().With("pd is unexpectedly crashed: %w", err)
		}
		if scheduler != "" {
			rtx.LeaderEvicting = true
		}

		return task.Complete().With("get store info")
	})
}

func TaskContextCluster(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextCluster", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiKV.Spec.Cluster.Name,
			Namespace: rtx.TiKV.Namespace,
		}, &cluster); err != nil {
			return task.Fail().With("cannot find cluster %s: %w", rtx.TiKV.Spec.Cluster.Name, err)
		}
		rtx.Cluster = &cluster
		return task.Complete().With("cluster is set")
	})
}

func TaskContextPod(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextPod", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var pod corev1.Pod
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiKV.PodName(),
			Namespace: rtx.TiKV.Namespace,
		}, &pod); err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod is not created")
			}
			return task.Fail().With("failed to get pod of tikv: %w", err)
		}

		rtx.Pod = &pod
		if !rtx.Pod.GetDeletionTimestamp().IsZero() {
			rtx.PodIsTerminating = true
		}
		return task.Complete().With("pod is set")
	})
}

func CondTiKVHasBeenDeleted() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().TiKV == nil
	})
}

func CondTiKVIsDeleting() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return !ctx.Self().TiKV.GetDeletionTimestamp().IsZero()
	})
}

func CondClusterIsPaused() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.ShouldPauseReconcile()
	})
}

func CondClusterIsSuspending() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.ShouldSuspendCompute()
	})
}
