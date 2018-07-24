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
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestTiKVMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		prepare                  func(cluster *v1.TidbCluster)
		errWhenCreateStatefulSet bool
		errWhenCreateService     bool
		err                      bool
		svcCreated               bool
		setCreated               bool
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		pmm, fakeSetControl, fakeSvcControl := newFakeTiKVMemberManager()

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := pmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred(), test.name)
		} else {
			g.Expect(err).NotTo(HaveOccurred(), test.name)
		}

		g.Expect(tc.Spec).To(Equal(oldSpec), test.name)

		svc1, err := pmm.svcLister.Services(ns).Get(controller.TiKVMemberName(tcName))
		if test.svcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		tc1, err := pmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
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
			errWhenCreateService:     false,
			err:                      false,
			svcCreated:               true,
			setCreated:               true,
		},
		{
			name: "tidbcluster's storage format is wrong",
			prepare: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			errWhenCreateStatefulSet: false,
			errWhenCreateService:     false,
			err:                      true,
			svcCreated:               true,
			setCreated:               false,
		},
		{
			name:                     "error when create statefulset",
			prepare:                  nil,
			errWhenCreateStatefulSet: true,
			errWhenCreateService:     false,
			err:                      true,
			svcCreated:               true,
			setCreated:               false,
		},
		{
			name:                     "error when create tikv service",
			prepare:                  nil,
			errWhenCreateStatefulSet: false,
			errWhenCreateService:     true,
			err:                      true,
			svcCreated:               false,
			setCreated:               false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiKVMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		modify                   func(cluster *v1.TidbCluster)
		errWhenUpdateStatefulSet bool
		errWhenUpdateService     bool
		err                      bool
		expectServiceFn          func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn      func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, fakeSetControl, fakeSvcControl := newFakeTiKVMemberManager()

		err := pmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = pmm.svcLister.Services(ns).Get(controller.TiKVMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdateService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
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

		if test.expectServiceFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.TiKVMemberName(tcName))
			test.expectServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := pmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
				tc.Spec.Services = []v1.Service{
					{Name: "tikv", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateStatefulSet: false,
			errWhenUpdateService:     false,
			err:                      false,
			expectServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
		},
		{
			name: "tidbcluster's storage format is wrong",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			errWhenUpdateStatefulSet: false,
			errWhenUpdateService:     false,
			err:                      true,
			expectServiceFn:          nil,
			expectStatefulSetFn:      nil,
		},
		{
			name: "error when update tikv service",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Services = []v1.Service{
					{Name: "tikv", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateStatefulSet: false,
			errWhenUpdateService:     true,
			err:                      true,
			expectServiceFn:          nil,
			expectStatefulSetFn:      nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
			},
			errWhenUpdateStatefulSet: true,
			errWhenUpdateService:     false,
			err:                      true,
			expectServiceFn:          nil,
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

func newFakeTiKVMemberManager() (*TikvMemberManager, *controller.FakeStatefulSetControl, *controller.FakeServiceControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)

	return NewTiKVMemberManager(
		setControl,
		svcControl,
		setInformer.Lister(),
		svcInformer.Lister(),
	), setControl, svcControl
}
