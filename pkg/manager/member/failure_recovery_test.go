// Copyright 2022 PingCAP, Inc.
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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	testNode1Name = "k8s-node-1"
)

type testPodPvcParams struct {
	podWithDeletionTimestamp bool
	pvcWithDeletionTimestamp bool
}

func TestRestartPodOnHostDown(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                string
		detectNodeFailure   bool
		autoFailureRecovery bool
		hostDown            bool
		podCreatedAt        time.Time
		hasPvcUIDSet        bool
		lastTransitionTime  time.Time
		errExpectFn         func(*GomegaWithT, error)
		expectFn            func(*v1alpha1.TidbCluster, cache.Indexer)
	}
	timeNow := time.Now()
	time1hAgo := timeNow.Add(-time.Hour)
	tests := []testcase{
		{
			name:        "detectNodeFailure is not set, no action",
			errExpectFn: errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer) {
				pd := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failureMember, exists := tc.Status.PD.FailureMembers[pd]
				g.Expect(exists).To(Equal(true))
				g.Expect(failureMember.HostDown).To(Equal(false))
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
			},
		},
		{
			name:              "hostDown is set, and autoFailureRecovery is not set, no action",
			detectNodeFailure: true,
			hostDown:          true,
			podCreatedAt:      time1hAgo,
			errExpectFn:       errExpectNil,
			expectFn: func(_ *v1alpha1.TidbCluster, podIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
			},
		},
		{
			name:              "hostDown is not set, and pvcUIDSet is not set, no action",
			detectNodeFailure: true,
			podCreatedAt:      time1hAgo,
			errExpectFn:       errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer) {
				pd := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failureMember, exists := tc.Status.PD.FailureMembers[pd]
				g.Expect(exists).To(Equal(true))
				g.Expect(failureMember.HostDown).To(Equal(false))
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
			},
		},
		{
			name:               "hostDown is not set, and store down for more than hard recovery period, set hostDown",
			detectNodeFailure:  true,
			podCreatedAt:       time1hAgo,
			hasPvcUIDSet:       true,
			lastTransitionTime: time.Now().Add(-2*time.Hour - time.Minute), // before PodHardRecoveryPeriod
			errExpectFn:        getErrContainsSubstring("Host down reason: " + hdReasonStoreDownTimeExceeded),
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
				pd := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				failureMember, exists := tc.Status.PD.FailureMembers[pd]
				g.Expect(exists).To(Equal(true))
				g.Expect(failureMember.HostDown).To(Equal(true))
			},
		},
		{
			name:                "hostDown is set, and pod not restarted once, restart pod",
			detectNodeFailure:   true,
			hostDown:            true,
			autoFailureRecovery: true,
			podCreatedAt:        time1hAgo,
			hasPvcUIDSet:        true,
			errExpectFn:         errExpectIgnoreError,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).NotTo(ContainElement("default/test-pd-1"))
			},
		},
		{
			name:                "hostDown is set, and pod restarted once, no action",
			detectNodeFailure:   true,
			hostDown:            true,
			autoFailureRecovery: true,
			podCreatedAt:        timeNow.Add(-time.Minute),
			hasPvcUIDSet:        true,
			errExpectFn:         errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := newTidbClusterWithPDFailureMember(test.hasPvcUIDSet, test.hostDown)
			if test.autoFailureRecovery {
				tc.Annotations = map[string]string{annAutoFailureRecovery: "true"}
			}
			if !test.lastTransitionTime.IsZero() {
				pd1 := ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1)
				pdMem1 := tc.Status.PD.Members[pd1]
				pdMem1.LastTransitionTime = metav1.NewTime(test.lastTransitionTime)
				tc.Status.PD.Members[pd1] = pdMem1
			}
			deps, pvcIndexer, podIndexer, _ := newFakeDependenciesForFailover(test.detectNodeFailure)

			pod, _ := getTestPDPodAndPvcs(pvcIndexer, podIndexer, tc, testPodPvcParams{})
			pod.CreationTimestamp = metav1.NewTime(test.podCreatedAt)

			failureRecovery := commonStatefulFailureRecovery{
				deps:                deps,
				failureObjectAccess: &pdFailureMemberAccess{},
			}
			err := failureRecovery.RestartPodOnHostDown(tc)
			test.errExpectFn(g, err)
			test.expectFn(tc, podIndexer)
		})
	}
}

func TestGetNodeAvailabilityStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name           string
		podPhase       corev1.PodPhase
		podConditions  []corev1.PodCondition
		nodeConditions []corev1.NodeCondition
		errExpectFn    func(*GomegaWithT, error)
		expectFn       func(NodeAvailabilityStatus)
	}
	tests := []testcase{
		{
			name:        "pod in unknown, node unavailable is true",
			podPhase:    corev1.PodUnknown,
			expectFn:    getNAStatusExpectFn(g, true, false),
			errExpectFn: errExpectNil,
		},
		{
			name:        "pod in unsupported phase, node unavailable is false",
			podPhase:    corev1.PodPending,
			expectFn:    getNAStatusExpectFn(g, false, false),
			errExpectFn: errExpectNil,
		},
		{
			name:          "pod ready, node unavailable is false",
			podPhase:      corev1.PodRunning,
			podConditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			expectFn:      getNAStatusExpectFn(g, false, false),
			errExpectFn:   errExpectNil,
		},
		{
			name:           "pod not ready, and node ready, node unavailable is false",
			podPhase:       corev1.PodRunning,
			podConditions:  []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}},
			nodeConditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			expectFn:       getNAStatusExpectFn(g, false, false),
			errExpectFn:    errExpectNil,
		},
		{
			name:           "pod not ready, and node not ready, node unavailable is true",
			podPhase:       corev1.PodRunning,
			podConditions:  []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}},
			nodeConditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}},
			expectFn:       getNAStatusExpectFn(g, true, false),
			errExpectFn:    errExpectNil,
		},
		{
			name:           "pod not ready, and node in unknown, node unavailable is true",
			podPhase:       corev1.PodRunning,
			podConditions:  []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}},
			nodeConditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionUnknown}},
			expectFn:       getNAStatusExpectFn(g, true, false),
			errExpectFn:    errExpectNil,
		},
		{
			name:          "pod ready, node ready, and ro-disk true, read-only disk found is true",
			podPhase:      corev1.PodRunning,
			podConditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			nodeConditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: nodeCondRODiskFound, Status: corev1.ConditionTrue}},
			expectFn:    getNAStatusExpectFn(g, false, true),
			errExpectFn: errExpectNil,
		},
		{
			name:          "pod ready, node ready, and ro-disk false, read-only disk found is false",
			podPhase:      corev1.PodRunning,
			podConditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			nodeConditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: nodeCondRODiskFound, Status: corev1.ConditionFalse}},
			expectFn:    getNAStatusExpectFn(g, false, false),
			errExpectFn: errExpectNil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			}
			deps, _, _, nodeIndexer := newFakeDependenciesForFailover(true)
			pod := getTestPodWithConditions(tc, test.podPhase, test.podConditions)
			nodeIndexer.Add(getTestNodeWithConditions(test.nodeConditions))
			failureRecovery := commonStatefulFailureRecovery{
				deps:                deps,
				failureObjectAccess: &pdFailureMemberAccess{},
			}
			naStatus, err := failureRecovery.getNodeAvailabilityStatus(pod)
			test.expectFn(naStatus)
			test.errExpectFn(g, err)
		})
	}
}

func TestCanDoCleanUpNow(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name          string
		podCreationTS time.Time
		expectedCheck bool
	}
	tests := []testcase{
		{
			name:          "pod restarted before gap deadline",
			podCreationTS: time.Now().Add(-restartToDeleteStoreGap + time.Second),
			expectedCheck: false,
		},
		{
			name:          "pod restarted after gap deadline",
			podCreationTS: time.Now().Add(-restartToDeleteStoreGap - time.Second),
			expectedCheck: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := newTidbClusterWithPDFailureMember(true, true)
			deps, _, podIndexer, _ := newFakeDependenciesForFailover(true)
			pod := newPodForFailover(tc, v1alpha1.PDMemberType, 1)
			pod.CreationTimestamp = metav1.NewTime(test.podCreationTS)
			podIndexer.Add(pod)
			failureRecovery := commonStatefulFailureRecovery{
				deps:                deps,
				failureObjectAccess: &pdFailureMemberAccess{},
			}
			check := failureRecovery.canDoCleanUpNow(tc, ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1))
			g.Expect(check).To(Equal(test.expectedCheck))
		})
	}
}

