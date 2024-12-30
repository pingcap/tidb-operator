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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const defaultDelWaitTime = 10 * time.Second

func TaskFinalizerDel(state State, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		var errList []error
		var names []string
		for _, peer := range state.TiKVSlice() {
			names = append(names, peer.Name)
			if peer.GetDeletionTimestamp().IsZero() {
				if err := c.Delete(ctx, peer); err != nil {
					if errors.IsNotFound(err) {
						continue
					}
					errList = append(errList, fmt.Errorf("try to delete the tikv instance %v failed: %w", peer.Name, err))
					continue
				}
			}
		}

		if len(errList) != 0 {
			return task.Fail().With("failed to delete all tikv instances: %v", utilerr.NewAggregate(errList))
		}

		if len(names) != 0 {
			return task.Retry(defaultDelWaitTime).With("wait for all tikv instances being removed, %v still exists", names)
		}

		wait, err := k8s.DeleteGroupSubresource(ctx, c, runtime.FromTiKVGroup(state.TiKVGroup()), &corev1.ServiceList{})
		if err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if wait {
			return task.Wait().With("wait all subresources deleted")
		}

		if err := k8s.RemoveFinalizer(ctx, c, state.TiKVGroup()); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
		}

		return task.Complete().With("finalizer has been removed")
	})
}
