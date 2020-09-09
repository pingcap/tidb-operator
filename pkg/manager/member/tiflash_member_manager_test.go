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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestTiFlashMemberManagerTiFlashStatefulSetIsUpgrading(t *testing.T) {
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
		pmm, _, _, _, podIndexer, _ := newFakeTiFlashMemberManager(tc)
		tc.Status.TiFlash.StatefulSet = &apps.StatefulSetStatus{
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
					Name:        ordinalPodName(v1alpha1.TiFlashMemberType, tc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.New().Instance(tc.GetInstanceName()).TiFlash().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			podIndexer.Add(pod)
		}
		b, err := pmm.tiflashStatefulSetIsUpgradingFn(pmm.podLister, pmm.pdControl, set, tc)
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

func TestTiFlashMemberManagerSetStoreLabelsForTiFlash(t *testing.T) {
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
		pmm, _, _, pdClient, podIndexer, nodeIndexer := newFakeTiFlashMemberManager(tc)
		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			return &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					LocationLabels: []string{"region", "zone", "rack", "host"},
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
						"region":             "region",
						"zone":               "zone",
						"rack":               "rack",
						corev1.LabelHostname: "host",
					},
				},
			}
			nodeIndexer.Add(node)
		}
		if test.hasPod {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tiflash-1",
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

		setCount, err := pmm.setStoreLabelsForTiFlash(tc)
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
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

func TestTiFlashMemberManagerSyncTidbClusterStatus(t *testing.T) {
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
			Spec: apps.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(0),
			},
			Status: status,
		}
		if test.updateTC != nil {
			test.updateTC(tc)
		}
		pmm, _, _, pdClient, _, _ := newFakeTiFlashMemberManager(tc)

		if test.upgradingFn != nil {
			pmm.tiflashStatefulSetIsUpgradingFn = test.upgradingFn
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
				g.Expect(tc.Status.TiFlash.StatefulSet.Replicas).To(Equal(int32(3)))
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
				g.Expect(tc.Status.TiFlash.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
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
				g.Expect(tc.Status.TiFlash.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.UpgradePhase))
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
				g.Expect(tc.Status.TiFlash.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiFlash.Phase).To(Equal(v1alpha1.NormalPhase))
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
				g.Expect(tc.Status.TiFlash.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiFlash.Synced).To(BeFalse())
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(0))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(0))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(0))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "LastHeartbeatTS is zero, TidbClulster LastHeartbeatTS is not zero",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Time{},
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(time.Time{}.IsZero()).To(BeTrue())
				g.Expect(tc.Status.TiFlash.Stores["333"].LastHeartbeatTime.Time.IsZero()).To(BeFalse())
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "LastHeartbeatTS is zero, TidbClulster LastHeartbeatTS is zero",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Time{},
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(time.Time{}.IsZero()).To(BeTrue())
				g.Expect(tc.Status.TiFlash.Stores["333"].LastHeartbeatTime.Time.IsZero()).To(BeTrue())
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "LastHeartbeatTS is not zero, TidbClulster LastHeartbeatTS is zero",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(time.Time{}.IsZero()).To(BeTrue())
				g.Expect(tc.Status.TiFlash.Stores["333"].LastHeartbeatTime.Time.IsZero()).To(BeFalse())
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "set LastTransitionTime first time",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				// tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{}
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(tc.Status.TiFlash.Stores["333"].LastTransitionTime.Time.IsZero()).To(BeFalse())
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "state not change, LastTransitionTime not change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(tc.Status.TiFlash.Stores["333"].LastTransitionTime).To(Equal(now))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "state change, LastTransitionTime change",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
							StateName: "Down",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(tc.Status.TiFlash.Stores["333"].LastTransitionTime).NotTo(Equal(now))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
		{
			name: "get tombstone stores failed",
			updateTC: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiFlash.Stores = map[string]v1alpha1.TiKVStore{}
				tc.Status.TiFlash.Stores["333"] = v1alpha1.TiKVStore{
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(0))
				g.Expect(tc.Status.TiFlash.Synced).To(BeFalse())
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
							StateName: "Up",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
								Address: fmt.Sprintf("%s-tiflash-1.%s-tiflash-peer.%s.svc:20160", "test", "test", "default"),
							},
							StateName: "Tombstone",
						},
						Status: &pdapi.StoreStatus{
							LastHeartbeatTS: time.Now(),
						},
					},
					{
						Store: &pdapi.MetaStore{
							Store: &metapb.Store{
								Id:      330,
								Address: fmt.Sprintf("%s-tikv-1.%s-tikv-peer.%s.svc:20160", "test", "test", "default"),
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
				g.Expect(len(tc.Status.TiFlash.Stores)).To(Equal(1))
				g.Expect(len(tc.Status.TiFlash.TombstoneStores)).To(Equal(1))
				g.Expect(tc.Status.TiFlash.Synced).To(BeTrue())
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func newFakeTiFlashMemberManager(tc *v1alpha1.TidbCluster) (
	*tiflashMemberManager, *controller.FakeStatefulSetControl,
	*controller.FakeServiceControl, *pdapi.FakePDClient, cache.Indexer, cache.Indexer) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	pdControl := pdapi.NewFakePDControl(kubeCli)
	pdClient := controller.NewFakePDClient(pdControl, tc)
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer, tcInformer)
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	nodeInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Nodes()
	tiflashScaler := NewFakeTiFlashScaler()
	tiflashUpgrader := NewFakeTiFlashUpgrader()
	genericControl := controller.NewFakeGenericControl()

	tmm := &tiflashMemberManager{
		pdControl:       pdControl,
		podLister:       podInformer.Lister(),
		nodeLister:      nodeInformer.Lister(),
		setControl:      setControl,
		svcControl:      svcControl,
		typedControl:    controller.NewTypedControl(genericControl),
		setLister:       setInformer.Lister(),
		svcLister:       svcInformer.Lister(),
		tiflashScaler:   tiflashScaler,
		tiflashUpgrader: tiflashUpgrader,
	}
	tmm.tiflashStatefulSetIsUpgradingFn = tiflashStatefulSetIsUpgrading
	return tmm, setControl, svcControl, pdClient, podInformer.Informer().GetIndexer(), nodeInformer.Informer().GetIndexer()
}

func TestGetNewServiceForTidbCluster(t *testing.T) {
	tests := []struct {
		name      string
		tc        v1alpha1.TidbCluster
		svcConfig SvcConfig
		expected  corev1.Service
	}{
		{
			name: "basic",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
			},
			svcConfig: SvcConfig{
				Name:       "peer",
				Port:       20160,
				Headless:   true,
				SvcLabel:   func(l label.Label) label.Label { return l.TiFlash() },
				MemberName: controller.TiFlashPeerMemberName,
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tiflash-peer",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tiflash",
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
							Name:       "peer",
							Port:       20160,
							TargetPort: intstr.FromInt(20160),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tiflash",
					},
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewServiceForTidbCluster(&tt.tc, tt.svcConfig)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewTiFlashSetForTidbCluster(t *testing.T) {
	enable := true
	tests := []struct {
		name    string
		tc      v1alpha1.TidbCluster
		wantErr bool
		testSts func(sts *apps.StatefulSet)
	}{
		{
			name: "tiflash network is not host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("10Gi"),
									},
								},
							},
						},
					},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "tiflash network is host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("10Gi"),
									},
								},
							},
						},
					},
				},
			},
			testSts: testHostNetwork(t, true, v1.DNSClusterFirstWithHostNet),
		},
		{
			name: "tiflash network is not host when pd is host",
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
					TiFlash: &v1alpha1.TiFlashSpec{
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("10Gi"),
									},
								},
							},
						},
					},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "tiflash network is not host when tidb is host",
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
					TiFlash: &v1alpha1.TiFlashSpec{
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("10Gi"),
									},
								},
							},
						},
					},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "tiflash delete slots",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
					Annotations: map[string]string{
						label.AnnTiFlashDeleteSlots: "[0,1]",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{},
					TiFlash: &v1alpha1.TiFlashSpec{
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("10Gi"),
									},
								},
							},
						},
					},
				},
			},
			testSts: testAnnotations(t, map[string]string{"delete-slots": "[0,1]"}),
		},
		{
			name: "tiflash should respect resources config",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{
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
								corev1.ResourceStorage:          resource.MustParse("100Gi"),
							},
						},
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
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
				tiflashContainer := nameToContainer[v1alpha1.TiFlashMemberType.String()]
				g.Expect(tiflashContainer.Resources).To(Equal(corev1.ResourceRequirements{
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
				var capacityEnvVar corev1.EnvVar
				for i := range tiflashContainer.Env {
					if tiflashContainer.Env[i].Name == "CAPACITY" {
						capacityEnvVar = tiflashContainer.Env[i]
						break
					}
				}
				g.Expect(capacityEnvVar).To(Equal(corev1.EnvVar{
					Name:  "CAPACITY",
					Value: "100GB",
				}), "Expected the CAPACITY of tiflash is properly set")
			},
		},
		// TODO add more tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, err := getNewStatefulSet(&tt.tc, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("error %v, wantErr %v", err, tt.wantErr)
			}
			tt.testSts(sts)
		})
	}
}
