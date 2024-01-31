package fedmetrics

import "github.com/prometheus/client_golang/prometheus"

const (
	LabelNamespace = "namespace"
	LabelTC        = "tc"
	LabelStatus    = "status"
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
		Name:      "size_gb",
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
