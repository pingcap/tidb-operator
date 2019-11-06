// Copyright 2019 PingCAP, Inc.
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

package pod

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	pdUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deletePDPodOrdinal = int32(2)
	pdStsName          = tcName + "-pd"
	deletePDPodName    = pdUtils.PdPodName(tcName, deletePDPodOrdinal)
)

func TestPDDeleterDelete(t *testing.T) {

	g := NewGomegaWithT(t)

	type testcase struct {
		name                   string
		isMember               bool
		isDeferDeleting        bool
		isOutOfOrdinal         bool
		isStatefulSetUpgrading bool
		isLeader               bool
		UpdatePVCErr           bool
		expectFn               func(g *GomegaWithT, response *v1beta1.AdmissionResponse)
	}

	testFn := func(test *testcase) {
		t.Log(test.name)

		/**
		init statefulset with 3 replicas
		deletePod := tc-pd-2
		*/

		deletePod := newPDPodForPDPodAdmissionControl()
		ownerStatefulSet := newOwnerStatefulSetForPDPodAdmissionControl()
		tc := newTidbClusterForPodAdmissionControl()
		pvc := newPVCForDeletePod()

		podAdmissionControl, fakePVCControl, pvcIndexer, _ := newPodAdmissionControl()
		pdControl := pdapi.NewFakePDControl()
		fakePDClient := controller.NewFakePDClient(pdControl, tc)

		if test.isMember {
			fakePDClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
				membersInfo := &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							Name: pdUtils.PdPodName(tcName, 3),
						},
						{
							Name: pdUtils.PdPodName(tcName, 2),
						},
						{
							Name: pdUtils.PdPodName(tcName, 1),
						},
						{
							Name: pdUtils.PdPodName(tcName, 0),
						},
					},
				}
				return membersInfo, nil
			})
		} else {
			fakePDClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
				membersInfo := &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							Name: pdUtils.PdPodName(tcName, 1),
						},
						{
							Name: pdUtils.PdPodName(tcName, 0),
						},
					},
				}
				return membersInfo, nil
			})
		}

		if test.isDeferDeleting {
			deletePod.Annotations = map[string]string{
				label.AnnPDDeferDeleting: "123",
			}
		}

		if test.isOutOfOrdinal {
			ownerStatefulSet.Spec.Replicas = func() *int32 { a := int32(2); return &a }()
			pvcIndexer.Add(pvc)
		}

		if !test.isMember && test.isOutOfOrdinal {

			if test.UpdatePVCErr {
				fakePVCControl.SetUpdatePVCError(fmt.Errorf("update pvc error"), 0)
			} else {
				fakePVCControl.SetUpdatePVCError(nil, 0)
			}
		}

		if test.isStatefulSetUpgrading {
			ownerStatefulSet.Status.UpdateRevision = "2"
		}

		if test.isLeader {
			fakePDClient.AddReaction(pdapi.GetPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				leader := pdpb.Member{
					Name: pdUtils.PdPodName(tcName, 3),
				}
				return &leader, nil
			})
		} else {
			fakePDClient.AddReaction(pdapi.GetPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				leader := pdpb.Member{
					Name: pdUtils.PdPodName(tcName, 0),
				}
				return &leader, nil
			})
			fakePDClient.AddReaction(pdapi.TransferPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, nil
			})
		}

		response := podAdmissionControl.admitDeletePdPods(deletePod, ownerStatefulSet, tc, fakePDClient)

		test.expectFn(g, response)
	}

	tests := []testcase{
		{
			name:                   "first normal upgraded",
			isMember:               true,
			isDeferDeleting:        false,
			isOutOfOrdinal:         false,
			isStatefulSetUpgrading: true,
			isLeader:               false,
			UpdatePVCErr:           false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
		{
			name:                   "final normal upgraded",
			isMember:               false,
			isDeferDeleting:        true,
			isOutOfOrdinal:         false,
			isStatefulSetUpgrading: true,
			isLeader:               false,
			UpdatePVCErr:           false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
		{
			name:                   "nonMember pd pod without deferDeleting Annotation",
			isMember:               false,
			isDeferDeleting:        false,
			isOutOfOrdinal:         false,
			isStatefulSetUpgrading: true,
			isLeader:               false,
			UpdatePVCErr:           false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
		{
			name:                   "leader Upgraded",
			isMember:               true,
			isDeferDeleting:        false,
			isOutOfOrdinal:         false,
			isStatefulSetUpgrading: true,
			isLeader:               true,
			UpdatePVCErr:           false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
		{
			name:                   "normal scale in",
			isMember:               true,
			isDeferDeleting:        false,
			isOutOfOrdinal:         true,
			isStatefulSetUpgrading: false,
			isLeader:               false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
		{
			name:                   "final scale in",
			isMember:               false,
			isDeferDeleting:        true,
			isOutOfOrdinal:         true,
			isStatefulSetUpgrading: false,
			isLeader:               false,
			UpdatePVCErr:           false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
		{
			name:                   "final scale in,update pvc error",
			isMember:               false,
			isDeferDeleting:        true,
			isOutOfOrdinal:         true,
			isStatefulSetUpgrading: false,
			isLeader:               false,
			UpdatePVCErr:           true,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, false)
				g.Expect(response.Result.Message, "update pvc error")
			},
		},
	}
	for _, test := range tests {
		testFn(&test)
	}

}

func newPDPodForPDPodAdmissionControl() *corev1.Pod {
	pod := corev1.Pod{}
	pod.Name = deletePDPodName
	pod.Labels = map[string]string{
		label.ComponentLabelKey: label.PDLabelVal,
	}
	pod.Namespace = namespace
	return &pod
}

func newOwnerStatefulSetForPDPodAdmissionControl() *apps.StatefulSet {
	sts := apps.StatefulSet{}
	sts.Spec.Replicas = func() *int32 { a := int32(deletePDPodOrdinal); return &a }()
	sts.Name = pdStsName
	sts.Namespace = namespace
	sts.Status.CurrentRevision = "1"
	sts.Status.UpdateRevision = "1"
	return &sts
}

func newPVCForDeletePod() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorUtils.OrdinalPVCName(v1alpha1.PDMemberType, pdStsName, deletePDPodOrdinal),
			Namespace: namespace,
		},
	}
}
