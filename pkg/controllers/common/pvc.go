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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes"
)

type TaskPVCState[F client.Object] interface {
	stateutil.ICluster
	stateutil.IObject[F]
	stateutil.IFeatureGates
}

func TaskPVC[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state TaskPVCState[F], c client.Client, vm volumes.ModifierFactory, n PVCNewer[F]) task.Task {
	return task.NameTaskFunc("PVC", func(ctx context.Context) task.Result {
		cluster := state.Cluster()
		obj := state.Object()

		pvcs := n.NewPVCs(cluster, obj, state.FeatureGates())

		if state.FeatureGates().Enabled(meta.VolumeAttributesClass) {
			if err := volumes.SyncPVCs(ctx, c, pvcs); err != nil {
				if task.IsWaitError(err) {
					return task.Wait().With("%v", err)
				}
				return task.Fail().With("%v", err)
			}

			return task.Complete().With("pvcs are synced")
		}

		logger := logr.FromContextOrDiscard(ctx)
		if wait, err := volumes.LegacySyncPVCs(ctx, c, pvcs, vm.New(), logger); err != nil {
			return task.Fail().With("legacy: failed to sync pvcs: %v", err)
		} else if wait {
			return task.Retry(task.DefaultRequeueAfter).With("legacy: waiting for pvcs to be synced")
		}

		return task.Complete().With("pvcs are synced")
	})
}

type PVCNewer[F client.Object] interface {
	NewPVCs(c *v1alpha1.Cluster, obj F, fg features.Gates) []*corev1.PersistentVolumeClaim
}

type PVCNewerFunc[F client.Object] func(c *v1alpha1.Cluster, obj F, fg features.Gates) []*corev1.PersistentVolumeClaim

func (f PVCNewerFunc[F]) NewPVCs(c *v1alpha1.Cluster, obj F, fg features.Gates) []*corev1.PersistentVolumeClaim {
	return f(c, obj, fg)
}

func DefaultPVCNewer[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
]() PVCNewer[F] {
	return PVCNewerFunc[F](
		func(cluster *v1alpha1.Cluster, obj F, fg features.Gates) []*corev1.PersistentVolumeClaim {
			pvcs := coreutil.PVCs[S](
				cluster,
				obj,
				coreutil.EnableVAC(fg.Enabled(meta.VolumeAttributesClass)),
			)

			return pvcs
		},
	)
}
