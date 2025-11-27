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

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

type TaskFinalizer struct {
	Logger logr.Logger
	Client client.Client

	PDClientManager pdm.PDClientManager
}

func NewTaskFinalizer(logger logr.Logger, c client.Client, m pdm.PDClientManager) task.Task[ReconcileContext] {
	return &TaskFinalizer{
		Logger:          logger,
		Client:          c,
		PDClientManager: m,
	}
}

func (*TaskFinalizer) Name() string {
	return "Finalizer"
}

//nolint:gocyclo // refactor if possible
func (t *TaskFinalizer) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	if rtx.Cluster.GetDeletionTimestamp().IsZero() {
		if err := k8s.EnsureFinalizer(ctx, t.Client, rtx.Cluster); err != nil {
			return task.Fail().With("can't ensure finalizer: %w", err)
		}
		return task.Complete().With("ensured finalizer")
	}

	t.PDClientManager.Deregister(timanager.PrimaryKey(rtx.Cluster.Namespace, rtx.Cluster.Name))

	if len(rtx.PDGroups) == 0 &&
		len(rtx.TiKVGroups) == 0 &&
		len(rtx.TiDBGroups) == 0 &&
		len(rtx.TiFlashGroups) == 0 &&
		len(rtx.TiCDCGroups) == 0 &&
		len(rtx.TSOGroups) == 0 &&
		len(rtx.SchedulingGroups) == 0 &&
		len(rtx.SchedulerGroups) == 0 &&
		len(rtx.TiProxyGroups) == 0 {
		if err := k8s.RemoveFinalizer(ctx, t.Client, rtx.Cluster); err != nil {
			return task.Fail().With("can't remove finalizer: %w", err)
		}
		return task.Complete().Break().With("removed finalizer")
	}

	// trigger the deletion of the components
	for _, pdg := range rtx.PDGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, pdg); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete pd group: %v", err)
		}
	}
	for _, tg := range rtx.TSOGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, tg); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete tso group: %v", err)
		}
	}
	for _, sg := range rtx.SchedulingGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, sg); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete scheduling group: %v", err)
		}
	}
	for _, sg := range rtx.SchedulerGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, sg); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete scheduler group: %v", err)
		}
	}
	for _, tikvGroup := range rtx.TiKVGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, tikvGroup); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete tikv group: %v", err)
		}
	}
	for _, tiflashGroup := range rtx.TiFlashGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, tiflashGroup); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete tiflash group: %v", err)
		}
	}
	for _, tidbGroup := range rtx.TiDBGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, tidbGroup); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete tidb group: %v", err)
		}
	}

	for _, ticdcGroup := range rtx.TiCDCGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, ticdcGroup); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete ticdc group: %v", err)
		}
	}

	for _, pg := range rtx.TiProxyGroups {
		//nolint:gocritic // not a real issue, see https://github.com/go-critic/go-critic/issues/1448
		if err := t.Client.Delete(ctx, pg); client.IgnoreNotFound(err) != nil {
			return task.Fail().With("can't delete tiproxy group: %v", err)
		}
	}

	// wait for the components to be deleted
	return task.Fail().With("deleting components")
}
