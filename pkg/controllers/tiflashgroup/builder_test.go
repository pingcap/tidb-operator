// Copyright 2026 PingCAP, Inc.
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

package tiflashgroup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestSurgePauseAndResume(t *testing.T) {
	for _, pauseCluster := range []bool{false, true} {
		name := "group"
		if pauseCluster {
			name = "cluster"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"}}
			fg := &v1alpha1.TiFlashGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "flash", Namespace: "default"},
				Spec: v1alpha1.TiFlashGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: cluster.Name}, Replicas: ptr.To[int32](1),
					Template: v1alpha1.TiFlashTemplate{Spec: v1alpha1.TiFlashTemplateSpec{Version: "v8.1.0"}},
				},
			}
			c := client.NewFakeClient(cluster, fg)
			r := &Reconciler{Client: c, AllocateFactory: tracker.New().AllocateFactory("tiflash")}
			reconcile := func() []v1alpha1.TiFlash {
				_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(fg)})
				require.NoError(t, err)
				var list v1alpha1.TiFlashList
				require.NoError(t, c.List(ctx, &list))
				return list.Items
			}
			instances := reconcile()
			require.Len(t, instances, 1)
			old := instances[0].DeepCopy()
			old.Status.CurrentRevision = runtime.FromTiFlash(old).GetUpdateRevision()
			old.Status.ObservedGeneration = old.Generation
			old.Status.Conditions = []metav1.Condition{{
				Type: v1alpha1.CondReady, Status: metav1.ConditionTrue,
				ObservedGeneration: old.Generation, LastTransitionTime: metav1.Unix(0, 0),
			}}
			require.NoError(t, c.Status().Update(ctx, old))

			// Changing the policy alone must not change the instance revision.
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(fg), fg))
			fg.Spec.MaxSurge = ptr.To[int32](1)
			require.NoError(t, c.Update(ctx, fg))
			instances = reconcile()
			require.Len(t, instances, 1)
			assert.Equal(t, old.Spec, instances[0].Spec)
			assert.Equal(t, old.Labels, instances[0].Labels)

			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(fg), fg))
			fg.Spec.Template.Spec.Image = ptr.To("pingcap/tiflash:v8.1.1")
			require.NoError(t, c.Update(ctx, fg))
			require.Len(t, reconcile(), 2)

			if pauseCluster {
				cluster.Spec.Paused = true
				require.NoError(t, c.Update(ctx, cluster))
			} else {
				require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(fg), fg))
				fg.Spec.Progressing = ptr.To(false)
				require.NoError(t, c.Update(ctx, fg))
			}
			// A ready spare must not advance a paused rollout.
			for _, instance := range reconcile() {
				instance.Status = old.Status
				instance.Status.CurrentRevision = runtime.FromTiFlash(&instance).GetUpdateRevision()
				require.NoError(t, c.Status().Update(ctx, &instance))
			}
			for _, instance := range reconcile() {
				assert.False(t, ptr.Deref(instance.Spec.Offline, false))
			}
			if pauseCluster {
				cluster.Spec.Paused = false
				require.NoError(t, c.Update(ctx, cluster))
			} else {
				fg.Spec.Progressing = ptr.To(true)
				require.NoError(t, c.Update(ctx, fg))
			}
			reconcile()
			var retiring v1alpha1.TiFlash
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(old), &retiring))
			assert.True(t, ptr.Deref(retiring.Spec.Offline, false))
		})
	}
}
