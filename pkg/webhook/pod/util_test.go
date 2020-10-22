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

package pod

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestCheckPDFormerPodStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		stsReplicas   int32
		name          string
		targetOrdinal int32
		deleteSlots   []int32
		permit        bool
	}
	tests := []testcase{
		{
			stsReplicas:   5,
			name:          "last target ordinal",
			targetOrdinal: 4,
			deleteSlots:   []int32{},
			permit:        true,
		},
		{
			stsReplicas:   5,
			name:          "FirstTargetOrdinal",
			targetOrdinal: 0,
			deleteSlots:   []int32{},
			permit:        true,
		},
		{
			stsReplicas:   4,
			name:          "mid target ordinal, check success",
			targetOrdinal: 1,
			deleteSlots:   []int32{0},
			permit:        true,
		},
		{
			stsReplicas:   4,
			name:          "mid target ordinal, check success",
			targetOrdinal: 1,
			deleteSlots:   []int32{2},
			permit:        true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeCli, _ := newFakeComponent()
			pdControl := pdapi.NewFakePDControl(kubeCli)
			slots := sets.NewInt32(test.deleteSlots...)
			tc := newTidbClusterForPodAdmissionControl(test.stsReplicas, test.stsReplicas)
			fakePDClient := controller.NewFakePDClient(pdControl, tc)
			sts := buildTargetStatefulSet(tc, v1alpha1.PDMemberType)
			err := helper.SetDeleteSlots(sts, slots)
			g.Expect(err).NotTo(HaveOccurred())
			healthInfo := &pdapi.HealthInfo{}
			for i := range helper.GetPodOrdinals(test.stsReplicas, sts) {
				healthInfo.Healths = append(healthInfo.Healths, pdapi.MemberHealth{
					Name:   memberUtils.PdPodName(tc.Name, i),
					Health: true,
				})
				pod := buildPod(tc, v1alpha1.PDMemberType, i)
				pod.Labels[apps.ControllerRevisionHashLabelKey] = sts.Status.UpdateRevision
				kubeCli.CoreV1().Pods(tc.Namespace).Create(pod)
			}
			fakePDClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
				return healthInfo, nil
			})

			err = checkFormerPDPodStatus(kubeCli, fakePDClient, tc, sts, test.targetOrdinal)
			if test.permit {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		})
	}
}

func TestCheckTiKVFormerPodStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		stsReplicas   int32
		name          string
		targetOrdinal int32
		deleteSlots   []int32
		permit        bool
	}
	tests := []testcase{
		{
			stsReplicas:   5,
			name:          "last target ordinal",
			targetOrdinal: 4,
			deleteSlots:   []int32{},
			permit:        true,
		},
		{
			stsReplicas:   5,
			name:          "FirstTargetOrdinal",
			targetOrdinal: 0,
			deleteSlots:   []int32{},
			permit:        true,
		},
		{
			stsReplicas:   4,
			name:          "mid target ordinal, check success",
			targetOrdinal: 1,
			deleteSlots:   []int32{0},
			permit:        true,
		},
		{
			stsReplicas:   4,
			name:          "mid target ordinal, check success",
			targetOrdinal: 1,
			deleteSlots:   []int32{2},
			permit:        true,
		},
	}

	for _, test := range tests {
		t.Log("test: ", test.name)
		kubeCli, _ := newFakeComponent()
		slots := sets.NewInt32(test.deleteSlots...)
		tc := newTidbClusterForPodAdmissionControl(test.stsReplicas, test.stsReplicas)
		sts := buildTargetStatefulSet(tc, v1alpha1.TiKVMemberType)
		err := helper.SetDeleteSlots(sts, slots)
		g.Expect(err).NotTo(HaveOccurred())
		for i := range helper.GetPodOrdinals(test.stsReplicas, sts) {
			pod := buildPod(tc, v1alpha1.TiKVMemberType, i)
			pod.Labels[apps.ControllerRevisionHashLabelKey] = sts.Status.UpdateRevision
			kubeCli.CoreV1().Pods(tc.Namespace).Create(pod)
		}
		desc := controllerDesc{
			name:      tc.Name,
			namespace: tc.Namespace,
			kind:      tc.Kind,
		}
		err = checkFormerTiKVPodStatus(kubeCli, desc, test.targetOrdinal, tc.Spec.TiKV.Replicas, sts, buildStoresInfo(tc, sts))
		if test.permit {
			g.Expect(err).NotTo(HaveOccurred())
		} else {
			g.Expect(err).Should(HaveOccurred())
		}
	}
}
func TestIsTiKVReadyToUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)
	store0 := &pdapi.StoreInfo{
		Store: &pdapi.MetaStore{
			Store: &metapb.Store{},
		},
		Status: &pdapi.StoreStatus{LeaderCount: 0},
	}

	store1 := &pdapi.StoreInfo{
		Store: &pdapi.MetaStore{
			Store: &metapb.Store{},
		},
		Status: &pdapi.StoreStatus{LeaderCount: 1},
	}

	tests := []struct {
		name    string
		pod     *core.Pod
		store   *pdapi.StoreInfo
		timeout time.Duration
		result  bool
	}{
		{
			name:    "leader count is zero should be ready",
			pod:     &core.Pod{},
			store:   store0,
			timeout: time.Second,
			result:  true,
		},
		{
			name:    "not EvictLeaderBeginTime should be false",
			pod:     &core.Pod{},
			store:   store1,
			timeout: time.Second,
			result:  false,
		},
		{
			name: "timeout should be true",
			pod: &core.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						EvictLeaderBeginTime: time.Now().AddDate(0, 0, -1 /* one day */).Format(time.RFC3339),
					},
				},
			},
			store:   store1,
			timeout: time.Hour * 12,
			result:  true,
		},
		{
			name: "not timeout should be false",
			pod: &core.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						EvictLeaderBeginTime: time.Now().AddDate(0, 0, -1 /* one day */).Format(time.RFC3339),
					},
				},
			},
			store:   store1,
			timeout: time.Hour * 25,
			result:  false,
		},
	}

	for _, test := range tests {
		t.Log("test: ", test.name)
		get := isTiKVReadyToUpgrade(test.pod, test.store, test.timeout)
		g.Expect(get).Should(Equal(test.result))
	}
}

