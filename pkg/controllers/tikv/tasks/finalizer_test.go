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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestWaitCacheTTLAfterStoreRemoved(t *testing.T) {
	for _, tt := range []struct {
		name        string
		ttl         *int64
		offlinedAgo time.Duration
		wait        bool
	}{
		{name: "persist missing condition", ttl: ptr.To[int64](60), wait: true},
		{name: "reuse persisted condition", ttl: ptr.To[int64](60), offlinedAgo: 20 * time.Second, wait: true},
		{name: "expired", ttl: ptr.To[int64](60), offlinedAgo: 2 * time.Minute},
		{name: "unset"},
		{name: "zero", ttl: ptr.To[int64](0)},
		{name: "negative", ttl: ptr.To[int64](-1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "ns"}}
			kv.Spec.Offline = ptr.To(true)
			kv.Spec.CacheTTLSeconds = tt.ttl
			// An old leader eviction timestamp must not bypass the scale-in wait.
			kv.Status.Conditions = []metav1.Condition{{
				Type: v1alpha1.TiKVCondLeadersEvicted, Status: metav1.ConditionTrue,
				Reason: "Evicted", LastTransitionTime: metav1.NewTime(time.Now().Add(-time.Hour)),
			}}
			if tt.offlinedAgo > 0 {
				kv.Status.Conditions = append(kv.Status.Conditions, metav1.Condition{
					Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue,
					Reason:             v1alpha1.ReasonOfflineCompleted,
					LastTransitionTime: metav1.NewTime(time.Now().Add(-tt.offlinedAgo).Truncate(time.Second)),
				})
			}
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "ns"}}
			c := client.NewFakeClient(kv, pod)
			var firstTransition metav1.Time
			for range 2 {
				// Reload persisted state, as a new reconcile or operator restart would.
				require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(kv), kv))
				st := &ReconcileContext{State: &state{tikv: kv, storeState: pdv1.NodeStateRemoved}}
				deleted := false
				result, _ := task.RunTask(t.Context(), task.Block(
					common.TaskInstanceConditionOffline[scope.TiKV](st),
					common.TaskStatusPersister[scope.TiKV](st, c),
					TaskWaitCacheTTLAfterStoreRemoved(st),
					task.NameTaskFunc("DeletePod", func(ctx context.Context) task.Result {
						deleted = true
						return task.Complete().With("finalizer can delete the pod")
					}),
				))
				require.Equal(t, !tt.wait, deleted)
				require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(kv), kv))
				cond := meta.FindStatusCondition(kv.Status.Conditions, v1alpha1.StoreOfflinedConditionType)
				require.NotNil(t, cond)
				require.Equal(t, metav1.ConditionTrue, cond.Status)
				if !firstTransition.IsZero() {
					require.Equal(t, firstTransition, cond.LastTransitionTime)
				}
				firstTransition = cond.LastTransitionTime
				if tt.wait {
					require.Equal(t, task.SRetry, result.Status())
					require.Positive(t, result.RequeueAfter())
					expected := time.Until(cond.LastTransitionTime.Add(time.Duration(*tt.ttl) * time.Second))
					require.InDelta(t, float64(expected), float64(result.RequeueAfter()), float64(time.Second))
					require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(pod), &corev1.Pod{}))
				}
			}
		})
	}
}
