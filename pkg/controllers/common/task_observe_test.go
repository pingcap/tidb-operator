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
	"go.uber.org/mock/gomock"
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
	state := &fakeState[v1alpha1.PD]{ns: ns, name: name, obj: obj}

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
	})
	require.True(t, hasGaugeSample(t, ns, name), "test precondition: seed series exists")

	// Simulate a reconcile where TaskContextObject saw NotFound.
	state := &fakeState[v1alpha1.PD]{ns: ns, name: name}

	res, done := task.RunTask(context.Background(), TaskObserveInstance[scope.PD](state))
	require.Equal(t, task.SComplete, res.Status())
	require.False(t, done)
	assert.False(t, hasGaugeSample(t, ns, name),
		"ObserveInstance must clear every series for the instance when the object is gone")
}

func TestTaskObserveInstanceSuspendResume(t *testing.T) {
	const ns, name = "ns-suspend", "pd-suspend"
	obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}}
	obj.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonSuspended}}
	defer metrics.ClearInstanceConditionMetrics(obj)
	state := &fakeState[v1alpha1.PD]{ns: ns, name: name, obj: obj}
	for _, tc := range []struct {
		reason string
		want   float64
	}{
		{v1alpha1.ReasonSuspended, 0}, {v1alpha1.ReasonSuspending, 0}, {v1alpha1.ReasonPodNotCreated, 1},
	} {
		obj.Status.Conditions[0].Reason = tc.reason
		res, done := task.RunTask(context.Background(), TaskObserveInstance[scope.PD](state))
		require.Equal(t, task.SComplete, res.Status())
		require.False(t, done)
		assert.InDelta(t, tc.want, testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(ns, "", "", "", name, v1alpha1.CondReady)), 1e-9)
	}
}

func TestTaskObserveInstanceBeforeClusterReadError(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ready         metav1.ConditionStatus
		reason        string
		internalError bool
		want          float64
	}{
		{"not found unready", metav1.ConditionFalse, v1alpha1.ReasonPodNotCreated, false, 1},
		{"not found ready", metav1.ConditionTrue, v1alpha1.ReasonReady, false, 0},
		{"read error unready", metav1.ConditionFalse, v1alpha1.ReasonPodNotCreated, true, 1},
		{"read error ready", metav1.ConditionTrue, v1alpha1.ReasonReady, true, 0},
		{"read error keeps suspended", metav1.ConditionFalse, v1alpha1.ReasonSuspended, true, 0},
		{"not found keeps suspending", metav1.ConditionFalse, v1alpha1.ReasonSuspending, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-read-error", Name: tc.name}}
			obj.Spec.Cluster.Name = "c"
			obj.Status.Conditions = []metav1.Condition{
				{Type: v1alpha1.CondReady, Status: tc.ready, Reason: tc.reason},
				{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotDeleted},
				{Type: v1alpha1.CondSuspended, Status: metav1.ConditionTrue},
			}
			defer metrics.ClearInstanceConditionMetrics(obj)
			state := newFakeObjectState(obj)
			fc := client.NewFakeClient()
			if tc.internalError {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("read failed")))
			}
			observeState := &fakeState[v1alpha1.PD]{ns: obj.Namespace, name: obj.Name, obj: obj}
			res, done := task.RunTask(context.Background(), task.Block(
				TaskObserveInstance[scope.PD](observeState), TaskContextCluster[scope.PD](state, fc),
			))
			assert.Equal(t, task.SFail, res.Status())
			assert.False(t, done)
			require.True(t, hasGaugeSample(t, obj.Namespace, obj.Name), "fallback must publish metrics before returning the error")
			assert.InDelta(t, tc.want, testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(obj.Namespace, "", "", "", obj.Name, v1alpha1.CondReady)), 1e-9)
			assert.InDelta(t, float64(1), testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(obj.Namespace, "", "", "", obj.Name, v1alpha1.CondSynced)), 1e-9)
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

func TestTaskObserveInstanceKeepsSuspendConditionUntilReadyRefresh(t *testing.T) {
	ctx := context.Background()
	obj := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Namespace: "observe-resume", Name: "pd"}}
	obj.Spec.Cluster.Name = "c"
	obj.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonSuspended}}
	defer metrics.ClearInstanceConditionMetrics(obj)
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Namespace: obj.Namespace, Name: "c"}}
	cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: false}
	fc := client.NewFakeClient(obj, cluster)
	state := newFakeObjectState(obj)
	observeState := &fakeState[v1alpha1.PD]{ns: obj.Namespace, name: obj.Name, obj: obj}
	for range 3 {
		res, _ := task.RunTask(ctx, task.Block(
			TaskObserveInstance[scope.PD](observeState), TaskContextCluster[scope.PD](state, fc),
			task.NameTaskFunc("Recovery", func(context.Context) task.Result { return task.Fail().With("recovery failed before Ready") }),
		))
		require.Equal(t, task.SFail, res.Status())
		assert.Equal(t, v1alpha1.ReasonSuspended, obj.Status.Conditions[0].Reason)
		assert.InDelta(t, float64(0), testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(
			obj.Namespace, "", "", "", obj.Name, v1alpha1.CondReady)), 1e-9)
	}
	// Once the normal Ready task runs, the actual Pod state replaces the old reason.
	health := NewMockInstanceCondReadyUpdater[*v1alpha1.PD](gomock.NewController(t))
	health.EXPECT().Object().Return(obj)
	health.EXPECT().Cluster().Return(cluster)
	health.EXPECT().Pod().Return(nil)
	health.EXPECT().SetStatusChanged()
	res, _ := task.RunTask(ctx, TaskInstanceConditionReady[scope.PD](health))
	require.Equal(t, task.SWait, res.Status())
	assert.Equal(t, v1alpha1.ReasonPodNotCreated, obj.Status.Conditions[0].Reason)
	assert.InDelta(t, float64(1), testutil.ToFloat64(metrics.AbnormalInstance.WithLabelValues(
		obj.Namespace, "", "", "", obj.Name, v1alpha1.CondReady)), 1e-9)
}
