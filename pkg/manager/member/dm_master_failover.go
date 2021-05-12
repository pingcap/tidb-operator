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
	"k8s.io/apimachinery/pkg/types"
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
func (f *masterFailover) Failover(dc *v1alpha1.DMCluster) error {
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
			f.deps.Recorder.Eventf(dc, apiv1.EventTypeWarning, "MasterMemberUnhealthy",
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
		return f.tryToMarkAPeerAsFailure(dc)
	}

	return f.tryToDeleteAFailureMember(dc)
}

func (f *masterFailover) Recover(dc *v1alpha1.DMCluster) {
	dc.Status.Master.FailureMembers = nil
	klog.Infof("dm-master failover: clearing dm-master failoverMembers, %s/%s", dc.GetNamespace(), dc.GetName())
}

func (f *masterFailover) RemoveUndesiredFailures(dc *v1alpha1.DMCluster) {}

func (f *masterFailover) tryToMarkAPeerAsFailure(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	for podName, masterMember := range dc.Status.Master.Members {
		if masterMember.LastTransitionTime.IsZero() {
			continue
		}
		if !f.isPodDesired(dc, podName) {
			continue
		}

		if dc.Status.Master.FailureMembers == nil {
			dc.Status.Master.FailureMembers = map[string]v1alpha1.MasterFailureMember{}
		}
		failoverDeadline := masterMember.LastTransitionTime.Add(f.deps.CLIConfig.MasterFailoverPeriod)
		_, exist := dc.Status.Master.FailureMembers[podName]
		if masterMember.Health || time.Now().Before(failoverDeadline) || exist {
			continue
		}

		pod, err := f.deps.PodLister.Pods(ns).Get(podName)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("tryToMarkAPeerAsFailure: failed to get pod %s/%s for tc %s/%s, error: %s", ns, podName, ns, dcName, err)
		}
		if pod == nil {
			klog.Infof("tryToMarkAPeerAsFailure: failure pod %s/%s not found, skip", ns, podName)
			return nil
		}

		pvcs, err := util.ResolvePVCFromPod(pod, f.deps.PVCLister)
		if err != nil {
			return fmt.Errorf("tryToMarkAPeerAsFailure: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, dcName, err)
		}

		f.deps.Recorder.Eventf(dc, apiv1.EventTypeWarning, "DMMasterMemberUnhealthy", "%s/%s(%s) is unhealthy", ns, podName, masterMember.ID)

		// mark a peer member failed and return an error to skip reconciliation
		// note that status of dm cluster will be updated always
		pvcUIDSet := make(map[types.UID]struct{})
		for _, pvc := range pvcs {
			pvcUIDSet[pvc.UID] = struct{}{}
		}
		dc.Status.Master.FailureMembers[podName] = v1alpha1.MasterFailureMember{
			PodName:       podName,
			MemberID:      masterMember.ID,
			PVCUIDSet:     pvcUIDSet,
			MemberDeleted: false,
			CreatedAt:     metav1.Now(),
		}
		return controller.RequeueErrorf("marking Pod: %s/%s dm-master member: %s as failure", ns, podName, masterMember.Name)
	}

	return nil
}

// tryToDeleteAFailureMember tries to delete a dm-master member and associated Pod & PVC.
// On success, new Pod & PVC will be created.
// Note that this will fail if the kubelet on the node on which failed Pod was running is not responding.
func (f *masterFailover) tryToDeleteAFailureMember(dc *v1alpha1.DMCluster) error {
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
		klog.Infof("No DM Master FailureMembers to delete for tc %s/%s", ns, dcName)
		return nil
	}

	// invoke deleteMember api to delete a member from the dm-master cluster
	if err := controller.GetMasterClient(f.deps.DMMasterControl, dc).DeleteMaster(failurePodName); err != nil {
		klog.Errorf("dm-master tryToDeleteAFailureMember: failed to delete member: %s/%s, %v", ns, failurePodName, err)
		return err
	}
	klog.Infof("dm-master tryToDeleteAFailureMember: delete member %s/%s successfully", ns, failurePodName)
	f.deps.Recorder.Eventf(dc, apiv1.EventTypeWarning, "DMMasterMemberDeleted", "failure member %s/%s deleted from dmcluster", ns, failurePodName)

	// The order of old PVC deleting and the new Pod creating is not guaranteed by Kubernetes.
	// If new Pod is created before old PVCs are deleted, the Statefulset will try to use the old PVCs and skip creating new PVCs.
	// This could result in 2 possible cases:
	// 1. If the old PVCs are first mounted successfully by the new Pod, the following pvc deletion will fail and return error.
	//    We will try to delete the Pod and PVCs again in the next requeued run.
	// 2. If the old PVCs are first deleted successfully here, the new Pods will try to mount non-existing PVCs, which will pend forever.
	//    This is where OrphanPodsCleaner kicks in, which will delete the pending Pods in this situation.
	//    Please refer to orphan_pods_cleaner.go for details.
	pod, err := f.deps.PodLister.Pods(ns).Get(failurePodName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("dm-master tryToDeleteAFailureMember: failed to get Pod %s/%s for dmcluster %s/%s, error: %s", ns, failurePodName, ns, dcName, err)
	}
	if pod != nil {
		if pod.DeletionTimestamp == nil {
			if err := f.deps.PodControl.DeletePod(dc, pod); err != nil {
				return err
			}
		}
	} else {
		klog.Infof("dm-master tryToDeleteAFailureMember: failure Pod %s/%s not found, skip deleting Pod", ns, failurePodName)
	}

	// FIXME: change to use label for PVC selection
	pvcs, err := util.ResolvePVCFromPod(pod, f.deps.PVCLister)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("dm-master tryToDeleteAFailureMember: failed to get PVCs for Pod %s/%s in dc %s/%s, error: %s", ns, pod.Name, ns, dcName, err)
	}
	for _, pvc := range pvcs {
		if pvc.DeletionTimestamp == nil && pvc.GetUID() == failureMember.PVCUID {
			if err = f.deps.PVCControl.DeletePVC(dc, pvc); err != nil {
				klog.Errorf("dm-master tryToDeleteAFailureMember: failed to delete PVC: %s/%s, error: %s", ns, pvc.Name, err)
				return err
			}
			klog.Infof("dm-master tryToDeleteAFailureMember: delete PVC %s/%s successfully", ns, pvc.Name)
		}
	}

	setDMMemberDeleted(dc, failurePodName)
	return nil
}

func (f *masterFailover) isPodDesired(dc *v1alpha1.DMCluster, podName string) bool {
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

func (f *fakeMasterFailover) Failover(_ *v1alpha1.DMCluster) error {
	return nil
}

func (f *fakeMasterFailover) Recover(_ *v1alpha1.DMCluster) {
}

func (f *fakeMasterFailover) RemoveUndesiredFailures(_ *v1alpha1.DMCluster) {
}
