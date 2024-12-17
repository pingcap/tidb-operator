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

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type TaskBoot struct {
	Logger logr.Logger
	Client client.Client
}

func NewTaskBoot(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskBoot{
		Logger: logger,
		Client: c,
	}
}

func (*TaskBoot) Name() string {
	return "Boot"
}

func (t *TaskBoot) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	if rtx.IsAvailable && !rtx.PDGroup.Spec.Bootstrapped {
		rtx.PDGroup.Spec.Bootstrapped = true
		if err := t.Client.Update(ctx, rtx.PDGroup); err != nil {
			return task.Fail().With("pd cluster is available but not marked as bootstrapped: %w", err)
		}
	}

	if !rtx.PDGroup.Spec.Bootstrapped {
		// TODO: use task.Retry?
		return task.Fail().Continue().With("pd cluster is not bootstrapped")
	}

	return task.Complete().With("pd cluster is bootstrapped")
}
