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
	"strconv"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/tikvapi"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	podinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/utils/pointer"
)

const (
	upgradeTcName       = "upgrader"
	upgradeInstanceName = "upgrader"
)

func TestTiKVUpgraderUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                string
		changeFn            func(*v1alpha1.TidbCluster)
		changeOldSet        func(set *apps.StatefulSet)
		changePods          func([]*corev1.Pod)
		beginEvictLeaderErr bool
		endEvictLeaderErr   bool
		getLeaderCountErr   bool
		leaderCount         int
		podName             string
		updatePodErr        bool
		modifyVolumesResult func() (bool /*should modify*/, error /*result of modify*/) // default to (true, nil)
		errExpectFn         func(*GomegaWithT, error)
		expectFn            func(*GomegaWithT, *v1alpha1.TidbCluster, *apps.StatefulSet, map[string]*corev1.Pod)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		upgrader, pdControl, podControl, podInformer, tikvControl, volumeModifier := newTiKVUpgrader()

		tc := newTidbClusterForTiKVUpgrader()
		if test.changeFn != nil {
			test.changeFn(tc)
		}

		oldSet := oldStatefulSetForTiKVUpgrader()
		if test.changeOldSet != nil {
			test.changeOldSet(oldSet)
		}
		newSet := newStatefulSetForTiKVUpgrader()

		pdClient := controller.NewFakePDClient(pdControl, tc)
		if test.beginEvictLeaderErr {
			pdClient.AddReaction(pdapi.BeginEvictLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to begin evict leader")
			})
		} else {
			pdClient.AddReaction(pdapi.BeginEvictLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, nil
			})
		}

		tikvClient := controller.NewFakeTiKVClient(tikvControl, tc, "upgrader-tikv-2")
		if len(test.podName) > 0 {
			tikvClient = controller.NewFakeTiKVClient(tikvControl, tc, test.podName)
		}
		if test.getLeaderCountErr {
			tikvClient.AddReaction(tikvapi.GetLeaderCountActionType, func(action *tikvapi.Action) (interface{}, error) {
				return 0, fmt.Errorf("failed to begin evict leader")
			})
		} else {
			tikvClient.AddReaction(tikvapi.GetLeaderCountActionType, func(action *tikvapi.Action) (interface{}, error) {
				return test.leaderCount, nil
			})
		}
		if test.endEvictLeaderErr {
			pdClient.AddReaction(pdapi.EndEvictLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to end evict leader")
			})
		} else {
			pdClient.AddReaction(pdapi.EndEvictLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, nil
			})
		}

		tikvPods := getTiKVPods(oldSet)
		if test.changePods != nil {
			test.changePods(tikvPods)
		}
		for _, pod := range tikvPods {
			podInformer.Informer().GetIndexer().Add(pod)
		}

		if test.updatePodErr {
			podControl.SetUpdatePodError(fmt.Errorf("failed to update pod"), 0)
		}

		// mock result of volume modification
		if test.modifyVolumesResult == nil {
			test.modifyVolumesResult = func() (bool, error) {
				return false, nil
			}
		}
		shouldModify, resultOfModify := test.modifyVolumesResult()
		volumeModifier.GetDesiredVolumesFunc = func(_ *v1alpha1.TidbCluster, _ v1alpha1.MemberType) ([]volumes.DesiredVolume, error) {
			return []volumes.DesiredVolume{}, nil
		}
		volumeModifier.ShouldModifyFunc = func(_ []volumes.ActualVolume) bool {
			return shouldModify
		}
		volumeModifier.ModifyFunc = func(_ []volumes.ActualVolume) error {
			return resultOfModify
		}

		err := upgrader.Upgrade(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		l, err := label.New().Instance(upgradeInstanceName).TiKV().Selector()
		g.Expect(err).NotTo(HaveOccurred())
		tikvPods, err = podInformer.Lister().Pods(tc.Namespace).List(l)
		g.Expect(err).NotTo(HaveOccurred())
		pods := map[string]*corev1.Pod{}
		for _, pod := range tikvPods {
			pods[pod.GetName()] = pod
		}
		test.expectFn(g, tc, newSet, pods)
	}

	tests := []*testcase{
		{
			name:     "modify oldSet update strategy to OnDelete",
			changeFn: nil,
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.OnDeleteStatefulSetStrategyType,
				}
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.OnDeleteStatefulSetStrategyType}))
			},
		},
		{
			name:     "set oldSet's RollingUpdate strategy to nil",
			changeFn: nil,
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
					Type: apps.RollingUpdateStatefulSetStrategyType,
				}
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(newSet.Spec.UpdateStrategy).To(Equal(apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType}))
			},
		},
		{
			name: "to upgrade the pod which ordinal is 2",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Synced = true
				// set leader to 0
				store := tc.Status.TiKV.Stores["3"]
				store.LeaderCount = 0
				tc.Status.TiKV.Stores["3"] = store
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 2) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Add(-1 * time.Minute).Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "to upgrade the pod which ordinal is 1",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
				// set leader to 0
				store := tc.Status.TiKV.Stores["2"]
				store.LeaderCount = 0
				tc.Status.TiKV.Stores["2"] = store
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Add(-1 * time.Minute).Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			podName:             "upgrader-tikv-1",
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
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				oldSet.Spec.Template.Spec.Containers[0].Image = "old-image"
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "update revision equals current revision",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.UpdateRevision = tc.Status.TiKV.StatefulSet.CurrentRevision
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "tikv can not upgrade when pd is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "tikv can not upgrade when tiflash is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "tikv can not upgrade when it is scaling",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.ScalePhase
				tc.Status.TiKV.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.ScalePhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "get last apply config error",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Synced = true
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				oldSet.SetAnnotations(map[string]string{LastAppliedConfigAnnotation: "fake apply config"})
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
		},
		{
			name: "begin evict leaders on store[2]",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("upgradeTiKVPod: evicting leader of pod upgrader-tikv-1 for tc default/upgrader"))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
				_, exist := pods[TikvPodName(upgradeTcName, 1)].Annotations[EvictLeaderBeginTime]
				g.Expect(exist).To(BeTrue())
			},
		},
		{
			name: "waiting leader count equals to 0",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			podName:             "upgrader-tikv-1",
			leaderCount:         10,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("upgradeTiKVPod: evicting leader of pod upgrader-tikv-1 for tc default/upgrader"))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name:              "get leader count error",
			getLeaderCountErr: true,
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			podName:             "upgrader-tikv-1",
			leaderCount:         10,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("upgradeTiKVPod: evicting leader of pod upgrader-tikv-1 for tc default/upgrader"))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},

		{
			name: "failed to begin evict leaders on store[2]",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods:          nil,
			beginEvictLeaderErr: true,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
				_, exist := pods[TikvPodName(upgradeTcName, 1)].Annotations[EvictLeaderBeginTime]
				g.Expect(exist).To(BeFalse())
			},
		},
		{
			name: "evict leaders time out",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Add(-2000 * time.Minute).Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			podName:             "upgrader-tikv-1",
			leaderCount:         10,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(1)))
			},
		},
		{
			name: "fast evict leaders when TiKV Stores are less than 2",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.Replicas = 1
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 1
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 1
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.Replicas = pointer.Int32Ptr(1)
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 0) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			podName:             "upgrader-tikv-0",
			leaderCount:         10,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(0)))
			},
		},
		{
			name: "can't fast evict leaders when current TiKV Stores are less than 2 but peer stores is exist",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = false
				tc.Status.TiKV.StatefulSet.Replicas = 1
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 1
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
				tc.Status.TiKV.PeerStores = map[string]v1alpha1.TiKVStore{"peer-0": {ID: "peer-0", State: v1alpha1.TiKVStateUp}}
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 1
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.Replicas = pointer.Int32Ptr(1)
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 0) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			podName:             "upgrader-tikv-0",
			leaderCount:         10,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("can not to be upgraded"))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
			},
		},
		{
			name: "end leader evict failed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   true,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "update pod failed after begin evict leaders on store[2]",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods:          nil,
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "update pod failed",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 2) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "pod is ready but not available",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
				if tc.Annotations == nil {
					tc.Annotations = map[string]string{}
				}
				tc.Annotations[annoKeyTiKVMinReadySeconds] = "300"
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Now().Format(time.RFC3339)}
						pod.Status.Conditions[0].LastTransitionTime = metav1.Now()
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "wait for volume modification for store[2]",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
				features.DefaultFeatureGate.Set(fmt.Sprintf("%s=true", features.VolumeModifying))
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Time{}.Format(time.RFC3339)} // skip evict leader
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			modifyVolumesResult: func() (bool, error) {
				return true, nil // volume modification is not finished
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("modifying volumes of pod upgrader-tikv-1 for tc default/upgrader"))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
		{
			name: "failed to modify volumes for store[2]",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet.CurrentReplicas = 2
				tc.Status.TiKV.StatefulSet.UpdatedReplicas = 1
				features.DefaultFeatureGate.Set(fmt.Sprintf("%s=true", features.VolumeModifying))
			},
			changeOldSet: func(oldSet *apps.StatefulSet) {
				mngerutils.SetStatefulSetLastAppliedConfigAnnotation(oldSet)
				oldSet.Status.CurrentReplicas = 2
				oldSet.Status.UpdatedReplicas = 1
				oldSet.Spec.UpdateStrategy.RollingUpdate.Partition = pointer.Int32Ptr(2)
			},
			changePods: func(pods []*corev1.Pod) {
				for _, pod := range pods {
					if pod.GetName() == TikvPodName(upgradeTcName, 1) {
						pod.Annotations = map[string]string{EvictLeaderBeginTime: time.Time{}.Format(time.RFC3339)} // skip evict leader
					}
				}
			},
			beginEvictLeaderErr: false,
			endEvictLeaderErr:   false,
			updatePodErr:        false,
			modifyVolumesResult: func() (bool, error) {
				return true, fmt.Errorf("test error") // volume modification is failed
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("upgradeTiKVPod: failed to modify volumes of pod upgrader-tikv-1 for tc default/upgrader, error: test error"))
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, pods map[string]*corev1.Pod) {
				g.Expect(*newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(2)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func newTiKVUpgrader() (TiKVUpgrader, *pdapi.FakePDControl, *controller.FakePodControl, podinformers.PodInformer, *tikvapi.FakeTiKVControl, *volumes.FakePodVolumeModifier) {
	fakeDeps := controller.NewFakeDependencies()
	pdControl := fakeDeps.PDControl.(*pdapi.FakePDControl)
	tikvControl := fakeDeps.TiKVControl.(*tikvapi.FakeTiKVControl)
	podControl := fakeDeps.PodControl.(*controller.FakePodControl)
	podInformer := fakeDeps.KubeInformerFactory.Core().V1().Pods()
	volumeModifier := &volumes.FakePodVolumeModifier{}
	return &tikvUpgrader{deps: fakeDeps, volumeModifier: volumeModifier}, pdControl, podControl, podInformer, tikvControl, volumeModifier
}

func newStatefulSetForTiKVUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-tikv",
			Namespace: metav1.NamespaceDefault,
			Labels:    label.New().Instance(upgradeInstanceName).TiKV().Labels(),
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tikv",
							Image: "tikv-test-image",
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

func oldStatefulSetForTiKVUpgrader() *apps.StatefulSet {
	set := newStatefulSetForTiKVUpgrader()
	set.Status = apps.StatefulSetStatus{
		Replicas:        3,
		CurrentReplicas: 3,
		UpdatedReplicas: 0,
		CurrentRevision: "1",
		UpdateRevision:  "2",
	}
	return set
}

func newTidbClusterForTiKVUpgrader() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeTcName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID(upgradeTcName),
			Labels:    label.New().Instance(upgradeInstanceName).TiKV().Labels(),
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
		},
		Status: v1alpha1.TidbClusterStatus{
			TiKV: v1alpha1.TiKVStatus{
				Synced:       true,
				BootStrapped: true,
				Phase:        v1alpha1.UpgradePhase,
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
						PodName:     TikvPodName(upgradeTcName, 0),
						LeaderCount: 10,
						State:       "Up",
					},
					"2": {
						ID:          "2",
						PodName:     TikvPodName(upgradeTcName, 1),
						LeaderCount: 10,
						State:       "Up",
					},
					"3": {
						ID:          "3",
						PodName:     TikvPodName(upgradeTcName, 2),
						LeaderCount: 10,
						State:       "Up",
					},
				},
			},
		},
	}
}

func getTiKVPods(set *apps.StatefulSet) []*corev1.Pod {
	var pods []*corev1.Pod
	for i := 0; i < int(set.Status.Replicas); i++ {
		l := label.New().Instance(upgradeInstanceName).TiKV().Labels()
		if i+1 <= int(set.Status.CurrentReplicas) {
			l[apps.ControllerRevisionHashLabelKey] = set.Status.CurrentRevision
		} else {
			l[apps.ControllerRevisionHashLabelKey] = set.Status.UpdateRevision
		}
		l[label.StoreIDLabelKey] = strconv.Itoa(i + 1)
		pods = append(pods, &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      TikvPodName(upgradeTcName, int32(i)),
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
