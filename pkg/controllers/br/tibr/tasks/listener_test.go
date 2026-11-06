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
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestMigrateDefaultAPIListener(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(fmt.Sprint(owned), func(t *testing.T) {
			cli := client.NewFakeClient()
			rtx := NewReconcileContext(context.Background(), types.NamespacedName{Namespace: "test", Name: "br"}, cli)
			rtx.tibr = &br.TiBR{ObjectMeta: metav1.ObjectMeta{Name: "br", Namespace: "test", UID: "br-uid"}}
			rtx.cluster = &core.Cluster{}
			sts := assembleSts(rtx)
			sts.Generation = 2
			sts.Status.ObservedGeneration = 1
			sts.Status.ReadyReplicas = 1
			sts.Spec.Template.Spec.Containers[0].Command[4] = fmt.Sprintf("0.0.0.0:%d", APIServerPort)
			sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, corev1.Container{Name: "sidecar", Image: "custom"})
			if !owned {
				sts.OwnerReferences = nil
			}
			require.NoError(t, cli.Create(context.Background(), sts))
			rtx.sts = sts
			result, _ := task.RunTask(context.Background(), taskRestrictAPIListener(rtx))
			if !owned {
				require.Equal(t, task.SFail, result.Status())
				return
			}
			require.Equal(t, task.SComplete, result.Status())
			require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(sts), sts))
			require.Contains(t, sts.Spec.Template.Spec.Containers[0].Command, fmt.Sprintf("127.0.0.1:%d", APIServerPort))
			require.NotContains(t, sts.Spec.Template.Spec.Containers[0].Command, fmt.Sprintf("0.0.0.0:%d", APIServerPort))
			require.Equal(t, "custom", sts.Spec.Template.Spec.Containers[1].Image)
			rtx.configmap = &corev1.ConfigMap{}
			rtx.headlessSvc = &corev1.Service{}
			require.Contains(t, evaluateSyncedStatus(rtx), "statefulset rollout is not complete")
			sts.Status.ObservedGeneration = sts.Generation
			sts.Status.UpdatedReplicas = 1
			sts.Status.CurrentRevision = "old"
			sts.Status.UpdateRevision = "new"
			rtx.sts = sts
			require.Contains(t, evaluateSyncedStatus(rtx), "statefulset rollout is not complete")
			sts.Status.CurrentRevision = "new"
			sts.Status.UpdatedReplicas = 0
			require.Contains(t, evaluateSyncedStatus(rtx), "statefulset rollout is not complete")
			sts.Status.UpdatedReplicas = 1
			require.Empty(t, evaluateSyncedStatus(rtx))
		})
	}
}

func TestRolloutStatusRespectsUpdateStrategy(t *testing.T) {
	for _, tc := range []struct {
		name             string
		strategy         appsv1.StatefulSetUpdateStrategyType
		observed         int64
		updated, ready   int32
		matching, synced bool
	}{
		{"OnDelete restarted", appsv1.OnDeleteStatefulSetStrategyType, 2, 1, 1, false, true},
		{"OnDelete old pod", appsv1.OnDeleteStatefulSetStrategyType, 2, 0, 1, false, false},
		{"OnDelete old generation", appsv1.OnDeleteStatefulSetStrategyType, 1, 1, 1, false, false},
		{"OnDelete pod not ready", appsv1.OnDeleteStatefulSetStrategyType, 2, 1, 0, false, false},
		{"RollingUpdate mismatched revision", appsv1.RollingUpdateStatefulSetStrategyType, 2, 1, 1, false, false},
		{"RollingUpdate complete", appsv1.RollingUpdateStatefulSetStrategyType, 2, 1, 1, true, true},
		{"default strategy mismatched revision", "", 2, 1, 1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cli := client.NewFakeClient()
			rtx := NewReconcileContext(t.Context(), types.NamespacedName{Namespace: "test", Name: "br"}, cli)
			rtx.tibr = &br.TiBR{ObjectMeta: metav1.ObjectMeta{Name: "br", Namespace: "test"}}
			rtx.cluster = &core.Cluster{}
			require.NoError(t, cli.Create(t.Context(), rtx.tibr))
			sts := assembleSts(rtx)
			sts.Spec.UpdateStrategy.Type = tc.strategy
			sts.Generation = 2
			sts.Status.ObservedGeneration = tc.observed
			sts.Status.UpdatedReplicas, sts.Status.ReadyReplicas = tc.updated, tc.ready
			sts.Status.CurrentRevision, sts.Status.UpdateRevision = "old", "new"
			if tc.matching {
				sts.Status.CurrentRevision = "new"
			}
			rtx.sts = sts
			rtx.configmap, rtx.headlessSvc = &corev1.ConfigMap{}, &corev1.Service{}
			result, _ := task.RunTask(t.Context(), taskWaitAllSubResourcesSynced(rtx))
			if tc.synced {
				require.Equal(t, task.SComplete, result.Status())
			} else {
				require.NotEqual(t, task.SComplete, result.Status())
			}
			result, _ = task.RunTask(t.Context(), TaskUpdateStatusIfNeeded(rtx))
			require.Equal(t, task.SComplete, result.Status())
			got := &br.TiBR{}
			require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(rtx.tibr), got))
			for _, name := range []string{br.TiBRCondReady, br.TiBRCondSynced} {
				condition := meta.FindStatusCondition(got.Status.Conditions, name)
				require.NotNil(t, condition)
				want := metav1.ConditionFalse
				if tc.synced {
					want = metav1.ConditionTrue
				}
				require.Equal(t, want, condition.Status)
			}
		})
	}
}
