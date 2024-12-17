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
	"github.com/pingcap/tidb-operator/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	TiDBClient tidbapi.TiDBClient

	IsAvailable bool
	Suspended   bool

	TiDBGroup      *v1alpha1.TiDBGroup
	TiDBs          []*v1alpha1.TiDB
	Cluster        *v1alpha1.Cluster
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

	var tidbg v1alpha1.TiDBGroup
	if err := t.Client.Get(ctx, rtx.Key, &tidbg); err != nil {
		if !errors.IsNotFound(err) {
			return task.Fail().With("can't get tidb group: %w", err)
		}

		return task.Complete().Break().With("tidb group has been deleted")
	}
	rtx.TiDBGroup = &tidbg

	var cluster v1alpha1.Cluster
	if err := t.Client.Get(ctx, client.ObjectKey{
		Name:      tidbg.Spec.Cluster.Name,
		Namespace: tidbg.Namespace,
	}, &cluster); err != nil {
		return task.Fail().With("cannot find cluster %s: %w", tidbg.Spec.Cluster.Name, err)
	}
	rtx.Cluster = &cluster

	if cluster.ShouldPauseReconcile() {
		return task.Complete().Break().With("cluster reconciliation is paused")
	}

	var tidbList v1alpha1.TiDBList
	if err := t.Client.List(ctx, &tidbList, client.InNamespace(tidbg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
		v1alpha1.LabelKeyGroup:     tidbg.Name,
	}); err != nil {
		return task.Fail().With("cannot list tidb instances: %w", err)
	}

	rtx.TiDBs = make([]*v1alpha1.TiDB, len(tidbList.Items))
	rtx.Suspended = len(tidbList.Items) > 0
	for i := range tidbList.Items {
		rtx.TiDBs[i] = &tidbList.Items[i]
		if meta.IsStatusConditionTrue(tidbList.Items[i].Status.Conditions, v1alpha1.TiDBCondHealth) {
			// TiDB Group is available if any of its members is available
			rtx.IsAvailable = true
		}
		if !meta.IsStatusConditionTrue(tidbList.Items[i].Status.Conditions, v1alpha1.TiDBCondSuspended) {
			// TiDB Group is not suspended if any of its members is not suspended
			rtx.Suspended = false
		}
	}

	slices.SortFunc(rtx.TiDBs, func(a, b *v1alpha1.TiDB) int {
		return cmp.Compare(a.Name, b.Name)
	})

	rtx.UpgradeChecker = action.NewUpgradeChecker(t.Client, rtx.Cluster, t.Logger)

	return task.Complete().With("new context completed")
}
