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
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// trackedConditions are the condition types observed by ObserveInstance* and
// cleared by ClearInstanceConditionMetrics. Add new types here when extending
// the AbnormalInstance gauge to additional condition signals.
var trackedConditions = []string{v1alpha1.CondSynced, v1alpha1.CondReady}

// instanceMetricBaseLabels reads the standard well-known labels carried by
// every managed instance (LabelKeyCluster / LabelKeyComponent / LabelKeyGroup)
// plus its namespace and name. The caller appends the per-condition label.
func instanceMetricBaseLabels(obj client.Object) []string {
	ls := obj.GetLabels()
	return []string{
		obj.GetNamespace(),
		ls[v1alpha1.LabelKeyCluster],
		ls[v1alpha1.LabelKeyComponent],
		ls[v1alpha1.LabelKeyGroup],
		obj.GetName(),
	}
}

// ObserveCondition writes 1 to the abnormal-instance gauge when the named
// condition is False; 0 otherwise (True or absent are treated as healthy).
// The series stays present so PromQL `for:` alerts can fire reliably without
// gaps, and so dashboards never see missing samples for managed instances.
//
// The 'suspended' label value is derived from the Suspended condition in
// conds: "true" when Suspended=True, "false" otherwise. This allows alert
// rules to use {suspended!="true"} to suppress noise on paused clusters.
//
// condType must be one of trackedConditions so the finalize-time cleanup in
// ClearInstanceConditionMetrics covers the same set of series this writes.
func ObserveCondition(obj client.Object, conds []metav1.Condition, condType string) {
	suspended := meta.IsStatusConditionTrue(conds, v1alpha1.CondSuspended)
	labels := append(instanceMetricBaseLabels(obj), condType)

	// Remove the previous state so paused instances cannot keep stale alerts alive.
	AbnormalInstance.DeleteLabelValues(append(labels, strconv.FormatBool(!suspended))...)
	labels = append(labels, strconv.FormatBool(suspended))

	value := 0.0
	if cond := meta.FindStatusCondition(conds, condType); cond != nil && cond.Status == metav1.ConditionFalse {
		value = 1
	}
	AbnormalInstance.WithLabelValues(labels...).Set(value)
}

// ObserveConditions records the gauge for every condition type tracked by this
// package. This is the convenience entry point from reconcile tasks that want
// to refresh the full picture in one call.
func ObserveConditions(obj client.Object, conds []metav1.Condition) {
	for _, condType := range trackedConditions {
		ObserveCondition(obj, conds, condType)
	}
}

// ClearInstanceConditionMetrics removes every tracked-condition series for
// the given instance.
//
// Called from TaskInstanceFinalizerDel after the finalizer is removed, so
// every component that uses the standard finalize task is covered without
// per-builder wiring. Component builders short-circuit the deletion path
// with task.IfBreak around CondClusterIsDeleting / CondObjectIsDeleting, so
// the normal TaskInstanceConditionSynced / TaskInstanceConditionReady tasks
// (where ObserveCondition lives) never run during finalization; without
// this explicit cleanup, the gauge series would stay present at its last
// value forever, triggering false-positive `metric == 1 for: <duration>`
// alerts on a non-existent instance and growing label cardinality across
// each cluster lifecycle.
//
// DeletePartialMatch is used (rather than DeleteLabelValues) because the
// 'suspended' label value may have changed over the instance lifetime; a
// partial match on the five stable dimensions sweeps all series regardless
// of the current suspended value.
func ClearInstanceConditionMetrics(obj client.Object) {
	base := instanceMetricBaseLabels(obj)
	baseLabels := prometheus.Labels{
		labelNamespace: base[0],
		"cluster":      base[1],
		labelComponent: base[2],
		"group":        base[3],
		labelInstance:  base[4],
	}
	for _, condType := range trackedConditions {
		m := make(prometheus.Labels, len(baseLabels)+1)
		for k, v := range baseLabels {
			m[k] = v
		}
		m["condition"] = condType
		AbnormalInstance.DeletePartialMatch(m)
	}
}

// ClearInstanceConditionMetricsByKey removes every tracked-condition series
// matching (namespace, component, instance) regardless of cluster / group
// labels. Use this from reconcile paths where the object has already been
// deleted from the API server and the business labels are no longer readable;
// passing component (which is a constant for a given reconcile scope) avoids
// sweeping up a different kind of instance that happens to share the same
// namespace and name (e.g. a PD and a TiDB both called "foo"). The partial
// match still covers series written under an earlier cluster / group label
// value if those labels ever shifted.
func ClearInstanceConditionMetricsByKey(namespace, component, instance string) {
	AbnormalInstance.DeletePartialMatch(prometheus.Labels{
		labelNamespace: namespace,
		labelComponent: component,
		labelInstance:  instance,
	})
}
