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

package metrics

import (
	"fmt"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func newTiKVForMetricTest(name string) *v1alpha1.TiKV {
	return &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      name,
			Labels: map[string]string{
				v1alpha1.LabelKeyCluster:   "test-cluster",
				v1alpha1.LabelKeyComponent: "tikv",
				v1alpha1.LabelKeyGroup:     "test-group",
			},
		},
	}
}

// abnormalGaugeValue inspects collected samples without creating a missing series.
func abnormalGaugeValue(t *testing.T, instance, condition, reason string) (float64, bool) {
	t.Helper()
	return labelCombinationValue(t, []string{"test-ns", "test-cluster", "tikv", "test-group", instance, condition, reason})
}

func gaugeSeriesExists(t *testing.T, instance, condition, reason string) bool {
	t.Helper()
	_, present := abnormalGaugeValue(t, instance, condition, reason)
	return present
}

func labelsEqual(got []*dto.LabelPair, wantValues []string) bool {
	if len(got) != len(InstanceAbnormalMetricLabels) {
		return false
	}
	want := map[string]string{}
	for i, k := range InstanceAbnormalMetricLabels {
		want[k] = wantValues[i]
	}
	for _, p := range got {
		if want[p.GetName()] != p.GetValue() {
			return false
		}
	}
	return true
}

func TestObserveCondition(t *testing.T) {
	cases := []struct {
		name      string
		condType  string
		status    metav1.ConditionStatus
		omitCond  bool
		wantValue float64
	}{
		{name: "Synced=False sets 1", condType: v1alpha1.CondSynced, status: metav1.ConditionFalse, wantValue: 1},
		{name: "Synced=True sets 0", condType: v1alpha1.CondSynced, status: metav1.ConditionTrue, wantValue: 0},
		{name: "Synced absent treated as healthy", condType: v1alpha1.CondSynced, omitCond: true, wantValue: 0},
		{name: "Ready=False sets 1", condType: v1alpha1.CondReady, status: metav1.ConditionFalse, wantValue: 1},
		{name: "Ready=True sets 0", condType: v1alpha1.CondReady, status: metav1.ConditionTrue, wantValue: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instanceName := "tikv-" + tc.name

			obj := newTiKVForMetricTest(instanceName)
			defer ClearInstanceConditionMetrics(obj)
			var conds []metav1.Condition
			if !tc.omitCond {
				conds = []metav1.Condition{{Type: tc.condType, Status: tc.status}}
			}
			ObserveCondition(obj, conds, tc.condType)

			val, present := abnormalGaugeValue(t, instanceName, tc.condType, "")
			require.True(t, present)
			assert.InDelta(t, tc.wantValue, val, 1e-9)
			assert.True(t, gaugeSeriesExists(t, instanceName, tc.condType, ""),
				"series must remain present so PromQL `for:` alerts have continuous samples")
		})
	}
}

func TestObserveConditionSuspend(t *testing.T) {
	cases := []struct {
		name     string
		condType string
		reason   string
		want     float64
	}{
		{"suspended", v1alpha1.CondReady, v1alpha1.ReasonSuspended, 1},
		{"suspending", v1alpha1.CondReady, v1alpha1.ReasonSuspending, 1},
		{"pod not ready", v1alpha1.CondReady, v1alpha1.ReasonPodNotReady, 1},
		{"unhealthy", v1alpha1.CondReady, v1alpha1.ReasonInstanceNotHealthy, 1},
		{"deletion stuck", v1alpha1.CondSynced, v1alpha1.ReasonPodNotDeleted, 1},
		{"synced false with suspend reason", v1alpha1.CondSynced, v1alpha1.ReasonSuspended, 1},
		{"missing pod", v1alpha1.CondReady, v1alpha1.ReasonPodNotCreated, 1},
		{"terminating pod", v1alpha1.CondReady, v1alpha1.ReasonPodTerminating, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := newTiKVForMetricTest(tc.name)
			defer ClearInstanceConditionMetrics(obj)
			conds := []metav1.Condition{{Type: tc.condType, Status: metav1.ConditionFalse, Reason: tc.reason}}
			ObserveCondition(obj, conds, tc.condType)
			val, present := abnormalGaugeValue(t, obj.Name, tc.condType, tc.reason)
			require.True(t, present)
			assert.InDelta(t, tc.want, val, 1e-9)
			assert.True(t, gaugeSeriesExists(t, obj.Name, tc.condType, tc.reason))
			assert.Equal(t, metav1.ConditionFalse, conds[0].Status, "observation must not change conditions")
		})
	}
}

