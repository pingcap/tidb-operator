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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/metrics"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

// hasGaugeSample returns true if the AbnormalInstance gauge has any sample
// whose (namespace, instance) labels match the inputs.
func hasGaugeSample(t *testing.T, namespace, instance string) bool {
	t.Helper()
	ch := make(chan prometheus.Metric, 32)
	metrics.AbnormalInstance.Collect(ch)
	close(ch)
	for m := range ch {
		dm := &dto.Metric{}
		if err := m.Write(dm); err != nil {
			continue
		}
		labels := map[string]string{}
		for _, lp := range dm.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		if labels["namespace"] == namespace && labels["instance"] == instance {
			return true
		}
	}
	return false
}

func TestTaskObserveInstance_ObservesConditions(t *testing.T) {
	const ns, name = "ns-observe", "pd-observe"
	defer metrics.ClearInstanceConditionMetricsByKey(ns, v1alpha1.LabelValComponentPD, name)

	obj := fake.FakeObj(name, func(o *v1alpha1.PD) *v1alpha1.PD {
		o.Namespace = ns
		o.Labels = map[string]string{
			v1alpha1.LabelKeyCluster:   "c",
			v1alpha1.LabelKeyComponent: "pd",
			v1alpha1.LabelKeyGroup:     "g",
		}
		o.Status.Conditions = []metav1.Condition{
			{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse},
			{Type: v1alpha1.CondSynced, Status: metav1.ConditionTrue},
		}
		return o
	})
	state := &struct {
		*fakeState[v1alpha1.PD]
		ClusterState
	}{&fakeState[v1alpha1.PD]{ns: ns, name: name, obj: obj}, ClusterStateFunc(func() *v1alpha1.Cluster { return nil })}

	res, done := task.RunTask(context.Background(), TaskObserveInstance[scope.PD](state))
	require.Equal(t, task.SComplete, res.Status())
	require.False(t, done)
	assert.True(t, hasGaugeSample(t, ns, name),
		"ObserveInstance must write at least one series for the instance")
}

func TestTaskObserveInstance_ClearsOnMissingObject(t *testing.T) {
	const ns, name = "ns-clear", "pd-clear"

	// Pre-populate a series as if a previous reconcile had observed the instance.
	seed := fake.FakeObj(name, func(o *v1alpha1.PD) *v1alpha1.PD {
		o.Namespace = ns
		o.Labels = map[string]string{
			v1alpha1.LabelKeyCluster:   "c",
			v1alpha1.LabelKeyComponent: "pd",
			v1alpha1.LabelKeyGroup:     "g",
		}
		return o
	})
	metrics.ObserveConditions(seed, []metav1.Condition{
		{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse},
	}, nil)
	require.True(t, hasGaugeSample(t, ns, name), "test precondition: seed series exists")

	// Simulate a reconcile where TaskContextObject saw NotFound.
	state := &struct {
		*fakeState[v1alpha1.PD]
		ClusterState
	}{fakeState: &fakeState[v1alpha1.PD]{ns: ns, name: name}}

	res, done := task.RunTask(context.Background(), TaskObserveInstance[scope.PD](state))
	require.Equal(t, task.SComplete, res.Status())
	require.False(t, done)
	assert.False(t, hasGaugeSample(t, ns, name),
		"ObserveInstance must clear every series for the instance when the object is gone")
}

func TestTaskObserveInstanceSuspendResume(t *testing.T) {
	const ns, name = "ns-suspend", "pd-suspend"
	obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}}
	obj.Status.Conditions = []metav1.Condition{
		{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotCreated},
		{Type: v1alpha1.CondSynced, Status: metav1.ConditionTrue},
		{Type: v1alpha1.CondSuspended, Status: metav1.ConditionTrue},
	}
	defer metrics.ClearInstanceConditionMetrics(obj)
	cluster := &v1alpha1.Cluster{}
	cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
	state := &struct {
		*fakeState[v1alpha1.PD]
		ClusterState
	}{&fakeState[v1alpha1.PD]{ns: ns, name: name, obj: obj}, ClusterStateFunc(func() *v1alpha1.Cluster { return cluster })}
	for _, suspend := range []bool{true, false} {
		cluster.Spec.SuspendAction.SuspendCompute = suspend
		res, done := task.RunTask(context.Background(), TaskObserveInstance[scope.PD](state))
		require.Equal(t, task.SComplete, res.Status())
		require.False(t, done)
		want := float64(1)
		if suspend {
			want = 0
		}
		assert.Equal(t, want, testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(ns, "", "", "", name, v1alpha1.CondReady)))
	}
}

