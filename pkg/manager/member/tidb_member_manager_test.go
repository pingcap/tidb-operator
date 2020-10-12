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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
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

		tmm, fakeSetControl, _, _, _ := newFakeTiDBMemberManager()

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
			setCreated:               false,
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

		tmm, fakeSetControl, _, _, _ := newFakeTiDBMemberManager()

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "tidb-1"
				set.Status.UpdateRevision = "tidb-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
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
				tc.Spec.TiDB.SeparateSlowLog = pointer.BoolPtr(true)
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
		pmm, _, podIndexer, _, _ := newFakeTiDBMemberManager()
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
					Labels:      label.New().Instance(tc.GetInstanceName()).TiDB().Labels(),
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

func TestTiDBMemberManagerSyncTidbClusterStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name        string
		updateTC    func(*v1alpha1.TidbCluster)
		updateSts   func(*apps.StatefulSet)
		upgradingFn func(corelisters.PodLister, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
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
	now := metav1.Time{Time: time.Now()}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		tc.Spec.TiDB.Replicas = int32(3)
		tc.Status.PD.Phase = v1alpha1.NormalPhase
		tc.Status.TiKV.Phase = v1alpha1.NormalPhase
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
		pmm, _, _, tidbControl, _ := newFakeTiDBMemberManager()

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
			updateSts: func(sts *apps.StatefulSet) {
				sts.Status = apps.StatefulSetStatus{
					Replicas: 0,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiDB.StatefulSet.Replicas).To(Equal(int32(0)))
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
				"test-tidb-2": true,
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(3))
				g.Expect(tc.Status.TiDB.Members["test-tidb-2"].LastTransitionTime.Time.IsZero()).To(BeFalse())
			},
		},
		{
			name: "state not change, LastTransitionTime not change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{}
				tc.Status.TiDB.Members["test-tidb-2"] = v1alpha1.TiDBMember{
					Health:             true,
					LastTransitionTime: now,
				}
			},
			healthInfo: map[string]bool{
				"test-tidb-2": true,
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(3))
				g.Expect(tc.Status.TiDB.Members["test-tidb-2"].LastTransitionTime).To(Equal(now))
			},
		},
		{
			name: "state change, LastTransitionTime change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{}
				tc.Status.TiDB.Members["test-tidb-2"] = v1alpha1.TiDBMember{
					Health:             false,
					LastTransitionTime: now,
				}
			},
			healthInfo: map[string]bool{
				"test-tidb-2": true,
			},
			upgradingFn: func(lister corelisters.PodLister, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiDB.Members)).To(Equal(3))
				g.Expect(tc.Status.TiDB.Members["test-tidb-2"].LastTransitionTime).NotTo(Equal(now))
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestTiDBMemberManagerSyncTidbService(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		prepare         func(*v1alpha1.TidbCluster, *fakeIndexers)
		errOnCreate     bool
		errOnUpdate     bool
		expectFn        func(*GomegaWithT, error, *corev1.Service)
		expectSvcAbsent bool
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTiDB()
		tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
			"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}

		tmm, _, _, _, indexers := newFakeTiDBMemberManager()
		if test.prepare != nil {
			test.prepare(tc, indexers)
		}

		if test.errOnCreate {
			tmm.svcControl.(*controller.FakeServiceControl).SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnUpdate {
			tmm.svcControl.(*controller.FakeServiceControl).SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		syncErr := tmm.syncTiDBService(tc)
		svc, err := tmm.svcLister.Services(tc.Namespace).Get(controller.TiDBMemberName(tc.Name))
		if test.expectSvcAbsent {
			g.Expect(err).To(WithTransform(errors.IsNotFound, BeTrue()))
		} else {
			g.Expect(err).NotTo(HaveOccurred())
			test.expectFn(g, syncErr, svc)
		}
	}
	policyLocal := corev1.ServiceExternalTrafficPolicyTypeLocal
	tests := []*testcase{
		{
			name: "Create service",
			prepare: func(tc *v1alpha1.TidbCluster, _ *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				}
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc).NotTo(BeNil())
			},
		},
		{
			name: "Update service",
			prepare: func(tc *v1alpha1.TidbCluster, indexers *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Annotations: map[string]string{
							"lb-type": "new-lb",
						},
					},
				}
				_ = indexers.svc.Add(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
						Name:            controller.TiDBMemberName(tc.Name),
						Namespace:       corev1.NamespaceDefault,
						Annotations: map[string]string{
							"lb-type": "old-lb",
							"k":       "v",
						},
					},
				})
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Annotations).To(HaveKeyWithValue("lb-type", "new-lb"), "Expected updating service will reconcile annotations")
				g.Expect(svc.Annotations).To(HaveKeyWithValue("k", "v"), "Expected updating service will not affect additional annotations")
			},
		},
		{
			name:            "Do not create TiDB service when the spec is absent",
			expectSvcAbsent: true,
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "Do not update TiDB service when the spec is absent",
			prepare: func(tc *v1alpha1.TidbCluster, indexers *fakeIndexers) {
				_ = indexers.svc.Add(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      controller.TiDBMemberName(tc.Name),
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							"helm.sh/chart": "tidb-cluster",
						},
					},
				})
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Labels).To(HaveKeyWithValue("helm.sh/chart", "tidb-cluster"))
			},
		},
		{
			name: "Create service error",
			prepare: func(tc *v1alpha1.TidbCluster, _ *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				}
			},
			errOnCreate:     true,
			expectSvcAbsent: true,
		},
		{
			name: "Update service error",
			prepare: func(tc *v1alpha1.TidbCluster, indexers *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				}
				_ = indexers.svc.Add(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      controller.TiDBMemberName(tc.Name),
					},
				})
			},
			errOnUpdate: true,
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).To(HaveOccurred())
			},
		},
		{
			name: "Adopt orphaned service",
			prepare: func(tc *v1alpha1.TidbCluster, indexers *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				}
				_ = indexers.svc.Add(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      controller.TiDBMemberName(tc.Name),
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							"helm.sh/chart": "tidb-cluster",
						},
					},
				})
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Labels).NotTo(HaveKey("helm.sh/chart"), "Expected labels to be overridden when adopting orphaned service")
				g.Expect(metav1.GetControllerOf(svc)).NotTo(BeNil(), "Expected adopted service is controlled by TidbCluster")
			},
		},
		{
			name: "Update service should not change ClusterIP",
			prepare: func(tc *v1alpha1.TidbCluster, indexers *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				}
				_ = indexers.svc.Add(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
						Name:            controller.TiDBMemberName(tc.Name),
						Namespace:       corev1.NamespaceDefault,
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "8.8.8.8",
					},
				})
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.ClusterIP).To(Equal("8.8.8.8"))
			},
		},
		{
			name: "Create service with portName",
			prepare: func(tc *v1alpha1.TidbCluster, _ *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type:     corev1.ServiceTypeClusterIP,
						PortName: pointer.StringPtr("mysql-tidb"),
					},
				}
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc).NotTo(BeNil())
				g.Expect(svc.Spec.Ports[0].Name).To(Equal("mysql-tidb"))
			},
		},
		{
			name: "Update service should remain healthcheck node port",
			prepare: func(tc *v1alpha1.TidbCluster, indexers *fakeIndexers) {
				tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
					},
					ExternalTrafficPolicy: &policyLocal,
				}
				_ = indexers.svc.Add(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
						Name:            controller.TiDBMemberName(tc.Name),
						Namespace:       corev1.NamespaceDefault,
					},
					Spec: corev1.ServiceSpec{
						Type:                  corev1.ServiceTypeLoadBalancer,
						ExternalTrafficPolicy: policyLocal,
						HealthCheckNodePort:   8888,
					},
				})
			},
			expectFn: func(g *GomegaWithT, err error, svc *corev1.Service) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.HealthCheckNodePort).To(Equal(int32(8888)))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

