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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type volumeCapacityState struct {
	*mockTaskPVCState
	changed bool
}

func (s *volumeCapacityState) SetStatusChanged()     { s.changed = true }
func (s *volumeCapacityState) IsStatusChanged() bool { return s.changed }

func TestTaskInstanceConditionVolumeCapacityExceedsRequest(t *testing.T) {
	ctx := context.Background()
	obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Name: "pd-0", Namespace: "default", Generation: 7}}
	obj.Spec.Volumes = []v1alpha1.Volume{{Name: "data", Storage: apiresource.MustParse("50Gi")}}
	obj.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionTrue, Reason: "Ready"}}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: coreutil.PersistentVolumeClaimName[scope.PD](obj, "data"), Namespace: obj.Namespace}, Status: corev1.PersistentVolumeClaimStatus{Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("100Gi")}}}
	cli := client.NewFakeClient(obj, pvc)
	state := &volumeCapacityState{mockTaskPVCState: &mockTaskPVCState{obj: obj}}
	// Condition tasks must not write status themselves, even when the condition changes.
	cli.WithError("update", "pds", fmt.Errorf("unexpected status write"))
	run := func() {
		res, _ := task.RunTask(ctx, TaskInstanceConditionVolumeCapacityExceedsRequest[scope.PD](state, cli))
		require.Equal(t, task.SComplete, res.Status())
	}
	run()
	require.True(t, state.changed)
	cond := meta.FindStatusCondition(obj.Status.Conditions, v1alpha1.CondVolumeCapacityExceedsRequest)
	require.NotNil(t, cond)
	require.Equal(t, metav1.ConditionTrue, cond.Status)
	require.EqualValues(t, 7, cond.ObservedGeneration)
	require.True(t, meta.IsStatusConditionTrue(obj.Status.Conditions, v1alpha1.CondReady))
	transition := cond.LastTransitionTime
	var actual v1alpha1.PD
	require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(obj), &actual))
	require.Nil(t, meta.FindStatusCondition(actual.Status.Conditions, v1alpha1.CondVolumeCapacityExceedsRequest))
	state.changed = false
	run()
	require.False(t, state.changed)
	require.Equal(t, transition, meta.FindStatusCondition(obj.Status.Conditions, v1alpha1.CondVolumeCapacityExceedsRequest).LastTransitionTime)
	obj.Spec.Volumes[0].Storage = apiresource.MustParse("100Gi")
	obj.Generation++
	run()
	require.True(t, state.changed)
	cond = meta.FindStatusCondition(obj.Status.Conditions, v1alpha1.CondVolumeCapacityExceedsRequest)
	require.Equal(t, metav1.ConditionFalse, cond.Status)
	require.EqualValues(t, 8, cond.ObservedGeneration)
	require.False(t, cond.LastTransitionTime.Before(&transition))
	// A wait from another task still lets the shared status persister run.
	persisterClient := client.NewFakeClient(&actual)
	res, _ := task.RunTask(ctx, task.Block(
		task.NameTaskFunc("Waiting", func(context.Context) task.Result { return task.Wait().With("waiting for PVC") }),
		TaskStatusPersister[scope.PD](state, persisterClient),
	))
	require.Equal(t, task.SWait, res.Status())
	require.NoError(t, persisterClient.Get(ctx, client.ObjectKeyFromObject(obj), &actual))
	require.True(t, meta.IsStatusConditionFalse(actual.Status.Conditions, v1alpha1.CondVolumeCapacityExceedsRequest))
}
