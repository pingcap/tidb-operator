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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestTiDBMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		prepare                  func(cluster *v1alpha1.TidbCluster)
		errWhenCreateStatefulSet bool
		errWhenCreateTiDBService bool
		err                      bool
		tidbSvcCreated           bool
		setCreated               bool
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForTiDB()
		ns := tc.GetNamespace()
		tcName := tc.GetName()
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		tmm, fakeSetControl, fakeSvcControl := newFakeTiDBMemberManager()

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateTiDBService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := tmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc, err := tmm.svcLister.Services(ns).Get(controller.TiDBMemberName(tcName))
		if test.tidbSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

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
			errWhenCreateTiDBService: false,
			err:            false,
			tidbSvcCreated: true,
			setCreated:     true,
		},
		{
			name:                     "error when create statefulset",
			prepare:                  nil,
			errWhenCreateStatefulSet: true,
			errWhenCreateTiDBService: false,
			err:            true,
			tidbSvcCreated: true,
			setCreated:     false,
		},
		{
			name:                     "error when create tidb service",
			prepare:                  nil,
			errWhenCreateStatefulSet: false,
			errWhenCreateTiDBService: true,
			err:            true,
			tidbSvcCreated: false,
			setCreated:     false,
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
		errWhenUpdateTiDBService bool
		statusChange             func(*apps.StatefulSet)
		err                      bool
		expectTiDBServieFn       func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn      func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForTiDB()
		ns := tc.GetNamespace()
		tcName := tc.GetName()

		tmm, fakeSetControl, fakeSvcControl := newFakeTiDBMemberManager()

		if test.statusChange != nil {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := tmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = tmm.svcLister.Services(ns).Get(controller.TiDBMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdateTiDBService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = tmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectTiDBServieFn != nil {
			svc, err := tmm.svcLister.Services(ns).Get(controller.TiDBMemberName(tcName))
			test.expectTiDBServieFn(g, svc, err)
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
				tc.Spec.Services = []v1alpha1.Service{
					{Name: "tidb", Type: string(corev1.ServiceTypeNodePort)},
				}
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			errWhenUpdateStatefulSet: false,
			errWhenUpdateTiDBService: false,
			err: false,
			expectTiDBServieFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
		},
		{
			name: "error when update tidb service",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.Services = []v1alpha1.Service{
					{Name: v1alpha1.TiDBMemberType.String(), Type: string(corev1.ServiceTypeNodePort)},
				}
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			errWhenUpdateStatefulSet: false,
			errWhenUpdateTiDBService: true,
			err: true,
			expectTiDBServieFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			},
			expectStatefulSetFn: nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 5
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			errWhenUpdateStatefulSet: true,
			errWhenUpdateTiDBService: false,
			err:                true,
			expectTiDBServieFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiDBMemberManager() (*tidbMemberManager, *controller.FakeStatefulSetControl, *controller.FakeServiceControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)

	return &tidbMemberManager{
		setControl,
		svcControl,
		setInformer.Lister(),
		svcInformer.Lister(),
	}, setControl, svcControl
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
