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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/utils/pointer"
)

func TestTiCDCMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name    string
		errSync bool
		err     bool
		tls     bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForCDC()
		if test.tls {
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
		}

		oldSpec := tc.Spec

		tmm, fakeSetControl, _, _ := newFakeTiCDCMemberManager()

		if test.errSync {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := tmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))
	}

	tests := []testcase{
		{
			name:    "normal",
			errSync: false,
			err:     false,
		},
		{
			name:    "normal with tls",
			errSync: false,
			err:     false,
			tls:     true,
		},
		{
			name:    "error when sync",
			errSync: true,
			err:     true,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiCDCMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.TidbCluster)
		errSync             bool
		statusChange        func(*apps.StatefulSet)
		status              v1alpha1.MemberPhase
		err                 bool
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForCDC()
		tc.Status.TiCDC.Phase = v1alpha1.NormalPhase
		tc.Status.TiCDC.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}

		ns := tc.GetNamespace()
		tcName := tc.GetName()

		tmm, fakeSetControl, _, _ := newFakeTiCDCMemberManager()

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "ticdc-1"
				set.Status.UpdateRevision = "ticdc-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := tmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tc.Status.TiCDC.Phase).To(Equal(test.status))

		_, err = tmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCDCMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errSync {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = tmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := tmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCDCMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiCDC.Replicas = 5
			},
			errSync: false,
			err:     false,
			status:  v1alpha1.NormalPhase,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiCDC.Replicas = 5
			},
			errSync: true,
			err:     true,
			status:  v1alpha1.NormalPhase,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiCDCMemberManagerTiCDCStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		setUpdate       func(*apps.StatefulSet)
		hasPod          bool
		updatePod       func(*corev1.Pod)
		errExpectFn     func(*GomegaWithT, error)
		expectUpgrading bool
	}
	testFn := func(test *testcase, t *testing.T) {
		pmm, _, _, indexers := newFakeTiCDCMemberManager()
		tc := newTidbClusterForCDC()
		tc.Status.TiCDC.StatefulSet = &apps.StatefulSetStatus{
			UpdateRevision: "v3",
		}

		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
		if test.setUpdate != nil {
			test.setUpdate(set)
		}

		if test.hasPod {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ordinalPodName(v1alpha1.TiCDCMemberType, tc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.New().Instance(tc.GetInstanceName()).TiCDC().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			indexers.pod.Add(pod)
		}
		b, err := pmm.statefulSetIsUpgradingFn(
			pmm.deps.PodLister,
			pmm.deps.PDControl,
			set, tc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		if test.expectUpgrading {
			g.Expect(b).To(BeTrue())
		} else {
			g.Expect(b).NotTo(BeTrue())
		}
	}
	tests := []testcase{
		{
			name: "stateful set is upgrading",
			setUpdate: func(set *apps.StatefulSet) {
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v2"
				set.Status.ObservedGeneration = 1000
			},
			hasPod:          false,
			updatePod:       nil,
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:            "pod don't have revision hash",
			setUpdate:       nil,
			hasPod:          true,
			updatePod:       nil,
			errExpectFn:     nil,
			expectUpgrading: false,
		},
		{
			name:      "pod have revision hash, not equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v2"
			},
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:      "pod have revision hash, equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v3"
			},
			errExpectFn:     nil,
			expectUpgrading: false,
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestTiCDCMemberManagerSyncTidbClusterStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name        string
		updateTC    func(*v1alpha1.TidbCluster)
		updateSts   func(*apps.StatefulSet)
		upgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
		healthInfo  map[string]bool
		errExpectFn func(*GomegaWithT, error)
		tcExpectFn  func(*GomegaWithT, *v1alpha1.TidbCluster)
	}
	spec := apps.StatefulSetSpec{
		Replicas: pointer.Int32Ptr(3),
	}
	status := apps.StatefulSetStatus{
		Replicas: int32(3),
	}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForCDC()
		tc.Spec.TiCDC.Replicas = int32(3)
		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{},
			Spec:       spec,
			Status:     status,
		}
		if test.updateTC != nil {
			test.updateTC(tc)
		}
		if test.updateSts != nil {
			test.updateSts(set)
		}
		pmm, _, tidbControl, _ := newFakeTiCDCMemberManager()

		if test.upgradingFn != nil {
			pmm.statefulSetIsUpgradingFn = test.upgradingFn
		}
		if test.healthInfo != nil {
			tidbControl.SetHealth(test.healthInfo)
		}

		err := pmm.syncTiCDCStatus(tc, set)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		if test.tcExpectFn != nil {
			test.tcExpectFn(g, tc)
		}
	}
	tests := []testcase{
		{
			name:     "whether statefulset is upgrading returns failed",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, fmt.Errorf("whether upgrading failed")
			},
			healthInfo: map[string]bool{},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "whether upgrading failed")).To(BeTrue())
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name:     "statefulset is upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
		{
			name:     "statefulset is not upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name:     "get health empty",
			updateTC: nil,
			updateSts: func(sts *apps.StatefulSet) {
				sts.Status = apps.StatefulSetStatus{
					Replicas: 0,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(0)))
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func newFakeTiCDCMemberManager() (*ticdcMemberManager, *controller.FakeStatefulSetControl, *controller.FakeTiDBControl, *fakeIndexers) {
	fakeDeps := controller.NewFakeDependencies()
	tmm := &ticdcMemberManager{
		deps: fakeDeps,
	}
	tmm.statefulSetIsUpgradingFn = ticdcStatefulSetIsUpgrading
	indexers := &fakeIndexers{
		pod:    fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer(),
		tc:     fakeDeps.InformerFactory.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer(),
		svc:    fakeDeps.KubeInformerFactory.Core().V1().Services().Informer().GetIndexer(),
		eps:    fakeDeps.KubeInformerFactory.Core().V1().Endpoints().Informer().GetIndexer(),
		secret: fakeDeps.KubeInformerFactory.Core().V1().Secrets().Informer().GetIndexer(),
		set:    fakeDeps.KubeInformerFactory.Apps().V1().StatefulSets().Informer().GetIndexer(),
	}
	setControl := fakeDeps.StatefulSetControl.(*controller.FakeStatefulSetControl)
	tidbControl := fakeDeps.TiDBControl.(*controller.FakeTiDBControl)
	return tmm, setControl, tidbControl, indexers
}

func newTidbClusterForCDC() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiCDC: &v1alpha1.TiCDCSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: v1alpha1.TiCDCMemberType.String(),
				},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
				Replicas: 3,
			},
		},
	}
}
