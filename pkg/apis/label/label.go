// Copyright 2017 PingCAP, Inc.
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

package label

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// The following labels are recommended by kubernetes https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/

	// ManagedByLabelKey is Kubernetes recommended label key, it represents the tool being used to manage the operation of an application
	// For resources managed by TiDB Operator, its value is always tidb-operator
	ManagedByLabelKey string = "app.kubernetes.io/managed-by"
	// ComponentLabelKey is Kubernetes recommended label key, it represents the component within the architecture
	ComponentLabelKey string = "app.kubernetes.io/component"
	// NameLabelKey is Kubernetes recommended label key, it represents the name of the application
	NameLabelKey string = "app.kubernetes.io/name"
	// InstanceLabelKey is Kubernetes recommended label key, it represents a unique name identifying the instance of an application
	// It's set by helm when installing a release
	InstanceLabelKey string = "app.kubernetes.io/instance"

	// NamespaceLabelKey is label key used in PV for easy querying
	NamespaceLabelKey string = "app.kubernetes.io/namespace"
	// UsedByLabelKey indicate where it is used. for example, tidb has two services,
	// one for internal component access and the other for end-user
	UsedByLabelKey string = "app.kubernetes.io/used-by"
	// ClusterIDLabelKey is cluster id label key
	ClusterIDLabelKey string = "tidb.pingcap.com/cluster-id"
	// StoreIDLabelKey is store id label key
	StoreIDLabelKey string = "tidb.pingcap.com/store-id"
	// MemberIDLabelKey is member id label key
	MemberIDLabelKey string = "tidb.pingcap.com/member-id"

	// InitLabelKey is the key for TiDB initializer
	InitLabelKey string = "tidb.pingcap.com/initializer"

	// BackupScheduleLabelKey is backup schedule key
	BackupScheduleLabelKey string = "tidb.pingcap.com/backup-schedule"

	// BackupLabelKey is backup key
	BackupLabelKey string = "tidb.pingcap.com/backup"

	// RestoreLabelKey is restore key
	RestoreLabelKey string = "tidb.pingcap.com/restore"

	// BackupProtectionFinalizer is the name of finalizer on backups
	BackupProtectionFinalizer string = "tidb.pingcap.com/backup-protection"

	// AutoScalingGroupLabelKey describes the autoscaling group of the TiDB
	AutoScalingGroupLabelKey = "tidb.pingcap.com/autoscaling-group"
	// AutoInstanceLabelKey is label key used in autoscaling, it represents the autoscaler name
	AutoInstanceLabelKey string = "tidb.pingcap.com/auto-instance"
	// AutoComponentLabelKey is label key used in autoscaling, it represents which component is auto scaled
	AutoComponentLabelKey string = "tidb.pingcap.com/auto-component"
	// BaseTCLabelKey is label key used for heterogeneous clusters to refer to its base TidbCluster
	BaseTCLabelKey string = "tidb.pingcap.com/base-tc"

	// AnnHATopologyKey defines the High availability topology key
	AnnHATopologyKey = "pingcap.com/ha-topology-key"

	// AnnFailTiDBScheduler is for injecting a failure into the TiDB custom scheduler
	// A pod with this annotation will produce an error when scheduled.
	AnnFailTiDBScheduler string = "tidb.pingcap.com/fail-scheduler"
	// AnnPodNameKey is pod name annotation key used in PV/PVC for synchronizing tidb cluster meta info
	AnnPodNameKey string = "tidb.pingcap.com/pod-name"
	// AnnPVCDeferDeleting is pvc defer deletion annotation key used in PVC for defer deleting PVC
	AnnPVCDeferDeleting = "tidb.pingcap.com/pvc-defer-deleting"
	// AnnPVCPodScheduling is pod scheduling annotation key, it represents whether the pod is scheduling
	AnnPVCPodScheduling = "tidb.pingcap.com/pod-scheduling"
	// AnnTiDBPartition is pod annotation which TiDB pod should upgrade to
	AnnTiDBPartition string = "tidb.pingcap.com/tidb-partition"
	// AnnTiKVPartition is pod annotation which TiKV pod should upgrade to
	AnnTiKVPartition string = "tidb.pingcap.com/tikv-partition"
	// AnnForceUpgradeKey is tc annotation key to indicate whether force upgrade should be done
	AnnForceUpgradeKey = "tidb.pingcap.com/force-upgrade"
	// AnnPDDeferDeleting is pd pod annotation key  in pod for defer for deleting pod
	AnnPDDeferDeleting = "tidb.pingcap.com/pd-defer-deleting"
	// AnnSysctlInit is pod annotation key to indicate whether configuring sysctls with init container
	AnnSysctlInit = "tidb.pingcap.com/sysctl-init"
	// AnnEvictLeaderBeginTime is pod annotation key to indicate the begin time for evicting region leader
	AnnEvictLeaderBeginTime = "tidb.pingcap.com/evictLeaderBeginTime"
	// AnnTiCDCGracefulShutdownBeginTime is pod annotation key to indicate the begin time for graceful shutdown TiCDC
	AnnTiCDCGracefulShutdownBeginTime = "tidb.pingcap.com/ticdc-graceful-shutdown-begin-time"
	// AnnStsLastSyncTimestamp is sts annotation key to indicate the last timestamp the operator sync the sts
	AnnStsLastSyncTimestamp = "tidb.pingcap.com/sync-timestamp"

	// AnnPVCScaleInTime is pvc scaled in time key used in PVC for e2e test only
	AnnPVCScaleInTime = "tidb.pingcap.com/scale-in-time"

	// AnnForceUpgradeVal is tc annotation value to indicate whether force upgrade should be done
	AnnForceUpgradeVal = "true"
	// AnnSysctlInitVal is pod annotation value to indicate whether configuring sysctls with init container
	AnnSysctlInitVal = "true"

	// AnnPDDeleteSlots is annotation key of pd delete slots.
	AnnPDDeleteSlots = "pd.tidb.pingcap.com/delete-slots"
	// AnnTiDBDeleteSlots is annotation key of tidb delete slots.
	AnnTiDBDeleteSlots = "tidb.tidb.pingcap.com/delete-slots"
	// AnnTiKVDeleteSlots is annotation key of tikv delete slots.
	AnnTiKVDeleteSlots = "tikv.tidb.pingcap.com/delete-slots"
	// AnnTiFlashDeleteSlots is annotation key of tiflash delete slots.
	AnnTiFlashDeleteSlots = "tiflash.tidb.pingcap.com/delete-slots"
	// AnnDMMasterDeleteSlots is annotation key of dm-master delete slots.
	AnnDMMasterDeleteSlots = "dm-master.tidb.pingcap.com/delete-slots"
	// AnnDMWorkerDeleteSlots is annotation key of dm-worker delete slots.
	AnnDMWorkerDeleteSlots = "dm-worker.tidb.pingcap.com/delete-slots"

	// AnnSkipTLSWhenConnectTiDB describes whether skip TLS when connecting to TiDB Server
	AnnSkipTLSWhenConnectTiDB = "tidb.tidb.pingcap.com/skip-tls-when-connect-tidb"

	// AnnBackupCloudSnapKey is the annotation key for backup metadata based cloud snapshot
	AnnBackupCloudSnapKey string = "tidb.pingcap.com/backup-cloud-snapshot"

	// AnnTiKVVolumesReadyKey is the annotation key to indicate whether the TiKV volumes are ready.
	// TiKV member manager will wait until the TiKV volumes are ready before starting the TiKV pod
	// when TiDB cluster is restored from volume snapshot based backup.
	AnnTiKVVolumesReadyKey = "tidb.pingcap.com/tikv-volumes-ready"

	// PDLabelVal is PD label value
	PDLabelVal string = "pd"
	// TiDBLabelVal is TiDB label value
	TiDBLabelVal string = "tidb"
	// TiKVLabelVal is TiKV label value
	TiKVLabelVal string = "tikv"
	// TiFlashLabelVal is TiFlash label value
	TiFlashLabelVal string = "tiflash"
	// TiCDCLabelVal is TiCDC label value
	TiCDCLabelVal string = "ticdc"
	// TiProxyLabelVal is TiProxy label value
	TiProxyLabelVal string = "tiproxy"
	// PumpLabelVal is Pump label value
	PumpLabelVal string = "pump"
	// DiscoveryLabelVal is Discovery label value
	DiscoveryLabelVal string = "discovery"
	// TiDBMonitorVal is Monitor label value
	TiDBMonitorVal string = "monitor"

	// CleanJobLabelVal is clean job label value
	CleanJobLabelVal string = "clean"
	// RestoreJobLabelVal is restore job label value
	RestoreJobLabelVal string = "restore"
	// BackupJobLabelVal is backup job label value
	BackupJobLabelVal string = "backup"
	// BackupScheduleJobLabelVal is backup schedule job label value
	BackupScheduleJobLabelVal string = "backup-schedule"
	// InitJobLabelVal is TiDB initializer job label value
	InitJobLabelVal string = "initializer"
	// TiDBOperator is ManagedByLabelKey label value
	TiDBOperator string = "tidb-operator"

	// DMMasterLabelVal is dm-master label value
	DMMasterLabelVal string = "dm-master"
	// DMWorkerLabelVal is dm-worker label value
	DMWorkerLabelVal string = "dm-worker"

	// NGMonitorLabelVal is ng-monitoring label value
	NGMonitorLabelVal string = "ng-monitoring"

	// PrometheusVal is Prometheus label value
	PrometheusVal string = "prometheus"

	// GrafanaVal is Grafana label value
	GrafanaVal string = "grafana"

	// ApplicationLabelKey is App label key
	ApplicationLabelKey string = "app.kubernetes.io/app"
)

