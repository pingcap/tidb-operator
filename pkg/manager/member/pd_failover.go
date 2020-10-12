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

package member

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type pdFailover struct {
	deps *controller.Dependencies
}

// NewPDFailover returns a pd Failover
func NewPDFailover(deps *controller.Dependencies) Failover {
	return &pdFailover{
		deps: deps,
	}
}

// Failover is used to failover broken pd member
// If there are 3 PD members in a tidb cluster with 1 broken member pd-0, pdFailover will do failover in 3 rounds:
// 1. mark pd-0 as a failure Member with non-deleted state (MemberDeleted=false)
// 2. delete the failure member pd-0, and mark it deleted (MemberDeleted=true)
// 3. PD member manager will add the count of deleted failure members more replicas
// If the count of the failure PD member with the deleted state (MemberDeleted=true) is equal or greater than MaxFailoverCount, we will skip failover.
func (pf *pdFailover) Failover(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed, can't failover", ns, tcName)
	}
	if tc.Status.PD.FailureMembers == nil {
		tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
	}

	healthCount := 0
	for podName, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			healthCount++
		} else {
			pf.deps.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDMemberUnhealthy",
				"%s(%s) is unhealthy", podName, pdMember.ID)
		}
	}
	inQuorum := healthCount > len(tc.Status.PD.Members)/2
	if !inQuorum {
		return fmt.Errorf("TidbCluster: %s/%s's pd cluster is not health: %d/%d, "+
			"replicas: %d, failureCount: %d, can't failover",
			ns, tcName, healthCount, tc.PDStsDesiredReplicas(), tc.Spec.PD.Replicas, len(tc.Status.PD.FailureMembers))
	}

	failureReplicas := getFailureReplicas(tc)
	if failureReplicas >= int(*tc.Spec.PD.MaxFailoverCount) {
		klog.Errorf("PD failover replicas (%d) reaches the limit (%d), skip failover", failureReplicas, *tc.Spec.PD.MaxFailoverCount)
		return nil
	}

	notDeletedCount := 0
	for _, pdMember := range tc.Status.PD.FailureMembers {
		if !pdMember.MemberDeleted {
			notDeletedCount++
		}
	}
	// we can only failover one at a time
	if notDeletedCount == 0 {
		return pf.tryToMarkAPeerAsFailure(tc)
	}

	return pf.tryToDeleteAFailureMember(tc)
}

func (pf *pdFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.PD.FailureMembers = nil
	klog.Infof("pd failover: clearing pd failoverMembers, %s/%s", tc.GetNamespace(), tc.GetName())
}

func (pf *pdFailover) tryToMarkAPeerAsFailure(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for podName, pdMember := range tc.Status.PD.Members {
		if pdMember.LastTransitionTime.IsZero() {
			continue
		}
		if !pf.isPodDesired(tc, podName) {
			continue
		}

		if tc.Status.PD.FailureMembers == nil {
			tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
		}
		deadline := pdMember.LastTransitionTime.Add(pf.deps.CLIConfig.PDFailoverPeriod)
		_, exist := tc.Status.PD.FailureMembers[podName]
		if pdMember.Health || time.Now().Before(deadline) || exist {
			continue
		}

		ordinal, err := util.GetOrdinalFromPodName(podName)
		if err != nil {
			return err
		}
		pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
		pvc, err := pf.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
		if err != nil {
			return fmt.Errorf("tryToMarkAPeerAsFailure: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
		}

		msg := fmt.Sprintf("pd member[%s] is unhealthy", pdMember.ID)
		pf.deps.Recorder.Event(tc, apiv1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "pd", podName, msg))

		// mark a peer member failed and return an error to skip reconciliation
		// note that status of tidb cluster will be updated always
		tc.Status.PD.FailureMembers[podName] = v1alpha1.PDFailureMember{
			PodName:       podName,
			MemberID:      pdMember.ID,
			PVCUID:        pvc.UID,
			MemberDeleted: false,
			CreatedAt:     metav1.Now(),
		}
		return controller.RequeueErrorf("marking Pod: %s/%s pd member: %s as failure", ns, podName, pdMember.Name)
	}

	return nil
}

// tryToDeleteAFailureMember tries to delete a PD member and associated pod &
// pvc. If this succeeds, new pod & pvc will be created by Kubernetes.
// Note that this will fail if the kubelet on the node which failed pod was
// running on is not responding.
func (pf *pdFailover) tryToDeleteAFailureMember(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	var failureMember *v1alpha1.PDFailureMember
	var failurePodName string

	for podName, pdMember := range tc.Status.PD.FailureMembers {
		if !pdMember.MemberDeleted {
			failureMember = &pdMember
			failurePodName = podName
			break
		}
	}
	if failureMember == nil {
		return nil
	}

	memberID, err := strconv.ParseUint(failureMember.MemberID, 10, 64)
	if err != nil {
		return err
	}
	// invoke deleteMember api to delete a member from the pd cluster
	err = controller.GetPDClient(pf.deps.PDControl, tc).DeleteMemberByID(memberID)
	if err != nil {
		klog.Errorf("pd failover: failed to delete member: %d, %v", memberID, err)
		return err
	}
	klog.Infof("pd failover: delete member: %d successfully", memberID)
	pf.deps.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDMemberDeleted",
		"%s(%d) deleted from cluster", failurePodName, memberID)

	// The order of old PVC deleting and the new Pod creating is not guaranteed by Kubernetes.
	// If new Pod is created before old PVC deleted, new Pod will reuse old PVC.
	// So we must try to delete the PVC and Pod of this PD peer over and over,
	// and let StatefulSet create the new PD peer with the same ordinal, but don't use the tombstone PV
	pod, err := pf.deps.PodLister.Pods(ns).Get(failurePodName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pods %s for cluster %s/%s, error: %s", failurePodName, ns, tcName, err)
	}

	ordinal, err := util.GetOrdinalFromPodName(failurePodName)
	if err != nil {
		return err
	}
	pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
	pvc, err := pf.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
	}

	if pod != nil && pod.DeletionTimestamp == nil {
		err := pf.deps.PodControl.DeletePod(tc, pod)
		if err != nil {
			return err
		}
	}
	if pvc != nil && pvc.DeletionTimestamp == nil && pvc.GetUID() == failureMember.PVCUID {
		err = pf.deps.PVCControl.DeletePVC(tc, pvc)
		if err != nil {
			klog.Errorf("pd failover: failed to delete pvc: %s/%s, %v", ns, pvcName, err)
			return err
		}
		klog.Infof("pd failover: pvc: %s/%s successfully", ns, pvcName)
	}

	setMemberDeleted(tc, failurePodName)
	return nil
}

func (pf *pdFailover) isPodDesired(tc *v1alpha1.TidbCluster, podName string) bool {
	ordinals := tc.PDStsDesiredOrdinals(true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func (pf *pdFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	return
}

func setMemberDeleted(tc *v1alpha1.TidbCluster, podName string) {
	failureMember := tc.Status.PD.FailureMembers[podName]
	failureMember.MemberDeleted = true
	tc.Status.PD.FailureMembers[podName] = failureMember
	klog.Infof("pd failover: set pd member: %s/%s deleted", tc.GetName(), podName)
}

type fakePDFailover struct{}

// NewFakePDFailover returns a fake Failover
func NewFakePDFailover() Failover {
	return &fakePDFailover{}
}

func (fpf *fakePDFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (fpf *fakePDFailover) Recover(_ *v1alpha1.TidbCluster) {
	return
}

func (fpf *fakePDFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	return
}
