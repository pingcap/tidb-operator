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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/order"
)

func TestSelectTiKV(t *testing.T) {
	for _, scenario := range []string{"sorted", "removing", "restart", "disabled", "patch failure", "canceled", "canceled stale state"} {
		t.Run(scenario, func(t *testing.T) {
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "tc", Namespace: "ns"}}
			cluster.Spec.FeatureGates = []meta.FeatureGate{{Name: meta.TiKVSequentialScaleIn}}
			if scenario == "disabled" {
				cluster.Spec.FeatureGates = nil
			}
			a := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "a", UID: "a", Namespace: "ns", ResourceVersion: "1",
				Labels: map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   "tc",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				},
			}}
			a.Spec.Offline = ptr.To(true)
			a.Spec.Cluster.Name = "tc"
			b := a.DeepCopy()
			b.Name, b.UID = "b", "b"
			if scenario == "removing" {
				b.Status.State = v1alpha1.StoreStateRemoving
			}
			if scenario == "restart" || scenario == "canceled" || scenario == "canceled stale state" {
				b.Annotations = map[string]string{v1alpha1.AnnoKeySelectedToDelete: v1alpha1.AnnoValTrue}
			}
			if scenario == "canceled" || scenario == "canceled stale state" {
				b.Spec.Offline = ptr.To(false)
				b.Status.State = map[string]string{
					"canceled":             v1alpha1.StoreStateRemoving,
					"canceled stale state": v1alpha1.StoreStateServing,
				}[scenario]
			}
			fc := client.NewFakeClient(a, b)
			fails := scenario == "patch failure"
			if fails {
				fc.WithError("patch", "*", errors.New("patch failed"))
			}
			o := order.New[types.NamespacedName](CompareScaleIn, func(kv *v1alpha1.TiKV) string { return kv.Name })
			ctx := &ReconcileContext{Context: context.Background(), Cluster: cluster}
			result := NewTaskSelectTiKV(fc, o).Sync(ctx)
			require.Equal(t, fails, result.IsFailed())
			for _, kv := range []*v1alpha1.TiKV{a, b} {
				stored := &v1alpha1.TiKV{}
				require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(kv), stored))
				selected := scenario == "sorted" && kv.Name == "a" ||
					(scenario == "removing" || scenario == "restart" || scenario == "canceled" || scenario == "canceled stale state") && kv.Name == "b"
				require.Equal(t, selected, stored.Annotations[v1alpha1.AnnoKeySelectedToDelete] == v1alpha1.AnnoValTrue)
			}
			if scenario == "canceled" || scenario == "canceled stale state" {
				// Simulate the instance controller's acknowledgment after PD confirms cancellation.
				require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(b), b))
				delete(b.Annotations, v1alpha1.AnnoKeySelectedToDelete)
				require.NoError(t, fc.Update(ctx, b))
				require.False(t, NewTaskSelectTiKV(fc, o).Sync(ctx).IsFailed())
				require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(a), a))
				require.Equal(t, v1alpha1.AnnoValTrue, a.Annotations[v1alpha1.AnnoKeySelectedToDelete])
			}
			if scenario == "sorted" {
				require.NoError(t, fc.Delete(ctx, a))
				require.False(t, NewTaskSelectTiKV(fc, o).Sync(ctx).IsFailed())
				stored := &v1alpha1.TiKV{}
				require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(b), stored))
				require.Equal(t, v1alpha1.AnnoValTrue, stored.Annotations[v1alpha1.AnnoKeySelectedToDelete])
			}
		})
	}
}

// Observing the old annotation value must not replace an in-memory selection.
func TestSelectTiKVCacheLag(t *testing.T) {
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "tc", Namespace: "ns"}}
	cluster.Spec.FeatureGates = []meta.FeatureGate{{Name: meta.TiKVSequentialScaleIn}}
	b := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "ns", UID: "b", ResourceVersion: "1", Labels: map[string]string{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   "tc",
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
	}}}
	b.Spec.Offline = ptr.To(true)
	a := b.DeepCopy()
	a.Name, a.UID = "a", "a"
	o := order.New[types.NamespacedName](CompareScaleIn, func(kv *v1alpha1.TiKV) string { return kv.Name })
	key := client.ObjectKeyFromObject(cluster)
	selected, ok := o.Sort(key, []*v1alpha1.TiKV{b})
	require.True(t, ok)
	require.Equal(t, b.Name, selected.Name)
	fc := client.NewFakeClient(a, b)
	ctx := &ReconcileContext{Context: context.Background(), Cluster: cluster}
	require.False(t, NewTaskSelectTiKV(fc, o).Sync(ctx).IsFailed())
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(a), a))
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(b), b))
	require.Empty(t, a.Annotations[v1alpha1.AnnoKeySelectedToDelete])
	require.Equal(t, v1alpha1.AnnoValTrue, b.Annotations[v1alpha1.AnnoKeySelectedToDelete])
}
