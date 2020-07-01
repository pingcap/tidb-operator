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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	podinformers "k8s.io/client-go/informers/core/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
)

func TestTiDBUpgrader_Upgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                    string
		changeFn                func(*v1alpha1.TidbCluster)
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
			SetStatefulSetLastAppliedConfigAnnotation(oldSet)
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
	}

	for _, test := range tests {
		testFn(test, t)
	}

}

func newTiDBUpgrader() (Upgrader, *controller.FakeTiDBControl, podinformers.PodInformer) {
	kubeCli := kubefake.NewSimpleClientset()
	tidbControl := controller.NewFakeTiDBControl()
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	return &tidbUpgrader{tidbControl: tidbControl, podLister: podInformer.Lister()}, tidbControl, podInformer
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
			PD: v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiKV: v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiDB: v1alpha1.TiDBSpec{
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
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tidbPodName(upgradeTcName, 1),
				Namespace: corev1.NamespaceDefault,
				Labels:    lu,
			},
		},
	}
	return pods
}
