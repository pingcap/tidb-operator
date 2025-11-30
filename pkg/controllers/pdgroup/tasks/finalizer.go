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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskFinalizerDel(state State, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		var errList []error
		for _, peer := range state.PDSlice() {
			if peer.GetDeletionTimestamp().IsZero() {
				if err := c.Delete(ctx, peer); err != nil {
					if errors.IsNotFound(err) {
						continue
					}

					errList = append(errList, err)
					continue
				}
			}

			// PD controller cannot clean up finalizer after quorum is lost
			// Forcely clean up all finalizers of pd instances
			// NOTE:
			//   Now we forcely clean up pd finalizers in pd group controller when the pd group is deleted,
			//   but forcely clean up tikv finalizers in tikv controller when the cluster is deleted.
			//   We can all forcely clean up finalizers when the cluster is deleted and change this task to
			//   the common one.
			// TODO(liubo02): refactor to use common task
			if err := k8s.RemoveFinalizer(ctx, c, peer); err != nil {
				errList = append(errList, err)
			}
		}

		if len(errList) != 0 {
			return task.Fail().With("failed to delete all pd instances: %v", utilerr.NewAggregate(errList))
		}

		if len(state.PDSlice()) != 0 {
			return task.Wait().With("wait for all pd instances being removed, %v still exists", len(state.PDSlice()))
		}

		wait, err := k8s.DeleteGroupSubresource(ctx, c, runtime.FromPDGroup(state.PDGroup()), &corev1.ServiceList{})
		if err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if wait {
			return task.Retry(task.DefaultRequeueAfter).With("wait all subresources deleted")
		}

		if err := k8s.RemoveFinalizer(ctx, c, state.PDGroup()); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
		}

		return task.Complete().With("finalizer has been removed")
	})
}
