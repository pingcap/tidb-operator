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
	"strings"
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
// 3. PD member manager will add the `count(deleted failure members)` more replicas
//
// If the count of the failure PD member with the deleted state (MemberDeleted=true) is equal or greater than MaxFailoverCount, we will skip failover.
func (f *pdFailover) Failover(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s .Status.PD.Synced = false, can't failover", ns, tcName)
	}
	if tc.Status.PD.FailureMembers == nil {
		tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
	}

	inQuorum, healthCount := f.isPDInQuorum(tc)
	if !inQuorum {
		return fmt.Errorf("TidbCluster: %s/%s's pd cluster is not health, health %d / desired %d,"+
			" replicas %d, failureCount %d, can't failover",
			ns, tcName, healthCount, tc.PDStsDesiredReplicas(), tc.Spec.PD.Replicas, len(tc.Status.PD.FailureMembers))
	}

	pdDeletedFailureReplicas := tc.GetPDDeletedFailureReplicas()
	if pdDeletedFailureReplicas >= *tc.Spec.PD.MaxFailoverCount {
		klog.Errorf("PD failover replicas (%d) reaches the limit (%d), skip failover", pdDeletedFailureReplicas, *tc.Spec.PD.MaxFailoverCount)
		return nil
	}

	notDeletedFailureReplicas := len(tc.Status.PD.FailureMembers) - int(pdDeletedFailureReplicas)

	// we can only failover one at a time
	if notDeletedFailureReplicas == 0 {
		return f.tryToMarkAPeerAsFailure(tc)
	}

	return f.tryToDeleteAFailureMember(tc)
}

func (f *pdFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.PD.FailureMembers = nil
	klog.Infof("pd failover: clearing pd failoverMembers, %s/%s", tc.GetNamespace(), tc.GetName())
}

func (f *pdFailover) tryToMarkAPeerAsFailure(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for pdName, pdMember := range tc.Status.PD.Members {
		if pdMember.LastTransitionTime.IsZero() {
			continue
		}
		podName := strings.Split(pdName, ".")[0]
		if !f.isPodDesired(tc, podName) {
			continue
		}

		if tc.Status.PD.FailureMembers == nil {
			tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
		}
		deadline := pdMember.LastTransitionTime.Add(f.deps.CLIConfig.PDFailoverPeriod)
		_, exist := tc.Status.PD.FailureMembers[pdName]

		if pdMember.Health || time.Now().Before(deadline) || exist {
			continue
		}

		ordinal, err := util.GetOrdinalFromPodName(podName)
		if err != nil {
			return err
		}
		pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
		pvc, err := f.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
		if err != nil {
			return fmt.Errorf("tryToMarkAPeerAsFailure: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
		}

		msg := fmt.Sprintf("pd member[%s] is unhealthy", pdMember.ID)
		f.deps.Recorder.Event(tc, apiv1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "pd", pdName, msg))

		// mark a peer member failed and return an error to skip reconciliation
		// note that status of tidb cluster will be updated always
		tc.Status.PD.FailureMembers[pdName] = v1alpha1.PDFailureMember{
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

// tryToDeleteAFailureMember tries to delete a PD member and associated Pod & PVC.
// On success, new Pod & PVC will be created.
// Note that this will fail if the kubelet on the node on which failed Pod was running is not responding.
func (f *pdFailover) tryToDeleteAFailureMember(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	var failureMember *v1alpha1.PDFailureMember
	var failurePodName string
	var failurePDName string

	for pdName, pdMember := range tc.Status.PD.FailureMembers {
		if !pdMember.MemberDeleted {
			failureMember = &pdMember
			failurePodName = strings.Split(pdName, ".")[0]
			failurePDName = pdName
			break
		}
	}
	if failureMember == nil {
		klog.Infof("No PD FailureMembers to delete for tc %s/%s", ns, tcName)
		return nil
	}

	memberID, err := strconv.ParseUint(failureMember.MemberID, 10, 64)
	if err != nil {
		return err
	}
	// invoke deleteMember api to delete a member from the pd cluster
	if err := controller.GetPDClient(f.deps.PDControl, tc).DeleteMemberByID(memberID); err != nil {
		klog.Errorf("pd failover: failed to delete member %d, %v", memberID, err)
		return err
	}
	klog.Infof("pd failover: delete member %d successfully", memberID)
	f.deps.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDMemberDeleted", "failure member %s(%d) deleted from PD cluster", failurePodName, memberID)

	// The order of old PVC deleting and the new Pod creating is not guaranteed by Kubernetes.
	// If new Pod is created before old PVC deleted, new Pod will reuse old PVC.
	// In this case, there is a period during which both new & old Pod is using the same PVC.
	// So we must try to delete the PVC and Pod of this PD peer over and over again,
	// and let StatefulSet to create the new PD peer with the same ordinal, but not to use the tombstone PV
	pod, err := f.deps.PodLister.Pods(ns).Get(failurePodName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pods %s for cluster %s/%s, error: %s", failurePodName, ns, tcName, err)
	}

	if pod != nil {
		pvcs, err := util.ResolvePVCFromPod(pod, f.deps.PVCLister)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("tryToDeleteAFailureMember: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
		}
		if pod.DeletionTimestamp == nil {
			if err := f.deps.PodControl.DeletePod(tc, pod); err != nil {
				return err
			}
		}
		for _, pvc := range pvcs {
			if pvc.DeletionTimestamp == nil && pvc.GetUID() == failureMember.PVCUID {
				if err := f.deps.PVCControl.DeletePVC(tc, pvc); err != nil {
					klog.Errorf("tryToDeleteAFailureMember: failed to delete pvc: %s/%s, error: %s", ns, pvc.Name, err)
					return err
				}
				klog.Infof("tryToDeleteAFailureMember: delete pvc %s/%s successfully", ns, pvc.Name)
			}
		}
	}

	setMemberDeleted(tc, failurePDName)
	return nil
}

func (f *pdFailover) isPodDesired(tc *v1alpha1.TidbCluster, podName string) bool {
	ordinals := tc.PDStsDesiredOrdinals(true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func (f *pdFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
}

func setMemberDeleted(tc *v1alpha1.TidbCluster, pdName string) {
	failureMember := tc.Status.PD.FailureMembers[pdName]
	failureMember.MemberDeleted = true
	tc.Status.PD.FailureMembers[pdName] = failureMember
	klog.Infof("pd failover: set pd member: %s/%s deleted", tc.GetName(), pdName)
}

// is healthy PD more than a half
func (f *pdFailover) isPDInQuorum(tc *v1alpha1.TidbCluster) (bool, int) {
	healthCount := 0
	for podName, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			healthCount++
		} else {
			f.deps.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDMemberUnhealthy", "%s(%s) is unhealthy", podName, pdMember.ID)
		}
	}
	for _, pdMember := range tc.Status.PD.PeerMembers {
		if pdMember.Health {
			healthCount++
		} else {
			f.deps.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDPeerMemberUnhealthy", "%s(%s) is unhealthy", pdMember.Name, pdMember.ID)
		}
	}
	return healthCount > (len(tc.Status.PD.Members)+len(tc.Status.PD.PeerMembers))/2, healthCount
}

type fakePDFailover struct{}

// NewFakePDFailover returns a fake Failover
func NewFakePDFailover() Failover {
	return &fakePDFailover{}
}

func (f *fakePDFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (f *fakePDFailover) Recover(_ *v1alpha1.TidbCluster) {
}

func (f *fakePDFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
}
