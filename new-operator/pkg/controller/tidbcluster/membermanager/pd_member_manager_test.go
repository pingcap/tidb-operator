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

package membermanager

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1.SchemeGroupVersion.WithKind("TidbCluster")

func TestPDMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                       string
		prepare                    func(cluster *v1.TidbCluster)
		errWhenCreateStatefulSet   bool
		errWhenCreatePDService     bool
		errWhenCreatePDPeerService bool
		err                        bool
		pdSvcCreated               bool
		pdPeerSvcCreated           bool
		setCreated                 bool
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbCluster()
		ns := tc.Namespace
		tcName := tc.Name
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		pmm, fakeSetControl, fakeSvcControl := newFakePDMemberManager()

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
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc1, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		if test.pdSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		svc2, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
		if test.pdPeerSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc2).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
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
			err:              false,
			pdSvcCreated:     true,
			pdPeerSvcCreated: true,
			setCreated:       true,
		},
		{
			name: "tidbcluster's storage format is wrong",
			prepare: func(tc *v1.TidbCluster) {
				tc.Spec.PD.Requests.Storage = "100xxxxi"
			},
			errWhenCreateStatefulSet:   false,
			errWhenCreatePDService:     false,
			errWhenCreatePDPeerService: false,
			err:              true,
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
			err:              true,
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
			err:              true,
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
			err:              true,
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
		modify                     func(cluster *v1.TidbCluster)
		errWhenUpdateStatefulSet   bool
		errWhenUpdatePDService     bool
		errWhenUpdatePDPeerService bool
		err                        bool
		expectPDServieFn           func(*GomegaWithT, *corev1.Service, error)
		expectPDPeerServieFn       func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn        func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbCluster()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, fakeSetControl, fakeSvcControl := newFakePDMemberManager()

		err := pmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
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

		if test.expectPDServieFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
			test.expectPDServieFn(g, svc, err)
		}
		if test.expectPDPeerServieFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
			test.expectPDPeerServieFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
				tc.Spec.Services = []v1.Service{
					{Name: "pd", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err: false,
			expectPDServieFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectPDPeerServieFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
		},
		{
			name: "tidbcluster's storage format is wrong",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.PD.Requests.Storage = "100xxxxi"
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                  true,
			expectPDServieFn:     nil,
			expectPDPeerServieFn: nil,
			expectStatefulSetFn:  nil,
		},
		{
			name: "error when update pd service",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Services = []v1.Service{
					{Name: "pd", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     true,
			errWhenUpdatePDPeerService: false,
			err:                  true,
			expectPDServieFn:     nil,
			expectPDPeerServieFn: nil,
			expectStatefulSetFn:  nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			errWhenUpdateStatefulSet:   true,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                  true,
			expectPDServieFn:     nil,
			expectPDPeerServieFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(3))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePDMemberManager() (*pdMemberManager, *controller.FakeStatefulSetControl, *controller.FakeServiceControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)

	return &pdMemberManager{
		setControl,
		svcControl,
		setInformer.Lister(),
		svcInformer.Lister(),
	}, setControl, svcControl
}

func newTidbCluster() *v1.TidbCluster {
	return &v1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1.TidbClusterSpec{
			PD: v1.PDSpec{
				ContainerSpec: v1.ContainerSpec{
					Image: "image",
					Requests: &v1.ResourceRequirement{
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
	g.Expect(err).NotTo(Equal(nil))
	g.Expect(errors.IsNotFound(err)).To(Equal(true))
}
