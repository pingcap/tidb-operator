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
// limitations under the License.package spec

package label

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// ClusterLabelKey is cluster label key
	ClusterLabelKey string = "cluster.pingcap.com/tidbCluster"
	// NamespaceLabelKey is cluster label key
	NamespaceLabelKey string = "cluster.pingcap.com/namespace"
	// AppLabelKey is app label key
	AppLabelKey string = "cluster.pingcap.com/app"
	// OwnerLabelKey is owner label key
	OwnerLabelKey string = "cluster.pingcap.com/owner"
	// ClusterIDLabelKey is cluster id label key
	ClusterIDLabelKey string = "cluster.pingcap.com/clusterId"
	// StoreIDLabelKey is store id label key
	StoreIDLabelKey string = "cluster.pingcap.com/storeId"
	// MemberIDLabelKey is member id label key
	MemberIDLabelKey string = "cluster.pingcap.com/memberId"
	// BackupNameLabelKey is TidbClusterBackup name label key
	BackupNameLabelKey = "cluster.pingcap.com/tidbClusterBackup"
	// BackupTypeLabelKey is bakcup type label key
	BackupTypeLabelKey = "cluster.pingcap.com/tidbClusterBackupType"
	// AnnBackupDirlKey is backup dir label key
	AnnBackupDirlKey = "cluster.pingcap.com/tidbClusterBackupDir"

	// AnnPodNameKey is podName annotations key
	AnnPodNameKey string = "volume.pingcap.com/podName"

	// AnnShouldDeletedPD is the PD name that should be deleted from PD
	AnnShouldDeletedPD string = "cluster.pingcap.com/should-deleted-pdname"

	// AnnShouldDeletedTiKV is the TiKV name that should be deleted from PD
	AnnShouldDeletedTiKV string = "cluster.pingcap.com/should-deleted-tikvname"

	// AnnPaused is the annotation that the object is paused
	AnnPaused string = "cluster.pingcap.com/paused"

	// AnnGracefulUpgradeTiKVStartTime is the time when to evict leaders from the store
	AnnGracefulUpgradeTiKVStartTime string = "cluster.pingcap.com/graceful-upgrade-tikv-time"

	// PDLabelVal is PD label value
	PDLabelVal string = "pd"
	// TiDBLabelVal is TiDB label value
	TiDBLabelVal string = "tidb"
	// TiKVLabelVal is TiKV label value
	TiKVLabelVal string = "tikv"
	// ClusterLabelVal is cluster label value
	ClusterLabelVal string = "tidbCluster"

	// FullBakcupTypeLabelValue is full bakcup type label value
	FullBakcupTypeLabelValue string = "full"
	// PhysicalBackupTypeLabelValue is physical backup type label value
	PhysicalBackupTypeLabelValue string = "physical"
)

// Label is the label field in metadata
type Label map[string]string

// New initialize a new Label
func New() Label {
	return Label{OwnerLabelKey: ClusterLabelVal}
}

// ClusterListOptions returns a cluster ListOptions filter
func ClusterListOptions(clusterName string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			New().Cluster(clusterName).Labels(),
		).String(),
	}
}

// Cluster adds cluster kv pair to label
func (l Label) Cluster(name string) Label {
	l[ClusterLabelKey] = name
	return l
}

// Namespace adds namespace kv pair to label
func (l Label) Namespace(name string) Label {
	l[NamespaceLabelKey] = name
	return l
}

// App adds app kv pair to label
func (l Label) App(name string) Label {
	l[AppLabelKey] = name
	return l
}

// AppType returns app type
func (l Label) AppType() string {
	return l[AppLabelKey]
}

// PD assigns pd to app key in label
func (l Label) PD() Label {
	l.App(PDLabelVal)
	return l
}

// IsPD returns whether label is a PD
func (l Label) IsPD() bool {
	return l[AppLabelKey] == PDLabelVal
}

// TiDB assigns tidb to app key in label
func (l Label) TiDB() Label {
	l.App(TiDBLabelVal)
	return l
}

// TiKV assigns tikv to app key in label
func (l Label) TiKV() Label {
	l.App(TiKVLabelVal)
	return l
}

// IsTiKV returns whether label is a TiKV
func (l Label) IsTiKV() bool {
	return l[AppLabelKey] == TiKVLabelVal
}

// IsTiDB returns whether label is a TiDB
func (l Label) IsTiDB() bool {
	return l[AppLabelKey] == TiDBLabelVal
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
