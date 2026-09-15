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
	"context"
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	podinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"
)

func TestTiDBUpgrader_Upgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                    string
		changeFn                func(*v1alpha1.TidbCluster)
		changePods              func(pods []*corev1.Pod)
		getLastAppliedConfigErr bool
		errorExpect             bool
		changeOldSet            func(set *apps.StatefulSet)
		expectFn                func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		upgrader, _, podInformer := newTiDBUpgrader()
		tc := newTidbClusterForTiDBUpgrader()
		if test.changeFn != nil {
			test.changeFn(tc)
		}
		pods := getTiDBPods()
		if test.changePods != nil {
			test.changePods(pods)
		}
		for _, pod := range pods {
			podInformer.Informer().GetIndexer().Add(pod)
		}

		oldSet := newStatefulSetForTiDBUpgrader()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}

		newSet := oldSet.DeepCopy()
		if test.getLastAppliedConfigErr {
			oldSet.SetAnnotations(map[string]string{LastAppliedConfigAnnotation: "fake apply config"})
		} else {
			mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
		}
		err := upgrader.Upgrade(tc, oldSet, newSet)
		if test.errorExpect {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		test.expectFn(g, tc, newSet)
	}

	tests := []*testcase{
		{
			name: "normal",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(0)))
			},
		},
		{
			name: "normal with notReady pod",
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					pod.Status = *new(corev1.PodStatus)
				}
			},
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
			errorExpect: true,
		},
		{
			name: "modify oldSet update strategy to OnDelete",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.OnDeleteStatefulSetStrategyType,
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name: "set oldSet's RollingUpdate strategy to nil",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.RollingUpdateStatefulSetStrategyType,
				}
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "pd is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "tikv is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "tiflash is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "pump is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.Pump.Phase = v1alpha1.UpgradePhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "upgrade revision equals current revision",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.StatefulSet.UpdateRevision = tc.Status.TiDB.StatefulSet.CurrentRevision
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "get apply config error",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
			},
			getLastAppliedConfigErr: true,
			errorExpect:             true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "upgraded pods are not ready",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.Members["upgrader-tidb-1"] = v1alpha1.TiDBMember{
					Name:   "upgrader-tidb-1",
					Health: false,
				}
			},
			getLastAppliedConfigErr: false,
			errorExpect:             true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "upgraded pod is ready but not available",
			changePods: func(pods []*corev1.Pod) {
				pods[1].Status.Conditions[0].LastTransitionTime = metav1.Now()
			},
			changeFn: func(tc *v1alpha1.TidbCluster) {
				if tc.Annotations == nil {
					tc.Annotations = map[string]string{}
				}
				// 5min is enough for unit test
				tc.Annotations[annoKeyTiDBMinReadySeconds] = "300"
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			errorExpect:             true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}

}

func TestTiDBUpgraderParallelUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	newPod := func(ordinal int32, revision string, ready bool, terminating bool) *corev1.Pod {
		l := label.New().Instance(upgradeInstanceName).TiDB().Labels()
		l[apps.ControllerRevisionHashLabelKey] = revision
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tidbPodName(upgradeTcName, ordinal),
				Namespace: corev1.NamespaceDefault,
				UID:       types.UID(fmt.Sprintf("uid-%d-rev%s", ordinal, revision)),
				Labels:    l,
			},
		}
		if ready {
			pod.Status = corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			}
		}
		if terminating {
			now := metav1.Now()
			pod.DeletionTimestamp = &now
		}
		return pod
	}

	iosPtr := func(v intstr.IntOrString) *intstr.IntOrString { return &v }

	type testcase struct {
		name           string
		maxUnavailable *intstr.IntOrString
		// the partition persisted on the old statefulset when the sync runs
		oldPartition int32
		// pods as the informer cache sees them
		pods []*corev1.Pod
		// pods as the API server holds them; defaults to pods
		clientPods      []*corev1.Pod
		errorExpect     bool
		expectErrSubstr string
		expectPartition int32
		expectDeleted   []int32
		expectRemaining []int32
	}

	tests := []*testcase{
		{
			name:           "default is serial: partition drops by one, no direct deletion",
			maxUnavailable: nil,
			oldPartition:   4,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 3,
			expectRemaining: []int32{0, 1, 2, 3},
		},
		{
			name:           "default with a missing pod returns the legacy error",
			maxUnavailable: nil,
			oldPartition:   4,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false),
			},
			errorExpect:     true,
			expectErrSubstr: "failed to get pods",
			expectPartition: 4,
			expectRemaining: []int32{0, 1, 2},
		},
		{
			name:           "maxUnavailable=2: first sync only lowers the partition",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   4,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 2,
			expectRemaining: []int32{0, 1, 2, 3},
		},
		{
			name:           "maxUnavailable=2: next sync deletes the exposed pods",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 2,
			expectDeleted:   []int32{2, 3},
			expectRemaining: []int32{0, 1},
		},
		{
			name:           "maxUnavailable=2 with one restarting pod advances by one",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "2", false, false),
			},
			expectPartition: 2,
			expectDeleted:   []int32{2},
			expectRemaining: []int32{0, 1, 3},
		},
		{
			name:           "maxUnavailable=2 with two restarting pods requeues",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "2", false, false), newPod(3, "2", false, false),
			},
			errorExpect:     true,
			expectPartition: 2,
			expectRemaining: []int32{0, 1, 2, 3},
		},
		{
			name:           "terminating pod consumes the budget",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, true),
			},
			expectPartition: 2,
			expectDeleted:   []int32{2},
			expectRemaining: []int32{0, 1, 3},
		},
		{
			name:           "missing pod consumes the budget",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false),
			},
			expectPartition: 2,
			expectDeleted:   []int32{2},
			expectRemaining: []int32{0, 1},
		},
		{
			name:           "a down outdated pod below the window consumes the budget",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   3,
			pods: []*corev1.Pod{
				newPod(0, "1", false, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 3,
			expectDeleted:   []int32{3},
			expectRemaining: []int32{0, 1, 2},
		},
		{
			name:           "restarting a down outdated pod is free",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", false, false),
			},
			expectPartition: 2,
			expectDeleted:   []int32{2, 3},
			expectRemaining: []int32{0, 1},
		},
		{
			name:           "a stale cache entry cannot delete a replaced pod",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   3,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			clientPods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "2", true, false),
			},
			expectPartition: 2,
			expectRemaining: []int32{0, 1, 2, 3},
		},
		{
			name:           "maxUnavailable is capped at replicas-1",
			maxUnavailable: iosPtr(intstr.FromInt(10)),
			oldPartition:   1,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 1,
			expectDeleted:   []int32{1, 2, 3},
			expectRemaining: []int32{0},
		},
		{
			name:           "maxUnavailable=50% of 4 replicas restarts two pods",
			maxUnavailable: iosPtr(intstr.FromString("50%")),
			oldPartition:   2,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 2,
			expectDeleted:   []int32{2, 3},
			expectRemaining: []int32{0, 1},
		},
		{
			name:           "maxUnavailable=5% is clamped to one pod",
			maxUnavailable: iosPtr(intstr.FromString("5%")),
			oldPartition:   4,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 3,
			expectRemaining: []int32{0, 1, 2, 3},
		},
		{
			name:           "maxUnavailable=200% is capped at replicas-1",
			maxUnavailable: iosPtr(intstr.FromString("200%")),
			oldPartition:   1,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 1,
			expectDeleted:   []int32{1, 2, 3},
			expectRemaining: []int32{0},
		},
		{
			name:           "invalid maxUnavailable falls back to serial",
			maxUnavailable: iosPtr(intstr.FromString("half")),
			oldPartition:   4,
			pods: []*corev1.Pod{
				newPod(0, "1", true, false), newPod(1, "1", true, false),
				newPod(2, "1", true, false), newPod(3, "1", true, false),
			},
			expectPartition: 3,
			expectRemaining: []int32{0, 1, 2, 3},
		},
		{
			name:           "no candidates left but a pod is still restarting",
			maxUnavailable: iosPtr(intstr.FromInt(2)),
			oldPartition:   0,
			pods: []*corev1.Pod{
				newPod(0, "2", false, false), newPod(1, "2", true, false),
				newPod(2, "2", true, false), newPod(3, "2", true, false),
			},
			errorExpect:     true,
			expectPartition: 0,
			expectRemaining: []int32{0, 1, 2, 3},
		},
	}

	for _, test := range tests {
		t.Log(test.name)

		fakeDeps := controller.NewFakeDependencies()
		upgrader := &tidbUpgrader{fakeDeps}
		podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()

		tc := newTidbClusterForTiDBUpgrader()
		tc.Spec.TiDB.Replicas = 4
		tc.Spec.TiDB.UpgradePolicy = v1alpha1.UpgradePolicy{MaxUnavailable: test.maxUnavailable}
		tc.Status.PD.Phase = v1alpha1.NormalPhase
		tc.Status.TiKV.Phase = v1alpha1.NormalPhase
		tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{}
		for i := int32(0); i < 4; i++ {
			name := tidbPodName(upgradeTcName, i)
			tc.Status.TiDB.Members[name] = v1alpha1.TiDBMember{Name: name, Health: true}
		}

		clientPods := test.clientPods
		if clientPods == nil {
			clientPods = test.pods
		}
		for _, pod := range test.pods {
			g.Expect(podInformer.Informer().GetIndexer().Add(pod)).To(Succeed())
		}
		for _, pod := range clientPods {
			_, err := fakeDeps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())
		}

		// the fake object tracker ignores delete preconditions: enforce them
		// like the API server does, and require a UID precondition on every delete
		fakeCli := fakeDeps.KubeClientset.(*fake.Clientset)
		podGVR := corev1.SchemeGroupVersion.WithResource("pods")
		fakeCli.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			da := action.(k8stesting.DeleteActionImpl)
			g.Expect(da.DeleteOptions.Preconditions).NotTo(BeNil(), "delete of %s must carry preconditions", da.Name)
			g.Expect(da.DeleteOptions.Preconditions.UID).NotTo(BeNil(), "delete of %s must carry a UID precondition", da.Name)
			obj, err := fakeCli.Tracker().Get(podGVR, da.Namespace, da.Name)
			if err == nil && obj.(*corev1.Pod).UID != *da.DeleteOptions.Preconditions.UID {
				return true, nil, apierrors.NewConflict(podGVR.GroupResource(), da.Name, fmt.Errorf("uid precondition failed"))
			}
			return false, nil, nil
		})

		oldSet := newStatefulSetForTiDBUpgrader()
		oldSet.Spec.Replicas = pointer.Int32Ptr(4)
		oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(test.oldPartition)
		mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
		newSet := oldSet.DeepCopy()

		err := upgrader.Upgrade(tc, oldSet, newSet)
		if test.errorExpect {
			g.Expect(err).To(HaveOccurred())
			if test.expectErrSubstr != "" {
				g.Expect(err.Error()).To(ContainSubstring(test.expectErrSubstr))
			}
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(test.expectPartition))

		for _, ordinal := range test.expectDeleted {
			_, err := fakeDeps.KubeClientset.CoreV1().Pods(corev1.NamespaceDefault).Get(context.TODO(), tidbPodName(upgradeTcName, ordinal), metav1.GetOptions{})
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "pod %d should have been deleted", ordinal)
		}
		for _, ordinal := range test.expectRemaining {
			_, err := fakeDeps.KubeClientset.CoreV1().Pods(corev1.NamespaceDefault).Get(context.TODO(), tidbPodName(upgradeTcName, ordinal), metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred(), "pod %d should not have been deleted", ordinal)
		}
	}
}

