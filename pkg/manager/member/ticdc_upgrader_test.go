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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	podinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/utils/pointer"
)

func TestTiCDCUpgrader_Upgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name           string
		changeFn       func(*v1alpha1.TidbCluster)
		invalidPod     bool
		changePods     func(pods []*corev1.Pod)
		missPod        bool
		errorExpect    bool
		changeOldSet   func(set *apps.StatefulSet)
		changeUpgrader func(u *ticdcUpgrader)
		expectFn       func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		upgrader, podInformer := newTiCDCUpgrader()

		// A version that is smaller than ticdcCrossUpgradeVersion, v6.3.0.
		cdcVersionOld := "v6.2.0"
		cdcControl := upgrader.(*ticdcUpgrader).deps.CDCControl.(*controller.FakeTiCDCControl)
		cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
			return &controller.CaptureStatus{Version: cdcVersionOld}, nil
		}
		cdcVersionNew := ticdcCrossUpgradeVersion
		tc := newTidbClusterForTiCDCUpgrader()
		tc.Spec.TiCDC.Version = &cdcVersionNew
		if test.changeFn != nil {
			test.changeFn(tc)
		}
		pods := getTiCDCPods()
		if test.invalidPod {
			pods[1].Labels = nil
		}
		if test.missPod {
			pods = pods[:0]
		}
		if test.changePods != nil {
			test.changePods(pods)
		}
		if test.changeUpgrader != nil {
			test.changeUpgrader(upgrader.(*ticdcUpgrader))
		}
		for _, pod := range pods {
			podInformer.Informer().GetIndexer().Add(pod)
		}

		oldSet := newStatefulSetForTiCDCUpgrader()
		newSet := oldSet.DeepCopy()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}
		mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)

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
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(0)))
			},
		},
		{
			name:        "graceful upgrade retry resign owner",
			errorExpect: true,
			changeUpgrader: func(u *ticdcUpgrader) {
				cdcControl := u.deps.CDCControl.(*controller.FakeTiCDCControl)
				cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
					return &controller.CaptureStatus{Version: ticdcCrossUpgradeVersion}, nil
				}

				// resignOwner returns false to let graceful shutdown retry.
				cdcControl.ResignOwnerFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error) {
					return false, nil
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name:        "graceful upgrade retry drain",
			errorExpect: true,
			changeUpgrader: func(u *ticdcUpgrader) {
				cdcControl := u.deps.CDCControl.(*controller.FakeTiCDCControl)
				cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
					return &controller.CaptureStatus{Version: ticdcCrossUpgradeVersion}, nil
				}

				// resignOwner always success.
				cdcControl.ResignOwnerFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error) {
					return true, nil
				}
				// drainCapture returns none zero table count to let graceful shutdown retry.
				cdcControl.DrainCaptureFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (int, bool, error) {
					return 1, false, nil
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name:        "graceful upgrade retry is healthy",
			errorExpect: true,
			changePods: func(pods []*corev1.Pod) {
				for i := range pods {
					// Set all pods to the old revision.
					pods[i].Labels[apps.ControllerRevisionHashLabelKey] = "1"
				}
			},
			changeOldSet: func(set *apps.StatefulSet) {
				// Upgrade from the ordinal 2.
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType,
					RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
						Partition: pointer.Int32Ptr(2),
					},
				}
				set.Status = apps.StatefulSetStatus{
					CurrentRevision: "1",
					UpdateRevision:  "2",
					ReadyReplicas:   2,
					Replicas:        2,
					CurrentReplicas: 2,
					UpdatedReplicas: 0,
				}
			},
			changeUpgrader: func(u *ticdcUpgrader) {
				cdcControl := u.deps.CDCControl.(*controller.FakeTiCDCControl)
				cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
					return &controller.CaptureStatus{Version: ticdcCrossUpgradeVersion}, nil
				}

				// resignOwner always success.
				cdcControl.ResignOwnerFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error) {
					return true, nil
				}
				// drainCapture always success.
				cdcControl.DrainCaptureFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (int, bool, error) {
					return 0, false, nil
				}
				// isHealthy returns false to let graceful shutdown retry.
				cdcControl.IsHealthyFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
					return false, nil
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name:        "last graceful upgrade does not wait healthy",
			errorExpect: false,
			changeUpgrader: func(u *ticdcUpgrader) {
				cdcControl := u.deps.CDCControl.(*controller.FakeTiCDCControl)
				cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
					return &controller.CaptureStatus{Version: ticdcCrossUpgradeVersion}, nil
				}

				// resignOwner always success.
				cdcControl.ResignOwnerFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error) {
					return true, nil
				}
				// drainCapture always success.
				cdcControl.DrainCaptureFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (int, bool, error) {
					return 0, false, nil
				}
				// isHealthy returns false to let graceful shutdown retry.
				cdcControl.IsHealthyFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
					return false, nil
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(0)))
			},
		},
		{
			name: "normal with pod notReady",
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					pod.Status = *new(corev1.PodStatus)
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
			errorExpect: true,
		},
		{
			name: "modify oldSet update strategy to OnDelete",
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.OnDeleteStatefulSetStrategyType,
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name: "set oldSet's RollingUpdate strategy to nil",
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.RollingUpdateStatefulSetStrategyType,
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "scale to 0",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiCDC.Replicas = int32(0)
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "template change",
			changeOldSet: func(oldSet *apps.StatefulSet) {
				oldSet.Spec.Template.Spec.SchedulerName = "test-scheduler"
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "upgrade revision equals current revision",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiCDC.StatefulSet.UpdateRevision = tc.Status.TiCDC.StatefulSet.CurrentRevision
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "upgraded pods are not ready",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				delete(tc.Status.TiCDC.Captures, "upgrader-ticdc-1")
			},
			errorExpect: true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name:        "invalid Pod revision",
			invalidPod:  true,
			errorExpect: true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name:        "cannot find Pod",
			missPod:     true,
			errorExpect: true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "ticdc can not upgrade when pd is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.Pump.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.Phase = v1alpha1.NormalPhase
				tc.Status.TiCDC.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			errorExpect: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
		{
			name: "ticdc can not upgrade when tiflash is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.Pump.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.Phase = v1alpha1.NormalPhase
				tc.Status.TiCDC.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			errorExpect: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
		{
			name: "ticdc can not upgrade when tikv is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.Pump.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.Phase = v1alpha1.NormalPhase
				tc.Status.TiCDC.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			errorExpect: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
		{
			name: "ticdc can not upgrade when pump is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.Pump.Phase = v1alpha1.UpgradePhase
				tc.Status.TiDB.Phase = v1alpha1.NormalPhase
				tc.Status.TiCDC.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			errorExpect: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
		{
			name: "ticdc can not upgrade when tidb is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.Pump.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
				tc.Status.TiCDC.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			errorExpect: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}

}

func newTiCDCUpgrader() (Upgrader, podinformers.PodInformer) {
	fakeDeps := controller.NewFakeDependencies()
	upgrader := &ticdcUpgrader{fakeDeps}
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	return upgrader, podInformer
}

func newStatefulSetForTiCDCUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-ticdc",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(2),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "ticdc",
							Image: "ticdc-test-image",
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

func newTidbClusterForTiCDCUpgrader() *v1alpha1.TidbCluster {
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
			TiCDC: &v1alpha1.TiCDCSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "ticdc-test-image",
				},
				BaseImage: "ticdc-test-image",
				Replicas:  2,
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiCDC: v1alpha1.TiCDCStatus{
				Phase: v1alpha1.NormalPhase,
				StatefulSet: &apps.StatefulSetStatus{
					CurrentReplicas: 1,
					UpdatedReplicas: 1,
					CurrentRevision: "1",
					UpdateRevision:  "2",
					Replicas:        2,
				},
				Captures: map[string]v1alpha1.TiCDCCapture{
					"upgrader-ticdc-0": {
						PodName: "upgrader-ticdc-0",
					},
					"upgrader-ticdc-1": {
						PodName: "upgrader-ticdc-1",
					},
				},
			},
		},
	}
}

func getTiCDCPods() []*corev1.Pod {
	lc := label.New().Instance(upgradeInstanceName).TiCDC().Labels()
	lc[apps.ControllerRevisionHashLabelKey] = "1"
	lu := label.New().Instance(upgradeInstanceName).TiCDC().Labels()
	lu[apps.ControllerRevisionHashLabelKey] = "2"
	pods := []*corev1.Pod{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ticdcPodName(upgradeTcName, 0),
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
				Name:      ticdcPodName(upgradeTcName, 1),
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
