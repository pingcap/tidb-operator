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

package member

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type masterFailover struct {
	deps *controller.Dependencies
}

// NewMasterFailover returns a master Failover
func NewMasterFailover(deps *controller.Dependencies) DMFailover {
	return &masterFailover{
		deps: deps,
	}
}

// Failover is used to failover broken dm-master member
// If there are 3 dm-master members in a dm cluster with 1 broken member dm-master-0, masterFailover will do failover in 3 rounds:
// 1. mark dm-master-0 as a failure Member with non-deleted state (MemberDeleted=false)
// 2. delete the failure member dm-master-0, and mark it deleted (MemberDeleted=true)
// 3. dm-master member manager will add the count of deleted failure members more replicas
// If the count of the failure dm-master member with the deleted state (MemberDeleted=true) is equal or greater than MaxFailoverCount, we will skip failover.
func (mf *masterFailover) Failover(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	if !dc.Status.Master.Synced {
		return fmt.Errorf("DMCluster: %s/%s's dm-master status sync failed, can't failover", ns, dcName)
	}
	if dc.Status.Master.FailureMembers == nil {
		dc.Status.Master.FailureMembers = map[string]v1alpha1.MasterFailureMember{}
	}

	healthCount := 0
	for podName, masterMember := range dc.Status.Master.Members {
		if masterMember.Health {
			healthCount++
		} else {
			mf.deps.Recorder.Eventf(dc, apiv1.EventTypeWarning, "MasterMemberUnhealthy",
				"%s(%s) is unhealthy", podName, masterMember.ID)
		}
	}
	inQuorum := healthCount > len(dc.Status.Master.Members)/2
	if !inQuorum {
		return fmt.Errorf("DMCluster: %s/%s's dm-master cluster is not healthy: %d/%d, "+
			"replicas: %d, failureCount: %d, can't failover",
			ns, dcName, healthCount, dc.MasterStsDesiredReplicas(), dc.Spec.Master.Replicas, len(dc.Status.Master.FailureMembers))
	}

	failureReplicas := getDMMasterFailureReplicas(dc)
	if failureReplicas >= int(*dc.Spec.Master.MaxFailoverCount) {
		klog.Errorf("dm-master failover replicas (%d) reaches the limit (%d), skip failover", failureReplicas, *dc.Spec.Master.MaxFailoverCount)
		return nil
	}

	notDeletedCount := 0
	for _, masterMember := range dc.Status.Master.FailureMembers {
		if !masterMember.MemberDeleted {
			notDeletedCount++
		}
	}
	// we can only failover one at a time
	if notDeletedCount == 0 {
		return mf.tryToMarkAPeerAsFailure(dc)
	}

	return mf.tryToDeleteAFailureMember(dc)
}

func (mf *masterFailover) Recover(dc *v1alpha1.DMCluster) {
	dc.Status.Master.FailureMembers = nil
	klog.Infof("dm-master failover: clearing dm-master failoverMembers, %s/%s", dc.GetNamespace(), dc.GetName())
}

func (mf *masterFailover) RemoveUndesiredFailures(dc *v1alpha1.DMCluster) {}

func (mf *masterFailover) tryToMarkAPeerAsFailure(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	for podName, masterMember := range dc.Status.Master.Members {
		if masterMember.LastTransitionTime.IsZero() {
			continue
		}
		if !mf.isPodDesired(dc, podName) {
			continue
		}

		if dc.Status.Master.FailureMembers == nil {
			dc.Status.Master.FailureMembers = map[string]v1alpha1.MasterFailureMember{}
		}
		deadline := masterMember.LastTransitionTime.Add(mf.deps.CLIConfig.MasterFailoverPeriod)
		_, exist := dc.Status.Master.FailureMembers[podName]
		if masterMember.Health || time.Now().Before(deadline) || exist {
			continue
		}

		ordinal, err := util.GetOrdinalFromPodName(podName)
		if err != nil {
			return err
		}
		pvcName := ordinalPVCName(v1alpha1.DMMasterMemberType, controller.DMMasterMemberName(dcName), ordinal)
		pvc, err := mf.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
		if err != nil {
			return fmt.Errorf("tryToMarkAPeerAsFailure: failed to get pvc %s for dmcluster %s/%s, error: %s", pvcName, ns, dcName, err)
		}

		msg := fmt.Sprintf("dm-master member[%s] is unhealthy", masterMember.ID)
		mf.deps.Recorder.Event(dc, apiv1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "dm-master", podName, msg))

		// mark a peer member failed and return an error to skip reconciliation
		// note that status of dm cluster will be updated always
		dc.Status.Master.FailureMembers[podName] = v1alpha1.MasterFailureMember{
			PodName:       podName,
			MemberID:      masterMember.ID,
			PVCUID:        pvc.UID,
			MemberDeleted: false,
			CreatedAt:     metav1.Now(),
		}
		return controller.RequeueErrorf("marking Pod: %s/%s dm-master member: %s as failure", ns, podName, masterMember.Name)
	}

	return nil
}