type fakeIndexers struct {
	pod    cache.Indexer
	tc     cache.Indexer
	svc    cache.Indexer
	eps    cache.Indexer
	secret cache.Indexer
	set    cache.Indexer
}

func newFakeTiDBMemberManager() (*tidbMemberManager, *controller.FakeStatefulSetControl, cache.Indexer, *controller.FakeTiDBControl, *fakeIndexers) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1().StatefulSets()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	secretInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Secrets()
	cmInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().ConfigMaps()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer, tcInformer)
	genericControl := controller.NewFakeGenericControl()
	tidbUpgrader := NewFakeTiDBUpgrader()
	tidbFailover := NewFakeTiDBFailover()
	tidbControl := controller.NewFakeTiDBControl()

	tmm := &tidbMemberManager{
		setControl,
		svcControl,
		tidbControl,
		controller.NewTypedControl(genericControl),
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		cmInformer.Lister(),
		secretInformer.Lister(),
		tidbUpgrader,
		true,
		tidbFailover,
		tidbStatefulSetIsUpgrading,
	}
	indexers := &fakeIndexers{
		pod:    podInformer.Informer().GetIndexer(),
		tc:     tcInformer.Informer().GetIndexer(),
		svc:    svcInformer.Informer().GetIndexer(),
		eps:    epsInformer.Informer().GetIndexer(),
		secret: secretInformer.Informer().GetIndexer(),
		set:    setInformer.Informer().GetIndexer(),
	}
	return tmm, setControl, podInformer.Informer().GetIndexer(), tidbControl, indexers
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
			TiDB: &v1alpha1.TiDBSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: v1alpha1.TiDBMemberType.String(),
				},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
				Replicas: 3,
			},
			PD:   &v1alpha1.PDSpec{},
			TiKV: &v1alpha1.TiKVSpec{},
		},
	}
}

