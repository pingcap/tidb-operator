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
	"cmp"
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/order"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

type TaskSelectTiKV struct {
	Client client.Client
	Order  order.Order[types.NamespacedName, string, *v1alpha1.TiKV]
}

func NewTaskSelectTiKV(
	c client.Client, o order.Order[types.NamespacedName, string, *v1alpha1.TiKV],
) task.Task[ReconcileContext] {
	return &TaskSelectTiKV{Client: c, Order: o}
}

func (*TaskSelectTiKV) Name() string { return "SelectTiKV" }

func (t *TaskSelectTiKV) Sync(ctx task.Context[ReconcileContext]) task.Result {
	cluster := ctx.Self().Cluster
	if !features.NewFromCluster(cluster.Spec.FeatureGates).Enabled(meta.TiKVSequentialScaleIn) {
		return task.Complete().With("sequential scale-in is disabled")
	}
	if !cluster.DeletionTimestamp.IsZero() {
		return task.Complete().With("cluster is deleting")
	}
	peers, err := apicall.ListClusterInstances[scope.TiKV](ctx, t.Client, cluster.Namespace, cluster.Name)
	if err != nil {
		return task.Fail().With("cannot list TiKV instances: %v", err)
	}
	candidates := make([]*v1alpha1.TiKV, 0, len(peers))
	for _, kv := range peers {
		if coreutil.IsOffline[scope.TiKV](kv) {
			candidates = append(candidates, kv)
		} else if kv.Annotations[v1alpha1.AnnoKeySelectedToDelete] == v1alpha1.AnnoValTrue {
			// The instance controller clears this only after PD confirms cancellation.
			// Do not infer completion from the CR's potentially stale store state.
			return task.Complete().With("waiting for TiKV %s to finish offline cancellation", kv.Name)
		}
	}
	key := client.ObjectKeyFromObject(cluster)
	kv, ok := t.Order.Sort(key, candidates)
	if !ok {
		return task.Complete().With("no offline TiKV instances")
	}
	if kv.Annotations[v1alpha1.AnnoKeySelectedToDelete] != v1alpha1.AnnoValTrue {
		if err := setSelectedToDelete(ctx, t.Client, kv); err != nil {
			return task.Fail().With("cannot persist selected-to-delete annotation: %v", err)
		}
	}
	return task.Complete().With("selected TiKV %s", kv.Name)
}

// CompareScaleIn restores persisted selections before considering store state and stable identity.
func CompareScaleIn(a, b *v1alpha1.TiKV) int {
	for _, preferred := range []func(*v1alpha1.TiKV) bool{
		func(kv *v1alpha1.TiKV) bool {
			return kv.Annotations[v1alpha1.AnnoKeySelectedToDelete] == v1alpha1.AnnoValTrue
		},
		func(kv *v1alpha1.TiKV) bool { return kv.Status.State == v1alpha1.StoreStateRemoving },
	} {
		if preferred(a) != preferred(b) {
			if preferred(a) {
				return -1
			}
			return 1
		}
	}
	if n := a.CreationTimestamp.Compare(b.CreationTimestamp.Time); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Name, b.Name); n != 0 {
		return n
	}
	return cmp.Compare(a.UID, b.UID)
}

func setSelectedToDelete(ctx context.Context, c client.Client, kv *v1alpha1.TiKV) error {
	updated := kv.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	updated.Annotations[v1alpha1.AnnoKeySelectedToDelete] = v1alpha1.AnnoValTrue
	// Check resourceVersion so a concurrent offline change invalidates this selection write.
	patch := ctrlclient.MergeFromWithOptions(kv, ctrlclient.MergeFromWithOptimisticLock{})
	if err := c.Patch(ctx, updated, patch); err != nil {
		return err
	}
	*kv = *updated
	return nil
}