// tryToDeleteAFailureMember tries to delete a dm-master member and associated pod &
// pvc. If this succeeds, new pod & pvc will be created by Kubernetes.
// Note that this will fail if the kubelet on the node which failed pod was
// running on is not responding.
func (mf *masterFailover) tryToDeleteAFailureMember(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()
	var failureMember *v1alpha1.MasterFailureMember
	var failurePodName string

	for podName, masterMember := range dc.Status.Master.FailureMembers {
		if !masterMember.MemberDeleted {
			failureMember = &masterMember
			failurePodName = podName
			break
		}
	}
	if failureMember == nil {
		return nil
	}

	// invoke deleteMember api to delete a member from the dm-master cluster
	err := controller.GetMasterClient(mf.deps.DMMasterControl, dc).DeleteMaster(failurePodName)
	if err != nil {
		klog.Errorf("dm-master failover: failed to delete member: [%s/%s], %v", ns, failurePodName, err)
		return err
	}
	klog.Infof("dm-master failover: delete member: [%s/%s] successfully", ns, failurePodName)
	mf.deps.Recorder.Eventf(dc, apiv1.EventTypeWarning, "DMMasterMemberDeleted",
		"[%s/%s] deleted from dmcluster", ns, failurePodName)

	// The order of old PVC deleting and the new Pod creating is not guaranteed by Kubernetes.
	// If new Pod is created before old PVC deleted, new Pod will reuse old PVC.
	// So we must try to delete the PVC and Pod of this dm-master peer over and over,
	// and let StatefulSet create the new dm-master peer with the same ordinal, but don't use the tombstone PV
	pod, err := mf.deps.PodLister.Pods(ns).Get(failurePodName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pods %s for dmcluster %s/%s, error: %s", failurePodName, ns, dcName, err)
	}

	ordinal, err := util.GetOrdinalFromPodName(failurePodName)
	if err != nil {
		return err
	}
	pvcName := ordinalPVCName(v1alpha1.DMMasterMemberType, controller.DMMasterMemberName(dcName), ordinal)
	pvc, err := mf.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pvc %s for dmcluster %s/%s, error: %s", pvcName, ns, dcName, err)
	}

	if pod != nil && pod.DeletionTimestamp == nil {
		err := mf.deps.PodControl.DeletePod(dc, pod)
		if err != nil {
			return err
		}
	}
	if pvc != nil && pvc.DeletionTimestamp == nil && pvc.GetUID() == failureMember.PVCUID {
		err = mf.deps.PVCControl.DeletePVC(dc, pvc)
		if err != nil {
			klog.Errorf("dm-master failover: failed to delete pvc: %s/%s, %v", ns, pvcName, err)
			return err
		}
		klog.Infof("dm-master failover: pvc: %s/%s successfully", ns, pvcName)
	}

	setDMMemberDeleted(dc, failurePodName)
	return nil
}

func (mf *masterFailover) isPodDesired(dc *v1alpha1.DMCluster, podName string) bool {
	ordinals := dc.MasterStsDesiredOrdinals(true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %s/%q: %v", dc.GetName(), podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func setDMMemberDeleted(dc *v1alpha1.DMCluster, podName string) {
	failureMember := dc.Status.Master.FailureMembers[podName]
	failureMember.MemberDeleted = true
	dc.Status.Master.FailureMembers[podName] = failureMember
	klog.Infof("dm-master failover: set dm-master member: %s/%s deleted", dc.GetName(), podName)
}

type fakeMasterFailover struct{}

// NewFakeMasterFailover returns a fake Failover
func NewFakeMasterFailover() DMFailover {
	return &fakeMasterFailover{}
}

func (fmf *fakeMasterFailover) Failover(_ *v1alpha1.DMCluster) error {
	return nil
}

func (fmf *fakeMasterFailover) Recover(_ *v1alpha1.DMCluster) {
}

func (fmf *fakeMasterFailover) RemoveUndesiredFailures(_ *v1alpha1.DMCluster) {
}
