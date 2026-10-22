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
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/metrics"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type suspendReadyState struct{ *fakeObjectState[*v1alpha1.PD] }

func (s *suspendReadyState) Key() types.NamespacedName {
	return types.NamespacedName{Namespace: s.obj.Namespace, Name: s.obj.Name}
}

func newSuspendReadyState(name string, suspend bool, ready *metav1.Condition) *suspendReadyState {
	obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: "suspend-ready", Name: strings.ReplaceAll(name, " ", "-"), Generation: 3}}
	obj.UID = types.UID(obj.Name)
	obj.Status.ObservedGeneration = 1
	obj.Status.ObservedClusterGeneration = 1
	obj.Status.Conditions = []metav1.Condition{*ready}
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Generation: 4}}
	cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: suspend}
	return &suspendReadyState{&fakeObjectState[*v1alpha1.PD]{obj: obj, cluster: cluster}}
}

func transitionPod(name string, terminating bool) *corev1.Pod {
	controller := true
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "suspend-ready", Name: "pod",
		OwnerReferences: []metav1.OwnerReference{{Name: name, UID: types.UID(name), Controller: &controller}}, Labels: map[string]string{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			v1alpha1.LabelKeyInstance:  name,
		},
	}}
	pod.Status.Phase = corev1.PodRunning
	pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	if terminating {
		pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
	return pod
}

func TestTaskInstanceSuspendReadyTransitions(t *testing.T) {
	for _, tc := range []struct {
		name        string
		suspend     bool
		pod         bool
		terminating bool
		wantReason  string
	}{
		{"suspend healthy pod", true, true, false, v1alpha1.ReasonSuspending},
		{"suspend terminating", true, true, true, v1alpha1.ReasonSuspending},
		{"suspend absent", true, false, false, v1alpha1.ReasonSuspended},
		{"resume existing pod", false, true, false, v1alpha1.ReasonPodNotReady},
		{"resume terminating", false, true, true, v1alpha1.ReasonPodTerminating},
		{"resume absent", false, false, false, v1alpha1.ReasonPodNotCreated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initial := coreutil.Ready()
			if !tc.suspend {
				initial = coreutil.Unready(v1alpha1.ReasonSuspended)
			}
			s := newSuspendReadyState(tc.name, tc.suspend, initial)
			defer metrics.ClearInstanceConditionMetrics(s.obj)
			objs := []client.Object{s.obj}
			if tc.pod {
				objs = append(objs, transitionPod(s.obj.Name, tc.terminating))
			}
			c := client.NewFakeClient(objs...)
			res, _ := task.RunTask(context.Background(), task.Block(
				TaskInstanceSuspendReady[scope.PD](s, c), TaskObserveInstance[scope.PD](s),
			))
			require.Equal(t, task.SComplete, res.Status(), res.Message())
			stored := &v1alpha1.PD{}
			require.NoError(t, c.Get(context.Background(), s.Key(), stored))
			ready := coreutil.FindStatusCondition[scope.PD](stored, v1alpha1.CondReady)
			require.NotNil(t, ready)
			assert.Equal(t, metav1.ConditionFalse, ready.Status)
			assert.Equal(t, tc.wantReason, ready.Reason)
			assert.Equal(t, int64(3), ready.ObservedGeneration)
			assert.Equal(t, int64(1), stored.Status.ObservedGeneration)
			assert.Equal(t, int64(1), stored.Status.ObservedClusterGeneration)
			want := float64(1)
			if tc.suspend {
				want = 0
			}
			assert.InDelta(t, want, testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(
				stored.Namespace, "", "", "", stored.Name, v1alpha1.CondReady)), 1e-9)
		})
	}
}

func TestTaskInstanceSuspendReadyResumeFailureRetry(t *testing.T) {
	ctx := context.Background()
	s := newSuspendReadyState("resume-retry", false, coreutil.Unready(v1alpha1.ReasonSuspended))
	defer metrics.ClearInstanceConditionMetrics(s.obj)
	c := client.NewFakeClient(s.obj)
	// Model recovery failing before the normal health check, across fresh reconciles.
	for range 3 {
		require.NoError(t, c.Get(ctx, s.Key(), s.obj))
		res, _ := task.RunTask(ctx, task.Block(
			TaskInstanceSuspendReady[scope.PD](s, c), TaskObserveInstance[scope.PD](s),
			task.NameTaskFunc("Recovery", func(context.Context) task.Result { return task.Fail().With("pod creation failed") }),
		))
		require.Equal(t, task.SFail, res.Status())
		stored := &v1alpha1.PD{}
		require.NoError(t, c.Get(ctx, s.Key(), stored))
		ready := coreutil.FindStatusCondition[scope.PD](stored, v1alpha1.CondReady)
		require.NotNil(t, ready)
		assert.Equal(t, metav1.ConditionFalse, ready.Status)
		assert.Equal(t, v1alpha1.ReasonPodNotCreated, ready.Reason)
		assert.InDelta(t, float64(1), testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(
			stored.Namespace, "", "", "", stored.Name, v1alpha1.CondReady)), 1e-9)
	}
}

