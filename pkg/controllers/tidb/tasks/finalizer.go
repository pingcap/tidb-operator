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
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

func TaskFinalizerDel(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("FinalizerDel", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		if err := k8s.EnsureInstanceSubResourceDeleted(ctx, c,
			rtx.TiDB.Namespace, rtx.TiDB.Name); err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if err := k8s.RemoveFinalizer(ctx, c, rtx.TiDB); err != nil {
			return task.Fail().With("cannot remove finalizer: %w", err)
		}

		return task.Complete().With("finalizer is removed")
	})
}

func TaskFinalizerAdd(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("FinalizerAdd", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		if err := k8s.EnsureFinalizer(ctx, c, rtx.TiDB); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %w", err)
		}

		return task.Complete().With("finalizer is added")
	})
}
