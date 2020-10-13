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
	"sort"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/pkg/dmapi"

	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
)

func TestMasterFailoverFailover(t *testing.T) {
	g := NewGomegaWithT(t)

	recorder := record.NewFakeRecorder(100)
	type testcase struct {
		name                     string
		update                   func(*v1alpha1.DMCluster)
		maxFailoverCount         int32
		hasPVC                   bool
		hasPod                   bool
		podWithDeletionTimestamp bool
		pvcWithDeletionTimestamp bool
		delMemberFailed          bool
		delPodFailed             bool
		delPVCFailed             bool
		statusSyncFailed         bool
		errExpectFn              func(*GomegaWithT, error)
		expectFn                 func(*v1alpha1.DMCluster, *masterFailover)
	}

	tests := []testcase{
		{
			name:                     "all dm-master members are ready",
			update:                   allMasterMembersReady,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(0))
			},
		},
		{
			name:                     "dm-master status sync failed",
			update:                   allMasterMembersReady,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         true,
			errExpectFn:              errExpectNotNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(0))
			},
		},
		{
			name:                     "two dm-master members are not ready, not in quorum",
			update:                   twoMasterMembersNotReady,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "dm-master cluster is not healthy")).To(Equal(true))
			},
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
				events := collectEvents(recorder.Events)
				sort.Strings(events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-0(0) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
			},
		},
		{
			name:                     "two dm-master members are ready and a failure dm-master member",
			update:                   oneFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(1))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(true))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("[default/test-dm-master-1] deleted from dmcluster"))
			},
		},
		{
			name: "has one not ready dm-master member, but not exceed deadline",
			update: func(dc *v1alpha1.DMCluster) {
				oneNotReadyMasterMember(dc)
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1 := dc.Status.Master.Members[master1Name]
				master1.LastTransitionTime = metav1.Time{Time: time.Now().Add(-2 * time.Minute)}
				dc.Status.Master.Members[master1Name] = master1
			},
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
			},
		},
		{
			name: "has one not ready dm-master member, and exceed deadline, lastTransitionTime is zero",
			update: func(dc *v1alpha1.DMCluster) {
				oneNotReadyMasterMember(dc)
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1 := dc.Status.Master.Members[master1Name]
				master1.LastTransitionTime = metav1.Time{}
				dc.Status.Master.Members[master1Name] = master1
			},
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
			},
		},
		{
			name:                     "has one not ready dm-master member, don't have pvc",
			update:                   oneNotReadyMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   false,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "persistentvolumeclaim \"dm-master-test-dm-master-1\" not found")).To(Equal(true))
			},
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
			},
		},
		{
			name:                     "has one not ready dm-master member",
			update:                   oneNotReadyMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "marking Pod: default/test-dm-master-1 dm-master member: test-dm-master-1 as failure")).To(Equal(true))
			},
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(1))
				failureMembers := dc.Status.Master.FailureMembers["test-dm-master-1"]
				g.Expect(failureMembers.PodName).To(Equal("test-dm-master-1"))
				g.Expect(failureMembers.MemberID).To(Equal("12891273174085095651"))
				g.Expect(string(failureMembers.PVCUID)).To(Equal("pvc-1-uid"))
				g.Expect(failureMembers.MemberDeleted).To(BeFalse())
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("Unhealthy dm-master pod[test-dm-master-1] is unhealthy, msg:dm-master member[12891273174085095651] is unhealthy"))
			},
		},
		{
			name:                     "has one not ready dm-master member but maxFailoverCount is 0",
			update:                   oneNotReadyMasterMember,
			maxFailoverCount:         0,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
			},
		},
		{
			name:                     "has one not ready dm-master member, and exceed deadline, don't have PVC, has Pod, delete pod success",
			update:                   oneNotReadyMasterMemberAndAFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   false,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(true))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("[default/test-dm-master-1] deleted from dmcluster"))
			},
		},
		{
			name:                     "has one not dm-master ready member, and exceed deadline, don't have PVC, has Pod, delete dm-master member failed",
			update:                   oneNotReadyMasterMemberAndAFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   false,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          true,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to delete member")).To(Equal(true))
			},
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(false))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
			},
		},
		{
			name:                     "has one not ready dm-master member, and exceed deadline, don't have PVC, has Pod, delete pod failed",
			update:                   oneNotReadyMasterMemberAndAFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   false,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             true,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "delete pod: API server failed")).To(Equal(true))
			},
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(false))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("[default/test-dm-master-1] deleted from dmcluster"))
			},
		},
		{
			name:                     "has one not ready dm-master member, and exceed deadline, has Pod, delete pvc failed",
			update:                   oneNotReadyMasterMemberAndAFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             true,
			statusSyncFailed:         false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "delete pvc: API server failed")).To(Equal(true))
			},
			expectFn: func(dc *v1alpha1.DMCluster, _ *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(false))
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("[default/test-dm-master-1] deleted from dmcluster"))
			},
		},
		{
			name:                     "has one not ready dm-master member, and exceed deadline, has Pod with deletion timestamp",
			update:                   oneNotReadyMasterMemberAndAFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: true,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, mf *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				pvcName := ordinalPVCName(v1alpha1.DMMasterMemberType, controller.DMMasterMemberName(dc.GetName()), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(true))
				_, err := mf.deps.PodLister.Pods(metav1.NamespaceDefault).Get(master1Name)
				g.Expect(err).NotTo(HaveOccurred())
				_, err = mf.deps.PVCLister.PersistentVolumeClaims(metav1.NamespaceDefault).Get(pvcName)
				g.Expect(err).To(HaveOccurred())
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("[default/test-dm-master-1] deleted from dmcluster"))
			},
		},
		{
			name:                     "has one not ready dm-master member, and exceed deadline, has PVC with deletion timestamp",
			update:                   oneNotReadyMasterMemberAndAFailureMasterMember,
			maxFailoverCount:         3,
			hasPVC:                   true,
			hasPod:                   true,
			podWithDeletionTimestamp: false,
			pvcWithDeletionTimestamp: true,
			delMemberFailed:          false,
			delPodFailed:             false,
			delPVCFailed:             false,
			statusSyncFailed:         false,
			errExpectFn:              errExpectNil,
			expectFn: func(dc *v1alpha1.DMCluster, mf *masterFailover) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				master1Name := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
				pvcName := ordinalPVCName(v1alpha1.DMMasterMemberType, controller.DMMasterMemberName(dc.GetName()), 1)
				master1, ok := dc.Status.Master.FailureMembers[master1Name]
				g.Expect(ok).To(Equal(true))
				g.Expect(master1.MemberDeleted).To(Equal(true))
				_, err := mf.deps.PodLister.Pods(metav1.NamespaceDefault).Get(master1Name)
				g.Expect(err).To(HaveOccurred())
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
				_, err = mf.deps.PVCLister.PersistentVolumeClaims(metav1.NamespaceDefault).Get(pvcName)
				g.Expect(err).NotTo(HaveOccurred())
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(2))
				g.Expect(events[0]).To(ContainSubstring("test-dm-master-1(12891273174085095651) is unhealthy"))
				g.Expect(events[1]).To(ContainSubstring("[default/test-dm-master-1] deleted from dmcluster"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dc := newDMClusterForMaster()
			dc.Spec.Master.MaxFailoverCount = pointer.Int32Ptr(test.maxFailoverCount)
			test.update(dc)

			masterFailover, pvcIndexer, podIndexer, fakeMasterControl, fakePodControl, fakePVCControl := newFakeMasterFailover()
			masterFailover.deps.Recorder = recorder
			masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)

			masterClient.AddReaction(dmapi.DeleteMasterActionType, func(action *dmapi.Action) (interface{}, error) {
				if test.delMemberFailed {
					return nil, fmt.Errorf("failed to delete member")
				}
				return nil, nil
			})

			if test.hasPVC {
				pvc := newPVCForMasterFailover(dc, v1alpha1.DMMasterMemberType, 1)
				if test.pvcWithDeletionTimestamp {
					pvc.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				}
				pvcIndexer.Add(pvc)
			}
			if test.hasPod {
				pod := newPodForMasterFailover(dc, v1alpha1.DMMasterMemberType, 1)
				if test.podWithDeletionTimestamp {
					pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				}
				podIndexer.Add(pod)
			}
			if test.delPodFailed {
				fakePodControl.SetDeletePodError(errors.NewInternalError(fmt.Errorf("delete pod: API server failed")), 0)
			}
			if test.delPVCFailed {
				fakePVCControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("delete pvc: API server failed")), 0)
			}

			dc.Status.Master.Synced = !test.statusSyncFailed

			err := masterFailover.Failover(dc)
			test.errExpectFn(g, err)
			test.expectFn(dc, masterFailover)
		})
	}
}

