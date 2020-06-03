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

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
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
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestPDMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                       string
		prepare                    func(cluster *v1alpha1.TidbCluster)
		errWhenCreateStatefulSet   bool
		errWhenCreatePDService     bool
		errWhenCreatePDPeerService bool
		errExpectFn                func(*GomegaWithT, error)
		pdSvcCreated               bool
		pdPeerSvcCreated           bool
		setCreated                 bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		pmm, fakeSetControl, fakeSvcControl, _, _, _, _ := newFakePDMemberManager()

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreatePDService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreatePDPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}

		err := pmm.Sync(tc)
		test.errExpectFn(g, err)
		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc1, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		eps1, eperr := pmm.epsLister.Endpoints(ns).Get(controller.PDMemberName(tcName))
		if test.pdSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc1).NotTo(Equal(nil))
			g.Expect(eperr).NotTo(HaveOccurred())
			g.Expect(eps1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
			expectErrIsNotFound(g, eperr)
		}

		svc2, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
		eps2, eperr := pmm.epsLister.Endpoints(ns).Get(controller.PDPeerMemberName(tcName))
		if test.pdPeerSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc2).NotTo(Equal(nil))
			g.Expect(eperr).NotTo(HaveOccurred())
			g.Expect(eps2).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
			expectErrIsNotFound(g, eperr)
		}

		tc1, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
		if test.setCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}
	}

	tests := []testcase{
		{
			name:                       "normal",
			prepare:                    nil,
			errWhenCreateStatefulSet:   false,
			errWhenCreatePDService:     false,
			errWhenCreatePDPeerService: false,
			errExpectFn:                errExpectRequeue,
			pdSvcCreated:               true,
			pdPeerSvcCreated:           true,
			setCreated:                 true,
		},
		{
			name:                       "error when create statefulset",
			prepare:                    nil,
			errWhenCreateStatefulSet:   true,
			errWhenCreatePDService:     false,
			errWhenCreatePDPeerService: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed")).To(BeTrue())
			},
			pdSvcCreated:     true,
			pdPeerSvcCreated: true,
			setCreated:       false,
		},
		{
			name:                       "error when create pd service",
			prepare:                    nil,
			errWhenCreateStatefulSet:   false,
			errWhenCreatePDService:     true,
			errWhenCreatePDPeerService: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed")).To(BeTrue())
			},
			pdSvcCreated:     false,
			pdPeerSvcCreated: false,
			setCreated:       false,
		},
		{
			name:                       "error when create pd peer service",
			prepare:                    nil,
			errWhenCreateStatefulSet:   false,
			errWhenCreatePDService:     false,
			errWhenCreatePDPeerService: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed")).To(BeTrue())
			},
			pdSvcCreated:     true,
			pdPeerSvcCreated: false,
			setCreated:       false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                       string
		modify                     func(cluster *v1alpha1.TidbCluster)
		pdHealth                   *pdapi.HealthInfo
		errWhenUpdateStatefulSet   bool
		errWhenUpdatePDService     bool
		errWhenUpdatePDPeerService bool
		errWhenGetCluster          bool
		errWhenGetPDHealth         bool
		statusChange               func(*apps.StatefulSet)
		err                        bool
		expectPDServiceFn          func(*GomegaWithT, *corev1.Service, error)
		expectPDPeerServiceFn      func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn        func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn        func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, fakeSetControl, fakeSvcControl, fakePDControl, _, _, _ := newFakePDMemberManager()
		pdClient := controller.NewFakePDClient(fakePDControl, tc)
		if test.errWhenGetPDHealth {
			pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get health of pd cluster")
			})
		} else {
			pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.pdHealth, nil
			})
		}

		if test.errWhenGetCluster {
			pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get cluster info")
			})
		} else {
			pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
				return &metapb.Cluster{Id: uint64(1)}, nil
			})
		}

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "pd-1"
				set.Status.UpdateRevision = "pd-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := pmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.epsLister.Endpoints(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		_, err = pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.epsLister.Endpoints(ns).Get(controller.PDPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		_, err = pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdatePDService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenUpdatePDPeerService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}
		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = pmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectPDServiceFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
			test.expectPDServiceFn(g, svc, err)
		}
		if test.expectPDPeerServiceFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
			test.expectPDPeerServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc1)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
				tc.Spec.Services = []v1alpha1.Service{
					{Name: "pd", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetPDHealth:         false,
			err:                        false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				// g.Expect(int(*set.Spec.Replicas)).To(Equal(4))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.ClusterID).To(Equal("1"))
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.NormalPhase))
				g.Expect(tc.Status.PD.StatefulSet.ObservedGeneration).To(Equal(int64(1)))
				g.Expect(len(tc.Status.PD.Members)).To(Equal(3))
				g.Expect(tc.Status.PD.Members["pd1"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["pd2"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["pd3"].Health).To(Equal(false))
			},
		},
		{
			name: "error when update pd service",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.Services = []v1alpha1.Service{
					{Name: "pd", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     true,
			errWhenUpdatePDPeerService: false,
			err:                        true,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn:        nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   true,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                        true,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "error when sync pd status",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetPDHealth:         true,
			err:                        false,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Synced).To(BeFalse())
				g.Expect(tc.Status.PD.Members).To(BeNil())
			},
		},
		{
			name: "error when sync cluster ID",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetCluster:          true,
			errWhenGetPDHealth:         false,
			err:                        false,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Synced).To(BeFalse())
				g.Expect(tc.Status.PD.Members).To(BeNil())
			},
		},
	}

	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestPDMemberManagerPdStatefulSetIsUpgrading(t *testing.T) {
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
		pmm, _, _, _, podIndexer, _, _ := newFakePDMemberManager()
		tc := newTidbClusterForPD()
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{
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
					Name:        ordinalPodName(v1alpha1.PDMemberType, tc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.New().Instance(tc.GetInstanceName()).PD().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			podIndexer.Add(pod)
		}
		b, err := pmm.pdStatefulSetIsUpgrading(set, tc)
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

func TestPDMemberManagerUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.TidbCluster)
		pdHealth            *pdapi.HealthInfo
		err                 bool
		statusChange        func(*apps.StatefulSet)
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, fakeSetControl, _, fakePDControl, _, _, _ := newFakePDMemberManager()
		pdClient := controller.NewFakePDClient(fakePDControl, tc)

		pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
			return test.pdHealth, nil
		})
		pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
			return &metapb.Cluster{Id: uint64(1)}, nil
		})

		fakeSetControl.SetStatusChange(test.statusChange)

		err := pmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		err = pmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc1)
		}
	}
	tests := []testcase{
		{
			name: "upgrade successful",
			modify: func(cluster *v1alpha1.TidbCluster) {
				cluster.Spec.PD.Image = "pd-test-image:v2"
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			err: false,
			statusChange: func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "pd-1"
				set.Status.UpdateRevision = "pd-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("pd-test-image:v2"))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(len(tc.Status.PD.Members)).To(Equal(3))
				g.Expect(tc.Status.PD.Members["pd1"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["pd2"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["pd3"].Health).To(Equal(false))
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestPDMemberManagerSyncPDSts(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.TidbCluster)
		pdHealth            *pdapi.HealthInfo
		err                 bool
		statusChange        func(*apps.StatefulSet)
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, fakeSetControl, _, fakePDControl, _, _, _ := newFakePDMemberManager()
		pdClient := controller.NewFakePDClient(fakePDControl, tc)

		pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
			return test.pdHealth, nil
		})
		pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
			return &metapb.Cluster{Id: uint64(1)}, nil
		})

		fakeSetControl.SetStatusChange(test.statusChange)

		err := pmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		test.modify(tc)
		pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
			return &metapb.Cluster{Id: uint64(1)}, fmt.Errorf("cannot get cluster")
		})
		err = pmm.syncPDStatefulSetForTidbCluster(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}
	}
	tests := []testcase{
		{
			name: "force upgrade",
			modify: func(cluster *v1alpha1.TidbCluster) {
				cluster.Spec.PD.Image = "pd-test-image:v2"
				cluster.Spec.PD.Replicas = 1
				cluster.ObjectMeta.Annotations = make(map[string]string)
				cluster.ObjectMeta.Annotations["tidb.pingcap.com/force-upgrade"] = "true"
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: false},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: false},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			err: true,
			statusChange: func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "pd-1"
				set.Status.UpdateRevision = "pd-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("pd-test-image:v2"))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(1)))
				g.Expect(*set.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(0)))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
		{
			name: "non force upgrade",
			modify: func(cluster *v1alpha1.TidbCluster) {
				cluster.Spec.PD.Image = "pd-test-image:v2"
				cluster.Spec.PD.Replicas = 1
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: false},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: false},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			err: true,
			statusChange: func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "pd-1"
				set.Status.UpdateRevision = "pd-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("pd-test-image"))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(3)))
				g.Expect(*set.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func newFakePDMemberManager() (*pdMemberManager, *controller.FakeStatefulSetControl, *controller.FakeServiceControl, *pdapi.FakePDControl, cache.Indexer, cache.Indexer, *controller.FakePodControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	pvcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumeClaims()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer, tcInformer)
	podControl := controller.NewFakePodControl(podInformer)
	pdControl := pdapi.NewFakePDControl(kubeCli)
	pdScaler := NewFakePDScaler()
	autoFailover := true
	pdFailover := NewFakePDFailover()
	pdUpgrader := NewFakePDUpgrader()
	genericControll := controller.NewFakeGenericControl()

	return &pdMemberManager{
		pdControl,
		setControl,
		svcControl,
		podControl,
		controller.NewTypedControl(genericControll),
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		epsInformer.Lister(),
		pvcInformer.Lister(),
		pdScaler,
		pdUpgrader,
		autoFailover,
		pdFailover,
	}, setControl, svcControl, pdControl, podInformer.Informer().GetIndexer(), pvcInformer.Informer().GetIndexer(), podControl
}

