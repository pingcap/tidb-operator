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
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/pdapi"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	podinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/utils/pointer"
)

func TestPDUpgraderUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name               string
		changeFn           func(*v1alpha1.TidbCluster)
		changePods         func(pods []*corev1.Pod)
		changeOldSet       func(set *apps.StatefulSet)
		transferLeaderErr  bool
		pdPeersAreUnstable bool
		errExpectFn        func(*GomegaWithT, error)
		expectFn           func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet)
	}

	testFn := func(test *testcase) {
		t.Log(test.name)
		upgrader, pdControl, _, podInformer := newPDUpgrader()
		tc := newTidbClusterForPDUpgrader()
		pdClient := controller.NewFakePDClient(pdControl, tc)

		if test.changeFn != nil {
			test.changeFn(tc)
		}

		if test.transferLeaderErr {
			pdClient.AddReaction(pdapi.TransferPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to transfer leader")
			})
		} else {
			pdClient.AddReaction(pdapi.TransferPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, nil
			})
		}

		pods := getPDPods()
		if test.changePods != nil {
			test.changePods(pods)
		}
		for i := range pods {
			podInformer.Informer().GetIndexer().Add(pods[i])
		}

		newSet := newStatefulSetForPDUpgrader()
		oldSet := newSet.DeepCopy()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}
		mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)

		newSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(3)

		pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
			healthInfo := &pdapi.HealthInfo{
				Healths: []pdapi.MemberHealth{
					{
						Name:   PdPodName(upgradeTcName, 1),
						Health: !test.pdPeersAreUnstable,
					},
				},
			}
			return healthInfo, nil
		})

		for _, member := range tc.Status.PD.Members {
			pdClientM := controller.NewFakePDClientForMember(pdControl, tc, &member)
			pdClientM.AddReaction(pdapi.GetReadyActionType, func(action *pdapi.Action) (interface{}, error) {
				return true, nil
			})
		}
		for _, member := range tc.Status.PD.PeerMembers {
			pdClientM := controller.NewFakePDClientForMember(pdControl, tc, &member)
			pdClientM.AddReaction(pdapi.GetReadyActionType, func(action *pdapi.Action) (interface{}, error) {
				return true, nil
			})
		}

		err := upgrader.Upgrade(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		test.expectFn(g, tc, newSet)
	}

	tests := []testcase{
		{
			name: "normal upgrade",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
			},
			changePods:        nil,
			changeOldSet:      nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "normal upgrade with notReady pod",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					pod.Status = *new(corev1.PodStatus)
				}
			},
			changeOldSet:      nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "modify oldSet update strategy to OnDelete",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.OnDeleteStatefulSetStrategyType,
				}
			},
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name: "set oldSet's RollingUpdate strategy to nil",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.RollingUpdateStatefulSetStrategyType,
				}
			},
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "newSet template changed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.Template.Spec.Containers[0].Image = "pd-test-image:old"
			},
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "pd scaling",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
				tc.Status.PD.Phase = v1alpha1.ScalePhase
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.Template.Spec.Containers[0].Image = "pd-test-image:old"
			},
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.ScalePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "skip to wait all members health",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
				tc.Status.PD.Members[PdPodName(upgradeTcName, 2)] = v1alpha1.PDMember{Name: PdPodName(upgradeTcName, 2), Health: false}
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err.Error()).To(Equal(fmt.Sprintf("tidbcluster: [default/upgrader]'s pd upgraded pod: [%s] is not health", PdPodName(upgradeTcName, 2))))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "transfer leader",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
				tc.Status.PD.Leader = v1alpha1.PDMember{Name: PdPodName(upgradeTcName, 1), Health: true}
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "pd sync failed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = false
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "error when transfer leader",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
				tc.Status.PD.Leader = v1alpha1.PDMember{Name: PdPodName(upgradeTcName, 1), Health: true}
			},
			changePods:        nil,
			transferLeaderErr: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "fail if pd peers are unstable",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
				if tc.Annotations == nil {
					tc.Annotations = map[string]string{}
				}
				tc.Annotations[annoKeyPDPeersCheck] = "true"
			},
			changePods:         nil,
			transferLeaderErr:  false,
			pdPeersAreUnstable: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect("Peer PDs is unstable: Only 0 out of 1 PDs are healthy").To(Equal(err.Error()))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "ignore pd peers health if annotation is not set",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
			},
			changePods:         nil,
			changeOldSet:       nil,
			transferLeaderErr:  false,
			pdPeersAreUnstable: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "upgraded pod is ready but not available",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Synced = true
				if tc.Annotations == nil {
					tc.Annotations = map[string]string{}
				}
				// 5min is enough for unit test
				tc.Annotations[annoKeyTiDBMinReadySeconds] = "300"
			},
			changePods: func(pods []*corev1.Pod) {
				pods[1].Status.Conditions[0].LastTransitionTime = metav1.Now()
			},
			changeOldSet:       nil,
			transferLeaderErr:  false,
			pdPeersAreUnstable: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i])
	}

}

