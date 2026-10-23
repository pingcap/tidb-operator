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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// InstanceAbnormalMetricLabels is the canonical label order for the
// per-instance abnormal-condition gauge. Keep in sync with WithLabelValues /
// DeleteLabelValues callers.
//
// The 'suspended' label carries the Suspended-condition state at the time the
// gauge is written ("true" / "false"). Alert rules can use {suspended!="true"}
// to suppress noise on intentionally paused clusters without needing a
// separate metric or a PromQL join.
//
// NOTE: adding 'suspended' changes the series identity. Existing time-series
// (without this label) disappear on operator upgrade; alert for: windows reset.
// This is acceptable because the window is short (30 m) and instances that
// were in a legitimate abnormal state will re-fire within one window.
var InstanceAbnormalMetricLabels = []string{"namespace", "cluster", "component", "group", "instance", "condition", "suspended"}

var (

	// ControllerPanic is a counter to record the number of panics in the controller.
	ControllerPanic = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tidb_operator",
			Subsystem: "controller",
			Name:      "panic_total",
			Help:      "The total number of panics in the controller",
		}, []string{},
	)

	// AbnormalInstance is 1 when the named condition on the instance is False
	// (abnormal), 0 otherwise. The series stays present while the operator
	// manages the instance and is removed only when the instance is finalized.
	//
	// Use `metric == 1` together with PromQL `for: <duration>` to alert on
	// instances stuck in an abnormal state, e.g. a rolling restart that cannot
	// converge or a pod that is up but cannot serve.
	AbnormalInstance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "tidb_operator",
			Name:      "abnormal_instance",
			Help: "1 when the named condition on the instance is False, 0 otherwise. " +
				"Use `metric == 1` with PromQL `for: <duration>` to alert on stuck state.",
		}, InstanceAbnormalMetricLabels,
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(ControllerPanic)
	metrics.Registry.MustRegister(AbnormalInstance)
}