func newTidbClusterForPD() *v1alpha1.TidbCluster {
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
			PD: v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("1"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiKV: v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("1"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				},
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
		},
	}
}

func expectErrIsNotFound(g *GomegaWithT, err error) {
	g.Expect(err).NotTo(BeNil())
	g.Expect(errors.IsNotFound(err)).To(Equal(true))
}

func TestGetNewPDHeadlessServiceForTidbCluster(t *testing.T) {
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
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd-peer",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
							Name:       "peer",
							Port:       2380,
							TargetPort: intstr.FromInt(2380),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
					},
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewPDHeadlessServiceForTidbCluster(&tt.tc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func testHostNetwork(t *testing.T, hostNetwork bool, dnsPolicy v1.DNSPolicy) func(sts *apps.StatefulSet) {
	return func(sts *apps.StatefulSet) {
		if hostNetwork != sts.Spec.Template.Spec.HostNetwork {
			t.Errorf("unexpected hostNetwork %v, want %v", sts.Spec.Template.Spec.HostNetwork, hostNetwork)
		}
		if dnsPolicy != sts.Spec.Template.Spec.DNSPolicy {
			t.Errorf("unexpected dnsPolicy %v, want %v", sts.Spec.Template.Spec.DNSPolicy, dnsPolicy)
		}
	}
}

func testAnnotations(t *testing.T, annotations map[string]string) func(sts *apps.StatefulSet) {
	return func(sts *apps.StatefulSet) {
		if diff := cmp.Diff(annotations, sts.Annotations); diff != "" {
			t.Errorf("unexpected annotations (-want, +got): %s", diff)
		}
	}
}

func testPDContainerEnv(t *testing.T, env []corev1.EnvVar) func(sts *apps.StatefulSet) {
	return func(sts *apps.StatefulSet) {
		got := []corev1.EnvVar{}
		for _, c := range sts.Spec.Template.Spec.Containers {
			if c.Name == v1alpha1.PDMemberType.String() {
				got = c.Env
			}
		}
		if diff := cmp.Diff(env, got); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
	}
}

func TestGetNewPDSetForTidbCluster(t *testing.T) {
	enable := true
	tests := []struct {
		name    string
		tc      v1alpha1.TidbCluster
		wantErr bool
		testSts func(sts *apps.StatefulSet)
	}{
		{
			name: "pd network is not host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "pd network is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
				},
			},
			testSts: testHostNetwork(t, true, v1.DNSClusterFirstWithHostNet),
		},
		{
			name: "pd network is not host when tidb is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "pd network is not host when tikv is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "PD should respect resources config",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ResourceRequirements: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:              resource.MustParse("1"),
								corev1.ResourceMemory:           resource.MustParse("2Gi"),
								corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
								corev1.ResourceStorage:          resource.MustParse("100Gi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:              resource.MustParse("1"),
								corev1.ResourceMemory:           resource.MustParse("2Gi"),
								corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.VolumeClaimTemplates[0].Spec.Resources).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				}))
				nameToContainer := MapContainers(&sts.Spec.Template.Spec)
				pdContainer := nameToContainer[v1alpha1.PDMemberType.String()]
				g.Expect(pdContainer.Resources).To(Equal(corev1.ResourceRequirements{
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
			name: "set custom env",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Env: []corev1.EnvVar{
								{
									Name: "DASHBOARD_SESSION_SECRET",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "dashboard-session-secret",
											},
											Key: "encryption_key",
										},
									},
								},
								{
									Name:  "TZ",
									Value: "ignored",
								},
							},
						},
					},
				},
			},
			testSts: testPDContainerEnv(t, []corev1.EnvVar{
				{
					Name: "NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
				{
					Name:  "PEER_SERVICE_NAME",
					Value: "tc-pd-peer",
				},
				{
					Name:  "SERVICE_NAME",
					Value: "tc-pd",
				},
				{
					Name:  "SET_NAME",
					Value: "tc-pd",
				},
				{
					Name: "TZ",
				},
				{
					Name: "DASHBOARD_SESSION_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "dashboard-session-secret",
							},
							Key: "encryption_key",
						},
					},
				},
			}),
		},
		{
			name: "tidb version v3.1.0, tidb client tls is enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v3",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v3.1.0",
						},
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(hasTLSVol(sts)).To(BeFalse())
				g.Expect(hasTLSVolMount(sts)).To(BeFalse())
			},
		},
		{
			name: "tidb version v4.0.0-rc.1, tidb client tls is enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v4",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v4.0.0-rc.1",
						},
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(hasTLSVol(sts)).To(BeTrue())
				g.Expect(hasTLSVolMount(sts)).To(BeTrue())
			},
		},
		{
			name: "tidb version nightly, tidb client tls is enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-nightly",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:nightly",
						},
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(hasTLSVol(sts)).To(BeTrue())
				g.Expect(hasTLSVolMount(sts)).To(BeTrue())
			},
		},
		{
			name: "tidbcluster with failureMember",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-nightly",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:nightly",
						},
						Replicas: 3,
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					PD: v1alpha1.PDStatus{
						FailureMembers: map[string]v1alpha1.PDFailureMember{
							"test": {
								MemberDeleted: false,
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(*sts.Spec.Replicas).To(Equal(int32(4)))
			},
		},
		{
			name: "tidbcluster with failureMember Deleted",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-nightly",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:nightly",
						},
						Replicas: 3,
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					PD: v1alpha1.PDStatus{
						FailureMembers: map[string]v1alpha1.PDFailureMember{
							"test": {
								MemberDeleted: true,
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(*sts.Spec.Replicas).To(Equal(int32(3)))
			},
		},
		// TODO add more tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, err := getNewPDSetForTidbCluster(&tt.tc, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error %v, wantErr %v", err, tt.wantErr)
			}
			tt.testSts(sts)
		})
	}
}

func TestGetPDConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	testCases := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected *corev1.ConfigMap
	}{
		{
			name: "PD config is nil",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
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
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							ConfigUpdateStrategy: &updateStrategy,
						},
						Config: &v1alpha1.PDConfig{
							Schedule: &v1alpha1.PDScheduleConfig{
								MaxStoreDownTime:         pointer.StringPtr("5m"),
								DisableRemoveDownReplica: pointer.BoolPtr(true),
							},
							Replication: &v1alpha1.PDReplicationConfig{
								MaxReplicas:    func() *uint64 { i := uint64(5); return &i }(),
								LocationLabels: []string{"node", "rack"},
							},
						},
					},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
					"config-file": `[schedule]
  max-store-down-time = "5m"
  disable-remove-down-replica = true

[replication]
  max-replicas = 5
  location-labels = ["node", "rack"]
`,
				},
			},
		},
		{
			name: "tidb version v3.1.0, tidb client tls is enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v3",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v3.1.0",
						},
						Config: &v1alpha1.PDConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v3-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "tls-v3",
						"app.kubernetes.io/component":  "pd",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "tls-v3",
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
					"config-file":    "",
				},
			},
		},
		{
			name: "tidb version v4.0.0-rc.1, tidb client tls is enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v4",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v4.0.0-rc.1",
						},
						Config: &v1alpha1.PDConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v4-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "tls-v4",
						"app.kubernetes.io/component":  "pd",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "tls-v4",
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
					"config-file": `[dashboard]
  tidb-cacert-path = "/var/lib/tidb-client-tls/ca.crt"
  tidb-cert-path = "/var/lib/tidb-client-tls/tls.crt"
  tidb-key-path = "/var/lib/tidb-client-tls/tls.key"
`,
				},
			},
		},
		{
			name: "tidb version nightly, tidb client tls is enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-nightly",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:nightly",
						},
						Config: &v1alpha1.PDConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-nightly-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "tls-nightly",
						"app.kubernetes.io/component":  "pd",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "tls-nightly",
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
					"config-file": `[dashboard]
  tidb-cacert-path = "/var/lib/tidb-client-tls/ca.crt"
  tidb-cert-path = "/var/lib/tidb-client-tls/tls.crt"
  tidb-key-path = "/var/lib/tidb-client-tls/tls.key"
`,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getPDConfigMap(&tt.tc)
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

