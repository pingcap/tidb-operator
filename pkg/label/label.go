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
	// It should always be tidb-cluster in our case.
	NameLabelKey string = "app.kubernetes.io/name"
	// InstanceLabelKey is Kubernetes recommended label key, it represents a unique name identifying the instance of an application
	// It's set by helm when installing a release
	InstanceLabelKey string = "app.kubernetes.io/instance"

	// NamespaceLabelKey is label key used in PV for easy querying
	NamespaceLabelKey string = "app.kubernetes.io/namespace"

	// ClusterIDLabelKey is cluster id label key
	ClusterIDLabelKey string = "tidb.pingcap.com/cluster-id"
	// StoreIDLabelKey is store id label key
	StoreIDLabelKey string = "tidb.pingcap.com/store-id"
	// MemberIDLabelKey is member id label key
	MemberIDLabelKey string = "tidb.pingcap.com/member-id"
	// AnnFailTiDBScheduler is for injecting a failure into the TiDB custom scheduler
	// A pod with this annotation will produce an error when scheduled.
	AnnFailTiDBScheduler string = "tidb.pingcap.com/fail-scheduler"
	// AnnPodNameKey is pod name annotation key used in PV/PVC for synchronizing tidb cluster meta info
	AnnPodNameKey string = "tidb.pingcap.com/pod-name"
	// AnnPVCDeferDeleting is pvc defer deletion annotation key used in PVC for defer deleting PVC
	AnnPVCDeferDeleting = "tidb.pingcap.com/pvc-defer-deleting"
	// AnnPVCPodScheduling is pod scheduling annotation key, it represents whether the pod is scheduling
	AnnPVCPodScheduling = "tidb.pingcap.com/pod-scheduling"
	// AnnTiDBPartition is pod annotation which TiDB pod chould upgrade to
	AnnTiDBPartition string = "tidb.pingcap.com/tidb-partition"
	// AnnForceUpgradeKey is tc annotation key to indicate whether force upgrade should be done
	AnnForceUpgradeKey = "tidb.pingcap.com/force-upgrade"
<<<<<<< HEAD
=======
	// AnnPDDeferDeleting is pd pod annotation key  in pod for defer for deleting pod
	AnnPDDeferDeleting = "tidb.pingcap.com/pd-defer-deleting"
	// AnnSysctlInit is pod annotation key to indicate whether configuring sysctls with init container
	AnnSysctlInit = "tidb.pingcap.com/sysctl-init"
	// AnnEvictLeaderBeginTime is pod annotation key to indicate the begin time for evicting region leader
	AnnEvictLeaderBeginTime = "tidb.pingcap.com/evictLeaderBeginTime"
	// AnnStsSyncTimestamp is sts annotation key to indicate the last timestamp the operator sync the sts
	AnnStsLastSyncTimestamp = "tidb.pingcap.com/sync-timestamp"
>>>>>>> 18703b9... Remove the PodRestarter controller and `tidb.pingcap.com/pod-defer-deleting` annotation (#3296)

	// AnnForceUpgradeVal is tc annotation value to indicate whether force upgrade should be done
	AnnForceUpgradeVal = "true"

	// PDLabelVal is PD label value
	PDLabelVal string = "pd"
	// TiDBLabelVal is TiDB label value
	TiDBLabelVal string = "tidb"
	// TiKVLabelVal is TiKV label value
	TiKVLabelVal string = "tikv"
)

// Label is the label field in metadata
type Label map[string]string

// New initialize a new Label
func New() Label {
	return Label{
		NameLabelKey:      "tidb-cluster",
		ManagedByLabelKey: "tidb-operator",
	}
}

// Instance adds instance kv pair to label
func (l Label) Instance(name string) Label {
	l[InstanceLabelKey] = name
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

// ComponentType returns component type
func (l Label) ComponentType() string {
	return l[ComponentLabelKey]
}

// PD assigns pd to component key in label
func (l Label) PD() Label {
	l.Component(PDLabelVal)
	return l
}

// IsPD returns whether label is a PD
func (l Label) IsPD() bool {
	return l[ComponentLabelKey] == PDLabelVal
}

// TiDB assigns tidb to component key in label
func (l Label) TiDB() Label {
	l.Component(TiDBLabelVal)
	return l
}

// TiKV assigns tikv to component key in label
func (l Label) TiKV() Label {
	l.Component(TiKVLabelVal)
	return l
}

// IsTiKV returns whether label is a TiKV
func (l Label) IsTiKV() bool {
	return l[ComponentLabelKey] == TiKVLabelVal
}

// IsTiDB returns whether label is a TiDB
func (l Label) IsTiDB() bool {
	return l[ComponentLabelKey] == TiDBLabelVal
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

// String converts label to a string
func (l Label) String() string {
	var arr []string

	for k, v := range l {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(arr, ",")
}
