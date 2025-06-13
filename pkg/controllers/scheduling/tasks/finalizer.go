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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskFinalizerDel(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		wait, err := EnsureSubResourcesDeleted(ctx, c, state.Object()) // state.Object() will be *v1alpha1.Scheduling
		if err != nil {
			return task.Fail().With("cannot delete subresources: %v", err)
		}

		if wait {
			return task.Retry(task.DefaultRequeueAfter).With("wait all subresources deleted")
		}

		if err := k8s.RemoveFinalizer(ctx, c, state.Object()); err != nil {
			return task.Fail().With("cannot remove finalizer: %v", err)
		}

		return task.Complete().With("finalizer is removed")
	})
}

func EnsureSubResourcesDeleted(ctx context.Context, c client.Client, scheduling *v1alpha1.Scheduling) (wait bool, _ error) {
	wait1, err := k8s.DeleteInstanceSubresource(ctx, c, runtime.FromScheduling(scheduling), &corev1.PodList{})
	if err != nil {
		return false, err
	}
	wait2, err := k8s.DeleteInstanceSubresource(ctx, c, runtime.FromScheduling(scheduling), &corev1.ConfigMapList{})
	if err != nil {
		return false, err
	}
	wait3, err := k8s.DeleteInstanceSubresource(ctx, c, runtime.FromScheduling(scheduling), &corev1.PersistentVolumeClaimList{})
	if err != nil {
		return false, err
	}

	return wait1 || wait2 || wait3, nil
}