func TestMasterFailoverRecovery(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*v1alpha1.DMCluster)
		expectFn func(*v1alpha1.DMCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		dc := newDMClusterForMaster()
		test.update(dc)

		masterFailover, _, _, _, _, _ := newFakeMasterFailover()
		masterFailover.Recover(dc)
		test.expectFn(dc)
	}
	tests := []testcase{
		{
			name: "two failure member, user don't modify the replicas",
			update: func(dc *v1alpha1.DMCluster) {
				twoFailureMasterMembers(dc)
				dc.Spec.Master.Replicas = 3
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user modify the replicas to 4",
			update: func(dc *v1alpha1.DMCluster) {
				twoFailureMasterMembers(dc)
				dc.Spec.Master.Replicas = 4
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(4))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user increase the replicas",
			update: func(dc *v1alpha1.DMCluster) {
				twoFailureMasterMembers(dc)
				dc.Spec.Master.Replicas = 7
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(7))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user decrease the replicas",
			update: func(dc *v1alpha1.DMCluster) {
				twoFailureMasterMembers(dc)
				dc.Spec.Master.Replicas = 1
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(1))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "one failure member, user don't modify the replicas",
			update: func(dc *v1alpha1.DMCluster) {
				oneFailureMasterMember(dc)
				dc.Spec.Master.Replicas = 3
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(3))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user increase the replicas",
			update: func(dc *v1alpha1.DMCluster) {
				oneFailureMasterMember(dc)
				dc.Spec.Master.Replicas = 5
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(5))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure member, user decrease the replicas",
			update: func(dc *v1alpha1.DMCluster) {
				oneFailureMasterMember(dc)
				dc.Spec.Master.Replicas = 1
			},
			expectFn: func(dc *v1alpha1.DMCluster) {
				g.Expect(int(dc.Spec.Master.Replicas)).To(Equal(1))
				g.Expect(len(dc.Status.Master.FailureMembers)).To(Equal(0))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeMasterFailover() (*masterFailover, cache.Indexer, cache.Indexer, *dmapi.FakeMasterControl, *controller.FakePodControl, *controller.FakePVCControl) {
	fakeDeps := controller.NewFakeDependencies()
	failover := &masterFailover{deps: fakeDeps}
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	masterControl := fakeDeps.DMMasterControl.(*dmapi.FakeMasterControl)
	podControl := fakeDeps.PodControl.(*controller.FakePodControl)
	pvcControl := fakeDeps.PVCControl.(*controller.FakePVCControl)
	return failover, pvcIndexer, podIndexer, masterControl, podControl, pvcControl
}

func oneFailureMasterMember(dc *v1alpha1.DMCluster) {
	master0 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0)
	master1 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
	master2 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 2)
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		master0: {Name: master0, ID: "0", Health: true},
		master2: {Name: master2, ID: "2", Health: true},
	}
	dc.Status.Master.FailureMembers = map[string]v1alpha1.MasterFailureMember{
		master1: {PodName: master1, PVCUID: "pvc-1-uid", MemberID: "12891273174085095651"},
	}
}

func twoFailureMasterMembers(dc *v1alpha1.DMCluster) {
	master0 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0)
	master1 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
	master2 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 2)
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		master2: {Name: master2, ID: "2", Health: true},
	}
	dc.Status.Master.FailureMembers = map[string]v1alpha1.MasterFailureMember{
		master0: {PodName: master0},
		master1: {PodName: master1},
	}
}

