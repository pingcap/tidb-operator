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

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
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
			name: "tidbcluster's storage format is wrong",
			prepare: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Requests.Storage = "100xxxxi"
			},
			errWhenCreateStatefulSet:   false,
			errWhenCreatePDService:     false,
			errWhenCreatePDPeerService: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "cant' get storage size: 100xxxxi for TidbCluster: default/test")).To(BeTrue())
			},
			pdSvcCreated:     true,
			pdPeerSvcCreated: true,
			setCreated:       false,
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
			name: "tidbcluster's storage format is wrong",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Requests.Storage = "100xxxxi"
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                        true,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn:        nil,
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
					Labels:      label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
	pdControl := pdapi.NewFakePDControl()
	pdScaler := NewFakePDScaler()
	autoFailover := true
	pdFailover := NewFakePDFailover()
	pdUpgrader := NewFakePDUpgrader()

	return &pdMemberManager{
		pdControl,
		setControl,
		svcControl,
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		epsInformer.Lister(),
		podControl,
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
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "pd-test-image",
					Requests: &v1alpha1.ResourceRequirement{
						CPU:     "1",
						Memory:  "2Gi",
						Storage: "100Gi",
					},
				},
				Replicas:         3,
				StorageClassName: "my-storage-class",
			},
			TiKV: v1alpha1.TiKVSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "tikv-test-image",
					Requests: &v1alpha1.ResourceRequirement{
						CPU:     "1",
						Memory:  "2Gi",
						Storage: "100Gi",
					},
				},
				Replicas:         3,
				StorageClassName: "my-storage-class",
			},
		},
	}
}

func expectErrIsNotFound(g *GomegaWithT, err error) {
	g.Expect(err).NotTo(BeNil())
	g.Expect(errors.IsNotFound(err)).To(Equal(true))
}
