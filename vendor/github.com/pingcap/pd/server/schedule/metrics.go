// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package schedule

import "github.com/prometheus/client_golang/prometheus"

var (
	checkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "checker",
			Name:      "event_count",
			Help:      "Counter of checker events.",
		}, []string{"type", "name"})

	operatorStepDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pd",
			Subsystem: "schedule",
			Name:      "finish_operator_steps_duration_seconds",
			Help:      "Bucketed histogram of processing time (s) of finished operator step.",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 16),
		}, []string{"type"})

	hotCacheStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "hotcache",
			Name:      "status",
			Help:      "Status of the hotspot.",
		}, []string{"name", "type"})

	filterCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "schedule",
			Name:      "filter",
			Help:      "Counter of the filter",
		}, []string{"action", "store", "type"})
)

func init() {
	prometheus.MustRegister(checkerCounter)
	prometheus.MustRegister(operatorStepDuration)
	prometheus.MustRegister(hotCacheStatusGauge)
	prometheus.MustRegister(filterCounter)
}
