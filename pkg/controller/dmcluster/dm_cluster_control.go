// Copyright 2020 PingCAP, Inc.
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

package dmcluster

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

// ControlInterface implements the control logic for updating DMClusters and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateDMCluster implements the control logic for StatefulSet creation, update, and deletion
	UpdateDMCluster(*v1alpha1.DMCluster) error
}

// NewDefaultDMClusterControl returns a new instance of the default implementation DMClusterControlInterface that
// implements the documented semantics for DMClusters.
func NewDefaultDMClusterControl(
	dcControl controller.DMClusterControlInterface,
	masterMemberManager manager.DMManager,
	workerMemberManager manager.DMManager,
	reclaimPolicyManager manager.DMManager,
	orphanPodsCleaner member.OrphanPodsCleaner,
	pvcCleaner member.PVCCleanerInterface,
	pvcResizer member.PVCResizerInterface,
	conditionUpdater DMClusterConditionUpdater,
	recorder record.EventRecorder) ControlInterface {
	return &defaultDMClusterControl{
		dcControl,
		masterMemberManager,
		workerMemberManager,
		reclaimPolicyManager,
		//metaManager,
		orphanPodsCleaner,
		pvcCleaner,
		pvcResizer,
		conditionUpdater,
		recorder,
	}
}

type defaultDMClusterControl struct {
	dcControl            controller.DMClusterControlInterface
	masterMemberManager  manager.DMManager
	workerMemberManager  manager.DMManager
	reclaimPolicyManager manager.DMManager
	//metaManager       manager.DMManager
	orphanPodsCleaner member.OrphanPodsCleaner
	pvcCleaner        member.PVCCleanerInterface
	pvcResizer        member.PVCResizerInterface
	conditionUpdater  DMClusterConditionUpdater
	recorder          record.EventRecorder
}

// UpdateStatefulSet executes the core logic loop for a dmcluster.
func (c *defaultDMClusterControl) UpdateDMCluster(dc *v1alpha1.DMCluster) error {
	c.defaulting(dc)
	if !c.validate(dc) {
		return nil // fatal error, no need to retry on invalid object
	}

	var errs []error
	oldStatus := dc.Status.DeepCopy()

	if err := c.updateDMCluster(dc); err != nil {
		errs = append(errs, err)
	}

	if err := c.conditionUpdater.Update(dc); err != nil {
		errs = append(errs, err)
	}

	if apiequality.Semantic.DeepEqual(&dc.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	if _, err := c.dcControl.UpdateDMCluster(dc.DeepCopy(), &dc.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (c *defaultDMClusterControl) defaulting(dc *v1alpha1.DMCluster) {
	defaulting.SetDMClusterDefault(dc)
}

func (c *defaultDMClusterControl) validate(dc *v1alpha1.DMCluster) bool {
	errs := v1alpha1validation.ValidateDMCluster(dc)
	if len(errs) > 0 {
		aggregatedErr := errs.ToAggregate()
		klog.Errorf("dm cluster %s/%s is not valid and must be fixed first, aggregated error: %v", dc.GetNamespace(), dc.GetName(), aggregatedErr)
		c.recorder.Event(dc, v1.EventTypeWarning, "FailedValidation", aggregatedErr.Error())
		return false
	}
	return true
}

func (c *defaultDMClusterControl) updateDMCluster(dc *v1alpha1.DMCluster) error {
	var errs []error
	if err := c.reclaimPolicyManager.SyncDM(dc); err != nil {
		return err
	}

	// cleaning all orphan pods(dm-master or dm-worker which don't have a related PVC) managed by operator
	skipReasons, err := c.orphanPodsCleaner.Clean(dc)
	if err != nil {
		return err
	}
	if klog.V(10) {
		for podName, reason := range skipReasons {
			klog.Infof("pod %s of cluster %s/%s is skipped, reason %q", podName, dc.Namespace, dc.Name, reason)
		}
	}

	// works that should do to making the dm-master cluster current state match the desired state:
	//   - create or update the dm-master service
	//   - create or update the dm-master headless service
	//   - create the dm-master statefulset
	//   - sync dm-master cluster status from dm-master to DMCluster object
	//   - set two annotations to the first dm-master member:
	// 	   - label.Bootstrapping
	// 	   - label.Replicas
	//   - upgrade the dm-master cluster
	//   - scale out/in the dm-master cluster
	//   - failover the dm-master cluster
	if err := c.masterMemberManager.SyncDM(dc); err != nil {
		errs = append(errs, err)
	}

	// works that should do to making the dm-worker cluster current state match the desired state:
	//   - waiting for the dm-master cluster available(dm-master cluster is in quorum)
	//   - create or update dm-worker headless service
	//   - create the dm-worker statefulset
	//   - sync dm-worker status from dm-master to DMCluster object s
	//   - upgrade the dm-worker cluster
	//   - scale out/in the dm-worker cluster
	//   - failover the dm-worker cluster
	if err := c.workerMemberManager.SyncDM(dc); err != nil {
		errs = append(errs, err)
	}

	// TODO: syncing labels for dm: syncing the labels from Pod to PVC and PV, these labels include:
	//   - label.StoreIDLabelKey
	//   - label.MemberIDLabelKey
	//   - label.NamespaceLabelKey
	// if err := c.metaManager.Sync(dc); err != nil {
	// 	return err
	// }

	pvcSkipReasons, err := c.pvcCleaner.Clean(dc)
	if err != nil {
		return err
	}
	if klog.V(10) {
		for pvcName, reason := range pvcSkipReasons {
			klog.Infof("pvc %s of cluster %s/%s is skipped, reason %q", pvcName, dc.Namespace, dc.Name, reason)
		}
	}

	// TODO: sync dm cluster attributes
	// syncing the some tidbcluster status attributes
	// 	- sync tidbmonitor reference
	// return c.tidbClusterStatusManager.Sync(dc)

	// resize PVC if necessary
	if err := c.pvcResizer.ResizeDM(dc); err != nil {
		errs = append(errs, err)
	}
	return errorutils.NewAggregate(errs)
}

var _ ControlInterface = &defaultDMClusterControl{}

type FakeDMClusterControlInterface struct {
	err error
}

func NewFakeDMClusterControlInterface() *FakeDMClusterControlInterface {
	return &FakeDMClusterControlInterface{}
}

func (ftcc *FakeDMClusterControlInterface) SetUpdateDCError(err error) {
	ftcc.err = err
}

func (ftcc *FakeDMClusterControlInterface) UpdateDMCluster(_ *v1alpha1.DMCluster) error {
	if ftcc.err != nil {
		return ftcc.err
	}
	return nil
}

var _ ControlInterface = &FakeDMClusterControlInterface{}
