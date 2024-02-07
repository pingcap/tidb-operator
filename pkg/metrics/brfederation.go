// Copyright 2023 PingCAP, Inc.
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

package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	LabelTC     = "tc"
	LabelStatus = "status"
)

var (
	FedVolumeBackupStatusCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fed",
		Subsystem: "volume_backup",
		Name:      "status",
	}, []string{LabelNamespace, LabelTC, LabelStatus})
	FedVolumeBackupTotalTimeCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fed",
		Subsystem: "volume_backup",
		Name:      "total_time_sec",
	}, []string{LabelNamespace, LabelTC})
	FedVolumeBackupTotalSizeCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fed",
		Subsystem: "volume_backup",
		Name:      "total_size_gb",
	}, []string{LabelNamespace, LabelTC})
	FedVolumeBackupCleanupStatusCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fed",
		Subsystem: "volume_backup_cleanup",
		Name:      "status",
	}, []string{LabelNamespace, LabelTC, LabelStatus})
	FedVolumeBackupCleanupTotalTimeCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fed",
		Subsystem: "volume_backup_cleanup",
		Name:      "total_time_sec",
	}, []string{LabelNamespace, LabelTC})
)

func init() {
	prometheus.MustRegister(
		FedVolumeBackupStatusCounterVec,
		FedVolumeBackupTotalTimeCounterVec,
		FedVolumeBackupTotalSizeCounterVec,
		FedVolumeBackupCleanupStatusCounterVec,
		FedVolumeBackupCleanupTotalTimeCounterVec,
	)
}
