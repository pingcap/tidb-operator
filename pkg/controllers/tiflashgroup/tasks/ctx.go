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
	"cmp"
	"context"
	"slices"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/action"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	Suspended bool

	Cluster *v1alpha1.Cluster

	TiFlashGroup   *v1alpha1.TiFlashGroup
	Peers          []*v1alpha1.TiFlash
	UpgradeChecker action.UpgradeChecker

	// Status fields
	v1alpha1.CommonStatus
}

func (ctx *ReconcileContext) Self() *ReconcileContext {
	return ctx
}

type TaskContext struct {
	Logger logr.Logger
	Client client.Client
}

func NewTaskContext(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskContext{
		Logger: logger,
		Client: c,
	}
}

func (*TaskContext) Name() string {
	return "Context"
}

func (t *TaskContext) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	var flashg v1alpha1.TiFlashGroup
	if err := t.Client.Get(ctx, rtx.Key, &flashg); err != nil {
		if !errors.IsNotFound(err) {
			return task.Fail().With("can't get tiflash group: %w", err)
		}

		return task.Complete().Break().With("tiflash group has been deleted")
	}
	rtx.TiFlashGroup = &flashg

	var cluster v1alpha1.Cluster
	if err := t.Client.Get(ctx, client.ObjectKey{
		Name:      flashg.Spec.Cluster.Name,
		Namespace: flashg.Namespace,
	}, &cluster); err != nil {
		return task.Fail().With("cannot find cluster %s: %w", flashg.Spec.Cluster.Name, err)
	}
	rtx.Cluster = &cluster

	if cluster.ShouldPauseReconcile() {
		return task.Complete().Break().With("cluster reconciliation is paused")
	}

	var flashList v1alpha1.TiFlashList
	if err := t.Client.List(ctx, &flashList, client.InNamespace(flashg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
		v1alpha1.LabelKeyGroup:     flashg.Name,
	}); err != nil {
		return task.Fail().With("cannot list tiflash peers: %w", err)
	}

	rtx.Peers = make([]*v1alpha1.TiFlash, len(flashList.Items))
	rtx.Suspended = len(flashList.Items) > 0
	for i := range flashList.Items {
		rtx.Peers[i] = &flashList.Items[i]
		if !meta.IsStatusConditionTrue(flashList.Items[i].Status.Conditions, v1alpha1.TiKVCondSuspended) {
			// TiFlash Group is not suspended if any of its members is not suspended
			rtx.Suspended = false
		}
	}
	slices.SortFunc(rtx.Peers, func(a, b *v1alpha1.TiFlash) int {
		return cmp.Compare(a.Name, b.Name)
	})

	rtx.UpgradeChecker = action.NewUpgradeChecker(t.Client, rtx.Cluster, t.Logger)
	return task.Complete().With("new context completed")
}
