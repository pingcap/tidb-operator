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
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	Cluster               *v1alpha1.Cluster
	PDGroups              []*v1alpha1.PDGroup
	ResourceManagerGroups []*v1alpha1.ResourceManagerGroup
	RouterGroups          []*v1alpha1.RouterGroup
	TiKVGroups            []*v1alpha1.TiKVGroup
	TiFlashGroups         []*v1alpha1.TiFlashGroup
	TiDBGroups            []*v1alpha1.TiDBGroup
	TiCDCGroups           []*v1alpha1.TiCDCGroup
	TSOGroups             []*v1alpha1.TSOGroup
	SchedulingGroups      []*v1alpha1.SchedulingGroup
	SchedulerGroups       []*v1alpha1.SchedulerGroup
	TiProxyGroups         []*v1alpha1.TiProxyGroup
	TiKVWorkerGroups      []*v1alpha1.TiKVWorkerGroup
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

	ns := rtx.Key.Namespace
	name := rtx.Key.Name
	pdgs, err := apicall.ListGroups[scope.PDGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list pd groups: %w", err)
	}
	rtx.PDGroups = pdgs

	rmgs, err := apicall.ListGroups[scope.ResourceManagerGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list resource manager groups: %w", err)
	}
	rtx.ResourceManagerGroups = rmgs

	rgs, err := apicall.ListGroups[scope.RouterGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list router manager groups: %w", err)
	}
	rtx.RouterGroups = rgs

	tgs, err := apicall.ListGroups[scope.TSOGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list tso groups: %w", err)
	}
	rtx.TSOGroups = tgs

	sgs, err := apicall.ListGroups[scope.SchedulingGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list scheduling groups: %w", err)
	}
	rtx.SchedulingGroups = sgs

	lsgs, err := apicall.ListGroups[scope.SchedulerGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list scheduler groups: %w", err)
	}
	rtx.SchedulerGroups = lsgs

	kvgs, err := apicall.ListGroups[scope.TiKVGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list tikv groups: %w", err)
	}
	rtx.TiKVGroups = kvgs

	fgs, err := apicall.ListGroups[scope.TiFlashGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list tiflash groups: %w", err)
	}
	rtx.TiFlashGroups = fgs

	dbgs, err := apicall.ListGroups[scope.TiDBGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list tidb groups: %w", err)
	}
	rtx.TiDBGroups = dbgs

	cgs, err := apicall.ListGroups[scope.TiCDCGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list ticdc groups: %w", err)
	}
	rtx.TiCDCGroups = cgs

	pgs, err := apicall.ListGroups[scope.TiProxyGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list tiproxy groups: %w", err)
	}
	rtx.TiProxyGroups = pgs

	wgs, err := apicall.ListGroups[scope.TiKVWorkerGroup](ctx, t.Client, ns, name)
	if err != nil {
		return task.Fail().With("can't list tikv worker groups: %w", err)
	}
	rtx.TiKVWorkerGroups = wgs

	return task.Complete().With("new context completed")
}