func TestEvictLeader(t *testing.T) {
	g := NewGomegaWithT(t)
	kubeCli := kubefake.NewSimpleClientset()
	pod := &core.Pod{}
	pod.Namespace = "ns"
	pod.Name = "name"
	_, err := kubeCli.CoreV1().Pods(pod.Namespace).Create(pod)
	g.Expect(err).Should(BeNil())
	store := &pdapi.StoreInfo{
		Store: &pdapi.MetaStore{
			Store: &metapb.Store{
				Id: 1,
			},
		},
		Status: &pdapi.StoreStatus{LeaderCount: 1},
	}
	pdClient := pdapi.NewFakePDClient()

	err = beginEvictLeader(kubeCli, store.Store.Id, pod, pdClient)
	g.Expect(err).Should(BeNil())
	err = endEvictLeader(store, pdClient)
	g.Expect(err).Should(BeNil())
}

func TestAddEvictLeaderAnnotation(t *testing.T) {
	g := NewGomegaWithT(t)
	kubeCli := kubefake.NewSimpleClientset()

	pod := &core.Pod{}
	pod.Namespace = "ns"
	pod.Name = "name"
	err := addEvictLeaderAnnotation(kubeCli, pod)
	g.Expect(err).ShouldNot(BeNil()) // not exist

	// create first and add again should success.
	_, err = kubeCli.CoreV1().Pods(pod.Namespace).Create(pod)
	g.Expect(err).Should(BeNil())
	err = addEvictLeaderAnnotation(kubeCli, pod)
	g.Expect(err).Should(BeNil())
}

func buildStoresInfo(tc *v1alpha1.TidbCluster, sts *apps.StatefulSet) *pdapi.StoresInfo {
	ssi := &pdapi.StoresInfo{
		Stores: []*pdapi.StoreInfo{},
	}
	for i := range helper.GetPodOrdinals(tc.Spec.TiKV.Replicas, sts) {
		si := buildStoreInfo(tc, i)
		ssi.Stores = append(ssi.Stores, si)
	}
	return ssi
}

func buildStoreInfo(tc *v1alpha1.TidbCluster, ordinal int32) *pdapi.StoreInfo {
	si := &pdapi.StoreInfo{
		Store: &pdapi.MetaStore{
			Store: &metapb.Store{
				Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tc.Name, ordinal, tc.Name, tc.Namespace),
			},
			StateName: v1alpha1.TiKVStateUp,
		},
	}
	return si
}

func buildPod(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, ordinal int32) *corev1.Pod {
	pod := &corev1.Pod{}
	pod.Namespace = tc.Namespace
	pod.Name = operatorUtils.GetPodName(tc, memberType, ordinal)
	pod.Labels = map[string]string{}
	return pod
}

func buildTargetStatefulSet(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) *apps.StatefulSet {
	sts := &apps.StatefulSet{}
	sts.Name = fmt.Sprintf("%s-%s", tcName, memberType.String())
	sts.Namespace = tc.Namespace
	sts.Status.UpdateRevision = "1"
	switch memberType {
	case v1alpha1.PDMemberType:
		sts.Spec.Replicas = &tc.Spec.PD.Replicas
		tc.Status.PD.StatefulSet = &sts.Status
	case v1alpha1.TiKVMemberType:
		sts.Spec.Replicas = &tc.Spec.TiKV.Replicas
		tc.Status.TiKV.StatefulSet = &sts.Status
	}
	return sts
}

func newFakeComponent() (*kubefake.Clientset, cache.Indexer) {
	kubeCli := kubefake.NewSimpleClientset()
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	podInformer := kubeInformerFactory.Core().V1().Pods()
	return kubeCli, podInformer.Informer().GetIndexer()
}
