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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestTiDBMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		prepare                  func(cluster *v1alpha1.TidbCluster)
		errWhenCreateStatefulSet bool
		err                      bool
		setCreated               bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTiDB()
		tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
			"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}

		ns := tc.GetNamespace()
		tcName := tc.GetName()
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		tmm, fakeSetControl, _, _ := newFakeTiDBMemberManager()

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := tmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))

		tc1, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
		if test.setCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}
	}

	tests := []testcase{
		{
			name:                     "normal",
			prepare:                  nil,
			errWhenCreateStatefulSet: false,
			err:                      false,
			setCreated:               true,
		},
		{
			name: "tikv is not available",
			prepare: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
			},
			errWhenCreateStatefulSet: false,
			err:                      true,
			setCreated:               true,
		},
		{
			name:                     "error when create statefulset",
			prepare:                  nil,
			errWhenCreateStatefulSet: true,
			err:                      true,
			setCreated:               false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiDBMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		modify                   func(cluster *v1alpha1.TidbCluster)
		errWhenUpdateStatefulSet bool
		statusChange             func(*apps.StatefulSet)
		err                      bool
		expectStatefulSetFn      func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTiDB()
		tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
			"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}

		ns := tc.GetNamespace()
		tcName := tc.GetName()

		tmm, fakeSetControl, _, _ := newFakeTiDBMemberManager()

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "tidb-1"
				set.Status.UpdateRevision = "tidb-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = &observedGeneration
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := tmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(err).NotTo(HaveOccurred())
		_, err = tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = tmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 5
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			errWhenUpdateStatefulSet: false,
			err:                      false,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 5
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			errWhenUpdateStatefulSet: true,
			err:                      true,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "enable separate slowlog on the fly",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.SeparateSlowLog = true
			},
			errWhenUpdateStatefulSet: false,
			err:                      false,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers).To(HaveLen(2))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiDBMemberManagerTiDBStatefulSetIsUpgrading(t *testing.T) {
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
		pmm, _, podIndexer, _ := newFakeTiDBMemberManager()
		tc := newTidbClusterForTiDB()
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{
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
					Name:        ordinalPodName(v1alpha1.TiDBMemberType, tc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).TiDB().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			podIndexer.Add(pod)
		}
		b, err := pmm.tidbStatefulSetIsUpgradingFn(pmm.podLister, set, tc)
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
				set.Status.ObservedGeneration = func() *int64 { var i int64; i = 1000; return &i }()
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

func TestTiDBMemberManagerSyncTidbClusterStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name        string
		updateTC    func(*v1alpha1.TidbCluster)
		upgradingFn func(corelisters.PodLister, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
		healthInfo  map[string]bool
		errExpectFn func(*GomegaWithT, error)
		tcExpectFn  func(*GomegaWithT, *v1alpha1.TidbCluster)
	}
	status := apps.StatefulSetStatus{
		Replicas: int32(3),
	}
	now := metav1.Time{Time: time.Now()}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		tc.Status.PD.Phase = v1alpha1.NormalPhase
		tc.Status.TiKV.Phase = v1alpha1.NormalPhase
		set := &apps.StatefulSet{
			Status: status,
		}
		if test.updateTC != nil {
			test.updateTC(tc)
		}
		pmm, _, _, tidbControl := newFakeTiDBMemberManager()

		if test.upgradingFn != nil {
			pmm.tidbStatefulSetIsUpgradingFn = test.upgradingFn
		}
		if test.healthInfo != nil {
			tidbControl.SetHealth(test.healthInfo)
		}

		err := pmm.syncTidbClusterStatus(tc, set)
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
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, fmt.Errorf("whether upgrading failed")
			},
			healthInfo: map[string]bool{},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "whether upgrading failed")).To(BeTrue())
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name:     "statefulset is upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
		{
			name: "statefulset is upgrading but pd is upgrading",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name: "statefulset is upgrading but tikv is upgrading",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name:     "statefulset is not upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiDB.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name:     "get health empty",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(0))
			},
		},
		{
			name: "set LastTransitionTime first time",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{}
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo: map[string]bool{
				"333": true,
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(1))
				g.Expect(tc.Status.TiDB.Members["333"].LastTransitionTime.Time.IsZero()).To(BeFalse())
			},
		},
		{
			name: "state not change, LastTransitionTime not change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{}
				tc.Status.TiDB.Members["333"] = v1alpha1.TiDBMember{
					Health:             true,
					LastTransitionTime: now,
				}
			},
			healthInfo: map[string]bool{
				"333": true,
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(1))
				g.Expect(tc.Status.TiDB.Members["333"].LastTransitionTime).To(Equal(now))
			},
		},
		{
			name: "state change, LastTransitionTime change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{}
				tc.Status.TiDB.Members["333"] = v1alpha1.TiDBMember{
					Health:             false,
					LastTransitionTime: now,
				}
			},
			healthInfo: map[string]bool{
				"333": true,
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(1))
				g.Expect(tc.Status.TiDB.Members["333"].LastTransitionTime).NotTo(Equal(now))
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func newFakeTiDBMemberManager() (*tidbMemberManager, *controller.FakeStatefulSetControl, cache.Indexer, *controller.FakeTiDBControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer, tcInformer)
	tidbUpgrader := NewFakeTiDBUpgrader()
	tidbFailover := NewFakeTiDBFailover()
	tidbControl := controller.NewFakeTiDBControl()
	operatorImage := "pingcap/tidb-operator:latest"

	tmm := &tidbMemberManager{
		setControl,
		svcControl,
		tidbControl,
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		tidbUpgrader,
		true,
		operatorImage,
		tidbFailover,
		tidbStatefulSetIsUpgrading,
	}
	return tmm, setControl, podInformer.Informer().GetIndexer(), tidbControl
}

func newTidbClusterForTiDB() *v1alpha1.TidbCluster {
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
			TiDB: v1alpha1.TiDBSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: v1alpha1.TiDBMemberType.String(),
					Requests: &v1alpha1.ResourceRequirement{
						CPU:    "1",
						Memory: "2Gi",
					},
				},
				Replicas: 3,
			},
		},
	}
}