// Label is the label field in metadata
type Label map[string]string

func NewOperatorManaged() Label {
	return Label{
		ManagedByLabelKey: TiDBOperator,
	}
}

// New initialize a new Label for components of tidb cluster
func New() Label {
	return Label{
		NameLabelKey:      "tidb-cluster",
		ManagedByLabelKey: TiDBOperator,
	}
}

// NewDM initialize a new Label for components of dm cluster
func NewDM() Label {
	return Label{
		NameLabelKey:      "dm-cluster",
		ManagedByLabelKey: TiDBOperator,
	}
}

// NewInitializer initialize a new Label for Jobs of TiDB initializer
func NewInitializer() Label {
	return Label{
		ComponentLabelKey: InitJobLabelVal,
		ManagedByLabelKey: TiDBOperator,
	}
}

// NewBackup initialize a new Label for Jobs of bakcup
func NewBackup() Label {
	return Label{
		NameLabelKey:      BackupJobLabelVal,
		ManagedByLabelKey: "backup-operator",
	}
}

// NewRestore initialize a new Label for Jobs of restore
func NewRestore() Label {
	return Label{
		NameLabelKey:      RestoreJobLabelVal,
		ManagedByLabelKey: "restore-operator",
	}
}

// NewBackupSchedule initialize a new Label for backups of bakcup schedule
func NewBackupSchedule() Label {
	return Label{
		NameLabelKey:      BackupScheduleJobLabelVal,
		ManagedByLabelKey: "backup-schedule-operator",
	}
}