func oneNotReadyMasterMember(dc *v1alpha1.DMCluster) {
	master0 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0)
	master1 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
	master2 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 2)
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		master0: {Name: master0, ID: "0", Health: true},
		master1: {Name: master1, ID: "12891273174085095651", Health: false, LastTransitionTime: metav1.Time{Time: time.Now().Add(-10 * time.Minute)}},
		master2: {Name: master2, ID: "2", Health: true},
	}
}

func oneNotReadyMasterMemberAndAFailureMasterMember(dc *v1alpha1.DMCluster) {
	master0 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0)
	master1 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
	master2 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 2)
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		master0: {Name: master0, ID: "0", Health: true},
		master1: {Name: master1, ID: "12891273174085095651", Health: false, LastTransitionTime: metav1.Time{Time: time.Now().Add(-10 * time.Minute)}},
		master2: {Name: master2, ID: "2", Health: true},
	}
	dc.Status.Master.FailureMembers = map[string]v1alpha1.MasterFailureMember{
		master1: {PodName: master1, PVCUID: "pvc-1-uid", MemberID: "12891273174085095651"},
	}
}

func allMasterMembersReady(dc *v1alpha1.DMCluster) {
	master0 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0)
	master1 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
	master2 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 2)
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		master0: {Name: master0, ID: "0", Health: true},
		master1: {Name: master1, ID: "12891273174085095651", Health: true},
		master2: {Name: master2, ID: "2", Health: true},
	}
}

func twoMasterMembersNotReady(dc *v1alpha1.DMCluster) {
	master0 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0)
	master1 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 1)
	master2 := ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 2)
	dc.Status.Master.Members = map[string]v1alpha1.MasterMember{
		master0: {Name: master0, ID: "0", Health: false},
		master1: {Name: master1, ID: "12891273174085095651", Health: false},
		master2: {Name: master2, ID: "2", Health: true},
	}
}

func newPVCForMasterFailover(dc *v1alpha1.DMCluster, memberType v1alpha1.MemberType, ordinal int32) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ordinalPVCName(memberType, controller.DMMasterMemberName(dc.GetName()), ordinal),
			Namespace: metav1.NamespaceDefault,
			UID:       types.UID("pvc-1-uid"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: fmt.Sprintf("pv-%d", ordinal),
		},
	}
}

func newPodForMasterFailover(dc *v1alpha1.DMCluster, memberType v1alpha1.MemberType, ordinal int32) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ordinalPodName(memberType, dc.GetName(), ordinal),
			Namespace: metav1.NamespaceDefault,
		},
	}
}
