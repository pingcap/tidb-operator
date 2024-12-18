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

	"github.com/pingcap/kvproto/pkg/metapb"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tiflashconfig "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
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

	Store       *pdv1.Store
	StoreID     string
	StoreState  string
	StoreLabels []*metapb.StoreLabel

	Cluster      *v1alpha1.Cluster
	TiFlash      *v1alpha1.TiFlash
	TiFlashGroup *v1alpha1.TiFlashGroup
	Pod          *corev1.Pod

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

func TaskContextTiFlash(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextTiFlash", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()

		var tiflash v1alpha1.TiFlash
		if err := c.Get(ctx, rtx.Key, &tiflash); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get tiflash instance: %w", err)
			}

			return task.Complete().With("tiflash instance has been deleted")
		}
		rtx.TiFlash = &tiflash
		return task.Complete().With("tiflash is set")
	})
}

func TaskContextTiFlashGroup(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextTiFlashGroup", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()

		if len(rtx.TiFlash.OwnerReferences) == 0 {
			return task.Fail().With("tiflash instance has no owner, this should not happen")
		}

		var tiflashGroup v1alpha1.TiFlashGroup
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiFlash.OwnerReferences[0].Name, // only one owner now
			Namespace: rtx.TiFlash.Namespace,
		}, &tiflashGroup); err != nil {
			return task.Fail().With("cannot find tiflash group %s: %w", rtx.TiFlash.OwnerReferences[0].Name, err)
		}
		rtx.TiFlashGroup = &tiflashGroup
		return task.Complete().With("tiflash group is set")
	})
}

func CondTiFlashHasBeenDeleted() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().TiFlash == nil
	})
}

func CondTiFlashIsDeleting() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return !ctx.Self().TiFlash.GetDeletionTimestamp().IsZero()
	})
}

func TaskContextCluster(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextCluster", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiFlash.Spec.Cluster.Name,
			Namespace: rtx.TiFlash.Namespace,
		}, &cluster); err != nil {
			return task.Fail().With("cannot find cluster %s: %w", rtx.TiFlash.Spec.Cluster.Name, err)
		}
		rtx.Cluster = &cluster
		return task.Complete().With("cluster is set")
	})
}

func CondClusterIsSuspending() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.ShouldSuspendCompute()
	})
}

func TaskContextPod(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextPod", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var pod corev1.Pod
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiFlash.PodName(),
			Namespace: rtx.TiFlash.Namespace,
		}, &pod); err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod is not created")
			}
			return task.Fail().With("failed to get pod of pd: %w", err)
		}

		rtx.Pod = &pod
		if !rtx.Pod.GetDeletionTimestamp().IsZero() {
			rtx.PodIsTerminating = true
		}
		return task.Complete().With("pod is set")
	})
}

func CondClusterIsPaused() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.ShouldPauseReconcile()
	})
}

func TaskContextInfoFromPD(cm pdm.PDClientManager) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		c, ok := cm.Get(pdm.PrimaryKey(rtx.TiFlash.Namespace, rtx.TiFlash.Spec.Cluster.Name))
		if !ok {
			return task.Complete().With("pd client is not registered")
		}
		rtx.PDClient = c.Underlay()

		if !c.HasSynced() {
			return task.Complete().With("store info is not synced, just wait for next sync")
		}

		s, err := c.Stores().Get(tiflashconfig.GetServiceAddr(rtx.TiFlash))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %w", err)
			}
			return task.Complete().With("store does not exist")
		}

		rtx.Store, rtx.StoreID, rtx.StoreState = s, s.ID, string(s.NodeState)
		rtx.StoreLabels = make([]*metapb.StoreLabel, len(s.Labels))
		for k, v := range s.Labels {
			rtx.StoreLabels = append(rtx.StoreLabels, &metapb.StoreLabel{Key: k, Value: v})
		}
		return task.Complete().With("got store info")
	})
}
