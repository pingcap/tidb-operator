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
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/pkg/typeutil"
	"github.com/pingcap/pd/server"
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
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/kubelet/apis"
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
		pdStores                     *pdapi.StoresInfo
		tombstoneStores              *pdapi.StoresInfo
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPD()
		tc.Status.PD.Members = map[string]v1alpha1.PDMember{
			"pd-0": {Name: "pd-0", Health: true},
			"pd-1": {Name: "pd-1", Health: true},
			"pd-2": {Name: "pd-2", Health: true},
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}

		ns := tc.Namespace
		tcName := tc.Name
		oldSpec := tc.Spec
		if test.prepare != nil {
			test.prepare(tc)
		}

		tkmm, fakeSetControl, fakeSvcControl, pdClient, _, _ := newFakeTiKVMemberManager(tc)
		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			return &server.Config{
				Replication: server.ReplicationConfig{
					LocationLabels: typeutil.StringSlice{"region", "zone", "rack", "host"},
				},
			}, nil
		})
		if test.errWhenGetStores {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get stores from tikv cluster")
			})
		} else {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.pdStores, nil
			})
			pdClient.AddReaction(pdapi.GetTombStoneStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.tombstoneStores, nil
			})
			pdClient.AddReaction(pdapi.SetStoreLabelsActionType, func(action *pdapi.Action) (interface{}, error) {
				return true, nil
			})
		}

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateTiKVPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := tkmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc, err := tkmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
		if test.tikvPeerSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		tc1, err := tkmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
		if test.setCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}
	}

	tests := []testcase{
		{
			name:                         "normal",
			prepare:                      nil,
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVPeerService: false,
			err:                          false,
			tikvPeerSvcCreated:           true,
			setCreated:                   true,
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
		},
		{
			name: "pd is not available",
			prepare: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Members = map[string]v1alpha1.PDMember{}
			},
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVPeerService: false,
			err:                          true,
			tikvPeerSvcCreated:           false,
			setCreated:                   false,
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
		},
		{
			name: "tidbcluster's storage format is wrong",
			prepare: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
			},
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVPeerService: false,
			err:                          true,
			tikvPeerSvcCreated:           true,
			setCreated:                   false,
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
		},
		{
			name:                         "error when create statefulset",
			prepare:                      nil,
			errWhenCreateStatefulSet:     true,
			errWhenCreateTiKVPeerService: false,
			err:                          true,
			tikvPeerSvcCreated:           true,
			setCreated:                   false,
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
		},
		{
			name:                         "error when create tikv peer service",
			prepare:                      nil,
			errWhenCreateStatefulSet:     false,
			errWhenCreateTiKVPeerService: true,
			err:                          true,
			tikvPeerSvcCreated:           false,
			setCreated:                   false,
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
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
		pdStores                     *pdapi.StoresInfo
		tombstoneStores              *pdapi.StoresInfo
		errWhenUpdateStatefulSet     bool
		errWhenUpdateTiKVPeerService bool
		errWhenGetStores             bool
		statusChange                 func(*apps.StatefulSet)
		err                          bool
		expectTiKVPeerServiceFn      func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn          func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn          func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPD()
		tc.Status.PD.Members = map[string]v1alpha1.PDMember{
			"pd-0": {Name: "pd-0", Health: true},
			"pd-1": {Name: "pd-1", Health: true},
			"pd-2": {Name: "pd-2", Health: true},
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}

		ns := tc.Namespace
		tcName := tc.Name

		tkmm, fakeSetControl, fakeSvcControl, pdClient, _, _ := newFakeTiKVMemberManager(tc)
		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			return &server.Config{
				Replication: server.ReplicationConfig{
					LocationLabels: typeutil.StringSlice{"region", "zone", "rack", "host"},
				},
			}, nil
		})
		if test.errWhenGetStores {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get stores from pd cluster")
			})
		} else {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.pdStores, nil
			})
			pdClient.AddReaction(pdapi.GetTombStoneStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.tombstoneStores, nil
			})
			pdClient.AddReaction(pdapi.SetStoreLabelsActionType, func(action *pdapi.Action) (interface{}, error) {
				return true, nil
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

		err := tkmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = tkmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = tkmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdateTiKVPeerService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = tkmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectTiKVPeerServiceFn != nil {
			svc, err := tkmm.svcLister.Services(ns).Get(controller.TiKVPeerMemberName(tcName))
			test.expectTiKVPeerServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := tkmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
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
				tc.Status.PD.Phase = v1alpha1.NormalPhase
			},
			// TODO add unit test for status sync
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVPeerService: false,
			errWhenGetStores:             false,
			err:                          false,
			expectTiKVPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(4))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiKV.StatefulSet.ObservedGeneration).To(Equal(int64(1)))
				g.Expect(tc.Status.TiKV.Stores).To(Equal(map[string]v1alpha1.TiKVStore{}))
				g.Expect(tc.Status.TiKV.TombstoneStores).To(Equal(map[string]v1alpha1.TiKVStore{}))
			},
		},
		{
			name: "tidbcluster's storage format is wrong",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Requests.Storage = "100xxxxi"
				tc.Status.PD.Phase = v1alpha1.NormalPhase
			},
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVPeerService: false,
			err:                          true,
			expectTiKVPeerServiceFn:      nil,
			expectStatefulSetFn:          nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
				tc.Status.PD.Phase = v1alpha1.NormalPhase
			},
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			errWhenUpdateStatefulSet:     true,
			errWhenUpdateTiKVPeerService: false,
			err:                          true,
			expectTiKVPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "error when sync tikv status",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.Replicas = 5
				tc.Status.PD.Phase = v1alpha1.NormalPhase
			},
			pdStores:                     &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			tombstoneStores:              &pdapi.StoresInfo{Count: 0, Stores: []*pdapi.StoreInfo{}},
			errWhenUpdateStatefulSet:     false,
			errWhenUpdateTiKVPeerService: false,
			errWhenGetStores:             true,
			err:                          true,
			expectTiKVPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(3))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(0))
			},
		},
	}

	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestTiKVMemberManagerTiKVStatefulSetIsUpgrading(t *testing.T) {
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
		tc := newTidbClusterForPD()
		pmm, _, _, _, podIndexer, _ := newFakeTiKVMemberManager(tc)
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{
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
					Name:        ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).TiKV().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			podIndexer.Add(pod)
		}
		b, err := pmm.tikvStatefulSetIsUpgradingFn(pmm.podLister, pmm.pdControl, set, tc)
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
			errExpectFn:     errExpectNil,
			expectUpgrading: true,
		},
		{
			name:            "pod don't have revision hash",
			setUpdate:       nil,
			hasPod:          true,
			updatePod:       nil,
			errExpectFn:     errExpectNil,
			expectUpgrading: false,
		},
		{
			name:      "pod have revision hash, not equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v2"
			},
			errExpectFn:     errExpectNil,
			expectUpgrading: true,
		},
		{
			name:      "pod have revision hash, equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v3"
			},
			errExpectFn:     errExpectNil,
			expectUpgrading: false,
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestTiKVMemberManagerSetStoreLabelsForTiKV(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		errWhenGetStores bool
		hasNode          bool
		hasPod           bool
		storeInfo        *pdapi.StoresInfo
		errExpectFn      func(*GomegaWithT, error)
		setCount         int
		labelSetFailed   bool
	}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		pmm, _, _, pdClient, podIndexer, nodeIndexer := newFakeTiKVMemberManager(tc)
		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			return &server.Config{
				Replication: server.ReplicationConfig{
					LocationLabels: typeutil.StringSlice{"region", "zone", "rack", "host"},
				},
			}, nil
		})
		if test.errWhenGetStores {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get stores")
			})
		}
		if test.storeInfo != nil {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.storeInfo, nil
			})
		}
		if test.hasNode {
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Labels: map[string]string{
						"region":           "region",
						"zone":             "zone",
						"rack":             "rack",
						apis.LabelHostname: "host",
					},
				},
			}
			nodeIndexer.Add(node)
		}
		if test.hasPod {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: corev1.PodSpec{
					NodeName: "node-1",
				},
			}
			podIndexer.Add(pod)
		}
		if test.labelSetFailed {
			pdClient.AddReaction(pdapi.SetStoreLabelsActionType, func(action *pdapi.Action) (interface{}, error) {
				return false, fmt.Errorf("label set failed")
			})
		} else {
			pdClient.AddReaction(pdapi.SetStoreLabelsActionType, func(action *pdapi.Action) (interface{}, error) {
				return true, nil
			})
		}

		setCount, err := pmm.setStoreLabelsForTiKV(tc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		g.Expect(setCount).To(Equal(test.setCount))
	}
	tests := []testcase{
		{
			name:             "get stores return error",
			errWhenGetStores: true,
			storeInfo:        nil,
			hasNode:          true,
			hasPod:           true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to get stores")).To(BeTrue())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "stores is empty",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			hasNode: true,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "status is nil",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Status: nil,
					},
				},
			},
			hasNode: true,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "store is nil",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: nil,
					},
				},
			},
			hasNode: true,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "don't have pod",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LeaderCount:     1,
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			hasNode: true,
			hasPod:  false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "not found")).To(BeTrue())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "don't have node",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LeaderCount:     1,
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			hasNode: false,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "already has labels",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
								Labels: []*metapb.StoreLabel{
									{
										Key:   "region",
										Value: "region",
									},
									{
										Key:   "zone",
										Value: "zone",
									},
									{
										Key:   "rack",
										Value: "rack",
									},
									{
										Key:   "host",
										Value: "host",
									},
								},
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LeaderCount:     1,
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			hasNode: true,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       0,
			labelSetFailed: false,
		},
		{
			name:             "labels not equal, but set failed",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
								Labels: []*metapb.StoreLabel{
									{
										Key:   "region",
										Value: "region",
									},
								},
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LeaderCount:     1,
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			hasNode: true,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       0,
			labelSetFailed: true,
		},
		{
			name:             "labels not equal, set success",
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
								Labels: []*metapb.StoreLabel{
									{
										Key:   "region",
										Value: "region",
									},
								},
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LeaderCount:     1,
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			hasNode: true,
			hasPod:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setCount:       1,
			labelSetFailed: false,
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestTiKVMemberManagerSyncTidbClusterStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                      string
		updateTC                  func(*v1alpha1.TidbCluster)
		upgradingFn               func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
		errWhenGetStores          bool
		storeInfo                 *pdapi.StoresInfo
		errWhenGetTombstoneStores bool
		tombstoneStoreInfo        *pdapi.StoresInfo
		errExpectFn               func(*GomegaWithT, error)
		tcExpectFn                func(*GomegaWithT, *v1alpha1.TidbCluster)
	}
	status := apps.StatefulSetStatus{
		Replicas: int32(3),
	}
	now := metav1.Time{Time: time.Now()}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		tc.Status.PD.Phase = v1alpha1.NormalPhase
		set := &apps.StatefulSet{
			Status: status,
		}
		if test.updateTC != nil {
			test.updateTC(tc)
		}
		pmm, _, _, pdClient, _, _ := newFakeTiKVMemberManager(tc)

		if test.upgradingFn != nil {
			pmm.tikvStatefulSetIsUpgradingFn = test.upgradingFn
		}
		if test.errWhenGetStores {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get stores")
			})
		} else if test.storeInfo != nil {
			pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.storeInfo, nil
			})
		}
		if test.errWhenGetTombstoneStores {
			pdClient.AddReaction(pdapi.GetTombStoneStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get tombstone stores")
			})
		} else if test.tombstoneStoreInfo != nil {
			pdClient.AddReaction(pdapi.GetTombStoneStoresActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.tombstoneStoreInfo, nil
			})
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
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, fmt.Errorf("whether upgrading failed")
			},
			errWhenGetStores:          false,
			storeInfo:                 nil,
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo:        nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "whether upgrading failed")).To(BeTrue())
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiKV.StatefulSet.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name:     "statefulset is upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			errWhenGetStores:          false,
			storeInfo:                 nil,
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo:        nil,
			errExpectFn:               nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiKV.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
		{
			name: "statefulset is upgrading but pd is upgrading",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			errWhenGetStores:          false,
			storeInfo:                 nil,
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo:        nil,
			errExpectFn:               nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiKV.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name:     "statefulset is not upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores:          false,
			storeInfo:                 nil,
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo:        nil,
			errExpectFn:               nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiKV.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name:     "get stores failed",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores:          true,
			storeInfo:                 nil,
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo:        nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to get stores")).To(BeTrue())
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiKV.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiKV.Synced).To(BeFalse())
			},
		},
		{
			name:     "stores is empty",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(0))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name:     "store is nil",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: nil,
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(0))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name:     "status is nil",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Status: nil,
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(0))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "LastHeartbeatTS is zero, TidbClulster LastHeartbeatTS is not zero",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{
					LastHeartbeatTime: now,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Time{},
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(time.Time{}.IsZero()).To(BeTrue())
				g.Expect(tc.Status.TiKV.Stores["333"].LastHeartbeatTime.Time.IsZero()).To(BeFalse())
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "LastHeartbeatTS is zero, TidbClulster LastHeartbeatTS is zero",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{
					LastHeartbeatTime: metav1.Time{Time: time.Time{}},
				}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Time{},
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(time.Time{}.IsZero()).To(BeTrue())
				g.Expect(tc.Status.TiKV.Stores["333"].LastHeartbeatTime.Time.IsZero()).To(BeTrue())
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "LastHeartbeatTS is not zero, TidbClulster LastHeartbeatTS is zero",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{
					LastHeartbeatTime: metav1.Time{Time: time.Time{}},
				}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(time.Time{}.IsZero()).To(BeTrue())
				g.Expect(tc.Status.TiKV.Stores["333"].LastHeartbeatTime.Time.IsZero()).To(BeFalse())
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "set LastTransitionTime first time",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				// tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(tc.Status.TiKV.Stores["333"].LastTransitionTime.Time.IsZero()).To(BeFalse())
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "state not change, LastTransitionTime not change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{
					LastTransitionTime: now,
					State:              v1alpha1.TiKVStateUp,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(tc.Status.TiKV.Stores["333"].LastTransitionTime).To(Equal(now))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "state change, LastTransitionTime change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{
					LastTransitionTime: now,
					State:              v1alpha1.TiKVStateUp,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Down",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(tc.Status.TiKV.Stores["333"].LastTransitionTime).NotTo(Equal(now))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
		{
			name: "get tombstone stores failed",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiKV.Stores["333"] = v1alpha1.TiKVStore{
					LastTransitionTime: now,
					State:              v1alpha1.TiKVStateUp,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errWhenGetTombstoneStores: true,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{},
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to get tombstone stores")).To(BeTrue())
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.Synced).To(BeFalse())
			},
		},
		{
			name:     "all works",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, controlInterface pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			errWhenGetStores: false,
			storeInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errWhenGetTombstoneStores: false,
			tombstoneStoreInfo: &pdapi.StoresInfo{
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      333,
								Address: "pod-1.ns-1",
							},
							StateName: "Tombstone",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
				},
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(1))
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func newFakeTiKVMemberManager(tc *v1alpha1.TidbCluster) (
	*tikvMemberManager, *controller.FakeStatefulSetControl,
	*controller.FakeServiceControl, *pdapi.FakePDClient, cache.Indexer, cache.Indexer) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	pdControl := pdapi.NewFakePDControl()
	pdClient := controller.NewFakePDClient(pdControl, tc)
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer, tcInformer)
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	nodeInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Nodes()
	tikvScaler := NewFakeTiKVScaler()
	tikvUpgrader := NewFakeTiKVUpgrader()

	tmm := &tikvMemberManager{
		pdControl:    pdControl,
		podLister:    podInformer.Lister(),
		nodeLister:   nodeInformer.Lister(),
		setControl:   setControl,
		svcControl:   svcControl,
		setLister:    setInformer.Lister(),
		svcLister:    svcInformer.Lister(),
		tikvScaler:   tikvScaler,
		tikvUpgrader: tikvUpgrader,
	}
	tmm.tikvStatefulSetIsUpgradingFn = tikvStatefulSetIsUpgrading
	return tmm, setControl, svcControl, pdClient, podInformer.Informer().GetIndexer(), nodeInformer.Informer().GetIndexer()
}
