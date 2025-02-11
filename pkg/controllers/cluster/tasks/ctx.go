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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	Cluster       *v1alpha1.Cluster
	PDGroup       *v1alpha1.PDGroup
	TiKVGroups    []*v1alpha1.TiKVGroup
	TiFlashGroups []*v1alpha1.TiFlashGroup
	TiDBGroups    []*v1alpha1.TiDBGroup
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

	var cluster v1alpha1.Cluster
	if err := t.Client.Get(ctx, rtx.Key, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			return task.Fail().With("can't get tidb cluster: %w", err)
		}

		return task.Complete().Break().With("tidb cluster has been deleted")
	}
	rtx.Cluster = &cluster

	if coreutil.ShouldPauseReconcile(rtx.Cluster) {
		return task.Complete().Break().With("cluster reconciliation is paused")
	}

	var pdGroupList v1alpha1.PDGroupList
	if err := t.Client.List(ctx, &pdGroupList, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list pd group: %w", err)
	}
	if len(pdGroupList.Items) > 1 {
		return task.Fail().With("more than one pd group")
	}
	if len(pdGroupList.Items) != 0 {
		rtx.PDGroup = &pdGroupList.Items[0]
	}

	var tikvGroupList v1alpha1.TiKVGroupList
	if err := t.Client.List(ctx, &tikvGroupList, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list tikv group: %w", err)
	}
	for i := range tikvGroupList.Items {
		rtx.TiKVGroups = append(rtx.TiKVGroups, &tikvGroupList.Items[i])
	}

	var tiflashGroupList v1alpha1.TiFlashGroupList
	if err := t.Client.List(ctx, &tiflashGroupList, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list tiflash group: %w", err)
	}
	for i := range tiflashGroupList.Items {
		rtx.TiFlashGroups = append(rtx.TiFlashGroups, &tiflashGroupList.Items[i])
	}

	var tidbGroupList v1alpha1.TiDBGroupList
	if err := t.Client.List(ctx, &tidbGroupList, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list tidb group: %w", err)
	}
	for i := range tidbGroupList.Items {
		rtx.TiDBGroups = append(rtx.TiDBGroups, &tidbGroupList.Items[i])
	}

	return task.Complete().With("new context completed")
}