func TestGetNewTiDBHeadlessServiceForTidbCluster(t *testing.T) {
	tests := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected corev1.Service
	}{
		{
			name: "basic",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tidb-peer",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
						"app.kubernetes.io/used-by":    "peer",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "None",
					Ports: []corev1.ServicePort{
						{
							Name:       "status",
							Port:       10080,
							TargetPort: intstr.FromInt(10080),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
					},
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewTiDBHeadlessServiceForTidbCluster(&tt.tc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewTiDBSetForTidbCluster(t *testing.T) {
	enable := true
	updateStrategy := v1alpha1.ConfigUpdateStrategyRollingUpdate
	tests := []struct {
		name    string
		tc      v1alpha1.TidbCluster
		cm      *corev1.ConfigMap
		wantErr bool
		testSts func(sts *apps.StatefulSet)
	}{
		{
			name: "tidb network is not host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "tidb network is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			testSts: testHostNetwork(t, true, v1.DNSClusterFirstWithHostNet),
		},
		{
			name: "tidb network is not host when pd is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "tidb network is not host when tikv is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
					TiDB: &v1alpha1.TiDBSpec{},
					PD:   &v1alpha1.PDSpec{},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "tidb should use the latest synced configmap",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							ConfigUpdateStrategy: &updateStrategy,
						},
						Config: v1alpha1.NewTiDBConfig(),
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tc-tidb-xxxxxxxx",
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				cmName := FindConfigMapVolume(&sts.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, controller.TiDBMemberName("tc"))
				})
				g := NewGomegaWithT(t)
				g.Expect(cmName).To(Equal("tc-tidb-xxxxxxxx"))
			},
		},
		{
			name: "tidb should respect resources config",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ResourceRequirements: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:              resource.MustParse("1"),
								corev1.ResourceMemory:           resource.MustParse("2Gi"),
								corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:              resource.MustParse("1"),
								corev1.ResourceMemory:           resource.MustParse("2Gi"),
								corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
							},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				nameToContainer := MapContainers(&sts.Spec.Template.Spec)
				g.Expect(nameToContainer[v1alpha1.TiDBMemberType.String()].Resources).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
				}))
			},
		},
		{
			name: "TiDB additional containers",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalContainers: []corev1.Container{customSideCarContainers[0]},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			testSts: testAdditionalContainers(t, []corev1.Container{customSideCarContainers[0]}),
		},
		{
			name: "TiDB additional volumes",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalVolumes: []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			testSts: testAdditionalVolumes(t, []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}),
		},
		// TODO add more tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts := getNewTiDBSetForTidbCluster(&tt.tc, tt.cm)
			tt.testSts(sts)
		})
	}
}