func newTiDBUpgrader() (Upgrader, *controller.FakeTiDBControl, podinformers.PodInformer) {
	fakeDeps := controller.NewFakeDependencies()
	upgrader := &tidbUpgrader{fakeDeps}
	tidbControl := fakeDeps.TiDBControl.(*controller.FakeTiDBControl)
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	return upgrader, tidbControl, podInformer
}

func newStatefulSetForTiDBUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-tidb",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(2),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tidb",
							Image: "tidb-test-image",
						},
					},
				},
			},
			UpdateStrategy: apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: pointer.Int32Ptr(1),
				},
			},
		},
		Status: apps.StatefulSetStatus{
			CurrentRevision: "1",
			UpdateRevision:  "2",
			ReadyReplicas:   2,
			Replicas:        2,
			CurrentReplicas: 1,
			UpdatedReplicas: 1,
		},
	}
}

func newTidbClusterForTiDBUpgrader() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("upgrader"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiKV: &v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiDB: &v1alpha1.TiDBSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tidb-test-image",
				},
				Replicas: 2,
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiDB: v1alpha1.TiDBStatus{
				StatefulSet: &apps.StatefulSetStatus{
					CurrentReplicas: 1,
					UpdatedReplicas: 1,
					CurrentRevision: "1",
					UpdateRevision:  "2",
					Replicas:        2,
				},
				Members: map[string]v1alpha1.TiDBMember{
					"upgrader-tidb-0": {
						Name:   "upgrader-tidb-0",
						Health: true,
					},
					"upgrader-tidb-1": {
						Name:   "upgrader-tidb-1",
						Health: true,
					},
				},
			},
		},
	}
}

func getTiDBPods() []*corev1.Pod {
	lc := label.New().Instance(upgradeInstanceName).TiDB().Labels()
	lc[apps.ControllerRevisionHashLabelKey] = "1"
	lu := label.New().Instance(upgradeInstanceName).TiDB().Labels()
	lu[apps.ControllerRevisionHashLabelKey] = "2"
	pods := []*corev1.Pod{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tidbPodName(upgradeTcName, 0),
				Namespace: corev1.NamespaceDefault,
				Labels:    lc,
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tidbPodName(upgradeTcName, 1),
				Namespace: corev1.NamespaceDefault,
				Labels:    lu,
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue},
				},
			},
		},
	}
	return pods
}

