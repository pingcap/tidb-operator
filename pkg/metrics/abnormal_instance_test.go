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

// abnormalGaugeValue reads the AbnormalInstance gauge for the given instance
// + condition. Returns (value, present). present is false when the series has
// not been touched (collector returns the default zero metric without a Gauge
// payload set).
func abnormalGaugeValue(t *testing.T, instance, condition string) (float64, bool) {
	t.Helper()
	g, err := AbnormalInstance.GetMetricWithLabelValues(
		"test-ns", "test-cluster", "tikv", "test-group", instance, condition,
	)
	require.NoError(t, err)
	m := &dto.Metric{}
	require.NoError(t, g.Write(m))
	if m.Gauge == nil {
		return 0, false
	}
	return m.Gauge.GetValue(), true
}

func gaugeSeriesExists(t *testing.T, instance, condition string) bool {
	t.Helper()
	ch := make(chan prometheus.Metric, 16)
	AbnormalInstance.Collect(ch)
	close(ch)
	want := []string{"test-ns", "test-cluster", "tikv", "test-group", instance, condition}
	for m := range ch {
		dm := &dto.Metric{}
		if err := m.Write(dm); err != nil {
			continue
		}
		if labelsEqual(dm.GetLabel(), want) {
			return true
		}
	}
	return false
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
			defer AbnormalInstance.DeleteLabelValues(
				"test-ns", "test-cluster", "tikv", "test-group", instanceName, tc.condType,
			)

			obj := newTiKVForMetricTest(instanceName)
			var conds []metav1.Condition
			if !tc.omitCond {
				conds = []metav1.Condition{{Type: tc.condType, Status: tc.status}}
			}
			ObserveCondition(obj, conds, tc.condType)

			val, present := abnormalGaugeValue(t, instanceName, tc.condType)
			require.True(t, present)
			assert.InDelta(t, tc.wantValue, val, 1e-9)
			assert.True(t, gaugeSeriesExists(t, instanceName, tc.condType),
				"series must remain present so PromQL `for:` alerts have continuous samples")
		})
	}
}

func TestClearInstanceConditionMetrics(t *testing.T) {
	name := "tikv-clear"
	obj := newTiKVForMetricTest(name)

	// Pre-populate both tracked conditions.
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse}}, v1alpha1.CondSynced)
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}, v1alpha1.CondReady)
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced))
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondReady))

	ClearInstanceConditionMetrics(obj)

	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced),
		"Synced series must be removed on clear")
	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondReady),
		"Ready series must be removed on clear")
}

func TestClearInstanceConditionMetricsByKey(t *testing.T) {
	const name = "tikv-byKey"
	obj := newTiKVForMetricTest(name)

	// Seed two series under the normal labels.
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse}}, v1alpha1.CondSynced)
	ObserveCondition(obj, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}, v1alpha1.CondReady)
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced))
	require.True(t, gaugeSeriesExists(t, name, v1alpha1.CondReady))

	// Also seed a series with the same (namespace, component, instance) but
	// a drifted cluster / group label, simulating a rename during the
	// instance lifetime.
	driftedLabels := []string{"test-ns", "drifted-cluster", "tikv", "drifted-group", name, v1alpha1.CondReady}
	AbnormalInstance.WithLabelValues(driftedLabels...).Set(1)

	// Seed a sibling instance (same namespace + same component) that must
	// survive the partial-match clear.
	sibling := newTiKVForMetricTest("tikv-sibling")
	ObserveCondition(sibling, []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse}}, v1alpha1.CondReady)
	defer ClearInstanceConditionMetrics(sibling)

	// Seed another component that happens to share namespace + instance name
	// with the target. It must NOT be swept because the clear qualifies by
	// component.
	collisionLabels := []string{"test-ns", "test-cluster", "tidb", "test-group", name, v1alpha1.CondReady}
	AbnormalInstance.WithLabelValues(collisionLabels...).Set(1)
	defer AbnormalInstance.DeleteLabelValues(collisionLabels...)

	ClearInstanceConditionMetricsByKey("test-ns", "tikv", name)

	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondSynced),
		"primary Synced series must be removed")
	assert.False(t, gaugeSeriesExists(t, name, v1alpha1.CondReady),
		"primary Ready series must be removed")

	// Drifted-label series must also be swept since namespace+component+instance match.
	assert.False(t, labelCombinationExists(t, driftedLabels),
		"series with drifted cluster / group labels must be swept by partial match")

	// Sibling instance under the same namespace and component must be untouched.
	assert.True(t, gaugeSeriesExists(t, "tikv-sibling", v1alpha1.CondReady),
		"unrelated instance must not be swept")

	// Same namespace + same name but different component must NOT be swept.
	assert.True(t, labelCombinationExists(t, collisionLabels),
		"series for a different component sharing namespace+name must not be swept")
}

func labelCombinationExists(t *testing.T, want []string) bool {
	t.Helper()
	ch := make(chan prometheus.Metric, 32)
	AbnormalInstance.Collect(ch)
	close(ch)
	for m := range ch {
		dm := &dto.Metric{}
		if err := m.Write(dm); err != nil {
			continue
		}
		if labelsEqual(dm.GetLabel(), want) {
			return true
		}
	}
	return false
}