func TestTiDBInitContainers(t *testing.T) {
	privileged := true
	asRoot := false
	tests := []struct {
		name             string
		tc               v1alpha1.TidbCluster
		expectedInit     []corev1.Container
		expectedSecurity *corev1.PodSecurityContext
	}{
		{
			name: "no init container",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							PodSecurityContext: &corev1.PodSecurityContext{
								RunAsNonRoot: &asRoot,
								Sysctls: []corev1.Sysctl{
									{
										Name:  "net.core.somaxconn",
										Value: "32768",
									},
									{
										Name:  "net.ipv4.tcp_syncookies",
										Value: "0",
									},
									{
										Name:  "net.ipv4.tcp_keepalive_time",
										Value: "300",
									},
									{
										Name:  "net.ipv4.tcp_keepalive_intvl",
										Value: "75",
									},
								},
							},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit: nil,
			expectedSecurity: &corev1.PodSecurityContext{
				RunAsNonRoot: &asRoot,
				Sysctls: []corev1.Sysctl{
					{
						Name:  "net.core.somaxconn",
						Value: "32768",
					},
					{
						Name:  "net.ipv4.tcp_syncookies",
						Value: "0",
					},
					{
						Name:  "net.ipv4.tcp_keepalive_time",
						Value: "300",
					},
					{
						Name:  "net.ipv4.tcp_keepalive_intvl",
						Value: "75",
					},
				},
			},
		},
		{
			name: "sysctl with init container",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Annotations: map[string]string{
								"tidb.pingcap.com/sysctl-init": "true",
							},
							PodSecurityContext: &corev1.PodSecurityContext{
								RunAsNonRoot: &asRoot,
								Sysctls: []corev1.Sysctl{
									{
										Name:  "net.core.somaxconn",
										Value: "32768",
									},
									{
										Name:  "net.ipv4.tcp_syncookies",
										Value: "0",
									},
									{
										Name:  "net.ipv4.tcp_keepalive_time",
										Value: "300",
									},
									{
										Name:  "net.ipv4.tcp_keepalive_intvl",
										Value: "75",
									},
								},
							},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit: []corev1.Container{
				{
					Name:  "init",
					Image: "busybox:1.26.2",
					Command: []string{
						"sh",
						"-c",
						"sysctl -w net.core.somaxconn=32768 net.ipv4.tcp_syncookies=0 net.ipv4.tcp_keepalive_time=300 net.ipv4.tcp_keepalive_intvl=75",
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
				},
			},
			expectedSecurity: &corev1.PodSecurityContext{
				RunAsNonRoot: &asRoot,
				Sysctls:      []corev1.Sysctl{},
			},
		},
		{
			name: "sysctl with init container",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Annotations: map[string]string{
								"tidb.pingcap.com/sysctl-init": "true",
							},
							PodSecurityContext: &corev1.PodSecurityContext{
								RunAsNonRoot: &asRoot,
							},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit: nil,
			expectedSecurity: &corev1.PodSecurityContext{
				RunAsNonRoot: &asRoot,
			},
		},
		{
			name: "sysctl with init container",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Annotations: map[string]string{
								"tidb.pingcap.com/sysctl-init": "true",
							},
							PodSecurityContext: nil,
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit:     nil,
			expectedSecurity: nil,
		},
		{
			name: "sysctl without init container due to invalid annotation",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Annotations: map[string]string{
								"tidb.pingcap.com/sysctl-init": "false",
							},
							PodSecurityContext: &corev1.PodSecurityContext{
								RunAsNonRoot: &asRoot,
								Sysctls: []corev1.Sysctl{
									{
										Name:  "net.core.somaxconn",
										Value: "32768",
									},
									{
										Name:  "net.ipv4.tcp_syncookies",
										Value: "0",
									},
									{
										Name:  "net.ipv4.tcp_keepalive_time",
										Value: "300",
									},
									{
										Name:  "net.ipv4.tcp_keepalive_intvl",
										Value: "75",
									},
								},
							},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit: nil,
			expectedSecurity: &corev1.PodSecurityContext{
				RunAsNonRoot: &asRoot,
				Sysctls: []corev1.Sysctl{
					{
						Name:  "net.core.somaxconn",
						Value: "32768",
					},
					{
						Name:  "net.ipv4.tcp_syncookies",
						Value: "0",
					},
					{
						Name:  "net.ipv4.tcp_keepalive_time",
						Value: "300",
					},
					{
						Name:  "net.ipv4.tcp_keepalive_intvl",
						Value: "75",
					},
				},
			},
		},
		{
			name: "no init container no securityContext",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit:     nil,
			expectedSecurity: nil,
		},
		{
			name: "Specitfy init container resourceRequirements",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ResourceRequirements: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("150m"),
								corev1.ResourceMemory: resource.MustParse("200Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("150m"),
								corev1.ResourceMemory: resource.MustParse("200Mi"),
							},
						},
						ComponentSpec: v1alpha1.ComponentSpec{
							Annotations: map[string]string{
								"tidb.pingcap.com/sysctl-init": "true",
							},
							PodSecurityContext: &corev1.PodSecurityContext{
								RunAsNonRoot: &asRoot,
								Sysctls: []corev1.Sysctl{
									{
										Name:  "net.core.somaxconn",
										Value: "32768",
									},
								},
							},
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expectedInit: []corev1.Container{
				{
					Name:  "init",
					Image: "busybox:1.26.2",
					Command: []string{
						"sh",
						"-c",
						"sysctl -w net.core.somaxconn=32768",
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("150m"),
							corev1.ResourceMemory: resource.MustParse("200Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("150m"),
							corev1.ResourceMemory: resource.MustParse("200Mi"),
						},
					},
				},
			},
			expectedSecurity: &corev1.PodSecurityContext{
				RunAsNonRoot: &asRoot,
				Sysctls:      []corev1.Sysctl{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts := getNewTiDBSetForTidbCluster(&tt.tc, nil)
			if diff := cmp.Diff(tt.expectedInit, sts.Spec.Template.Spec.InitContainers); diff != "" {
				t.Errorf("unexpected InitContainers in Statefulset (-want, +got): %s", diff)
			}
			if tt.expectedSecurity == nil {
				if sts.Spec.Template.Spec.SecurityContext != nil {
					t.Errorf("unexpected SecurityContext in Statefulset (want nil, got %#v)", *sts.Spec.Template.Spec.SecurityContext)
				}
			} else if sts.Spec.Template.Spec.SecurityContext == nil {
				t.Errorf("unexpected SecurityContext in Statefulset (want %#v, got nil)", *tt.expectedSecurity)
			} else if diff := cmp.Diff(*(tt.expectedSecurity), *(sts.Spec.Template.Spec.SecurityContext)); diff != "" {
				t.Errorf("unexpected SecurityContext in Statefulset (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewTiDBService(t *testing.T) {
	g := NewGomegaWithT(t)
	trafficPolicy := corev1.ServiceExternalTrafficPolicyTypeLocal
	loadBalancerSourceRanges := []string{
		"10.0.0.0/8",
		"130.211.204.1/32",
	}
	testCases := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected *corev1.Service
	}{
		{
			name: "TiDB service spec is nil",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: nil,
		},
		{
			name: "TiDB service with no status exposed",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Service: &v1alpha1.TiDBServiceSpec{
							ExposeStatus: pointer.BoolPtr(false),
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tidb",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "mysql-client",
							Port:       4000,
							TargetPort: intstr.FromInt(4000),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
					},
				},
			},
		},
		{
			name: "TiDB service with status exposed",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Service: &v1alpha1.TiDBServiceSpec{
							ExposeStatus: pointer.BoolPtr(true),
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tidb",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "mysql-client",
							Port:       4000,
							TargetPort: intstr.FromInt(4000),
							Protocol:   corev1.ProtocolTCP,
						},
						{
							Name:       "status",
							Port:       10080,
							TargetPort: intstr.FromInt(10080),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
					},
				},
			},
		},
		{
			name: "TiDB service in typical public cloud",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Service: &v1alpha1.TiDBServiceSpec{
							ServiceSpec: v1alpha1.ServiceSpec{
								Type: corev1.ServiceTypeLoadBalancer,
								Annotations: map[string]string{
									"lb-type": "testlb",
								},
								LoadBalancerSourceRanges: loadBalancerSourceRanges,
							},
							ExternalTrafficPolicy: &trafficPolicy,
							ExposeStatus:          pointer.BoolPtr(true),
						},
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tidb",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
						"app.kubernetes.io/used-by":    "end-user",
					},
					Annotations: map[string]string{
						"lb-type": "testlb",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					Type:                  corev1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
					LoadBalancerSourceRanges: []string{
						"10.0.0.0/8",
						"130.211.204.1/32",
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "mysql-client",
							Port:       4000,
							TargetPort: intstr.FromInt(4000),
							Protocol:   corev1.ProtocolTCP,
						},
						{
							Name:       "status",
							Port:       10080,
							TargetPort: intstr.FromInt(10080),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewTiDBServiceOrNil(&tt.tc)
			if tt.expected == nil {
				g.Expect(svc).To(BeNil())
				return
			}
			if diff := cmp.Diff(*tt.expected, *svc); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetTiDBConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	testCases := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected *corev1.ConfigMap
	}{
		{
			name: "TiDB config is nil",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: nil,
		},
		{
			name: "basic",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							ConfigUpdateStrategy: &updateStrategy,
						},
						Config: mustConfig(&v1alpha1.TiDBConfig{
							Lease: pointer.StringPtr("45s"),
						}),
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tidb",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Data: map[string]string{
					"startup-script": "",
					"config-file": `lease = "45s"
`,
				},
			},
		},
		{
			name: "TiDB config with tls enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							ConfigUpdateStrategy: &updateStrategy,
						},
						TLSClient: &v1alpha1.TiDBTLSClient{Enabled: true},
						Config:    v1alpha1.NewTiDBConfig(),
					},
					PD:   &v1alpha1.PDSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tidb",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tidb",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Data: map[string]string{
					"startup-script": "",
					"config-file": `[security]
  cluster-ssl-ca = "/var/lib/tidb-tls/ca.crt"
  cluster-ssl-cert = "/var/lib/tidb-tls/tls.crt"
  cluster-ssl-key = "/var/lib/tidb-tls/tls.key"
  ssl-ca = "/var/lib/tidb-server-tls/ca.crt"
  ssl-cert = "/var/lib/tidb-server-tls/tls.crt"
  ssl-key = "/var/lib/tidb-server-tls/tls.key"
`,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getTiDBConfigMap(&tt.tc)
			g.Expect(err).To(Succeed())
			if tt.expected == nil {
				g.Expect(cm).To(BeNil())
				return
			}
			// startup-script is better to be tested in e2e
			cm.Data["startup-script"] = ""
			if diff := cmp.Diff(*tt.expected, *cm); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestTiDBMemberManagerScaleToZeroReplica(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		setStatus                func(cluster *v1alpha1.TidbCluster)
		errWhenUpdateStatefulSet bool
		err                      bool
		expectStatefulSetFn      func(*GomegaWithT, *apps.StatefulSet, *v1alpha1.TidbCluster, error)
	}

	syncTiDBCluster := func(tmm *tidbMemberManager, tc *v1alpha1.TidbCluster, test *testcase) {
		err := tmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTiDB()
		tc.Spec.TiDB.MaxFailoverCount = pointer.Int32Ptr(3)
		tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(3)
		tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
			"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}

		if test.setStatus != nil {
			test.setStatus(tc)
		}

		ns := tc.GetNamespace()
		tcName := tc.GetName()

		tmm, fakeSetControl, _, _, _ := newFakeTiDBMemberManager()

		err := tmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(err).NotTo(HaveOccurred())
		_, err = tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		// scale to 0
		tc1.Spec.TiDB.Replicas = 0

		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		syncTiDBCluster(tmm, tc1, test)
		syncTiDBCluster(tmm, tc1, test)

		if test.expectStatefulSetFn != nil {
			set, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
			test.expectStatefulSetFn(g, set, tc1, err)
		}

	}

	tests := []testcase{
		{
			name: "TiDB should clear failureMembers when scale to 0",
			setStatus: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"tidb-0": {CreatedAt: metav1.Now(), PodName: "tidb-0"},
					"tidb-1": {CreatedAt: metav1.Now(), PodName: "tidb-1"},
				}
			},
			errWhenUpdateStatefulSet: false,
			err:                      false,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(0)))
			},
		},
		{
			name: "TiDB should clear failureMembers when scale to 0 and tidb is upgrading",
			setStatus: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"tidb-0": {CreatedAt: metav1.Now(), PodName: "tidb-0"},
					"tidb-1": {CreatedAt: metav1.Now(), PodName: "tidb-1"},
				}
			},
			errWhenUpdateStatefulSet: false,
			err:                      false,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(0)))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiDBShouldRecover(t *testing.T) {
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tidb-0",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tidb-1",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	podsWithFailover := append(pods, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failover-tidb-2",
			Namespace: v1.NamespaceDefault,
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	})
	tests := []struct {
		name string
		tc   *v1alpha1.TidbCluster
		pods []*v1.Pod
		want bool
	}{
		{
			name: "should not recover if no failure members",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Status: v1alpha1.TidbClusterStatus{},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should not recover if a member is not healthy",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiDB: v1alpha1.TiDBStatus{
						Members: map[string]v1alpha1.TiDBMember{
							"failover-tidb-0": {
								Name:   "failover-tidb-0",
								Health: false,
							},
							"failover-tidb-1": {
								Name:   "failover-tidb-1",
								Health: true,
							},
						},
						FailureMembers: map[string]v1alpha1.TiDBFailureMember{
							"failover-tidb-0": {
								PodName: "failover-tidb-0",
							},
						},
					},
				},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should recover if all members are ready and healthy",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiDB: v1alpha1.TiDBStatus{
						Members: map[string]v1alpha1.TiDBMember{
							"failover-tidb-0": {
								Name:   "failover-tidb-0",
								Health: true,
							},
							"failover-tidb-1": {
								Name:   "failover-tidb-1",
								Health: true,
							},
						},
						FailureMembers: map[string]v1alpha1.TiDBFailureMember{
							"failover-tidb-0": {
								PodName: "failover-tidb-0",
							},
						},
					},
				},
			},
			pods: pods,
			want: true,
		},
		{
			name: "should recover if all members are ready and healthy (ignore auto-created failover pods)",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiDB: v1alpha1.TiDBStatus{
						Members: map[string]v1alpha1.TiDBMember{
							"failover-tidb-0": {
								Name:   "failover-tidb-0",
								Health: true,
							},
							"failover-tidb-1": {
								Name:   "failover-tidb-1",
								Health: true,
							},
							"failover-tidb-2": {
								Name:   "failover-tidb-1",
								Health: false,
							},
						},
						FailureMembers: map[string]v1alpha1.TiDBFailureMember{
							"failover-tidb-0": {
								PodName: "failover-tidb-0",
							},
						},
					},
				},
			},
			pods: podsWithFailover,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := kubefake.NewSimpleClientset()
			for _, pod := range tt.pods {
				client.CoreV1().Pods(pod.Namespace).Create(pod)
			}
			kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, 0)
			podLister := kubeInformerFactory.Core().V1().Pods().Lister()
			kubeInformerFactory.Start(ctx.Done())
			kubeInformerFactory.WaitForCacheSync(ctx.Done())
			tidbMemberManager := &tidbMemberManager{podLister: podLister}
			got := tidbMemberManager.shouldRecover(tt.tc)
			if got != tt.want {
				t.Fatalf("wants %v, got %v", tt.want, got)
			}
		})
	}
}

func mustConfig(x interface{}) *v1alpha1.TiDBConfigWraper {
	data, err := MarshalTOML(x)
	if err != nil {
		panic(err)
	}

	c := v1alpha1.NewTiDBConfig()
	c.UnmarshalTOML(data)

	return c
}
