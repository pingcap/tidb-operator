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
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	pdUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
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
		PVCNotFound            bool
		expectFn               func(g *GomegaWithT, response *admission.AdmissionResponse)
	}

	testFn := func(test *testcase) {
		t.Log(test.name)

		/**
		init statefulset with 3 replicas
		deletePod := tc-pd-2
		*/

		deletePod := newPDPodForPDPodAdmissionControl()
		ownerStatefulSet := newOwnerStatefulSetForPDPodAdmissionControl()
		tc := newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas)
		kubeCli := kubefake.NewSimpleClientset()

		if test.UpdatePVCErr {

			if test.PVCNotFound {
				kubeCli.PrependReactor("get", "persistentvolumeclaims", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.NewNotFound(action.GetResource().GroupResource(), "name")
				})
			} else {
				kubeCli.PrependReactor("get", "persistentvolumeclaims", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("some errors")
				})
			}
		}

		cli := fake.NewSimpleClientset()
		podAdmissionControl := newPodAdmissionControl(nil, kubeCli, cli)
		pdControl := pdapi.NewFakePDControl(kubeCli)
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

		payload := &admitPayload{
			pod:              deletePod,
			ownerStatefulSet: ownerStatefulSet,
			controller:       tc,
			controllerDesc: controllerDesc{
				name:      tc.Name,
				namespace: tc.Namespace,
				kind:      tc.Kind,
			},
			pdClient: fakePDClient,
		}

		response := podAdmissionControl.admitDeletePdPods(payload)

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
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
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
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(true))
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
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
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
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:                   "normal scale in",
			isMember:               true,
			isDeferDeleting:        false,
			isOutOfOrdinal:         true,
			isStatefulSetUpgrading: false,
			isLeader:               false,
			UpdatePVCErr:           false,
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
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
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(true))
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
			PVCNotFound:            false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:                   "final scale in,update pvc error,pvc not found",
			isMember:               false,
			isDeferDeleting:        true,
			isOutOfOrdinal:         true,
			isStatefulSetUpgrading: false,
			isLeader:               false,
			UpdatePVCErr:           true,
			PVCNotFound:            true,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(true))
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
