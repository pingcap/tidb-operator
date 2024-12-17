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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/action"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	PDClient pdapi.PDClient
	Members  []Member

	// check whether pd is available
	IsAvailable bool

	Suspended bool

	PDGroup        *v1alpha1.PDGroup
	Peers          []*v1alpha1.PD
	Cluster        *v1alpha1.Cluster
	UpgradeChecker action.UpgradeChecker

	// Status fields
	v1alpha1.CommonStatus
}

// TODO: move to pdapi
type Member struct {
	ID   string
	Name string
}

func (ctx *ReconcileContext) Self() *ReconcileContext {
	return ctx
}

type TaskContext struct {
	Logger          logr.Logger
	Client          client.Client
	PDClientManager pdm.PDClientManager
}

func NewTaskContext(logger logr.Logger, c client.Client, pdcm pdm.PDClientManager) task.Task[ReconcileContext] {
	return &TaskContext{
		Logger:          logger,
		Client:          c,
		PDClientManager: pdcm,
	}
}

func (*TaskContext) Name() string {
	return "Context"
}

//nolint:gocyclo // refactor if possible
func (t *TaskContext) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	var pdg v1alpha1.PDGroup
	if err := t.Client.Get(ctx, rtx.Key, &pdg); err != nil {
		if !errors.IsNotFound(err) {
			return task.Fail().With("can't get pd group: %w", err)
		}

		return task.Complete().Break().With("pd group has been deleted")
	}
	rtx.PDGroup = &pdg

	var cluster v1alpha1.Cluster
	if err := t.Client.Get(ctx, client.ObjectKey{
		Name:      pdg.Spec.Cluster.Name,
		Namespace: pdg.Namespace,
	}, &cluster); err != nil {
		return task.Fail().With("cannot find cluster %s: %w", pdg.Spec.Cluster.Name, err)
	}
	rtx.Cluster = &cluster

	if cluster.ShouldPauseReconcile() {
		return task.Complete().Break().With("cluster reconciliation is paused")
	}

	var pdList v1alpha1.PDList
	if err := t.Client.List(ctx, &pdList, client.InNamespace(pdg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
		v1alpha1.LabelKeyGroup:     pdg.Name,
	}); err != nil {
		return task.Fail().With("cannot list pd peers: %w", err)
	}

	rtx.Peers = make([]*v1alpha1.PD, len(pdList.Items))
	rtx.Suspended = len(pdList.Items) > 0
	for i := range pdList.Items {
		rtx.Peers[i] = &pdList.Items[i]
		if !meta.IsStatusConditionTrue(rtx.Peers[i].Status.Conditions, v1alpha1.PDGroupCondSuspended) {
			// PD Group is not suspended if any of its members is not suspended
			rtx.Suspended = false
		}
	}
	slices.SortFunc(rtx.Peers, func(a, b *v1alpha1.PD) int {
		return cmp.Compare(a.Name, b.Name)
	})

	if rtx.PDGroup.GetDeletionTimestamp().IsZero() && len(rtx.Peers) > 0 {
		// TODO: register pd client after it is ready
		if err := t.PDClientManager.Register(rtx.PDGroup); err != nil {
			return task.Fail().With("cannot register pd client: %v", err)
		}
	}

	if rtx.Suspended {
		return task.Complete().With("context without member info is completed, pd is suspended")
	}

	c, ok := t.PDClientManager.Get(pdm.PrimaryKey(pdg.Namespace, pdg.Spec.Cluster.Name))
	if !ok {
		return task.Complete().With("context without pd client is completed, pd cannot be visited")
	}
	rtx.PDClient = c.Underlay()

	if !c.HasSynced() {
		return task.Complete().With("context without pd client is completed, cache of pd info is not synced")
	}

	rtx.IsAvailable = true

	ms, err := c.Members().List(labels.Everything())
	if err != nil {
		return task.Fail().With("cannot list members: %w", err)
	}

	for _, m := range ms {
		rtx.Members = append(rtx.Members, Member{
			Name: m.Name,
			ID:   m.ID,
		})
	}
	slices.SortFunc(rtx.Members, func(a, b Member) int {
		return cmp.Compare(a.Name, b.Name)
	})

	rtx.UpgradeChecker = action.NewUpgradeChecker(t.Client, rtx.Cluster, t.Logger)
	return task.Complete().With("context is fully completed")
}
