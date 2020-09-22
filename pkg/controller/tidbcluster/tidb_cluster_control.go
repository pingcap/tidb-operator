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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
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
<<<<<<< HEAD
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
		recorder,
=======
	pvcResizer member.PVCResizerInterface,
	pumpMemberManager manager.Manager,
	tiflashMemberManager manager.Manager,
	ticdcMemberManager manager.Manager,
	discoveryManager member.TidbDiscoveryManager,
	tidbClusterStatusManager manager.Manager,
	conditionUpdater TidbClusterConditionUpdater,
	recorder record.EventRecorder) ControlInterface {
	return &defaultTidbClusterControl{
		tcControl:                tcControl,
		pdMemberManager:          pdMemberManager,
		tikvMemberManager:        tikvMemberManager,
		tidbMemberManager:        tidbMemberManager,
		reclaimPolicyManager:     reclaimPolicyManager,
		metaManager:              metaManager,
		orphanPodsCleaner:        orphanPodsCleaner,
		pvcCleaner:               pvcCleaner,
		pvcResizer:               pvcResizer,
		pumpMemberManager:        pumpMemberManager,
		tiflashMemberManager:     tiflashMemberManager,
		ticdcMemberManager:       ticdcMemberManager,
		discoveryManager:         discoveryManager,
		tidbClusterStatusManager: tidbClusterStatusManager,
		conditionUpdater:         conditionUpdater,
		recorder:                 recorder,
>>>>>>> 18703b9... Remove the PodRestarter controller and `tidb.pingcap.com/pod-defer-deleting` annotation (#3296)
	}
}

type defaultTidbClusterControl struct {
<<<<<<< HEAD
	tcControl            controller.TidbClusterControlInterface
	pdMemberManager      manager.Manager
	tikvMemberManager    manager.Manager
	tidbMemberManager    manager.Manager
	reclaimPolicyManager manager.Manager
	metaManager          manager.Manager
	orphanPodsCleaner    member.OrphanPodsCleaner
	pvcCleaner           member.PVCCleanerInterface
	recorder             record.EventRecorder
=======
	tcControl                controller.TidbClusterControlInterface
	pdMemberManager          manager.Manager
	tikvMemberManager        manager.Manager
	tidbMemberManager        manager.Manager
	reclaimPolicyManager     manager.Manager
	metaManager              manager.Manager
	orphanPodsCleaner        member.OrphanPodsCleaner
	pvcCleaner               member.PVCCleanerInterface
	pvcResizer               member.PVCResizerInterface
	pumpMemberManager        manager.Manager
	tiflashMemberManager     manager.Manager
	ticdcMemberManager       manager.Manager
	discoveryManager         member.TidbDiscoveryManager
	tidbClusterStatusManager manager.Manager
	conditionUpdater         TidbClusterConditionUpdater
	recorder                 record.EventRecorder
>>>>>>> 18703b9... Remove the PodRestarter controller and `tidb.pingcap.com/pod-defer-deleting` annotation (#3296)
}

// UpdateStatefulSet executes the core logic loop for a tidbcluster.
func (tcc *defaultTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster) error {
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

func (tcc *defaultTidbClusterControl) updateTidbCluster(tc *v1alpha1.TidbCluster) error {
	// syncing all PVs managed by operator's reclaim policy to Retain
	if err := tcc.reclaimPolicyManager.Sync(tc); err != nil {
		return err
	}

<<<<<<< HEAD
	// cleaning all orphan pods(pd or tikv which don't have a related PVC) managed by operator
	if _, err := tcc.orphanPodsCleaner.Clean(tc); err != nil {
=======
	// cleaning all orphan pods(pd, tikv or tiflash which don't have a related PVC) managed by operator
	skipReasons, err := tcc.orphanPodsCleaner.Clean(tc)
	if err != nil {
		return err
	}
	if klog.V(10) {
		for podName, reason := range skipReasons {
			klog.Infof("pod %s of cluster %s/%s is skipped, reason %q", podName, tc.Namespace, tc.Name, reason)
		}
	}

	// reconcile TiDB discovery service
	if err := tcc.discoveryManager.Reconcile(tc); err != nil {
>>>>>>> 18703b9... Remove the PodRestarter controller and `tidb.pingcap.com/pod-defer-deleting` annotation (#3296)
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

	// syncing the labels from Pod to PVC and PV, these labels include:
	//   - label.StoreIDLabelKey
	//   - label.MemberIDLabelKey
	//   - label.NamespaceLabelKey
	if err := tcc.metaManager.Sync(tc); err != nil {
		return err
	}

	// cleaning the pod scheduling annotation for pd and tikv
	_, err := tcc.pvcCleaner.Clean(tc)
	return err
}
