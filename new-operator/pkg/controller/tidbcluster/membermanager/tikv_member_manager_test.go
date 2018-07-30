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
	"github.com/pingcap/kvproto/pkg/metapb"
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
		name                         string
		prepare                      func(cluster *v1.TidbCluster)
		errWhenCreateStatefulSet     bool
		errWhenCreateTiKVService     bool
		errWhenCreateTiKVPeerService bool
		errWhenGetStores             bool
		err                          bool
		tikvSvcCreated               bool
		tikvPeerSvcCreated           bool
		setCreated                   bool
		pdStores                     *controller.StoresInfo
		tombstoneStores              *controller.StoresInfo
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		pmm, fakeSetControl, fakeSvcControl, pdClient := newFakeTiKVMemberManager(tc)

		if test.errWhenGetStores {
			pdClient.AddReaction(controller.GetStoresActionType, func(action *controller.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get stores from tikv cluster")
			})
		} else {
			pdClient.AddReaction(controller.GetStoresActionType, func(action *controller.Action) (interface{}, error) {
				return test.pdStores, nil
			})
			pdClient.AddReaction(controller.GetTombStoneStoresActionType, func(action *controller.Action) (interface{}, error) {
				return test.tombstoneStores, nil
			})
			pdClient.AddReaction(controller.SetStoreLabelsActionType, func(action *controller.Action) (interface{}, error) {
				return true, nil
			})
		}

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateTiKVService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateTiKVPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}

		err := pmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc1, err := pmm.svcLister.Services(ns).Get(controller.TiKVMemberName(tcName))
		if test.tikvSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		svc2, err := pmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
		if test.tikvPeerSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc2).NotTo(Equal(nil))
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
			name:    "normal",
			prepare: nil,

			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVService:     false,
			errWhenCreateTiKVPeerService: false,
			err:                false,
			tikvSvcCreated:     true,
			tikvPeerSvcCreated: true,
			setCreated:         true,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name: "tidbcluster's storage format is wrong",
			prepare: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVService:     false,
			errWhenCreateTiKVPeerService: false,
			err:                true,
			tikvSvcCreated:     true,
			tikvPeerSvcCreated: true,
			setCreated:         false,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name:                         "error when create statefulset",
			prepare:                      nil,
			errWhenCreateStatefulSet:     true,
			errWhenCreateTiKVService:     false,
			errWhenCreateTiKVPeerService: false,
			err:                true,
			tikvSvcCreated:     true,
			tikvPeerSvcCreated: true,
			setCreated:         false,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name:                         "error when create tikv service",
			prepare:                      nil,
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVService:     true,
			errWhenCreateTiKVPeerService: false,
			err:                true,
			tikvSvcCreated:     false,
			tikvPeerSvcCreated: false,
			setCreated:         false,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name:                         "error when create tikv peer service",
			prepare:                      nil,
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVService:     false,
			errWhenCreateTiKVPeerService: true,
			err:                true,
			tikvSvcCreated:     true,
			tikvPeerSvcCreated: false,
			setCreated:         false,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiKVMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                         string
		modify                       func(cluster *v1.TidbCluster)
		pdStores                     *controller.StoresInfo
		tombstoneStores              *controller.StoresInfo
		errWhenUpdateStatefulSet     bool
		errWhenUpdateTiKVService     bool
		errWhenUpdateTiKVPeerService bool
		errWhenGetStores             bool
		err                          bool
		expectPDServiceFn            func(*GomegaWithT, *corev1.Service, error)
		expectPDPeerServiceFn        func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn          func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn          func(*GomegaWithT, *v1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, fakeSetControl, fakeSvcControl, pdClient := newFakeTiKVMemberManager(tc)
		if test.errWhenGetStores {
			pdClient.AddReaction(controller.GetStoresActionType, func(action *controller.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get stores from pd cluster")
			})
		} else {
			pdClient.AddReaction(controller.GetStoresActionType, func(action *controller.Action) (interface{}, error) {
				return test.pdStores, nil
			})
			pdClient.AddReaction(controller.GetTombStoneStoresActionType, func(action *controller.Action) (interface{}, error) {
				return test.tombstoneStores, nil
			})
			pdClient.AddReaction(controller.SetStoreLabelsActionType, func(action *controller.Action) (interface{}, error) {
				return true, nil
			})
		}

		err := pmm.Sync(tc)

		_, err = pmm.svcLister.Services(ns).Get(controller.TiKVMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdateTiKVService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenUpdateTiKVPeerService {
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
			svc, err := pmm.svcLister.Services(ns).Get(controller.TiKVMemberName(tcName))
			test.expectPDServiceFn(g, svc, err)
		}
		if test.expectPDPeerServiceFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
			test.expectPDPeerServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := pmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc1)
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
			pdStores: &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{
				{Status: &controller.StoreStatus{},
					Store: &controller.MetaStore{
						StateName: "ok",
						Store:     &metapb.Store{Id: 1, Address: "here", State: metapb.StoreState_Up, Labels: nil},
					}},
				{Status: &controller.StoreStatus{}, Store: &controller.MetaStore{}},
			}},
			tombstoneStores: &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{
				{Status: &controller.StoreStatus{}, Store: &controller.MetaStore{}},
			}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVService:     false,
			errWhenUpdateTiKVPeerService: false,
			errWhenGetStores:             false,
			err:                          false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(*tc.Status.TiKV.StatefulSet.ObservedGeneration).To(Equal(int64(2)))
				expectedStores := map[string]v1.TiKVStores{}
				g.Expect(tc.Status.TiKV.Stores).To(Equal(expectedStores))
			},
		},
		{
			name: "tidbcluster's storage format is wrong",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVService:     false,
			errWhenUpdateTiKVPeerService: false,
			err:                   true,
			expectPDServiceFn:     nil,
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn:   nil,
		},
		{
			name: "error when update tikv service",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Services = []v1.Service{
					{Name: "tikv", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVService:     true,
			errWhenUpdateTiKVPeerService: false,
			err:                   true,
			expectPDServiceFn:     nil,
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn:   nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     true,
			errWhenUpdateTiKVService:     false,
			errWhenUpdateTiKVPeerService: false,
			err:                   true,
			expectPDServiceFn:     nil,
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(3))
			},
		},
		{
			name: "error when sync tikv status",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVService:     false,
			errWhenUpdateTiKVPeerService: false,
			errWhenGetStores:             true,
			err:                          true,
			expectPDServiceFn:            nil,
			expectPDPeerServiceFn:        nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(3))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.TiKV.Stores).To(BeNil())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiKVMemberManager(tc *v1.TidbCluster) (
	*TikvMemberManager, *controller.FakeStatefulSetControl,
	*controller.FakeServiceControl, *controller.FakePDClient) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	pdControl := controller.NewFakePDControl()
	pdClient := controller.NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)

	return NewTiKVMemberManager(
		kubeCli,
		pdControl,
		setControl,
		svcControl,
		setInformer.Lister(),
		svcInformer.Lister(),
	), setControl, svcControl, pdClient
}
