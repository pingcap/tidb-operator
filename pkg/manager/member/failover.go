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

// Failover implements the logic for pd/tikv/tidb's failover and recovery.
type Failover interface {
	Failover(*v1alpha1.TidbCluster) error
	Recovery(*v1alpha1.TidbCluster)
}

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

	healthCount := 0
	for _, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			healthCount++
		}
	}
	inQuorum := healthCount > int(tc.Spec.PD.Replicas/2)

	notDeletedCount := 0
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			notDeletedCount++
		}
	}

	// mark a peer as failure
	for podName, pdMember := range tc.Status.PD.Members {
		deadline := pdMember.LastTransitionTime.Add(pf.pdFailoverPeriod)
		_, exist := tc.Status.PD.FailureMembers[podName]
		if !pdMember.Health && time.Now().After(deadline) && !exist && notDeletedCount == 0 {
			if !inQuorum {
				return fmt.Errorf("TidbCluster: %s/%s's pd cluster is not health: %d/%d, can't failover",
					ns, tcName, healthCount, tc.Spec.PD.Replicas)
			}
			err := pf.markThisMemberAsFailure(tc, pdMember)
			if err != nil {
				return err
			}
			break
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

	// try to delete the PVC and Pod of this PD peer over and over,
	// let StatefulSet create the new PD peer with the same ordinal,
	// but not use the tombstone PV
	for podName, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			continue
		}

		// increase the replicas to add a new PD peer
		if failureMember.Replicas+1 > tc.Spec.PD.Replicas {
			tc.Spec.PD.Replicas = failureMember.Replicas + 1
		}

		ordinal, err := getOrdinalFromPodName(podName)
		if err != nil {
			return err
		}
		pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
		pvc, err := pf.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
		if errors.IsNotFound(err) {
			// pvc deleted, pod should be deleted already:
			// https://kubernetes.io/docs/concepts/storage/persistent-volumes/#storage-object-in-use-protection
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

		pod, err := pf.podLister.Pods(ns).Get(podName)
		if err != nil && !errors.IsNotFound(err) {
			return err
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

func (pf *pdFailover) Recovery(tc *v1alpha1.TidbCluster) {
	defer func() {
		tc.Status.PD.FailureMembers = nil
	}()

	modified := true
	minReplicas := tc.Spec.PD.Replicas
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if failureMember.Replicas+1 == tc.Spec.PD.Replicas {
			modified = false
		}

		if failureMember.Replicas < minReplicas {
			minReplicas = failureMember.Replicas
		}
	}

	// if user modify the TidbCluster's PD replicas, don't need recovery
	if modified {
		return
	}
	if minReplicas != tc.Spec.PD.Replicas {
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

func (fpf *fakePDFailover) Failover(tc *v1alpha1.TidbCluster) error {
	return nil
}

func (fpf *fakePDFailover) Recovery(tc *v1alpha1.TidbCluster) {
	return
}