func TestTaskContextClusterObservesInstanceOnError(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ready         metav1.ConditionStatus
		internalError bool
		want          float64
	}{
		{"not found unready", metav1.ConditionFalse, false, 1},
		{"not found ready", metav1.ConditionTrue, false, 0},
		{"read error unready", metav1.ConditionFalse, true, 1},
		{"read error ready", metav1.ConditionTrue, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-read-error", Name: tc.name}}
			obj.Spec.Cluster.Name = "c"
			obj.Status.Conditions = []metav1.Condition{
				{Type: v1alpha1.CondReady, Status: tc.ready, Reason: v1alpha1.ReasonPodNotCreated},
				{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotDeleted},
				{Type: v1alpha1.CondSuspended, Status: metav1.ConditionTrue},
			}
			defer metrics.ClearInstanceConditionMetrics(obj)
			state := newFakeObjectState(obj)
			// A stale suspended Cluster must not suppress the fallback observation.
			state.cluster = &v1alpha1.Cluster{}
			state.cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
			fc := client.NewFakeClient()
			if tc.internalError {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("read failed")))
			}
			res, done := task.RunTask(context.Background(), TaskContextCluster[scope.PD](state, fc))
			assert.Equal(t, task.SFail, res.Status())
			assert.False(t, done)
			require.True(t, hasGaugeSample(t, obj.Namespace, obj.Name), "fallback must publish metrics before returning the error")
			assert.Equal(t, tc.want, testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(obj.Namespace, "", "", "", obj.Name, v1alpha1.CondReady)))
			assert.Equal(t, float64(1), testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(obj.Namespace, "", "", "", obj.Name, v1alpha1.CondSynced)))
		})
	}
}

func TestTaskContextClusterDoesNotObserveGroupOnError(t *testing.T) {
	obj := &v1alpha1.PDGroup{ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-read-error", Name: "group"}}
	obj.Spec.Cluster.Name = "c"
	obj.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}
	defer metrics.ClearInstanceConditionMetrics(obj)
	res, done := task.RunTask(context.Background(), TaskContextCluster[scope.PDGroup](newFakeObjectState(obj), client.NewFakeClient()))
	assert.Equal(t, task.SFail, res.Status())
	assert.False(t, done)
	assert.False(t, hasGaugeSample(t, obj.Namespace, obj.Name), "Groups must not produce instance metrics")
}

func TestTaskContextClusterObservationRecovers(t *testing.T) {
	ctx := context.Background()
	obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-read-error", Name: "recovery"}}
	obj.Spec.Cluster.Name = "c"
	obj.Status.Conditions = []metav1.Condition{
		{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotCreated},
		{Type: v1alpha1.CondSynced, Status: metav1.ConditionTrue},
		{Type: v1alpha1.CondSuspended, Status: metav1.ConditionTrue},
	}
	defer metrics.ClearInstanceConditionMetrics(obj)
	state := newFakeObjectState(obj)
	fc := client.NewFakeClient()
	res, _ := task.RunTask(ctx, TaskContextCluster[scope.PD](state, fc))
	require.Equal(t, task.SFail, res.Status())
	require.True(t, hasGaugeSample(t, obj.Namespace, obj.Name))
	gauge := metrics.AbnormalInstance.WithLabelValues(obj.Namespace, "", "", "", obj.Name, v1alpha1.CondReady)
	assert.Equal(t, float64(1), testutil.ToFloat64(gauge))

	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Namespace: obj.Namespace, Name: "c"}}
	cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
	require.NoError(t, fc.Create(ctx, cluster))
	observeState := &struct {
		*fakeState[v1alpha1.PD]
		ClusterState
	}{&fakeState[v1alpha1.PD]{ns: obj.Namespace, name: obj.Name, obj: obj}, state}
	res, _ = task.RunTask(ctx, task.Block(TaskContextCluster[scope.PD](state, fc), TaskObserveInstance[scope.PD](observeState)))
	require.Equal(t, task.SComplete, res.Status())
	assert.Equal(t, float64(0), testutil.ToFloat64(gauge), "successful Cluster read must restore suspend filtering")
}