func TestObserveConditionsSuspendResume(t *testing.T) {
	obj := newTiKVForMetricTest("suspend-resume")
	defer ClearInstanceConditionMetrics(obj)
	conds := []metav1.Condition{
		{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonSuspended},
		{Type: v1alpha1.CondSynced, Status: metav1.ConditionTrue},
	}
	ObserveConditions(obj, conds)
	val, _ := abnormalGaugeValue(t, obj.Name, v1alpha1.CondReady, conds[0].Reason)
	assert.InDelta(t, float64(1), val, 1e-9)
	// A reason refresh replaces the label set while False still maps to 1.
	conds[0].Reason = v1alpha1.ReasonPodNotCreated
	ObserveConditions(obj, conds)
	val, _ = abnormalGaugeValue(t, obj.Name, v1alpha1.CondReady, conds[0].Reason)
	assert.InDelta(t, float64(1), val, 1e-9)
}

func TestClearInstanceConditionMetrics(t *testing.T) {
	name := "tikv-clear"
	obj := newTiKVForMetricTest(name)

	// Pre-populate both tracked conditions.
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse}}, v1alpha1.CondSynced)
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}, v1alpha1.CondReady)
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced, ""))
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondReady, ""))

	ClearInstanceConditionMetrics(obj)

	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced, ""),
		"Synced series must be removed on clear")
	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondReady, ""),
		"Ready series must be removed on clear")
}

func TestClearInstanceConditionMetricsByKey(t *testing.T) {
	const name = "tikv-byKey"
	obj := newTiKVForMetricTest(name)

	// Seed two series under the normal labels.
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse}}, v1alpha1.CondSynced)
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}, v1alpha1.CondReady)
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced, ""))
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondReady, ""))

	// Also seed a series with the same (namespace, component, instance) but
	// a drifted cluster / group label, simulating a rename during the
	// instance lifetime.
	driftedLabels := []string{"test-ns", "drifted-cluster", "tikv", "drifted-group", name, v1alpha1.CondReady, "old-reason"}
	AbnormalInstance.WithLabelValues(driftedLabels...).Set(1)

	// Seed a sibling instance (same namespace + same component) that must
	// survive the partial-match clear.
	sibling := newTiKVForMetricTest("tikv-sibling")
	ObserveCondition(sibling, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}, v1alpha1.CondReady)
	defer ClearInstanceConditionMetrics(sibling)

	// Seed another component that happens to share namespace + instance name
	// with the target. It must NOT be swept because the clear qualifies by
	// component.
	collisionLabels := []string{"test-ns", "test-cluster", "tidb", "test-group", name, v1alpha1.CondReady, "old-reason"}
	AbnormalInstance.WithLabelValues(collisionLabels...).Set(1)
	defer AbnormalInstance.DeleteLabelValues(collisionLabels...)

	ClearInstanceConditionMetricsByKey("test-ns", "tikv", name)

	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced, ""),
		"primary Synced series must be removed")
	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondReady, ""),
		"primary Ready series must be removed")

	// Drifted-label series must also be swept since namespace+component+instance match.
	assert.False(t, labelCombinationExists(t, driftedLabels),
		"series with drifted cluster / group labels must be swept by partial match")

	// Sibling instance under the same namespace and component must be untouched.
	assert.True(t, gaugeSeriesExists(t, "tikv-sibling", v1alpha1.CondReady, ""),
		"unrelated instance must not be swept")

	// Same namespace + same name but different component must NOT be swept.
	assert.True(t, labelCombinationExists(t, collisionLabels),
		"series for a different component sharing namespace+name must not be swept")
}

func labelCombinationExists(t *testing.T, want []string) bool {
	t.Helper()
	_, present := labelCombinationValue(t, want)
	return present
}

func labelCombinationValue(t *testing.T, want []string) (float64, bool) {
	t.Helper()
	ch := make(chan prometheus.Metric, 128)
	AbnormalInstance.Collect(ch)
	close(ch)
	for sample := range ch {
		metric := &dto.Metric{}
		require.NoError(t, sample.Write(metric))
		if labelsEqual(metric.GetLabel(), want) {
			return metric.GetGauge().GetValue(), true
		}
	}
	return 0, false
}