func TestDeletePodAndPvcs(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		podWithDeletionTimestamp bool
		pvcWithDeletionTimestamp bool
		name                     string
		oldPvcUIDSet             bool
		errExpectFn              func(*GomegaWithT, error)
		expectFn                 func(*v1alpha1.TidbCluster, cache.Indexer, cache.Indexer)
	}
	tests := []testcase{
		{
			name:        "pod has no deletion ts, and pvc has no deletion ts, both pod and pvc is deleted",
			errExpectFn: errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).NotTo(ContainElement("default/test-pd-1"))
				g.Expect(pvcIndexer.ListKeys()).NotTo(ContainElement("default/pd-test-pd-1-1"))
			},
		},
		{
			name:                     "pod has no deletion ts, and pvc has deletion ts, pod is deleted",
			pvcWithDeletionTimestamp: true,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).NotTo(ContainElement("default/test-pd-1"))
				g.Expect(pvcIndexer.ListKeys()).To(ContainElement("default/pd-test-pd-1-1"))
			},
		},
		{
			name:                     "pod has deletion ts, and pvc has no deletion ts, pvc is deleted",
			podWithDeletionTimestamp: true,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
				g.Expect(pvcIndexer.ListKeys()).NotTo(ContainElement("default/pd-test-pd-1-1"))
			},
		},
		{
			name:                     "pod has deletion ts, and pvc has deletion ts, none is deleted",
			podWithDeletionTimestamp: true,
			pvcWithDeletionTimestamp: true,
			errExpectFn:              errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-pd-1"))
				g.Expect(pvcIndexer.ListKeys()).To(ContainElement("default/pd-test-pd-1-1"))
			},
		},
		{
			name:         "pod has no deletion ts, pvc has no deletion ts, and PVCUIDSet has old pvcUID, only pod is deleted",
			oldPvcUIDSet: true,
			errExpectFn:  errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				g.Expect(podIndexer.ListKeys()).NotTo(ContainElement("default/test-pd-1"))
				g.Expect(pvcIndexer.ListKeys()).To(ContainElement("default/pd-test-pd-1-1"))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := newTidbClusterWithPDFailureMember(true, true)
			deps, pvcIndexer, podIndexer, _ := newFakeDependenciesForFailover(true)
			_, pvcs := getTestPDPodAndPvcs(pvcIndexer, podIndexer, tc, testPodPvcParams{
				podWithDeletionTimestamp: test.podWithDeletionTimestamp,
				pvcWithDeletionTimestamp: test.pvcWithDeletionTimestamp})
			if test.oldPvcUIDSet {
				pvcs[0].UID = pvcs[0].UID + "-new"
			}

			failureRecovery := commonStatefulFailureRecovery{
				deps:                deps,
				failureObjectAccess: &pdFailureMemberAccess{},
			}
			err := failureRecovery.deletePodAndPvcs(tc, ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 1))
			test.errExpectFn(g, err)
			test.expectFn(tc, podIndexer, pvcIndexer)
		})
	}
}

func newFakeDependenciesForFailover(detectNodeFailure bool) (*controller.Dependencies, cache.Indexer, cache.Indexer, cache.Indexer) {
	fakeDeps := controller.NewFakeDependencies()
	fakeDeps.CLIConfig.DetectNodeFailure = detectNodeFailure
	fakeDeps.CLIConfig.PodHardRecoveryPeriod = 2 * time.Hour
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	nodeIndexer := fakeDeps.KubeInformerFactory.Core().V1().Nodes().Informer().GetIndexer()
	return fakeDeps, pvcIndexer, podIndexer, nodeIndexer
}

func getTestPDPodAndPvcs(pvcIndexer cache.Indexer, podIndexer cache.Indexer, tc *v1alpha1.TidbCluster, testPodPvcParams testPodPvcParams) (*corev1.Pod, []*corev1.PersistentVolumeClaim) {
	pvc1 := newPVCForPDFailover(tc, v1alpha1.PDMemberType, 1)
	pvc1.Name = pvc1.Name + "-1"
	pvc1.UID = pvc1.UID + "-1"

	if testPodPvcParams.pvcWithDeletionTimestamp {
		pvc1.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
	pvcIndexer.Add(pvc1)
	pod := newPodForFailover(tc, v1alpha1.PDMemberType, 1)
	if testPodPvcParams.podWithDeletionTimestamp {
		pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
	pvc1.ObjectMeta.Labels[label.AnnPodNameKey] = pod.GetName()
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc1.Name,
				},
			},
		})

	podIndexer.Add(pod)
	return pod, []*corev1.PersistentVolumeClaim{pvc1}
}

func getTestPodWithConditions(tc *v1alpha1.TidbCluster, phase corev1.PodPhase, conditions []corev1.PodCondition) *corev1.Pod {
	pod := newPodForFailover(tc, v1alpha1.PDMemberType, 1)
	pod.Spec.NodeName = testNode1Name
	pod.Status.Phase = phase
	pod.Status.Conditions = conditions
	return pod
}

func getTestNodeWithConditions(conditions []corev1.NodeCondition) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNode1Name,
		},
		Status: corev1.NodeStatus{
			Conditions: conditions,
		},
	}
}

func getNAStatusExpectFn(g *GomegaWithT, nodeUnavailable, readOnlyDiskFound bool) func(naStatus NodeAvailabilityStatus) {
	return func(naStatus NodeAvailabilityStatus) {
		g.Expect(naStatus.NodeUnavailable).To(Equal(nodeUnavailable))
		g.Expect(naStatus.ReadOnlyDiskFound).To(Equal(readOnlyDiskFound))
	}
}

func getErrContainsSubstring(substr string) func(*GomegaWithT, error) {
	return func(g *GomegaWithT, err error) {
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring(substr))
	}
}
