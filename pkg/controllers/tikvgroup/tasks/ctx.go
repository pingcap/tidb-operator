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

	TiKVGroup      *v1alpha1.TiKVGroup
	Peers          []*v1alpha1.TiKV
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

	var kvg v1alpha1.TiKVGroup
	if err := t.Client.Get(ctx, rtx.Key, &kvg); err != nil {
		if !errors.IsNotFound(err) {
			return task.Fail().With("can't get tikv group: %w", err)
		}

		return task.Complete().Break().With("tikv group has been deleted")
	}
	rtx.TiKVGroup = &kvg

	var cluster v1alpha1.Cluster
	if err := t.Client.Get(ctx, client.ObjectKey{
		Name:      kvg.Spec.Cluster.Name,
		Namespace: kvg.Namespace,
	}, &cluster); err != nil {
		return task.Fail().With("cannot find cluster %s: %w", kvg.Spec.Cluster.Name, err)
	}
	rtx.Cluster = &cluster

	if cluster.ShouldPauseReconcile() {
		return task.Complete().Break().With("cluster reconciliation is paused")
	}

	var kvList v1alpha1.TiKVList
	if err := t.Client.List(ctx, &kvList, client.InNamespace(kvg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
		v1alpha1.LabelKeyGroup:     kvg.Name,
	}); err != nil {
		return task.Fail().With("cannot list tikv peers: %w", err)
	}

	rtx.Peers = make([]*v1alpha1.TiKV, len(kvList.Items))
	rtx.Suspended = len(kvList.Items) > 0
	for i := range kvList.Items {
		rtx.Peers[i] = &kvList.Items[i]
		if !meta.IsStatusConditionTrue(kvList.Items[i].Status.Conditions, v1alpha1.TiKVCondSuspended) {
			// TiKV Group is not suspended if any of its members is not suspended
			rtx.Suspended = false
		}
	}
	slices.SortFunc(rtx.Peers, func(a, b *v1alpha1.TiKV) int {
		return cmp.Compare(a.Name, b.Name)
	})

	rtx.UpgradeChecker = action.NewUpgradeChecker(t.Client, rtx.Cluster, t.Logger)
	return task.Complete().With("new context completed")
}