func NewMonitor() Label {
	return Label{
		// NameLabelKey is used to be compatible with helm monitor
		NameLabelKey:      "tidb-cluster",
		ManagedByLabelKey: TiDBOperator,
	}
}

func NewTiDBNGMonitoring() Label {
	return Label{
		NameLabelKey:      "tidb-ng-monitoring",
		ManagedByLabelKey: TiDBOperator,
	}
}

func NewGroup() Label {
	return Label{
		NameLabelKey:      "tidb-cluster-group",
		ManagedByLabelKey: TiDBOperator,
	}
}

// Instance adds instance kv pair to label
func (l Label) Instance(name string) Label {
	l[InstanceLabelKey] = name
	return l
}

// UserBy adds use-by kv pair to label
func (l Label) UsedBy(name string) Label {
	l[UsedByLabelKey] = name
	return l
}

// UsedByPeer adds used-by=peer label
func (l Label) UsedByPeer() Label {
	l[UsedByLabelKey] = "peer"
	return l
}

// UsedByEndUser adds use-by=end-user label
func (l Label) UsedByEndUser() Label {
	l[UsedByLabelKey] = "end-user"
	return l
}

// Namespace adds namespace kv pair to label
func (l Label) Namespace(name string) Label {
	l[NamespaceLabelKey] = name
	return l
}

// Component adds component kv pair to label
func (l Label) Component(name string) Label {
	l[ComponentLabelKey] = name
	return l
}

// Application adds application kv pair to label
func (l Label) Application(name string) Label {
	l[ApplicationLabelKey] = name
	return l
}

// ComponentType returns component type
func (l Label) ComponentType() string {
	return l[ComponentLabelKey]
}

// Initializer assigns specific value to initializer key in label
func (l Label) Initializer(val string) Label {
	l[InitLabelKey] = val
	return l
}

// CleanJob assigns clean to component key in label
func (l Label) CleanJob() Label {
	return l.Component(CleanJobLabelVal)
}

// BackupJob assigns backup to component key in label
func (l Label) BackupJob() Label {
	return l.Component(BackupJobLabelVal)
}

// RestoreJob assigns restore to component key in label
func (l Label) RestoreJob() Label {
	return l.Component(RestoreJobLabelVal)
}

// Backup assigns specific value to backup key in label
func (l Label) Backup(val string) Label {
	l[BackupLabelKey] = val
	return l
}

// BackupSchedule assigns specific value to backup schedule key in label
func (l Label) BackupSchedule(val string) Label {
	l[BackupScheduleLabelKey] = val
	return l
}

// Restore assigns specific value to restore key in label
func (l Label) Restore(val string) Label {
	l[RestoreLabelKey] = val
	return l
}

