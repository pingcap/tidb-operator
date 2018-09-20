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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPDFailoverFailover(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                     string
		update                   func(*v1alpha1.TidbCluster)
		hasPVC                   bool
		hasPV                    bool
		hasPod                   bool
		podWithDeletionTimestamp bool
		delMemberFailed          bool
		delPodFailed             bool
		delPVCFailed             bool
		errExpectFn              func(*GomegaWithT, error)
		expectFn                 func(*v1alpha1.TidbCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		test.update(tc)

		pdFailover, pvcIndexer, pvIndexer, podIndexer, fakePDControl, fakePodControl, fakePVCControl := newFakePDFailover()
		pdClient := controller.NewFakePDClient()
		fakePDControl.SetPDClient(tc, pdClient)

		pdClient.AddReaction(controller.DeleteMemberActionType, func(action *controller.Action) (interface{}, error) {
			if test.delMemberFailed {
				return nil, fmt.Errorf("failed to delete member")
			}
			return nil, nil
		})

		pvc := newPVCForPDFailover(tc, v1alpha1.PDMemberType, 1)
		if test.hasPVC {
			pvcIndexer.Add(pvc)
		}
		if test.hasPV {
			pv := newPVForPDFailover(pvc)
			pvIndexer.Add(pv)
		}
		if test.hasPod {
			pod := newPodForPDFailover(tc, v1alpha1.PDMemberType, 1)
			if test.podWithDeletionTimestamp {
				pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			}
			podIndexer.Add(pod)
		}
		if test.delPodFailed {
			fakePodControl.SetDeletePodError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.delPVCFailed {
			fakePVCControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := pdFailover.Failover(tc)
		test.errExpectFn(g, err)
		test.expectFn(tc)
	}
	tests := []testcase{
		{
			name:   "all members are ready",
			update: allMembersReady,
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name:   "two members are not ready, not in quorum",
			update: twoMembersNotReady,
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "pd cluster is not health")).To(Equal(true))
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name:   "two members are ready and a failure member",
			update: oneFailureMember,
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(1))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(pd1.MemberDeleted).To(Equal(true))
				g.Expect(int(pd1.Replicas)).To(Equal(3))
			},
		},
		{
			name: "has one not ready member, but not exceed deadline",
			update: func(tc *v1alpha1.TidbCluster) {
				oneNotReadyMember(tc)
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1 := tc.Status.PD.Members[pd1Name]
				pd1.LastTransitionTime = metav1.Time{Time: time.Now().Add(-2 * time.Minute)}
				tc.Status.PD.Members[pd1Name] = pd1
			},
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "has one not ready member, and exceed deadline, lastTransitionTime is zero",
			update: func(tc *v1alpha1.TidbCluster) {
				oneNotReadyMember(tc)
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1 := tc.Status.PD.Members[pd1Name]
				pd1.LastTransitionTime = metav1.Time{}
				tc.Status.PD.Members[pd1Name] = pd1
			},
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name:   "has one not ready member, and exceed deadline, don't have PVC, have PV",
			update: oneNotReadyMember,
			hasPVC: true,
			hasPV:  false,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNotNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name:   "has one not ready member, and exceed deadline, have PVC, don't have PV",
			update: oneNotReadyMember,
			hasPVC: true,
			hasPV:  false,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNotNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name:   "has one not ready member, and exceed deadline, return requeue error",
			update: oneNotReadyMember,
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectRequeue,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(pd1.MemberDeleted).To(Equal(false))
				g.Expect(int(pd1.Replicas)).To(Equal(3))
			},
		},
		{
			name:   "has one not ready member, and exceed deadline, don't have PVC, has Pod, delete pod success",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: false,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(pd1.MemberDeleted).To(Equal(true))
				g.Expect(int(pd1.Replicas)).To(Equal(3))
			},
		},
		{
			name:   "has one not ready member, and exceed deadline, don't have PVC, has Pod, delete pod failed",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: false,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             true,
			delPVCFailed:             false,
			errExpectFn:              errExpectNotNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(pd1.MemberDeleted).To(Equal(false))
				g.Expect(int(pd1.Replicas)).To(Equal(3))
			},
		},
		{
			name:   "one failure member, delete member failed",
			update: oneFailureMember,
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          true,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNotNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(false))
			},
		},
		{
			name:   "one failure member, don't have pvc, have pod with deletetimestamp",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: false,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: true,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(true))
			},
		},
		{
			name:   "one failure member, don't have pvc, have pod without deletetimestamp, delete pod success",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: false,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(true))
			},
		},
		{
			name:   "one failure member, don't have pvc, have pod without deletetimestamp, delete pod failed",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: false,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             true,
			delPVCFailed:             false,
			errExpectFn:              errExpectNotNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(false))
			},
		},
		{
			name:   "one failure member, don't have pv",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: true,
			hasPV:  false,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(true))
			},
		},
		{
			name: "one failure members, pv uid changed",
			update: func(tc *v1alpha1.TidbCluster) {
				oneNotReadyMemberAndAFailureMember(tc)
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pd1 := tc.Status.PD.FailureMembers[pd1Name]
				pd1.PVUID = "xxx"
				tc.Status.PD.FailureMembers[pd1Name] = pd1
			},
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(true))
			},
		},
		{
			name:   "one failure members, pvc delete fail",
			update: oneNotReadyMemberAndAFailureMember,
			hasPVC: true,
			hasPV:  true,
			hasPod: true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             true,
			errExpectFn:              errExpectNotNil,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				pd1Name := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failMember, ok := tc.Status.PD.FailureMembers[pd1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(failMember.MemberDeleted).To(Equal(false))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDFailoverRecovery(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		expectFn func(*v1alpha1.TidbCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		test.update(tc)

		pdFailover, _, _, _, _, _, _ := newFakePDFailover()
		pdFailover.Recover(tc)
		test.expectFn(tc)
	}
	tests := []testcase{
		{
			name: "two failure member, user don't modify the replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				twoFailureMembers(tc)
				tc.Spec.PD.Replicas = 3
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user modify the replicas to 4",
			update: func(tc *v1alpha1.TidbCluster) {
				twoFailureMembers(tc)
				tc.Spec.PD.Replicas = 4
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(4))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user increase the replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				twoFailureMembers(tc)
				tc.Spec.PD.Replicas = 7
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(7))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user decrease the replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				twoFailureMembers(tc)
				tc.Spec.PD.Replicas = 1
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(1))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "one failure member, user don't modify the replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				oneFailureMember(tc)
				tc.Spec.PD.Replicas = 3
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user increase the replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				oneFailureMember(tc)
				tc.Spec.PD.Replicas = 5
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(5))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user decrease the replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				oneFailureMember(tc)
				tc.Spec.PD.Replicas = 1
			},
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.PD.Replicas)).To(Equal(1))
				g.Expect(len(tc.Status.PD.FailureMembers)).To(Equal(0))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePDFailover() (*pdFailover, cache.Indexer, cache.Indexer, cache.Indexer, *controller.FakePDControl, *controller.FakePodControl, *controller.FakePVCControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	pdControl := controller.NewFakePDControl()
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	podInformer := kubeInformerFactory.Core().V1().Pods()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()
	podControl := controller.NewFakePodControl(podInformer)
	pvcControl := controller.NewFakePVCControl(pvcInformer)

	return &pdFailover{
			cli,
			pdControl,
			5 * time.Minute,
			podInformer.Lister(),
			podControl,
			pvcInformer.Lister(),
			pvcControl,
			pvInformer.Lister()},
		pvcInformer.Informer().GetIndexer(),
		pvInformer.Informer().GetIndexer(),
		podInformer.Informer().GetIndexer(),
		pdControl, podControl, pvcControl
}

