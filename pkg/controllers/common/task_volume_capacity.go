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

package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type InstanceCondVolumeCapacityExceedsRequestUpdater[T client.Object] interface {
	StatusUpdater
	Object() T
}

// TaskInstanceConditionVolumeCapacityExceedsRequest observes PVC capacity and updates
// the in-memory condition. The controller's status task persists the change.
func TaskInstanceConditionVolumeCapacityExceedsRequest[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state InstanceCondVolumeCapacityExceedsRequestUpdater[F], c client.Client) task.Task {
	return task.NameTaskFunc("CondVolumeCapacityExceedsRequest", func(ctx context.Context) task.Result {
		obj := state.Object()
		vols := coreutil.Volumes[S](obj)
		pvcs := make(map[string]*corev1.PersistentVolumeClaim, len(vols))
		for _, vol := range vols {
			var pvc corev1.PersistentVolumeClaim
			key := client.ObjectKey{Namespace: obj.GetNamespace(), Name: coreutil.PersistentVolumeClaimName[S](obj, vol.Name)}
			if err := c.Get(ctx, key, &pvc); err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return task.Fail().With("cannot observe PVC capacity: %w", err)
			}
			pvcs[vol.Name] = &pvc
		}
		if coreutil.SetStatusCondition[S](obj, coreutil.VolumeCapacityCondition(vols, pvcs)) {
			state.SetStatusChanged()
		}
		return task.Complete().With("volume capacity is observed")
	})
}
