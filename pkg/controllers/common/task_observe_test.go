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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
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
	state := &fakeState[v1alpha1.PD]{ns: ns, name: name, obj: nil}

	res, done := task.RunTask(context.Background(), TaskObserveInstance[scope.PD](state))
	require.Equal(t, task.SComplete, res.Status())
	require.False(t, done)
	assert.False(t, hasGaugeSample(t, ns, name),
		"ObserveInstance must clear every series for the instance when the object is gone")
}
