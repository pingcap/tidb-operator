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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// TODO add maxFailoverCount
type pdFailover struct {
	cli              versioned.Interface
	tcControl        controller.TidbClusterControlInterface
	pdControl        controller.PDControlInterface
	pdFailoverPeriod time.Duration
	podLister        corelisters.PodLister
	podControl       controller.PodControlInterface
	pvcLister        corelisters.PersistentVolumeClaimLister
	pvcControl       controller.PVCControlInterface
	pvLister         corelisters.PersistentVolumeLister
}

// NewPDFailover returns a pd Failover
func NewPDFailover(cli versioned.Interface,
	tcControl controller.TidbClusterControlInterface,
	pdControl controller.PDControlInterface,
	pdFailoverPeriod time.Duration,
	podLister corelisters.PodLister,
	podControl controller.PodControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface,
	pvLister corelisters.PersistentVolumeLister) Failover {
	return &pdFailover{
		cli,
		tcControl,
		pdControl,
		pdFailoverPeriod,
		podLister,
		podControl,
		pvcLister,
		pvcControl,
		pvLister}
}

func (pf *pdFailover) Failover(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.Status.PD.SyncSuccess {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed, can't failover", ns, tcName)
	}

	healthCount := 0
	for _, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			healthCount++
		}
	}
	inQuorum := healthCount > int(tc.Spec.PD.Replicas/2)
	if !inQuorum {
		return fmt.Errorf("TidbCluster: %s/%s's pd cluster is not health: %d/%d, can't failover",
			ns, tcName, healthCount, tc.Spec.PD.Replicas)
	}
	if tc.Status.PD.FailureMembers == nil {
		tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
	}
	notDeletedCount := 0
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			notDeletedCount++
		}
	}
	// we can only failover one at a time
	if notDeletedCount == 0 {
		// mark a peer as failure
		for podName, pdMember := range tc.Status.PD.Members {
			if pdMember.LastTransitionTime.IsZero() {
				continue
			}
			deadline := pdMember.LastTransitionTime.Add(pf.pdFailoverPeriod)
			_, exist := tc.Status.PD.FailureMembers[podName]
			if !pdMember.Health && time.Now().After(deadline) && !exist {
				err := pf.markThisMemberAsFailure(tc, pdMember)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	// invoke deleteMember api to delete a member from the pd cluster
	for podName, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			err := pf.pdControl.GetPDClient(tc).DeleteMember(failureMember.PodName)
			if err != nil {
				return err
			}
			failureMember.MemberDeleted = true
			tc.Status.PD.FailureMembers[podName] = failureMember
			break
		}
	}
	// The order of old PVC deleting and the new Pod creating is not guaranteed by Kubernetes.
	// If new Pod is created before old PVC deleted, new Pod will reuse old PVC.
	// So we must try to delete the PVC and Pod of this PD peer over and over,
	// and let StatefulSet create the new PD peer with the same ordinal, but don't use the tombstone PV
	for podName, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			continue
		}
		// increase the replicas to add a new PD peer
		if failureMember.Replicas+1 > tc.Spec.PD.Replicas {
			tc.Spec.PD.Replicas = failureMember.Replicas + 1
		}
		pod, err := pf.podLister.Pods(ns).Get(podName)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		ordinal, err := getOrdinalFromPodName(podName)
		if err != nil {
			return err
		}
		pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
		pvc, err := pf.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
		if errors.IsNotFound(err) {
			if pod != nil && pod.DeletionTimestamp == nil {
				err := pf.podControl.DeletePod(tc, pod)
				if err != nil {
					return err
				}
			}
			continue
		}
		if err != nil {
			return err
		}
		pv, err := pf.pvLister.Get(pvc.Spec.VolumeName)
		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if string(pv.UID) != string(failureMember.PVUID) {
			continue
		}
		if pod != nil && pod.DeletionTimestamp == nil {
			err := pf.podControl.DeletePod(tc, pod)
			if err != nil {
				return err
			}
		}
		if pvc.DeletionTimestamp == nil {
			err = pf.pvcControl.DeletePVC(tc, pvc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pf *pdFailover) Recover(tc *v1alpha1.TidbCluster) {
	defer func() {
		tc.Status.PD.FailureMembers = nil
	}()
	maxReplicas := int32(0)
	minReplicas := int32(0)
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if minReplicas == int32(0) {
			minReplicas = failureMember.Replicas
		}
		if failureMember.Replicas > maxReplicas {
			maxReplicas = failureMember.Replicas
		} else if failureMember.Replicas < minReplicas {
			minReplicas = failureMember.Replicas
		}
	}
	if maxReplicas+1 == tc.Spec.PD.Replicas {
		tc.Spec.PD.Replicas = minReplicas
	}
}

func (pf *pdFailover) markThisMemberAsFailure(tc *v1alpha1.TidbCluster, pdMember v1alpha1.PDMember) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	podName := pdMember.Name
	ordinal, err := getOrdinalFromPodName(podName)
	if err != nil {
		return err
	}
	pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
	pvc, err := pf.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return err
	}
	pv, err := pf.pvLister.Get(pvc.Spec.VolumeName)
	if err != nil {
		return err
	}
	if tc.Status.PD.FailureMembers == nil {
		tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
	}
	tc.Status.PD.FailureMembers[podName] = v1alpha1.PDFailureMember{
		PodName:       podName,
		MemberID:      pdMember.ID,
		PVUID:         pv.UID,
		Replicas:      tc.Spec.PD.Replicas,
		MemberDeleted: false,
	}
	// we must update TidbCluster immediately before delete member, or data may not be consistent
	tc, err = pf.tcControl.UpdateTidbCluster(tc)
	return err
}

func getOrdinalFromPodName(podName string) (int32, error) {
	ordinalStr := podName[strings.LastIndex(podName, "-")+1:]
	ordinalInt, err := strconv.Atoi(ordinalStr)
	if err != nil {
		return int32(0), err
	}
	return int32(ordinalInt), nil
}

func allPDMembersAreReady(tc *v1alpha1.TidbCluster) bool {
	if int(tc.Spec.PD.Replicas) != len(tc.Status.PD.Members) {
		return false
	}
	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			return false
		}
	}
	return true
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
