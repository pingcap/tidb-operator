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

package tidbcluster

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/defaulting"
	v1alpha1validation "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// ControlInterface implements the control logic for updating TidbClusters and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateTidbCluster implements the control logic for StatefulSet creation, update, and deletion
	UpdateTidbCluster(*v1alpha1.TidbCluster) error
}

// NewDefaultTidbClusterControl returns a new instance of the default implementation TidbClusterControlInterface that
// implements the documented semantics for TidbClusters.
func NewDefaultTidbClusterControl(
	tcControl controller.TidbClusterControlInterface,
	pdMemberManager manager.Manager,
	tikvMemberManager manager.Manager,
	tidbMemberManager manager.Manager,
	reclaimPolicyManager manager.Manager,
	metaManager manager.Manager,
	orphanPodsCleaner member.OrphanPodsCleaner,
	pvcCleaner member.PVCCleanerInterface,
	pumpMemberManager manager.Manager,
	tiflashMemberManager manager.Manager,
	discoveryManager member.TidbDiscoveryManager,
	podRestarter member.PodRestarter,
	recorder record.EventRecorder) ControlInterface {
	return &defaultTidbClusterControl{
		tcControl,
		pdMemberManager,
		tikvMemberManager,
		tidbMemberManager,
		reclaimPolicyManager,
		metaManager,
		orphanPodsCleaner,
		pvcCleaner,
		pumpMemberManager,
		tiflashMemberManager,
		discoveryManager,
		podRestarter,
		recorder,
	}
}

type defaultTidbClusterControl struct {
	tcControl            controller.TidbClusterControlInterface
	pdMemberManager      manager.Manager
	tikvMemberManager    manager.Manager
	tidbMemberManager    manager.Manager
	reclaimPolicyManager manager.Manager
	metaManager          manager.Manager
	orphanPodsCleaner    member.OrphanPodsCleaner
	pvcCleaner           member.PVCCleanerInterface
	pumpMemberManager    manager.Manager
	tiflashMemberManager manager.Manager
	discoveryManager     member.TidbDiscoveryManager
	podRestarter         member.PodRestarter
	recorder             record.EventRecorder
}

// UpdateStatefulSet executes the core logic loop for a tidbcluster.
func (tcc *defaultTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster) error {
	tcc.defaulting(tc)
	if !tcc.validate(tc) {
		return nil // fatal error, no need to retry on invalid object
	}

	var errs []error
	oldStatus := tc.Status.DeepCopy()

	if err := tcc.updateTidbCluster(tc); err != nil {
		errs = append(errs, err)
	}
	if apiequality.Semantic.DeepEqual(&tc.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	if _, err := tcc.tcControl.UpdateTidbCluster(tc.DeepCopy(), &tc.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (tcc *defaultTidbClusterControl) validate(tc *v1alpha1.TidbCluster) bool {
	errs := v1alpha1validation.ValidateTidbCluster(tc)
	if len(errs) > 0 {
		aggregatedErr := errs.ToAggregate()
		klog.Errorf("tidb cluster %s/%s is not valid and must be fixed first, aggregated error: %v", tc.GetNamespace(), tc.GetName(), aggregatedErr)
		tcc.recorder.Event(tc, v1.EventTypeWarning, "FailedValidation", aggregatedErr.Error())
		return false
	}
	return true
}

func (tcc *defaultTidbClusterControl) defaulting(tc *v1alpha1.TidbCluster) {
	defaulting.SetTidbClusterDefault(tc)
}

func (tcc *defaultTidbClusterControl) updateTidbCluster(tc *v1alpha1.TidbCluster) error {
	// syncing all PVs managed by operator's reclaim policy to Retain
	if err := tcc.reclaimPolicyManager.Sync(tc); err != nil {
		return err
	}

	// cleaning all orphan pods(pd or tikv which don't have a related PVC) managed by operator
	if _, err := tcc.orphanPodsCleaner.Clean(tc); err != nil {
		return err
	}

	// reconcile TiDB discovery service
	if err := tcc.discoveryManager.Reconcile(tc); err != nil {
		return err
	}

	// sync all the pods which need to be restarted
	if err := tcc.podRestarter.Sync(tc); err != nil {
		return err
	}

	// works that should do to making the pd cluster current state match the desired state:
	//   - create or update the pd service
	//   - create or update the pd headless service
	//   - create the pd statefulset
	//   - sync pd cluster status from pd to TidbCluster object
	//   - set two annotations to the first pd member:
	// 	   - label.Bootstrapping
	// 	   - label.Replicas
	//   - upgrade the pd cluster
	//   - scale out/in the pd cluster
	//   - failover the pd cluster
	if err := tcc.pdMemberManager.Sync(tc); err != nil {
		return err
	}

	// works that should do to making the tikv cluster current state match the desired state:
	//   - waiting for the pd cluster available(pd cluster is in quorum)
	//   - create or update tikv headless service
	//   - create the tikv statefulset
	//   - sync tikv cluster status from pd to TidbCluster object
	//   - set scheduler labels to tikv stores
	//   - upgrade the tikv cluster
	//   - scale out/in the tikv cluster
	//   - failover the tikv cluster
	if err := tcc.tikvMemberManager.Sync(tc); err != nil {
		return err
	}

	// works that should do to making the tidb cluster current state match the desired state:
	//   - waiting for the tikv cluster available(at least one peer works)
	//   - create or update tidb headless service
	//   - create the tidb statefulset
	//   - sync tidb cluster status from pd to TidbCluster object
	//   - upgrade the tidb cluster
	//   - scale out/in the tidb cluster
	//   - failover the tidb cluster
	if err := tcc.tidbMemberManager.Sync(tc); err != nil {
		return err
	}

	// works that should do to making the tiflash cluster current state match the desired state:
	//   - waiting for the tidb cluster available
	//   - create or update tiflash headless service
	//   - create the tiflash statefulset
	//   - sync tiflash cluster status from pd to TidbCluster object
	//   - set scheduler labels to tiflash stores
	//   - upgrade the tiflash cluster
	//   - scale out/in the tiflash cluster
	//   - failover the tiflash cluster
	if err := tcc.tiflashMemberManager.Sync(tc); err != nil {
		return err
	}

	// syncing the labels from Pod to PVC and PV, these labels include:
	//   - label.StoreIDLabelKey
	//   - label.MemberIDLabelKey
	//   - label.NamespaceLabelKey
	if err := tcc.metaManager.Sync(tc); err != nil {
		return err
	}

	// cleaning the pod scheduling annotation for pd and tikv
	if _, err := tcc.pvcCleaner.Clean(tc); err != nil {
		return err
	}

	// syncing the pump cluster
	return tcc.pumpMemberManager.Sync(tc)
}

var _ ControlInterface = &defaultTidbClusterControl{}

type FakeTidbClusterControlInterface struct {
	err error
}

func NewFakeTidbClusterControlInterface() *FakeTidbClusterControlInterface {
	return &FakeTidbClusterControlInterface{}
}

func (ftcc *FakeTidbClusterControlInterface) SetUpdateTCError(err error) {
	ftcc.err = err
}

func (ftcc *FakeTidbClusterControlInterface) UpdateTidbCluster(_ *v1alpha1.TidbCluster) error {
	if ftcc.err != nil {
		return ftcc.err
	}
	return nil
}

var _ ControlInterface = &FakeTidbClusterControlInterface{}
