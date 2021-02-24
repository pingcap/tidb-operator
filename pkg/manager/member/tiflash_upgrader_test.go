// Copyright 2021 PingCAP, Inc.
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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	podinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/utils/pointer"
)

func TestTiFlashUpgraderUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name         string
		changeFn     func(*v1alpha1.TidbCluster)
		changeOldSet func(set *apps.StatefulSet)
		changePods   func([]*corev1.Pod)
		updatePodErr bool
		errExpectFn  func(*GomegaWithT, error)
		expectFn     func(*GomegaWithT, *v1alpha1.TidbCluster, *apps.StatefulSet, map[string]*corev1.Pod)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		upgrader, _, podControl, podInformer := newTiFlashUpgrader()

		tc := newTidbClusterForTiFlashUpgrader()
		if test.changeFn != nil {
			test.changeFn(tc)
		}

		oldSet := oldStatefulSetForTiFlashUpgrader()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}
		newSet := newStatefulSetForTiFlashUpgrader()

		tiflashPods := getTiFlashPods(oldSet)
		if test.changePods != nil {
			test.changePods(tiflashPods)
		}
		for _, pod := range tiflashPods {
			podInformer.Informer().GetIndexer().Add(pod)
		}

		if test.updatePodErr {
			podControl.SetUpdatePodError(fmt.Errorf("failed to update pod"), 0)
		}

		err := upgrader.Upgrade(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		l, err := label.New().Instance(upgradeInstanceName).TiFlash().Selector()
		g.Expect(err).NotTo(HaveOccurred())
		tiflashPods, err = podInformer.Lister().Pods(tc.Namespace).List(l)
		g.Expect(err).NotTo(HaveOccurred())
		pods := map[string]*corev1.Pod{}
		for _, pod := range tiflashPods {
			pods[pod.GetName()] = pod
		}
		test.expectFn(g, tc, newSet, pods)
	}

	tests := []*testcase{
		{
			name:     "modify oldSet update strategy to OnDelete",
			changeFn: nil,
			changeOldSet: func(oldSet *apps.StatefulSet) {
				SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.OnDeleteStatefulSetStrategyType,
				}
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name:     "set oldSet's RollingUpdate strategy to nil",
			changeFn: nil,
			changeOldSet: func(oldSet *apps.StatefulSet) {
				SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.RollingUpdateStatefulSetStrategyType,
				}
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "to upgrade the pod which ordinal is 2",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Synced = true
				// set leader to 0
				store := tc.Status.TiFlash.Stores["3"]
				store.LeaderCount = 0
				tc.Status.TiFlash.Stores["3"] = store
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "to upgrade the pod which ordinal is 1",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
				tc.Status.TiFlash.Synced = true
				tc.Status.TiFlash.StatefulSet.CurrentReplicas = 2
				tc.Status.TiFlash.StatefulSet.UpdatedReplicas = 1
				// set leader to 0
				store := tc.Status.TiFlash.Stores["2"]
				store.LeaderCount = 0
				tc.Status.TiFlash.Stores["2"] = store
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
		{
			name: "newSet template changed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				oldSet.Spec.Template.Spec.Containers[0].Image = "old-image"
				SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "update revision equals current revision",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Synced = true
				tc.Status.TiFlash.StatefulSet.UpdateRevision = tc.Status.TiFlash.StatefulSet.CurrentRevision
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "tiflash can not upgrade when pd is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "get last apply config error",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				oldSet.SetAnnotations(map[string]string{LastAppliedConfigAnnotation: "fake apply config"})
			},
			changePods:   nil,
			updatePodErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func newTiFlashUpgrader() (Upgrader, *pdapi.FakePDControl, *controller.FakePodControl, podinformers.PodInformer) {
	fakeDeps := controller.NewFakeDependencies()
	pdControl := fakeDeps.PDControl.(*pdapi.FakePDControl)
	podControl := fakeDeps.PodControl.(*controller.FakePodControl)
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	return &tiflashUpgrader{deps: fakeDeps}, pdControl, podControl, podInformer
}

func newStatefulSetForTiFlashUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-tiflash",
			Namespace: metav1.NamespaceDefault,
			Labels:    label.New().Instance(upgradeInstanceName).TiFlash().Labels(),
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tiflash",
							Image: "tiflash-test-image",
						},
					},
				},
			},
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: pointer.Int32Ptr(3),
				}},
		},
	}
}

func oldStatefulSetForTiFlashUpgrader() *apps.StatefulSet {
	set := newStatefulSetForTiFlashUpgrader()
	set.Status = apps.StatefulSetStatus{
		Replicas:        3,
		CurrentReplicas: 3,
		UpdatedReplicas: 0,
		CurrentRevision: "1",
		UpdateRevision:  "2",
	}
	return set
}

func newTidbClusterForTiFlashUpgrader() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeTcName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID(upgradeTcName),
			Labels:    label.New().Instance(upgradeInstanceName).TiFlash().Labels(),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiFlash: &v1alpha1.TiFlashSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tiflash-test-image",
				},
				Replicas: 3,
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiFlash: v1alpha1.TiFlashStatus{
				Synced: true,
				Phase:  v1alpha1.UpgradePhase,
				StatefulSet: &apps.StatefulSetStatus{
					Replicas:        3,
					CurrentReplicas: 3,
					UpdatedReplicas: 0,
					CurrentRevision: "1",
					UpdateRevision:  "2",
				},
				Stores: map[string]v1alpha1.TiKVStore{
					"1": {
						ID:          "1",
						PodName:     TiFlashPodName(upgradeTcName, 0),
						LeaderCount: 10,
						State:       "Up",
					},
					"2": {
						ID:          "2",
						PodName:     TiFlashPodName(upgradeTcName, 1),
						LeaderCount: 10,
						State:       "Up",
					},
					"3": {
						ID:          "3",
						PodName:     TiFlashPodName(upgradeTcName, 2),
						LeaderCount: 10,
						State:       "Up",
					},
				},
			},
		},
	}
}

func getTiFlashPods(set *apps.StatefulSet) []*corev1.Pod {
	var pods []*corev1.Pod
	for i := 0; i < int(set.Status.Replicas); i++ {
		l := label.New().Instance(upgradeInstanceName).TiFlash().Labels()
		if i+1 <= int(set.Status.CurrentReplicas) {
			l[apps.ControllerRevisionHashLabelKey] = set.Status.CurrentRevision
		} else {
			l[apps.ControllerRevisionHashLabelKey] = set.Status.UpdateRevision
		}
		l[label.StoreIDLabelKey] = strconv.Itoa(i + 1)
		pods = append(pods, &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      TiFlashPodName(upgradeTcName, int32(i)),
				Namespace: corev1.NamespaceDefault,
				Labels:    l,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		})
	}
	return pods
}