func TestObserveConditionReplacesReason(t *testing.T) {
	obj := newTiKVForMetricTest("reason-switch")
	defer ClearInstanceConditionMetrics(obj)
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse, Reason: "SyncError"}}, v1alpha1.CondSynced)
	previous := ""
	for _, tc := range []struct {
		reason string
		status metav1.ConditionStatus
		want   float64
	}{
		{v1alpha1.ReasonPodNotReady, metav1.ConditionFalse, 1},
		{v1alpha1.ReasonSuspending, metav1.ConditionFalse, 1},
		{v1alpha1.ReasonSuspended, metav1.ConditionFalse, 1},
		{v1alpha1.ReasonPodNotCreated, metav1.ConditionFalse, 1},
		{v1alpha1.ReasonReady, metav1.ConditionTrue, 0},
		{v1alpha1.ReasonUnknown, metav1.ConditionUnknown, 0},
	} {
		ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: tc.status, Reason: tc.reason}}, v1alpha1.CondReady)
		if previous != "" {
			assert.False(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondReady, previous), "old reason must be removed")
		}
		value, present := abnormalGaugeValue(t, obj.Name, v1alpha1.CondReady, tc.reason)
		require.True(t, present)
		assert.InDelta(t, tc.want, value, 1e-9)
		assert.True(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondSynced, "SyncError"), "other conditions must survive")
		previous = tc.reason
	}
	ObserveCondition(obj, nil, v1alpha1.CondReady)
	assert.False(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondReady, previous))
	value, present := abnormalGaugeValue(t, obj.Name, v1alpha1.CondReady, "")
	require.True(t, present)
	assert.InDelta(t, float64(0), value, 1e-9)
}

func TestObserveConditionPreservesUnchangedSeries(t *testing.T) {
	obj := newTiKVForMetricTest("same-reason")
	defer ClearInstanceConditionMetrics(obj)
	conds := []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotReady}}
	ObserveCondition(obj, conds, v1alpha1.CondReady)
	labels := append(instanceMetricBaseLabels(obj), v1alpha1.CondReady, v1alpha1.ReasonPodNotReady)
	original := AbnormalInstance.WithLabelValues(labels...)
	ObserveCondition(obj, conds, v1alpha1.CondReady)
	assert.Same(t, original, AbnormalInstance.WithLabelValues(labels...), "unchanged labels must not be deleted and recreated")
}

func TestClearInstanceConditionMetricsRemovesEveryReason(t *testing.T) {
	for _, byKey := range []bool{false, true} {
		t.Run(fmt.Sprint(byKey), func(t *testing.T) {
			obj := newTiKVForMetricTest("all-reasons")
			defer ClearInstanceConditionMetrics(obj)
			ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: "current"}}, v1alpha1.CondReady)
			// Simulate stale series written under earlier labels or reasons.
			old := []string{"test-ns", "old-cluster", "tikv", "old-group", obj.Name, v1alpha1.CondReady, "old"}
			AbnormalInstance.WithLabelValues(old...).Set(1)
			if byKey {
				ClearInstanceConditionMetricsByKey(obj.Namespace, "tikv", obj.Name)
			} else {
				ClearInstanceConditionMetrics(obj)
			}
			assert.False(t, labelCombinationExists(t, old))
			assert.False(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondReady, "current"))
			for key := range instanceConditionLabels {
				assert.NotEqual(t, obj.Name, key.instance, "cleanup must also release the tracked label set")
			}
			ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: "recreated"}}, v1alpha1.CondReady)
			assert.True(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondReady, "recreated"))
		})
	}
}

func TestConcurrentObserveAndClearInstanceMetrics(t *testing.T) {
	obj := newTiKVForMetricTest("concurrent")
	defer ClearInstanceConditionMetrics(obj)
	var workers sync.WaitGroup
	for worker := range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 10 {
				ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: fmt.Sprint(worker)}}, v1alpha1.CondReady)
				ClearInstanceConditionMetrics(obj)
			}
		}()
	}
	workers.Wait()
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: "final"}}, v1alpha1.CondReady)
	assert.True(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondReady, "final"))
	for worker := range 4 {
		assert.False(t, gaugeSeriesExists(t, obj.Name, v1alpha1.CondReady, fmt.Sprint(worker)))
	}
}
