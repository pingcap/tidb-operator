// Copyright 2016 PingCAP, Inc.
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

package server

import "github.com/prometheus/client_golang/prometheus"

var (
	txnCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "txn",
			Name:      "txns_count",
			Help:      "Counter of txns.",
		}, []string{"result"})

	txnDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pd",
			Subsystem: "txn",
			Name:      "handle_txns_duration_seconds",
			Help:      "Bucketed histogram of processing time (s) of handled txns.",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 13),
		}, []string{"result"})

	operatorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "schedule",
			Name:      "operators_count",
			Help:      "Counter of schedule operators.",
		}, []string{"type", "event"})

	operatorDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pd",
			Subsystem: "schedule",
			Name:      "finish_operators_duration_seconds",
			Help:      "Bucketed histogram of processing time (s) of finished operator.",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 16),
		}, []string{"type"})

	clusterStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "cluster",
			Name:      "status",
			Help:      "Status of the cluster.",
		}, []string{"type", "namespace"})

	healthStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "cluster",
			Name:      "health_status",
			Help:      "Status of the cluster.",
		}, []string{"name"})

	regionStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "regions",
			Name:      "status",
			Help:      "Status of the regions.",
		}, []string{"type"})

	regionLabelLevelGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "regions",
			Name:      "label_level",
			Help:      "Number of regions in the different label level.",
		}, []string{"type"})

	timeJumpBackCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "monitor",
			Name:      "time_jump_back_total",
			Help:      "Counter of system time jumps backward.",
		})

	schedulerStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "scheduler",
			Name:      "status",
			Help:      "Status of the scheduler.",
		}, []string{"kind", "type"})

	regionHeartbeatCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "scheduler",
			Name:      "region_heartbeat",
			Help:      "Counter of region hearbeat.",
		}, []string{"store", "type", "status"})

	regionHeartbeatLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pd",
			Subsystem: "scheduler",
			Name:      "region_heartbeat_latency_seconds",
			Help:      "Bucketed histogram of latency (s) of receiving heartbeat.",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{"store"})

	storeStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "scheduler",
			Name:      "store_status",
			Help:      "Store status for schedule",
		}, []string{"namespace", "store", "type"})

	hotSpotStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "hotspot",
			Name:      "status",
			Help:      "Status of the hotspot.",
		}, []string{"store", "type"})

	tsoCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pd",
			Subsystem: "server",
			Name:      "tso",
			Help:      "Counter of tso events",
		}, []string{"type"})

	metadataGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pd",
			Subsystem: "cluster",
			Name:      "metadata",
			Help:      "Record critical metadata.",
		}, []string{"type"})

	patrolCheckRegionsHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "pd",
			Subsystem: "patrol",
			Name:      "checks_regions",
			Help:      "Bucketed histogram of time spend(s) of patrol checks region.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
		})
)

func init() {
	prometheus.MustRegister(txnCounter)
	prometheus.MustRegister(txnDuration)
	prometheus.MustRegister(operatorCounter)
	prometheus.MustRegister(operatorDuration)
	prometheus.MustRegister(clusterStatusGauge)
	prometheus.MustRegister(timeJumpBackCounter)
	prometheus.MustRegister(schedulerStatusGauge)
	prometheus.MustRegister(regionHeartbeatCounter)
	prometheus.MustRegister(regionHeartbeatLatency)
	prometheus.MustRegister(hotSpotStatusGauge)
	prometheus.MustRegister(tsoCounter)
	prometheus.MustRegister(storeStatusGauge)
	prometheus.MustRegister(regionStatusGauge)
	prometheus.MustRegister(regionLabelLevelGauge)
	prometheus.MustRegister(metadataGauge)
	prometheus.MustRegister(patrolCheckRegionsHistogram)
}
