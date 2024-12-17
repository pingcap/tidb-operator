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
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskFinalizerDel(ctx *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func() task.Result {
		switch {
		// get member info successfully and the member still exists
		case ctx.IsAvailable && ctx.MemberID != "":
			// TODO: check whether quorum will be lost?
			if err := ctx.PDClient.Underlay().DeleteMember(ctx, ctx.PD.Name); err != nil {
				return task.Fail().With("cannot delete member: %v", err)
			}

			if err := k8s.EnsureInstanceSubResourceDeleted(ctx, c,
				ctx.PD.Namespace, ctx.PD.Name); err != nil {
				return task.Fail().With("cannot delete subresources: %v", err)
			}

			if err := k8s.RemoveFinalizer(ctx, c, ctx.PD); err != nil {
				return task.Fail().With("cannot remove finalizer: %v", err)
			}
		case ctx.IsAvailable:
			if err := k8s.EnsureInstanceSubResourceDeleted(ctx, c,
				ctx.PD.Namespace, ctx.PD.Name); err != nil {
				return task.Fail().With("cannot delete subresources: %v", err)
			}

			if err := k8s.RemoveFinalizer(ctx, c, ctx.PD); err != nil {
				return task.Fail().With("cannot remove finalizer: %v", err)
			}
		case !ctx.IsAvailable:
			// it may block some unsafe operations
			return task.Fail().With("pd cluster is not available")
		}

		return task.Complete().With("finalizer is removed")
	})
}

func TaskFinalizerAdd(ctx *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func() task.Result {
		if err := k8s.EnsureFinalizer(ctx, c, ctx.PD); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}