func oneFailureMember(tc *v1alpha1.TidbCluster) {
	pd0 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0)
	pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
	pd2 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 2)
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		pd0: {Name: pd0, ID: "0", Health: true},
		pd2: {Name: pd2, ID: "2", Health: true},
	}
	tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{
		pd1: {Replicas: 3},
	}
}

func twoFailureMembers(tc *v1alpha1.TidbCluster) {
	pd0 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0)
	pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
	pd2 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 2)
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		pd2: {Name: pd2, ID: "2", Health: true},
	}
	tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{
		pd0: {Replicas: 3},
		pd1: {Replicas: 4},
	}
}

func oneNotReadyMember(tc *v1alpha1.TidbCluster) {
	pd0 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0)
	pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
	pd2 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 2)
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		pd0: {Name: pd0, ID: "0", Health: true},
		pd1: {Name: pd1, ID: "1", Health: false, LastTransitionTime: metav1.Time{Time: time.Now().Add(-10 * time.Minute)}},
		pd2: {Name: pd2, ID: "2", Health: true},
	}
}

func oneNotReadyMemberAndAFailureMember(tc *v1alpha1.TidbCluster) {
	pd0 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0)
	pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
	pd2 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 2)
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		pd0: {Name: pd0, ID: "0", Health: true},
		pd1: {Name: pd1, ID: "1", Health: false, LastTransitionTime: metav1.Time{Time: time.Now().Add(-10 * time.Minute)}},
		pd2: {Name: pd2, ID: "2", Health: true},
	}
	tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{
		pd1: {Replicas: 3, PVUID: "uid-1"},
	}
}

func allMembersReady(tc *v1alpha1.TidbCluster) {
	pd0 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0)
	pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
	pd2 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 2)
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		pd0: {Name: pd0, ID: "0", Health: true},
		pd1: {Name: pd1, ID: "1", Health: true},
		pd2: {Name: pd2, ID: "2", Health: true},
	}
}

func twoMembersNotReady(tc *v1alpha1.TidbCluster) {
	pd0 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0)
	pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
	pd2 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 2)
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		pd0: {Name: pd0, ID: "0", Health: false},
		pd1: {Name: pd1, ID: "1", Health: false},
		pd2: {Name: pd2, ID: "2", Health: true},
	}
}

func errExpectNil(g *GomegaWithT, err error) {
	g.Expect(err).NotTo(HaveOccurred())
}

func errExpectNotNil(g *GomegaWithT, err error) {
	g.Expect(err).To(HaveOccurred())
}

func newPVCForPDFailover(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, ordinal int32) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ordinalPVCName(memberType, controller.PDMemberName(tc.GetName()), ordinal),
			Namespace: metav1.NamespaceDefault,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: fmt.Sprintf("pv-%d", ordinal),
		},
	}
}

func newPVForPDFailover(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvc.Spec.VolumeName,
			UID:  "uid-1",
		},
	}
}

func newPodForPDFailover(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, ordinal int32) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ordinalPodName(memberType, tc.GetName(), ordinal),
			Namespace: metav1.NamespaceDefault,
		},
	}
}