func TestTiDBUpgraderSmoothUpgradeStart(t *testing.T) {
	g := NewGomegaWithT(t)
	upgrader, tidbControl, podInformer := newTiDBUpgrader()
	tc := newTidbClusterForTiDBUpgrader()
	tc.Status.PD.Phase = v1alpha1.NormalPhase
	tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	for _, pod := range getTiDBPods() {
		g.Expect(podInformer.Informer().GetIndexer().Add(pod)).To(Succeed())
	}
	oldSet := newStatefulSetForTiDBUpgrader()
	oldSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.4.0"
	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.5.0"
	g.Expect(mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)).To(Succeed())

	err := upgrader.Upgrade(tc, oldSet, newSet)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tidbControl.StartUpgradeOrdinals).To(Equal([]int32{0}))
	g.Expect(isSmoothUpgradePaused(tc)).To(BeTrue())
	g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
}

func TestTiDBUpgraderSmoothUpgradeStartFailureBlocksRollout(t *testing.T) {
	g := NewGomegaWithT(t)
	upgrader, tidbControl, podInformer := newTiDBUpgrader()
	tidbControl.SetStartUpgradeError(fmt.Errorf("boom"))
	tc := newTidbClusterForTiDBUpgrader()
	tc.Status.PD.Phase = v1alpha1.NormalPhase
	tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	for _, pod := range getTiDBPods() {
		g.Expect(podInformer.Informer().GetIndexer().Add(pod)).To(Succeed())
	}
	oldSet := newStatefulSetForTiDBUpgrader()
	oldSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.4.0"
	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.5.0"
	g.Expect(mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)).To(Succeed())

	err := upgrader.Upgrade(tc, oldSet, newSet)
	g.Expect(err).To(HaveOccurred())
	g.Expect(tidbControl.StartUpgradeOrdinals).To(Equal([]int32{0}))
	g.Expect(isSmoothUpgradePaused(tc)).To(BeFalse())
	g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
}

func TestTiDBUpgraderSmoothUpgradeActiveAnnotationSkipsDuplicate(t *testing.T) {
	g := NewGomegaWithT(t)
	upgrader, tidbControl, podInformer := newTiDBUpgrader()
	tc := newTidbClusterForTiDBUpgrader()
	tc.Status.PD.Phase = v1alpha1.NormalPhase
	tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	for _, pod := range getTiDBPods() {
		g.Expect(podInformer.Informer().GetIndexer().Add(pod)).To(Succeed())
	}
	oldSet := newStatefulSetForTiDBUpgrader()
	oldSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.4.0"
	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.5.0"
	g.Expect(mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)).To(Succeed())
	setSmoothUpgradeAnnotations(tc, "v7.4.0", "v7.5.0")

	err := upgrader.Upgrade(tc, oldSet, newSet)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tidbControl.StartUpgradeOrdinals).To(BeEmpty())
	g.Expect(isSmoothUpgradePaused(tc)).To(BeTrue())
}

func TestTiDBUpgraderSmoothUpgradeStaleAnnotationRecovery(t *testing.T) {
	g := NewGomegaWithT(t)
	upgrader, tidbControl, podInformer := newTiDBUpgrader()
	tc := newTidbClusterForTiDBUpgrader()
	tc.Status.PD.Phase = v1alpha1.NormalPhase
	tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	for _, pod := range getTiDBPods() {
		g.Expect(podInformer.Informer().GetIndexer().Add(pod)).To(Succeed())
	}
	oldSet := newStatefulSetForTiDBUpgrader()
	oldSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.4.0"
	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.5.0"
	g.Expect(mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)).To(Succeed())
	// stale annotations from a prior upgrade to a different target
	setSmoothUpgradeAnnotations(tc, "v7.4.0", "v7.4.1")

	err := upgrader.Upgrade(tc, oldSet, newSet)
	g.Expect(err).To(HaveOccurred()) // requeue error
	g.Expect(tidbControl.FinishUpgradeOrdinals).To(Equal([]int32{0}))
	g.Expect(tidbControl.StartUpgradeOrdinals).To(BeEmpty())
	g.Expect(isSmoothUpgradePaused(tc)).To(BeFalse())
}

func TestTiDBUpgraderSmoothUpgradeSkipsNonSwitchPairs(t *testing.T) {
	g := NewGomegaWithT(t)
	upgrader, tidbControl, podInformer := newTiDBUpgrader()
	tc := newTidbClusterForTiDBUpgrader()
	tc.Status.PD.Phase = v1alpha1.NormalPhase
	tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	for _, pod := range getTiDBPods() {
		g.Expect(podInformer.Informer().GetIndexer().Add(pod)).To(Succeed())
	}
	oldSet := newStatefulSetForTiDBUpgrader()
	oldSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.3.0"
	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Spec.Containers[0].Image = "pingcap/tidb:v7.4.0"
	g.Expect(mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)).To(Succeed())

	err := upgrader.Upgrade(tc, oldSet, newSet)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tidbControl.StartUpgradeOrdinals).To(BeEmpty())
}
