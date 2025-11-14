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
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	Cluster          *v1alpha1.Cluster
	PDGroup          *v1alpha1.PDGroup
	TiKVGroups       []*v1alpha1.TiKVGroup
	TiFlashGroups    []*v1alpha1.TiFlashGroup
	TiDBGroups       []*v1alpha1.TiDBGroup
	TiCDCGroups      []*v1alpha1.TiCDCGroup
	TSOGroups        []*v1alpha1.TSOGroup
	SchedulingGroups []*v1alpha1.SchedulingGroup
	SchedulerGroups  []*v1alpha1.SchedulerGroup
	TiProxyGroups    []*v1alpha1.TiProxyGroup
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

// TODO: refactor to use task v3
//
//nolint:gocyclo,staticcheck // refactor later
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

	var tgl v1alpha1.TSOGroupList
	if err := t.Client.List(ctx, &tgl, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list tso group: %w", err)
	}
	for i := range tgl.Items {
		rtx.TSOGroups = append(rtx.TSOGroups, &tgl.Items[i])
	}

	var sgl v1alpha1.SchedulingGroupList
	if err := t.Client.List(ctx, &sgl, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list scheduling group: %w", err)
	}
	for i := range sgl.Items {
		rtx.SchedulingGroups = append(rtx.SchedulingGroups, &sgl.Items[i])
	}

	var dsgl v1alpha1.SchedulerGroupList
	if err := t.Client.List(ctx, &dsgl, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list scheduler group: %w", err)
	}
	for i := range dsgl.Items {
		rtx.SchedulerGroups = append(rtx.SchedulerGroups, &dsgl.Items[i])
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

	var ticdcGroupList v1alpha1.TiCDCGroupList
	if err := t.Client.List(ctx, &ticdcGroupList, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list ticdc group: %w", err)
	}
	for i := range ticdcGroupList.Items {
		rtx.TiCDCGroups = append(rtx.TiCDCGroups, &ticdcGroupList.Items[i])
	}

	var tiproxyGroupList v1alpha1.TiProxyGroupList
	if err := t.Client.List(ctx, &tiproxyGroupList, client.InNamespace(rtx.Key.Namespace),
		client.MatchingFields{"spec.cluster.name": rtx.Key.Name}); err != nil {
		return task.Fail().With("can't list tiproxy group: %w", err)
	}
	for i := range tiproxyGroupList.Items {
		rtx.TiProxyGroups = append(rtx.TiProxyGroups, &tiproxyGroupList.Items[i])
	}

	return task.Complete().With("new context completed")
}