func TestTaskInstanceSuspendReadyDoesNotFlap(t *testing.T) {
	ctx := context.Background()
	s := newSuspendReadyState("stable", false, coreutil.Unready(v1alpha1.ReasonSuspended))
	c := client.NewFakeClient(s.obj, transitionPod(s.obj.Name, false))
	res, _ := task.RunTask(ctx, TaskInstanceSuspendReady[scope.PD](s, c))
	require.Equal(t, task.SComplete, res.Status(), res.Message())
	pod := transitionPod(s.obj.Name, false)
	pod.Status.Phase = corev1.PodRunning
	pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	health := NewMockInstanceCondReadyUpdater[*v1alpha1.PD](gomock.NewController(t))
	health.EXPECT().Object().Return(s.obj).AnyTimes()
	health.EXPECT().Cluster().Return(s.cluster).AnyTimes()
	health.EXPECT().Pod().Return(pod).AnyTimes()
	health.EXPECT().IsPodTerminating().Return(false).AnyTimes()
	health.EXPECT().IsHealthy().Return(true).AnyTimes()
	health.EXPECT().SetStatusChanged().Times(1)
	res, _ = task.RunTask(ctx, TaskInstanceConditionReady[scope.PD](health))
	require.Equal(t, task.SComplete, res.Status(), res.Message())
	require.NoError(t, c.Status().Update(ctx, s.obj))
	ready := *coreutil.FindStatusCondition[scope.PD](s.obj, v1alpha1.CondReady)
	require.Equal(t, metav1.ConditionTrue, ready.Status)
	defer metrics.ClearInstanceConditionMetrics(s.obj)
	// Normal reconciles must not list Pods or write a pessimistic Ready=False first.
	c.WithError("list", "pods", errors.NewInternalError(fmt.Errorf("unexpected pod list")))
	c.WithError("update", "pds", errors.NewInternalError(fmt.Errorf("unexpected status write")))
	for range 3 {
		require.NoError(t, c.Get(ctx, s.Key(), s.obj))
		res, _ = task.RunTask(ctx, task.Block(
			TaskInstanceSuspendReady[scope.PD](s, c), TaskInstanceConditionReady[scope.PD](health),
		))
		require.Equal(t, task.SComplete, res.Status(), res.Message())
		assert.Equal(t, ready, *coreutil.FindStatusCondition[scope.PD](s.obj, v1alpha1.CondReady),
			"Ready and LastTransitionTime must remain unchanged after recovery")
	}
}

func TestTaskInstanceSuspendReadyErrors(t *testing.T) {
	for _, verb := range []string{"list", "update"} {
		t.Run(verb, func(t *testing.T) {
			s := newSuspendReadyState(verb, false, coreutil.Unready(v1alpha1.ReasonSuspended))
			defer metrics.ClearInstanceConditionMetrics(s.obj)
			gauge := metrics.AbnormalInstance.WithLabelValues(s.obj.Namespace, "", "", "", s.obj.Name, v1alpha1.CondReady)
			gauge.Set(0) // The previous suspend reconcile suppressed the Ready alert.
			c := client.NewFakeClient(s.obj)
			resource := "pods"
			if verb == "update" {
				resource = "pds"
			}
			c.WithError(verb, resource, errors.NewInternalError(fmt.Errorf("injected error")))
			res, _ := task.RunTask(context.Background(), TaskInstanceSuspendReady[scope.PD](s, c))
			require.Equal(t, task.SFail, res.Status())
			assert.InDelta(t, float64(1), testutil.ToFloat64(gauge), 1e-9, "transition failures must stop suppressing resume alerts")
			stored := &v1alpha1.PD{}
			require.NoError(t, c.Get(context.Background(), s.Key(), stored))
			assert.Equal(t, v1alpha1.ReasonSuspended, coreutil.FindStatusCondition[scope.PD](stored, v1alpha1.CondReady).Reason)
			// A subsequent reconcile retries from persisted state.
			retry := client.NewFakeClient(stored)
			s.obj = stored
			res, _ = task.RunTask(context.Background(), TaskInstanceSuspendReady[scope.PD](s, retry))
			assert.Equal(t, task.SComplete, res.Status())
			assert.Equal(t, v1alpha1.ReasonPodNotCreated, coreutil.FindStatusCondition[scope.PD](s.obj, v1alpha1.CondReady).Reason)
		})
	}
}

func TestTaskInstanceSuspendReadySkipsPausedAndDeleting(t *testing.T) {
	for _, scenario := range []string{"paused", "cluster deleting", "instance deleting"} {
		t.Run(scenario, func(t *testing.T) {
			s := newSuspendReadyState(scenario, true, coreutil.Ready())
			now := metav1.Now()
			switch scenario {
			case "paused":
				s.cluster.Spec.Paused = true
			case "cluster deleting":
				s.cluster.DeletionTimestamp = &now
			case "instance deleting":
				s.obj.DeletionTimestamp = &now
			}
			before := s.obj.DeepCopy()
			c := client.NewFakeClient(s.obj)
			c.WithError("*", "*", errors.NewInternalError(fmt.Errorf("unexpected API request")))
			res, _ := task.RunTask(context.Background(), TaskInstanceSuspendReady[scope.PD](s, c))
			require.Equal(t, task.SComplete, res.Status(), res.Message())
			assert.Equal(t, before, s.obj)
		})
	}
}

func TestTaskInstanceConditionReadyHealthyPodSuspending(t *testing.T) {
	s := newSuspendReadyState("healthy-suspending", true, coreutil.Ready())
	defer metrics.ClearInstanceConditionMetrics(s.obj)
	health := NewMockInstanceCondReadyUpdater[*v1alpha1.PD](gomock.NewController(t))
	health.EXPECT().Object().Return(s.obj)
	health.EXPECT().Cluster().Return(s.cluster)
	health.EXPECT().Pod().Return(transitionPod(s.obj.Name, false))
	health.EXPECT().SetStatusChanged()
	res, _ := task.RunTask(context.Background(), TaskInstanceConditionReady[scope.PD](health))
	require.Equal(t, task.SWait, res.Status())
	ready := coreutil.FindStatusCondition[scope.PD](s.obj, v1alpha1.CondReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionFalse, ready.Status)
	assert.Equal(t, v1alpha1.ReasonSuspending, ready.Reason)
}
