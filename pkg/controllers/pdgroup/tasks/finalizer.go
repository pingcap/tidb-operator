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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/client"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type TaskFinalizer struct {
	Client          client.Client
	Logger          logr.Logger
	PDClientManager pdm.PDClientManager
}

func NewTaskFinalizer(logger logr.Logger, c client.Client, pdcm pdm.PDClientManager) task.Task[ReconcileContext] {
	return &TaskFinalizer{
		Client:          c,
		Logger:          logger,
		PDClientManager: pdcm,
	}
}

func (*TaskFinalizer) Name() string {
	return "Finalizer"
}

func (t *TaskFinalizer) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	if rtx.PDGroup.GetDeletionTimestamp().IsZero() {
		if err := k8s.EnsureFinalizer(ctx, t.Client, rtx.PDGroup); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %w", err)
		}
		return task.Complete().With("finalizer is synced")
	}

	errList := []error{}
	for _, peer := range rtx.Peers {
		if err := t.Client.Delete(ctx, peer); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			errList = append(errList, err)
			continue
		}

		// PD controller cannot clean up finalizer after quorum is lost
		// Forcely clean up all finalizers of pd instances
		if err := k8s.RemoveFinalizer(ctx, t.Client, peer); err != nil {
			errList = append(errList, err)
		}
	}

	if len(errList) != 0 {
		return task.Fail().With("failed to delete all pd instances: %v", utilerr.NewAggregate(errList))
	}

	if len(rtx.Peers) != 0 {
		return task.Fail().With("wait for all pd instances being removed, %v still exists", rtx.Peers)
	}

	if err := k8s.EnsureGroupSubResourceDeleted(ctx, t.Client,
		rtx.PDGroup.Namespace, rtx.PDGroup.Name); err != nil {
		return task.Fail().With("cannot delete subresources: %w", err)
	}

	if err := k8s.RemoveFinalizer(ctx, t.Client, rtx.PDGroup); err != nil {
		return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
	}

	t.PDClientManager.Deregister(pdm.PrimaryKey(rtx.PDGroup.Namespace, rtx.PDGroup.Spec.Cluster.Name))

	return task.Complete().With("finalizer has been removed")
}