func TestChoosePDToTransferFromMembers(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name             string
		changeFn         func(*v1alpha1.TidbCluster, *apps.StatefulSet)
		ordinal          int32
		expectTargetName string
	}

	cases := []testcase{
		{
			name: "ordinal is max",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
			},
			ordinal:          2,
			expectTargetName: "upgrader-pd-0",
		},
		{
			name: "ordinal is max but min ordinal pod is unhealthy",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
			},
			ordinal:          2,
			expectTargetName: "upgrader-pd-1",
		},
		{
			name: "ordinal is max but others are unhealthy",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
			},
			ordinal:          2,
			expectTargetName: "",
		},
		{
			name: "ordinal is mid",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
			},
			ordinal:          1,
			expectTargetName: "upgrader-pd-2",
		},
		{
			name: "ordinal is mid but max ordinal pod is unhealthy",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
			},
			ordinal:          1,
			expectTargetName: "upgrader-pd-0",
		},
		{
			name: "ordinal is mid but others are unhealthy",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
			},
			ordinal:          1,
			expectTargetName: "",
		},
		{
			name: "ordinal is min",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
			},
			ordinal:          0,
			expectTargetName: "upgrader-pd-2",
		},
		{
			name: "ordinal is min but max ordinal pod is unhealthy",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
			},
			ordinal:          0,
			expectTargetName: "upgrader-pd-1",
		},
		{
			name: "ordinal is min but others are unhealthy",
			changeFn: func(tc *v1alpha1.TidbCluster, ss *apps.StatefulSet) {
				tc.Status.PD.Members[PdName(tc.Name, 0, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: true}
				tc.Status.PD.Members[PdName(tc.Name, 1, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
				tc.Status.PD.Members[PdName(tc.Name, 2, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)] = v1alpha1.PDMember{Health: false}
			},
			ordinal:          0,
			expectTargetName: "",
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		oriTC := newTidbClusterForPDUpgrader()
		oriNewSet := newStatefulSetForPDUpgrader()
		if testcase.changeFn != nil {
			testcase.changeFn(oriTC, oriNewSet)
		}

		tc := oriTC.DeepCopy()
		newSet := oriNewSet.DeepCopy()
		ordinal := testcase.ordinal

		targetName := choosePDToTransferFromMembers(tc, newSet, ordinal)
		g.Expect(targetName).Should(Equal(testcase.expectTargetName))
		g.Expect(tc).Should(Equal(oriTC))
		g.Expect(newSet).Should(Equal(oriNewSet))
	}
}

func newPDUpgrader() (Upgrader, *pdapi.FakePDControl, *controller.FakePodControl, podinformers.PodInformer) {
	fakeDeps := controller.NewFakeDependencies()
	pdUpgrader := &pdUpgrader{deps: fakeDeps}
	pdControl := fakeDeps.PDControl.(*pdapi.FakePDControl)
	podControl := fakeDeps.PodControl.(*controller.FakePodControl)
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	return pdUpgrader, pdControl, podControl, podInformer
}

func newStatefulSetForPDUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.PDMemberName(upgradeTcName),
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

func newTidbClusterForPDUpgrader() *v1alpha1.TidbCluster {
	podName0 := PdPodName(upgradeTcName, 0)
	podName1 := PdPodName(upgradeTcName, 1)
	podName2 := PdPodName(upgradeTcName, 2)
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeTcName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID(upgradeTcName),
			Labels:    label.New().Instance(upgradeInstanceName),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			PD: v1alpha1.PDStatus{
				Phase: v1alpha1.NormalPhase,
				StatefulSet: &apps.StatefulSetStatus{
					CurrentRevision: "1",
					UpdateRevision:  "2",
					ReadyReplicas:   3,
					Replicas:        3,
					CurrentReplicas: 2,
					UpdatedReplicas: 1,
				},
				Members: map[string]v1alpha1.PDMember{
					podName0: {Name: podName0, Health: true},
					podName1: {Name: podName1, Health: true},
					podName2: {Name: podName2, Health: true},
				},
				PeerMembers: map[string]v1alpha1.PDMember{
					podName0: {Name: podName0, Health: true},
					podName1: {Name: podName1, Health: true},
					podName2: {Name: podName2, Health: true},
				},
				Leader: v1alpha1.PDMember{Name: podName2, Health: true},
			},
		},
	}
}

func getPDPods() []*corev1.Pod {
	lc := label.New().Instance(upgradeInstanceName).PD().Labels()
	lc[apps.ControllerRevisionHashLabelKey] = "1"
	lu := label.New().Instance(upgradeInstanceName).PD().Labels()
	lu[apps.ControllerRevisionHashLabelKey] = "2"
	pods := []*corev1.Pod{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PdPodName(upgradeTcName, 0),
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
				Name:      PdPodName(upgradeTcName, 1),
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
				Name:      PdPodName(upgradeTcName, 2),
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