// PD assigns pd to component key in label
func (l Label) PD() Label {
	return l.Component(PDLabelVal)
}

// IsPD returns whether label is a PD component
func (l Label) IsPD() bool {
	return l[ComponentLabelKey] == PDLabelVal
}

// TiProxy assigns tiproxy to component key in label
func (l Label) TiProxy() Label {
	return l.Component(TiProxyLabelVal)
}

// IsTiProxy returns whether label is a TiProxy component
func (l Label) IsTiProxy() bool {
	return l[ComponentLabelKey] == TiProxyLabelVal
}

// Pump assigns pump to component key in label
func (l Label) Pump() Label {
	return l.Component(PumpLabelVal)
}

// IsPump returns whether label is a Pump component
func (l Label) IsPump() bool {
	return l[ComponentLabelKey] == PumpLabelVal
}

// DMMaster assigns dm-master to component key in label
func (l Label) DMMaster() Label {
	return l.Component(DMMasterLabelVal)
}

// IsDMMaster returns whether label is a DMMaster component
func (l Label) IsDMMaster() bool {
	return l[ComponentLabelKey] == DMMasterLabelVal
}

// DMWorker assigns dm-worker to component key in label
func (l Label) DMWorker() Label {
	return l.Component(DMWorkerLabelVal)
}

// IsDMWorker returns whether label is a DMWorker component
func (l Label) IsDMWorker() bool {
	return l[ComponentLabelKey] == DMWorkerLabelVal
}

// Monitor assigns monitor to component key in label
func (l Label) Monitor() Label {
	return l.Component(TiDBMonitorVal)
}

// Prometheus assigns prometheus to app key in the label
func (l Label) Prometheus() Label {
	return l.Application(PrometheusVal)
}

// Grafana assigns grafana to app key in the label
func (l Label) Grafana() Label {
	return l.Application(GrafanaVal)
}

// NGMonitoring assigns ng monitoring to component key in label
func (l Label) NGMonitoring() Label {
	return l.Component(NGMonitorLabelVal)
}

// IsNGMonitoring returns whether label is a NGMonitoring component
func (l Label) IsNGMonitoring() bool {
	return l[ComponentLabelKey] == NGMonitorLabelVal
}

// IsMonitor returns whether label is a Monitor component
func (l Label) IsMonitor() bool {
	return l[ComponentLabelKey] == TiDBMonitorVal
}

// Discovery assigns discovery to component key in label
func (l Label) Discovery() Label {
	return l.Component(DiscoveryLabelVal)
}

// TiDB assigns tidb to component key in label
func (l Label) TiDB() Label {
	return l.Component(TiDBLabelVal)
}

// IsTiDB returns whether label is a TiDB component
func (l Label) IsTiDB() bool {
	return l[ComponentLabelKey] == TiDBLabelVal
}

// TiKV assigns tikv to component key in label
func (l Label) TiKV() Label {
	return l.Component(TiKVLabelVal)
}

// IsTiKV returns whether label is a TiKV component
func (l Label) IsTiKV() bool {
	return l[ComponentLabelKey] == TiKVLabelVal
}

// TiFlash assigns tiflash to component key in label
func (l Label) TiFlash() Label {
	return l.Component(TiFlashLabelVal)
}

// IsTiFlash returns whether label is a TiFlash component
func (l Label) IsTiFlash() bool {
	return l[ComponentLabelKey] == TiFlashLabelVal
}

// TiCDC assigns ticdc to component key in label
func (l Label) TiCDC() Label {
	return l.Component(TiCDCLabelVal)
}

// IsTiCDC returns whether label is a TiCDC component
func (l Label) IsTiCDC() bool {
	return l[ComponentLabelKey] == TiCDCLabelVal
}

// Selector gets labels.Selector from label
func (l Label) Selector() (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(l.LabelSelector())
}

// LabelSelector gets LabelSelector from label
func (l Label) LabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: l}
}

// Labels converts label to map[string]string
func (l Label) Labels() map[string]string {
	return l
}

// Copy copy the value of label to avoid pointer copy
func (l Label) Copy() Label {
	copyLabel := make(Label)
	for k, v := range l {
		copyLabel[k] = v
	}
	return copyLabel
}

// String converts label to a string
func (l Label) String() string {
	var arr []string

	for k, v := range l {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(arr, ",")
}

// IsManagedByTiDBOperator returns whether label is a Managed by tidb-operator
func (l Label) IsManagedByTiDBOperator() bool {
	return l[ManagedByLabelKey] == TiDBOperator
}

// IsTidbClusterPod returns whether it is a TidbCluster-controlled pod
func (l Label) IsTidbClusterPod() bool {
	return l[NameLabelKey] == "tidb-cluster"
}