func TestGetNewPdServiceForTidbCluster(t *testing.T) {
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
					Services: []v1alpha1.Service{
						{Name: "pd", Type: string(corev1.ServiceTypeClusterIP)},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       2379,
							TargetPort: intstr.FromInt(2379),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
					},
				},
			},
		},
		{
			name: "basic and  specify ClusterIP type,clusterIP",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "pd", Type: string(corev1.ServiceTypeClusterIP)},
					},
					PD: v1alpha1.PDSpec{
						Service: &v1alpha1.ServiceSpec{ClusterIP: pointer.StringPtr("172.20.10.1")},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       2379,
							TargetPort: intstr.FromInt(2379),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
					},
				},
			},
		},
		{
			name: "basic and specify LoadBalancerIP type , LoadBalancerType",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "pd", Type: string(corev1.ServiceTypeLoadBalancer)},
					},
					PD: v1alpha1.PDSpec{
						Service: &v1alpha1.ServiceSpec{LoadBalancerIP: pointer.StringPtr("172.20.10.1")},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
					LoadBalancerIP: "172.20.10.1",
					Type:           corev1.ServiceTypeLoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       2379,
							TargetPort: intstr.FromInt(2379),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
					},
				},
			},
		},
		{
			name: "basic and specify pd service overwrite",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "pd", Type: string(corev1.ServiceTypeLoadBalancer)},
					},
					PD: v1alpha1.PDSpec{
						Service: &v1alpha1.ServiceSpec{Type: corev1.ServiceTypeClusterIP,
							ClusterIP: pointer.StringPtr("172.20.10.1")},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       2379,
							TargetPort: intstr.FromInt(2379),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
					},
				},
			},
		},
		{
			name: "basic and specify pd service portname",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "pd", Type: string(corev1.ServiceTypeLoadBalancer)},
					},
					PD: v1alpha1.PDSpec{
						Service: &v1alpha1.ServiceSpec{Type: corev1.ServiceTypeClusterIP,
							ClusterIP: pointer.StringPtr("172.20.10.1"),
							PortName:  pointer.StringPtr("http-pd"),
						},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pd",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
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
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "http-pd",
							Port:       2379,
							TargetPort: intstr.FromInt(2379),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pd",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pmm, _, _, _, _, _, _ := newFakePDMemberManager()
			svc := pmm.getNewPDServiceForTidbCluster(&tt.tc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func TestPDMemberManagerSyncPDStsWhenPdNotJoinCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer)
		pdHealth            *pdapi.HealthInfo
		tcStatusChange      func(cluster *v1alpha1.TidbCluster)
		err                 bool
		expectTidbClusterFn func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, _, _, fakePDControl, podIndexer, pvcIndexer, _ := newFakePDMemberManager()
		pdClient := controller.NewFakePDClient(fakePDControl, tc)

		pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
			return test.pdHealth, nil
		})
		pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
			return &metapb.Cluster{Id: uint64(1)}, nil
		})

		err := pmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())
		_, err = pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		if test.tcStatusChange != nil {
			test.tcStatusChange(tc)
		}
		test.modify(tc, podIndexer, pvcIndexer)
		err = pmm.syncPDStatefulSetForTidbCluster(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}
	}
	tests := []testcase{
		{
			name: "add pd unjoin cluster member info ",
			modify: func(cluster *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				for ordinal := 0; ordinal < 3; ordinal++ {
					pod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPodName(v1alpha1.PDMemberType, cluster.GetName(), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.New().Instance(cluster.GetInstanceName()).PD().Labels(),
						},
					}
					podIndexer.Add(pod)
				}
				for ordinal := 0; ordinal < 3; ordinal++ {
					pvc := &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(cluster.GetName()), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.New().Instance(cluster.GetInstanceName()).PD().Labels(),
						},
					}
					pvcIndexer.Add(pvc)
				}

			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "test-pd-0", MemberID: uint64(1), ClientUrls: []string{"http://test-pd-0:2379"}, Health: false},
				{Name: "test-pd-1", MemberID: uint64(2), ClientUrls: []string{"http://test-pd-1:2379"}, Health: false},
			}},
			err: false,
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.UnjoinedMembers["test-pd-2"]).NotTo(BeNil())
			},
		},
		{
			name: "clear unjoin cluster member info when the member join the cluster ",
			tcStatusChange: func(cluster *v1alpha1.TidbCluster) {
				cluster.Status.PD.UnjoinedMembers = map[string]v1alpha1.UnjoinedMember{
					"test-pd-0": {
						PodName:   "test-pd-0",
						CreatedAt: metav1.Now(),
					},
				}
			},
			modify: func(cluster *v1alpha1.TidbCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				for ordinal := 0; ordinal < 3; ordinal++ {
					pod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPodName(v1alpha1.PDMemberType, cluster.GetName(), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.New().Instance(cluster.GetInstanceName()).PD().Labels(),
						},
					}
					podIndexer.Add(pod)
				}
				for ordinal := 0; ordinal < 3; ordinal++ {
					pvc := &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(cluster.GetName()), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.New().Instance(cluster.GetInstanceName()).PD().Labels(),
						},
					}
					pvcIndexer.Add(pvc)
				}

			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "test-pd-0", MemberID: uint64(1), ClientUrls: []string{"http://test-pd-0:2379"}, Health: false},
				{Name: "test-pd-1", MemberID: uint64(2), ClientUrls: []string{"http://test-pd-1:2379"}, Health: false},
				{Name: "test-pd-2", MemberID: uint64(2), ClientUrls: []string{"http://test-pd-2:2379"}, Health: false},
			}},
			err: false,
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.UnjoinedMembers).To(BeEmpty())
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestPDShouldRecover(t *testing.T) {
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-pd-0",
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
				Name:      "failover-pd-1",
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
			Name:      "failover-pd-2",
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
					PD: v1alpha1.PDSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					PD: v1alpha1.PDStatus{
						Members: map[string]v1alpha1.PDMember{
							"failover-pd-0": {
								Name:   "failover-pd-0",
								Health: false,
							},
							"failover-pd-1": {
								Name:   "failover-pd-1",
								Health: true,
							},
						},
						FailureMembers: map[string]v1alpha1.PDFailureMember{
							"failover-pd-0": {
								PodName: "failover-pd-0",
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
					PD: v1alpha1.PDSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					PD: v1alpha1.PDStatus{
						Members: map[string]v1alpha1.PDMember{
							"failover-pd-0": {
								Name:   "failover-pd-0",
								Health: true,
							},
							"failover-pd-1": {
								Name:   "failover-pd-1",
								Health: true,
							},
						},
						FailureMembers: map[string]v1alpha1.PDFailureMember{
							"failover-pd-0": {
								PodName: "failover-pd-0",
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
					PD: v1alpha1.PDSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					PD: v1alpha1.PDStatus{
						Members: map[string]v1alpha1.PDMember{
							"failover-pd-0": {
								Name:   "failover-pd-0",
								Health: true,
							},
							"failover-pd-1": {
								Name:   "failover-pd-1",
								Health: true,
							},
							"failover-pd-2": {
								Name:   "failover-pd-1",
								Health: false,
							},
						},
						FailureMembers: map[string]v1alpha1.PDFailureMember{
							"failover-pd-0": {
								PodName: "failover-pd-0",
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
			pdMemberManager := &pdMemberManager{podLister: podLister}
			got := pdMemberManager.shouldRecover(tt.tc)
			if got != tt.want {
				t.Fatalf("wants %v, got %v", tt.want, got)
			}
		})
	}
}

func hasTLSVol(sts *apps.StatefulSet) bool {
	for _, vol := range sts.Spec.Template.Spec.Volumes {
		if vol.Name == "tidb-client-tls" {
			return true
		}
	}
	return false
}

func hasTLSVolMount(sts *apps.StatefulSet) bool {
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == v1alpha1.PDMemberType.String() {
			for _, vm := range container.VolumeMounts {
				if vm.Name == "tidb-client-tls" {
					return true
				}
			}
		}
	}
	return false
}
