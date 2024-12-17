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
	"fmt"

	"github.com/go-logr/logr"
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type TaskFinalizer struct {
	Client client.Client
	Logger logr.Logger
}

func NewTaskFinalizer(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskFinalizer{
		Client: c,
		Logger: logger,
	}
}

func (*TaskFinalizer) Name() string {
	return "Finalizer"
}

func (t *TaskFinalizer) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	if !rtx.TiDBGroup.GetDeletionTimestamp().IsZero() {
		errList := []error{}
		names := []string{}
		for _, tidb := range rtx.TiDBs {
			names = append(names, tidb.Name)
			if tidb.GetDeletionTimestamp().IsZero() {
				if err := t.Client.Delete(ctx, tidb); err != nil {
					errList = append(errList, fmt.Errorf("try to delete the tidb instance %v failed: %w", tidb.Name, err))
				}
			}
		}

		if len(errList) != 0 {
			return task.Fail().With("failed to delete all tidb instances: %v", utilerr.NewAggregate(errList))
		}

		if len(rtx.TiDBs) != 0 {
			return task.Fail().With("wait for all tidb instances being removed, %v still exists", names)
		}

		if err := k8s.EnsureGroupSubResourceDeleted(ctx, t.Client,
			rtx.TiDBGroup.Namespace, rtx.TiDBGroup.Name); err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if err := k8s.RemoveFinalizer(ctx, t.Client, rtx.TiDBGroup); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
		}
	} else {
		if err := k8s.EnsureFinalizer(ctx, t.Client, rtx.TiDBGroup); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %w", err)
		}
	}

	return task.Complete().With("finalizer is synced")
}
