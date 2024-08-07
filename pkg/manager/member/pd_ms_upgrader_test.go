// Copyright 2024 PingCAP, Inc.
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

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/utils/pointer"
)

const tsoService = "tso"

func TestPDMSUpgraderUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name         string
		changeFn     func(*v1alpha1.TidbCluster)
		changePods   func(pods []*corev1.Pod)
		changeOldSet func(set *apps.StatefulSet)
		errExpectFn  func(*GomegaWithT, error)
		expectFn     func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet)
	}

	testFn := func(test *testcase) {
		t.Log(test.name)
		upgrader, podInformer := newPDMSUpgrader()
		tc := newTidbClusterForPDMSUpgrader()

		if test.changeFn != nil {
			test.changeFn(tc)
		}

		pods := getPDMSPods()
		if test.changePods != nil {
			test.changePods(pods)
		}
		for i := range pods {
			podInformer.Informer().GetIndexer().Add(pods[i])
		}

		newSet := newStatefulSetForPDMSUpgrader()
		oldSet := newSet.DeepCopy()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}
		mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)

		newSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(3)

		err := upgrader.Upgrade(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		test.expectFn(g, tc, newSet)
	}

	tests := []testcase{
		{
			name: "normal upgrade",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
			},
			changePods:   nil,
			changeOldSet: nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "normal upgrade with notReady pod",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					pod.Status = *new(corev1.PodStatus)
				}
			},
			changeOldSet: nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "modify oldSet update strategy to OnDelete",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.OnDeleteStatefulSetStrategyType,
				}
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name: "set oldSet's RollingUpdate strategy to nil",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.RollingUpdateStatefulSetStrategyType,
				}
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "newSet template changed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.Template.Spec.Containers[0].Image = "pd-test-image:old"
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "pdms scaling",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
				tc.Status.PDMS[tsoService].Phase = v1alpha1.ScalePhase
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.Template.Spec.Containers[0].Image = "pd-test-image:old"
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.ScalePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "pdms sync failed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = false
			},
			changePods: nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},

		{
			name: "ignore pd peers health if annotation is not set",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PDMS[tsoService].Synced = true
			},
			changePods:   nil,
			changeOldSet: nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i])
	}
}

func newPDMSUpgrader() (Upgrader, podinformers.PodInformer) {
	fakeDeps := controller.NewFakeDependencies()
	pdMSUpgrader := &pdMSUpgrader{deps: fakeDeps}
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	return pdMSUpgrader, podInformer
}

func newStatefulSetForPDMSUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.PDMSMemberName(upgradeTcName, tsoService),
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pd",
							Image: "pd-test-image",
						},
					},
				},
			},
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type:          apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{Partition: pointer.Int32Ptr(2)},
			},
		},
		Status: apps.StatefulSetStatus{
			CurrentRevision: "1",
			UpdateRevision:  "2",
			ReadyReplicas:   3,
			Replicas:        3,
			CurrentReplicas: 2,
			UpdatedReplicas: 1,
		},
	}
}

func newTidbClusterForPDMSUpgrader() *v1alpha1.TidbCluster {
	podName0 := PDMSPodName(upgradeTcName, 0, tsoService)
	podName1 := PDMSPodName(upgradeTcName, 1, tsoService)
	podName2 := PDMSPodName(upgradeTcName, 2, tsoService)
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeTcName,
			Namespace: corev1.NamespaceDefault,
			UID:       upgradeTcName,
			Labels:    label.New().Instance(upgradeInstanceName),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pingcap/pd:v7.3.0",
				},
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
				Mode:             "ms",
			},
			PDMS: []*v1alpha1.PDMSSpec{
				{
					Name: tsoService,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: "pd-test-image",
					},
					Replicas:         3,
					StorageClassName: pointer.StringPtr("my-storage-class"),
				},
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			PDMS: map[string]*v1alpha1.PDMSStatus{
				tsoService: {
					Phase: v1alpha1.NormalPhase,
					StatefulSet: &apps.StatefulSetStatus{
						CurrentRevision: "1",
						UpdateRevision:  "2",
						ReadyReplicas:   3,
						Replicas:        3,
						CurrentReplicas: 2,
						UpdatedReplicas: 1,
					},
					Members: []string{podName0, podName1, podName2},
				},
			},
		},
	}
}

func getPDMSPods() []*corev1.Pod {
	lc := label.New().Instance(upgradeInstanceName).PDMS(tsoService).Labels()
	lc[apps.ControllerRevisionHashLabelKey] = "1"
	lu := label.New().Instance(upgradeInstanceName).PDMS(tsoService).Labels()
	lu[apps.ControllerRevisionHashLabelKey] = "2"
	pods := []*corev1.Pod{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PDMSPodName(upgradeTcName, 0, tsoService),
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
				Name:      PDMSPodName(upgradeTcName, 1, tsoService),
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
				Name:      PDMSPodName(upgradeTcName, 2, tsoService),
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
