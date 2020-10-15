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
	"testing"

	"github.com/pingcap/tidb-operator/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/pkg/dmapi"
	podinformers "k8s.io/client-go/informers/core/v1"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func TestMasterUpgraderUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name              string
		changeFn          func(*v1alpha1.DMCluster)
		changePods        func(pods []*corev1.Pod)
		changeOldSet      func(set *apps.StatefulSet)
		transferLeaderErr bool
		errExpectFn       func(*GomegaWithT, error)
		expectFn          func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet)
	}

	testFn := func(test *testcase) {
		t.Log(test.name)
		upgrader, masterControl, _, podInformer := newMasterUpgrader()
		dc := newDMClusterForMasterUpgrader()
		leaderPodName := DMMasterPodName(upgradeTcName, 1)
		masterPeerClient := controller.NewFakeMasterPeerClient(masterControl, dc, leaderPodName)

		if test.changeFn != nil {
			test.changeFn(dc)
		}

		if test.transferLeaderErr {
			masterPeerClient.AddReaction(dmapi.EvictLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to transfer leader")
			})
		} else {
			masterPeerClient.AddReaction(dmapi.EvictLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, nil
			})
		}

		pods := getMasterPods()
		if test.changePods != nil {
			test.changePods(pods)
		}
		for i := range pods {
			podInformer.Informer().GetIndexer().Add(pods[i])
		}

		newSet := newStatefulSetForMasterUpgrader()
		oldSet := newSet.DeepCopy()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}
		SetStatefulSetLastAppliedConfigAnnotation(oldSet)

		newSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(3)

		err := upgrader.Upgrade(dc, oldSet, newSet)
		test.errExpectFn(g, err)
		test.expectFn(g, dc, newSet)
	}

	tests := []testcase{
		{
			name: "normal upgrade",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
			},
			changePods:        nil,
			changeOldSet:      nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(1)))
			},
		},
		{
			name: "modify oldSet update strategy to OnDelete",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
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
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name: "set oldSet's RollingUpdate strategy to nil",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
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
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "newSet template changed",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.Template.Spec.Containers[0].Image = "dm-test-image:old"
			},
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "dm-master scaling",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
				dc.Status.Master.Phase = v1alpha1.ScalePhase
			},
			changePods: nil,
			changeOldSet: func(set *apps.StatefulSet) {
				set.Spec.Template.Spec.Containers[0].Image = "dm-test-image:old"
			},
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.ScalePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "update revision equals current revision",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
				dc.Status.Master.StatefulSet.UpdateRevision = dc.Status.Master.StatefulSet.CurrentRevision
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "skip to wait all members health",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
				dc.Status.Master.Members[DMMasterPodName(upgradeTcName, 2)] = v1alpha1.MasterMember{Name: DMMasterPodName(upgradeTcName, 2), Health: false}
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err.Error()).To(Equal(fmt.Sprintf("dmcluster: [default/upgrader]'s dm-master upgraded pod: [%s] is not ready", DMMasterPodName(upgradeTcName, 2))))
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "transfer leader",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
				dc.Status.Master.Leader = v1alpha1.MasterMember{Name: DMMasterPodName(upgradeTcName, 1), Health: true}
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
		{
			name: "dm-master sync failed",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = false
			},
			changePods:        nil,
			transferLeaderErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(3)))
			},
		},
		{
			name: "error when transfer leader",
			changeFn: func(dc *v1alpha1.DMCluster) {
				dc.Status.Master.Synced = true
				dc.Status.Master.Leader = v1alpha1.MasterMember{Name: DMMasterPodName(upgradeTcName, 1), Health: true}
			},
			changePods:        nil,
			transferLeaderErr: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster, newSet *apps.StatefulSet) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(pointer.Int32Ptr(2)))
			},
		},
	}

	for _, test := range tests {
		testFn(&test)
	}

}

func newMasterUpgrader() (DMUpgrader, *dmapi.FakeMasterControl, *controller.FakePodControl, podinformers.PodInformer) {
	fakeDeps := controller.NewFakeDependencies()
	upgrader := &masterUpgrader{deps: fakeDeps}
	masterControl := fakeDeps.DMMasterControl.(*dmapi.FakeMasterControl)
	podControl := fakeDeps.PodControl.(*controller.FakePodControl)
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	return upgrader, masterControl, podControl, podInformer
}

func newStatefulSetForMasterUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.DMMasterMemberName(upgradeTcName),
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "dm-master",
							Image: "dm-test-image",
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

func newDMClusterForMasterUpgrader() *v1alpha1.DMCluster {
	podName0 := DMMasterPodName(upgradeTcName, 0)
	podName1 := DMMasterPodName(upgradeTcName, 1)
	podName2 := DMMasterPodName(upgradeTcName, 2)
	return &v1alpha1.DMCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DMCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeTcName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID(upgradeTcName),
			Labels:    label.NewDM().Instance(upgradeInstanceName),
		},
		Spec: v1alpha1.DMClusterSpec{
			Master: v1alpha1.MasterSpec{
				BaseImage:        "dm-test-image",
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			Version: "v2.0.0-rc.2",
		},
		Status: v1alpha1.DMClusterStatus{
			Master: v1alpha1.MasterStatus{
				Phase: v1alpha1.NormalPhase,
				StatefulSet: &apps.StatefulSetStatus{
					CurrentRevision: "1",
					UpdateRevision:  "2",
					ReadyReplicas:   3,
					Replicas:        3,
					CurrentReplicas: 2,
					UpdatedReplicas: 1,
				},
				Members: map[string]v1alpha1.MasterMember{
					podName0: {Name: podName0, Health: true},
					podName1: {Name: podName1, Health: true},
					podName2: {Name: podName2, Health: true},
				},
				Leader: v1alpha1.MasterMember{Name: podName2, Health: true},
			},
		},
	}
}

func getMasterPods() []*corev1.Pod {
	lc := label.NewDM().Instance(upgradeInstanceName).DMMaster().Labels()
	lc[apps.ControllerRevisionHashLabelKey] = "1"
	lu := label.NewDM().Instance(upgradeInstanceName).DMMaster().Labels()
	lu[apps.ControllerRevisionHashLabelKey] = "2"
	pods := []*corev1.Pod{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      DMMasterPodName(upgradeTcName, 0),
				Namespace: corev1.NamespaceDefault,
				Labels:    lc,
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      DMMasterPodName(upgradeTcName, 1),
				Namespace: corev1.NamespaceDefault,
				Labels:    lc,
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      DMMasterPodName(upgradeTcName, 2),
				Namespace: corev1.NamespaceDefault,
				Labels:    lu,
			},
		},
	}
	return pods
}
