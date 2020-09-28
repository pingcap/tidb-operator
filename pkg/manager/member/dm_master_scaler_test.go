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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestMasterScalerScaleOut(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		update           func(cluster *v1alpha1.DMCluster)
		masterUpgrading  bool
		hasPVC           bool
		hasDeferAnn      bool
		annoIsNil        bool
		pvcDeleteErr     bool
		statusSyncFailed bool
		err              bool
		changed          bool
	}

	testFn := func(test testcase, t *testing.T) {
		dc := newDMClusterForMaster()
		test.update(dc)

		if test.masterUpgrading {
			dc.Status.Master.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForDMScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)

		scaler, _, pvcIndexer, pvcControl := newFakeMasterScaler()

		pvc := newPVCForStatefulSet(oldSet, v1alpha1.DMMasterMemberType, dc.Name)
		pvc.Name = ordinalPVCName(v1alpha1.DMMasterMemberType, oldSet.GetName(), *oldSet.Spec.Replicas)
		if !test.annoIsNil {
			pvc.Annotations = map[string]string{}
		}

		if test.hasDeferAnn {
			pvc.Annotations = map[string]string{}
			pvc.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
		}
		if test.hasPVC {
			pvcIndexer.Add(pvc)
		}

		if test.pvcDeleteErr {
			pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		dc.Status.Master.Synced = !test.statusSyncFailed

		err := scaler.ScaleOut(dc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(6))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:             "normal",
			update:           normalMasterMember,
			masterUpgrading:  false,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "dm-master is upgrading",
			update:           normalMasterMember,
			masterUpgrading:  true,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "cache don't have pvc",
			update:           normalMasterMember,
			masterUpgrading:  false,
			hasPVC:           false,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "pvc annotation is not nil but doesn't contain defer deletion annotation",
			update:           normalMasterMember,
			masterUpgrading:  false,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        false,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "pvc annotations defer deletion is not nil, pvc delete failed",
			update:           normalMasterMember,
			masterUpgrading:  false,
			hasPVC:           true,
			hasDeferAnn:      true,
			annoIsNil:        false,
			pvcDeleteErr:     true,
			statusSyncFailed: false,
			err:              true,
			changed:          false,
		},
		{
			name:             "dm-master status sync failed",
			update:           normalMasterMember,
			masterUpgrading:  false,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: true,
			err:              true,
			changed:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestMasterScalerScaleIn(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		masterUpgrading     bool
		hasPVC              bool
		pvcUpdateErr        bool
		deleteMemberErr     bool
		statusSyncFailed    bool
		err                 bool
		changed             bool
		isMemberStillRemain bool
		isLeader            bool
	}

	testFn := func(test testcase, t *testing.T) {
		dc := newDMClusterForMaster()
		leaderPodName := DMMasterPodName(dc.GetName(), 4)

		if test.masterUpgrading {
			dc.Status.Master.Phase = v1alpha1.UpgradePhase
		}
		if test.isLeader {
			dc.Status.Master.Leader = v1alpha1.MasterMember{Name: leaderPodName, Health: true}
		}

		oldSet := newStatefulSetForDMScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(3)

		scaler, masterControl, pvcIndexer, pvcControl := newFakeMasterScaler()

		if test.hasPVC {
			pvc := newScaleInPVCForStatefulSet(oldSet, v1alpha1.DMMasterMemberType, dc.Name)
			pvcIndexer.Add(pvc)
		}

		masterClient := controller.NewFakeMasterClient(masterControl, dc)

		if test.deleteMemberErr {
			masterClient.AddReaction(dmapi.DeleteMasterActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("error")
			})
		} else {
			masterClient.AddReaction(dmapi.DeleteMasterActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, nil
			})
		}
		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		if test.isLeader {
			masterPeerClient := controller.NewFakeMasterPeerClient(masterControl, dc, leaderPodName)
			masterPeerClient.AddReaction(dmapi.EvictLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, nil
			})
		}

		var membersInfo []*dmapi.MastersInfo
		if test.isMemberStillRemain {
			membersInfo = []*dmapi.MastersInfo{
				{
					Name: fmt.Sprintf("%s-dm-master-%d", dc.GetName(), 4),
				},
			}
		} else {
			membersInfo = []*dmapi.MastersInfo{
				{
					Name: fmt.Sprintf("%s-dm-master-%d", dc.GetName(), 1),
				},
			}
		}
		masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (i interface{}, err error) {
			return membersInfo, nil
		})

		dc.Status.Master.Synced = !test.statusSyncFailed

		err := scaler.ScaleIn(dc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
			if test.isLeader {
				g.Expect(controller.IsRequeueError(err)).To(BeTrue())
			}
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(4))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:                "normal",
			masterUpgrading:     false,
			hasPVC:              true,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 false,
			changed:             true,
			isMemberStillRemain: false,
		},
		{
			name:                "able to scale in while dm-master is upgrading",
			masterUpgrading:     true,
			hasPVC:              true,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 false,
			changed:             true,
			isMemberStillRemain: false,
		},
		{
			name:                "error when delete dm-master",
			hasPVC:              true,
			pvcUpdateErr:        false,
			masterUpgrading:     false,
			deleteMemberErr:     true,
			statusSyncFailed:    false,
			err:                 true,
			changed:             false,
			isMemberStillRemain: false,
		},
		{
			name:                "cache don't have pvc",
			masterUpgrading:     false,
			hasPVC:              false,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 true,
			changed:             false,
			isMemberStillRemain: false,
		},
		{
			name:                "error when update pvc",
			masterUpgrading:     false,
			hasPVC:              true,
			pvcUpdateErr:        true,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 true,
			changed:             false,
			isMemberStillRemain: false,
		},
		{
			name:                "dm-master status sync failed",
			masterUpgrading:     false,
			hasPVC:              true,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    true,
			err:                 true,
			changed:             false,
			isMemberStillRemain: false,
		},
		{
			name:                "delete dm-master success, but get dm-master still remain",
			masterUpgrading:     false,
			hasPVC:              true,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 true,
			changed:             false,
			isMemberStillRemain: true,
		},
		{
			name:                "delete dm-master success, but get dm-master still remain",
			masterUpgrading:     false,
			hasPVC:              true,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 true,
			changed:             false,
			isMemberStillRemain: true,
		},
		{
			name:                "scaled dm-master is leader",
			masterUpgrading:     false,
			hasPVC:              true,
			pvcUpdateErr:        false,
			deleteMemberErr:     false,
			statusSyncFailed:    false,
			err:                 true,
			changed:             false,
			isMemberStillRemain: false,
			isLeader:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func newFakeMasterScaler() (*masterScaler, *dmapi.FakeMasterControl, cache.Indexer, *controller.FakePVCControl) {
	fakeDeps := controller.NewFakeDependencies()
	scaler := &masterScaler{generalScaler{deps: fakeDeps}}
	masterControl := fakeDeps.DMMasterControl.(*dmapi.FakeMasterControl)
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	pvcControl := fakeDeps.PVCControl.(*controller.FakePVCControl)
	return scaler, masterControl, pvcIndexer, pvcControl
}

func newStatefulSetForDMScale() *apps.StatefulSet {
	set := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scaler",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(5),
		},
	}
	return set
}

func normalMasterMember(dc *v1alpha1.DMCluster) {
	dcName := dc.GetName()
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		ordinalPodName(v1alpha1.DMMasterMemberType, dcName, 0): {Health: true},
		ordinalPodName(v1alpha1.DMMasterMemberType, dcName, 1): {Health: true},
		ordinalPodName(v1alpha1.DMMasterMemberType, dcName, 2): {Health: true},
		ordinalPodName(v1alpha1.DMMasterMemberType, dcName, 3): {Health: true},
		ordinalPodName(v1alpha1.DMMasterMemberType, dcName, 4): {Health: true},
	}
}
