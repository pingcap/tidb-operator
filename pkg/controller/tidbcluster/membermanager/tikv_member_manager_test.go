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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
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
		prepare                      func(cluster *v1alpha1.TidbCluster)
		errWhenCreateStatefulSet     bool
		errWhenCreateTiKVPeerService bool
		errWhenGetStores             bool
		err                          bool
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
		if test.errWhenCreateTiKVPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := pmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc, err := pmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
		if test.tikvPeerSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc).NotTo(Equal(nil))
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
			errWhenCreateTiKVPeerService: false,
			err:                false,
			tikvPeerSvcCreated: true,
			setCreated:         true,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name: "tidbcluster's storage format is wrong",
			prepare: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVPeerService: false,
			err:                true,
			tikvPeerSvcCreated: true,
			setCreated:         false,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name:                         "error when create statefulset",
			prepare:                      nil,
			errWhenCreateStatefulSet:     true,
			errWhenCreateTiKVPeerService: false,
			err:                true,
			tikvPeerSvcCreated: true,
			setCreated:         false,
			pdStores:           &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:    &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
		},
		{
			name:                         "error when create tikv peer service",
			prepare:                      nil,
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVPeerService: true,
			err:                true,
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
		modify                       func(cluster *v1alpha1.TidbCluster)
		pdStores                     *controller.StoresInfo
		tombstoneStores              *controller.StoresInfo
		errWhenUpdateStatefulSet     bool
		errWhenUpdateTiKVPeerService bool
		errWhenGetStores             bool
		err                          bool
		expectTiKVPeerServiceFn      func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn          func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn          func(*GomegaWithT, *v1alpha1.TidbCluster)
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

		_, err = pmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdateTiKVPeerService {
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

		if test.expectTiKVPeerServiceFn != nil {
			svc, err := pmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
			test.expectTiKVPeerServiceFn(g, svc, err)
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
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
				tc.Spec.Services = []v1alpha1.Service{
					{Name: "tikv", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			// TODO add unit test for status sync
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVPeerService: false,
			errWhenGetStores:             false,
			err:                          false,
			expectTiKVPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(5))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(*tc.Status.TiKV.StatefulSet.ObservedGeneration).To(Equal(int64(1)))
				expectedStores := v1alpha1.TiKVStores{
					CurrentStores:   map[string]v1alpha1.TiKVStore{},
					TombStoneStores: map[string]v1alpha1.TiKVStore{},
				}
				g.Expect(tc.Status.TiKV.Stores).To(Equal(expectedStores))
			},
		},
		{
			name: "tidbcluster's storage format is wrong",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVPeerService: false,
			err: true,
			expectTiKVPeerServiceFn: nil,
			expectStatefulSetFn:     nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     true,
			errWhenUpdateTiKVPeerService: false,
			err: true,
			expectTiKVPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(3))
			},
		},
		{
			name: "error when sync tikv status",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
			},
			pdStores:                     &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			tombstoneStores:              &controller.StoresInfo{Count: 0, Stores: []*controller.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVPeerService: false,
			errWhenGetStores:             true,
			err:                          true,
			expectTiKVPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(3))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				expectedStores := v1alpha1.TiKVStores{}
				g.Expect(tc.Status.TiKV.Stores).To(Equal(expectedStores))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiKVMemberManager(tc *v1alpha1.TidbCluster) (
	*TikvMemberManager, *controller.FakeStatefulSetControl,
	*controller.FakeServiceControl, *controller.FakePDClient) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	pdControl := controller.NewFakePDControl()
	pdClient := controller.NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	nodeInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Nodes()

	return NewTiKVMemberManager(
		pdControl,
		setControl,
		svcControl,
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		nodeInformer.Lister(),
	), setControl, svcControl, pdClient
}
